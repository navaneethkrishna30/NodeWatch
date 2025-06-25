package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

var (
	mode       string
	name       string
	port       string
	autoRefresh int
	logfile    string
)

type PageData struct {
	Mode        string
	Name        string
	Status      string
	Logs        string
	StatusOK    bool
	AutoRefresh int
}

func getDockerStatus(name string) (string, bool) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name)
	out, err := cmd.Output()
	if err != nil {
		return "Not Found", false
	}
	status := strings.TrimSpace(string(out))
	return status, status == "true"
}

func getDockerLogs(name string, logfile string) string {
	if logfile != "" {
		cmd := exec.Command("docker", "exec", name, "tail", "-n", "100", logfile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return err.Error()
		}
		return string(out)
	}
	cmd := exec.Command("docker", "logs", "--tail", "100", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err.Error()
	}
	return string(out)
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

func getSystemdLogs(name string) string {
	cmd := exec.Command("journalctl", "-u", name, "--no-pager", "-n", "100")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err.Error()
	}
	return string(out)
}

func handler(w http.ResponseWriter, r *http.Request) {
	var status, logs string
	var ok bool

	if mode == "docker" {
		status, ok = getDockerStatus(name)
		logs = getDockerLogs(name, logfile)
	} else {
		status, ok = getSystemdStatus(name)
		logs = getSystemdLogs(name)
	}

	data := PageData{
		Mode:        strings.Title(mode),
		Name:        name,
		Status:      status,
		Logs:        logs,
		StatusOK:    ok,
		AutoRefresh: autoRefresh,
	}
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, data)
}

func main() {
	flag.StringVar(&mode, "mode", "docker", "Mode: docker or systemd")
	flag.StringVar(&name, "name", "", "Name of container or service")
	flag.StringVar(&port, "port", "6969", "Port to run the server on")
	flag.IntVar(&autoRefresh, "refresh", 30, "Auto-refresh interval in seconds")
	flag.StringVar(&logfile, "logfile", "", "Path to log file inside container (for docker mode)")
	flag.Parse()

	if name == "" {
		log.Fatal("--name is required")
	}

	http.HandleFunc("/", handler)
	log.Printf("Starting server at http://localhost:%s/\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
