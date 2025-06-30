package main

import (
	"fmt"
	"net/http"
	"time"
)

func registerMetricsEndpoint(mux *http.ServeMux, name, logfile, lokiURL, subscriptionID, nodeType *string) {
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		_, _, ok, _ := getFileLogs(*lokiURL, *name, *logfile, *subscriptionID, *nodeType)
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
		fmt.Fprintf(w, "node_status{name=\"%s\",subscription_id=\"%s\",node_type=\"%s\"} %d\n", *name, *subscriptionID, *nodeType, nodeStatus)

		fmt.Fprintf(w, "# HELP node_last_updated_at Last updated timestamp (UTC, seconds since epoch)\n")
		fmt.Fprintf(w, "# TYPE node_last_updated_at gauge\n")
		fmt.Fprintf(w, "node_last_updated_at{name=\"%s\",subscription_id=\"%s\",node_type=\"%s\"} %d\n", *name, *subscriptionID, *nodeType, nodeLastUpdatedAt)
	})
}
