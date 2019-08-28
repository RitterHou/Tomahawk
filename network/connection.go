// 一些与网络连接相关的操作

package network

import (
	"../tog"
	"encoding/binary"
	"io"
	"log"
	"net"
	"strings"
)

// 创建一个包装类型的连接
func NewConn(c net.Conn) *Conn {
	return &Conn{c: c, closed: false}
}

// 对基础的网络连接对象进行封装
type Conn struct {
	c      net.Conn
	closed bool
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
	if tog.LogLevel(tog.INFO) {
		log.Printf("colse connection %s -> %s\n", conn.LocalAddr(), conn.RemoteAddr())
	}

	if conn.closed {
		return nil
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
	log.Fatal("Current address convert failed")
	return 0
}

// 向远程主机写数据
func (conn *Conn) Write(data []byte) error {
	_, err := conn.c.Write(data)
	return err
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
		log.Fatal(err)
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
