package handlers

import (
	"log"
	"net/http"
)

// Healthz is a liveness probe: it returns 200 as long as the process is
// serving requests. It performs no dependency checks so orchestrators can
// distinguish "process up" from "dependencies ready".
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Readyz is a readiness probe: it returns 200 only when the storage backend is
// actually readable (see storage.Ping — it reads a real page, not `select 1`),
// otherwise 503. Use this for load-balancer/tunnel health gating.
//
// The container HEALTHCHECK deliberately stays on /healthz: pointing it here
// would let a transient SQLITE_BUSY flip the container to unhealthy, and under
// an orchestrator that restarts on unhealthy that turns a slow moment into a
// restart loop under exactly the load that caused the lock. Operators who want
// the stricter signal can point their own probe at this endpoint.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err := h.storage.Ping(); err != nil {
		// Log it. The response body is a fixed string, so without this line the
		// only record of *why* readiness failed was gone — an operator saw a
		// 503 and had nothing to go on.
		log.Printf("Readiness check failed: storage unreadable: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("storage unavailable"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
