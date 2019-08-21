package node

import (
	"../common"
	"../tog"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 节点列表
var nodes sync.Map

func AddNode(node Node) {
	if node.NodeId == "" {
		log.Fatal("node id can't be none")
	}
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s(me) add node %s", common.LocalNodeId, node.NodeId)
	}
	if node.AppendSuccess == nil { // 初始化
		node.AppendSuccess = make(chan bool, 10) // 增加缓冲区防止leader的AppendEntries超时导致response被阻塞
	}
	nodes.Store(node.NodeId, node)
}

func UpdateMatchIndex(nodeId string) {
	n := GetNode(nodeId)
	n.MatchIndex = n.NextIndex
	nodes.Store(nodeId, n)
}

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

func RemoveNodeById(nodeId string) {
	if nodeId == "" {
		log.Fatal("node id can't be none")
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

func ExistNode(nodeId string) bool {
	if nodeId == "" {
		log.Fatal("node id can't be none")
	}
	_, ok := nodes.Load(nodeId)
	return ok
}

func GetNode(nodeId string) Node {
	if nodeId == "" {
		log.Fatal("node id can't be none")
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

func GetNodes() TheNodeList {
	n := make([]Node, 0)
	nodes.Range(func(key, value interface{}) bool {
		n = append(n, value.(Node))
		return true
	})
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
		n := GetNode(common.LocalNodeId)
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("Update local IP from %s to %s\n", n.Ip, ip)
		}
		n.Ip = ip
		nodes.Store(common.LocalNodeId, n)
	}
}

// 向指定节点发送数据
func sendData(n Node, data []byte) error {
	_, err := n.Conn.Write(data)
	if err != nil {
		log.Println(err)
		if tog.LogLevel(tog.WARN) {
			log.Printf("colse connection %s -> %s\n",
				n.Conn.LocalAddr().String(), n.Conn.RemoteAddr().String())
		}
		// 关闭连接
		err = n.Conn.Close()
		if err != nil {
			log.Fatal(err)
		}
		RemoveNodeById(n.NodeId) // 连接出现异常，移除该节点
		return errors.New(fmt.Sprintf("%v write data failed", n.Conn))
	}
	return nil
}

// 给所有的节点发送数据
func SendDataToFollowers(nodes []Node, data []byte) {
	for _, n := range nodes {
		if n.Conn == nil {
			continue
		}
		// 并行的发送数据到follower，提升发送效率
		go func(n Node) { // 这里的变量不能使用闭包，否则会有问题
			_ = sendData(n, data)
		}(n)
	}
}

// 发送entries信息给follower
func SendAppendEntries() {
	replicatedNum := uint32(1) // 记录已经成功复制的follower

	for _, n := range GetNodes() {
		if n.Conn == nil {
			if tog.LogLevel(tog.DEBUG) {
				//log.Printf("%s's conn is nil\n", n.NodeId)
			}
			continue
		}
		// 这里的变量不能使用闭包，否则会有问题
		go func(n Node) {
			// 如果上次的leader超时了，但是后来follower又返回了数据，则channel不为空
			// 因此为了排除影响，需要清空channel
			for len(n.AppendSuccess) > 0 {
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("Channel AppendSuccess Length: %d\n", len(n.AppendSuccess))
				}
				<-n.AppendSuccess
			}

			entries := common.GetEntries()[n.NextIndex:]
			entriesLength := len(entries)
			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s(me) for %s entriesLength: %d and nextIndex: %d\n", common.LocalNodeId, n.NodeId, entriesLength, n.NextIndex)
			}

			// 该follower在leader中所记录的最后一个节点的index
			followerIndex := n.NextIndex - 1

			data := []byte{common.AppendEntries}
			data = append(data, common.Uint32ToBytes(common.CurrentTerm)...)                         // 当前任期
			data = append(data, common.Uint32ToBytes(followerIndex)...)                              // 此follower的最后一个节点
			data = append(data, common.Uint32ToBytes(common.GetEntryByIndex(followerIndex).Term)...) // 此follower的最后一个节点的任期
			data = append(data, common.Uint32ToBytes(common.CommittedIndex)...)                      // leader的commitIndex
			data = append(data, common.Uint32ToBytes(uint32(entriesLength))...)                      // 发送的entries的长度

			commitIndex := uint32(0)
			for i := 0; i < entriesLength; i++ {
				data = append(data, common.EncodeEntry(entries[i])...)
				commitIndex = common.Max(commitIndex, entries[i].Index) // 获取到最大的Index
			}

			// 发送appendEntries给follower
			err := sendData(n, data)
			if err != nil {
				// 如果发送失败则无需再进行接下来的操作
				return
			}

			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s(me) send AppendEntries to %s\n", common.LocalNodeId, n.NodeId)
			}
			select {
			case appendSuccess := <-n.AppendSuccess:
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) Get AppendEntries Reponse from %s: %v\n",
						common.LocalNodeId, n.NodeId, appendSuccess)
				}
				if appendSuccess {
					// Append成功，该follower已经追上leader的entries进度
					UpdateNextIndexByNodeId(n.NodeId, common.GetEntriesLength())
					UpdateMatchIndex(n.NodeId)
					atomic.AddUint32(&replicatedNum, 1)
					if replicatedNum == common.Quorum { // 只触发一次
						common.LeaderAppendSuccess <- true  // client返回
						common.CommittedIndex = commitIndex // commit
					}
				} else {
					// 发送失败，减小nextIndex进行重试

					// 回退nextIndex到前一个term的最后一个entry的index
					// 相较于把nextIndex直接减一，这样做更加高效
					var lastTerm = uint32(0)                        // 后一个entry的term
					allEntries := common.GetEntries()[:n.NextIndex] // 剔除当前的nextIndex的后面的内容
					for i := len(allEntries) - 1; i >= 0; i-- {
						e := allEntries[i]
						if lastTerm > e.Term { // 找到当前的term所对应初始index
							n.NextIndex = e.Index + 1
							break
						}
						lastTerm = e.Term
					}

					UpdateNextIndexByNodeId(n.NodeId, n.NextIndex)
				}
			case <-time.After(time.Duration(common.LeaderAppendEntriesTimeout) * time.Millisecond):
				// AppendEntries超时，不需要手动重试，因为leader的下一次心跳
				// 将在 LeaderCycleTimeout - LeaderAppendEntriesTimeout 后发生，心跳会实现下一次重试
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) to %s AppendEntries timeout and retry\n", common.LocalNodeId, n.NodeId)
				}
			}
		}(n)
	}
}
