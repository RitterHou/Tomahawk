package node

import (
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
	log.Printf("Add node %s", node.NodeId)
	mutex.Lock()
	nodes[node.NodeId] = node
	mutex.Unlock()
}

func RemoveNodeById(nodeId string) {
	if nodeId == "" {
		log.Fatal("node id can't be none")
	}
	log.Printf("Remove node %s", nodeId)
	mutex.Lock()
	delete(nodes, nodeId)
	mutex.Unlock()
}

func NodeExist(nodeId string) bool {
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

func GetNodes() []Node {
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
	return fmt.Sprintf("{id: %s, connection: %s, ip: %s, port: %d, http: %d}",
		node.NodeId, node.Conn.RemoteAddr(), node.Ip, node.TCPPort, node.HTTPPort)
}
