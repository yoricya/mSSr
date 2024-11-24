package main

import (
	"bufio"
	"flag"
	"fmt"
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

func TidyConnect(conn net.Conn, logStr string, originHost string) {
	if !strings.Contains(originHost, ":") {
		originHost = originHost + ":443"
	}

	//Parse host:port
	host, port := parseHost(originHost)

	//Check is host contains in ban list
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
		targetConn, err = DialWithProxy("tcp", host, uint16(port))
	} else {
		targetConn, err = net.Dial("tcp", originHost)
	}

	if err != nil {
		log.Println(GetPrefix("TidyConnect", colorBrightBlue, typeColorError)+logStr+" ERROR: ", err)
		return
	}

	go func() {
		defer targetConn.Close()
		io.Copy(targetConn, conn)
	}()

	io.Copy(conn, targetConn)
}

var isVerbose = false
var isUsingBanList = false
var server_port = 8080
var server_pac_addr = ""

func main() {
	p := flag.Int("port", 8080, "Proxy Server port")
	isB := flag.Bool("banlist", false, "Using ban list?")
	v := flag.Bool("v", false, "Verbose?")
	pacAddr := flag.String("pac", "", "Using PAC Autoconf? Set your global ip of proxy")

	ver := flag.Bool("version", false, "Version")

	flag.Parse()

	if *ver {
		fmt.Println("V0.4")
		return
	}

	rand.Seed(time.Now().UnixNano())

	isVerbose = *v
	server_port = *p
	server_pac_addr = *pacAddr

	//GOOS=linux GOARCH=amd64 go build

	//Proxy Worker
	{
		log.Println(GetPrefix("PROXIES WORKER", colorBrightPurple, typeColorInfo) + "Preparing and checking proxies...")
		file, err := os.Open("proxieslist.txt")
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)
		var wg sync.WaitGroup

		var workedProxies = 0
		var allProxies = 0
		var mu sync.Mutex

		for scanner.Scan() {
			t := scanner.Text()
			wg.Add(1)

			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}

			allProxies++

			go func() {
				err := AddProxy(t)
				if err == nil {
					log.Println(GetPrefix("PROXIES WORKER", colorBrightPurple, typeColorDone) + "Avail proxy: " + t)

					mu.Lock()
					workedProxies++
					mu.Unlock()
				} else {
					var e = ""
					if isVerbose {
						e = "\n" + err.Error()
					}
					log.Println(GetPrefix("PROXIES WORKER", colorBrightPurple, typeColorWarn) + "Not Avail proxy: " + t + "." + e)
				}
				wg.Done()
			}()
		}

		file.Close()
		wg.Wait()

		log.Printf(GetPrefix("PROXIES WORKER", colorBrightPurple, typeColorDone)+"Proxies count: Available/All - %d/%d \n", workedProxies, allProxies)
	}

	//Ban list worker
	if *isB {
		log.Println(GetPrefix("BANLIST WORKER", colorBrightPurple, typeColorInfo) + "Preparing banlist...")
		isUsingBanList = true
		file, err := os.Open("banlist.txt")
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)

		count := 0

		for scanner.Scan() {
			t := scanner.Text()
			t = strings.TrimSpace(t)

			if t == "" {
				continue
			}

			count++

			BanList.Add(t)
			log.Println(GetPrefix("BANLIST WORKER", colorBrightPurple, typeColorInfo) + "Add to banlist: " + t)
		}

		file.Close()
		log.Printf(GetPrefix("BANLIST WORKER", colorBrightPurple, typeColorDone)+"Banned sites count: %d", count)
	}

	httpProxy(server_port)
}
