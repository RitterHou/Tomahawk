package http

import (
	"../node"
	"fmt"
	"log"
	"net/http"
)

// 启动HTTP服务器
func StartHttpServer(port uint) {
	// 显示所有的节点信息
	http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		nodes := node.GetNodes()
		for _, n := range nodes {
			_, err := fmt.Fprintf(w, "%s\n", n.NodeId)
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	log.Println("HTTP Server Listening", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
}
