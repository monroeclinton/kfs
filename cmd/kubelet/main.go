package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/monroeclinton/kfs/cmd/kubelet/cni"
	"github.com/monroeclinton/kfs/pkg/apiserver"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

func main() {
	nodeName := flag.String("node-name", "", "Name of the node")
	apiServer := flag.String("apiserver", "", "API server to use")
	flag.Parse()

	if *nodeName == "" {
		log.Fatal("You must provide the --node-name flag")
	}

	if *apiServer == "" {
		log.Fatal("You must provide the --apiserver flag")
	}

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatalf("Failed to connect to containerd: %v", err)
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.TODO(), "k8s.io")

	// TODO: Have this assigned
	network, err := cni.NewNetwork("10.88.0.0/16")
	if err != nil {
		log.Fatalf("Failed to initialize network: %v", err)
	}

	for {
		pods, err := fetchPods(*apiServer, *nodeName)
		if err != nil {
			log.Printf("Failed to fetch pods: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, pod := range pods {
			if pod.Spec.NodeName == *nodeName {
				for _, container := range pod.Spec.Containers {
					containerName := pod.Metadata.Name + "-" + container.Name

					// Check if the image is pulled
					image, err := getImage(ctx, client, container.Image)
					if err != nil {
						log.Printf("Failed to get or pull image %s: %v", container.Image, err)
						continue
					}

					// Check if the container exists
					cont, err := getOrCreateContainer(ctx, client, containerName, image)
					if err != nil {
						log.Printf("Failed to get or create container %s: %v", containerName, err)
						continue
					}

					// Ensure the container is running
					if err := ensureContainerRunning(ctx, network, cont); err != nil {
						log.Printf("Failed to start container %s: %v", containerName, err)
					}
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func fetchPods(apiServer string, nodeName string) ([]apiserver.Pod, error) {
	resp, err := http.Get(apiServer + "/pods?fieldSelector=spec.nodeName=" + nodeName)
	if err != nil {
		return nil, fmt.Errorf("Failed to query pods endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to fetch pods with status code: %d", resp.StatusCode)
	}

	var pods []apiserver.Pod
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		return nil, fmt.Errorf("Failed to decode response: %w", err)
	}

	return pods, nil
}

func getImage(ctx context.Context, client *containerd.Client, imageName string) (containerd.Image, error) {
	image, err := client.GetImage(ctx, imageName)
	if err == nil {
		log.Printf("Image %s already pulled", imageName)
		return image, nil
	}

	// Pull the image if it doesn't exist
	log.Printf("Pulling image %s", imageName)
	image, err = client.Pull(ctx, imageName, containerd.WithPullUnpack)
	if err != nil {
		return nil, fmt.Errorf("Failed to pull image %s: %w", imageName, err)
	}

	return image, nil
}

func getOrCreateContainer(ctx context.Context, client *containerd.Client, containerName string, image containerd.Image) (containerd.Container, error) {
	// Check if the container exists
	container, err := client.LoadContainer(ctx, containerName)
	if err == nil {
		log.Printf("Container %s already exists", containerName)
		return container, nil
	}

	// Create the container if it doesn't exist
	log.Printf("Creating container %s", containerName)
	container, err = client.NewContainer(
		ctx,
		containerName,
		containerd.WithNewSnapshot(containerName+"-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create container %s: %w", containerName, err)
	}

	return container, nil
}

func ensureContainerRunning(ctx context.Context, network *cni.Network, container containerd.Container) error {
	task, err := container.Task(ctx, nil)
	if err != nil {
		// Task not found, create a new one
		log.Printf("No existing task for container %s. Creating a new task.", container.ID())
		task, err = container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
		if err != nil {
			return fmt.Errorf("Failed to create task for container %s: %w", container.ID(), err)
		}
	} else {
		// Task exists, check its status
		status, err := task.Status(ctx)
		if err != nil {
			return fmt.Errorf("Failed to retrieve status for task of container %s: %w", container.ID(), err)
		}

		switch status.Status {
		case containerd.Running:
			log.Printf("Container %s is already running.", container.ID())
			return nil
		case containerd.Stopped, containerd.Paused, containerd.Created, containerd.Unknown:
			log.Printf("Task for container %s is in state %s. Cleaning up and restarting.", container.ID(), status.Status)
			if err := network.DeleteNetwork(ctx, task); err != nil {
				return fmt.Errorf("Failed to delete network for container %s: %w", container.ID(), err)
			}

			if _, err := task.Delete(ctx); err != nil {
				return fmt.Errorf("Failed to delete task for container %s: %w", container.ID(), err)
			}
			task, err = container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
			if err != nil {
				return fmt.Errorf("Failed to recreate task for container %s: %w", container.ID(), err)
			}
		default:
			log.Printf("Unhandled task state %s for container %s. Cleaning up and restarting.", status.Status, container.ID())
			if err := network.DeleteNetwork(ctx, task); err != nil {
				return fmt.Errorf("Failed to delete network for container %s: %w", container.ID(), err)
			}

			if _, err := task.Delete(ctx); err != nil {
				return fmt.Errorf("Failed to delete task for container %s: %w", container.ID(), err)
			}
			task, err = container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
			if err != nil {
				return fmt.Errorf("Failed to recreate task for container %s: %w", container.ID(), err)
			}
		}
	}

	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("Failed to start task for container %s: %w", container.ID(), err)
	}

	if _, err := network.SetupNetwork(ctx, task); err != nil {
		return fmt.Errorf("Failed to delete network for container %s: %w", container.ID(), err)
	}

	log.Printf("Container %s started successfully.", container.ID())
	return nil
}
