package http

import (
	"../common"
	"../node"
	"../tog"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
)

// 启动HTTP服务器
func StartHttpServer(port uint) {
	http.HandleFunc("/", root)                  // 根目录
	http.HandleFunc("/status", status)          // 获取节点的状态信息
	http.HandleFunc("/entries", handlerEntries) // 读写entries

	// 返回图片，不包含数据信息
	http.HandleFunc("/rikka", rikka)         // rikka图片
	http.HandleFunc("/favicon.ico", favicon) // favicon.ico

	if tog.LogLevel(tog.INFO) {
		log.Println("HTTP Server Listening Port", port)
	}
	log.Panic(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
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
					entry.Key, entry.Value, entry.Index, entry.Term, timestampFormat(entry.Time)))
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
			http.Error(w, "Please send a request body", http.StatusBadRequest)
			return
		}
		bodyBuf, err := ioutil.ReadAll(r.Body) // 获取HTTP请求的body
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
				http.Error(w, "Post body can't be decode to json: "+string(bodyBuf), http.StatusBadRequest)
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
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 获取节点的状态信息
func status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodGet {
		sendResponse(w, "Only allow method [GET].")
		return
	}

	local := r.URL.Query().Get("local")
	// 只查询当前节点的数据
	if local == "true" {
		localNode := node.GetNode(common.LocalNodeId)
		// Host
		sendResponse(w, fmt.Sprintf("Host:\t\t%s:%d\n", localNode.Ip, localNode.HTTPPort))
		// 角色
		sendResponse(w, fmt.Sprintf("Role:\t\t%v\n", common.Role))
		// 节点ID
		sendResponse(w, fmt.Sprintf("NodeId:\t\t%s\n", common.LocalNodeId))
		// goroutine的数量
		sendResponse(w, fmt.Sprintf("Goroutines:\t%d\n", getGoroutineNum()))
		// 内存信息
		alloc, totalAlloc, s, gc := getMemoryInfo()
		sendResponse(w, fmt.Sprintf("Memory:\t\tAlloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n",
			alloc, totalAlloc, s, gc))
		// 进程启动时间
		sendResponse(w, fmt.Sprintf("Start:\t\t%s\nRuntime:\t%s\n", timestampFormat(common.StartingTimeStamp),
			secondToHumanReadable(common.MakeTimestamp()-common.StartingTimeStamp)))
		return
	}

	// 查询所有节点的信息
	nodes := node.GetNodes()
	sort.Sort(nodes) // 对节点列表进行排序
	for _, n := range nodes {
		res, err := http.Get(fmt.Sprintf("http://%s:%d/status?local=true", n.Ip, n.HTTPPort))
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if res == nil {
			log.Println("Response is nil")
			return
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = res.Body.Close()
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sendResponse(w, string(body)+"\n")
	}
}
