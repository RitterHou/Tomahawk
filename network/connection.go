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
				if tog.LEVEL >= tog.WARN {
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
		case common.ShareNodes: // TODO 节点同步只存在一轮，可能存在节点丢失的情况
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
			leaderTerm := binary.LittleEndian.Uint32(leaderTermBuf)

			leaderPrevLogIndexBuf, success := read(4)
			if !success {
				return
			}
			leaderPrevLogIndex := binary.LittleEndian.Uint32(leaderPrevLogIndexBuf)

			leaderPrevLogTermBuf, success := read(4)
			if !success {
				return
			}
			leaderPrevLogTerm := binary.LittleEndian.Uint32(leaderPrevLogTermBuf)

			leaderCommittedIndexBuf, success := read(4)
			if !success {
				return
			}
			leaderCommittedIndex := binary.LittleEndian.Uint32(leaderCommittedIndexBuf)

			log.Printf("leaderPrevLogIndex: %d, leaderPrevLogTerm: %d, leaderCommittedIndex: %d\n",
				leaderPrevLogIndex, leaderPrevLogTerm, leaderCommittedIndex)

			appendEntriesLengthBuf, success := read(4)
			if !success {
				return
			}
			appendEntriesLength := binary.LittleEndian.Uint32(appendEntriesLengthBuf)
			for i := uint32(0); i < appendEntriesLength; i++ {
				// TODO 对Entry进行解码，目前因为只考虑心跳，长度皆为零所以暂时不需要
			}

			if leaderTerm < common.CurrentTerm {
				if tog.LogLevel(tog.WARN) {
					log.Printf("Remote %s(%d) < local %s(%d) but sent entries\n",
						remoteNodeId, leaderTerm, common.LocalNodeId, common.CurrentTerm)
				}
			}

			switch common.Role {
			case common.Leader:
				if tog.LogLevel(tog.WARN) {
					log.Printf("local %s(%d) is leader but remote %s(%d) sent entries\n",
						common.LocalNodeId, common.CurrentTerm, remoteNodeId, leaderTerm)
				}
			case common.Candidate:
				// 虽然我是候选人，但是别人的任期比我高，我选举失败重新变成follower
				if leaderTerm >= common.CurrentTerm {
					common.VoteSuccessCh <- false
					common.CurrentTerm = leaderTerm
					common.LeaderNodeId = remoteNodeId // 设置leader节点
				}
			case common.Follower:
				// 重置超时定时器
				common.HeartbeatTimeoutCh <- true
				common.CurrentTerm = leaderTerm
				common.LeaderNodeId = remoteNodeId // 设置leader节点
			}

			// AppendEntries的响应
			var response = []byte{common.AppendEntriesResponse}
			if true {
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
		case common.VoteRequest:
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
				if common.Votes > uint32(common.Quorum) {
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
