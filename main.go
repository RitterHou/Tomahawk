package main

import (
	"./common"
	"./election"
	"./http"
	"./network"
	"./node"
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	buildStamp = ""
	version    = ""
	goVersion  = ""
)

func init() {
	args := os.Args
	if len(args) == 2 && (args[1] == "--version" || args[1] == "-v") {
		fmt.Printf("Build TimeStamp : %s\n", buildStamp)
		fmt.Printf("Version         : %s\n", version)
		fmt.Printf("Go Version      : %s\n", goVersion)
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	flag.UintVar(&common.Port, "port", 6300, "Bind port")
	flag.UintVar(&common.HTTPPort, "http", 6200, "HTTP server bind port")
	flag.StringVar(&common.LocalNodeId, "id", common.RandomString(10), "Node id")
	flag.Var(&common.Hosts, "hosts", "Seed hosts")
	flag.Parse()
}

func main() {
	if common.LogLevel(common.DEBUG) {
		log.Printf("Host seeds %v\n", common.Hosts)
	}

	node.AddNode(node.Node{
		NodeId:   common.LocalNodeId,
		TCPPort:  uint32(common.Port),
		HTTPPort: uint32(common.HTTPPort),
	})

	// 先连接，后启动监听，主要是为了防止连接到自己
	for _, host := range common.Hosts {
		network.Connect(host)
	}
	go network.Listen(common.Port)

	go election.Do()
	http.StartHttpServer(common.HTTPPort)
}
