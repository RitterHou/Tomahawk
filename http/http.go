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
	"sort"
	"strconv"
	"strings"
	"time"
)

// 启动HTTP服务器
func StartHttpServer(port uint) {
	// 显示服务器信息
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, err := fmt.Fprint(w, common.NotFoundHtml)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		switch r.Method {
		case http.MethodGet:
			key := r.URL.Query().Get("key")
			if key == "" {
				entries := common.GetEntries()
				from, size := getFromAndSize(r.URL.Query().Get("from"), r.URL.Query().Get("size"), len(entries))
				_, err := fmt.Fprintf(w, "Total entries: %d, from: %d, size: %d\n", len(entries), from, size)
				if err != nil {
					log.Fatal(err)
				}
				for _, entry := range entries[from : from+size] {
					_, err := fmt.Fprintf(w, `{"key": "%s", "value": "%s", "index": "%d", "term": "%d", "time": "%s"}`+"\n",
						entry.Key, entry.Value, entry.Index, entry.Term,
						time.Unix(int64(entry.Time), 0).Format("2006-01-02 15:04:05"))
					if err != nil {
						log.Fatal(err)
					}
				}
				return
			}
			value, ok := common.GetEntryByKey(key)
			if ok {
				_, err := fmt.Fprintf(w, `{"%s": "%s"}`, key, value)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				w.WriteHeader(http.StatusNotFound) // 没找到相应的key
				_, err := fmt.Fprintf(w, "can't found key [%s]", key)
				if err != nil {
					log.Fatal(err)
				}
			}
		case http.MethodPost:
			if common.LeaderNodeId == "" {
				_, err := fmt.Fprint(w, "CLUSTER HAS DOWN AND YOU CAN'T POST ANY DATA!!!")
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

			// 仅可以向leader写数据
			if common.LocalNodeId != common.LeaderNodeId {
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
					w.Header().Set(k, strings.Join(v, " "))
				}
				_, err = w.Write(body)
				if err != nil {
					log.Fatal(err)
				}
				return
			}

			var entries []common.Entry
			err = json.Unmarshal(bodyBuf, &entries) // 优先解析JSON数组
			if err != nil {
				var entry common.Entry
				err := json.Unmarshal(bodyBuf, &entry) // 如果数组解析失败，则解析JSON对象
				if err != nil {
					//http.Error(w, err.Error(), 400)
					http.Error(w, "Post body can't be decode to json: "+string(bodyBuf), 400)
					return
				}
				entries = make([]common.Entry, 1)
				entries[0] = entry
			}

			response := ""
			for _, e := range entries {
				if e.Key == "" {
					_, err = fmt.Fprint(w, "Post failed because key can't be nil")
					if err != nil {
						log.Fatal(err)
					}
					return
				}
				response += fmt.Sprintf("Post Success: {\"%s\": \"%s\"}\n", e.Key, e.Value)
			}

			_, err = fmt.Fprint(w, response)
			if err != nil {
				log.Fatal(err)
			}

			// 把entries加入到leader本地的log[]中
			common.AppendEntryList(entries)

			// leader向follower发送数据，此周期内不再需要主动发送心跳
			common.LeaderSendEntryCh <- true
			// leader向follower发送消息
			node.SendAppendEntries()

			<-common.LeaderAppendSuccess // 如果大部分的follower返回，则leader返回给client
		default:
			_, err := fmt.Fprintln(w, "Only allow method [GET, POST].")
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	http.HandleFunc("/rikka", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, err := fmt.Fprintln(w, "Only allow method [GET].")
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		w.Header().Set("Content-Length", common.RikkaLength)
		w.Header().Set("Content-Type", "image/png")
		_, err := w.Write(common.Rikka)
		if err != nil {
			log.Fatal(err)
		}
	})

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, err := fmt.Fprintln(w, "Only allow method [GET].")
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		w.Header().Set("Content-Length", common.IconLength)
		_, err := w.Write(common.Icon)
		if err != nil {
			log.Fatal(err)
		}
	})

	if tog.LogLevel(tog.INFO) {
		log.Println("HTTP Server Listening Port", port)
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
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

	if size > entriesLength || size < 0 {
		size = 10
	}

	if from+size > entriesLength {
		size = entriesLength - from
	}
	return
}
