package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
)

func socksProxy(port int) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error while starting SOCKS5: ", err)
		return
	}
	defer listener.Close()

	log.Println("nahuiDPI SOCKS5 proxy started at 0.0.0.0:" + strconv.Itoa(port))
	log.Println("Please setting SOCKS5 proxy in system")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleSocksConnection(conn)
	}
}

func handleSocksConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()

	// Handshake
	buf := make([]byte, 256)
	_, err := conn.Read(buf)
	if err != nil {
		if isVerbose {
			log.Println("[SOCKS] Error: client " + remoteAddr + ", error while reading request.")
		}
		return
	}

	if buf[0] != 0x05 {
		if isVerbose {
			log.Println("[SOCKS] Error: client " + remoteAddr + ", not supported socks version.")
		}
		return
	}

	// Отправляем ответ на handshake
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		if isVerbose {
			log.Println("[SOCKS] Error: client " + remoteAddr + ", error while send handshake.")
		}
		return
	}

	// Читаем запрос клиента
	_, err = conn.Read(buf)
	if err != nil {
		if isVerbose {
			log.Println("[SOCKS] Error: client " + remoteAddr + ", error while reading request.")
		}
		return
	}

	//Check is command CONNECT
	if buf[1] != 0x01 {
		if isVerbose {
			log.Println("[SOCKS] Error: client " + remoteAddr + " send no connect command.")
		}
		return
	}

	var addr string
	var port int

	switch buf[3] {
	case 0x01: // IPv4
		addr = fmt.Sprintf("%d.%d.%d.%d", buf[4], buf[5], buf[6], buf[7])
		port = int(buf[8])<<8 + int(buf[9])
	case 0x03: // Domain name
		addrLen := buf[4]
		addr = string(buf[5 : 5+addrLen])
		port = int(buf[5+addrLen])<<8 + int(buf[6+addrLen])
	case 0x04: // IPv6
		addr = net.IP(buf[4:20]).String() // 16 байт для IPv6
		port = int(buf[20])<<8 + int(buf[21])
	default:
		if isVerbose {
			log.Println("[SOCKS] Error: client " + remoteAddr + ", unsupported address type.")
		}
		return
	}

	// Отправляем ответ клиенту о успешном подключении
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		if isVerbose {
			log.Println("[SOCKS] Error: client "+remoteAddr+", error while send response:", err)
		}
		return
	}

	TidyConnect(conn, "[SOCKS] Connect: client "+remoteAddr+" -> "+addr+" socks? =>", addr+":"+strconv.Itoa(port))
}
