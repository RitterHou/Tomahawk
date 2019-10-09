// 一些与网络连接相关的操作

package network

import (
	"../tog"
	"encoding/binary"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

// 创建一个包装类型的连接
func NewConn(c net.Conn) *Conn {
	return &Conn{c: c, closed: false}
}

// 对基础的网络连接对象进行封装
type Conn struct {
	c            net.Conn   // 对应的TCP连接
	closeLock    sync.Mutex // 关闭连接的时候需要加锁
	closed       bool       // 记录连接是否已经关闭
	remoteNodeId string     // 该连接所对应的远程节点ID
}

func (conn *Conn) SetRemoteNodeId(remoteNodeId string) {
	conn.remoteNodeId = remoteNodeId
}

func (conn *Conn) GetRemoteNodeId() string {
	return conn.remoteNodeId
}

// 获取连接的本地地址
func (conn *Conn) LocalAddr() string {
	return conn.c.LocalAddr().String()
}

// 获取连接的远程地址
func (conn *Conn) RemoteAddr() string {
	return conn.c.RemoteAddr().String()
}

// 关闭连接
func (conn *Conn) Close() error {
	defer conn.closeLock.Unlock()
	conn.closeLock.Lock()

	if conn.closed {
		if tog.LogLevel(tog.DEBUG) {
			log.Printf("Connection has been closed %s -> %s\n", conn.LocalAddr(), conn.RemoteAddr())
		}
		return nil
	}

	if tog.LogLevel(tog.INFO) {
		log.Printf("Colse connection %s -> %s\n", conn.LocalAddr(), conn.RemoteAddr())
	}

	err := conn.c.Close()
	if err != nil { // 关闭失败
		return err
	}

	conn.closed = true
	return nil
}

// 获取TCP连接本地所对应的端口
func (conn *Conn) LocalPort() uint {
	if addr, ok := conn.c.LocalAddr().(*net.TCPAddr); ok {
		return uint(addr.Port)
	}
	log.Panic("Current address convert failed")
	return 0
}

// 向远程主机写数据
func (conn *Conn) Write(data []byte) bool {
	_, err := conn.c.Write(data)
	if err != nil {
		if tog.LogLevel(tog.WARN) {
			log.Printf("Write data error: %v\n", err)
		}
		// 关闭连接
		err = conn.Close()
		if err != nil {
			log.Panic(err)
		}
		return false
	}
	return true
}

// 对数据读取做了简单的封装
func (conn *Conn) ReadBytes(length uint32) ([]byte, bool) {
	data := make([]byte, length)
	_, err := conn.c.Read(data)
	if err != nil {
		if err == io.EOF || strings.ContainsAny(err.Error(), "connection reset by peer") {
			if err == io.EOF {
				if tog.LogLevel(tog.WARN) {
					log.Println("connection closed by remote:", conn.RemoteAddr())
				}
			} else {
				if tog.LogLevel(tog.WARN) {
					log.Println(err)
				}
			}
			return nil, false
		}
		log.Panic(err)
	}
	return data, true
}

// 读取一个字节
func (conn *Conn) ReadByte() (byte, bool) {
	buf, success := conn.ReadBytes(1)
	if !success {
		return 0, false
	}
	return buf[0], true
}

// 读取一个字符串
func (conn *Conn) ReadStr() ([]byte, bool) {
	head, success := conn.ReadByte()
	if !success {
		return nil, success
	}
	if head < 0xff {
		body, success := conn.ReadBytes(uint32(head))
		return body, success
	} else {
		lengthBuf, success := conn.ReadBytes(4)
		if !success {
			return nil, success
		}
		length := binary.LittleEndian.Uint32(lengthBuf)
		body, success := conn.ReadBytes(length)
		return body, success
	}
}

// 读取一个uint32数字
func (conn *Conn) ReadUint32() (uint32, bool) {
	buf, success := conn.ReadBytes(4)
	if !success {
		return 0, false
	}
	return binary.LittleEndian.Uint32(buf), true
}
