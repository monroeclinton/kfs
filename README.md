# Kubernetes from Scratch

### Work in progress, not near completion

The purpose of this project is to create a basic implementation of Kubernetes in order to understand the different components of Kubernetes and the design choices that went into them. This guide expects you have some knowledge of what Kubernetes is and how to use it going in.

This project will implement enough functionality so that:

- Given a deployment, can create and run the pods on appropriate nodes.
- Given a service expose and load balance the appropriate application.

In short, the ability to run `kubectl apply -f` on [example-deployment.yaml](example-deployment.yaml).

## Components

For each of these Kubernetes components we will build a subsection of it for our implementation.

#### Control Plane

- kube-apiserver: The API server used by other components to read and manage the cluster's state.
- kube-scheduler: Assigns pods to nodes.
- kube-controller-manager: A collection of controllers that monitors and adjusts resources to match specifications.

#### Data Plane

These components run a node.

- kubelet: Manages containers based on assigned Pods.
- kube-proxy: Manages the network traffic to containers.
- CRI: The [Container Runtime Interface](https://kubernetes.io/docs/concepts/architecture/cri/) allows the kubelet to manage containers through a
  container runtime like [containerd](https://containerd.io/) or [cri-o](https://cri-o.io/) over a
  gRPC API.
- CNI: The [Container Network Interface](https://www.cni.dev/) is used to configure networking for containers through
  a configuration file and CNI plugins.

## Design Choices

## Storage

Kubernetes uses the key-value store [etcd](https://etcd.io/) for storing the state of the cluster.

This might seem unsual, the resources listed in the [API Specification](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/) fit a relational model well. Looking through the specification, many resources reference other resources. The API uses the standard CRUD operations for these resources and relationships. So why not use a relational database like PostgreSQL?

The short answer is that etcd offers high availability, watching keys, [strict serializability](https://jepsen.io/consistency/models/strong-serializable) with the simplicity of a key-value store. PostgreSQL can be configured to have these things (except strict serializability), but it requires more work.

### High Availability

While relational databases like PostgreSQL do have [high availability](https://www.postgresql.org/docs/current/high-availability.html) modes, they tend to require more configuration and setup steps than an etcd cluster. Some of these tools such as [stolon](https://github.com/sorintlab/stolon) or [Patroni](https://github.com/patroni/patroni) use etcd to handle cluster state.

### Watching Keys

Kubernetes needs to be able to watch for updates to a particular key in etcd. When an update is
made, for example to the key-value pair containing a deployment resource, Kubernetes needs to take action
such that the desired states becomes the actual state.

The `watch` operation in etcd provides the ability to easily do this. To achieve the same
functionality in something like this in PostgreSQL requires more work.

```go
// For example in Go, this creates a channel where updates are sent
watchChan := client.Watch(context.Background(), key)
```

### Strict Serializability

The concept of strict serializability and its implementation in etcd is core to Kubernetes. Strict
serializability simply means that all operations take place in a serial (one operation at a time) manner,
and once an operation takes effect, it is visible to all subsequent operations.

Most traditional databases do not have strict serializability due to the way they handle [transaction isolation](https://www.postgresql.org/docs/current/transaction-iso.html).
Transactions modifying data, but this is not visible to other transactions until they commit,
meaning it does not meet the level of strict serializability.

To understand why this might be a problem, let's consider an example in which strict serializability
does not exist. Suppose you have three etcd nodes, two followers and a leader. You have two
processes, Process One that creates a deployment and Process Two that scales the deployment.
Process One connects to the leader and creates a deployment, it then alerts Process Two of the
deployment. The followers have not yet replicated the deployment from the leader. Process Two
connects to a follower, but cannot get the deployment, causing an error.

This scenero of a stale read is possible in databases without the consistency guarantees of strict
serializability.

## Control Plane

The control plane manages the cluster state, schedules workloads, and provides an API to manage
this. Each component of the control plane was built for resilience and to scale independently.

### kube-apiserver

The API Server is made up of several components that work together to form an interface to manage
the resources in the cluster. Some of these components include:

- API Server: The RESTful interface, handles authentication, and validation.
- Aggregator: Allows for extending Kubernetes with custom APIs by implmeenting the `APIService`.
- Controllers: Several controllers are run with the API server, including `kubernetesservice` and `systemnamespaces`.
- The `kubernetesservice` controller ensures the service named `kubernetes` routes to the API server.
  The `systemnamespaces` controller ensures the default namespaces exist (kube-system, kube-public, default, and kube-node-lease).
- Endpoint Reconciler: This is for when there are multiple API servers, it sets the proper endpoints
  on the `kubernetes` service.

I recommend the blog post by [sobyte](https://www.sobyte.net/post/2022-07/kube-apiserver/) for a
more in-depth overview.

### kube-scheduler

The scheduler assigns pods to nodes based on resource availability, policies, affinity, and taints. When
a pod is created it goes through two cycles in order to be assigned to a node.

- Scheduling: The scheduler filters out nodes, scores nodes based on resources,
- Binding: The scheduler sets the `nodeName` on the pod, assigning that pod to a node. It handles
  failures, putting the pod back in the scheduling queue if necessary.

I recommend the [blog post](https://www.awelm.com/posts/kube-scheduler/) by Akila Welihinda for more.

### kube-controller-manager

The controller manager is made up of many different controllers that provide the default
functionality to Kubernetes. These controllers watch the API server, and reconcile changes so that
the cluster's actual state matches the desired state.

These are some of those controllers, and what they do. Here is the [full list](https://github.com/kubernetes/kubernetes/blob/925cf7db71c5e36072f99e8b7129523f659ee3a1/cmd/kube-controller-manager/names/controller_names.go#L44).

- EndpointSliceController: Ensures that endpoint slices point to the proper pods via endpoints.
  By default there are a maximum of 100 endpoints per endpoint slice. This is to limit the size of
  endpoint slices, if there was no limit the updates could become quite large.
- PodGarbageCollectorController: Deletes pods that are terminated or who's node no longer exists.
- DeploymentController: Creates and manages replica sets across deployments.
- ReplicaSetController: Ensures that the proper number of pods are running for a replica set.
- NodeIpamController: Allocates the IP range to nodes that will be used by pods.
- NodeLifecycleController: Monitors that nodes are updating their health, updates taints based on
  health checks, and updates pod `Ready` condition for pods on the node if the node is unhealthy.

## Data Plane
