package main

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

func registerEndpoints(mux *http.ServeMux, staticContent fs.FS, indexHTML []byte, mode, name, logfile *string) {
	// Static assets
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	// Homepage
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(indexHTML)
	})

	// Status API endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		_, stream, ok := getStatusAndLogs(*mode, *name, *logfile)
		now := time.Now().UTC()

		var logs []string
		if stream != nil {
			for _, v := range stream.Values {
				if len(v) > 1 {
					logs = append(logs, v[1])
				}
			}
		}

		response := MetricsResponse{
			Status:        ok,
			LastUpdatedAt: now,
			Logs:          logs,
		}

		json.NewEncoder(w).Encode(response)
	})

	// Health endpoint
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

	// Legacy data endpoint for backward compatibility with frontend
	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, stream, ok := getStatusAndLogs(*mode, *name, *logfile)
		timestamp := strings.ToLower(jsonTimeFormat())

		var logs []string
		if stream != nil {
			for _, v := range stream.Values {
				if len(v) > 1 {
					logs = append(logs, v[1])
				}
			}
		}
		logsString := strings.Join(logs, "\n")

		json.NewEncoder(w).Encode(struct {
			Status    string `json:"status"`
			StatusOK  bool   `json:"statusOK"`
			Logs      string `json:"logs"`
			UpdatedAt string `json:"updatedAt"`
		}{
			Status:    "Unknown",
			StatusOK:  ok,
			Logs:      logsString,
			UpdatedAt: timestamp,
		})
	})
}
