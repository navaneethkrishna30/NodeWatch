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
	name           string
	port           string
	logfile        string
	subscriptionID string
	nodeType       string
)

func main() {
	var lokiURL string
	flag.StringVar(&name, "name", "", "Name of the service being monitored")
	flag.StringVar(&port, "port", "6969", "Port")
	flag.StringVar(&logfile, "logfile", "", "Log file path to monitor")
	flag.StringVar(&subscriptionID, "subscription-id", "", "Subscription ID")
	flag.StringVar(&nodeType, "node-type", "", "Node type")
	flag.StringVar(&lokiURL, "loki-url", "http://localhost:3100/loki/api/v1/push", "Loki push URL")
	flag.Parse()

	if name == "" || logfile == "" || subscriptionID == "" || nodeType == "" {
		log.Fatal("-name, -logfile, -subscription-id, and -node-type parameters are required.")
	}

	staticContent, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	// Register Prometheus metrics endpoint
	registerMetricsEndpoint(mux, &name, &logfile, &lokiURL, &subscriptionID, &nodeType)

	// Register all other endpoints
	registerEndpoints(mux, staticContent, indexHTML, &name, &logfile, &lokiURL, &subscriptionID, &nodeType)

	log.Printf("Listening on http://localhost:%s/\n", port)
	log.Printf("Status endpoint available at http://localhost:%s/status\n", port)
	log.Printf("Health endpoint available at http://localhost:%s/health\n", port)
	log.Printf("Metrics endpoint available at http://localhost:%s/metrics\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
