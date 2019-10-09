// Raft算法中数据分布在多个节点上

package node

import (
	"../common"
	"../network"
	"../tog"
	"fmt"
	"log"
	"strings"
	"sync"
)

// 节点列表
var nodes sync.Map

// 添加一个节点，如果节点已经存在则会替换
func AddNode(node Node) {
	if node.NodeId == "" {
		log.Panic("node id can't be none")
	}
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s(me) add node %s", common.LocalNodeId, node.NodeId)
	}
	if node.AppendSuccess == nil { // 初始化
		node.AppendSuccess = make(chan bool, 10) // 增加缓冲区防止leader的AppendEntries超时导致response被阻塞
	}
	nodes.Store(node.NodeId, node)
}

// 更新matchIndex
func UpdateMatchIndex(nodeId string) {
	n := GetNode(nodeId)
	n.MatchIndex = n.NextIndex
	nodes.Store(nodeId, n)
}

// 更新指定节点的nextIndex
func UpdateNextIndexByNodeId(nodeId string, nextIndex uint32) {
	if nodeId == common.LocalNodeId {
		// 只更新follower
		return
	}
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("Update %s.nextIndex -> %d\n", nodeId, nextIndex)
	}

	n := GetNode(nodeId)
	n.NextIndex = nextIndex
	nodes.Store(nodeId, n)
}

// 移除一个节点
func RemoveNodeById(nodeId string) {
	if nodeId == "" {
		log.Panic("node id can't be none")
	}
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s(me) remove node %s", common.LocalNodeId, nodeId)
	}
	nodes.Delete(nodeId)

	if common.Role == common.Leader {
		// leader可能会因为节点的丢失导致quorum不满足选举的要求
		if uint32(len(GetNodes())) < common.Quorum {
			if tog.LogLevel(tog.INFO) {
				log.Printf("Leader %s became follower because [node size %d] < [quorum %d]\n",
					common.LocalNodeId, len(GetNodes()), common.Quorum)
			}
			common.ChangeRole(common.Follower)
		}
	}
}

// 判断指定节点是否存在
func ExistNode(nodeId string) bool {
	if nodeId == "" {
		log.Panic("node id can't be none")
	}
	_, ok := nodes.Load(nodeId)
	return ok
}

// 获取指定节点
func GetNode(nodeId string) Node {
	if nodeId == "" {
		log.Panic("node id can't be none")
	}
	n, _ := nodes.Load(nodeId)
	return n.(Node)
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

// 获取所有节点
func GetNodes() TheNodeList {
	n := make([]Node, 0)
	nodes.Range(func(key, value interface{}) bool {
		n = append(n, value.(Node))
		return true
	})
	return n
}

type Node struct {
	NodeId        string        // 节点的唯一ID
	Conn          *network.Conn // 当前节点与该节点的TCP连接，如果是当前节点，则为nil
	Ip            string        // 节点的IP地址
	TCPPort       uint32        // 节点所监听的TCP端口
	HTTPPort      uint32        // 节点所监听的HTTP服务端口
	NextIndex     uint32        // 如果当前节点是leader，记录了发送给该节点的下一个log entry下标
	MatchIndex    uint32        // 如果当前节点是leader，记录了该节点已经复制的最大的log entry下标，即leader与follower相匹配的最后一个log的下标
	AppendSuccess chan bool     // follower告知leader此次AppendEntries是否成功
}

func (node Node) String() string {
	return fmt.Sprintf("{id: %s, connection: %s -> %s, ip: %s, port: %d, http: %d, nextIndex: %d, matchIndex: %d}",
		node.NodeId, node.Conn.LocalAddr(), node.Conn.RemoteAddr(), node.Ip, node.TCPPort, node.HTTPPort, node.NextIndex, node.MatchIndex)
}

// IP是否已经被更新
var ipUpdated = false

// 因为本地的IP可能不一定准确，所以如果有网络连接就可以更新本地IP
func UpdateLocalIp(conn *network.Conn) {
	if !ipUpdated {
		ip := strings.Split(conn.LocalAddr(), ":")[0]
		ipUpdated = true
		n := GetNode(common.LocalNodeId)
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("Update local IP from %s to %s\n", n.Ip, ip)
		}
		n.Ip = ip
		nodes.Store(common.LocalNodeId, n)
	}
}
