//go:build linux

package handlers

import "syscall"

// statfsFreeBytes reports the bytes available to an unprivileged writer on the
// filesystem holding path (Bavail, not Bfree: the root-reserved blocks are not
// usable by the app user).
func statfsFreeBytes(path string) (int64, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, false
	}
	// Bsize is int64 on amd64/arm64 but int32 on 386/arm, so the conversion
	// stays for the 32-bit builds.
	return int64(st.Bavail) * int64(st.Bsize), true //nolint:unconvert
}
