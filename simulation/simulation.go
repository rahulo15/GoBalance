package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	go runStableServer("8081")

	go runChaosServer("8082")

	go runZombieServer("8083")

	select {}
}

// ---------------------------------------------------------
// SERVER TYPE 1: STABLE
// ---------------------------------------------------------
func runStableServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("😇 STABLE Server (%s)", port)))
	})

	fmt.Printf("✅ Stable Server running on %s\n", port)
	http.ListenAndServe(":"+port, mux)
}

// ---------------------------------------------------------
// SERVER TYPE 2: CHAOS (Random Failures)
// ---------------------------------------------------------
func runChaosServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		risk := rand.Intn(100)

		if risk < 25 {
			fmt.Printf("[%s] 💥 CHAOS: Killing Connection!\n", port)
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "Error", 500)
				return
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}

		if risk >= 25 && risk < 50 {
			fmt.Printf("[%s] 🐢 CHAOS: Lagging (2s)...\n", port)
			time.Sleep(2 * time.Second)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("😈 CHAOS Server (%s)", port)))
	})

	fmt.Printf("⚠️  Chaos Server running on %s\n", port)
	http.ListenAndServe(":"+port, mux)
}

// ---------------------------------------------------------
// SERVER TYPE 3: ZOMBIE (Dies and Restarts)
// ---------------------------------------------------------
func runZombieServer(port string) {
	for {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("🧟 ZOMBIE Server (%s)", port)))
		})

		server := &http.Server{Addr: ":" + port, Handler: mux}

		go func() {
			fmt.Printf("✅ [%s] Zombie is ALIVE\n", port)
			server.ListenAndServe()
		}()

		time.Sleep(time.Duration(rand.Intn(4)+8) * time.Second)

		fmt.Printf("💀 [%s] Zombie DIED (Rebooting in 5s)...\n", port)
		server.Close()

		time.Sleep(5 * time.Second)
	}
}
