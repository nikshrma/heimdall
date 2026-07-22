// Package dummybakcend here is the dummy backend used to test and debug HEIMDALL
package dummybackend

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(Response{
			Name:      name,
			Path:      r.URL.Path,
			Method:    r.Method,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	})
	log.Printf("%s listening on :%s (%s)", name, port, endpoint)
	http.ListenAndServe(":"+port, mux)
}
