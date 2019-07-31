package http

import (
	"../common"
	"../node"
	"../tog"
	"fmt"
	"log"
	"net/http"
	"sort"
)

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
			err := r.ParseForm()
			if err != nil {
				log.Fatal(err)
			}
			key := r.Form.Get("key")
			value := r.Form.Get("value")
			_, err = fmt.Fprintf(w, `Post Success: {"%s": "%s"}`, key, value)
			if err != nil {
				log.Fatal(err)
			}
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
