package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type NamespacePort struct {
	Namespace string
	NodePorts []int32
}

func ListNamespaces(kubeClient *kubernetes.Clientset) []string {
	var namespaces []string
	
	ns, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Fatalln("error happened when list namespaces from cluster.")
	}

	for _, n := range ns.Items {
		namespaces = append(namespaces, n.Name)
	}

	return namespaces
}

func GetNamespacedAllocatedNodePort(kubeClient *kubernetes.Clientset, namespace string) NamespacePort {
	var np NamespacePort

	nodePortSelector := labels.Set{"type": "NodePort"}.AsSelector()

	services, err := kubeClient.CoreV1().Services(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		LabelSelector: nodePortSelector.String(),
	})
	if err != nil {
		klog.Fatalln("error list service in the cluster", err)
	}

	// 遍历服务列表
	for _, service := range services.Items {
		// 获取命名空间对应的端口列表，并追加到 s 中
		var ports []int32
		for _, port := range service.Spec.Ports {
			ports = append(ports, port.NodePort)
		}
		np = NamespacePort{Namespace: service.Namespace, NodePorts: ports}
	}

	return np
}
