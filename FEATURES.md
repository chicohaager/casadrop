# CasaDrop Feature Overview

## What's New in v2.0

| Version | Feature | Description |
|---------|---------|-------------|
| v1.6 | SQLite Database | Migrated from JSON to SQLite with WAL mode |
| v1.7 | Prometheus Metrics | Monitor at `/metrics` endpoint |
| v1.8 | Image Thumbnails | Automatic thumbnail generation |
| v1.9 | Folder Shares | Browse folders, download as ZIP |
| v2.0 | Receive Links | Let others upload files to you |

---

## Architecture Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ZIMA-SHARE                                     │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │    User      │
                              └──────┬───────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
              ▼                      ▼                      ▼
    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
    │  Local Network  │    │    ZeroTier     │    │   Cloudflare    │
    │ 192.168.x.x:80  │    │ 10.147.x.x:80   │    │    Tunnel       │
    └────────┬────────┘    └────────┬────────┘    └────────┬────────┘
             │                      │                      │
             └──────────────────────┼──────────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │       Docker Container        │
                    │      ┌─────────────────┐      │
                    │      │   Go Backend    │      │
                    │      │  (Port 8080)    │      │
                    │      └────────┬────────┘      │
                    │               │               │
                    │    ┌──────────┼──────────┐    │
                    │    │          │          │    │
                    │    ▼          ▼          ▼    │
                    │ ┌────────┐ ┌────────┐ ┌─────┐ │
                    │ │Handlers│ │ SQLite │ │Thumb│ │
                    │ └────────┘ └────────┘ └─────┘ │
                    │                              │
                    └───────────────────────────────┘
                                    │
                                    ▼
                          ┌─────────────────┐
                          │  Docker Volume  │
                          │   /data         │
                          │  ├─ shares.db   │
                          │  ├─ uploads/    │
                          │  ├─ received/   │
                          │  └─ thumbnails/ │
                          └─────────────────┘
```

---

## Upload Flow

```
┌────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  User  │────▶│  Web UI     │────▶│  /api/upload │────▶│   Storage   │
│        │     │  (Drag&Drop)│     │              │     │             │
└────────┘     └─────────────┘     └──────────────┘     └─────────────┘
                                          │
                     ┌────────────────────┼────────────────────┐
                     │                    │                    │
                     ▼                    ▼                    ▼
              ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
              │  Validate   │     │   Hash      │     │   Save      │
              │  File Type  │     │  Password   │     │   File      │
              │  (Security) │     │  (bcrypt)   │     │  + Metadata │
              └─────────────┘     └─────────────┘     └─────────────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │  Generate   │
                                   │  Share URL  │
                                   │  + QR Code  │
                                   └─────────────┘
```

---

## Download Flow

```
┌────────┐     ┌─────────────┐     ┌──────────────┐
│  User  │────▶│  /s/{id}    │────▶│  Check Share │
│        │     │  Share Page │     │  Exists?     │
└────────┘     └─────────────┘     └──────┬───────┘
                                          │
                          ┌───────────────┴───────────────┐
                          │                               │
                          ▼                               ▼
                   ┌─────────────┐                 ┌─────────────┐
                   │  Expired?   │                 │  Not Found  │
                   │  Max DL?    │                 │  404 Page   │
                   └──────┬──────┘                 └─────────────┘
                          │
              ┌───────────┴───────────┐
              │                       │
              ▼                       ▼
       ┌─────────────┐         ┌─────────────┐
       │  Password   │         │  No Password│
       │  Required   │         │  Direct DL  │
       └──────┬──────┘         └─────────────┘
              │
              ▼
       ┌─────────────┐
       │  Verify     │────▶ Success ────▶ Download
       │  Password   │
       └─────────────┘────▶ Fail ────▶ Retry (Rate Limited)
```

---

## Network Detection Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                         start.sh                                    │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │  1. Detect ZeroTier IP (interface: zt*)                     │   │
│   │  2. Detect Local IP (first non-docker, non-loopback)        │   │
│   │  3. Export as environment variables                         │   │
│   └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Docker Container                               │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │  Environment Variables:                                      │   │
│   │  - LOCAL_IP=192.168.x.x                                     │   │
│   │  - ZEROTIER_IP=10.147.x.x                                   │   │
│   │  - EXTERNAL_PORT=8080                                       │   │
│   └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        /api/network                                 │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │  Priority for ZeroTier IP:                                  │   │
│   │  1. User config (tunnel_config.json)                        │   │
│   │  2. Environment variable (ZEROTIER_IP)                      │   │
│   │  3. Auto-detect from network interfaces                     │   │
│   └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Web UI                                     │
│   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐       │
│   │ Cloudflare URL  │ │  ZeroTier URL   │ │   Local URL     │       │
│   │ https://...     │ │ http://10.x:80  │ │ http://192.x:80 │       │
│   └─────────────────┘ └─────────────────┘ └─────────────────┘       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Core Features

### File Sharing
- **Drag & Drop Upload** - Simply drag files onto the upload area
- **Large File Support** - Upload files up to 100 GB (configurable)
- **QR Code Generation** - Automatic QR code for easy mobile sharing
- **One-Click Copy** - Copy share links instantly to clipboard
- **Image Thumbnails** - Automatic thumbnail generation for images

### Folder Sharing (v1.9)
- **Share Directories** - Share entire folders with a single link
- **Browse UI** - Navigate folder contents in web browser
- **ZIP Download** - Download entire folder as ZIP (on-demand)
- **Individual Files** - Download single files from shared folder
- **Password Protection** - Optional password for folder shares

### Receive Links (v2.0)
- **Upload Links** - Create links for others to upload files to you
- **Upload Limits** - Set maximum number of uploads per link
- **File Size Limits** - Restrict file sizes per receive link
- **Extension Filter** - Whitelist allowed file extensions
- **Auto-Share** - Automatically create share links for received files
- **Webhooks** - Get notified when files are received

### Security
- **Password Protection** - Optional password for each share
- **Password Generator** - Built-in secure password generator
- **Blocked File Types** - Executable files (.exe, .bat, .ps1, etc.) are automatically blocked
- **bcrypt Hashing** - Secure password storage
- **Rate Limiting** - Protection against brute-force attacks
- **Security Headers** - CSP, X-Frame-Options, HSTS, X-Content-Type-Options
- **SQLite Database** - WAL mode for safe concurrent access

### Expiration & Limits
- **Flexible Expiration** - 1 hour, 6 hours, 24 hours, 3 days, 7 days, or 30 days
- **Download Limits** - Set maximum number of downloads (0 = unlimited)
- **Auto-Cleanup** - Expired files are automatically removed (hourly check)

### Monitoring (v1.7)
- **Prometheus Metrics** - `/metrics` endpoint for monitoring
- **Upload/Download Counts** - Track file operations
- **Active Shares** - Monitor current share count
- **HTTP Metrics** - Request counts and latency

---

## Network Access Options

### Three Ways to Access

| Method | URL Format | Use Case |
|--------|------------|----------|
| **Local Network** | `http://192.168.x.x:8080` | Same WiFi/LAN |
| **ZeroTier VPN** | `http://10.147.x.x:8080` | Remote access via VPN |
| **Cloudflare Tunnel** | `https://xxx.trycloudflare.com` | Public internet access |

### Automatic IP Detection

```bash
# start.sh automatically detects:
./start.sh up -d

# Output:
# Detected IPs:
#   ZeroTier: 10.147.19.1
#   Local:    192.168.1.100
```

### Network Settings UI
- **Gear Icon** - Access network settings from the header
- **ZeroTier IP Input** - Manual override if auto-detection fails
- **Cloudflare Checkbox** - Enable external tunnel URL
- **Persistent Config** - Settings saved to `tunnel_config.json`

---

## Cloudflare Tunnel Options

### Three Tunnel Options

| Option | Description | Use Case |
|--------|-------------|----------|
| **Quick Tunnel** | Temporary random URL | Testing, quick sharing |
| **Token Tunnel** | Permanent custom domain | Production use |
| **External Tunnel** | Use existing Cloudflared-Web | ZimaOS integration |

### Quick Tunnel (No Config)
```bash
docker compose --profile tunnel up -d
# URL: https://random-words.trycloudflare.com
```

### Token Tunnel (Permanent Domain)
```bash
CLOUDFLARE_TUNNEL_TOKEN=xxx docker compose --profile tunnel up -d
# URL: https://share.yourdomain.com
```

### External Tunnel (Cloudflared-Web)
1. Configure tunnel in Cloudflared-Web UI
2. Point to `http://zima-share:8080`
3. Enable checkbox in Zima-Share settings
4. Enter your public URL

---

## User Interface

### Themes
- **Dark Mode** - Default dark theme, easy on the eyes
- **Light Mode** - Bright theme for daytime use
- **Auto-Detection** - Follows system preference
- **Persistent** - Theme choice saved in browser

### Languages
- **English** - Default language
- **German (Deutsch)** - Full German translation
- **Language Switcher** - EN/DE toggle buttons
- **Persistent** - Language choice saved in browser

### Responsive Design
- **Mobile-Friendly** - Works on phones and tablets
- **Desktop-Optimized** - Clean layout on large screens

---

## Administration

### Share Management
- **Active Shares List** - View all current shares
- **File Information** - Name, size, download count
- **Quick Actions** - Copy link, show QR, delete
- **Password Indicator** - Lock icon for protected shares

### Optional Admin Authentication
- **Password Protection** - Protect upload/management functions
- **Session-Based** - Secure HttpOnly cookies
- **Rate-Limited Login** - 5 attempts per minute

---

## Technical Features

### API Endpoints
```
# Shares
POST   /api/upload           - Upload file
POST   /api/upload/multi     - Upload multiple files
POST   /api/share-from-path  - Share from server path
POST   /api/share-folder     - Share a folder
GET    /api/shares           - List all shares
GET    /api/shares/{id}      - Get share info
DELETE /api/shares/{id}      - Delete share

# Receive Links
GET    /api/receive-links           - List receive links
POST   /api/receive-links           - Create receive link
GET    /api/receive-links/{id}      - Get receive link
DELETE /api/receive-links/{id}      - Delete receive link
GET    /api/receive-links/{id}/files - Get received files

# Public Endpoints
GET    /s/{id}               - Share page (files and folders)
GET    /d/{id}               - Direct download
GET    /stream/{id}          - Media streaming
GET    /thumbnail/{id}       - Image thumbnail
GET    /folder/{id}/contents - Folder contents JSON
GET    /folder/{id}/download - Download file from folder
GET    /folder/{id}/zip      - Download folder as ZIP
GET    /r/{id}               - Receive upload page
POST   /r/{id}/upload        - Upload to receive link

# Config & Monitoring
GET    /api/tunnel       - Get tunnel config
POST   /api/tunnel       - Save tunnel/network config
GET    /api/network      - Get all network URLs
GET    /api/stats        - Statistics
GET    /metrics          - Prometheus metrics
```

### Docker Support
- **Multi-Stage Build** - Minimal image size (~11 MB compressed)
- **Docker Compose** - Easy deployment
- **Volume Persistence** - Data survives restarts
- **Health Checks** - Container health monitoring

### ZimaOS Integration
- **x-casaos Metadata** - Native ZimaOS app support
- **Cloudflared-Web Compatible** - Works with existing tunnels
- **Auto IP Detection** - Works with ZeroTier out of the box
- **start.sh Script** - Automatic network configuration

---

## File Structure

```
zima-share/
├── cmd/server/main.go        # Application entry point
├── internal/
│   ├── auth/                 # Password hashing (bcrypt)
│   ├── handlers/             # HTTP handlers (upload, folder, receive)
│   ├── metrics/              # Prometheus metrics
│   ├── middleware/           # Auth, rate-limit, security
│   ├── models/               # Data structures
│   ├── preview/              # Thumbnail generation
│   └── storage/              # SQLite database backend
├── web/
│   ├── static/
│   │   ├── css/style.css     # Dark/light themes
│   │   ├── js/app.js         # Frontend logic + i18n
│   │   └── img/              # Logo & icons
│   └── templates/            # HTML templates (share, folder, receive)
├── scripts/
│   └── tunnel-wrapper.sh     # Quick tunnel URL extraction
├── start.sh                  # Auto IP detection script
├── Dockerfile                # Main app container (CGO for SQLite)
├── Dockerfile.tunnel         # Tunnel container
├── docker-compose.yaml       # Deployment config
└── .env.example              # Configuration template
```

---

## Data Storage

```
/data/                        # Docker volume
├── shares.db                 # SQLite database (WAL mode)
├── shares.db-wal             # WAL file
├── shares.db-shm             # Shared memory file
├── uploads/                  # Uploaded files (UUID-named)
├── uploads/received/{id}/    # Received files per link
├── thumbnails/               # Generated thumbnails
├── admin_config.json         # Admin password (bcrypt)
├── sessions.json             # Active sessions
└── tunnel_config.json        # Network settings
```

### SQLite Tables

**shares** - File and folder shares
```sql
id, file_name, original_name, file_size, mime_type,
password_hash, has_password, expires_at, created_at,
downloads, max_downloads, source_path, is_symlink, is_directory
```

**folder_contents** - Files within folder shares
```sql
id, share_id, relative_path, file_name, file_size, mime_type, is_directory
```

**receive_links** - Upload links for others
```sql
id, name, password_hash, has_password, expires_at, created_at,
max_uploads, current_uploads, max_file_size, allowed_extensions,
auto_share, webhook_url, total_size
```

**received_files** - Files uploaded via receive links
```sql
id, receive_link_id, file_name, original_name, file_size,
mime_type, uploader_ip, uploader_agent, created_at, share_id
```

---

## Quick Start

```bash
# Local only
docker compose up -d

# With auto IP detection
./start.sh up -d

# With public tunnel
docker compose --profile tunnel up -d

# Custom port
WEBUI_PORT=9000 ./start.sh up -d
```

Access at: `http://localhost:8080`
