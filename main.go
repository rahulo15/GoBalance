package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
	Alive        bool
	mux          sync.RWMutex
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

func (s *ServerPool) GetNextPeer() *Backend {
	next := atomic.AddUint64(&s.current, 1)
	length := uint64(len(s.backends))

	for i := 0; i < int(length); i++ {
		idx := (int(next) + i) % int(length)
		if s.backends[idx].IsAlive() {
			if i != 0 {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

func (s *ServerPool) AddBackend(serverURL string) {
	u, _ := url.Parse(serverURL)
	proxy := httputil.NewSingleHostReverseProxy(u)

	s.backends = append(s.backends, &Backend{
		URL:          u,
		ReverseProxy: proxy,
		Alive:        true,
	})
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mux.Lock()
	alive := b.Alive
	b.mux.Unlock()
	return alive
}

func (s *ServerPool) HealthCheck() {
	for _, b := range s.backends {
		status := "up"
		alive := true

		con, err := net.DialTimeout("tcp", b.URL.Host, 2*time.Second)
		if err != nil {
			status = "DOWN"
			alive = false
		} else {
			con.Close()
		}

		b.SetAlive(alive)
		fmt.Printf("%s [%s]\n", b.URL, status)
	}
}

func (s *ServerPool) StartHealthCheck() {
	for {
		s.HealthCheck()
		time.Sleep(20 * time.Second)
	}
}

func main() {
	serverPool := &ServerPool{}

	serverPool.AddBackend("http://localhost:8081")
	serverPool.AddBackend("http://localhost:8082")
	serverPool.AddBackend("http://localhost:8083")

	go serverPool.StartHealthCheck()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		peer := serverPool.GetNextPeer()

		if peer != nil {
			fmt.Printf("Redirecting to: %s\n", peer.URL)
			peer.ReverseProxy.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
	})

	fmt.Println("Load Balancer running on port :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
