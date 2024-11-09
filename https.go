package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"strconv"
)

func handleHTTPSConnection(conn net.Conn) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Println("[HTTPS] Error reading request:", err)
		return
	}

	// Обработка HTTPS
	if req.Method == http.MethodConnect {
		handleHTTPSConnect(req, conn)
	} else {
		conn.Close()
	}
}

func handleHTTPSConnect(req *http.Request, conn net.Conn) {
	host := req.Host
	remoteAddr := conn.RemoteAddr().String()

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	TidyConnect(conn, "[HTTPS] Connect: client "+remoteAddr+" -> "+host+" http? =>", host)
}

func httpProxy(port int) {
	//Start server
	log.Println("tunProxy https proxy started at 0.0.0.0:" + strconv.Itoa(port))
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	//Working server
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleHTTPSConnection(conn)
	}
}
