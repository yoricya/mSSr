package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func parseHTTPRequest(conn net.Conn) (method string, host string, err error) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", "", err
	}

	request := string(buffer[:n])
	lines := strings.Split(request, "\r\n")
	if len(lines) == 0 {
		return "", "", fmt.Errorf("invalid request")
	}

	parts := strings.Fields(lines[0])
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid request line")
	}

	method, host = parts[0], parts[1]

	if method != "GET" {
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "Host:") {
				host = strings.TrimSpace(strings.TrimPrefix(line, "Host:"))
				break
			}
		}
	}

	return method, host, nil
}

func handleHTTPSConnection(conn net.Conn) {
	defer conn.Close()

	method, h, err := parseHTTPRequest(conn)
	if err != nil {
		log.Println("[HTTP] Error reading request:", err)
		conn.Write([]byte("HTTP/1.1 426 Upgrade Required\r\n"))
		conn.Write([]byte("Upgrade: HTTP/1.1\r\n"))
		conn.Write([]byte("Connection: Upgrade\r\n\r\n"))
		return
	}

	if method == http.MethodGet {
		if h == "/proxy.pac" {
			handlePac(conn)
		}
		return
	} else if method == http.MethodConnect {
		handleProxy(h, conn)
		return
	}
}

func handleProxy(host string, conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	TidyConnect(conn, "[HTTPS] Connect: client "+remoteAddr+" -> "+host+" http? =>", host)
}

func handlePac(conn net.Conn) {
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
	pac := "function FindProxyForURL(url, host){" +
		"	 let res = dnsResolve(host);\n" +
		"    if (isPlainHostName(host) ||\n " +
		"       shExpMatch(host, \"*.local\") ||\n" +
		"        isInNet(res, \"10.0.0.0\", \"255.0.0.0\") ||\n " +
		"       isInNet(res, \"172.16.0.0\", \"255.240.0.0\") ||\n" +
		"        isInNet(res, \"192.168.0.0\", \"255.255.0.0\") ||\n " +
		"       isInNet(res, \"127.0.0.0\", \"255.255.255.0\")) {\n " +
		"       return \"DIRECT\";\n    }\n" +
		"	return \"PROXY " + server_pac_addr + ":" + strconv.Itoa(https_port) + "\";" +
		"\n}"
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
