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
)

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
		err := c.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

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
		return
	}

	// 记录此连接所对应的远程节点id，可以作为一个索引方便后续操作
	var remoteNodeId string

	// 对数据读取做了简单的封装
	read := func(length uint32) ([]byte, bool) {
		data := make([]byte, length)
		_, err := c.Read(data)
		if err != nil {
			if err == io.EOF {
				if tog.LogLevel(tog.WARN) {
					log.Println("connection closed by remote:", c.RemoteAddr())
				}
				node.RemoveNodeById(remoteNodeId)
				return nil, false
			}
			log.Fatal(err)
		}
		return data, true
	}

	// 进一步包装数据的读取，包括了头部信息的解析
	readBuf := func() ([]byte, bool) {
		headBuf, success := read(1)
		if !success {
			return nil, success
		}
		head := headBuf[0]
		if head < 0xff {
			body, success := read(uint32(head))
			return body, success
		} else {
			lengthBuf, success := read(4)
			if !success {
				return nil, success
			}
			length := binary.LittleEndian.Uint32(lengthBuf)
			body, success := read(length)
			return body, success
		}
	}

	// 读取远程主机发送的数据
	for {
		dataType, success := read(1)
		if !success {
			return
		}

		switch dataType[0] {
		case common.ExchangeNodeInfo:
			// 节点ID
			nodeIdBuf, success := readBuf()
			if !success {
				return
			}
			nodeId := string(nodeIdBuf)
			remoteNodeId = nodeId

			// 端口号
			remotePortBuf, success := read(4)
			if !success {
				return
			}
			remotePort := binary.LittleEndian.Uint32(remotePortBuf)

			// HTTP端口号
			remoteHTTPPortBuf, success := read(4)
			if !success {
				return
			}
			remoteHTTPPort := binary.LittleEndian.Uint32(remoteHTTPPortBuf)

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
			nodeIdBuf, success := readBuf()
			if !success {
				return
			}
			nodeId := string(nodeIdBuf)

			remoteIPBuf, success := readBuf()
			if !success {
				return
			}
			remoteIp := string(remoteIPBuf)

			remotePortBuf, success := read(4)
			if !success {
				return
			}
			remotePort := binary.LittleEndian.Uint32(remotePortBuf)

			// 只有这个节点不存在于节点列表的时候才需要去连接
			if !node.ExistNode(nodeId) {
				Connect(fmt.Sprintf("%s:%d", remoteIp, remotePort))
			}
		case common.AppendEntries:
			leaderTermBuf, success := read(4)
			if !success {
				return
			}
			// leader的任期
			leaderTerm := binary.LittleEndian.Uint32(leaderTermBuf)

			leaderPrevLogIndexBuf, success := read(4)
			if !success {
				return
			}
			// leader所记录的当前follower的最后一个log索引
			leaderPrevLogIndex := binary.LittleEndian.Uint32(leaderPrevLogIndexBuf)

			leaderPrevLogTermBuf, success := read(4)
			if !success {
				return
			}
			// leader所记录的当前follower的最后一个log索引的任期
			leaderPrevLogTerm := binary.LittleEndian.Uint32(leaderPrevLogTermBuf)

			leaderCommittedIndexBuf, success := read(4)
			if !success {
				return
			}
			// leader的commitIndex
			leaderCommittedIndex := binary.LittleEndian.Uint32(leaderCommittedIndexBuf)

			log.Printf("leaderPrevLogIndex: %d, leaderPrevLogTerm: %d, leaderCommittedIndex: %d\n",
				leaderPrevLogIndex, leaderPrevLogTerm, leaderCommittedIndex)

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

			appendEntriesLengthBuf, success := read(4)
			if !success {
				return
			}
			// 此次AppendEntries的entries长度
			appendEntriesLength := binary.LittleEndian.Uint32(appendEntriesLengthBuf)
			// 遍历所有的entry
			for i := uint32(0); i < appendEntriesLength; i++ {
				keyBuf, success := readBuf()
				if !success {
					return
				}
				key := string(keyBuf)

				valueBuf, success := readBuf()
				if !success {
					return
				}
				value := string(valueBuf)

				termBuf, success := read(4)
				if !success {
					return
				}
				// 该Entry所对应的任期
				term := binary.LittleEndian.Uint32(termBuf)

				indexBuf, success := read(4)
				if !success {
					return
				}
				// 该Entry在log中的位置
				index := binary.LittleEndian.Uint32(indexBuf)

				log.Printf("AppendEntries from leader, key: %s, value: %s, term: %d, index: %d\n",
					key, value, term, index)

				// 如果可以把entry append到当前节点中
				if appendSuccess {
					entries = common.GetEntries() // 获取当前节点的当前entries
					// 一旦产生冲突，从当前节点开始进行cutOff
					if uint32(len(entries)) > index && entries[index].Term != term {
						common.CutoffEntries(index)
					}
					// 把entry保存到follower中去
					common.SetEntryByIndex(index, common.Entry{Key: key, Value: value, Index: index, Term: term})
				}
			}

			// 根据leader的committedIndex更新当前节点的committedIndex
			if leaderCommittedIndex > common.CommittedIndex {
				common.CommittedIndex = common.Min(leaderCommittedIndex, uint32(len(common.GetEntries())))
			}

			switch common.Role {
			case common.Leader:
				if leaderTerm > common.CurrentTerm {
					common.Role = common.Follower
				}
			case common.Candidate:
				if leaderTerm >= common.CurrentTerm {
					common.Role = common.Follower
					common.VoteSuccessCh <- false
				}
			case common.Follower:
				if appendSuccess {
					// 重置超时定时器
					common.HeartbeatTimeoutCh <- true
					common.CurrentTerm = leaderTerm
					common.LeaderNodeId = remoteNodeId // 设置leader节点
				}
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
		case common.AppendEntriesResponse:
			resSuccessBuf, success := read(1)
			if !success {
				return
			}
			resSuccess := true
			if resSuccessBuf[0] == 0 {
				resSuccess = false
			}

			termBuf, success := read(4)
			if !success {
				return
			}
			term := binary.LittleEndian.Uint32(termBuf)

			if tog.LogLevel(tog.DEBUG) {
				log.Printf("AppendEntriesResponse, term: %d, success: %t\n", term, resSuccess)
			}

			n := node.GetNode(remoteNodeId)
			n.AppendSuccess <- resSuccess // 获取follower的返回结果并通过channel进行同步
		case common.VoteRequest:
			if common.Role == common.Follower {
				// 重置超时定时器
				common.HeartbeatTimeoutCh <- true
			}
			candidateTermBuf, success := read(4)
			if !success {
				return
			}
			candidateTerm := binary.LittleEndian.Uint32(candidateTermBuf)

			lastEntryIndexBuf, success := read(4)
			if !success {
				return
			}
			lastEntryIndex := binary.LittleEndian.Uint32(lastEntryIndexBuf)

			lastEntryTermBuf, success := read(4)
			if !success {
				return
			}
			lastEntryTerm := binary.LittleEndian.Uint32(lastEntryTermBuf)

			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s(me) term %d -> remote %s term %d ",
					common.LocalNodeId, common.CurrentTerm, remoteNodeId, candidateTerm)
			}

			voteSuccess := false

			// 大于当前的任期
			if candidateTerm >= common.CurrentTerm {
				common.CurrentTerm = candidateTerm
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
			voteBuf, success := read(1)
			if !success {
				return
			}

			termBuf, success := read(4)
			if !success {
				return
			}
			term := binary.LittleEndian.Uint32(termBuf)

			vote := voteBuf[0]
			if vote == 1 {
				atomic.AddUint32(&common.Votes, 1)
				if common.Votes > common.Quorum {
					common.VoteSuccessCh <- true
				}
			} else {
				if term > common.CurrentTerm {
					common.CurrentTerm = term
					common.VoteSuccessCh <- false
				}
			}
		}
	}
}
