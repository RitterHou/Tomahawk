package network

import (
	"../common"
	"../node"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

func Connect(host string) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		if err.Error() == "dial tcp "+host+": connect: connection refused" {
			log.Println("Connection refused by node:", host)
			return
		}
		log.Fatal(err)
	}
	go handleConnection(conn)
}

// 监听在本地的指定端口
func Listen(port uint) {
	host := fmt.Sprintf("0.0.0.0:%d", port)
	l, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening host", host)
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

	var localIp string
	var isServer bool
	if addr, ok := c.LocalAddr().(*net.TCPAddr); ok {
		localIp = addr.IP.String()
		localPort := uint(addr.Port)
		// 如果连接的port和本地监听的port一致，则说明这是个server
		if localPort == common.Port {
			isServer = true
		}
	} else {
		log.Fatal("Current address convert failed")
	}

	currentHost := fmt.Sprintf("%s:%d", localIp, common.Port)
	log.Println("Current host:", currentHost)

	nodeInfo := []byte{common.ExchangeNodeId}                                     // 操作类型
	nodeInfo = append(nodeInfo, common.AddBufHead([]byte(common.LocalNodeId))...) // 节点id
	nodeInfo = append(nodeInfo, common.AddBufHead([]byte(currentHost))...)        // 当前节点的host

	// 一旦与远程主机连接，立即告知其自己的nodeId
	_, err := c.Write(nodeInfo)
	if err != nil {
		log.Println(err)
		return
	}

	var remoteNodeId string // 此连接所对应的远程节点id

	// 对数据读取做了简单的封装
	read := func(length uint32) ([]byte, bool) {
		data := make([]byte, length)
		_, err := c.Read(data)
		if err != nil {
			if err == io.EOF {
				log.Println("connection closed by remote:", c.RemoteAddr())
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
		case common.ExchangeNodeId:
			nodeIdBuf, success := readBuf()
			if !success {
				return
			}
			nodeId := string(nodeIdBuf)
			remoteNodeId = nodeId

			remoteNode := node.Node{NodeId: nodeId, Conn: c}
			node.AddNode(remoteNode)

			log.Println(remoteNode)

			remoteHostBuf, success := readBuf()
			if !success {
				return
			}
			// 该连接所对应的远程主机的host
			// 注意：该host和此连接的remoteAddr不一定一致，因为远程主机有可能是客户端，而我们要求host的端口号必须为节点所监听的port
			remoteHost := string(remoteHostBuf)
			log.Println("Current connection remote host:", remoteHost)
			// 我们需要广播的就是这个客户端的数据，如果是客户端，那么与该客户端相连的节点已经知道这个节点了，不再需要数据广播
			// 所以只有服务端才需要同步？
			if isServer {
				log.Println("I am server")
				nodes := node.GetNodes()
				for _, n := range nodes {
					nodeConn := n.Conn
					if nodeConn != nil {
						info := []byte{common.ShareNodes}
						info = append(info, common.AddBufHead(nodeIdBuf)...)
						info = append(info, common.AddBufHead(remoteHostBuf)...)
						_, err := nodeConn.Write(info)
						if err != nil {
							log.Fatal(err)
						}
					}
				}
			}
		case common.ShareNodes:
			nodeIdBuf, success := readBuf()
			if !success {
				return
			}

			nodeId := string(nodeIdBuf)

			remoteHostBuf, success := readBuf()
			if !success {
				return
			}

			remoteHost := string(remoteHostBuf)

			if node.NodeExist(nodeId) {
				log.Println("Remote node has exist in current node:", nodeId)
			} else {
				// 只有这个节点不存在于节点列表的时候才需要去连接
				Connect(remoteHost)
			}
		}
	}
}
