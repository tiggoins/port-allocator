package store

import (
	"fmt"
	"sort"
	"sync"
)

type NamespaceNodePortConfig struct {
	NamespaceConfigs map[string]*NamespaceConfig
	lock             sync.Mutex
}

type NamespaceConfig struct {
	NodePortRange  PortRange
	AllocatedPorts map[int32]bool
}

type PortRange struct {
	Min int32
	Max int32
}

func NewNamespaceNodePortConfig() *NamespaceNodePortConfig {
	return &NamespaceNodePortConfig{
		NamespaceConfigs: make(map[string]*NamespaceConfig),
	}
}

func (c *NamespaceNodePortConfig) getNamespace(namespace string) (*NamespaceConfig, bool) {
	nsConfig, ok := c.NamespaceConfigs[namespace]
	if !ok {
		return &NamespaceConfig{}, false
	}

	return nsConfig, true
}

func (c *NamespaceNodePortConfig) AddNamespace(namespace string, minPort, maxPort int32) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否已存在
	if _, ok := c.getNamespace(namespace); ok {
		return fmt.Errorf("namespace %s already exists", namespace)
	}

	// 添加命名空间配置
	c.NamespaceConfigs[namespace] = &NamespaceConfig{
		NodePortRange:  PortRange{Min: minPort, Max: maxPort},
		AllocatedPorts: make(map[int32]bool),
	}
	return nil
}

func (c *NamespaceNodePortConfig) AddPortToNamespace(namespace string, ports []int32) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否存在
	nsConfig, ok := c.getNamespace(namespace)
	if !ok {
		return fmt.Errorf("namespace %s does not exist", namespace)
	}

	// 对切片进行排序
	sort.Slice(ports, func(i, j int) bool {
		return ports[i] < ports[j]
	})

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

		// 添加到已分配列表，并将端口添加到 OrderedKeys 中
		nsConfig.AllocatedPorts[port] = true
	}

	return nil
}

func (c *NamespaceNodePortConfig) RemovePortFromAPI(namespace string, ports []int32) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否存在
	nsConfig, ok := c.getNamespace(namespace)
	if !ok {
		return fmt.Errorf("namespace %s does not exist", namespace)
	}

	// 遍历要移除的端口，如果已分配则从列表中移除
	for _, port := range ports {
		nsConfig.AllocatedPorts[port] = false
	}

	return nil
}

func (c *NamespaceNodePortConfig) FindAvailablePort(namespace string) (int32, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否存在
	nsConfig, ok := c.getNamespace(namespace)
	if !ok {
		return -1, fmt.Errorf("namespace %s does not exist", namespace)
	}

	// 遍历 NodePort 范围内的端口，找到第一个未分配的端口并返回
	for port := nsConfig.NodePortRange.Min; port <= nsConfig.NodePortRange.Max; port++ {
		if !nsConfig.AllocatedPorts[port] {
			return port, nil
		}
	}

	// 如果未找到可用端口，则返回错误
	return -1, fmt.Errorf("no available port in namespace %s", namespace)
}

// check if port is in the range of requirements
func (c *NamespaceNodePortConfig) IfMeetRequirements(namespace string, port int32) bool {
	nsConfig, ok := c.getNamespace(namespace)
	if !ok {
		return false
	}

	return port <= nsConfig.NodePortRange.Max && port >= nsConfig.NodePortRange.Min
}

func (c *NamespaceNodePortConfig) Len() int {
	return len(c.NamespaceConfigs)
}
