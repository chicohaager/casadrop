package models

import (
	"time"
)

// Role represents a user's permission level
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
)

// User represents a CasaDrop user
type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	Role         Role       `json:"role"`
	PasswordHash string     `json:"-"` // Never serialize password hash
	OIDCSubject  string     `json:"oidc_subject,omitempty"`
	OIDCIssuer   string     `json:"oidc_issuer,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsUser returns true if the user has user role
func (u *User) IsUser() bool {
	return u.Role == RoleUser
}

// IsViewer returns true if the user has viewer role
func (u *User) IsViewer() bool {
	return u.Role == RoleViewer
}

// CanCreateShares returns true if the user can create shares
func (u *User) CanCreateShares() bool {
	return u.Role == RoleAdmin || u.Role == RoleUser
}

// CanManageAllShares returns true if the user can manage all shares (admin only)
func (u *User) CanManageAllShares() bool {
	return u.Role == RoleAdmin
}

// CanManageUsers returns true if the user can manage other users (admin only)
func (u *User) CanManageUsers() bool {
	return u.Role == RoleAdmin
}

// CanManageSettings returns true if the user can manage system settings (admin only)
func (u *User) CanManageSettings() bool {
	return u.Role == RoleAdmin
}

// CanAccessShare returns true if the user can access a specific share
func (u *User) CanAccessShare(share *Share) bool {
	// Admin can access all shares
	if u.Role == RoleAdmin {
		return true
	}
	// Users can access their own shares
	if share.UserID == u.ID {
		return true
	}
	// Viewers can view shares but not manage them
	return false
}

// CanDeleteShare returns true if the user can delete a specific share
func (u *User) CanDeleteShare(share *Share) bool {
	// Admin can delete all shares
	if u.Role == RoleAdmin {
		return true
	}
	// Users can delete their own shares
	if u.Role == RoleUser && share.UserID == u.ID {
		return true
	}
	// Viewers cannot delete shares
	return false
}

// CanAccessReceiveLink returns true if the user can access a specific receive link
func (u *User) CanAccessReceiveLink(link *ReceiveLink) bool {
	// Admin can access all receive links
	if u.Role == RoleAdmin {
		return true
	}
	// Users can access their own receive links
	if link.UserID == u.ID {
		return true
	}
	return false
}

// CanDeleteReceiveLink returns true if the user can delete a specific receive link
func (u *User) CanDeleteReceiveLink(link *ReceiveLink) bool {
	// Admin can delete all receive links
	if u.Role == RoleAdmin {
		return true
	}
	// Users can delete their own receive links
	if u.Role == RoleUser && link.UserID == u.ID {
		return true
	}
	return false
}

// IsOIDCUser returns true if this user was created via OIDC
func (u *User) IsOIDCUser() bool {
	return u.OIDCSubject != "" && u.OIDCIssuer != ""
}

// HasLocalPassword returns true if user has a local password set
func (u *User) HasLocalPassword() bool {
	return u.PasswordHash != ""
}

// ValidRoles returns all valid role values
func ValidRoles() []Role {
	return []Role{RoleAdmin, RoleUser, RoleViewer}
}

// IsValidRole checks if a role string is valid
func IsValidRole(role Role) bool {
	for _, r := range ValidRoles() {
		if r == role {
			return true
		}
	}
	return false
}
