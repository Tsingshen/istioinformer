package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tsingshen/k8scrd/client"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

func main() {
	cs := client.GetDynamicClient()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer cancel()
	WatchVsResource(cs, ctx.Done())

}

func WatchVsResource(cs dynamic.Interface, stop <-chan struct{}) {

	var vsResource = schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	dynInformer := dynamicinformer.NewDynamicSharedInformerFactory(cs, time.Minute*10)
	informer := dynInformer.ForResource(vsResource).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			uo, ok := obj.(*unstructured.Unstructured)
			if !ok {
				panic("uo get nothing")
			}
			if uo.GetNamespace() == "shencq" {
				fmt.Printf("cache add vs %s/%s\n", uo.GetName(), uo.GetNamespace())
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldV, ok1 := oldObj.(*unstructured.Unstructured)
			newV, ok2 := oldObj.(*unstructured.Unstructured)

			if !ok1 || !ok2 {
				return
			}

			//
			oldGw, ok1, _ := unstructured.NestedStringSlice(oldV.Object, "spec", "gateways")
			_, _, _ = unstructured.NestedStringSlice(newV.Object, "spec", "gateways")
			if ok1 {
				fmt.Printf("%v\n", oldGw)
			}

		},
	})

	dynInformer.Start(stop)
	dynInformer.WaitForCacheSync(stop)

	select {
	case <-stop:
		fmt.Println("vsInformer received stopped signal, exitting...")
		return
	}

}
