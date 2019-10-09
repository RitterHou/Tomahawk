// 创建网络连接

package node

import (
	"../network"
	"../tog"
	"fmt"
	"log"
	"net"
)

// 连接远程主机
func Connect(host string) {
	c, err := net.Dial("tcp", host)
	if err != nil {
		if err.Error() == "dial tcp "+host+": connect: connection refused" {
			if tog.LogLevel(tog.WARN) {
				log.Println("Connection refused by node:", host)
			}
			return
		}
		log.Panic(err)
	}

	conn := network.NewConn(c)

	UpdateLocalIp(conn) // 主动连接别的节点时，可以更新当前节点的IP地址
	go handleConnection(conn)
}

// 监听在本地的指定端口
func Listen(port uint) {
	host := fmt.Sprintf("0.0.0.0:%d", port)
	l, err := net.Listen("tcp", host)
	if err != nil {
		log.Panic(err)
	}
	if tog.LogLevel(tog.INFO) {
		log.Println("TCP Server Listening Port", port)
	}
	defer func() {
		err = l.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}

		conn := network.NewConn(c)
		UpdateLocalIp(conn) // 当别的节点连接自己时，可以更新当前节点的IP地址
		go handleConnection(conn)
	}
}
