package store

import (
	"container/list"
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
	OrderedKeys    *list.List // 使用链表来维护端口的顺序
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

func (c *NamespaceNodePortConfig) getNamespace(namespace string) (*NamespaceConfig, bool) {
	nsConfig, ok := c.NamespaceConfigs[namespace]
	if !ok {
		return &NamespaceConfig{}, false
	}

	return nsConfig, true
}

func (c *NamespaceNodePortConfig) addNamespace(namespace string, minPort, maxPort int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否已存在
	if _, ok := c.getNamespace(namespace); ok {
		return fmt.Errorf("namespace %s already exists", namespace)
	}

	// 添加命名空间配置
	c.NamespaceConfigs[namespace] = &NamespaceConfig{
		NodePortRange:  PortRange{Min: minPort, Max: maxPort},
		AllocatedPorts: make(map[int]bool),
		OrderedKeys:    list.New(), // 初始化为链表类型
	}
	return nil
}

func (c *NamespaceNodePortConfig) addPort(namespace string, ports []int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查命名空间是否存在
	nsConfig, ok := c.getNamespace(namespace)
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

		// 添加到已分配列表，并将端口添加到 OrderedKeys 中
		nsConfig.AllocatedPorts[port] = true
		nsConfig.OrderedKeys.PushBack(port)
	}

	return nil
}

func (c *NamespaceNodePortConfig) removePortFromAPI(namespace string, ports []int) error {
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

		// 移除 OrderedKeys 中的端口
		for e := nsConfig.OrderedKeys.Front(); e != nil; e = e.Next() {
			if e.Value.(int) == port {
				nsConfig.OrderedKeys.Remove(e)
				break
			}
		}
	}

	return nil
}

func (c *NamespaceNodePortConfig) findAvailablePort(namespace string) (int, error) {
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
func (c *NamespaceNodePortConfig) ifMeetRequirements(namespace string, port int) bool {
	nsConfig, ok := c.getNamespace(namespace)
	if !ok {
		return false
	}

	if port <= nsConfig.NodePortRange.Max && port >= nsConfig.NodePortRange.Min {
		return true
	}

	return false
}

func (c *NamespaceNodePortConfig) len() int {
	return len(c.NamespaceConfigs)
}
