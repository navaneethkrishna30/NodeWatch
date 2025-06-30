package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
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

	// Register all other endpoints
	registerEndpoints(mux, staticContent, indexHTML, &mode, &name, &logfile)

	log.Printf("Listening on http://localhost:%s/\n", port)
	log.Printf("Status endpoint available at http://localhost:%s/status\n", port)
	log.Printf("Health endpoint available at http://localhost:%s/health\n", port)
	log.Printf("Metrics endpoint available at http://localhost:%s/metrics\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
