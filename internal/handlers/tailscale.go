package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"casadrop/internal/models"
)

var (
	tailscaleConfig     *models.TailscaleConfig
	tailscaleConfigLock sync.RWMutex
	tailscaleConfigFile string
)

func initTailscaleConfig(dataDir string) {
	tailscaleConfigFile = filepath.Join(dataDir, "tailscale_config.json")
	tailscaleConfig = &models.TailscaleConfig{
		Status: "disconnected",
	}

	// Load existing config
	if data, err := os.ReadFile(tailscaleConfigFile); err == nil {
		json.Unmarshal(data, tailscaleConfig)
	}

	// Check if Tailscale is available and get status
	go updateTailscaleStatus()
}

func saveTailscaleConfig() error {
	tailscaleConfigLock.RLock()
	data, err := json.MarshalIndent(tailscaleConfig, "", "  ")
	tailscaleConfigLock.RUnlock()

	if err != nil {
		return err
	}
	return os.WriteFile(tailscaleConfigFile, data, 0600)
}

func updateTailscaleStatus() {
	tailscaleConfigLock.Lock()
	defer tailscaleConfigLock.Unlock()

	// Check if tailscaled is running. Bound the exec so a wedged tailscaled
	// can't block indefinitely while this holds tailscaleConfigLock.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tailscale", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		tailscaleConfig.Status = "disconnected"
		return
	}

	var status struct {
		BackendState string `json:"BackendState"`
		Self         struct {
			DNSName string `json:"DNSName"`
		} `json:"Self"`
	}

	if err := json.Unmarshal(output, &status); err != nil {
		tailscaleConfig.Status = "disconnected"
		return
	}

	switch status.BackendState {
	case "Running":
		tailscaleConfig.Status = "connected"
		if status.Self.DNSName != "" {
			// Remove trailing dot from DNS name
			dnsName := strings.TrimSuffix(status.Self.DNSName, ".")
			tailscaleConfig.FunnelURL = "https://" + dnsName
		}
	case "Starting", "NoState":
		tailscaleConfig.Status = "connecting"
	default:
		tailscaleConfig.Status = "disconnected"
	}

	// Check funnel status (bounded for the same reason as above)
	fctx, fcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer fcancel()
	funnelCmd := exec.CommandContext(fctx, "tailscale", "funnel", "status")
	if funnelOutput, err := funnelCmd.Output(); err == nil {
		if strings.Contains(string(funnelOutput), "https://") {
			// Extract funnel URL
			lines := strings.Split(string(funnelOutput), "\n")
			for _, line := range lines {
				if strings.Contains(line, "https://") {
					parts := strings.Fields(line)
					for _, part := range parts {
						if strings.HasPrefix(part, "https://") {
							tailscaleConfig.FunnelURL = part
							break
						}
					}
				}
			}
		}
	}
}

// GetTailscaleConfig returns current Tailscale configuration
func (h *Handler) GetTailscaleConfig(w http.ResponseWriter, r *http.Request) {
	updateTailscaleStatus()

	tailscaleConfigLock.RLock()
	defer tailscaleConfigLock.RUnlock()

	// Don't send auth key to frontend
	response := models.TailscaleConfig{
		Enabled:   tailscaleConfig.Enabled,
		Hostname:  tailscaleConfig.Hostname,
		FunnelURL: tailscaleConfig.FunnelURL,
		Status:    tailscaleConfig.Status,
		Error:     tailscaleConfig.Error,
	}

	// Mask auth key if present (guard against short keys to avoid a slice panic)
	if len(tailscaleConfig.AuthKey) >= 4 {
		response.AuthKey = "****" + tailscaleConfig.AuthKey[len(tailscaleConfig.AuthKey)-4:]
	} else if tailscaleConfig.AuthKey != "" {
		response.AuthKey = "****"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SaveTailscaleConfig saves Tailscale configuration and optionally starts Tailscale
func (h *Handler) SaveTailscaleConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AuthKey  string `json:"auth_key"`
		Hostname string `json:"hostname"`
		Enabled  bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tailscaleConfigLock.Lock()

	// Only update auth key if provided (not masked)
	if req.AuthKey != "" && !strings.HasPrefix(req.AuthKey, "****") {
		tailscaleConfig.AuthKey = req.AuthKey
	}
	if req.Hostname != "" {
		tailscaleConfig.Hostname = req.Hostname
	}
	tailscaleConfig.Enabled = req.Enabled

	tailscaleConfigLock.Unlock()

	if err := saveTailscaleConfig(); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	// If enabled and we have auth key, start Tailscale
	if req.Enabled && tailscaleConfig.AuthKey != "" {
		go startTailscale()
	} else if !req.Enabled {
		go stopTailscale()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// StartTailscaleHandler manually starts Tailscale
func (h *Handler) StartTailscaleHandler(w http.ResponseWriter, r *http.Request) {
	tailscaleConfigLock.RLock()
	authKey := tailscaleConfig.AuthKey
	tailscaleConfigLock.RUnlock()

	if authKey == "" {
		http.Error(w, "No auth key configured", http.StatusBadRequest)
		return
	}

	go startTailscale()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "starting"})
}

// StopTailscaleHandler stops Tailscale
func (h *Handler) StopTailscaleHandler(w http.ResponseWriter, r *http.Request) {
	go stopTailscale()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopping"})
}

func startTailscale() {
	tailscaleConfigLock.Lock()
	tailscaleConfig.Status = "connecting"
	tailscaleConfig.Error = ""
	authKey := tailscaleConfig.AuthKey
	hostname := tailscaleConfig.Hostname
	tailscaleConfigLock.Unlock()

	log.Println("Starting Tailscale...")

	// Check if tailscaled is running, if not start it
	if !isTailscaledRunning() {
		log.Println("Starting tailscaled daemon...")
		cmd := exec.Command("tailscaled",
			"--state=/data/tailscale/tailscaled.state",
			"--socket=/var/run/tailscale/tailscaled.sock",
			"--tun=userspace-networking",
			"--socks5-server=localhost:1055",
			"--outbound-http-proxy-listen=localhost:1056")
		cmd.Env = append(os.Environ(), "HOME=/tmp")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			setTailscaleError(fmt.Sprintf("Failed to start tailscaled: %v", err))
			return
		}
		// Wait for daemon to be ready
		time.Sleep(3 * time.Second)
	}

	// Build tailscale up command
	args := []string{"up", "--authkey=" + authKey}
	if hostname != "" {
		args = append(args, "--hostname="+hostname)
	}
	args = append(args, "--accept-routes=false")

	cmd := exec.Command("tailscale", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		setTailscaleError(fmt.Sprintf("Failed to connect: %v - %s", err, string(output)))
		return
	}

	log.Println("Tailscale connected, enabling Funnel...")

	// Wait a moment for connection to establish
	time.Sleep(2 * time.Second)

	// Enable Funnel on port 8080
	funnelCmd := exec.Command("tailscale", "funnel", "--bg", "8080")
	funnelOutput, err := funnelCmd.CombinedOutput()
	if err != nil {
		log.Printf("Funnel warning: %v - %s", err, string(funnelOutput))
		// Don't treat as error, funnel might need to be enabled in admin console first
	}

	updateTailscaleStatus()
	saveTailscaleConfig()

	log.Printf("Tailscale started successfully. URL: %s", tailscaleConfig.FunnelURL)
}

func stopTailscale() {
	log.Println("Stopping Tailscale...")

	// Disable funnel first
	exec.Command("tailscale", "funnel", "off").Run()

	// Logout
	exec.Command("tailscale", "logout").Run()

	tailscaleConfigLock.Lock()
	tailscaleConfig.Status = "disconnected"
	tailscaleConfig.FunnelURL = ""
	tailscaleConfig.Error = ""
	tailscaleConfigLock.Unlock()

	saveTailscaleConfig()
}

func isTailscaledRunning() bool {
	cmd := exec.Command("tailscale", "status")
	err := cmd.Run()
	return err == nil
}

func setTailscaleError(errMsg string) {
	log.Printf("Tailscale error: %s", errMsg)
	tailscaleConfigLock.Lock()
	tailscaleConfig.Status = "error"
	tailscaleConfig.Error = errMsg
	tailscaleConfigLock.Unlock()
	saveTailscaleConfig()
}

// InitTailscaleOnStartup initializes Tailscale if configured
func InitTailscaleOnStartup(dataDir string) {
	initTailscaleConfig(dataDir)

	tailscaleConfigLock.RLock()
	enabled := tailscaleConfig.Enabled
	hasAuthKey := tailscaleConfig.AuthKey != ""
	tailscaleConfigLock.RUnlock()

	if enabled && hasAuthKey {
		log.Println("Tailscale is configured, starting...")
		go startTailscale()
	} else {
		log.Println("Tailscale not configured or disabled")
	}
}
