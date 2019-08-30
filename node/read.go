// 处理网络连接，进行网络数据的读取工作

package node

import (
	"../common"
	"../network"
	"../tog"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"
)

// 处理TCP连接，此时已经不区分client与server
func handleConnection(conn *network.Conn) {
	defer func() {
		// 出现任何意外情况导致连接出现问题，移除该连接所对应的节点
		RemoveNodeById(conn.GetRemoteNodeId())

		err := conn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// 第一件事就是把当前节点的信息告知远程节点
	success := sendNodeInfo(conn)
	if !success {
		return
	}

	// 读取远程主机发送的数据
	for {
		dataType, success := conn.ReadByte()
		if !success {
			return
		}

		socketDataType := common.GetSocketDataType(dataType)
		readRemoteDataStart := time.Now().UnixNano()
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("%s Read data type from %s: %v\n",
				common.LocalNodeId, conn.GetRemoteNodeId(), socketDataType)
		}

		switch dataType {
		case common.ExchangeNodeInfo:
			success = exchangeNodeInfo(conn)
			if !success {
				return
			}
		case common.ShareNodes:
			success = shareNodes(conn)
			if !success {
				return
			}
		case common.AppendEntries:
			success = appendEntries(conn)
			if !success {
				return
			}
		case common.AppendEntriesResponse:
			success = appendEntriesResponse(conn)
			if !success {
				return
			}
		case common.VoteRequest:
			success = voteRequest(conn)
			if !success {
				return
			}
		case common.VoteResponse:
			success = voteResponse(conn)
			if !success {
				return
			}
		}

		readRemoteDataEnd := time.Now().UnixNano()
		readRemoteDataCost := readRemoteDataEnd - readRemoteDataStart
		if tog.LogLevel(tog.DEBUG) {
			if readRemoteDataCost/1e6 > 0 {
				log.Printf("Cost %dms, %s\n", readRemoteDataCost/1e6, socketDataType)
			} else {
				log.Printf("Cost %dns, %s\n", readRemoteDataCost, socketDataType)
			}
		}
	}
}

// 发送当前节点的信息给远程节点
func sendNodeInfo(c *network.Conn) bool {
	localNodeIdBuf := common.AddBufHead([]byte(common.LocalNodeId))
	portBuf := common.Uint32ToBytes(uint32(common.Port))
	httpPortBuf := common.Uint32ToBytes(uint32(common.HTTPPort))

	nodeInfo := []byte{common.ExchangeNodeInfo}    // 操作类型
	nodeInfo = append(nodeInfo, localNodeIdBuf...) // 节点id
	nodeInfo = append(nodeInfo, portBuf...)        // TCP服务端口
	nodeInfo = append(nodeInfo, httpPortBuf...)    // HTTP服务端口

	// 一旦与远程主机连接，立即告知其自己的节点信息
	success := c.Write(nodeInfo)
	return success
}

// 交换节点信息
func exchangeNodeInfo(conn *network.Conn) bool {
	// 节点ID
	nodeIdBuf, success := conn.ReadStr()
	if !success {
		return false
	}
	nodeId := string(nodeIdBuf)
	conn.SetRemoteNodeId(nodeId) // 设置远程节点的ID

	// 端口号
	remotePortBuf, success := conn.ReadBytes(4)
	if !success {
		return false
	}
	remotePort := binary.LittleEndian.Uint32(remotePortBuf)

	// HTTP端口号
	remoteHTTPPort, success := conn.ReadUint32()
	if !success {
		return false
	}

	// IP地址
	remoteIp := strings.Split(conn.RemoteAddr(), ":")[0]

	remoteNode := Node{
		NodeId:   nodeId,
		Conn:     conn,
		Ip:       remoteIp,
		TCPPort:  remotePort,
		HTTPPort: remoteHTTPPort,
	}
	AddNode(remoteNode) // 添加节点
	if tog.LogLevel(tog.INFO) {
		log.Println("Connected to remote node:", remoteNode)
	}

	// 如果是新连接的节点并且当前节点是leader，则需要设置远程节点的nextIndex
	if common.Role == common.Leader {
		UpdateNextIndexByNodeId(nodeId, common.GetEntriesLength())
	}

	if conn.LocalPort() != common.Port {
		return true
	}

	// 把此节点广播给其它节点
	nodes := GetNodes()
	for _, n := range nodes {
		nodeConn := n.Conn
		if nodeConn != nil {
			info := []byte{common.ShareNodes}
			info = append(info, common.AddBufHead(nodeIdBuf)...)
			info = append(info, common.AddBufHead([]byte(remoteIp))...)
			info = append(info, remotePortBuf...)
			success := nodeConn.Write(info)
			if !success {
				return false
			}
		}
	}
	return true
}

// 共享节点列表
func shareNodes(conn *network.Conn) bool {
	// 为什么不需要使用Gossip这样的算法呢，因为raft这样的系统一般不会直接存储海量的数据，而是存储一些meta
	// 数据、或者作为另一个集群的master来使用，所以节点数量不会太多，因此不需要Gossip这样的节点间数据同步协议
	nodeIdBuf, success := conn.ReadStr()
	if !success {
		return false
	}
	nodeId := string(nodeIdBuf)

	remoteIPBuf, success := conn.ReadStr()
	if !success {
		return false
	}
	remoteIp := string(remoteIPBuf)

	remotePort, success := conn.ReadUint32()
	if !success {
		return false
	}

	// 只有这个节点不存在于节点列表的时候才需要去连接
	if !ExistNode(nodeId) {
		Connect(fmt.Sprintf("%s:%d", remoteIp, remotePort))
	}

	return true
}

// 发送日志信息
func appendEntries(conn *network.Conn) bool {
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s(me) Get AppendEntries from %s\n", common.LocalNodeId, conn.GetRemoteNodeId())
	}
	// leader的任期
	leaderTerm, success := conn.ReadUint32()
	if !success {
		return false
	}
	// leader所记录的当前follower的最后一个log索引
	leaderPrevLogIndex, success := conn.ReadUint32()
	if !success {
		return false
	}
	// leader所记录的当前follower的最后一个log索引的任期
	leaderPrevLogTerm, success := conn.ReadUint32()
	if !success {
		return false
	}
	// leader的commitIndex
	leaderCommittedIndex, success := conn.ReadUint32()
	if !success {
		return false
	}
	// leader的此次append是否成功
	appendSuccess := true
	if leaderTerm < common.CurrentTerm {
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("Leader term %d little than local term %d, append fialed\n", leaderTerm, common.CurrentTerm)
		}
		appendSuccess = false
	}

	entries := common.GetEntries()
	if uint32(len(entries)) > leaderPrevLogIndex {
		// leader记录的当前节点最后一个log的term和本地的不一致，appendEntries失败
		if entry := entries[leaderPrevLogIndex]; entry.Term != leaderPrevLogTerm {
			if tog.LogLevel(tog.DEBUG) {
				log.Printf("LeaderPrevLogTerm %d not equals local term %d at index %d, append fialed\n",
					leaderPrevLogIndex, entry.Term, leaderPrevLogIndex)
			}
			appendSuccess = false
		}
	} else {
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("Local entriesLegnth %d little than leaderPrevLogIndex %d, append fialed\n",
				len(entries), leaderPrevLogIndex)
		}
		// 如果leader记录的当前节点index超出限制，也是一种不匹配
		appendSuccess = false
	}
	// 此次AppendEntries的entries长度
	appendEntriesLength, success := conn.ReadUint32()
	if !success {
		return false
	}

	if tog.LogLevel(tog.DEBUG) {
		log.Printf("AppendEntries from leader, entries length: %d\n", appendEntriesLength)
	}

	// 遍历所有的entry
	for i := uint32(0); i < appendEntriesLength; i++ {
		keyBuf, success := conn.ReadStr()
		if !success {
			return false
		}
		key := string(keyBuf)

		valueBuf, success := conn.ReadStr()
		if !success {
			return false
		}
		value := string(valueBuf)

		// 该Entry所对应的任期
		term, success := conn.ReadUint32()
		if !success {
			return false
		}

		// 该Entry在log中的位置
		index, success := conn.ReadUint32()
		if !success {
			return false
		}

		createTime, success := conn.ReadUint32()
		if !success {
			return false
		}

		if tog.LogLevel(tog.TRACE) {
			log.Printf("AppendEntry from leader, key: %s, value: %s, term: %d, index: %d\n",
				key, value, term, index)
		}

		// 如果可以把entry append到当前节点中
		if appendSuccess {
			entries = common.GetEntries() // 获取当前节点的当前entries
			// 一旦产生冲突，从当前节点开始进行cutOff
			if uint32(len(entries)) > index && entries[index].Term != term {
				common.CutoffEntries(index)
			}
			// 把entry保存到follower中去
			common.SetEntry(common.Entry{Key: key, Value: value, Index: index, Term: term, Time: createTime})
		}
	}

	// 根据leader的committedIndex更新当前节点的committedIndex
	if leaderCommittedIndex > common.CommittedIndex {
		common.CommittedIndex = common.Min(leaderCommittedIndex, common.GetEntriesLength())
	}

	switch common.Role {
	case common.Leader:
		// 比当前term要大，当前节点恢复follower状态
		if leaderTerm > common.CurrentTerm {
			common.ChangeRole(common.Follower)
		} else {
			appendSuccess = false // 拒绝这次AppendEntries
		}
	case common.Candidate:
		// leader的term不小于自己的term，重新变为follower状态
		if leaderTerm >= common.CurrentTerm {
			common.ChangeRole(common.Follower)
			common.VoteSuccessCh <- false
		} else {
			appendSuccess = false // 拒绝这次AppendEntries
		}
	case common.Follower:
		// 重置超时定时器
		common.HeartbeatTimeoutCh <- true
		common.ChangeTerm(leaderTerm)
		common.LeaderNodeId = conn.GetRemoteNodeId() // 设置leader节点
	}

	// AppendEntries的响应
	var response = []byte{common.AppendEntriesResponse}
	if appendSuccess {
		response = append(response, byte(1))
	} else {
		response = append(response, byte(0))
	}

	response = append(response, common.Uint32ToBytes(common.CurrentTerm)...)
	writeSuccess := conn.Write(response)
	if !writeSuccess {
		return false
	}

	if tog.LogLevel(tog.DEBUG) {
		log.Printf("AppendEntries leaderPrevLogIndex: %d, leaderPrevLogTerm: %d, leaderCommittedIndex: %d,"+
			" entriesLength: %d, result: %v, resultTerm: %d\n", leaderPrevLogIndex, leaderPrevLogTerm,
			leaderCommittedIndex, appendEntriesLength, appendSuccess, common.CurrentTerm)
	}

	return true
}

// 发送日志信息响应
func appendEntriesResponse(conn *network.Conn) bool {
	resSuccessBuf, success := conn.ReadByte()
	if !success {
		return false
	}
	resSuccess := true
	if resSuccessBuf == 0 {
		resSuccess = false
	}

	term, success := conn.ReadUint32()
	if !success {
		return false
	}
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s Get AppendEntriesResponse from %s, term: %d, local term: %d, success: %t\n",
			common.LocalNodeId, conn.GetRemoteNodeId(), term, common.CurrentTerm, resSuccess)
	}
	if term > common.CurrentTerm {
		if tog.LogLevel(tog.WARN) {
			log.Printf("Update current term %d->%d and become follower\n", common.CurrentTerm, term)
		}
		common.ChangeTerm(term)
		common.ChangeRole(common.Follower)
	}

	n := GetNode(conn.GetRemoteNodeId())
	n.AppendSuccess <- resSuccess // 获取follower的返回结果并通过channel进行同步

	return true
}

// 发送投票请求
func voteRequest(conn *network.Conn) bool {
	if common.Role == common.Follower {
		// 重置超时定时器
		common.HeartbeatTimeoutCh <- true
	}
	candidateTerm, success := conn.ReadUint32()
	if !success {
		return false
	}

	lastEntryIndex, success := conn.ReadUint32()
	if !success {
		return false
	}

	lastEntryTerm, success := conn.ReadUint32()
	if !success {
		return false
	}

	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s(me) term %d -> remote %s term %d ",
			common.LocalNodeId, common.CurrentTerm, conn.GetRemoteNodeId(), candidateTerm)
	}

	voteSuccess := false

	// 大于当前的任期
	if candidateTerm >= common.CurrentTerm {
		common.ChangeTerm(candidateTerm)
		common.ChangeRole(common.Follower)
		// 尚未投票或者投给了candidate
		if nodeId, ok := common.VoteFor[candidateTerm]; !ok || nodeId == conn.GetRemoteNodeId() {
			// candidate的最新数据比当前节点的数据要新
			lastEntry := common.GetLastEntry()
			if lastEntryIndex >= lastEntry.Index && lastEntryTerm >= lastEntry.Term {
				voteSuccess = true
			}
		}
	}

	var response = []byte{common.VoteResponse}
	if voteSuccess {
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("%s(me) send vote for %s, term %d\n",
				common.LocalNodeId, conn.GetRemoteNodeId(), common.CurrentTerm)
		}
		response = append(response, byte(1)) // 投票
	} else {
		response = append(response, byte(0)) // 不投票
	}

	response = append(response, common.Uint32ToBytes(common.CurrentTerm)...)
	writeSuccess := conn.Write(response)
	if !writeSuccess {
		return false
	}

	return true
}

// 发送投票响应
func voteResponse(conn *network.Conn) bool {
	voteBuf, success := conn.ReadByte()
	if !success {
		return false
	}

	term, success := conn.ReadUint32()
	if !success {
		return false
	}

	vote := voteBuf
	if vote == 1 {
		atomic.AddUint32(&common.Votes, 1)
		if common.Votes >= common.Quorum {
			common.VoteSuccessCh <- true
		}
	} else {
		if term > common.CurrentTerm {
			common.ChangeTerm(term)
			common.ChangeRole(common.Follower)
			common.VoteSuccessCh <- false
		}
	}

	return true
}
