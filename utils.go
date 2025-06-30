package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func getDockerStatus(name string) (string, bool) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name)
	out, err := cmd.Output()
	if err != nil {
		return "Not Found", false
	}
	status := strings.TrimSpace(string(out))
	return status, status == "true"
}

func getDockerLogs(name, logfile string) *LokiStream {
	var logs string
	if logfile != "" {
		cmd := exec.Command("docker", "exec", name, "tail", "-n", "100", logfile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logs = err.Error()
		} else {
			logs = string(out)
		}
	} else {
		cmd := exec.Command("docker", "logs", "--tail", "100", name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logs = err.Error()
		} else {
			logs = string(out)
		}
	}

	logLines := strings.Split(logs, "\n")
	return createLokiStream(name, logLines)
}

func getSystemdStatus(name string) (string, bool) {
	cmd := exec.Command("systemctl", "is-active", name)
	out, err := cmd.Output()
	if err != nil {
		return "Unknown", false
	}
	status := strings.TrimSpace(string(out))
	return status, status == "active"
}

func getSystemdLogs(name string) *LokiStream {
	cmd := exec.Command("journalctl", "-u", name, "--no-pager", "-n", "100")
	out, err := cmd.CombinedOutput()
	var logs string
	if err != nil {
		logs = err.Error()
	} else {
		logs = string(out)
	}

	logLines := strings.Split(logs, "\n")
	return createLokiStream(name, logLines)
}

func getStatusAndLogs(mode, name, logfile string) (string, *LokiStream, bool) {
	var status string
	var stream *LokiStream
	var ok bool
	if mode == "docker" {
		status, ok = getDockerStatus(name)
		stream = getDockerLogs(name, logfile)
	} else {
		status, ok = getSystemdStatus(name)
		stream = getSystemdLogs(name)
	}
	go pushToLoki(name, stream)
	return status, stream, ok
}

func jsonTimeFormat() string {
	return time.Now().UTC().Format("02-01-2006 15:04:05")
}

func createLokiStream(name string, logLines []string) *LokiStream {
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
			"job": name,
		},
		Values: values,
	}
}
