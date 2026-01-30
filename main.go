package main

import (
	"context"
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

	backend := &Backend{
		URL:          u,
		ReverseProxy: proxy,
		Alive:        true,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		fmt.Printf("[%s] Request Failed: %s\n", u.Host, e.Error())

		backend.SetAlive(false)

		if r.Context().Value("retried") != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service Unavailable (Max retries reached)"))
			return
		}

		if peer := s.GetNextPeer(); peer != nil {
			fmt.Printf("... Retrying request on %s\n", peer.URL)

			ctx := context.WithValue(r.Context(), "retried", true)

			peer.ReverseProxy.ServeHTTP(w, r.WithContext(ctx))

			atomic.AddUint64(&peer.ActiveConnections, ^uint64(0))
			return
		}

		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service Unavailable (No healthy backends)"))
	}

	s.backends = append(s.backends, backend)
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
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
			currentConns := atomic.LoadUint64(&peer.ActiveConnections)
			fmt.Printf("[LB] Forwarding to %s | Active Conns: %d\n", peer.URL.Host, currentConns)
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
