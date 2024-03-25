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
	kubeClient := k8s.BuildKubernetesClient()
	election.Election(kubeClient)

	allocatedPorts := k8s.GetAllocatedNodePort(kubeClient)

	controller := controller.NewController(kubeClient)

}
