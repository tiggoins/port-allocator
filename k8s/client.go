package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func BuildKubernetesClient() *kubernetes.Clientset {
	var restConf *rest.Config

	restConf, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Warningf("cannot build rest config,%s", err)
		restConf, err = rest.InClusterConfig()
		if err != nil {
			klog.Fatalln("still cannot build rest config", err)
		}
	}
	
	kubeClient := kubernetes.NewForConfigOrDie(restConf)

	return kubeClient
}
