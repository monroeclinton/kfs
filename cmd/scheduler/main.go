package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/monroeclinton/kfs/pkg/apiserver"
)

func getJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return err
	}

	return nil
}

func fetchNodes(apiServer string) []string {
	var nodes []apiserver.Node
	if err := getJSON(apiServer+apiserver.NodesRoute, &nodes); err != nil {
		log.Printf("Error fetching nodes: %v", err)
		return nil
	}

	nodesList := make([]string, len(nodes))
	for i, node := range nodes {
		nodesList[i] = node.Metadata.Name
	}
	return nodesList
}

func schedulePods(apiServer string, nodes []string) {
	if len(nodes) == 0 {
		log.Println("No nodes available")
		return
	}

	var pods []apiserver.Pod
	if err := getJSON(apiServer+apiserver.PodsRoute+"?fieldSelector=spec.nodeName=", &pods); err != nil {
		log.Printf("Error fetching pods: %v", err)
		return
	}

	for i, pod := range pods {
		nodeName := nodes[i%len(nodes)]

		pod.Spec.NodeName = nodeName

		patch, err := json.Marshal(pod)
		if err != nil {
			log.Print(err)
			continue
		}

		url := fmt.Sprintf("%s%s/%s", apiServer, apiserver.PodsRoute, pod.Metadata.Name)
		req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(patch))
		if err != nil {
			log.Print(err)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Print(err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("Scheduled pod %s to node %s", pod.Metadata.Name, nodeName)
		} else {
			log.Printf("Failed to schedule pod %s to node %s", pod.Metadata.Name, nodeName)
		}
	}
}

func main() {
	apiServer := flag.String("apiserver", "", "API server to use")
	flag.Parse()

	if *apiServer == "" {
		log.Fatal("You must provide the --apiserver flag")
	}

	// TODO: Refresh nodes every so often
	nodes := fetchNodes(*apiServer)

	for {
		schedulePods(*apiServer, nodes)
		time.Sleep(time.Second)
	}
}
