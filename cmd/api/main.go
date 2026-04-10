package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	handler "github.com/ikennarichard/genderize/internal/api"
)

func main() {
    mux := http.NewServeMux()
      mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Up and running")
  })
    mux.HandleFunc("GET /api/classify", handler.Classify)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    server := &http.Server{
        Addr:         ":" + port,
        Handler:      mux,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    log.Printf("Server starting on :%s", port)
    log.Fatal(server.ListenAndServe())
}