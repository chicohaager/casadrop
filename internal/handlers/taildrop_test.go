package handlers

import (
	"encoding/json"
	"testing"
)

// A trimmed but realistic `tailscale status --json` payload.
const sampleTailscaleStatus = `{
  "BackendState": "Running",
  "Self": { "DNSName": "server.tail-scale.ts.net.", "HostName": "server", "OS": "linux", "Online": true },
  "Peer": {
    "nodekey:aaa": { "DNSName": "pixel-7.tail-scale.ts.net.", "HostName": "pixel-7", "OS": "android", "Online": true },
    "nodekey:bbb": { "DNSName": "laptop.tail-scale.ts.net.",  "HostName": "laptop",  "OS": "macOS",   "Online": false }
  }
}`

func parseSample(t *testing.T) tsStatus {
	t.Helper()
	var st tsStatus
	if err := json.Unmarshal([]byte(sampleTailscaleStatus), &st); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if st.BackendState != "Running" {
		t.Fatalf("BackendState = %q, want Running", st.BackendState)
	}
	return st
}

func TestPeerDevices_SortedAndTrimmed(t *testing.T) {
	st := parseSample(t)
	devices := peerDevices(st)
	if len(devices) != 2 {
		t.Fatalf("got %d devices, want 2", len(devices))
	}
	// Sorted by short host name: laptop before pixel-7.
	if devices[0].Name != "laptop" || devices[1].Name != "pixel-7" {
		t.Fatalf("unexpected order: %q, %q", devices[0].Name, devices[1].Name)
	}
	// Trailing dot must be stripped from the FQDN.
	if devices[1].DNSName != "pixel-7.tail-scale.ts.net" {
		t.Fatalf("DNSName not trimmed: %q", devices[1].DNSName)
	}
	if devices[1].Online != true || devices[0].Online != false {
		t.Fatalf("online flags wrong: %+v", devices)
	}
}

func TestResolveTaildropTarget(t *testing.T) {
	st := parseSample(t)

	cases := []struct {
		name   string
		device string
		want   string
	}{
		{"by short hostname", "pixel-7", "pixel-7.tail-scale.ts.net"},
		{"by FQDN", "pixel-7.tail-scale.ts.net", "pixel-7.tail-scale.ts.net"},
		{"by FQDN with trailing colon", "laptop.tail-scale.ts.net:", "laptop.tail-scale.ts.net"},
		{"with surrounding space", "  laptop  ", "laptop.tail-scale.ts.net"},
		{"unknown device rejected", "attacker; rm -rf /", ""},
		{"empty rejected", "", ""},
		{"self is not a target", "server", ""},
		{"partial match rejected", "pixel", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveTaildropTarget(st, c.device); got != c.want {
				t.Fatalf("resolveTaildropTarget(%q) = %q, want %q", c.device, got, c.want)
			}
		})
	}
}
