package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
)

type config struct {
	Filename string `env:"FILENAME" envDefault:"./fousekis_all.mp3"`
	Port     int    `env:"PORT" envDefault:"8000"`
}

type client chan []byte

var (
	clients   = make(map[client]struct{})
	clientsMu sync.Mutex
)

func main() {
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Error parsing env vars: %v", err)
	}

	go broadcaster(cfg.Filename)

	http.HandleFunc("/", streamHandler)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Streaming on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func broadcaster(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}

	for {
		buf := make([]byte, 4096)

		for {
			n, err := file.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				broadcast(data)
			}

			if err == io.EOF {
				break // start over
			}
			if err != nil {
				log.Printf("Read error: %v", err)
				break
			}

			// Control stream pace (approximate)
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func broadcast(data []byte) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	for c := range clients {
		select {
		case c <- data:
		default:
			// Client too slow or disconnected
			close(c)
			delete(clients, c)
		}
	}
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/mpeg")

	c := make(client, 100)
	clientsMu.Lock()
	clients[c] = struct{}{}
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, c)
		clientsMu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Send data from broadcaster
	for data := range c {
		_, err := w.Write(data)
		if err != nil {
			return
		}
		flusher.Flush()
	}
}
