package main

import "time"

// MetricsResponse represents the JSON API response structure
type MetricsResponse struct {
	Status         bool      `json:"status"`
	LastUpdatedAt  time.Time `json:"last_updated_at"`
	Logs           []string  `json:"logs"`
	SubscriptionID string    `json:"subscription_id"`
	NodeType       string    `json:"node_type"`
}

// LokiPayload represents the data structure for sending logs to Loki.
type LokiPayload struct {
	Streams []LokiStream `json:"streams"`
}

// LokiStream represents a single log stream in Loki.
type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}
