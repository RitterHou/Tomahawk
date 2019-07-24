package network

import (
	"fmt"
	"log"
	"net"
)

func Connect(host string) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		if err.Error() == "dial tcp "+host+": connect: connection refused" {
			log.Println("Connection refused by node:", host)
			return
		}
		log.Fatal(err)
	}
	handleRequest(conn)
}

// 监听在本地的指定端口
func Listen(port uint) {
	host := fmt.Sprintf("0.0.0.0:%d", port)
	l, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening host", host)
	defer func() {
		err = l.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	for {
		conn, err := l.Accept()
		log.Println(conn)
		if err != nil {
			log.Fatal(err)
		}
		//go handleRequest(conn)
	}
}

func handleRequest(c net.Conn) {

}
