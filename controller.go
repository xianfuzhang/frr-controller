/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	frrv1alpha1 "github.com/guohao117/frr-controller/pkg/apis/frrcontroller/v1alpha1"
	clientset "github.com/guohao117/frr-controller/pkg/generated/clientset/versioned"
	frrscheme "github.com/guohao117/frr-controller/pkg/generated/clientset/versioned/scheme"
	informers "github.com/guohao117/frr-controller/pkg/generated/informers/externalversions/frrcontroller/v1alpha1"
	listers "github.com/guohao117/frr-controller/pkg/generated/listers/frrcontroller/v1alpha1"

	"github.com/guohao117/frr-controller/pkg/range_manager"
)

const controllerAgentName = "sample-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Frr is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Frr fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Frr"
	// MessageResourceSynced is the message used for an Event fired when a Frr
	// is synced successfully
	MessageResourceSynced = "Frr synced successfully"
)

var (
	frrUID int64 = 100
	frrGID int64 = 101
)

// Controller is the controller implementation for Frr resources
type Controller struct {
	// vni allocator
	vniManager *rangemanager.RangeManager
	// asn allocator
	asnManager *rangemanager.RangeManager
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	frrclientset clientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	frrsLister        listers.FrrLister
	frrsSynced        cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	frrclientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	frrInformer informers.FrrInformer,
	minVNI, maxVNI int,
	minASN, maxASN int) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(frrscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	vniMan, err := rangemanager.NewRangeManager(minVNI, maxVNI)
	if err != nil {
		return nil
	}
	asnMan, err := rangemanager.NewRangeManager(minASN, maxASN)
	if err != nil {
		return nil
	}
	controller := &Controller{
		vniManager:        vniMan,
		asnManager:        asnMan,
		kubeclientset:     kubeclientset,
		frrclientset:      frrclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		frrsLister:        frrInformer.Lister(),
		frrsSynced:        frrInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Frrs"),
		recorder:          recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Frr resources change
	frrInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueFrr,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueFrr(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Frr resource then the handler will enqueue that Frr resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Frr controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.frrsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Frr resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Frr resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Frr resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Frr resource with this namespace/name
	frr, err := c.frrsLister.Frrs(namespace).Get(name)
	if err != nil {
		// The Frr resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("frr '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	frrscopedName := frr.Namespace + "/" + frr.Name

	deploymentName := frr.Spec.DeploymentName
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}

	// Get the deployment with the name specified in Frr.spec
	deployment, err := c.deploymentsLister.Deployments(frr.Namespace).Get(deploymentName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		asn, err := c.asnManager.Allocate(frrscopedName)
		if err != nil {
			return err
		}
		vni, err := c.vniManager.Allocate(frrscopedName)
		if err != nil {
			return err
		}
		deployment, err = c.kubeclientset.AppsV1().Deployments(frr.Namespace).Create(context.TODO(), newDeployment(frr, asn, vni), metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("Failed to create deployment: %v", err)
			return err
		}
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this Frr resource, we should log
	// a warning to the event recorder and return error msg.
	if !metav1.IsControlledBy(deployment, frr) {
		msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
		c.recorder.Event(frr, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf("%s", msg)
	}

	// If this number of the replicas on the Frr resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	// if frr.Spec.Replicas != nil && *frr.Spec.Replicas != *deployment.Spec.Replicas {
	// 	klog.V(4).Infof("Frr %s replicas: %d, deployment replicas: %d", name, *frr.Spec.Replicas, *deployment.Spec.Replicas)
	// 	deployment, err = c.kubeclientset.AppsV1().Deployments(frr.Namespace).Update(context.TODO(), newDeployment(frr), metav1.UpdateOptions{})
	// }

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// Finally, we update the status block of the Frr resource to reflect the
	// current state of the world
	err = c.updateFrrStatus(frr, deployment)
	if err != nil {
		return err
	}

	c.recorder.Event(frr, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) updateFrrStatus(frr *frrv1alpha1.Frr, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	frrCopy := frr.DeepCopy()
	frrCopy.Status.VNI = frr.Spec.VNI
	frrCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas

	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Frr resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.frrclientset.FrrcontrollerV1alpha1().Frrs(frr.Namespace).UpdateStatus(context.TODO(), frrCopy, metav1.UpdateOptions{})
	return err
}

// enqueueFrr takes a Frr resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Frr.
func (c *Controller) enqueueFrr(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Frr resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Frr resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Frr, we should not do anything more
		// with it.
		if ownerRef.Kind != "Frr" {
			return
		}

		frr, err := c.frrsLister.Frrs(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s/%s' of frr '%s'", object.GetNamespace(), object.GetName(), ownerRef.Name)
			return
		}

		c.enqueueFrr(frr)
		return
	}
}

// func newInitContainers(frr *frrv1alpha1.Frr) []corev1.Container {
// }

// newDeployment creates a new Deployment for a Frr resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Frr resource that 'owns' it.
func newDeployment(frr *frrv1alpha1.Frr, asn, vni int) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "frr",
		"controller": frr.Name,
	}
	frrContainerEnv := make([]corev1.EnvVar, 0)
	frrContainerEnv = append(frrContainerEnv, corev1.EnvVar{
		Name:  "ASNUMBER",
		Value: fmt.Sprintf("%d", frr.Spec.ASNumber),
	})
	frrContainerEnv = append(frrContainerEnv, corev1.EnvVar{
		Name:  "NEIGHBORS",
		Value: strings.Join(frr.Spec.Neighbors, ","),
	})
	frrContainerEnv = append(frrContainerEnv, corev1.EnvVar{
		Name:  "VNI",
		Value: fmt.Sprintf("%d", vni),
	})
	frrContainerEnv = append(frrContainerEnv, corev1.EnvVar{
		Name:  "TINT_SUBREAPER",
		Value: "true",
	})
	// add an envfrom status.podIP
	frrContainerEnv = append(frrContainerEnv, corev1.EnvVar{
		Name: "VTEP_LOCAL",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "status.podIP",
			},
		},
	})
	frrContainerSecurityContext := &corev1.SecurityContext{}
	frrContainerSecurityContext.Capabilities = &corev1.Capabilities{
		Add: []corev1.Capability{
			"NET_ADMIN",
			"SYS_ADMIN",
		},
	}

	volumes := make([]corev1.Volume, 0)
	// add a emptyDir volume for frr-conf
	volumes = append(volumes, corev1.Volume{
		Name: "frr-startup",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	ovsVols := make([]corev1.Volume, 0)
	ovsVols = append(ovsVols, corev1.Volume{
		Name: "host-var-run-ovs",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/var/run/openvswitch",
			},
		},
	})
	ovsVols = append(ovsVols, corev1.Volume{
		Name: "host-run-ovs",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/run/openvswitch",
			},
		},
	})
	ovsVols = append(ovsVols, corev1.Volume{
		Name: "host-modules",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/lib/modules",
			},
		},
	})

	initContainerVolumeMounts := make([]corev1.VolumeMount, 0)
	initContainerVolumeMounts = append(initContainerVolumeMounts, corev1.VolumeMount{
		Name:      "frr-startup",
		MountPath: "/tmp/frr",
	})

	frrContainerVolumeMounts := make([]corev1.VolumeMount, 0)
	// add a volume mount for frr-conf
	// frrContainerVolumeMounts = append(frrContainerVolumeMounts, corev1.VolumeMount{
	// 	Name:      "frr-conf",
	// 	MountPath: "/etc/frr",
	// })
	frrContainerVolumeMounts = append(frrContainerVolumeMounts, corev1.VolumeMount{
		Name:      "host-var-run-ovs",
		MountPath: "/var/run/openvswitch",
	})
	frrContainerVolumeMounts = append(frrContainerVolumeMounts, corev1.VolumeMount{
		Name:      "host-run-ovs",
		MountPath: "/run/openvswitch",
	})
	frrContainerVolumeMounts = append(frrContainerVolumeMounts, corev1.VolumeMount{
		Name:      "host-modules",
		MountPath: "/lib/modules",
		ReadOnly:  true,
	})
	frrContainerVolumeMounts = append(frrContainerVolumeMounts, initContainerVolumeMounts...)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      frr.Spec.DeploymentName,
			Namespace: frr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(frr, frrv1alpha1.SchemeGroupVersion.WithKind("Frr")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: frr.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					Volumes:     append(volumes, ovsVols...),
					InitContainers: []corev1.Container{
						{
							Name:            "frr-conf-init",
							Image:           frr.Spec.InitConfigImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             frrContainerEnv,
							VolumeMounts:    initContainerVolumeMounts,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &frrUID,
								RunAsGroup: &frrGID,
							},
							Args: []string{
								"/tmp/frr/frr.conf",
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "frr",
							Image:           frr.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             frrContainerEnv,
							VolumeMounts:    frrContainerVolumeMounts,
							Command: []string{
								"/bin/sh",
							},
							Args: []string{
								"-c",
								"/sbin/tini -- cp /tmp/frr/frr.conf /etc/frr/ && /usr/lib/frr/docker-start",
								// `/sbin/tini -- /usr/lib/frr/docker-start &
								// attempts=0
								// until [[ -f /var/log/frr/frr.log || $attempts -eq 60 ]]; do
								// 	sleep 1
								// 	attempts=$(( $attempts + 1 ))
								// done
								// tail -f /var/log/frr/frr.log`,
							},
							SecurityContext: frrContainerSecurityContext,
						},
					},
					NodeSelector: frr.Spec.NodeSelector.MatchLabels,
				},
			},
		},
	}
}
