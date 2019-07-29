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
	// 显示所有的节点信息
	http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
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

	if tog.LogLevel(tog.INFO) {
		log.Println("HTTP Server Listening Port", port)
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
}
