# 🚀 GoBalance

A high-performance, fault-tolerant **Load Balancer** built from scratch in **Go**. 

Unlike simple Round-Robin load balancers, this project implements the **Least Connections** algorithm to efficiently distribute traffic across multiple backend servers, ensuring no single server is overwhelmed. It is designed to demonstrate core Distributed Systems concepts, including **Concurrency, Atomic Operations, Health Checks, and Fault Tolerance.**

---

## ✨ Key Features

* **⚡ Least Connections Strategy:** intelligently routes traffic to the server with the lowest number of active connections.
* **🛡️ Fault Tolerance & Self-Healing:**
    * **Active Health Checks:** A background process (goroutine) pings servers every 20 seconds to update their status.
    * **Passive Health Checks:** Instantly detects if a request fails (e.g., Connection Refused) and marks the server as `DOWN` immediately.
    * **Automatic Retries:** If a backend crashes during a request, the Load Balancer seamlessly intercepts the error and retries a healthy server. Users experience **Zero Downtime**.
* **🔒 Concurrency Safe:** Uses `sync/atomic` for connection counters and `sync.RWMutex` to protect shared state, preventing race conditions under high load.
* **🐳 Dockerized:** Includes a multi-stage Docker build resulting in a lightweight (~15MB) production image.

---

## 🛠️ Tech Stack

* **Language:** Go (Golang)
* **Standard Library:** `net/http`, `net/http/httputil` (Reverse Proxy), `sync`, `sync/atomic`, `context`.
* **Infrastructure:** Docker (Alpine Linux).

---

## 🚀 Getting Started

### Option 1: Run Locally (Quickest)

1.  **Start the Simulation Backends** (Terminal 1)
    ```bash
    go run simulation.go
    ```

2.  **Start the Load Balancer** (Terminal 2)
    ```bash
    go run main.go
    ```

3.  **Verify**
    Open [http://localhost:8080](http://localhost:8080).

---

### Option 2: Run with Docker

1.  **Start the Simulation Backends** (Terminal 1)
    *Keep this running on your host machine so the container has something to connect to.*
    ```bash
    go run simulation.go
    ```

2.  **Build the Docker Image** (Terminal 2)
    ```bash
    docker build -t go-load-balancer .
    ```

3.  **Run the Container**
    ```bash
    docker run -p 8080:8080 go-load-balancer
    ```

4.  **Verify**
    Open [http://localhost:8080](http://localhost:8080).
