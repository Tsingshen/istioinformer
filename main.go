package main

import (
	"context"
	"fmt"
	"log"
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
	WatchResource(cs, ctx.Done())

}

func WatchResource(cs dynamic.Interface, stop <-chan struct{}) {

	var vsResource = schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	var certResource = schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1alpha2",
		Resource: "certificates",
	}

	dynInformer := dynamicinformer.NewDynamicSharedInformerFactory(cs, time.Minute*10)

	vsInformer := dynInformer.ForResource(vsResource).Informer()
	certInformer := dynInformer.ForResource(certResource).Informer()

	certInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			uo, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Println("obj add, interface assert centInformer failed")
				return
			}
			if uo.GetNamespace() == "shencq" {
				log.Printf("cache add cert manager %s/%s\n", uo.GetName(), uo.GetNamespace())
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldC, ok1 := oldObj.(*unstructured.Unstructured)
			newC, ok2 := oldObj.(*unstructured.Unstructured)

			if !ok1 || !ok2 {
				log.Println("obj update, interface assert centInformer failed")
				return
			}

			//
			oldDnsNames, ok1, _ := unstructured.NestedStringSlice(oldC.Object, "spec", "dnsNames")
			if ok1 {
				log.Printf(" %s/%s updated with DnsNames=%v\n", oldC.GetName(), oldC.GetNamespace(), oldDnsNames)
			}

			newSomeStr, ok2, err := unstructured.NestedStringSlice(newC.Object, "spec", "gateway")
			if err != nil || !ok2 {
				log.Printf("newC get gateway not found or err: %v\n", err)
			}

			if ok2 {
				log.Printf("newC get gateway string = %v\n", newSomeStr)
			}

		},
	})

	vsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			uo, ok := obj.(*unstructured.Unstructured)
			if !ok {
				log.Println("vs add, interface assert vsInformer failed")
				return
			}
			if uo.GetNamespace() == "shencq" {
				log.Printf("cache add vs %s/%s\n", uo.GetName(), uo.GetNamespace())
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldV, ok1 := oldObj.(*unstructured.Unstructured)
			newV, ok2 := oldObj.(*unstructured.Unstructured)

			if !ok1 || !ok2 {
				log.Println("vs update, interface assert vsInformer failed")
				return
			}

			//
			oldGw, ok1, _ := unstructured.NestedStringSlice(oldV.Object, "spec", "gateways")
			_, _, _ = unstructured.NestedStringSlice(newV.Object, "spec", "gateways")
			if ok1 {
				log.Printf("vs %s/%s updated with gw=%v\n", oldV.GetName(), oldV.GetNamespace(), oldGw)
			}

		},
	})

	dynInformer.Start(stop)
	dynInformer.WaitForCacheSync(stop)

	<-stop
	fmt.Println("vsInformer received stopped signal, exitting...")

}
