package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func pushToLoki(lokiURL, name string, stream *LokiStream) {
	if stream == nil || len(stream.Values) == 0 {
		return
	}

	payload := LokiPayload{
		Streams: []LokiStream{*stream},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling Loki payload: %v", err)
		return
	}

	// Send to the proxy (NGINX)
	req, err := http.NewRequest("POST", lokiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Error creating Loki request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Set the Monitoring-Access-Token header
	req.Header.Set("Monitoring-Access-Token", monitoringAccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending logs to Loki: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Unexpected status code from Loki: %d, response: %s", resp.StatusCode, string(body))
	}
}
