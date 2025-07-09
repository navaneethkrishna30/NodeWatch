package main

import (
	"fmt"
	"os"
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
