package node

import (
	"../common"
	"../tog"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 节点列表
var nodes = make(map[string]Node)
var mutex sync.Mutex

func AddNode(node Node) {
	if node.NodeId == "" {
		log.Fatal("node id can't be none")
	}
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s(me) add node %s", common.LocalNodeId, node.NodeId)
	}
	mutex.Lock()
	if node.AppendSuccess == nil { // 初始化
		node.AppendSuccess = make(chan bool, 1)
	}
	nodes[node.NodeId] = node
	mutex.Unlock()
}

func UpdateMatchIndex(nodeId string) {
	n := nodes[nodeId]
	n.MatchIndex = n.NextIndex
	nodes[nodeId] = n
}

func UpdateNextIndexByNodeId(nodeId string, nextIndex uint32) {
	if nodeId == common.LocalNodeId {
		// 只更新follower
		return
	}
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("Update %s.nextIndex: %d\n", nodeId, nextIndex)
	}
	n := nodes[nodeId]
	n.NextIndex = nextIndex
	nodes[nodeId] = n
}

func RemoveNodeById(nodeId string) {
	if nodeId == "" {
		log.Fatal("node id can't be none")
	}
	if tog.LogLevel(tog.DEBUG) {
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
	NodeId        string    // 节点的唯一ID
	Conn          net.Conn  // 当前节点与该节点的TCP连接，如果是当前节点，则为nil
	Ip            string    // 节点的IP地址
	TCPPort       uint32    // 节点所监听的TCP端口
	HTTPPort      uint32    // 节点所监听的HTTP服务端口
	NextIndex     uint32    // 如果当前节点是leader，记录了发送给该节点的下一个log entry下标
	MatchIndex    uint32    // 如果当前节点是leader，记录了该节点已经复制的最大的log entry下标，即leader与follower相匹配的最后一个log的下标
	AppendSuccess chan bool // follower告知leader此次AppendEntries是否成功
}

func (node Node) String() string {
	return fmt.Sprintf("{id: %s, connection: %s -> %s, ip: %s, port: %d, http: %d, nextIndex: %d, matchIndex: %d}",
		node.NodeId, node.Conn.LocalAddr(), node.Conn.RemoteAddr(), node.Ip, node.TCPPort, node.HTTPPort, node.NextIndex, node.MatchIndex)
}

// IP是否已经被更新
var ipUpdated = false

// 因为本地的IP可能不一定准确，所以如果有网络连接就可以更新本地IP
func UpdateLocalIp(conn net.Conn) {
	if !ipUpdated {
		ip := strings.Split(conn.LocalAddr().String(), ":")[0]
		ipUpdated = true
		n := nodes[common.LocalNodeId]
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("Update local IP from %s to %s\n", n.Ip, ip)
		}
		n.Ip = ip
		nodes[common.LocalNodeId] = n
	}
}

// 给所有的节点发送数据
func SendDataToFollowers(nodes []Node, data []byte) {
	for _, n := range nodes {
		// 并行的发送数据到follower，提升发送效率
		go func() {
			if n.Conn != nil {
				_, err := n.Conn.Write(data)
				if err != nil {
					log.Println(err)
				}
			}
		}()
	}
}

// 发送entries信息给follower
func SendAppendEntries() {
	replicatedNum := uint32(0) // 记录已经成功复制的follower

	for _, n := range GetNodes() {
		if n.Conn == nil {
			if tog.LogLevel(tog.DEBUG) {
				//log.Printf("%s's conn is nil\n", n.NodeId)
			}
			continue
		}
		// 这里的变量不能使用闭包，否则会有问题
		go func(n Node) {
		SendData:
			entries := common.GetEntries()[n.NextIndex-1:]
			entriesLength := len(entries)
			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s(me) for %s entriesLength: %d and nextIndex: %d\n", common.LocalNodeId, n.NodeId, entriesLength, n.NextIndex)
			}
			data := append([]byte{common.AppendEntries}, common.Uint32ToBytes(common.CurrentTerm)...)
			// 此节点对应的最后一个index
			data = append(data, common.Uint32ToBytes(n.NextIndex-1)...)
			// 此节点对应的最后一个index的term
			data = append(data, common.Uint32ToBytes(common.GetEntryByIndex(n.NextIndex - 1).Term)...)
			data = append(data, common.Uint32ToBytes(common.CommittedIndex)...)
			data = append(data, common.Uint32ToBytes(uint32(entriesLength))...)

			commitIndex := uint32(0)
			for i := 0; i < entriesLength; i++ {
				data = append(data, common.EncodeEntry(entries[i])...)
				commitIndex = common.Max(commitIndex, entries[i].Index) // 获取到最大的Index
			}
		Timeout:
			// 发送appendEntries给follower
			_, err := n.Conn.Write(data)
			if err != nil {
				log.Println(err)
				return
			}

			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s(me) send AppendEntries to %s\n", common.LocalNodeId, n.NodeId)
			}
			select {
			case appendSuccess := <-n.AppendSuccess:
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) get AppendEntries Reponse: %v\n", common.LocalNodeId, appendSuccess)
				}
				if appendSuccess {
					UpdateMatchIndex(n.NodeId)
					atomic.AddUint32(&replicatedNum, 1)
					if replicatedNum == common.Quorum { // 只触发一次
						common.LeaderAppendSuccess <- true  // client返回
						common.CommittedIndex = commitIndex // commit
					}
					return
				} else {
					// 发送失败，减小nextIndex进行重试
					n.NextIndex = n.NextIndex - 1 // 局部变量，只能在当前环境使用，需要进一步同步
					UpdateNextIndexByNodeId(n.NodeId, n.NextIndex)
					goto SendData
				}
			case <-time.After(time.Duration(common.LeaderResendAppendEntriesTimeout) * time.Millisecond):
				// 超时重试
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) to %s AppendEntries timeout and retry\n", common.LocalNodeId, n.NodeId)
				}
				goto Timeout
			}
		}(n)
	}
}
