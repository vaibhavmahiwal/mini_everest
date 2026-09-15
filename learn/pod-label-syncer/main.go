package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
)

func main() {
	// 1. Locate the config file path
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	// 2. Build REST config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// 3. Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// 4. Query all pods across all namespaces
	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	// 5. Print results
	for _, pod := range pods.Items {
		fmt.Printf("[%s] Found pod: %s\n", pod.Namespace, pod.Name)
	}

	//this is a real time informer to watch for pod events
	//usign kubernetes informer
	//

	//every 30 sec the informer resync the chache with the api server
	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset, 30*time.Second, informers.WithNamespace("default"),
	)

	//this opens an http stream connection to the api server
	//and watch

	podInformer := factory.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) { //this is called when a new pos is created
			fmt.Println("add event")
		},
		UpdateFunc: func(oldObj, newObj interface{}) { //when a pod is updated
			fmt.Println("update event")
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	fmt.Println("informer cache synced waiting for live events in default")
	<-stopCh

	//adding a workqueue decouple events from processing
	//right now our event handlers do work directly inside the
	//call back bad idea slow reconcil blocks the informer and can cause miss events
	//

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
    
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}){
			key,_:=cache.MetaNamespaceKeyFunc(obj)
			queue.Add(key)
		},
		UpdateFunc: func(oldObj,newObj interface{}){
			key,_:=cache.MetaNamespaceKeyFunc(newObj)
			queue.Add(key)
		},
	})

	func runWorker(queue workqueue.RateLimitingInterface ,lister cache.Indexer,clientset *kubernetes.Clientset){
        for {
			key,shutdown:=queue.Get()
			if shutdown{
				return
			}
			func(){
				defer queue.Done(key)
					err := syncPod(key.(string), lister, clientset)
					if err!=nil{
						queue.AddRateLimited(key)
						return
					}
					queue.Forget(key)
			}()
		}
	}
    
}

