package routes

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/monroeclinton/kfs/pkg/apiserver"

	"github.com/gin-gonic/gin"
	"go.etcd.io/etcd/client/v3"
	uuid "k8s.io/apimachinery/pkg/util/uuid"
)

func GetNodes(c *gin.Context) {
	etcd := c.MustGet("etcd").(*clientv3.Client)

	nodes := []apiserver.Node{}

	response, err := etcd.Get(context.TODO(), apiserver.NodesRegistry, clientv3.WithPrefix())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, kv := range response.Kvs {
		var node apiserver.Node
		if err := json.Unmarshal(kv.Value, &node); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		nodes = append(nodes, node)
	}

	c.IndentedJSON(http.StatusOK, nodes)
}

func PostNodes(c *gin.Context) {
	var node apiserver.Node

	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	node.Metadata.Uid = uuid.NewUUID()

	etcd := c.MustGet("etcd").(*clientv3.Client)

	jsonBytes, err := json.Marshal(node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = etcd.Put(context.TODO(), apiserver.NodesRegistry+node.Metadata.Name, string(jsonBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.IndentedJSON(http.StatusCreated, node)
}

func PatchNodeStatus(c *gin.Context) {
	name := c.Param("name")

	var nodeStatus apiserver.NodeStatus

	if err := c.ShouldBindJSON(&nodeStatus); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	etcd := c.MustGet("etcd").(*clientv3.Client)

	response, err := etcd.Get(context.TODO(), apiserver.NodesRegistry+name, clientv3.WithPrefix())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var node apiserver.Node
	if err := json.Unmarshal(response.Kvs[0].Value, &node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	node.Status = nodeStatus

	jsonBytes, err := json.Marshal(node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = etcd.Put(context.TODO(), apiserver.NodesRegistry+node.Metadata.Name, string(jsonBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.IndentedJSON(http.StatusOK, node)
}
