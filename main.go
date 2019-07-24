package main

import (
	"./common"
	"./network"
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
	id    string    // node_id
	hosts arrayFlag // 种子节点
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	flag.UintVar(&port, "port", 6300, "Bind port")
	flag.StringVar(&id, "id", common.RandomString(10), "Node id")
	flag.Var(&hosts, "hosts", "Seed hosts")
	flag.Parse()
}

func main() {
	log.Println(port)
	log.Println(id)
	log.Println(hosts)

	go network.Listen(port)
	for _, host := range hosts {
		network.Connect(host)
	}

	time.Sleep(1 * time.Hour)
}
