package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

const (
	NodePortMinPort int64  = 30000
	NodePortMaxPort int64  = 32767
)

type Items map[string][]Item

type Item struct {
	Namespace     string `yaml:"namespace"`
	NodePortRange string `yaml:"nodePortRange"`
}

type Result struct {
	Namespace string
	portStart int32
	portEnd   int32
}

type Results []Result

func LoadConfigFromFile(configFile string) Results {
	y, err := os.ReadFile(configFile)
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

func ParsePortRange(portRange string) (int32, int32, error) {
	ports := strings.Split(portRange, "-")
	min, err := strconv.ParseInt(ports[0], 10, 32)
	if err != nil {
		klog.Warning(err)
		return -1, -1, err
	}
	max, err := strconv.ParseInt(ports[1], 10, 32)
	if err != nil {
		klog.Warning(err)
		return -1, -1, err
	}

	if min > max || min >= NodePortMinPort || max <= NodePortMaxPort {
		msg := fmt.Errorf("nodeport range MUST from small to big,and MUST between 30000 to 32767")
		klog.Warning(msg)
		return -1, -1, msg
	}

	return int32(min), int32(max), nil
}

func (r Results) checkOverlap() {
	for i := 0; i < len(r); i++ {
		for j := 0; j < len(r); j++ {
			if i != j {
				if (r[i].portStart >= r[j].portStart && r[i].portStart <= r[j].portEnd) ||
					(r[i].portEnd >= r[j].portStart && r[i].portEnd <= r[j].portEnd) {
					klog.Infof("nodeport range of namespace %s/(%d-%d) overlaps with port range of namespace %s/(%d-%d),exit the program.\n",
						r[i].Namespace, r[i].portStart, r[i].portEnd,
						r[j].Namespace, r[j].portStart, r[j].portEnd)
					os.Exit(1)
				}
			}
		}
	}
}
