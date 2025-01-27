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

While relational databases like PostgreSQL

### Watching Keys

### Strong Serializability

#### TODO: Explain serializability, linearizability, MVCC, and why strict serializability is necessary.

## Control Plane

## Data Plane
