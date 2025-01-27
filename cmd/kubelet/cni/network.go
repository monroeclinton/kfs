package cni

import (
	"context"
	"path/filepath"

	"github.com/containerd/containerd"
	gocni "github.com/containerd/go-cni"
)

type Network struct {
	cni gocni.CNI
}

func NewNetwork(subnet string) (*Network, error) {
	cni, err := gocni.New(
		gocni.WithPluginConfDir(gocni.DefaultNetDir),
		gocni.WithPluginDir([]string{gocni.DefaultCNIDir}),
		gocni.WithInterfacePrefix(gocni.DefaultPrefix),
	)

	if err != nil {
		return nil, err
	}

	if err := cni.Load(gocni.WithLoNetwork, gocni.WithDefaultConf); err != nil {
		return nil, err
	}

	return &Network{
		cni,
	}, nil
}

func namespace(pid uint32) string {
	return filepath.Join("/proc", string(pid), "ns", "net")
}

func (n *Network) SetupNetwork(ctx context.Context, task containerd.Task) (*gocni.Result, error) {
	return n.cni.Setup(ctx, task.ID(), namespace(task.Pid()))
}

func (n *Network) DeleteNetwork(ctx context.Context, task containerd.Task) error {
	return n.cni.Remove(ctx, task.ID(), namespace(task.Pid()))
}
