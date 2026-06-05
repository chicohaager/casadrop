package handlers

import "net/http"

// Healthz is a liveness probe: it returns 200 as long as the process is
// serving requests. It performs no dependency checks so orchestrators can
// distinguish "process up" from "dependencies ready".
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Readyz is a readiness probe: it returns 200 only when the storage backend
// is reachable, otherwise 503. Use this for load-balancer/tunnel health gating.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err := h.storage.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("storage unavailable"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
