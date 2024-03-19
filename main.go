package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/tiggoins/port-allocator/leaderelection"
)

func main() {
	restConf, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Fatalln("Error in rest config", err)
	}
	client := kubernetes.NewForConfigOrDie(restConf)
	// leaderelection
	leaderelection.Election(client)
}
