package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"casadrop/internal/models"
	"casadrop/internal/utils"
)

// SessionInfo is the safe, outward view of a session: enough to recognise a
// device ("Firefox on Linux, from this address, since Tuesday") without ever
// exposing anything token-derived. The opaque ID is a random handle, not the
// token hash, so listing sessions leaks nothing about the credential.
type SessionInfo struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Current   bool      `json:"current"`
	UserEmail string    `json:"user_email,omitempty"` // populated only in the admin, all-users view
}

// currentSessionKey returns the storage key (token hash) of the session the
// request is authenticated with, or "" if none — so a listing can mark "this
// device" and revoke-others can spare it. It never returns the raw token.
func (aa *AdminAuth) currentSessionKey(r *http.Request) string {
	if cookie, err := r.Cookie("casadrop_session"); err == nil && cookie.Value != "" {
		return hashToken(cookie.Value)
	}
	if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		return hashToken(h[7:])
	}
	return ""
}

// ListSessionsHandler answers GET /api/sessions. An admin sees every active
// session (to spot and kick a stranger); anyone else sees only their own
// devices. The caller's current session is flagged so the UI can label it and
// refuse to let them cut the branch they are sitting on by accident.
func (aa *AdminAuth) ListSessionsHandler(w http.ResponseWriter, r *http.Request) {
	caller := GetUserFromContext(r.Context())
	currentKey := aa.currentSessionKey(r)
	isAdmin := caller != nil && caller.Role == models.RoleAdmin

	aa.mu.RLock()
	out := make([]SessionInfo, 0, len(aa.sessions))
	for key, s := range aa.sessions {
		if !isAdmin && !sameIdentity(caller, s) {
			continue
		}
		info := SessionInfo{
			ID:        s.ID,
			IP:        s.IP,
			UserAgent: s.UserAgent,
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
			Current:   key == currentKey,
		}
		if isAdmin {
			info.UserEmail = s.UserEmail
		}
		out = append(out, info)
	}
	aa.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sessions": out})
}

// RevokeSessionHandler answers DELETE /api/sessions/{id}. A user may end their
// own sessions; an admin may end anyone's (to remove a compromised device). A
// session that does not exist and one the caller may not touch are both 404, so
// the endpoint cannot be used to enumerate other people's session IDs.
func (aa *AdminAuth) RevokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	caller := GetUserFromContext(r.Context())
	isAdmin := caller != nil && caller.Role == models.RoleAdmin

	aa.mu.Lock()
	var foundKey string
	var revoked Session
	for key, s := range aa.sessions {
		if s.ID == id && (isAdmin || sameIdentity(caller, s)) {
			foundKey, revoked = key, s
			break
		}
	}
	if foundKey != "" {
		delete(aa.sessions, foundKey)
		aa.saveSessions()
	}
	aa.mu.Unlock()

	if foundKey == "" {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	aa.audit(AuditSessionRevoked, utils.GetClientIP(r), r.Header.Get("User-Agent"),
		"Revoked session "+id+" ("+revoked.UserEmail+")")
	w.WriteHeader(http.StatusNoContent)
}

// RevokeOtherSessionsHandler answers POST /api/sessions/revoke-others: end every
// session that shares the caller's identity except the one making the request —
// the "log out everywhere else" a user reaches for after losing a laptop. It
// never touches other users' sessions, admin or not.
func (aa *AdminAuth) RevokeOtherSessionsHandler(w http.ResponseWriter, r *http.Request) {
	caller := GetUserFromContext(r.Context())
	currentKey := aa.currentSessionKey(r)

	aa.mu.Lock()
	revoked := 0
	for key, s := range aa.sessions {
		if key == currentKey {
			continue // spare the branch we are sitting on
		}
		if sameIdentity(caller, s) {
			delete(aa.sessions, key)
			revoked++
		}
	}
	if revoked > 0 {
		aa.saveSessions()
	}
	aa.mu.Unlock()

	if revoked > 0 {
		aa.audit(AuditSessionRevoked, utils.GetClientIP(r), r.Header.Get("User-Agent"),
			"Revoked all other sessions")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"revoked": revoked})
}

// sameIdentity reports whether a session belongs to the caller. A per-user
// account is matched by user ID; the single-admin session (env password / setup
// wizard) has no user record, so all such sessions share the empty identity and
// an admin-role caller with no ID matches them.
func sameIdentity(caller *SessionUser, s Session) bool {
	if caller == nil {
		return false
	}
	if caller.ID != "" {
		return s.UserID == caller.ID
	}
	// Legacy single-admin: no user id. Match the other id-less admin sessions,
	// never a real user's session (which always carries an id).
	return s.UserID == "" && caller.Role == models.RoleAdmin
}
