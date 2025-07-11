package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// activityTimeout defines the duration of inactivity after which a node is considered 'dead'.
var activityTimeout = 30 * time.Second

// lastLogOffset tracks the last byte offset read from the log file (in-memory, not persistent).
var lastLogOffset int64 = 0

// readNewLogLines reads only new lines from the log file since the last read and updates the offset.
func readNewLogLines(logfile string) ([]string, error) {
	f, err := os.Open(logfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = f.Seek(lastLogOffset, 0)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 4096)
	var content []byte
	for {
		n, err := f.Read(buf)
		if n > 0 {
			content = append(content, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	logContent := string(content)
	logLines := strings.Split(logContent, "\n")
	var logs []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			logs = append(logs, line)
		}
	}

	// Update the offset to the new end of file
	if newOffset, err := f.Seek(0, 1); err == nil {
		lastLogOffset = newOffset
	}

	return logs, nil
}

// getNodeStatus checks the log file's mod time and returns status and ok.
func getNodeStatus(logfile string) (string, bool, error) {
	info, err := os.Stat(logfile)
	if err != nil {
		if os.IsNotExist(err) {
			return "Not Found", false, nil
		}
		return "error stating file", false, err
	}

	if time.Since(info.ModTime()) > activityTimeout {
		return "dead", false, nil
	}
	return "running", true, nil
}

func createLokiStream(name, subscriptionID, nodeType string, logLines []string) *LokiStream {
	var filteredLogs []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			filteredLogs = append(filteredLogs, line)
		}
	}

	if len(filteredLogs) == 0 {
		return nil
	}

	values := make([][]string, 0, len(filteredLogs))
	for _, line := range filteredLogs {
		values = append(values, []string{fmt.Sprintf("%d", time.Now().UnixNano()), line})
	}

	return &LokiStream{
		Stream: map[string]string{
			"job":             name,
			"subscription_id": subscriptionID,
			"node_type":       nodeType,
		},
		Values: values,
	}
}

// ListDockerContainers returns a slice of running Docker container names.
func ListDockerContainers() ([]string, error) {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			containers = append(containers, line)
		}
	}
	return containers, nil
}

// GetDockerContainerStatus returns the status of a Docker container (e.g., running, exited).
func GetDockerContainerStatus(name string) (string, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", name)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ReadNewDockerLogLinesSince returns new log lines for a container since lastTimestamp.
func ReadNewDockerLogLinesSince(container string, lastTimestamp time.Time) ([]string, time.Time, error) {
	since := lastTimestamp.Format(time.RFC3339)
	cmd := exec.Command("docker", "logs", "--since", since, container)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[ERROR] docker logs --since failed for container %s: %v", container, err)
		return nil, lastTimestamp, err
	}
	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	// Remove empty trailing line if present
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		// Possible log rotation or no new logs
		log.Printf("[DEBUG] No new logs for container %s since %v. Checking for possible log rotation...", container, lastTimestamp)
		// Try reading with an earlier timestamp to recover missed logs
		retryTs := lastTimestamp.Add(-30 * time.Second) // Go back 30s to catch any missed logs
		cmdRetry := exec.Command("docker", "logs", "--since", retryTs.Format(time.RFC3339), container)
		outputRetry, errRetry := cmdRetry.Output()
		if errRetry != nil {
			log.Printf("[ERROR] Retry docker logs --since failed for container %s: %v", container, errRetry)
			return nil, lastTimestamp, nil // Don't treat as fatal, just return no logs
		}
		linesRetry := strings.Split(strings.ReplaceAll(string(outputRetry), "\r\n", "\n"), "\n")
		if len(linesRetry) > 0 && linesRetry[len(linesRetry)-1] == "" {
			linesRetry = linesRetry[:len(linesRetry)-1]
		}
		// Filter logs to only those newer than lastTimestamp
		var recovered []string
		for _, line := range linesRetry {
			// Optionally, parse timestamp from log line if available
			// For now, just include all lines (could be improved)
			recovered = append(recovered, line)
		}
		if len(recovered) > 0 {
			log.Printf("[INFO] Recovered %d logs for container %s after suspected log rotation.", len(recovered), container)
			return recovered, time.Now().UTC(), nil
		}
		log.Printf("[DEBUG] No logs recovered for container %s after log rotation check.", container)
		return nil, lastTimestamp, nil
	}

	return lines, time.Now().UTC(), nil
}
