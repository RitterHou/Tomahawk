package main

import (
	"./common"
	"./network"
	"./node"
	"flag"
	"log"
	"os"
	"time"
)

// 数组类型的flag
type arrayFlag []string

func (flag *arrayFlag) String() string {
	return ""
}

func (flag *arrayFlag) Set(value string) error {
	*flag = append(*flag, value)
	return nil
}

var (
	port  uint      // 监听的端口
	hosts arrayFlag // 种子节点

	role string // follower, candidate, leader
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	flag.UintVar(&port, "port", 6300, "Bind port")
	flag.StringVar(&common.LocalNodeId, "id", common.RandomString(10), "Node id")
	flag.Var(&hosts, "hosts", "Seed hosts")
	flag.Parse()
}

func main() {
	node.AddNode(node.Node{NodeId: common.LocalNodeId})

	go network.Listen(port)
	for _, host := range hosts {
		network.Connect(host)
	}

	time.Sleep(1 * time.Hour)
}
