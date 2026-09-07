// Package dummybakcend here is the dummy backend used to test and debug HEIMDALL
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Response struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Method    string `json:"method"`
	Timestamp string `json:"timestamp"`
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	name := getenv("NAME", "Backend")
	port := getenv("PORT", "8080")
	endpoint := getenv("ENDPOINT", "/")
	delayProb, _ := strconv.ParseFloat(getenv("DELAY_PROB", "0"), 64)

	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chance := rand.Float64()
		if chance < delayProb {
			time.Sleep(3200 * time.Millisecond)
		}
		json.NewEncoder(w).Encode(Response{
			Name:      name,
			Path:      r.URL.Path,
			Method:    r.Method,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	log.Printf("%s listening on :%s (%s)", name, port, endpoint)
	http.ListenAndServe(":"+port, mux)
}
