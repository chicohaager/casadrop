package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Taildrop lets an admin push an existing share's file to one of their own
// devices in the tailnet via `tailscale file cp`. It reuses the host's
// Tailscale identity, so it is strictly admin-only.
//
// Security model:
//   - Admin-only (wired with RequireAdmin in internal/routes).
//   - The file is always resolved from an existing share we already manage —
//     never from a caller-supplied path — so there is no new path-traversal or
//     arbitrary-read surface beyond what the share already exposes.
//   - The target device must match a peer reported by `tailscale status --json`
//     exactly; an unknown target is rejected. exec.Command is used with an
//     argv slice (no shell), so the device string can never be interpreted.

const (
	taildropStatusTimeout = 5 * time.Second
	taildropSendTimeout   = 10 * time.Minute
)

// taildropDevice is a tailnet peer that can receive files.
type taildropDevice struct {
	Name    string `json:"name"`    // short host name (display)
	DNSName string `json:"dnsName"` // stable FQDN, used as the send target
	OS      string `json:"os"`
	Online  bool   `json:"online"`
}

// tsNode mirrors the subset of a `tailscale status --json` node we read.
type tsNode struct {
	DNSName  string `json:"DNSName"`
	HostName string `json:"HostName"`
	OS       string `json:"OS"`
	Online   bool   `json:"Online"`
}

// tsStatus mirrors the subset of `tailscale status --json` we read.
type tsStatus struct {
	BackendState string            `json:"BackendState"`
	Self         tsNode            `json:"Self"`
	Peer         map[string]tsNode `json:"Peer"`
}

// queryTailscaleStatus runs `tailscale status --json` and reports whether the
// backend is Running (i.e. Taildrop is usable).
func queryTailscaleStatus() (tsStatus, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), taildropStatusTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "tailscale", "status", "--json").Output()
	if err != nil {
		return tsStatus{}, false
	}
	var st tsStatus
	if err := json.Unmarshal(out, &st); err != nil {
		return tsStatus{}, false
	}
	return st, st.BackendState == "Running"
}

// peerDevices flattens the tailnet peers into the sorted device list we expose.
// Pure (no exec) so it can be unit-tested.
func peerDevices(st tsStatus) []taildropDevice {
	devices := []taildropDevice{}
	for _, p := range st.Peer {
		dns := strings.TrimSuffix(p.DNSName, ".")
		if dns == "" {
			continue
		}
		devices = append(devices, taildropDevice{
			Name:    p.HostName,
			DNSName: dns,
			OS:      p.OS,
			Online:  p.Online,
		})
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Name < devices[j].Name
	})
	return devices
}

// resolveTaildropTarget returns the canonical FQDN of the peer matching the
// requested device (by FQDN or short host name), or "" if no peer matches.
// This is the injection guard: only values present in the live peer list are
// ever passed to the CLI. Pure (no exec) so it can be unit-tested.
func resolveTaildropTarget(st tsStatus, device string) string {
	device = strings.TrimSuffix(strings.TrimSpace(device), ":")
	if device == "" {
		return ""
	}
	for _, p := range st.Peer {
		dns := strings.TrimSuffix(p.DNSName, ".")
		if device == dns || (p.HostName != "" && device == p.HostName) {
			return dns
		}
	}
	return ""
}

// TaildropStatus reports whether Taildrop is available and lists target devices.
func (h *Handler) TaildropStatus(w http.ResponseWriter, r *http.Request) {
	st, ok := queryTailscaleStatus()

	resp := struct {
		Available bool             `json:"available"`
		Self      string           `json:"self,omitempty"`
		Devices   []taildropDevice `json:"devices"`
	}{Available: ok, Devices: []taildropDevice{}}

	if ok {
		resp.Self = strings.TrimSuffix(st.Self.DNSName, ".")
		resp.Devices = peerDevices(st)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TaildropSend pushes a share's file to a tailnet device via Taildrop.
func (h *Handler) TaildropSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShareID string `json:"shareId"`
		Device  string `json:"device"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.ShareID = strings.TrimSpace(req.ShareID)
	req.Device = strings.TrimSuffix(strings.TrimSpace(req.Device), ":")
	if req.ShareID == "" || req.Device == "" {
		http.Error(w, "shareId and device are required", http.StatusBadRequest)
		return
	}

	st, ok := queryTailscaleStatus()
	if !ok {
		http.Error(w, "Taildrop is not available (Tailscale not running)", http.StatusServiceUnavailable)
		return
	}

	// Validate the target against the known peer list. Never hand an
	// unvalidated string to the CLI, even though exec uses no shell.
	target := resolveTaildropTarget(st, req.Device)
	if target == "" {
		http.Error(w, "Unknown target device", http.StatusBadRequest)
		return
	}

	// Resolve the share to its on-disk file (file shares only).
	share, found := h.storage.Get(req.ShareID)
	if !found {
		http.Error(w, "Share not found or expired", http.StatusNotFound)
		return
	}
	if share.IsDirectory {
		http.Error(w, "Folder shares cannot be sent via Taildrop", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(h.storage.UploadsDir(), share.FileName)
	if fi, err := os.Stat(filePath); err != nil || fi.IsDir() {
		http.Error(w, "Share file unavailable", http.StatusNotFound)
		return
	}

	// Present the original filename to the receiver. Base() strips any path,
	// so the receiver can never be steered into a subdirectory.
	sendName := filepath.Base(share.OriginalName)

	ctx, cancel := context.WithTimeout(r.Context(), taildropSendTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tailscale", "file", "cp", "--name="+sendName, filePath, target+":")
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		log.Printf("Taildrop send failed (share=%s device=%s): %v - %s", req.ShareID, target, err, msg)
		if msg == "" {
			msg = err.Error()
		}
		http.Error(w, "Taildrop send failed: "+msg, http.StatusBadGateway)
		return
	}

	log.Printf("Taildrop: sent share %s (%s) to %s", req.ShareID, sendName, target)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "sent",
		"device": target,
		"file":   sendName,
	})
}
