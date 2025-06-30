package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed static/index.html
var indexHTML []byte

var (
	mode    string
	name    string
	port    string
	logfile string
)

// MetricsResponse represents the JSON API response structure
type MetricsResponse struct {
	Status        bool      `json:"status"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
	Logs          []string  `json:"logs"`
}

func getDockerStatus(name string) (string, bool) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name)
	out, err := cmd.Output()
	if err != nil {
		return "Not Found", false
	}
	status := strings.TrimSpace(string(out))
	return status, status == "true"
}

func getDockerLogs(name, logfile string) []string {
	var logs string
	if logfile != "" {
		cmd := exec.Command("docker", "exec", name, "tail", "-n", "100", logfile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logs = err.Error()
		} else {
			logs = string(out)
		}
	} else {
		cmd := exec.Command("docker", "logs", "--tail", "100", name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logs = err.Error()
		} else {
			logs = string(out)
		}
	}

	// Split logs into array by lines and filter empty lines
	logLines := strings.Split(logs, "\n")
	var filteredLogs []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			filteredLogs = append(filteredLogs, line)
		}
	}
	return filteredLogs
}

func getSystemdStatus(name string) (string, bool) {
	cmd := exec.Command("systemctl", "is-active", name)
	out, err := cmd.Output()
	if err != nil {
		return "Unknown", false
	}
	status := strings.TrimSpace(string(out))
	return status, status == "active"
}

func getSystemdLogs(name string) []string {
	cmd := exec.Command("journalctl", "-u", name, "--no-pager", "-n", "100")
	out, err := cmd.CombinedOutput()
	var logs string
	if err != nil {
		logs = err.Error()
	} else {
		logs = string(out)
	}

	// Split logs into array by lines and filter empty lines
	logLines := strings.Split(logs, "\n")
	var filteredLogs []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			filteredLogs = append(filteredLogs, line)
		}
	}
	return filteredLogs
}

func getStatusAndLogs() (string, []string, bool) {
	var status string
	var logs []string
	var ok bool
	if mode == "docker" {
		status, ok = getDockerStatus(name)
		logs = getDockerLogs(name, logfile)
	} else {
		status, ok = getSystemdStatus(name)
		logs = getSystemdLogs(name)
	}
	return status, logs, ok
}

func main() {
	flag.StringVar(&mode, "mode", "docker", "Mode: docker or systemd")
	flag.StringVar(&name, "name", "", "Container or service name")
	flag.StringVar(&port, "port", "6969", "Port")
	flag.StringVar(&logfile, "logfile", "", "Docker container log file path")
	flag.Parse()

	staticContent, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	// Register Prometheus metrics endpoint
	registerMetricsEndpoint(mux)

	// Static assets
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	// Homepage
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(indexHTML)
	})

	// Status API endpoint (moved from /metrics)
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		_, logs, ok := getStatusAndLogs()
		now := time.Now().UTC()

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
		_, logs, ok := getStatusAndLogs()
		timestamp := strings.ToLower(jsonTimeFormat())

		// Convert logs array back to string for legacy compatibility
		logsString := strings.Join(logs, "\n")

		json.NewEncoder(w).Encode(struct {
			Status    string `json:"status"`
			StatusOK  bool   `json:"statusOK"`
			Logs      string `json:"logs"`
			UpdatedAt string `json:"updatedAt"`
		}{
			Status:    "Unknown", // Placeholder since we don't need the actual status string
			StatusOK:  ok,
			Logs:      logsString,
			UpdatedAt: timestamp,
		})
	})

	log.Printf("Listening on http://localhost:%s/\n", port)
	log.Printf("Status API available at http://localhost:%s/status\n", port)
	log.Printf("Health API available at http://localhost:%s/health\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func jsonTimeFormat() string {
	return time.Now().UTC().Format("02-01-2006 15:04:05")
}
