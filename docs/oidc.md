# OIDC / SSO Configuration

CasaDrop supports OpenID Connect (OIDC) for single sign-on with identity providers like Authentik, Keycloak, and others.

## Supported Providers

- Authentik
- Keycloak
- Authelia
- Google Workspace
- Azure AD / Entra ID
- Okta
- Any OIDC-compliant provider

## Configuration

### Environment Variables

```bash
# Enable OIDC
OIDC_ENABLED=true

# Provider Configuration
OIDC_ISSUER_URL=https://authentik.example.com/application/o/casadrop/
OIDC_CLIENT_ID=casadrop
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://share.example.com/auth/oidc/callback

# Optional
OIDC_SCOPES=openid,profile,email    # Default scopes
OIDC_DISABLE_LOCAL_AUTH=false       # Keep local admin login
OIDC_DEFAULT_ROLE=viewer            # Default role for new users (admin/user/viewer)
OIDC_AUTO_PROVISION=true            # Auto-create users on first login
```

### Docker Compose Example

```yaml
services:
  casadrop:
    image: chicohaager/casadrop:latest
    environment:
      - OIDC_ENABLED=true
      - OIDC_ISSUER_URL=https://authentik.example.com/application/o/casadrop/
      - OIDC_CLIENT_ID=casadrop
      - OIDC_CLIENT_SECRET=${OIDC_CLIENT_SECRET}
      - OIDC_REDIRECT_URL=https://share.example.com/auth/oidc/callback
      - OIDC_DEFAULT_ROLE=user
```

## Provider Setup

### Authentik

1. Create Application in Authentik:
   - Name: `CasaDrop`
   - Slug: `casadrop`
   - Provider: Create new OAuth2/OpenID Provider

2. Configure OAuth2 Provider:
   - Client type: Confidential
   - Client ID: `casadrop` (auto-generated or custom)
   - Client Secret: Copy this value
   - Redirect URIs: `https://share.example.com/auth/oidc/callback`
   - Scopes: `openid`, `profile`, `email`

3. Issuer URL format:
   ```
   https://authentik.example.com/application/o/casadrop/
   ```

### Keycloak

1. Create Client in your Realm:
   - Client ID: `casadrop`
   - Client Protocol: `openid-connect`
   - Access Type: `confidential`

2. Settings:
   - Valid Redirect URIs: `https://share.example.com/auth/oidc/callback`
   - Web Origins: `https://share.example.com`

3. Credentials tab:
   - Copy the Client Secret

4. Issuer URL format:
   ```
   https://keycloak.example.com/realms/your-realm
   ```

### Authelia

1. Add to Authelia configuration:

```yaml
identity_providers:
  oidc:
    clients:
      - id: casadrop
        description: CasaDrop File Sharing
        secret: '$argon2id$...'  # Generate with authelia hash-password
        redirect_uris:
          - https://share.example.com/auth/oidc/callback
        scopes:
          - openid
          - profile
          - email
```

2. Issuer URL:
   ```
   https://auth.example.com
   ```

## Login Flow

1. User visits CasaDrop login page
2. Clicks "Login with SSO"
3. Redirected to identity provider
4. After authentication, redirected back to `/auth/oidc/callback`
5. User is auto-provisioned (if enabled) with default role
6. Session created, user logged in

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /auth/oidc/login` | Initiate OIDC login |
| `GET /auth/oidc/callback` | Handle provider callback |
| `GET /auth/oidc/logout` | OIDC logout (clears session + IdP logout) |
| `GET /api/auth/oidc/status` | OIDC status (public) |
| `GET /api/auth/oidc/config` | OIDC config (admin only) |

## Combining with Local Auth

By default, both OIDC and local admin login are available:

- `/login` - Shows both local login form and "Login with SSO" button

To disable local auth when OIDC is enabled:

```bash
OIDC_DISABLE_LOCAL_AUTH=true
```

## Multi-User Support

CasaDrop supports three user roles with OIDC:

| Role | Capabilities |
|------|-------------|
| **Admin** | Full access, manage users/settings, see all shares |
| **User** | Create/manage own shares and receive links |
| **Viewer** | Read-only, can only download shared files |

### User Auto-Provisioning

When a user logs in via OIDC for the first time:

1. CasaDrop checks if a user with that OIDC subject exists
2. If not, a new user is created with:
   - Email from the ID token
   - Name from the ID token
   - Role set to `OIDC_DEFAULT_ROLE` (default: `viewer`)
3. The user can then access CasaDrop according to their role

### Managing User Roles

Admins can change user roles via the User Management UI:

1. Click the users icon in the header
2. Find the user in the list
3. Click Edit
4. Change the role
5. Save

### Disabling Auto-Provisioning

To require admin-created accounts:

```bash
OIDC_AUTO_PROVISION=false
```

With this setting, users must be pre-created by an admin before they can log in via OIDC.

### Restricting Access (Provider-side)

You can also control access at the IdP level:

**Authentik**: Add authorization policies to the Application

**Keycloak**: Use Client Scopes or Roles

**Authelia**: Use access control rules

## API Configuration (Admin)

Admins can configure OIDC via the API (when not configured via environment variables):

```bash
# Get current config
curl -X GET http://localhost:8080/api/auth/oidc/config \
  -H "Cookie: casadrop_session=..."

# Update config
curl -X POST http://localhost:8080/api/auth/oidc/config \
  -H "Cookie: casadrop_session=..." \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "issuerUrl": "https://auth.example.com",
    "clientId": "casadrop",
    "clientSecret": "secret",
    "redirectUrl": "https://share.example.com/auth/oidc/callback"
  }'
```

Note: If OIDC is configured via environment variables, API updates are blocked.

## Troubleshooting

### "Invalid redirect URI"

Ensure the redirect URL in CasaDrop matches exactly what's configured in your IdP:

```bash
# Must match exactly
OIDC_REDIRECT_URL=https://share.example.com/auth/oidc/callback
```

### "OIDC discovery failed"

Check that the issuer URL is correct and accessible:

```bash
curl https://authentik.example.com/application/o/casadrop/.well-known/openid-configuration
```

### "User account not found"

If `OIDC_AUTO_PROVISION=false`, the user must be pre-created by an admin.

### "Your account has been disabled"

An admin has deactivated the user account. Contact your administrator.

### Behind Reverse Proxy

Ensure `X-Forwarded-Proto` and `X-Forwarded-Host` headers are passed correctly:

```nginx
# nginx example
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Forwarded-Host $host;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
```

## Security Notes

- Always use HTTPS for OIDC
- Keep `OIDC_CLIENT_SECRET` secure (use Docker secrets or env file)
- Regularly rotate client secrets
- Monitor login attempts in your IdP
- Consider setting `OIDC_DEFAULT_ROLE=viewer` for principle of least privilege
- Use `OIDC_DISABLE_LOCAL_AUTH=true` for maximum security in OIDC-only environments
