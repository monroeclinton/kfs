package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/monroeclinton/kfs/pkg/apiserver"

	"github.com/gin-gonic/gin"
	"github.com/imdario/mergo"
	"go.etcd.io/etcd/client/v3"
	uuid "k8s.io/apimachinery/pkg/util/uuid"
)

type RestStorageController[T any] struct {
	BasePath   string
	FilterFunc func(obj *T, key, value string) bool
}

func (ctrl *RestStorageController[T]) RegisterRoutes(router *gin.Engine) {
	router.GET(ctrl.BasePath, ctrl.Get)
	router.POST(ctrl.BasePath, ctrl.Post)
	router.PATCH(ctrl.BasePath+"/:name", ctrl.Patch)
	router.PATCH(ctrl.BasePath+"/:name/status", ctrl.PatchStatus)
}

func (ctrl *RestStorageController[T]) Get(c *gin.Context) {
	etcd := c.MustGet("etcd").(*clientv3.Client)
	items := []T{}

	response, err := etcd.Get(context.TODO(), apiserver.EtcdPrefix+ctrl.BasePath+"/", clientv3.WithPrefix())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fieldSelector := c.Query("fieldSelector")
	selector := strings.Split(fieldSelector, "=")
	if len(fieldSelector) != 0 && len(selector) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fieldSelector must use format: foo.bar=baz"})
		return
	}

	for _, kv := range response.Kvs {
		var item T
		if err := json.Unmarshal(kv.Value, &item); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		shouldAppend := true
		if len(selector) == 2 && ctrl.FilterFunc != nil {
			shouldAppend = ctrl.FilterFunc(&item, selector[0], selector[1])
		}

		if shouldAppend {
			items = append(items, item)
		}
	}

	c.IndentedJSON(http.StatusOK, items)
}

func (ctrl *RestStorageController[T]) Post(c *gin.Context) {
	var item T

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add unique ID
	val := reflect.ValueOf(&item).Elem()
	if field := val.FieldByName("Metadata"); field.IsValid() {
		metadata := field.Addr().Interface().(*apiserver.ObjectMeta)
		metadata.Uid = uuid.NewUUID()
	}

	etcd := c.MustGet("etcd").(*clientv3.Client)
	jsonBytes, err := json.Marshal(item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = etcd.Put(context.TODO(), apiserver.EtcdPrefix+ctrl.BasePath+"/"+reflect.ValueOf(item).FieldByName("Metadata").FieldByName("Name").String(), string(jsonBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusCreated, item)
}

func (ctrl *RestStorageController[T]) Patch(c *gin.Context) {
	name := c.Param("name")
	var patchItem T

	if err := c.ShouldBindJSON(&patchItem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	etcd := c.MustGet("etcd").(*clientv3.Client)
	response, err := etcd.Get(context.TODO(), apiserver.EtcdPrefix+ctrl.BasePath+"/"+name)
	if err != nil || len(response.Kvs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	var existingItem T
	if err := json.Unmarshal(response.Kvs[0].Value, &existingItem); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Merge the patch with the existing item
	if err := mergo.Merge(&existingItem, patchItem, mergo.WithOverride); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	jsonBytes, err := json.Marshal(existingItem)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = etcd.Put(context.TODO(), apiserver.EtcdPrefix+ctrl.BasePath+"/"+name, string(jsonBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, existingItem)
}

func (ctrl *RestStorageController[T]) PatchStatus(c *gin.Context) {
	name := c.Param("name")
	var patchStatus map[string]interface{}

	if err := c.ShouldBindJSON(&patchStatus); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	etcd := c.MustGet("etcd").(*clientv3.Client)
	response, err := etcd.Get(context.TODO(), apiserver.EtcdPrefix+ctrl.BasePath+"/"+name)
	if err != nil || len(response.Kvs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	var existingItem T
	if err := json.Unmarshal(response.Kvs[0].Value, &existingItem); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	val := reflect.ValueOf(&existingItem).Elem()
	statusField := val.FieldByName("Status")
	if !statusField.IsValid() || statusField.Kind() != reflect.Struct {
		c.JSON(http.StatusInternalServerError, gin.H{"error": reflect.Ptr})
		return
	}

	statusBytes, err := json.Marshal(patchStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := json.Unmarshal(statusBytes, statusField.Addr().Interface()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	jsonBytes, err := json.Marshal(existingItem)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = etcd.Put(context.TODO(), apiserver.EtcdPrefix+ctrl.BasePath+"/"+name, string(jsonBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, existingItem)
}
