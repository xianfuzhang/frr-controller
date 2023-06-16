# frr-controller

## Fetch frr-controller and its dependencies

Like the rest of Kubernetes, frr-controller has used
[godep](https://github.com/tools/godep) and `$GOPATH` for years and is
now adopting go 1.11 modules.  There are thus two alternative ways to
go about fetching this demo and its dependencies.

### When using go 1.11 modules

When using go 1.11 modules (`GO111MODULE=on`), issue the following
commands --- starting in whatever working directory you like.

```sh
git clone https://github.com/guohao117/frr-controller.git
cd frr-controller
```

Note, however, that if you intend to
generate code then you will also need the
code-generator repo to exist in an old-style location.  One easy way
to do this is to use the command `go mod vendor` to create and
populate the `vendor` directory.

### update codegen
It makes use of the generators in k8s.io/code-generator to generate a typed client, informers, listers and deep-copy functions. You can do this yourself using the ./hack/update-codegen.sh script.

The update-codegen script will automatically generate the following files & directories:

- pkg/apis/samplecontroller/v1alpha1/zz_generated.deepcopy.go
- pkg/generated/

In this case, you should clone the repo in an old-style localtion. for example $GOPATH/src, or anywhere which make the directory tree looks like github.com/username/frr-controller, then run `hack/update-codegen.sh`

```sh
git clone https://github.com/guohao117/frr-controller.git $GOPATH/src/github.com/guohao117/frr-controller
cd frr-controller
go mod vendor
# make changes on types.go
./hack/update-codegen.sh
```

## build
```sh
cd frr-controller
go build -o frr-controller .
```

## build frr.conf
```sh
cd frr-controller
docker build -t nocsyscn/frr_conf:0.1 -f build/frr.conf.Dockerfile .
# default output dir /etc/frr/frr.conf
docker run -d --name=frr-conf --env ASNUMBER=345 --env VTEP_LOCAL=11.10.10.10 --env NEIGHBORS=10.0.0.30,10.0.0.40 nocsyscn/frr_conf:0.1
# or you can output a destination dir
docker run -d --name=frr-conf --env ASNUMBER=345 --env VTEP_LOCAL=11.10.10.10 --env NEIGHBORS=10.0.0.30,10.0.0.40 nocsyscn/frr_conf:0.1 /usr/local/etc/frr/frr.conf

```

## Running

**Prerequisite**: Since the frr-controller uses `apps/v1` deployments, the Kubernetes cluster version should be greater than 1.9.

```sh
# assumes you have a working kubeconfig, not required if operating in-cluster
go build -o frr-controller .
./frr-controller -kubeconfig=$HOME/.kube/config

# create a CustomResourceDefinition
kubectl create -f artifacts/examples/frrcontroller.nocsys.cn_frrs.yaml

# create a custom resource of type Foo
kubectl create -f artifacts/examples/example-frr.yaml
# check deployments created through the custom resource
kubectl get deployments
```

## Cleanup

You can clean up the created CustomResourceDefinition with:
```sh
kubectl delete crd frrs.frrcontroller.nocsys.cn
```
