//go:build !linux

package handlers

// statfsFreeBytes is unknown off Linux; the free-space preflight is skipped.
func statfsFreeBytes(string) (int64, bool) { return 0, false }
