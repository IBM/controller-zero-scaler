## Background

Kubernetes is built upon an extensible declarative state model implemeted by the API server combined with a runtime component known as a controller. A controller watches the API server and responds to changes to the declarative state model, reconciling or reflects changes to the state model into other objects. 

Each object within the Kubernetes API server is versioned and typed via the `kind` attribute. Individual controller implementations will generally watch for changes to specific kinds. 

The Kubernetes platform includes a controller collective referred to as the 'controller manager' which provides the functionality of the core API server types such as Deployments, Pods, etc. Additional controllers usually run on the same Kubernetes cluster which they are watching, and are deployed using the Deployment or Statefulset constructs. 

When a given controller implementation is started, it will reconcile the state of any interesting objects and then enter an event wait state. Once in the event wait state, the controller remains idle until other API server events occur. 

## Zero Scaler Implementation

The zero scaler implementation relies on the fact that most controllers react primarily to API server events and are deployed as either Deployments or Statefulsets, and upon starting, the controller process will reconcile all relevant objects. 

