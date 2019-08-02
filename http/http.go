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

// 客户端发送的数据
type entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// 给所有的节点发送数据
func sendDataToFollowers(nodes []node.Node, data []byte) {
	for _, n := range nodes {
		if n.Conn != nil {
			_, err := n.Conn.Write(data)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

// 发送entries信息给follower，如果entries为空则为心跳
func sendAppendEntries(entries []common.Entry) {
	entriesLength := 0 // entries的长度，默认为零
	if entries != nil {
		entriesLength = len(entries)
	}
	data := append([]byte{common.AppendEntries}, common.Uint32ToBytes(common.CurrentTerm)...)
	data = append(data, common.Uint32ToBytes(common.PrevLogIndex)...)
	data = append(data, common.Uint32ToBytes(common.PrevLogTerm)...)
	data = append(data, common.Uint32ToBytes(common.CommittedIndex)...)
	data = append(data, common.Uint32ToBytes(uint32(entriesLength))...)

	for i := 0; i < entriesLength; i++ {
		// TODO 对Entry进行编码，目前因为只考虑心跳，长度皆为零所以暂时不需要
	}

	nodes := node.GetNodes()
	sendDataToFollowers(nodes, data)
}

// 启动HTTP服务器
func StartHttpServer(port uint) {
	// 显示服务器信息
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		if r.Method != http.MethodGet {
			_, err := fmt.Fprintln(w, "Only allow method [GET].")
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		_, err := fmt.Fprintf(w, "Build TimeStamp : %s\n", common.BuildStamp)
		if err != nil {
			log.Fatal(err)
		}
		_, err = fmt.Fprintf(w, "Version         : %s\n", common.Version)
		if err != nil {
			log.Fatal(err)
		}
		_, err = fmt.Fprintf(w, "Go Version      : %s\n", common.GoVersion)
		if err != nil {
			log.Fatal(err)
		}
	})

	// 显示所有的节点信息
	http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		if r.Method != http.MethodGet {
			_, err := fmt.Fprintln(w, "Only allow method [GET].")
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		_, err := fmt.Fprintln(w, "       NodeId      Host")
		if err != nil {
			log.Fatal(err)
		}

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
			_, err := fmt.Fprintf(w, "%s%s %10s %15s:%-5d\n", star, me, n.NodeId, n.Ip, n.HTTPPort)
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	// 读写数据
	http.HandleFunc("/entries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		switch r.Method {
		case http.MethodGet:
			key := r.URL.Query().Get("key")
			_, err := fmt.Fprintf(w, `{"%s": "%s"}`, key, "233333")
			if err != nil {
				log.Fatal(err)
			}
		case http.MethodPost:
			// 仅可以向leader写数据
			if common.LocalNodeId != common.LeaderNodeId {
				leaderHttp := ""
				for _, n := range node.GetNodes() {
					if n.NodeId == common.LeaderNodeId {
						leaderHttp = fmt.Sprintf("http://%s:%d/", n.Ip, n.HTTPPort)
					}
				}
				_, err := fmt.Fprintf(w, `This node is not leader, please post data to leader %s: %s`,
					common.LeaderNodeId, leaderHttp)
				if err != nil {
					log.Fatal(err)
				}
				return
			}

			if r.Body == nil {
				http.Error(w, "Please send a request body", 400)
				return
			}

			bodyBuf, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Fatal(err)
			}
			body := string(bodyBuf)

			var eArray []entry
			err = json.Unmarshal(bodyBuf, &eArray) // 优先解析JSON数组
			if err != nil {
				var e entry
				err := json.Unmarshal(bodyBuf, &e) // 如果数组解析失败，则解析JSON对象
				if err != nil {
					//http.Error(w, err.Error(), 400)
					http.Error(w, "Post body can't be decode to json: "+body, 400)
					return
				}
				eArray = make([]entry, 1)
				eArray[0] = e
			}

			var entries = make([]common.Entry, 0)
			for _, e := range eArray {
				_, err = fmt.Fprintf(w, "Post Success: {\"%s\": \"%s\"}\n", e.Key, e.Value)
				if err != nil {
					log.Fatal(err)
				}
			}

			//common.AppendEntry(key, value)

			common.LeaderSendEntryCh <- true // leader向follower发送数据，此周期内不再需要主动发送心跳
			sendAppendEntries(entries)
		default:
			_, err := fmt.Fprintln(w, "Only allow method [GET, POST].")
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	if tog.LogLevel(tog.INFO) {
		log.Println("HTTP Server Listening Port", port)
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
}
