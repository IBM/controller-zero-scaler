## controller-zero-scaler

This project provides automatic scaling of Kubernetes controller deployments to zero based Kubernetes API server activity. When API server activity which is relevant to a particular controller ceases for a time, the controller deployment will be scaled down to zero. The deployment remains scaled to zero until relevant API server activity is detected, at which point the controller-zero-scaler will restore the controller to its original scale. The controller will remain running until it is once again determined to be idle. 

### Status

This project is under active development, but is suitable for demonstration purposes. 

* Scale of statefulsets is implemented. 
* Scale of deployments is not implemented
* Watch of owned objects is not fully implemented
* Annotation names may change. Backward compatiblity is not guaranteed. 
* Prior scale is not respected, scale up always goes to 1

### Quick Start (Minikube)

Use requires deploying the zero-scale-controller to a running Kubernetes cluster along with annotating the individual controller deployments. 

To deploy the controller, clone this repository, then issue the commands: 

```
# Target Docker at the minikube Docker engine instnace
eval $(minikube docker-env)

# Build the Docker image
make docker-build IMG=controller-zero-scaler:latest

# Deploy
make deploy
```

Deploy a controller which will be scaled on idle. Clone the project [https://github.com/banzaicloud/istio-operator](https://github.com/banzaicloud/istio-operator) and then run `make deploy`. 

Annotate the example controller:
```
kubectl annotate -n istio-system statefulset -l \
    controller-tools.k8s.io='1.0' \
    controller-zero-scaler/idleTimeout='5s' \
    controller-zero-scaler/watchedKinds='[{"apiVersion": "istio.banzaicloud.io/v1beta1", "Kind": "Istio"}]' \
    --overwrite
```

Watch the deployment to observe scale to zero: 

```
kubectl get statefulsets -n istio-system --watch
```

Touch a resource to see scale. From the istio-operator directory: 

```
kubectl apply -f config/samples/istio_v1beta1_istio.yaml
```

### How It Works

The controller-zero-scaler watches for Statefulsets or Deployments which have annotations describing the API types relevant to that particular controller as well as an idle duration specifier. If no API server events occur which are relevant to the annotated controller and the indicated idle duration has elapsed, the deployment or statefulset will be scaled to zero. Future API server activity to the same set of types will result in restoring the prior scale. 

The following annotation attributes are recognized: 

controller-zero-scaler/idleDuration: determines how long  to wait before determining that a controller deployment is idle
controller-zero/watchedKinds: contains a json encoded list of watched kinds
controller-zero/ownedKinds: contains a json encoded list of kinds which are owned by this controller

For example: 
```
    # Indicates this deployment should remain running until 60 seconds has elapsed since the last API server event to either the watched or owned kinds
    controller-zero/zeroScaleTimeout: "60s"

    # A JSON encoded list of those API types which are relevant to this controller
    controller-zero/watchedKinds: |
      [{"apiVersion": "apps/v1", "Kind": "Deployment"}]

    # A JSON encoded list of those API types which this controller owns
    controller-zero/ownedKinds: "[]"
```

## Alternatives

It is also possible to implement zero scaling of controllers using a combination of the KNative API event source along with a KNative serve controller implementation. 

Which approach is best is likely situation dependent: 

Use controller-zero-scaler when:
* The controller is already deployed to the target system
* No code changes to the controller are desired
* KNative runtime components are not available

Use KNative when: 
* KNative is available in the target runtime environment
* Controller code changes are acceptable
* The controller is responding to multiple KNative event sources

## Limitations

The zero-scale-controller works for many commonly available controller implementations, but might not work for all possible situations. 

Some examples include: 

* controllers which watch for events in systems other than the Kubernetes API server
* controllers which incorporate extensive background processing, for example internal timer functions