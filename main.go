package main

import (
	"./common"
	"./http"
	"./network"
	"./node"
	"./raft"
	"./tog"
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	buildStamp = ""
	version    = ""
	goVersion  = ""
	logLevel   = ""
)

func init() {
	args := os.Args
	if len(args) == 2 && (args[1] == "--version" || args[1] == "-v") {
		fmt.Printf("Build TimeStamp : %s\n", buildStamp)
		fmt.Printf("Version         : %s\n", version)
		fmt.Printf("Go Version      : %s\n", goVersion)
		os.Exit(0)
	}

	common.BuildStamp = buildStamp
	common.Version = version
	common.GoVersion = goVersion

	flag.UintVar(&common.Port, "port", 6300, "Bind port")
	flag.UintVar(&common.HTTPPort, "http", 6200, "HTTP server bind port")
	flag.StringVar(&common.LocalNodeId, "id", common.RandomString(10), "Node id")
	flag.Var(&common.Hosts, "hosts", "Seed hosts")
	flag.UintVar(&common.Quorum, "quorum", 1, "At we know, most of the nodes")
	flag.StringVar(&logLevel, "level", "debug", "Log level")
	flag.Parse()

	tog.Init(logLevel)
}

func main() {
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("Host seeds %v\n", common.Hosts)
	}

	node.AddNode(node.Node{
		NodeId:   common.LocalNodeId,
		TCPPort:  uint32(common.Port),
		HTTPPort: uint32(common.HTTPPort),
		Ip:       common.GetLocalIp(),
	})

	// 先连接，后启动监听，主要是为了防止连接到自己
	for _, host := range common.Hosts {
		network.Connect(host)
	}
	go network.Listen(common.Port)

	go raft.Run()
	http.StartHttpServer(common.HTTPPort)
}
