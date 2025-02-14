package apiserver

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

var EtcdPrefix = "/registry"

var NodesRoute = "/nodes"
var PodsRoute = "/pods"

var NodeKind = "Node"
var PodKind = "Pod"

type ObjectMeta struct {
	Uid    types.UID `json:"uid"`
	Name   string    `json:"name"`
	Labels []string  `json:"labels"`
}

type LabelSelector struct {
	MatchLabels map[string]string `json:"matchLabels"`
}

type Node struct {
	ApiVersion string     `json:"apiVersion"`
	Kind       string     `json:"kind"`
	Metadata   ObjectMeta `json:"metadata"`
	Status     NodeStatus `json:"status"`
}

type NodeStatus struct {
	Addresses []NodeAddress `json:"addresses"`
}

type NodeAddress struct {
	Address string `json:"address"`
	Type    string `json:"type"` // Address type, one of Hostname, ExternalIP or InternalIP.
}

type Container struct {
	Image string `json:"image"`
	Name  string `json:"name"`
}

type PodSpec struct {
	Containers []Container `json:"containers"`
	NodeName   string      `json:"nodeName"`
}

type PodStatus struct {
	PodIP     *string     `json:"podIp"`
	StartTime metav1.Time `json:"startTime"`
}

type Pod struct {
	ApiVersion string     `json:"apiVersion"`
	Kind       string     `json:"kind"`
	Metadata   ObjectMeta `json:"metadata"`
	Spec       PodSpec    `json:"spec"`
	Status     PodStatus  `json:"status"`
}
