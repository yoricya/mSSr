package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/wzshiming/shadowsocks"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Proxies хранилище для Dialer'ов
var Proxies []*shadowsocks.Dialer
var DiallerItems = make(map[string]*shadowsocks.Dialer)
var mu sync.RWMutex

// AddProxy добавляет новый Dialer в пул
func AddProxy(proxyLink string) error {
	dialler, err := shadowsocks.NewDialer(proxyLink)
	if err != nil {
		return err
	}

	// Проверка доступности прокси
	ctx, cancel := context.WithTimeout(context.Background(), 3500*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://2ip.ru", nil)
	req.Header.Set("User-Agent", "curl/7.5.2")
	client := &http.Client{
		Transport: &http.Transport{DialContext: dialler.DialContext},
		Timeout:   3500 * time.Millisecond,
	}

	_, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("Proxy not working: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	Proxies = append(Proxies, dialler)
	return nil
}

// getProxyIndex вычисляет индекс прокси для данного домена
func getProxyIndex(domain string) int {
	hash := sha256.Sum256([]byte(domain))
	return int(hash[0]) % len(Proxies)
}

// GetProxyDialler получает Dialer для указанного домена
func GetProxyDialler(domain string) (*shadowsocks.Dialer, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))

	mu.RLock()
	dialler := DiallerItems[domain]
	mu.RUnlock()

	if dialler != nil {
		return dialler, nil
	}

	index := getProxyIndex(domain)
	dialler = Proxies[index]

	mu.Lock()
	DiallerItems[domain] = dialler
	mu.Unlock()

	return dialler, nil
}
