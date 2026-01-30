package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL               *url.URL
	ReverseProxy      *httputil.ReverseProxy
	Alive             bool
	mux               sync.RWMutex
	ActiveConnections uint64
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

type EndPoints struct {
	LBport  string   `json:"lbport"`
	Servers []string `json:"servers"`
}

func (s *ServerPool) GetNextPeer() *Backend {
	var bestPeer *Backend = nil
	var lowestActive uint64 = math.MaxUint64

	for _, b := range s.backends {
		if b.IsAlive() {
			conns := atomic.LoadUint64(&b.ActiveConnections)
			if conns < lowestActive {
				lowestActive = conns
				bestPeer = b
			}
		}
	}
	if bestPeer != nil {
		atomic.AddUint64(&bestPeer.ActiveConnections, 1)
	}
	return bestPeer
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
	b.mux.RLock()
	b.Alive = alive
	b.mux.RUnlock()
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.Alive
	b.mux.RUnlock()
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
		fmt.Printf("%s %s [%s] Active: %d\n", time.Now().Format("15:04:05"), b.URL, status, b.ActiveConnections)
	}
}

func (s *ServerPool) StartHealthCheck() {
	for {
		s.HealthCheck()
		time.Sleep(10 * time.Second)
	}
}

func (e *EndPoints) LoadEnv(s *ServerPool) {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()
	decoder := json.NewDecoder(jsonFile)
	err = decoder.Decode(e)
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(e.Servers); i++ {
		s.AddBackend(e.Servers[i])
	}
}

func main() {
	serverPool := &ServerPool{}

	var endPoints EndPoints
	endPoints.LoadEnv(serverPool)

	go serverPool.StartHealthCheck()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		peer := serverPool.GetNextPeer()

		if peer != nil {
			fmt.Printf("Redirecting to: %s (Active: %d)\n", peer.URL, atomic.LoadUint64(&peer.ActiveConnections))
			defer func() {
				atomic.AddUint64(&peer.ActiveConnections, ^uint64(0))
			}()
			peer.ReverseProxy.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
	})

	fmt.Printf("Load Balancer running on port %s\n", endPoints.LBport)
	if err := http.ListenAndServe(endPoints.LBport, nil); err != nil {
		panic(err)
	}
}
