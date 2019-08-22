package network

import (
	"../common"
	"../node"
	"../tog"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

// 连接远程主机
func Connect(host string) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		if err.Error() == "dial tcp "+host+": connect: connection refused" {
			if tog.LogLevel(tog.WARN) {
				log.Println("Connection refused by node:", host)
			}
			return
		}
		log.Fatal(err)
	}
	node.UpdateLocalIp(conn) // 主动连接别的节点时，可以更新当前节点的IP地址
	go handleConnection(conn)
}

// 监听在本地的指定端口
func Listen(port uint) {
	host := fmt.Sprintf("0.0.0.0:%d", port)
	l, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatal(err)
	}
	if tog.LogLevel(tog.INFO) {
		log.Println("TCP Server Listening Port", port)
	}
	defer func() {
		err = l.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		node.UpdateLocalIp(conn) // 当别的节点连接自己时，可以更新当前节点的IP地址
		go handleConnection(conn)
	}
}

// 处理TCP连接，此时已经不区分client与server
func handleConnection(c net.Conn) {
	defer func() {
		if tog.LogLevel(tog.INFO) {
			log.Printf("colse connection %s -> %s\n",
				c.LocalAddr().String(), c.RemoteAddr().String())
		}
		err := c.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// 第一件事就是把当前节点的信息告知远程节点
	sendNodeInfo(c)

	// 记录此连接所对应的远程节点id，可以作为一个索引方便后续操作
	var remoteNodeId string

	// 对数据读取做了简单的封装
	readBytes := func(length uint32) ([]byte, bool) {
		data := make([]byte, length)
		_, err := c.Read(data)
		if err != nil {
			if err == io.EOF || strings.ContainsAny(err.Error(), "connection reset by peer") {
				if err == io.EOF {
					if tog.LogLevel(tog.WARN) {
						log.Println("connection closed by remote:", c.RemoteAddr())
					}
				} else {
					if tog.LogLevel(tog.WARN) {
						log.Println(err)
					}
				}

				node.RemoveNodeById(remoteNodeId)
				return nil, false
			}
			log.Fatal(err)
		}
		return data, true
	}

	// 读取一个字节
	readByte := func() (byte, bool) {
		buf, success := readBytes(1)
		if !success {
			return 0, false
		}
		return buf[0], true
	}

	// 读取一个字符串
	readStr := func() ([]byte, bool) {
		head, success := readByte()
		if !success {
			return nil, success
		}
		if head < 0xff {
			body, success := readBytes(uint32(head))
			return body, success
		} else {
			lengthBuf, success := readBytes(4)
			if !success {
				return nil, success
			}
			length := binary.LittleEndian.Uint32(lengthBuf)
			body, success := readBytes(length)
			return body, success
		}
	}

	// 读取一个uint32数字
	readUint32 := func() (uint32, bool) {
		buf, success := readBytes(4)
		if !success {
			return 0, false
		}
		return binary.LittleEndian.Uint32(buf), true
	}

	// 读取远程主机发送的数据
	for {
		dataType, success := readByte()
		if !success {
			return
		}

		socketDataType := common.GetSocketDataType(dataType)
		readRemoteDataStart := time.Now().UnixNano()
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("%s read data type: %v\n", common.LocalNodeId, socketDataType)
		}

		switch dataType {
		case common.ExchangeNodeInfo:
			// 节点ID
			nodeIdBuf, success := readStr()
			if !success {
				return
			}
			nodeId := string(nodeIdBuf)
			remoteNodeId = nodeId

			// 端口号
			remotePortBuf, success := readBytes(4)
			if !success {
				return
			}
			remotePort := binary.LittleEndian.Uint32(remotePortBuf)

			// HTTP端口号
			remoteHTTPPort, success := readUint32()
			if !success {
				return
			}

			// IP地址
			remoteIp := strings.Split(c.RemoteAddr().String(), ":")[0]

			remoteNode := node.Node{
				NodeId:   nodeId,
				Conn:     c,
				Ip:       remoteIp,
				TCPPort:  remotePort,
				HTTPPort: remoteHTTPPort,
			}
			node.AddNode(remoteNode) // 添加节点
			if tog.LogLevel(tog.INFO) {
				log.Println("Connected to remote node:", remoteNode)
			}

			// 如果是新连接的节点并且当前节点是leader，则需要设置远程节点的nextIndex
			if common.Role == common.Leader {
				node.UpdateNextIndexByNodeId(remoteNodeId, common.GetEntriesLength())
			}

			if addr, ok := c.LocalAddr().(*net.TCPAddr); ok {
				if uint(addr.Port) != common.Port {
					continue // 不是服务器就不进行广播
				}
			} else {
				log.Fatal("Current address convert failed")
			}

			// 把此节点广播给其它节点
			nodes := node.GetNodes()
			for _, n := range nodes {
				nodeConn := n.Conn
				if nodeConn != nil {
					info := []byte{common.ShareNodes}
					info = append(info, common.AddBufHead(nodeIdBuf)...)
					info = append(info, common.AddBufHead([]byte(remoteIp))...)
					info = append(info, remotePortBuf...)
					_, err := nodeConn.Write(info)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		case common.ShareNodes:
			// 为什么不需要使用Gossip这样的算法呢，因为raft这样的系统一般不会直接存储海量的数据，而是存储一些meta
			// 数据、或者作为另一个集群的master来使用，所以节点数量不会太多，因此不需要Gossip这样的节点间数据同步协议
			nodeIdBuf, success := readStr()
			if !success {
				return
			}
			nodeId := string(nodeIdBuf)

			remoteIPBuf, success := readStr()
			if !success {
				return
			}
			remoteIp := string(remoteIPBuf)

			remotePort, success := readUint32()
			if !success {
				return
			}

			// 只有这个节点不存在于节点列表的时候才需要去连接
			if !node.ExistNode(nodeId) {
				Connect(fmt.Sprintf("%s:%d", remoteIp, remotePort))
			}
		case common.AppendEntries:
			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s(me) Get AppendEntries from %s\n", common.LocalNodeId, remoteNodeId)
			}
			// leader的任期
			leaderTerm, success := readUint32()
			if !success {
				return
			}
			// leader所记录的当前follower的最后一个log索引
			leaderPrevLogIndex, success := readUint32()
			if !success {
				return
			}
			// leader所记录的当前follower的最后一个log索引的任期
			leaderPrevLogTerm, success := readUint32()
			if !success {
				return
			}
			// leader的commitIndex
			leaderCommittedIndex, success := readUint32()
			if !success {
				return
			}
			// leader的此次append是否成功
			appendSuccess := true
			if leaderTerm < common.CurrentTerm {
				appendSuccess = false
			}

			entries := common.GetEntries()
			if uint32(len(entries)) > leaderPrevLogIndex {
				// leader记录的当前节点最后一个log的term和本地的不一致，appendEntries失败
				if entry := entries[leaderPrevLogIndex]; entry.Term != leaderPrevLogTerm {
					appendSuccess = false
				}
			} else {
				// 如果leader记录的当前节点index超出限制，也是一种不匹配
				appendSuccess = false
			}
			// 此次AppendEntries的entries长度
			appendEntriesLength, success := readUint32()
			if !success {
				return
			}
			// 遍历所有的entry
			for i := uint32(0); i < appendEntriesLength; i++ {
				keyBuf, success := readStr()
				if !success {
					return
				}
				key := string(keyBuf)

				valueBuf, success := readStr()
				if !success {
					return
				}
				value := string(valueBuf)

				// 该Entry所对应的任期
				term, success := readUint32()
				if !success {
					return
				}

				// 该Entry在log中的位置
				index, success := readUint32()
				if !success {
					return
				}

				createTime, success := readUint32()
				if !success {
					return
				}

				if tog.LogLevel(tog.INFO) {
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
				common.LeaderNodeId = remoteNodeId // 设置leader节点
			}

			// AppendEntries的响应
			var response = []byte{common.AppendEntriesResponse}
			if appendSuccess {
				response = append(response, byte(1))
			} else {
				response = append(response, byte(0))
			}

			response = append(response, common.Uint32ToBytes(common.CurrentTerm)...)
			_, err := c.Write(response)
			if err != nil {
				log.Fatal(err)
			}

			if tog.LogLevel(tog.DEBUG) {
				log.Printf("AppendEntries leaderPrevLogIndex: %d, leaderPrevLogTerm: %d, leaderCommittedIndex: %d,"+
					" entriesLength: %d, result: %v, resultTerm: %d\n", leaderPrevLogIndex, leaderPrevLogTerm,
					leaderCommittedIndex, appendEntriesLength, appendSuccess, common.CurrentTerm)
			}
		case common.AppendEntriesResponse:
			resSuccessBuf, success := readByte()
			if !success {
				return
			}
			resSuccess := true
			if resSuccessBuf == 0 {
				resSuccess = false
			}

			term, success := readUint32()
			if !success {
				return
			}
			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s Get AppendEntriesResponse from %s, term: %d, local term: %d, success: %t\n",
					common.LocalNodeId, remoteNodeId, term, common.CurrentTerm, resSuccess)
			}
			if term > common.CurrentTerm {
				if tog.LogLevel(tog.WARN) {
					log.Printf("Update current term %d->%d and become follower\n", common.CurrentTerm, term)
				}
				common.ChangeTerm(term)
				common.ChangeRole(common.Follower)
			}

			n := node.GetNode(remoteNodeId)
			n.AppendSuccess <- resSuccess // 获取follower的返回结果并通过channel进行同步
		case common.VoteRequest:
			if common.Role == common.Follower {
				// 重置超时定时器
				common.HeartbeatTimeoutCh <- true
			}
			candidateTerm, success := readUint32()
			if !success {
				return
			}

			lastEntryIndex, success := readUint32()
			if !success {
				return
			}

			lastEntryTerm, success := readUint32()
			if !success {
				return
			}

			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s(me) term %d -> remote %s term %d ",
					common.LocalNodeId, common.CurrentTerm, remoteNodeId, candidateTerm)
			}

			voteSuccess := false

			// 大于当前的任期
			if candidateTerm >= common.CurrentTerm {
				common.ChangeTerm(candidateTerm)
				common.ChangeRole(common.Follower)
				// 尚未投票或者投给了candidate
				if nodeId, ok := common.VoteFor[candidateTerm]; !ok || nodeId == remoteNodeId {
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
						common.LocalNodeId, remoteNodeId, common.CurrentTerm)
				}
				response = append(response, byte(1)) // 投票
			} else {
				response = append(response, byte(0)) // 不投票
			}

			response = append(response, common.Uint32ToBytes(common.CurrentTerm)...)
			_, err := c.Write(response)
			if err != nil {
				log.Fatal(err)
			}
		case common.VoteResponse:
			voteBuf, success := readByte()
			if !success {
				return
			}

			term, success := readUint32()
			if !success {
				return
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
		}

		readRemoteDataEnd := time.Now().UnixNano()
		readRemoteDataCost := readRemoteDataEnd - readRemoteDataStart
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("Cost %dms %dns, %s\n", readRemoteDataCost/1e6, readRemoteDataCost, socketDataType)
		}
	}
}

// 发送当前节点的信息给远程节点
func sendNodeInfo(c net.Conn) {
	localNodeIdBuf := common.AddBufHead([]byte(common.LocalNodeId))
	portBuf := common.Uint32ToBytes(uint32(common.Port))
	httpPortBuf := common.Uint32ToBytes(uint32(common.HTTPPort))

	nodeInfo := []byte{common.ExchangeNodeInfo}    // 操作类型
	nodeInfo = append(nodeInfo, localNodeIdBuf...) // 节点id
	nodeInfo = append(nodeInfo, portBuf...)        // TCP服务端口
	nodeInfo = append(nodeInfo, httpPortBuf...)    // HTTP服务端口

	// 一旦与远程主机连接，立即告知其自己的节点信息
	_, err := c.Write(nodeInfo)
	if err != nil {
		log.Fatal(err)
	}
}
