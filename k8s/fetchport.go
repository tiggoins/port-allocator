package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type NamespacePort struct {
	Namespace string
	NodePorts []int
}

type NamespacePortALL []NamespacePort

func GetAllocatedNodePort(kubeClient *kubernetes.Clientset) NamespacePortALL {
	var npa NamespacePortALL

	services, err := kubeClient.CoreV1().Services(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Fatalln("error list service in the cluster", err)
	}

	// 遍历服务列表
	for _, service := range services.Items {
		// 只处理 NodePort 类型的服务
		if service.Spec.Type == corev1.ServiceTypeNodePort {
			// 获取命名空间对应的端口列表，并追加到 s 中
			var ports []int
			for _, port := range service.Spec.Ports {
				ports = append(ports, int(port.NodePort))
			}
			np := NamespacePort{Namespace: service.Namespace, NodePorts: ports}
			npa = append(npa, np)
		}
	}

	return npa
}
