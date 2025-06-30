package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// activityTimeout defines the duration of inactivity after which a node is considered 'dead'.
var activityTimeout = 30 * time.Second

// getFileLogs reads the tail of a log file, determines node status based on
// log file modification time, and sends logs to Loki.
func getFileLogs(lokiURL, name, logfile, subscriptionID, nodeType string) (string, *LokiStream, bool, []string) {
	info, err := os.Stat(logfile)
	var logs []string
	if err != nil {
		if os.IsNotExist(err) {
			return "Not Found", nil, false, logs
		}
		return "error stating file", nil, false, logs
	}

	cmd := exec.Command("tail", "-n", "100", logfile)
	out, err := cmd.CombinedOutput()
	var logContent string
	if err != nil {
		logContent = "error reading logs: " + err.Error()
	} else {
		logContent = string(out)
	}

	logLines := strings.Split(logContent, "\n")
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			logs = append(logs, line)
		}
	}

	var status string
	var ok bool
	if time.Since(info.ModTime()) > activityTimeout {
		status = "dead"
		ok = false
	} else {
		status = "running"
		ok = true
	}

	stream := createLokiStream(name, subscriptionID, nodeType, logs)
	if stream != nil && ok {
		go pushToLoki(lokiURL, name, stream)
	}

	return status, stream, ok, logs
}

func jsonTimeFormat() string {
	return time.Now().UTC().Format("02-01-2006 15:04:05")
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
