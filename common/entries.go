// 所有的与Entries相关的操作

package common

import (
	"log"
	"sync"
)

// 一条数据
type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Term  uint32 // 添加该条数据时的任期号
	Index uint32 // 在log中的索引
	Time  uint32 // 创建时间
}

// 所有的日志数据
var logEntries = []Entry{{Key: "", Value: "", Term: 0, Index: 0}} // 初始化一条数据可以简化一些操作
var entryMutex sync.Mutex

// 向logEntries中Append数据，只有leader可以这样顺序的append数据
func AppendEntryList(entryList []Entry) {
	if Role != Leader {
		log.Fatalf("Current node %s, and leader is %s, only leader can append entries",
			LocalNodeId, LeaderNodeId)
	}
	entryMutex.Lock()
	for i := 0; i < len(entryList); i++ {
		entryList[i].Term = CurrentTerm
		entryList[i].Index = GetEntriesLength() // Index自增
		entryList[i].Time = MakeTimestamp()
		logEntries = append(logEntries, entryList[i])
	}
	entryMutex.Unlock()
}

// 根据key获取entry
func GetEntryByKey(key string) (string, bool) {
	defer entryMutex.Unlock()
	entryMutex.Lock()
	for i := len(logEntries) - 1; i >= 0; i-- {
		if logEntries[i].Key == key {
			return logEntries[i].Value, true
		}
	}
	return "", false
}

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
	data = append(data, Uint32ToBytes(entry.Time)...)
	return data
}
