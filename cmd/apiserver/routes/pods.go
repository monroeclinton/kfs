package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/monroeclinton/kfs/pkg/apiserver"

	"github.com/gin-gonic/gin"
	"github.com/imdario/mergo"
	"go.etcd.io/etcd/client/v3"
	uuid "k8s.io/apimachinery/pkg/util/uuid"
)

func GetPods(c *gin.Context) {
	etcd := c.MustGet("etcd").(*clientv3.Client)

	pods := []apiserver.Pod{}

	response, err := etcd.Get(context.TODO(), apiserver.PodsRegistry, clientv3.WithPrefix())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	fieldSelector := c.Query("fieldSelector")
	selector := strings.Split(fieldSelector, "=")
	if len(fieldSelector) != 0 && len(selector) != 2 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "fieldSelector must use format: foo.bar=baz",
		})
		return
	}

	for _, kv := range response.Kvs {
		var pod apiserver.Pod
		if err := json.Unmarshal(kv.Value, &pod); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		shouldAppend := true

		// Check fieldSelector
		if len(selector) == 2 {
			switch selector[0] {
			case "spec.nodeName":
				shouldAppend = pod.Spec.NodeName == selector[1]
			}
		}

		if shouldAppend {
			pods = append(pods, pod)
		}
	}

	c.IndentedJSON(http.StatusOK, pods)
}

func PostPods(c *gin.Context) {
	var pod apiserver.Pod

	if err := c.ShouldBindJSON(&pod); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	pod.Metadata.Uid = uuid.NewUUID()

	etcd := c.MustGet("etcd").(*clientv3.Client)

	jsonBytes, err := json.Marshal(pod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = etcd.Put(context.TODO(), apiserver.PodsRegistry+pod.Metadata.Name, string(jsonBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.IndentedJSON(http.StatusCreated, pod)
}

func PatchPods(c *gin.Context) {
	name := c.Param("name")

	var patchPod apiserver.Pod

	if err := c.ShouldBindJSON(&patchPod); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	etcd := c.MustGet("etcd").(*clientv3.Client)

	response, err := etcd.Get(context.TODO(), apiserver.PodsRegistry+name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var existingPod apiserver.Pod
	if err := json.Unmarshal(response.Kvs[0].Value, &existingPod); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Merge the patch with the existing pod
	if err := mergo.Merge(&existingPod, patchPod, mergo.WithOverride); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	jsonBytes, err := json.Marshal(existingPod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = etcd.Put(context.TODO(), apiserver.PodsRegistry+name, string(jsonBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.IndentedJSON(http.StatusOK, existingPod)
}
