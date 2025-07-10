package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func registerEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		now := time.Now().UTC()
		json.NewEncoder(w).Encode(struct {
			Running   bool      `json:"running"`
			Timestamp time.Time `json:"timestamp"`
		}{
			Running:   true,
			Timestamp: now,
		})
	})
}
