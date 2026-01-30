package main

import (
	"fmt"
	"net/http"
	"time"
)

func startServer(port string, delay time.Duration, name string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		fmt.Printf("[%s] Request received\n", name)
		w.Write([]byte("Hello from " + name))
	})

	fmt.Printf("Backend %s started on port %s (Delay: %s)\n", name, port, delay)
	http.ListenAndServe(port, mux)
}

func main() {
	go startServer(":8081", time.Second, "Server 1 (Fast)")
	go startServer(":8083", 2*time.Second, "Server 2 (Slow)")
	go startServer(":8082", 5*time.Second, "Server 3 (Really Slow)")
	select {}
}
