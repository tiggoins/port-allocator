package store

import (
	"fmt"
	"sync"
)

type NamespaceNodePortConfig struct {
	NamespaceConfigs map[string]*NamespaceConfig
	lock             sync.Mutex
}

type NamespaceConfig struct {
	NodePortRange  PortRange
	AllocatedPorts map[int]bool
}

type PortRange struct {
	Min int
	Max int
}

func NewNamespaceNodePortConfig() *NamespaceNodePortConfig {
	return &NamespaceNodePortConfig{
		NamespaceConfigs: make(map[string]*NamespaceConfig),
	}
}

func (c *NamespaceNodePortConfig) GetNamespace(namespace string) (*NamespaceConfig, bool) {
	nsConfig, ok := c.NamespaceConfigs[namespace]
	if !ok {
		return &NamespaceConfig{}, false
	}

	return nsConfig, true
}

func (c *NamespaceNodePortConfig) AddNamespace(namespace string, minPort, maxPort int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否已存在
	if _, ok := c.GetNamespace(namespace); ok {
		return fmt.Errorf("namespace %s already exists", namespace)
	}

	// 添加命名空间配置
	c.NamespaceConfigs[namespace] = &NamespaceConfig{
		NodePortRange: PortRange{
			Min: minPort,
			Max: maxPort,
		},
		AllocatedPorts: make(map[int]bool), // 初始化为map[int]bool类型
	}
	return nil
}

func (c *NamespaceNodePortConfig) AddPort(namespace string, ports []int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否存在
	nsConfig, ok := c.GetNamespace(namespace)
	if !ok {
		return fmt.Errorf("namespace %s does not exist", namespace)
	}

	// 遍历要添加的端口，如果未分配则添加到列表中
	for _, port := range ports {
		// 检查是否超出范围
		if port < nsConfig.NodePortRange.Min || port > nsConfig.NodePortRange.Max {
			return fmt.Errorf("port %d is out of range for namespace %s", port, namespace)
		}

		// 检查是否已经分配
		if _, exists := nsConfig.AllocatedPorts[port]; exists {
			return fmt.Errorf("port %d is already allocated in namespace %s", port, namespace)
		}

		// 添加到已分配列表
		nsConfig.AllocatedPorts[port] = true
	}

	return nil
}

func (c *NamespaceNodePortConfig) RemovePortFromAPI(namespace string, ports []int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否存在
	nsConfig, ok := c.GetNamespace(namespace)
	if !ok {
		return fmt.Errorf("namespace %s does not exist", namespace)
	}

	// 遍历要移除的端口，如果已分配则从列表中移除
	for _, port := range ports {
		nsConfig.AllocatedPorts[port] = false
	}

	return nil
}

func (c *NamespaceNodePortConfig) FindAvailablePort(namespace string) (int, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否存在
	nsConfig, ok := c.GetNamespace(namespace)
	if !ok {
		return -1, fmt.Errorf("namespace %s does not exist", namespace)
	}

	// 遍历已分配的端口，找到第一个为false的端口并返回
	for port, allocated := range nsConfig.AllocatedPorts {
		if !allocated {
			return port, nil
		}
	}

	// 如果未找到可用端口，则返回错误
	return -1, fmt.Errorf("no available port in namespace %s", namespace)
}

func (c *NamespaceNodePortConfig) Len() int {
	return len(c.NamespaceConfigs)
}
