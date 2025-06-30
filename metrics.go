package main

import (
	"fmt"
	"net/http"
	"time"
)

func registerMetricsEndpoint(mux *http.ServeMux) {
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		_, _, ok := getStatusAndLogs()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		var nodeStatus int
		if ok {
			nodeStatus = 1
		} else {
			nodeStatus = 0
		}

		nodeLastUpdatedAt := time.Now().UTC().Unix()

		fmt.Fprintf(w, "# HELP node_status Node running status (1=running, 0=not running)\n")
		fmt.Fprintf(w, "# TYPE node_status gauge\n")
		fmt.Fprintf(w, "node_status %d\n", nodeStatus)

		fmt.Fprintf(w, "# HELP node_last_updated_at Last updated timestamp (UTC, seconds since epoch)\n")
		fmt.Fprintf(w, "# TYPE node_last_updated_at gauge\n")
		fmt.Fprintf(w, "node_last_updated_at %d\n", nodeLastUpdatedAt)
	})
}
