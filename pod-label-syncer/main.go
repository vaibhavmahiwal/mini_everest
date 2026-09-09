package main

import (
	"context"
	"fmt"
	"path"

	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	//locate the config file path
	kubeconfig := path.Join(homeDir.HomeDir(), ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	//this create a clientset
	clientset, err := kuberntes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for _, pod := range pods.Items {
		fmt.Println("Found pod:", pod.Name)
	}
}
