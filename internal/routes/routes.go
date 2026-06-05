// Package routes wires the HTTP router for CasaDrop.
//
// It's intentionally stateless: New() takes fully-constructed dependencies
// and returns a configured *mux.Router. This lets tests build a router
// with stub dependencies via httptest.NewServer without ever touching
// main().
package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"casadrop/internal/auth"
	"casadrop/internal/handlers"
	"casadrop/internal/metrics"
	"casadrop/internal/middleware"
	"casadrop/internal/utils"
)

// Deps bundles everything the router needs. Keeping this in one struct
// means callers can't accidentally forget to pass (say) the email handler
// and end up with a broken /api/email/send route at runtime.
type Deps struct {
	Handler         *handlers.Handler
	AdminAuth       *middleware.AdminAuth
	OIDC            *auth.Handlers
	EmailHandler    *handlers.EmailHandler
	DownloadLimiter *middleware.RateLimiter
	StaticDir       string
}

// New returns a configured router. It is the single source of truth for
// URL → handler mapping so main.go stays ~60 lines.
func New(d Deps) *mux.Router {
	r := mux.NewRouter()

	// Global middleware
	r.Use(metrics.Middleware)
	r.Use(middleware.SecurityHeaders)

	registerPublic(r, d)
	registerProtected(r, d)

	return r
}

func registerPublic(r *mux.Router, d Deps) {
	h := d.Handler

	// Health probes (public, no auth) for container/tunnel orchestration
	r.HandleFunc("/healthz", h.Healthz).Methods("GET")
	r.HandleFunc("/readyz", h.Readyz).Methods("GET")

	// Auth + OIDC (public by definition)
	r.HandleFunc("/login", d.AdminAuth.LoginHandler).Methods("GET", "POST")
	r.HandleFunc("/logout", d.AdminAuth.LogoutHandler).Methods("GET", "POST")
	r.HandleFunc("/setup", d.AdminAuth.SetupHandler).Methods("GET", "POST")
	r.HandleFunc("/api/auth/status", d.AdminAuth.AuthStatusHandler).Methods("GET")

	r.HandleFunc("/auth/oidc/login", d.OIDC.LoginHandler).Methods("GET")
	r.HandleFunc("/auth/oidc/callback", d.OIDC.CallbackHandler).Methods("GET")
	r.HandleFunc("/auth/oidc/logout", d.OIDC.LogoutHandler).Methods("GET")
	r.HandleFunc("/api/auth/oidc/status", d.OIDC.StatusHandler).Methods("GET")

	// Share landing / download (rate-limited)
	r.HandleFunc("/s/{id}", h.SharePage).Methods("GET")
	r.HandleFunc("/d/{id}", rateLimitDownload(h.DownloadFile, d.DownloadLimiter)).Methods("GET")
	r.HandleFunc("/stream/{id}", h.StreamFile).Methods("GET", "HEAD", "OPTIONS")
	r.HandleFunc("/qr/{id}", h.QRCode).Methods("GET")
	r.HandleFunc("/thumbnail/{id}", h.GetThumbnail).Methods("GET")

	// Folder share public routes (password protected at handler level)
	r.HandleFunc("/folder/{id}/contents", h.GetFolderContents).Methods("GET")
	r.HandleFunc("/folder/{id}/download", h.DownloadFolderFile).Methods("GET")
	r.HandleFunc("/folder/{id}/zip", h.DownloadFolderZip).Methods("GET")

	// Receive link public routes (for uploaders)
	r.HandleFunc("/r/{id}", h.ReceivePage).Methods("GET")
	r.HandleFunc("/r/{id}/upload", h.ReceiveUpload).Methods("POST")

	// Force the browser to revalidate cached assets on every load. FileServer
	// already answers If-Modified-Since with cheap 304s via file mtime, so this
	// costs almost nothing while guaranteeing an updated app.js/style.css is
	// picked up immediately after a redeploy (instead of serving a stale copy).
	static := http.StripPrefix("/static/", http.FileServer(http.Dir(d.StaticDir)))
	r.PathPrefix("/static/").Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		static.ServeHTTP(w, req)
	}))
}

func registerProtected(r *mux.Router, d Deps) {
	h := d.Handler
	aa := d.AdminAuth

	protected := r.PathPrefix("").Subrouter()
	protected.Use(aa.Middleware)

	// Index page
	protected.HandleFunc("/", h.IndexPage).Methods("GET")

	// API root
	api := protected.PathPrefix("/api").Subrouter()
	// Defense-in-depth CSRF: reject cross-site cookie-authenticated mutations.
	api.Use(middleware.CrossSiteGuard)
	api.Use(middleware.MaxBodySizeSkipPaths(1<<20,
		"/api/upload", "/api/upload/multi", "/api/upload/chunk"))

	registerAPIShares(api, aa, h)
	registerAPIMisc(api, aa, h, d.OIDC)
	registerAPIFolderReceive(api, aa, h)
	registerAPIUsers(api, aa, h)
	registerAPIEmail(api, aa, d.EmailHandler)
	registerAPITailscale(api, aa, h)
}

func registerAPIShares(api *mux.Router, aa *middleware.AdminAuth, h *handlers.Handler) {
	// Upload/share creation (User + Admin)
	api.Handle("/upload", aa.RequireCanCreateShares()(http.HandlerFunc(h.UploadFile))).Methods("POST")
	api.Handle("/upload/multi", aa.RequireCanCreateShares()(http.HandlerFunc(h.UploadMultipleFiles))).Methods("POST")
	api.Handle("/upload/chunk/init", aa.RequireCanCreateShares()(http.HandlerFunc(h.InitChunkUpload))).Methods("POST")
	api.Handle("/upload/chunk/{uploadId}", aa.RequireCanCreateShares()(http.HandlerFunc(h.UploadChunk))).Methods("POST")
	api.Handle("/upload/chunk/{uploadId}/finalize", aa.RequireCanCreateShares()(http.HandlerFunc(h.FinalizeChunkUpload))).Methods("POST")
	// Sharing an arbitrary server path is a host-filesystem operation gated by
	// SHARE_ALLOWED_PATHS (defaults include /home, /DATA). It must be admin-only:
	// a non-admin "user" must not be able to browse and exfiltrate other users'
	// files (e.g. ~/.ssh keys) via the allow-list roots.
	api.Handle("/share-from-path", aa.RequireAdmin()(http.HandlerFunc(h.ShareFromPath))).Methods("POST")

	api.HandleFunc("/shares", h.ListShares).Methods("GET")
	// NOTE: bulk-delete must be registered before /shares/{id}
	// otherwise {id} matches the literal "bulk-delete".
	api.Handle("/shares/bulk-delete", aa.RequireCanCreateShares()(http.HandlerFunc(h.BulkDeleteShares))).Methods("POST")
	api.HandleFunc("/shares/{id}", h.GetShareInfo).Methods("GET")
	api.HandleFunc("/shares/{id}", h.UpdateShare).Methods("PUT")
	api.HandleFunc("/shares/{id}", h.DeleteShare).Methods("DELETE")
}

func registerAPIMisc(api *mux.Router, aa *middleware.AdminAuth, h *handlers.Handler, oidc *auth.Handlers) {
	api.HandleFunc("/stats", h.GetStats).Methods("GET")
	// Filesystem browser is admin-only (see /share-from-path rationale).
	api.Handle("/browse", aa.RequireAdmin()(http.HandlerFunc(h.BrowseFiles))).Methods("GET")
	api.HandleFunc("/network", h.GetNetworkInfo).Methods("GET")
	api.Handle("/metrics", aa.RequireAdmin()(promhttp.Handler())).Methods("GET")
	api.Handle("/webhook", aa.RequireAdmin()(http.HandlerFunc(h.WebhookConfig))).Methods("GET", "POST")
	api.Handle("/webhook/test", aa.RequireAdmin()(http.HandlerFunc(h.TestWebhook))).Methods("POST")
	api.Handle("/tunnel", aa.RequireAdmin()(http.HandlerFunc(h.TunnelURL))).Methods("GET", "POST")
	api.Handle("/auth/oidc/config", aa.RequireAdmin()(http.HandlerFunc(oidc.ConfigHandler))).Methods("GET", "POST")

	// Admin 2FA (TOTP) management
	api.Handle("/admin/2fa", aa.RequireAdmin()(http.HandlerFunc(aa.TOTPStatusHandler))).Methods("GET")
	api.Handle("/admin/2fa/setup", aa.RequireAdmin()(http.HandlerFunc(aa.TOTPSetupHandler))).Methods("GET")
	api.Handle("/admin/2fa/enable", aa.RequireAdmin()(http.HandlerFunc(aa.TOTPEnableHandler))).Methods("POST")
	api.Handle("/admin/2fa/disable", aa.RequireAdmin()(http.HandlerFunc(aa.TOTPDisableHandler))).Methods("POST")
}

func registerAPIFolderReceive(api *mux.Router, aa *middleware.AdminAuth, h *handlers.Handler) {
	// Sharing a host folder browses the filesystem under SHARE_ALLOWED_PATHS —
	// admin-only, same rationale as /browse and /share-from-path.
	api.Handle("/share-folder", aa.RequireAdmin()(http.HandlerFunc(h.ShareFolder))).Methods("POST")

	api.HandleFunc("/receive-links", h.ListReceiveLinks).Methods("GET")
	api.Handle("/receive-links", aa.RequireCanCreateShares()(http.HandlerFunc(h.CreateReceiveLink))).Methods("POST")
	api.HandleFunc("/receive-links/{id}", h.GetReceiveLink).Methods("GET")
	api.HandleFunc("/receive-links/{id}", h.DeleteReceiveLink).Methods("DELETE")
	api.HandleFunc("/receive-links/{id}/files", h.GetReceivedFiles).Methods("GET")
	api.HandleFunc("/receive-links/{id}/files/{fileId}", h.DownloadReceivedFile).Methods("GET")
}

func registerAPIUsers(api *mux.Router, aa *middleware.AdminAuth, h *handlers.Handler) {
	// /api/me is writable by any authenticated user
	api.HandleFunc("/me", h.GetCurrentUser).Methods("GET")
	api.HandleFunc("/me", h.UpdateCurrentUser).Methods("PUT")

	users := api.PathPrefix("/users").Subrouter()
	users.Use(aa.RequireAdmin())
	users.HandleFunc("", h.ListUsers).Methods("GET")
	users.HandleFunc("", h.CreateUser).Methods("POST")
	users.HandleFunc("/{id}", h.GetUser).Methods("GET")
	users.HandleFunc("/{id}", h.UpdateUser).Methods("PUT")
	users.HandleFunc("/{id}", h.DeleteUser).Methods("DELETE")

	api.Handle("/api-keys", aa.RequireAdmin()(http.HandlerFunc(h.ListAPIKeys))).Methods("GET")
	api.Handle("/api-keys", aa.RequireAdmin()(http.HandlerFunc(h.CreateAPIKey))).Methods("POST")
	api.Handle("/api-keys/{id}", aa.RequireAdmin()(http.HandlerFunc(h.DeleteAPIKey))).Methods("DELETE")
}

func registerAPIEmail(api *mux.Router, aa *middleware.AdminAuth, eh *handlers.EmailHandler) {
	api.Handle("/smtp", aa.RequireAdmin()(http.HandlerFunc(eh.GetSMTPConfig))).Methods("GET")
	api.Handle("/smtp", aa.RequireAdmin()(http.HandlerFunc(eh.SaveSMTPConfig))).Methods("POST")
	api.Handle("/smtp/test", aa.RequireAdmin()(http.HandlerFunc(eh.TestSMTPConnection))).Methods("POST")
	api.HandleFunc("/email/status", eh.GetEmailStatus).Methods("GET")
	api.Handle("/email/send", aa.RequireCanCreateShares()(http.HandlerFunc(eh.SendEmailTransfer))).Methods("POST")
}

func registerAPITailscale(api *mux.Router, aa *middleware.AdminAuth, h *handlers.Handler) {
	api.Handle("/tailscale", aa.RequireAdmin()(http.HandlerFunc(h.GetTailscaleConfig))).Methods("GET")
	api.Handle("/tailscale", aa.RequireAdmin()(http.HandlerFunc(h.SaveTailscaleConfig))).Methods("POST")
	api.Handle("/tailscale/start", aa.RequireAdmin()(http.HandlerFunc(h.StartTailscaleHandler))).Methods("POST")
	api.Handle("/tailscale/stop", aa.RequireAdmin()(http.HandlerFunc(h.StopTailscaleHandler))).Methods("POST")

	// Taildrop — send an existing share's file to a tailnet device (admin only)
	api.Handle("/taildrop/status", aa.RequireAdmin()(http.HandlerFunc(h.TaildropStatus))).Methods("GET")
	api.Handle("/taildrop/send", aa.RequireAdmin()(http.HandlerFunc(h.TaildropSend))).Methods("POST")
}

// rateLimitDownload wraps a handler with per-IP rate limiting. Extracted
// from the old cmd/server/main.go so it lives next to the router that
// actually uses it.
func rateLimitDownload(handler http.HandlerFunc, limiter *middleware.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := utils.GetClientIP(r)
		if !limiter.Allow(clientIP) {
			http.Error(w, "Zu viele Anfragen. Bitte warten.", http.StatusTooManyRequests)
			return
		}
		handler(w, r)
	}
}
