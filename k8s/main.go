package k8s

import (
	"os"
	"fmt"
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	PodDetails *PodInfo
)

type PodInfo struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func GetIngressPod(kubeClient *kubernetes.Clientset) error {
	podName := os.Getenv("POD_NAME")
	podNs := os.Getenv("POD_NAMESPACE")

	if podName == "" || podNs == "" {
		return fmt.Errorf("unable to get POD information (missing POD_NAME or POD_NAMESPACE environment variable")
	}

	pod, err := kubeClient.CoreV1().Pods(podNs).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get POD information: %v", err)
	}

	PodDetails = &PodInfo{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
	}

	pod.ObjectMeta.DeepCopyInto(&PodDetails.ObjectMeta)

	return nil
}