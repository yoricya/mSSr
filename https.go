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
		log.Println("[HTTP] Error reading request:", err)
		return
	}

	if req.Method == http.MethodConnect {
		handleProxy(req, conn)
		return
	} else if req.Method == http.MethodGet {
		if req.RequestURI == "/proxy.pac" {
			handlePac(req, conn)
			return
		}
	}

	conn.Close()
}

func handleProxy(req *http.Request, conn net.Conn) {
	host := req.Host
	remoteAddr := conn.RemoteAddr().String()

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	TidyConnect(conn, "[HTTPS] Connect: client "+remoteAddr+" -> "+host+" http? =>", host)
}

func handlePac(req *http.Request, conn net.Conn) {
	if server_pac_addr == "" {
		conn.Write([]byte("HTTP/1.1 500 Connection Established\r\n"))
		conn.Write([]byte("Content-Type: text/plain\r\n"))
		conn.Write([]byte("\r\n"))
		conn.Write([]byte("Global Server Address not configured\r\n\r\n"))
		return
	}

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n"))
	conn.Write([]byte("Content-Type: application/x-ns-proxy-autoconfig\r\n"))
	conn.Write([]byte("\r\n"))
	pac := "function FindProxyForURL(url, host){\n\treturn \"PROXY " + server_pac_addr + ":" + strconv.Itoa(https_port) + "\";\n}"
	conn.Write([]byte(pac))
}

var https_port int

func httpProxy(port int) {
	//Start server
	https_port = port
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
