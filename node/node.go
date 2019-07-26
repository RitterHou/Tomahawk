package node

import (
	"../common"
	"fmt"
	"log"
	"net"
	"sync"
)

// 节点列表
var nodes = make(map[string]Node)
var mutex sync.Mutex

func AddNode(node Node) {
	if node.NodeId == "" {
		log.Fatal("node id can't be none")
	}
	if common.LogLevel(common.DEBUG) {
		log.Printf("%s(me) add node %s", common.LocalNodeId, node.NodeId)
	}
	mutex.Lock()
	nodes[node.NodeId] = node
	mutex.Unlock()
}

func RemoveNodeById(nodeId string) {
	if nodeId == "" {
		log.Fatal("node id can't be none")
	}
	if common.LogLevel(common.DEBUG) {
		log.Printf("%s(me) remove node %s", common.LocalNodeId, nodeId)
	}
	mutex.Lock()
	delete(nodes, nodeId)
	mutex.Unlock()
}

func ExistNode(nodeId string) bool {
	if nodeId == "" {
		log.Fatal("node id can't be none")
	}
	_, ok := nodes[nodeId]
	return ok
}

func GetNode(nodeId string) Node {
	if nodeId == "" {
		log.Fatal("node id can't be none")
	}
	return nodes[nodeId]
}

// 用于对结果进行排序
type TheNodeList []Node

func (l TheNodeList) Len() int {
	return len(l)
}
func (l TheNodeList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
func (l TheNodeList) Less(i, j int) bool {
	return l[j].NodeId > l[i].NodeId
}

func GetNodes() TheNodeList {
	mutex.Lock()
	n := make([]Node, 0, len(nodes))
	for _, val := range nodes {
		n = append(n, val)
	}
	mutex.Unlock()
	return n
}

type Node struct {
	NodeId   string
	Conn     net.Conn
	Ip       string
	TCPPort  uint32
	HTTPPort uint32
}

func (node Node) String() string {
	return fmt.Sprintf("{id: %s, connection: %s -> %s, ip: %s, port: %d, http: %d}",
		node.NodeId, node.Conn.LocalAddr(), node.Conn.RemoteAddr(), node.Ip, node.TCPPort, node.HTTPPort)
}
