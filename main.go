package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

var (
	name             string
	port             string
	logfile          string
	subscriptionID   string
	nodeType         string
	lokiPushInterval int
)

func main() {
	var lokiURL string
	flag.StringVar(&name, "name", "", "Name of the service being monitored")
	flag.StringVar(&port, "port", "6969", "Port")
	flag.StringVar(&logfile, "logfile", "", "Log file path to monitor")
	flag.StringVar(&subscriptionID, "subscription-id", "", "Subscription ID")
	flag.StringVar(&nodeType, "node-type", "", "Node type")
	flag.StringVar(&lokiURL, "loki-url", "", "Loki push URL")
	flag.IntVar(&lokiPushInterval, "loki-push-interval", 10, "Interval (in seconds) to push logs to Loki")
	flag.Parse()

	if name == "" || logfile == "" || subscriptionID == "" || nodeType == "" {
		log.Fatal("-name, -logfile, -subscription-id, and -node-type parameters are required.")
	}

	mux := http.NewServeMux()

	registerMetricsEndpoint(mux, &name, &logfile, &lokiURL, &subscriptionID, &nodeType)
	registerEndpoints(mux)

	go func() {
		ticker := time.NewTicker(time.Duration(lokiPushInterval) * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			logs, err := readNewLogLines(logfile)
			if err != nil || len(logs) == 0 {
				continue
			}
			// Remove unused status variable
			_, ok, _ := getNodeStatus(logfile)
			if ok {
				stream := createLokiStream(name, subscriptionID, nodeType, logs)
				if stream != nil {
					go pushToLoki(lokiURL, name, stream)
				}
			}
		}
	}()
	log.Printf("Health endpoint available at http://localhost:%s/health\n", port)
	log.Printf("Metrics endpoint available at http://localhost:%s/metrics\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
