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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Frr is a specification for a Frr resource
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=frrs,scope=Namespaced
// +kubebuilder:printcolumn:name="AS Number",type="integer",JSONPath=".spec.asNumber",description="AS Number"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas",description="Replicas"
// +kubebuilder:printcolumn:name="Available Replicas",type="integer",JSONPath=".status.availableReplicas",description="Available Replicas"
// +kubebuilder:printcolumn:name="Nodes",type="string",JSONPath=".status.nodes",description="Nodes"
type Frr struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FrrSpec `json:"spec"`
	// +optional
	Status FrrStatus `json:"status"`
}

// FrrSpec is the spec for a Frr resource
type FrrSpec struct {
	DeploymentName string   `json:"deploymentName"`
	Replicas       *int32   `json:"replicas"`
	Image          string   `json:"image"`
	ASNumber       int      `json:"asNumber"`
	Neighbors      []string `json:"neighbor,omitempty"`
	// +kubebuilder:default={matchLabels: {frrcontroller.nocsys.cn/frr-assignable: ""}}
	// +optional
	NodeSelector metav1.LabelSelector `json:"nodeSelector,omitempty"`
}

// FrrStatus is the status for a Frr resource
type FrrStatus struct {
	AvailableReplicas int32  `json:"availableReplicas"`
	Nodes             string `json:"nodes,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FrrList is a list of Frr resources
type FrrList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Frr `json:"items"`
}
