package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"

	"github.com/spf13/pflag"
	"github.com/tiggoins/port-allocator/config"
	"github.com/tiggoins/port-allocator/election"
	"github.com/tiggoins/port-allocator/k8s"
	"github.com/tiggoins/port-allocator/queue"
	"github.com/tiggoins/port-allocator/store"
	"github.com/tiggoins/port-allocator/webhook"
)

const (
	configFile string = "port-range.yaml"
)

func NewServerFlagSet() *pflag.FlagSet {
	serverFlags := pflag.NewFlagSet("server", pflag.ExitOnError)
	serverFlags.String("tls-cert-file", "", "Path to the certificate file (MUST specify)")
	serverFlags.String("tls-key-file", "", "Path to the key file (MUST Specify)")
	serverFlags.IntP("port", "p", 443, "Port to listen on (default to 443)")

	return serverFlags
}

func main() {
	klog.InitFlags(nil)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	stopCh := make(chan struct{})

	// 从配置文件中加载配置
	yamlConfig := config.LoadConfigFromFile(configFile)
	// 初始化k8s客户端
	k8sClient := k8s.BuildKubernetesClient()
	// 取出当前Pod的信息供leaderelection使用
	k8s.GetPodInfo(k8sClient)
	// 新建底层的存储，用于维护已分配和namespace的nodePort定义
	s := store.NewNamespaceNodePortConfig()
	// 运行leaderelection
	go election.Election(k8sClient)

	// 1. 从yaml中载入配置
	for _, config := range yamlConfig {
		s.AddNamespace(config.Namespace, config.PortStart, config.PortEnd)
	}

	// 2. list namespaces and add allocated port to store
	namespaces := k8s.ListNamespaces(k8sClient)
	for _, namespace := range namespaces {
		allocatedPorts := k8s.GetNamespacedAllocatedNodePort(k8sClient, namespace)
		s.AddPortToNamespace(allocatedPorts.Namespace, allocatedPorts.NodePorts)
	}

	// 3. startqueue to watch the delete event of service
	q := queue.NewQueue(k8sClient, stopCh, s)
	go q.Run()

	// 4. start webhook to mutating the creation and update of incoming service
	hookServer := webhook.NewServer(ctx, *NewServerFlagSet(), s)
	hookServer.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-stopCh

	// stop the server gracefully
	if err := hookServer.Shutdown(); err != nil {
		klog.Error(err)
	}
	os.Exit(0)
}
