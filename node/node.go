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

func RmvNode(node Node) {
	if node.NodeId == "" {
		log.Fatal("node id can't be none")
	}
	mutex.Lock()
	delete(nodes, node.NodeId)
	mutex.Unlock()
}

func RmvNodeById(nodeId string) {
	if nodeId == "" {
		log.Fatal("node id can't be none")
	}
	log.Printf("Remove node %s", nodeId)
	mutex.Lock()
	delete(nodes, nodeId)
	mutex.Unlock()
}

func GetNode(nodeId string) Node {
	return nodes[nodeId]
}

func GetNodes() []Node {
	m := make([]Node, 0, len(nodes))
	for _, val := range nodes {
		m = append(m, val)
	}
	return m
}

type Node struct {
	NodeId string
	Conn   net.Conn
}

func (node Node) String() string {
	return fmt.Sprintf("{id: %s, connection: %s}", node.NodeId, node.Conn.RemoteAddr())
}
