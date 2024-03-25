package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

type Items map[string][]Item

type Item struct {
	Namespace     string `yaml:"namespace"`
	NodePortRange string `yaml:"nodePortRange"`
}

type Result struct {
	Namespace string
	portStart int
	portEnd   int
}

type Results []Result

func LoadConfigFromFile() Results {
	y, err := os.ReadFile("port-range.yaml")
	if err != nil {
		klog.Fatalln("error read file", err)
	}

	var items Items
	err = yaml.Unmarshal(y, &items)
	if err != nil {
		klog.Fatalln(err)
	}

	var results Results
	for _, v := range items {
		for _, vv := range v {
			start, end, err := ParsePortRange(vv.NodePortRange)
			if err != nil {
				klog.Warningf("error parse nodeportrange of %s, got %s, skip this", vv.Namespace, vv.NodePortRange)
				continue
			}
			result := Result{
				Namespace: vv.Namespace,
				portStart: start,
				portEnd:   end,
			}
			results = append(results, result)
		}
	}

	// check if nodePort overlap, if so, exit
	results.checkOverlap()

	return results
}

func ParsePortRange(portRange string) (int, int, error) {
	ports := strings.Split(portRange, "-")
	min, err := strconv.Atoi(ports[0])
	if err != nil {
		klog.Warning(err)
		return -1, -1, err
	}
	max, err := strconv.Atoi(ports[1])
	if err != nil {
		klog.Warning(err)
		return -1, -1, err
	}

	if min > max {
		msg := fmt.Errorf("nodeport range must from smaller to big")
		klog.Warning(msg)
		return -1, -1, msg
	}

	return min, max, nil
}

func (results Results) checkOverlap() {
	for i := 0; i < len(results); i++ {
		for j := 0; j < len(results); j++ {
			if i != j {
				if (results[i].portStart >= results[j].portStart && results[i].portStart <= results[j].portEnd) ||
					(results[i].portEnd >= results[j].portStart && results[i].portEnd <= results[j].portEnd) {
					klog.Infof("nodeport range of namespace %s/(%d-%d) overlaps with port range of namespace %s/(%d-%d),exit the program.\n",
						results[i].Namespace, results[i].portStart, results[i].portEnd,
						results[j].Namespace, results[j].portStart, results[j].portEnd)
					os.Exit(1)
				}
			}
		}
	}
}
