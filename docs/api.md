# API Reference

CasaDrop provides a REST API for all operations. Authentication is required for most endpoints.

## Authentication

When `ADMIN_PASSWORD` is set or setup is completed, requests to protected endpoints require a valid session cookie (`casadrop_session`).

### Check Auth Status

```bash
GET /api/auth/status
```

Response:
```json
{
  "authenticated": true,
  "setupRequired": false,
  "envPassword": false
}
```

### Login

```bash
POST /login
Content-Type: application/x-www-form-urlencoded

password=your-password
```

Returns session cookie on success.

## File Sharing

### Upload Single File

```bash
POST /api/upload
Content-Type: multipart/form-data

file: <binary>
password: optional-password
expires: 24h|7d|30d|never
maxDownloads: 0 (unlimited) or number
```

Response:
```json
{
  "id": "abc123",
  "filename": "document.pdf",
  "size": 1048576,
  "url": "https://share.example.com/s/abc123",
  "expiresAt": "2025-01-20T12:00:00Z"
}
```

### Upload Multiple Files

```bash
POST /api/upload/multi
Content-Type: multipart/form-data

files[]: <binary>
files[]: <binary>
password: optional-password
expires: 24h
```

### Share From Server Path

Share existing files without copying:

```bash
POST /api/share-from-path
Content-Type: application/json

{
  "path": "/media/movies/movie.mp4",
  "password": "",
  "expires": "7d",
  "maxDownloads": 0
}
```

### List Shares

```bash
GET /api/shares
```

Response:
```json
{
  "shares": [
    {
      "id": "abc123",
      "filename": "document.pdf",
      "size": 1048576,
      "downloads": 5,
      "maxDownloads": 10,
      "expiresAt": "2025-01-20T12:00:00Z",
      "createdAt": "2025-01-13T12:00:00Z",
      "hasPassword": true,
      "type": "file"
    }
  ]
}
```

### Get Share Info

```bash
GET /api/shares/{id}
```

### Delete Share

```bash
DELETE /api/shares/{id}
```

## Download

### Download File (Public)

```bash
GET /d/{id}
GET /d/{id}?password=xxx
```

### Stream File (Public)

For media streaming with range support:

```bash
GET /stream/{id}
GET /stream/{id}?password=xxx
```

### Share Page (Public)

```bash
GET /s/{id}
```

## Folder Sharing

### Share Folder

```bash
POST /api/share-folder
Content-Type: application/json

{
  "path": "/media/photos/vacation",
  "password": "",
  "expires": "7d"
}
```

### Get Folder Contents (Public)

```bash
GET /folder/{id}/contents?path=/subdir
GET /folder/{id}/contents?password=xxx
```

### Download Folder File (Public)

```bash
GET /folder/{id}/download?path=/photo.jpg
```

### Download Folder as ZIP (Public)

```bash
GET /folder/{id}/zip
GET /folder/{id}/zip?path=/subdir
```

## Receive Links

Receive links allow others to upload files to you.

### Create Receive Link

```bash
POST /api/receive-links
Content-Type: application/json

{
  "name": "Project Files",
  "password": "",
  "expires": "7d",
  "maxFiles": 10,
  "maxFileSize": 104857600
}
```

### List Receive Links

```bash
GET /api/receive-links
```

### Get Receive Link Details

```bash
GET /api/receive-links/{id}
```

### Delete Receive Link

```bash
DELETE /api/receive-links/{id}
```

### Get Received Files

```bash
GET /api/receive-links/{id}/files
```

### Download Received File

```bash
GET /api/receive-links/{id}/files/{fileId}
```

### Upload to Receive Link (Public)

```bash
POST /r/{id}/upload
Content-Type: multipart/form-data

file: <binary>
password: optional-password
```

## File Browser

### Browse Server Files

```bash
GET /api/browse?path=/media
```

Response:
```json
{
  "path": "/media",
  "entries": [
    {
      "name": "movies",
      "path": "/media/movies",
      "isDir": true,
      "size": 0,
      "modTime": "2025-01-10T10:00:00Z"
    },
    {
      "name": "photo.jpg",
      "path": "/media/photo.jpg",
      "isDir": false,
      "size": 2048576,
      "modTime": "2025-01-12T15:30:00Z"
    }
  ]
}
```

## Network & Configuration

### Get Network Info

```bash
GET /api/network
```

Response:
```json
{
  "tunnelURL": "https://xxx.trycloudflare.com",
  "localIP": "192.168.1.100",
  "zerotierIP": "10.147.17.50",
  "tailscaleURL": "https://casadrop.tailnet.ts.net",
  "pangolinURL": "",
  "customURL": "",
  "primaryNetwork": "tunnel",
  "maxFileSize": 10737418240
}
```

### Get/Set Tunnel Config

```bash
GET /api/tunnel
POST /api/tunnel
Content-Type: application/json

{
  "primaryNetwork": "tunnel",
  "maxFileSize": 10737418240,
  "customURL": ""
}
```

## Webhooks

### Get/Set Webhook Config

```bash
GET /api/webhook
POST /api/webhook
Content-Type: application/json

{
  "url": "https://n8n.example.com/webhook/xxx",
  "secret": "hmac-secret",
  "events": ["share.created", "share.downloaded", "share.expired"]
}
```

### Test Webhook

```bash
POST /api/webhook/test
```

## Statistics

### Get Stats

```bash
GET /api/stats
```

Response:
```json
{
  "totalShares": 42,
  "totalDownloads": 1337,
  "totalSize": 10737418240,
  "activeShares": 15
}
```

## Utilities

### Generate QR Code

```bash
GET /qr/{id}
```

Returns PNG image.

### Get Thumbnail

```bash
GET /thumbnail/{id}
```

Returns thumbnail for image/video files.

## Prometheus Metrics

```bash
GET /metrics
```

Available metrics:
- `casadrop_http_requests_total`
- `casadrop_http_request_duration_seconds`
- `casadrop_shares_total`
- `casadrop_downloads_total`
- `casadrop_storage_bytes`
