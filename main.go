package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

var (
	name                  string
	port                  string
	logfile               string
	subscriptionID        string
	nodeType              string
	lokiPushInterval      int
	mode                  string // 'multi-container' or 'logfile'
	pushgatewayURL        string
	metricsPushInterval   int
	monitoringAccessToken string
)

func main() {
	var lokiURL string
	flag.StringVar(&name, "name", "", "Name of the service being monitored")
	flag.StringVar(&port, "port", "6969", "Port")
	flag.StringVar(&logfile, "logfile", "", "Log file path to monitor")
	flag.StringVar(&subscriptionID, "subscription-id", "", "Subscription ID")
	flag.StringVar(&nodeType, "node-type", "", "Node type")
	flag.StringVar(&lokiURL, "loki-url", "http://20.52.1.37/loki/api/v1/push", "Loki push URL")
	flag.IntVar(&lokiPushInterval, "loki-push-interval", 10, "Interval (in seconds) to push logs to Loki")
	flag.StringVar(&mode, "mode", "logfile", "Monitoring mode: 'multi-container' or 'logfile'")
	flag.StringVar(&pushgatewayURL, "pushgateway-url", "http://20.52.1.37/pushgateway", "Prometheus Pushgateway URL (optional)")
	flag.IntVar(&metricsPushInterval, "metrics-push-interval", 10, "Interval (in seconds) to push metrics to Pushgateway (optional)")
	flag.StringVar(&monitoringAccessToken, "monitoring-access-token", "8cf42e37b1bd4483b816be0d3f832ccb", "Token for Monitoring-Access-Token header (optional)")
	flag.Parse()

	if pushgatewayURL != "" {
		go func() {
			ticker := time.NewTicker(time.Duration(metricsPushInterval) * time.Second)
			defer ticker.Stop()
			for {
				<-ticker.C
				metrics, err := collectMetrics(&name, &logfile, &subscriptionID, &nodeType)
				if err != nil {
					log.Printf("Error collecting metrics for Pushgateway: %v", err)
					continue
				}
				job := name
				if job == "" {
					job = "nodewatch" // fallback job name
				}
				err = pushMetricsToGateway(pushgatewayURL, job, monitoringAccessToken, metrics)
				if err != nil {
					log.Printf("Error pushing metrics to Pushgateway: %v", err)
				} else {
					//log.Printf("Successfully pushed metrics to Pushgateway at %s (job=%s)", pushgatewayURL, job)
				}
			}
		}()
	}

	switch mode {
	case "multi-container":
		if nodeType == "" {
			log.Fatal("-node-type is required in multi-container mode.")
		}
	case "logfile":
		if name == "" || logfile == "" || subscriptionID == "" || nodeType == "" {
			log.Fatal("-name, -logfile, -subscription-id, and -node-type parameters are required in logfile mode.")
		}
	default:
		log.Fatalf("Unknown mode: %s. Supported modes: 'multi-container', 'logfile'", mode)
	}

	mux := http.NewServeMux()

	registerMetricsEndpoint(mux, &name, &logfile, &lokiURL, &subscriptionID, &nodeType)
	registerEndpoints(mux)

	if mode == "multi-container" {
		containerLogTimestamp := make(map[string]time.Time)
		var containerList []string
		var containerIdx int
		var tickCount int // For heartbeat
		go func() {
			ticker := time.NewTicker(time.Duration(lokiPushInterval) * time.Second)
			defer ticker.Stop()
			for {
				<-ticker.C
				tickCount++
				containers, err := ListDockerContainers()
				if err != nil {
					log.Printf("[ERROR] Error listing containers: %v", err)
					continue
				}
				if len(containers) == 0 {
					log.Printf("[DEBUG] No containers found")
					continue
				}
				// Update container list and index if containers changed
				if len(containerList) != len(containers) || !equalStringSlices(containerList, containers) {
					log.Printf("[DEBUG] Container list changed. Old: %v, New: %v", containerList, containers)
					containerList = containers
					if containerIdx >= len(containerList) {
						containerIdx = 0
					}
				}
				cname := containerList[containerIdx]
				lastTs := containerLogTimestamp[cname]
				logs, newTs, err := ReadNewDockerLogLinesSince(cname, lastTs)
				if err != nil {
					log.Printf("[ERROR] Error reading logs for container %s: %v", cname, err)
					// Do not update timestamp on error
				} else if len(logs) == 0 {
					log.Printf("[DEBUG] No new logs for container %s since %v", cname, lastTs)
					// Do not update timestamp if no logs
				} else {
					stream := createLokiStream(cname, cname, nodeType, logs)
					if stream != nil {
						pushToLoki(lokiURL, cname, stream)
						containerLogTimestamp[cname] = newTs
					} else {
						log.Printf("[WARN] createLokiStream returned nil for container %s", cname)
					}
				}
				// Advance to next container
				containerIdx = (containerIdx + 1) % len(containerList)
			}
		}()
	} else {
		go func() {
			ticker := time.NewTicker(time.Duration(lokiPushInterval) * time.Second)
			defer ticker.Stop()
			for {
				<-ticker.C
				logs, err := readNewLogLines(logfile)
				if err != nil || len(logs) == 0 {
					continue
				}
				_, ok, _ := getNodeStatus(logfile)
				if ok {
					stream := createLokiStream(name, subscriptionID, nodeType, logs)
					if stream != nil {
						go pushToLoki(lokiURL, name, stream)
					}
				}
			}
		}()
	}
	log.Printf("Health endpoint available at http://localhost:%s/health\n", port)
	log.Printf("Metrics endpoint available at http://localhost:%s/metrics\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// Helper to compare two string slices for equality
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
