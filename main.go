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

func getDockerStatus(name string) (string, bool) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name)
	out, err := cmd.Output()
	if err != nil {
		return "Not Found", false
	}
	status := strings.TrimSpace(string(out))
	return status, status == "true"
}

func getDockerLogs(name, logfile string) string {
	if logfile != "" {
		cmd := exec.Command("docker", "exec", name, "tail", "-n", "100", logfile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return err.Error()
		}
		return string(out)
	}
	cmd := exec.Command("docker", "logs", "--tail", "100", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err.Error()
	}
	return string(out)
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

func getSystemdLogs(name string) string {
	cmd := exec.Command("journalctl", "-u", name, "--no-pager", "-n", "100")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err.Error()
	}
	return string(out)
}

func getStatusAndLogs() (string, string, bool) {
	var status, logs string
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

	// Static assets
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	// Homepage
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(indexHTML)
	})

	// Async data endpoint
	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status, logs, ok := getStatusAndLogs()
		timestamp := strings.ToLower(jsonTimeFormat())
		json.NewEncoder(w).Encode(struct {
			Status    string `json:"status"`
			StatusOK  bool   `json:"statusOK"`
			Logs      string `json:"logs"`
			UpdatedAt string `json:"updatedAt"`
		}{
			Status:    status,
			StatusOK:  ok,
			Logs:      logs,
			UpdatedAt: timestamp,
		})
	})

	log.Printf("Listening on http://localhost:%s/\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func jsonTimeFormat() string {
	return time.Now().UTC().Format("02-01-2006 15:04:05")
}
