package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func main() {
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()
	r.HandleFunc("/api/generate", generateHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/health", healthHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/export", exportHandler).Methods("POST", "OPTIONS")
	
	// Применяем CORS middleware ко всем роутам
	handler := corsMiddleware(r)

	fmt.Println("Starting optimized server on :9000")
	log.Fatal(http.ListenAndServe(":9000", handler))
}
