package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/shadowsocks/go-shadowsocks2/core"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

//github.com/wzshiming/shadowsocks

func extractSSLinkData(ssLink string) (cipher, password, host string, err error) {
	ssLink = strings.TrimSpace(ssLink)

	if ssLink[:5] != "ss://" {
		return "", "", "", fmt.Errorf("Not a shadowsocks Link!")
	}
	ssLink = ssLink[5:]

	autData := ""

	//Trim Auth vs Host
	for i := 0; i < len(ssLink); i++ {
		if ssLink[i] == '@' {
			autData = ssLink[:i]
			host = ssLink[i+1:]
			break
		}
	}

	//Trim Host vs Comment or End line
	for i := 0; i < len(host); i++ {
		if host[i] == '#' || host[i] == '/' || host[i] == ' ' || host[i] == '\n' {
			host = host[:i]
			break
		}
	}

	autDataRaw, err := base64.RawStdEncoding.DecodeString(autData) // Используем RawStdEncoding
	if err != nil {
		return "", "", "", fmt.Errorf("Base64 Decode Error: %w", err)
	}
	autData = string(autDataRaw)

	//Trim Password And Cipher
	for i := 0; i < len(autData); i++ {
		if autData[i] == ':' {
			cipher = autData[:i]
			password = autData[i+1:]
			break
		}
	}

	return cipher, password, host, nil
}

func createSocksHeader(ipAddress string, port uint16) ([]byte, error) {
	var packet []byte

	// Определяем тип адреса
	ip := net.ParseIP(ipAddress)
	if ip.To4() != nil { // IPv4
		packet = append(packet, 0x01)
		packet = append(packet, ip.To4()...)
	} else if ip.To16() != nil { // IPv6
		packet = append(packet, 0x04)
		packet = append(packet, ip.To16()...)
	} else { // Доменное имя
		hostBytes := []byte(ipAddress)
		packet = append(packet, 0x03)
		packet = append(packet, byte(len(hostBytes)))
		packet = append(packet, hostBytes...)
	}

	packet = append(packet, make([]byte, 2)...)

	// Добавляем порт
	binary.BigEndian.PutUint16(packet[len(packet)-2:], port)

	return packet, nil
}

func AddProxy(proxyLink string) error {
	//Parse SS link
	cipherTp, password, host, e := extractSSLinkData(proxyLink)
	if e != nil {
		return e
	}

	//Create Cipher
	cipher, e := core.PickCipher(cipherTp, nil, password)
	if e != nil {
		return fmt.Errorf("cipher error: %w", e)
	}

	//Start Dial Proxy Server
	conn, err := net.DialTimeout("tcp", host, 3500*time.Millisecond)
	if err != nil {
		return err
	}

	//Create Cipher Dialler with conn
	conn = cipher.StreamConn(conn)

	//Create and send Socks Header
	header, err := createSocksHeader("2ip.io", 443)
	if err != nil {
		return err
	}
	conn.Write(header)

	//Create Dial Context With Proxy Conn
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return conn, nil
	}

	//Create Timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 3500*time.Millisecond)
	defer cancel()

	//Create http req
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://2ip.io", nil)
	req.Header.Set("User-Agent", "curl/7.5.2")
	client := &http.Client{
		Transport: &http.Transport{DialContext: dialContext},
	}

	//Start and check Http req
	startTime := time.Now().UnixMilli()
	b, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Proxy not working: %v", err)
	}

	//Get Ip
	rd, e := io.ReadAll(b.Body)
	if e != nil {
		return err
	}

	remoteIpOfProxy := string(rd)
	endTime := time.Now().UnixMilli() - startTime

	//Create Proxy struct
	px := &Proxy{
		SSlink:    proxyLink,
		pingTime:  int32(endTime),
		cipher:    cipher,
		proxyHost: host,
		remoteIp:  remoteIpOfProxy,
		mutex:     sync.RWMutex{},
	}

	//Add proxy
	Amu.Lock()
	AllProxies = append(AllProxies, px)
	Amu.Unlock()

	return nil
}

type Proxy struct {
	SSlink   string
	pingTime int32

	cipher    core.Cipher
	proxyHost string
	remoteIp  string

	mutex sync.RWMutex
}

type CachedProxy struct {
	proxy *Proxy

	host string
	port uint16

	mutex sync.RWMutex
}

var AllProxies []*Proxy
var Amu sync.RWMutex

var distribInt = 0
var DImu sync.Mutex

var CachedWithDomainProxies = make(map[string]*CachedProxy)
var Cmu sync.RWMutex

func DialWithProxy(net, originHost string) (net.Conn, error) {
	Cmu.RLock()
	cprx, is := CachedWithDomainProxies[originHost]
	Cmu.RUnlock()

	if is {
		conn, err := CreateDialler(cprx.proxy, cprx.host, cprx.port)
		if err == nil {
			return conn, nil
		}

		fmt.Println(err)
	}

	DImu.Lock()
	i := distribInt
	DImu.Unlock()

	i++
	if i > len(AllProxies)-1 {
		i = 0
	}

	DImu.Lock()
	distribInt = i
	DImu.Unlock()

	Amu.RLock()
	prx := AllProxies[i]
	Amu.RUnlock()

	host, port := parseHost(originHost)

	cprx = &CachedProxy{
		proxy: prx,
		host:  host,
		port:  uint16(port),
		mutex: sync.RWMutex{},
	}

	Cmu.Lock()
	CachedWithDomainProxies[originHost] = cprx
	Cmu.Unlock()

	return CreateDialler(cprx.proxy, cprx.host, cprx.port)
}

func CreateDialler(proxy *Proxy, host string, port uint16) (net.Conn, error) {
	//Create and send Socks Header
	header, err := createSocksHeader(host, port)
	if err != nil {
		return nil, err
	}

	//Start Dial Proxy Server
	conn, err := net.Dial("tcp", proxy.proxyHost)
	if err != nil {
		return nil, err
	}

	//Create Cipher Dialler with conn
	conn = proxy.cipher.StreamConn(conn)
	conn.Write(header)

	return conn, nil
}

func parseHost(origin string) (host string, port int) {
	for i := len(origin) - 1; i > 0; i-- {
		if origin[i] == ':' {
			host = strings.TrimSpace(origin[:i])
			port, _ = strconv.Atoi(strings.TrimSpace(origin[i+1:]))
			break
		}
	}

	return host, port
}
