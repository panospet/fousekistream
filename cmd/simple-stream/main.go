package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/caarlos0/env/v11"
)

type config struct {
	Filename string `env:"FILENAME" envDefault:"./fousekis_all.mp3"`
	Port     int    `env:"PORT" envDefault:"8000"`
}

func main() {
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Error parsing environment variables: %v", err)
	}

	http.HandleFunc(
		"/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/mpeg")
			for {
				file, err := os.Open(cfg.Filename)
				if err != nil {
					log.Printf("Error opening file: %v", err)
					return
				}
				_, err = copyBuffer(w, file)
				file.Close()
				if err != nil {
					log.Println("Client disconnected or error occurred:", err)
					return
				}
			}
		},
	)

	log.Printf("Streaming on http://localhost:%v", cfg.Port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil)
	if err != nil {
		log.Fatal("Server error:", err)
	}
}

func copyBuffer(dst http.ResponseWriter, src *os.File) (written int64, err error) {
	buf := make([]byte, 4096)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if ew != nil {
				return int64(nw), ew
			}
			if nr != nw {
				return int64(nw), io.ErrShortWrite
			}
		}
		if er != nil {
			if er.Error() == "EOF" {
				return written, nil
			}
			return written, er
		}
		written += int64(nr)
	}
}
