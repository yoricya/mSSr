package main

import (
	"bufio"
	"flag"
	"fmt"
	_ "github.com/wzshiming/shadowsocks/init"
	_ "github.com/wzshiming/shadowsocks/stream/chacha20"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var BanList = StringList{
	items:  []*Word{},
	cached: make(map[string]bool),
}

func TidyConnect(conn net.Conn, logStr string, host string) {
	var isProxing = !isUsingBanList
	if isUsingBanList && BanList.Contains(host) {
		isProxing = true
	}

	//Verbose Log
	if isVerbose {
		log.Println(logStr, isProxing)
	}
	//VLog END

	var targetConn net.Conn
	var err error
	if isProxing {
		dialler, e := GetProxyDialler(host)
		if e != nil {
			if isVerbose {
				log.Println("Error while get proxy dialer for '"+host+"' : ", e)
			}
			return
		}
		targetConn, err = dialler.Dial("tcp", host)
	} else {
		targetConn, err = net.Dial("tcp", host)
	}

	if isVerbose && err != nil {
		log.Println(logStr+" ERROR: ", err)
		return
	}

	go io.Copy(targetConn, conn)
	defer targetConn.Close()
	io.Copy(conn, targetConn)
}

var isVerbose = false
var isUsingBanList = false
var server_port = 8080

func main() {
	p := flag.Int("port", 8080, "Proxy Server port")
	isB := flag.Bool("banlist", false, "Using ban list?")
	v := flag.Bool("v", false, "Verbose?")
	ver := flag.Bool("version", false, "Version")

	flag.Parse()

	if *ver {
		fmt.Println("V0.1")
		return
	}

	rand.Seed(time.Now().UnixNano())

	isVerbose = *v
	server_port = *p

	//GOOS=linux GOARCH=amd64 go build

	//Proxy Worker
	{
		fmt.Println("Preparing and checking proxies...")
		file, err := os.Open("proxieslist.txt")
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)
		var wg sync.WaitGroup

		for scanner.Scan() {
			t := scanner.Text()
			wg.Add(1)

			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}

			go func() {
				err := AddProxy(t)
				if err == nil {
					fmt.Println("Avail proxy: " + t)
				} else {
					fmt.Println("Not Avail proxy: " + t)
				}
				wg.Done()
			}()
		}

		file.Close()
		wg.Wait()
	}

	//Ban list worker
	if *isB {
		fmt.Println("Preparing banlist...")
		isUsingBanList = true
		file, err := os.Open("banlist.txt")
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			t := scanner.Text()
			t = strings.TrimSpace(t)

			if t == "" {
				continue
			}

			BanList.Add(t)
			fmt.Println("Add to banlist: " + t)
		}

		file.Close()
	}

	httpProxy(server_port)
}
