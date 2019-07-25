package network

import (
	"../common"
	"../node"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
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

	localNodeIdBuf := common.AddBufHead([]byte(common.LocalNodeId))
	portBuf := common.Uint32ToBytes(uint32(common.Port))
	httpPortBuf := common.Uint32ToBytes(uint32(common.HTTPPort))

	nodeInfo := []byte{common.ExchangeNodeId}      // 操作类型
	nodeInfo = append(nodeInfo, localNodeIdBuf...) // 节点id
	nodeInfo = append(nodeInfo, portBuf...)        // TCP服务端口
	nodeInfo = append(nodeInfo, httpPortBuf...)    // HTTP服务端口

	// 一旦与远程主机连接，立即告知其自己的节点信息
	_, err := c.Write(nodeInfo)
	if err != nil {
		log.Println(err)
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
			log.Println("Remote node:", remoteNode)

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
			if !node.NodeExist(nodeId) {
				Connect(fmt.Sprintf("%s:%d", remoteIp, remotePort))
			}
		}
	}
}
