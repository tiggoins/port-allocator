package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type Items map[string][]Item

type Item struct {
	Namespace     string     `yaml:"namespace"`
	NodePortRange string     `yaml:"nodePortRange"`
	allocated     []int      
	lock          sync.Mutex 
}

func main() {
	y, err := os.ReadFile("port-range.yaml")
	if err != nil {
		log.Fatalln("error read file", err)
	}
	var item Items

	err = yaml.Unmarshal(y, &item)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(item)
}
