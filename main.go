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
	quorum     uint
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

	flag.UintVar(&common.Port, "port", 6300, "Bind port")                         // 绑定的TCP端口
	flag.UintVar(&common.HTTPPort, "http", 6200, "HTTP server bind port")         // 绑定的HTTP端口
	flag.StringVar(&common.LocalNodeId, "id", common.RandomString(10), "Node id") // 当前节点ID
	flag.Var(&common.Hosts, "hosts", "Seed hosts")                                // 种子节点的host
	flag.UintVar(&quorum, "quorum", 1, "As we know, most of the nodes")           // 投票时所需要的法定人数
	flag.StringVar(&logLevel, "level", "debug", "Log level")                      // 日志等级
	flag.Parse()

	common.Quorum = uint32(quorum) // flag不支持uint32类型
	tog.Init(logLevel)             // 初始化日志设置
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
