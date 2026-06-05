package models

// TailscaleConfig holds Tailscale Funnel configuration
type TailscaleConfig struct {
	Enabled   bool   `json:"enabled"`
	AuthKey   string `json:"auth_key,omitempty"`
	Hostname  string `json:"hostname"`
	FunnelURL string `json:"funnel_url,omitempty"`
	Status    string `json:"status"` // "disconnected", "connecting", "connected", "error"
	Error     string `json:"error,omitempty"`
}
