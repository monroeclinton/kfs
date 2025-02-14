package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/monroeclinton/kfs/cmd/apiserver/routes"
	"github.com/monroeclinton/kfs/pkg/apiserver"
	"go.etcd.io/etcd/client/v3"
)

func main() {
	etcdServersFlag := flag.String("etcd-servers", "", "Comma-separated list of etcd server endpoints")
	flag.Parse()

	if *etcdServersFlag == "" {
		log.Fatal("You must provide the --etcd-servers flag")
	}

	endpoints := strings.Split(*etcdServersFlag, ",")

	etcd, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})

	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}

	defer etcd.Close()

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Set("etcd", etcd)
		c.Next()
	})

	podController := &routes.RestStorageController[apiserver.Pod]{
		BasePath: apiserver.PodsRoute,
		FilterFunc: func(pod *apiserver.Pod, key, value string) bool {
			if key == "spec.nodeName" {
				return pod.Spec.NodeName == value
			}
			return false
		},
	}

	nodeController := &routes.RestStorageController[apiserver.Node]{
		BasePath: apiserver.NodesRoute,
	}

	deploymentController := &routes.RestStorageController[apiserver.Deployment]{
		BasePath: apiserver.DeploymentsRoute,
	}

	podController.RegisterRoutes(router)
	nodeController.RegisterRoutes(router)
	deploymentController.RegisterRoutes(router)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "This route does not exist"})
	})

	router.Run("0.0.0.0:6443")
}
