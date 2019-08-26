package http

import (
	"../common"
	"../node"
	"../tog"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 启动HTTP服务器
func StartHttpServer(port uint) {
	http.HandleFunc("/", root)                  // 根目录
	http.HandleFunc("/nodes", nodes)            // 显示节点信息
	http.HandleFunc("/entries", handlerEntries) // 读写entries
	http.HandleFunc("/rikka", rikka)            // rikka图片
	http.HandleFunc("/favicon.ico", favicon)    // favicon.ico

	if tog.LogLevel(tog.INFO) {
		log.Println("HTTP Server Listening Port", port)
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
}

// 根路径，显示服务器信息
func root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" { // 路径不等于根路径，说明该路径未能被匹配，返回404
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		sendResponse(w, common.NotFoundHtml)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodGet {
		sendResponse(w, "Only allow method [GET].")
		return
	}

	sendResponse(w, fmt.Sprintf("Build TimeStamp : %s\n", common.BuildStamp))
	sendResponse(w, fmt.Sprintf("Version         : %s\n", common.Version))
	sendResponse(w, fmt.Sprintf("Go Version      : %s\n", common.GoVersion))
}

// 显示节点信息
func nodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		sendResponse(w, "Only allow method [GET].")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	table := ""

	nodes := node.GetNodes()
	sort.Sort(nodes) // 对节点列表进行排序
	for _, n := range nodes {
		star := " "
		if n.NodeId == common.LeaderNodeId {
			star = "*"
		}
		me := " "
		if n.NodeId == common.LocalNodeId {
			me = "▴"
		}
		table += fmt.Sprintf(common.NodePageTableTemplate, star, me, n.NodeId, n.Ip, n.HTTPPort)
	}
	sendResponse(w, fmt.Sprintf(common.NodesPage, strings.Replace(table, " ", "&nbsp;", -1)))
}

// 读写数据
func handlerEntries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	switch r.Method {
	case http.MethodGet: // 查询数据
		key := r.URL.Query().Get("key")

		// key为空，返回所有的数据
		if key == "" {
			entries := common.GetEntries()
			from, size := getFromAndSize(r.URL.Query().Get("from"), r.URL.Query().Get("size"), len(entries))
			sendResponse(w, fmt.Sprintf("Total entries: %d, from: %d, size: %d\n", len(entries), from, size))
			for _, entry := range entries[from : from+size] {
				sendResponse(w, fmt.Sprintf(`{"key": "%s", "value": "%s", "index": "%d", "term": "%d", "time": "%s"}`+"\n",
					entry.Key, entry.Value, entry.Index, entry.Term,
					time.Unix(int64(entry.Time), 0).Format("2006-01-02 15:04:05")))
			}
			return
		}

		// key不为空，返回指定数据
		value, ok := common.GetEntryByKey(key)
		if ok {
			sendResponse(w, fmt.Sprintf(`Result for key [%s]: {"%s": "%s"}`, key, key, value))
		} else {
			w.WriteHeader(http.StatusNotFound) // 没找到相应的key
			sendResponse(w, fmt.Sprintf("can't found key [%s]", key))
		}
	case http.MethodPost: // 写入数据
		if common.LeaderNodeId == "" {
			sendResponse(w, "CLUSTER HAS DOWN AND YOU CAN'T POST ANY DATA!!!")
			return
		}

		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
		bodyBuf, err := ioutil.ReadAll(r.Body) // 获取HTTP请求的body
		if err != nil {
			log.Fatal(err)
		}

		// 仅可以向leader写数据
		if common.LocalNodeId != common.LeaderNodeId {
			forwardPostDataToLeader(w, bodyBuf)
			return
		}

		// 把数据写入到了本地的logEntries中
		var entries []common.Entry
		err = json.Unmarshal(bodyBuf, &entries) // 优先解析JSON数组
		if err != nil {
			var entry common.Entry
			err := json.Unmarshal(bodyBuf, &entry) // 如果数组解析失败，则解析JSON对象
			if err != nil {
				http.Error(w, "Post body can't be decode to json: "+string(bodyBuf), 400)
				return
			}
			entries = make([]common.Entry, 1)
			entries[0] = entry
		}

		response := ""
		for _, e := range entries {
			if e.Key == "" {
				sendResponse(w, "Post failed because key can't be nil")
				return
			}
			response += fmt.Sprintf("Post Success: {\"%s\": \"%s\"}\n", e.Key, e.Value)
		}

		// 设置响应内容
		sendResponse(w, response)

		// 把entries加入到leader本地的log[]中
		common.AppendEntryList(entries)
		// 因为leader主动向follower发送了数据，所以此周期内不再需要主动发送心跳
		common.LeaderSendEntryCh <- true

		// leader向follower发送消息
		appendSuccess := make(chan bool)
		node.SendAppendEntries(appendSuccess)

		<-appendSuccess // 如果大部分的follower返回，则leader返回给client
	default:
		sendResponse(w, "Only allow method [GET, POST].")
	}
}

// 返回图片
func rikka(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		sendResponse(w, "Only allow method [GET].")
		return
	}
	w.Header().Set("Content-Length", common.RikkaLength)
	w.Header().Set("Content-Type", "image/png")
	_, err := w.Write(common.Rikka)
	if err != nil {
		log.Fatal(err)
	}
}

// 返回图标
func favicon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		sendResponse(w, "Only allow method [GET].")
		return
	}
	w.Header().Set("Content-Length", common.IconLength)
	_, err := w.Write(common.Icon)
	if err != nil {
		log.Fatal(err)
	}
}

// 对发送响应的行为做了封装
func sendResponse(w http.ResponseWriter, content string) {
	_, err := fmt.Fprint(w, content)
	if err != nil {
		log.Printf("%v", debug.Stack())
		log.Fatal(err)
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
	status, header, body := postJson(leaderUrl, bodyBuf)
	w.WriteHeader(status)
	for k, v := range header {
		w.Header().Set(k, strings.Join(v, " ")) // TODO 此时是直接把多个value进行合并的，不清楚是否合适
	}
	_, err := w.Write(body)
	if err != nil {
		log.Fatal(err)
	}
}

// POST JSON data
func postJson(url string, data []byte) (int, http.Header, []byte) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	return res.StatusCode, res.Header, body
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
