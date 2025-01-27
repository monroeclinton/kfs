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

	router.GET(apiserver.PodsRoute, routes.GetPods)
	router.POST(apiserver.PodsRoute, routes.PostPods)
	router.PATCH(apiserver.PodsRoute+"/:name", routes.PatchPods)
	router.GET(apiserver.NodesRoute, routes.GetNodes)
	router.POST(apiserver.NodesRoute, routes.PostNodes)
	router.PATCH(apiserver.NodesRoute+"/:name/status", routes.PatchNodeStatus)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "This route does not exist"})
	})

	router.Run("0.0.0.0:6443")
}
