package main

import (
	"./common"
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

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	flag.UintVar(&common.Port, "port", 6300, "Bind port")
	flag.UintVar(&common.HTTPPort, "http", 6200, "HTTP server bind port")
	flag.StringVar(&common.LocalNodeId, "id", common.RandomString(10), "Node id")
	flag.Var(&common.Hosts, "hosts", "Seed hosts")
	flag.Parse()
}

func main() {
	node.AddNode(node.Node{
		NodeId:   common.LocalNodeId,
		TCPPort:  uint32(common.Port),
		HTTPPort: uint32(common.HTTPPort),
	})

	go network.Listen(common.Port)
	for _, host := range common.Hosts {
		network.Connect(host)
	}

	http.StartHttpServer(common.HTTPPort)
}
