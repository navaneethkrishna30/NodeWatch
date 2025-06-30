package main

import "time"

// MetricsResponse represents the JSON API response structure
type MetricsResponse struct {
	Status        bool      `json:"status"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
	Logs          []string  `json:"logs"`
}

type LokiPayload struct {
	Streams []LokiStream `json:"streams"`
}

type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}
