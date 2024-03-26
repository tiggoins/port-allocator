package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/tiggoins/port-allocator/config"
	"github.com/tiggoins/port-allocator/controller"
	"github.com/tiggoins/port-allocator/election"
	"github.com/tiggoins/port-allocator/k8s"
	"github.com/tiggoins/port-allocator/store"
	"github.com/tiggoins/port-allocator/webhook"
)

const (
	configFile string = "port-range.yaml"
)

type Controller struct {
	queue   *controller.Queue
	results config.Results
	client  *kubernetes.Clientset
	store   *store.NamespaceNodePortConfig
	webhook *webhook.Mutator
}

func main() {
	klog.InitFlags(nil)
	// 从配置文件中加载配置
	yamlConfig := config.LoadConfigFromFile(configFile)
	// 初始化k8s客户端
	c := k8s.BuildKubernetesClient()
	// 新建底层的存储，用于维护已分配和namespace的nodePort定义
	s := store.NewNamespaceNodePortConfig()
	// 初始化webhook
	w := webhook.NewMutator()
	// 运行leaderelection
	go election.Election(c)

	controller := &Controller{
		client:  c,
		store:   s,
		results: yamlConfig,
		webhook: w,
	}
}
