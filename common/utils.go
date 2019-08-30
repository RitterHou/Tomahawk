// 工具

package common

import (
	"../tog"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

// 生成一个在指定范围内的随机整数
func RandomInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

// 生成一个指定长度的随机字符串
func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// 将一个uint32的数字转化为字节数组
func Uint32ToBytes(num uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, num)
	return buf
}

// 对数据包进行编码，加上头部信息
func AddBufHead(buf []byte) []byte {
	length := len(buf)
	head := make([]byte, 0, 1)
	if length < 0xff {
		head = append(head, byte(length))
	} else {
		head = append(head, byte(0xff))
		head = append(head, Uint32ToBytes(uint32(length))...)
	}
	body := append(head, buf...)
	return body
}

// 获取本地的网卡IP地址（可能不准确，因为无法保证数据包一定是从这块网卡发出去的）
func GetLocalIp() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, i := range interfaces {
		if i.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if i.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addresses, err := i.Addrs()
		if err != nil {
			return ""
		}
		for _, addr := range addresses {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String()
		}
	}
	return ""
}

// 更新角色的状态
func ChangeRole(role RoleType) {
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s(me) Change role from %v to %v\n", LocalNodeId, Role, role)
	}

	Role = role

	// 角色变成了follower，重置记录的leader节点
	if role == Follower {
		LeaderNodeId = ""
	}

	if role == Candidate {
		LeaderNodeId = ""
	}

	// 当前节点为leader节点
	if role == Leader {
		LeaderNodeId = LocalNodeId
	}
}

// 更新选举任期
func ChangeTerm(term uint32) {
	if tog.LogLevel(tog.DEBUG) {
		log.Printf("%s(me) Change term from %v to %v\n", LocalNodeId, CurrentTerm, term)
	}

	if CurrentTerm > term {
		log.Fatalf("Can't change term from %d to %d", CurrentTerm, term)
	}

	CurrentTerm = term
}

// 取出两个数中的较小值
func Min(m, n uint32) uint32 {
	if m < n {
		return m
	}
	return n
}

// 取出两个数中的较大值
func Max(m, n uint32) uint32 {
	if m < n {
		return n
	}
	return m
}

// 获取当前时间戳
func MakeTimestamp() uint32 {
	return uint32(time.Now().Unix())
}

// 根据网络传输类型获取相应的字符串，主要是为了打印好看
func GetSocketDataType(socketType byte) string {
	switch socketType {
	case ExchangeNodeInfo:
		return "ExchangeNodeInfo"
	case ShareNodes:
		return "ShareNodes"
	case AppendEntries:
		return "AppendEntries"
	case AppendEntriesResponse:
		return "AppendEntriesResponse"
	case VoteRequest:
		return "VoteRequest"
	case VoteResponse:
		return "VoteResponse"
	default:
		return fmt.Sprintf("Unknown Socket Type: %d", socketType)
	}
}
