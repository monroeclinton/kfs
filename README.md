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

## Control Plane

## Data Plane
