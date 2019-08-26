// 一些HTTP服务器所需要的工具方法

package http

import (
	"../common"
	"../node"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// 对发送响应的行为做了封装
func sendResponse(w http.ResponseWriter, content string) {
	_, err := fmt.Fprint(w, content)
	if err != nil {
		log.Printf("%v", debug.Stack())
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 因为当前节点是follower，所以将数据转发给leader
func forwardPostDataToLeader(w http.ResponseWriter, bodyBuf []byte) {
	leaderUrl := ""
	for _, n := range node.GetNodes() {
		if n.NodeId == common.LeaderNodeId {
			leaderUrl = fmt.Sprintf("http://%s:%d/entries", n.Ip, n.HTTPPort)
		}
	}
	// 进行entries转发，因为当前节点不是leader，把entries转发到leader节点上去
	status, header, body, err := postJson(leaderUrl, bodyBuf)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	for k, v := range header {
		w.Header().Set(k, strings.Join(v, ", ")) // TODO 此时是直接把多个value进行合并的，不清楚是否合适
	}
	_, err = w.Write(body)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// POST JSON data
func postJson(url string, data []byte) (int, http.Header, []byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, nil, nil, err
	}

	err = res.Body.Close()
	if err != nil {
		return 0, nil, nil, err
	}
	return res.StatusCode, res.Header, body, nil
}

// 解析得到合适的from和size属性的值
func getFromAndSize(fromStr, sizeStr string, entriesLength int) (from, size int) {
	var err error
	from, err = strconv.Atoi(fromStr)
	if err != nil {
		from = 0
	}
	size, err = strconv.Atoi(sizeStr)
	if err != nil {
		size = 10
	}

	if from >= entriesLength || from < 0 {
		from = 0
	}

	if size > entriesLength {
		size = entriesLength
	}

	if size < 0 {
		size = 10
	}

	if from+size > entriesLength {
		size = entriesLength - from
	}
	return
}

// 从字节转化为兆
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// 获取内存信息
// 参考：https://golangcode.com/print-the-current-memory-usage/
func getMemoryInfo() (uint64, uint64, uint64, uint32) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC
}

// 获取当前进程Goroutine的数量
func getGoroutineNum() int {
	return runtime.NumGoroutine()
}
