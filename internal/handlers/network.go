package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"casadrop/internal/utils"
)

// NetworkItemInfo represents a single network's status
type NetworkItemInfo struct {
	Enabled  bool   `json:"enabled"`
	URL      string `json:"url"`
	Detected string `json:"detected"`
}

// NetworkInfo represents available network access points
type NetworkInfo struct {
	LocalIP        string `json:"localIp"`
	TunnelURL      string `json:"tunnelUrl"`
	TailscaleURL   string `json:"tailscaleUrl"`
	EasyTierIP     string `json:"easytierIp"`
	CustomURL      string `json:"customUrl"`
	Port           string `json:"port"`
	PrimaryNetwork string `json:"primaryNetwork"` // Active network for share links
	PrimaryURL     string `json:"primaryUrl"`     // The URL to use for share links
	MaxFileSizeGB  int    `json:"maxFileSizeGB"`  // Maximum file size in GB

	// Per-network enabled status and detected values
	Networks map[string]NetworkItemInfo `json:"networks"`
}

// TunnelURL handles GET and POST for tunnel configuration
func (h *Handler) TunnelURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.saveTunnelURL(w, r)
		return
	}
	h.getTunnelURL(w, r)
}

// getTunnelURL returns the Cloudflare tunnel URL if available
// Priority: 1. User config (if enabled) 2. Auto-detected quick tunnel 3. Environment variable
func (h *Handler) getTunnelURL(w http.ResponseWriter, r *http.Request) {
	tunnelURL := ""
	isExternal := false

	// 1. Check user-configured external tunnel (highest priority)
	config, _ := h.loadTunnelConfig()
	if config.Enabled && config.URL != "" {
		tunnelURL = config.URL
		isExternal = true
	}

	// 2. Try to read auto-detected URL from file (set by tunnel container)
	if tunnelURL == "" {
		tunnelFile := filepath.Join(h.getDataDir(), "tunnel_url.txt")
		if data, err := os.ReadFile(tunnelFile); err == nil {
			url := strings.TrimSpace(string(data))
			// "token" means a token-based tunnel is active but URL is configured externally
			if url != "" && url != "token" {
				tunnelURL = url
			}
		}
	}

	// 3. Fall back to environment variable
	if tunnelURL == "" {
		tunnelURL = os.Getenv("TUNNEL_URL")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":        tunnelURL,
		"isExternal": isExternal,
		"config":     config,
	})
}

// saveTunnelURL saves the user's external tunnel configuration
func (h *Handler) saveTunnelURL(w http.ResponseWriter, r *http.Request) {
	var config TunnelConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate Cloudflare URL (requires HTTPS)
	if config.CloudflareURL != "" {
		if err := utils.ValidateURL(config.CloudflareURL, true); err != nil {
			http.Error(w, fmt.Sprintf("Cloudflare URL: %s", err.Error()), http.StatusBadRequest)
			return
		}
	}
	if config.Enabled && config.URL != "" {
		if err := utils.ValidateURL(config.URL, true); err != nil {
			http.Error(w, fmt.Sprintf("Cloudflare URL: %s", err.Error()), http.StatusBadRequest)
			return
		}
	}

	// Validate Tailscale URL (requires HTTPS)
	if config.TailscaleURL != "" {
		if err := utils.ValidateURL(config.TailscaleURL, true); err != nil {
			http.Error(w, fmt.Sprintf("Tailscale URL: %s", err.Error()), http.StatusBadRequest)
			return
		}
	}

	// Validate Custom URL (allows HTTP and HTTPS for flexibility)
	if config.CustomURL != "" {
		if err := utils.ValidateURL(config.CustomURL, false); err != nil {
			http.Error(w, fmt.Sprintf("Custom URL: %s", err.Error()), http.StatusBadRequest)
			return
		}
	}

	// Validate EasyTier IP (should be a valid IP address or empty)
	if config.EasyTierIP != "" {
		if net.ParseIP(config.EasyTierIP) == nil {
			http.Error(w, "EasyTier IP: invalid IP address", http.StatusBadRequest)
			return
		}
	}

	// Validate maxFileSizeGB (must be between 1 and 100 GB)
	if config.MaxFileSizeGB < 0 {
		config.MaxFileSizeGB = 0 // Use default
	}
	if config.MaxFileSizeGB > 100 {
		config.MaxFileSizeGB = 100 // Cap at 100 GB
	}

	if err := h.saveTunnelConfig(&config); err != nil {
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// detectTailscaleIP attempts to detect the Tailscale IPv4 address
func detectTailscaleIP() string {
	out, err := exec.Command("tailscale", "ip", "-4").Output()
	if err == nil {
		ip := strings.TrimSpace(string(out))
		if ip != "" {
			return ip
		}
	}
	// Fallback: scan interfaces
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "tailscale") {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	return ""
}

// detectTailscaleFunnelURL attempts to detect the Tailscale Funnel HTTPS URL
func detectTailscaleFunnelURL() string {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return ""
	}
	var status struct {
		Self struct {
			DNSName string `json:"DNSName"`
			Online  bool   `json:"Online"`
		} `json:"Self"`
	}
	if json.Unmarshal(out, &status) == nil && status.Self.Online && status.Self.DNSName != "" {
		dnsName := strings.TrimSuffix(status.Self.DNSName, ".")
		return "https://" + dnsName
	}
	return ""
}

// detectCloudflareTunnel attempts to detect a Cloudflare Tunnel URL
func detectCloudflareTunnel() string {
	// Method 1: Check for tunnel_url.txt in data dir
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	if data, err := os.ReadFile(filepath.Join(dataDir, "tunnel_url.txt")); err == nil {
		url := strings.TrimSpace(string(data))
		if url != "" && url != "token" {
			return url
		}
	}
	// Method 2: Try cloudflared tunnel info
	out, err := exec.Command("cloudflared", "tunnel", "info", "--output", "json").Output()
	if err == nil {
		var info map[string]interface{}
		if json.Unmarshal(out, &info) == nil {
			// Look for connectors or hostname
		}
	}
	// Method 3: TUNNEL_URL env var
	if url := os.Getenv("TUNNEL_URL"); url != "" {
		return url
	}
	return ""
}

// detectEasyTierIP attempts to detect the EasyTier virtual IP address
func detectEasyTierIP() string {
	// Method 1: easytier-cli peer list
	out, err := exec.Command("easytier-cli", "peer", "list").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "local") || strings.Contains(line, "self") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if net.ParseIP(part) != nil {
						return part
					}
				}
			}
		}
	}
	// Method 2: easytier-cli connector list
	out, err = exec.Command("easytier-cli", "connector", "list").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			parts := strings.Fields(strings.TrimSpace(line))
			for _, part := range parts {
				ip := net.ParseIP(part)
				if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
					return part
				}
			}
		}
	}
	// Method 3: Scan tun interfaces
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "tun") || strings.HasPrefix(iface.Name, "easytier") {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
					return ipnet.IP.String()
				}
			}
		}
	}
	// Method 4: env var
	return os.Getenv("EASYTIER_IP")
}

// detectLocalIP tries to find the local network IP
func detectLocalIP() string {
	if ip := os.Getenv("LOCAL_IP"); ip != "" {
		return ip
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if strings.HasPrefix(iface.Name, "tailscale") || strings.HasPrefix(iface.Name, "tun") || strings.HasPrefix(iface.Name, "easytier") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if strings.HasPrefix(ip.String(), "172.") {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

// detectTailscaleURL detects the Tailscale Funnel URL from multiple sources
func (h *Handler) detectTailscaleURL() string {
	tailscaleFile := filepath.Join(h.getDataDir(), "tailscale_url.txt")
	if data, err := os.ReadFile(tailscaleFile); err == nil {
		if url := strings.TrimSpace(string(data)); url != "" {
			return url
		}
	}
	if url := detectTailscaleFunnelURL(); url != "" {
		return url
	}
	return os.Getenv("TAILSCALE_URL")
}

// GetNetworkInfo returns all available network interfaces for accessing the service
func (h *Handler) GetNetworkInfo(w http.ResponseWriter, r *http.Request) {
	info := NetworkInfo{
		Port: os.Getenv("EXTERNAL_PORT"),
	}
	if info.Port == "" {
		info.Port = os.Getenv("PORT")
	}
	if info.Port == "" {
		info.Port = "8080"
	}

	config, _ := h.loadTunnelConfig()

	// Always detect for hints, regardless of enabled state
	detectedLocal := detectLocalIP()
	detectedCloudflare := detectCloudflareTunnel()
	detectedTailscale := h.detectTailscaleURL()
	detectedEasyTier := detectEasyTierIP()
	detectedCustom := os.Getenv("CUSTOM_URL")

	// Build per-network info: active URL uses config override or detected value (only if enabled)
	networks := map[string]NetworkItemInfo{
		"local": {
			Enabled:  config.IsLocalEnabled(),
			URL:      firstNonEmpty(config.LocalIP, detectedLocal),
			Detected: detectedLocal,
		},
		"cloudflare": {
			Enabled:  config.IsCloudflareEnabled(),
			URL:      firstNonEmpty(config.CloudflareURL, config.URL, detectedCloudflare),
			Detected: detectedCloudflare,
		},
		"tailscale": {
			Enabled:  config.IsTailscaleEnabled(),
			URL:      firstNonEmpty(config.TailscaleURL, detectedTailscale),
			Detected: detectedTailscale,
		},
		"easytier": {
			Enabled:  config.IsEasyTierEnabled(),
			URL:      firstNonEmpty(config.EasyTierIP, detectedEasyTier),
			Detected: detectedEasyTier,
		},
		"custom": {
			Enabled:  config.IsCustomEnabled(),
			URL:      firstNonEmpty(config.CustomURL, detectedCustom),
			Detected: detectedCustom,
		},
	}
	info.Networks = networks

	// Populate legacy flat fields (only for enabled networks)
	if networks["local"].Enabled {
		info.LocalIP = networks["local"].URL
	}
	if networks["cloudflare"].Enabled {
		info.TunnelURL = networks["cloudflare"].URL
	}
	if networks["tailscale"].Enabled {
		info.TailscaleURL = networks["tailscale"].URL
	}
	if networks["easytier"].Enabled {
		info.EasyTierIP = networks["easytier"].URL
	}
	if networks["custom"].Enabled {
		info.CustomURL = networks["custom"].URL
	}

	// Determine primary network
	info.PrimaryNetwork = config.PrimaryNetwork
	if info.PrimaryNetwork == "" {
		info.PrimaryNetwork = "local"
	}

	// Build primary URL based on selected network
	switch info.PrimaryNetwork {
	case "tailscale":
		info.PrimaryURL = info.TailscaleURL
	case "easytier":
		if info.EasyTierIP != "" {
			info.PrimaryURL = fmt.Sprintf("http://%s:%s", info.EasyTierIP, info.Port)
		}
	case "custom":
		info.PrimaryURL = info.CustomURL
	case "local":
		if info.LocalIP != "" {
			info.PrimaryURL = fmt.Sprintf("http://%s:%s", info.LocalIP, info.Port)
		}
	case "cloudflare":
		info.PrimaryURL = info.TunnelURL
	default:
		info.PrimaryURL = info.TunnelURL
	}

	if config.MaxFileSizeGB > 0 {
		info.MaxFileSizeGB = config.MaxFileSizeGB
	} else {
		info.MaxFileSizeGB = 10
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// firstNonEmpty returns the first non-empty string
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
