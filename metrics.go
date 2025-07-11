package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func registerMetricsEndpoint(mux *http.ServeMux, name, logfile, lokiURL, subscriptionID, nodeType *string) {
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		if mode == "multi-container" {
			containers, err := ListDockerContainers()
			if err != nil {
				fmt.Fprintf(w, "# Error listing containers: %v\n", err)
				return
			}
			fmt.Fprintf(w, "# HELP node_status Node running status (1=running, 0=not running)\n")
			fmt.Fprintf(w, "# TYPE node_status gauge\n")
			fmt.Fprintf(w, "# HELP node_last_updated_at Last updated timestamp (UTC, seconds since epoch)\n")
			fmt.Fprintf(w, "# TYPE node_last_updated_at gauge\n")
			for _, cname := range containers {
				status, err := GetDockerContainerStatus(cname)
				var nodeStatus int
				if err == nil && status == "running" {
					nodeStatus = 1
				} else {
					nodeStatus = 0
				}
				nodeLastUpdatedAt := time.Now().UTC().Unix()
				fmt.Fprintf(w, "node_status{name=\"%s\",subscription_id=\"%s\",node_type=\"%s\"} %d\n", cname, cname, *nodeType, nodeStatus)
				fmt.Fprintf(w, "node_last_updated_at{name=\"%s\",subscription_id=\"%s\",node_type=\"%s\"} %d\n", cname, cname, *nodeType, nodeLastUpdatedAt)
			}
			return
		}

		_, ok, _ := getNodeStatus(*logfile)
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

// Collects metrics in Prometheus exposition format as a string
func collectMetrics(name, logfile, subscriptionID, nodeType *string) (string, error) {
	var metrics string
	if mode == "multi-container" {
		containers, err := ListDockerContainers()
		if err != nil {
			return "", err
		}
		metrics += "# HELP node_status Node running status (1=running, 0=not running)\n"
		metrics += "# TYPE node_status gauge\n"
		metrics += "# HELP node_last_updated_at Last updated timestamp (UTC, seconds since epoch)\n"
		metrics += "# TYPE node_last_updated_at gauge\n"
		for _, cname := range containers {
			status, err := GetDockerContainerStatus(cname)
			var nodeStatus int
			if err == nil && status == "running" {
				nodeStatus = 1
			} else {
				nodeStatus = 0
			}
			nodeLastUpdatedAt := time.Now().UTC().Unix()
			metrics += fmt.Sprintf("node_status{name=\"%s\",subscription_id=\"%s\",node_type=\"%s\"} %d\n", cname, cname, *nodeType, nodeStatus)
			metrics += fmt.Sprintf("node_last_updated_at{name=\"%s\",subscription_id=\"%s\",node_type=\"%s\"} %d\n", cname, cname, *nodeType, nodeLastUpdatedAt)
		}
		return metrics, nil
	}

	_, ok, _ := getNodeStatus(*logfile)
	var nodeStatus int
	if ok {
		nodeStatus = 1
	} else {
		nodeStatus = 0
	}
	nodeLastUpdatedAt := time.Now().UTC().Unix()
	metrics += "# HELP node_status Node running status (1=running, 0=not running)\n"
	metrics += "# TYPE node_status gauge\n"
	metrics += fmt.Sprintf("node_status{name=\"%s\",subscription_id=\"%s\",node_type=\"%s\"} %d\n", *name, *subscriptionID, *nodeType, nodeStatus)
	metrics += "# HELP node_last_updated_at Last updated timestamp (UTC, seconds since epoch)\n"
	metrics += "# TYPE node_last_updated_at gauge\n"
	metrics += fmt.Sprintf("node_last_updated_at{name=\"%s\",subscription_id=\"%s\",node_type=\"%s\"} %d\n", *name, *subscriptionID, *nodeType, nodeLastUpdatedAt)
	return metrics, nil
}

// Pushes metrics to the Prometheus Pushgateway
func pushMetricsToGateway(pushgatewayURL, job, token string, metrics string) error {
	url := fmt.Sprintf("%s/metrics/job/%s", pushgatewayURL, job)
	client := &http.Client{}
	request, err := http.NewRequest("POST", url, strings.NewReader(metrics))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain")
	if token != "" {
		request.Header.Set("Monitoring-Access-Token", token)
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("Pushgateway returned status: %s", resp.Status)
}
