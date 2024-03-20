package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/tiggoins/port-allocator/controller"
	"github.com/tiggoins/port-allocator/election"
	"github.com/tiggoins/port-allocator/k8s"
)

func main() {
	restConf, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Fatalln("Error in rest config", err)
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConf)
	// leaderelection
	election.Election(kubeClient)

	namespacePorts := k8s.GetAllocatedNodePort(kubeClient)
	
	controller := controller.NewController(kubeClient)
	
}
