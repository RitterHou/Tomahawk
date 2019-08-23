package main

import (
	"./common"
	"./http"
	"./network"
	"./node"
	"./raft"
	"./tog"
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
	// 打印文件信息
	if len(args) == 2 && (args[1] == "--version" || args[1] == "-v") {
		fmt.Printf("Build TimeStamp : %s\n", buildStamp)
		fmt.Printf("Version         : %s\n", version)
		fmt.Printf("Go Version      : %s\n", goVersion)
		os.Exit(0)
	}

	// 保存信息
	common.BuildStamp = buildStamp
	common.Version = version
	common.GoVersion = goVersion

	common.InitParams()       // 初始化参数信息
	tog.Init(common.LogLevel) // 初始化日志设置

	if tog.LogLevel(tog.DEBUG) {
		log.Printf("Node Id:    %s\n", common.LocalNodeId)
		log.Printf("TCP Port:   %d\n", common.Port)
		log.Printf("HTTP Port:  %d\n", common.HTTPPort)
		log.Printf("Seed Hosts: %v", common.Hosts)
		log.Printf("Quorum:     %d\n", common.Quorum)
		log.Printf("LogLevel:   %s\n", common.LogLevel)
	}
}

func main() {
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("Host seeds %v\n", common.Hosts)
	}

	// 把自己加入到节点列表中
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
	go network.Listen(common.Port)        // 监听TCP端口
	go raft.Run()                         // 启动Raft协程
	http.StartHttpServer(common.HTTPPort) // 监听HTTP端口
}
