package common

import (
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"sync"
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

func Uint32ToBytes(num uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, num)
	return buf
}

// 数据数据包进行编码，加上头部信息
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

// 从一段被编码的数据中解析出数据包，即 AddBufHead 的逆操作
// 返回值：解析出来的报文，被解析数据的完整长度
func ParseBuf(buf []byte) ([]byte, uint32) {
	offset := uint32(1)
	head := buf[0]
	if head < 0xff {
		offset += uint32(head)
		body := buf[1 : 1+head]
		return body, offset
	} else {
		length := binary.LittleEndian.Uint32(buf[1:])
		offset = offset + 4 + length
		body := buf[5 : 5+length]
		return body, offset
	}
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

// 所有的数据
var logEntries = []Entry{{Key: "", Value: "", Term: 0, Index: 0}} // 初始化一条数据可以简化一些操作
var entryMutex sync.Mutex

// 只有leader可以这样顺序的append entry数据
func AppendEntryList(entryList []Entry) []Entry {
	if Role != Leader {
		return nil
	}
	entryMutex.Lock()
	for i := 0; i < len(entryList); i++ {
		entryList[i].Term = CurrentTerm
		entryList[i].Index = GetEntriesLength() // Index自增
	}
	logEntries = append(logEntries, entryList...)
	entryMutex.Unlock()
	return entryList // 被append的数据
}

//// 根据key获取entry
//func GetEntryByKey(key string) string {
//	defer entryMutex.Unlock()
//	entryMutex.Lock()
//	for i := len(logEntries); i > 0; i-- {
//		if logEntries[i].Key == key {
//			return logEntries[i].Value
//		}
//	}
//	return ""
//}

// 根据索引获取Entry
func GetEntryByIndex(index uint32) Entry {
	return logEntries[index]
}

// 获取当前Entries的长度
func GetEntriesLength() uint32 {
	return uint32(len(logEntries))
}

// 根据索引设置Entry
func SetEntry(entry Entry) {
	// 理论上来说，当前节点的entries最多只能比index低一位
	if GetEntriesLength() == entry.Index {
		logEntries = append(logEntries, entry)
	} else {
		logEntries[entry.Index] = entry
	}
}

// 如果产生冲突，从当前节点进行裁剪
func CutoffEntries(index uint32) {
	if Role != Follower {
		log.Println("只有follower才会被裁掉entries")
		return
	}
	logEntries = logEntries[:index]
}

func GetEntries() []Entry {
	return logEntries
}

func GetLastEntry() Entry {
	return logEntries[len(logEntries)-1]
}

// 把entry编码为字节数组
func EncodeEntry(entry Entry) []byte {
	data := make([]byte, 0)
	data = append(data, AddBufHead([]byte(entry.Key))...)
	data = append(data, AddBufHead([]byte(entry.Value))...)
	data = append(data, Uint32ToBytes(entry.Term)...)
	data = append(data, Uint32ToBytes(entry.Index)...)
	return data
}

func Min(m, n uint32) uint32 {
	if m < n {
		return m
	}
	return n
}

func Max(m, n uint32) uint32 {
	if m < n {
		return n
	}
	return m
}
