package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
)

type config struct {
	Filename string `env:"FILENAME" envdefault:"fousekis_all.mp3"`
}

type broadcaster struct {
	clients map[chan []byte]struct{}
	mtx     sync.Mutex
}

func main() {
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v", err)
	}

	mp3File, err := os.ReadFile(cfg.Filename)
	if err != nil {
		log.Fatalf("failed to read file %s: %v", cfg.Filename, err)
	}

	bc := &broadcaster{
		clients: make(map[chan []byte]struct{}),
	}

	// Broadcast loop
	go func() {
		for {
			for _, chunk := range chunkData(mp3File, 1600) {
				bc.broadcast(chunk)
				time.Sleep(100 * time.Millisecond) // The 1600B chunks are sent every 100ms, creating a bitrate of 128kbps
			}
		}
	}()

	http.HandleFunc(
		"/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/mpeg")

			clientChan := make(chan []byte, 10) // Buffered channel to avoid blocking
			bc.addClient(clientChan)
			defer bc.removeClient(clientChan)

			notify := r.Context().Done()

			for {
				select {
				case chunk := <-clientChan:
					_, err := w.Write(chunk)
					if err != nil {
						return // Client gone, exit handler
					}
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				case <-notify:
					close(clientChan) // Close channel to signal client disconnect
					return            // Client closed connection, exit handler
				}
			}
		},
	)

	http.HandleFunc(
		"/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Welcome to the radio-style stream server!"))
		},
	)

	log.Printf("Serving stream :8088")
	if err := http.ListenAndServe(":8088", nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

// Chunks file to array of byte slices
func chunkData(data []byte, size int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += size {
		end := i + size
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

func (b *broadcaster) addClient(c chan []byte) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.clients[c] = struct{}{}
	log.Printf("new client connected, total clients: %d", len(b.clients))
}

func (b *broadcaster) removeClient(c chan []byte) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	delete(b.clients, c)
	log.Printf("client disconnected, total clients: %d", len(b.clients))
}

func (b *broadcaster) broadcast(data []byte) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for client := range b.clients {
		select {
		case client <- data:
		default: // Skip if client buffer is full
		}
	}
}
