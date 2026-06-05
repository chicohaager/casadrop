package auth

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"casadrop/internal/models"
	"casadrop/internal/storage"
)

// Environment variables for user provisioning
const (
	EnvOIDCDefaultRole   = "OIDC_DEFAULT_ROLE"   // Default role for auto-provisioned users
	EnvOIDCAutoProvision = "OIDC_AUTO_PROVISION" // Enable auto-provisioning (default: true)
)

// Auto-provisioning rate limit: guard against a compromised or misbehaving
// IdP trying to spam account creation. 20 new accounts per issuer per hour
// is generous for legit onboarding while hard-capping abuse.
const (
	autoProvisionMaxPerWindow = 20
	autoProvisionWindow       = 1 * time.Hour
)

// provisionBucket tracks auto-provision rate per issuer.
type provisionBucket struct {
	count   int
	resetAt time.Time
}

// UserService handles user lookup and provisioning for authentication
type UserService struct {
	storage       storage.StorageBackend
	defaultRole   models.Role
	autoProvision bool

	provMu      sync.Mutex
	provBuckets map[string]*provisionBucket
}

// NewUserService creates a new user service
func NewUserService(storage storage.StorageBackend) *UserService {
	defaultRole := models.RoleViewer
	if role := os.Getenv(EnvOIDCDefaultRole); role != "" {
		switch models.Role(role) {
		case models.RoleAdmin, models.RoleUser, models.RoleViewer:
			defaultRole = models.Role(role)
		}
	}

	// Auto-provision is enabled by default
	autoProvision := true
	if os.Getenv(EnvOIDCAutoProvision) == "false" {
		autoProvision = false
	}

	return &UserService{
		storage:       storage,
		defaultRole:   defaultRole,
		autoProvision: autoProvision,
		provBuckets:   make(map[string]*provisionBucket),
	}
}

// allowAutoProvision consumes one slot from the per-issuer provision budget.
// Returns nil when the caller is allowed to create a user, or an error when
// the window budget is exhausted.
func (s *UserService) allowAutoProvision(issuer string) error {
	if issuer == "" {
		issuer = "_empty_"
	}
	s.provMu.Lock()
	defer s.provMu.Unlock()

	now := time.Now()
	b, ok := s.provBuckets[issuer]
	if !ok || now.After(b.resetAt) {
		s.provBuckets[issuer] = &provisionBucket{count: 1, resetAt: now.Add(autoProvisionWindow)}
		return nil
	}
	if b.count >= autoProvisionMaxPerWindow {
		return fmt.Errorf("auto-provision budget exhausted for issuer %q (%d / %s)",
			issuer, autoProvisionMaxPerWindow, autoProvisionWindow)
	}
	b.count++
	return nil
}

// FindOrCreateOIDCUser finds an existing user by OIDC credentials or creates a new one
func (s *UserService) FindOrCreateOIDCUser(userInfo *UserInfo) (*models.User, error) {
	// First try to find by OIDC subject+issuer
	user, err := s.storage.GetUserByOIDC(userInfo.Subject, userInfo.Issuer)
	if err != nil {
		return nil, err
	}
	if user != nil {
		// Update last login time
		now := time.Now().UTC()
		user.LastLoginAt = &now
		// Update email/name if changed
		if userInfo.Email != "" && userInfo.Email != user.Email {
			user.Email = userInfo.Email
		}
		if userInfo.Name != "" && userInfo.Name != user.Name {
			user.Name = userInfo.Name
		}
		if err := s.storage.UpdateUser(user); err != nil {
			// Non-fatal, just log
		}
		return user, nil
	}

	// Try to find by email. Only link/provision by email when the IdP asserts
	// the address is verified — otherwise a malicious or misconfigured provider
	// could claim an existing admin's email and take over the account.
	if userInfo.Email != "" && userInfo.EmailVerified {
		user, err = s.storage.GetUserByEmail(userInfo.Email)
		if err != nil {
			return nil, err
		}
		if user != nil {
			// Link OIDC to existing user
			user.OIDCSubject = userInfo.Subject
			user.OIDCIssuer = userInfo.Issuer
			now := time.Now().UTC()
			user.LastLoginAt = &now
			if err := s.storage.UpdateUser(user); err != nil {
				return nil, err
			}
			return user, nil
		}
	}

	// Auto-provision new user if enabled
	if !s.autoProvision {
		return nil, nil // Return nil user (will be treated as unauthorized)
	}

	// Refuse to provision off an unverified email: it would either collide with
	// the existing account we just declined to link, or silently claim an
	// address the user hasn't proven they own. Fail closed → unauthorized.
	if userInfo.Email != "" && !userInfo.EmailVerified {
		return nil, nil
	}

	// Rate-limit per issuer so a compromised IdP can't spam account creation.
	if err := s.allowAutoProvision(userInfo.Issuer); err != nil {
		return nil, err
	}

	// Create new user
	now := time.Now().UTC()
	user = &models.User{
		ID:          uuid.New().String(),
		Email:       userInfo.Email,
		Name:        userInfo.Name,
		Role:        s.defaultRole,
		OIDCSubject: userInfo.Subject,
		OIDCIssuer:  userInfo.Issuer,
		IsActive:    true,
		CreatedAt:   now,
		LastLoginAt: &now,
	}

	// Use email prefix as name if no name provided
	if user.Name == "" && user.Email != "" {
		for i, c := range user.Email {
			if c == '@' {
				user.Name = user.Email[:i]
				break
			}
		}
	}
	if user.Name == "" {
		user.Name = "User"
	}

	if err := s.storage.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetDefaultRole returns the default role for auto-provisioned users
func (s *UserService) GetDefaultRole() models.Role {
	return s.defaultRole
}

// IsAutoProvisionEnabled returns whether auto-provisioning is enabled
func (s *UserService) IsAutoProvisionEnabled() bool {
	return s.autoProvision
}
