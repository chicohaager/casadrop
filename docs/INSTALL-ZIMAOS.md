# CasaDrop - ZimaOS Installation Guide

## Übersicht

Diese Anleitung beschreibt die Installation von CasaDrop auf ZimaOS, einschließlich:
- Entfernen einer alten Installation
- Neue Installation via Docker Compose
- Konfigurationsoptionen
- Integration mit Pangolin/Newt, Tailscale, ZeroTier

---

## 1. Alte Installation entfernen

### Option A: Über ZimaOS UI

1. Öffne **ZimaOS Dashboard** → **Apps**
2. Finde **CasaDrop** (oder **Zima-Share**)
3. Klicke auf **⋮** (Drei Punkte) → **Remove**
4. Wähle **"Delete app data"** wenn gewünscht

### Option B: Über SSH/Terminal

```bash
# Container stoppen und entfernen
docker stop casadrop
docker rm casadrop

# Alternativ: Container mit altem Namen
docker stop zima-share
docker rm zima-share

# Optional: Altes Image entfernen
docker rmi ghcr.io/user/casadrop:latest

# Optional: Daten löschen (VORSICHT: Löscht alle Shares!)
# sudo rm -rf /DATA/AppData/casadrop

# Alternativ: Nur Konfiguration behalten, Uploads löschen
# sudo rm -rf /DATA/AppData/casadrop/uploads/*
```

### Option C: Alles bereinigen (Kompletter Neustart)

```bash
# Alle CasaDrop Container finden und stoppen
docker ps -a | grep -E "casadrop|zima-share" | awk '{print $1}' | xargs -r docker stop
docker ps -a | grep -E "casadrop|zima-share" | awk '{print $1}' | xargs -r docker rm

# Alle Images entfernen
docker images | grep -E "casadrop|zima-share" | awk '{print $3}' | xargs -r docker rmi

# Ungenutzte Volumes bereinigen (optional)
docker volume prune -f

# AppData löschen (VORSICHT!)
sudo rm -rf /DATA/AppData/casadrop
sudo rm -rf /DATA/AppData/zima-share
```

---

## 2. Neue Installation

### Verzeichnisse erstellen

```bash
# Erstelle Konfigurationsverzeichnis
mkdir -p /DATA/AppData/casadrop
cd /DATA/AppData/casadrop
```

### docker-compose.yml erstellen

```bash
cat > docker-compose.yml << 'EOF'
version: "3.8"

services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      # Daten-Verzeichnis (Datenbank, Konfiguration, Uploads)
      - ./data:/data
      
      # Medien-Verzeichnisse für Datei-Browser (read-only empfohlen)
      - /DATA:/DATA:ro
      - /media:/media:ro
      
    environment:
      # ============= ALLGEMEIN =============
      - TZ=Europe/Berlin
      - PORT=8080
      - EXTERNAL_PORT=8787
      
      # ============= SICHERHEIT =============
      # Admin-Passwort (ÄNDERN!)
      - ADMIN_PASSWORD=dein-sicheres-passwort
      
      # Erlaubte Pfade für Datei-Browser
      - SHARE_ALLOWED_PATHS=/DATA,/media
      
      # ============= NETZWERK (Optional) =============
      # Manuell setzen falls Auto-Erkennung nicht funktioniert
      # - LOCAL_IP=192.168.1.100
      # - ZEROTIER_IP=10.147.20.50
      # - TAILSCALE_URL=https://zima.tail1234.ts.net
      # - PANGOLIN_URL=https://share.example.com
      
    # Für ZeroTier/Tailscale Auto-Erkennung:
    network_mode: host
    # ODER mit Port-Mapping (dann manuelle IPs setzen):
    # networks:
    #   - default

EOF
```

### Installation starten

```bash
cd /DATA/AppData/casadrop
docker compose pull
docker compose up -d
```

### Installation überprüfen

```bash
# Logs anzeigen
docker logs casadrop

# Status prüfen
docker ps | grep casadrop

# Web-UI öffnen
echo "CasaDrop läuft auf: http://$(hostname -I | awk '{print $1}'):8787"
```

---

## 3. Konfigurationsvarianten

### Variante A: Minimal (Lokales Netzwerk)

```yaml
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Berlin
      - ADMIN_PASSWORD=mein-passwort
```

### Variante B: Mit Pangolin/Newt

```yaml
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      - ./data:/data
      - /DATA:/DATA:ro
    environment:
      - TZ=Europe/Berlin
      - ADMIN_PASSWORD=mein-passwort
      - EXTERNAL_PORT=8787
      - PANGOLIN_URL=https://share.meine-domain.de
      - SHARE_ALLOWED_PATHS=/DATA,/media
```

### Variante C: Mit OIDC (Authentik/Keycloak)

```yaml
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Berlin
      
      # OIDC Konfiguration
      - OIDC_ENABLED=true
      - OIDC_ISSUER_URL=https://auth.example.com/application/o/casadrop/
      - OIDC_CLIENT_ID=casadrop
      - OIDC_CLIENT_SECRET=geheimer-schluessel
      - OIDC_REDIRECT_URL=https://share.example.com/auth/oidc/callback
      - OIDC_DEFAULT_ROLE=user
      
      # Optional: Lokale Anmeldung deaktivieren
      # - OIDC_DISABLE_LOCAL_AUTH=true
```

### Variante D: Mit Cloudflare Tunnel

```yaml
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    ports:
      - "8787:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=Europe/Berlin
      - ADMIN_PASSWORD=mein-passwort
      - TUNNEL_URL=https://share.meine-domain.de

  # Cloudflare Tunnel Container
  tunnel:
    image: cloudflare/cloudflared:latest
    container_name: casadrop-tunnel
    restart: unless-stopped
    command: tunnel run
    environment:
      - TUNNEL_TOKEN=dein-tunnel-token
    depends_on:
      - casadrop
```

### Variante E: Vollständig mit Host-Netzwerk

```yaml
services:
  casadrop:
    image: ghcr.io/user/casadrop:latest
    container_name: casadrop
    restart: unless-stopped
    network_mode: host
    volumes:
      - ./data:/data
      - /DATA:/DATA:ro
      - /media:/media:ro
    environment:
      - TZ=Europe/Berlin
      - PORT=8787
      - EXTERNAL_PORT=8787
      - ADMIN_PASSWORD=mein-passwort
      - SHARE_ALLOWED_PATHS=/DATA,/media,/home
```

---

## 4. Umgebungsvariablen Referenz

### Allgemein

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `PORT` | 8080 | Interner Port |
| `EXTERNAL_PORT` | 8080 | Externer Port (für Share-URLs) |
| `DATA_DIR` | /data | Datenverzeichnis |
| `ADMIN_PASSWORD` | - | Admin-Passwort |
| `TZ` | UTC | Zeitzone |

### Datei-Browser

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `SHARE_ALLOWED_PATHS` | /DATA,/media,/home | Erlaubte Pfade (kommagetrennt) |

### Netzwerk

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `LOCAL_IP` | auto | Lokale IP-Adresse |
| `ZEROTIER_IP` | auto | ZeroTier IP |
| `TAILSCALE_URL` | auto | Tailscale Funnel URL |
| `PANGOLIN_URL` | - | Pangolin/Newt öffentliche URL |
| `TUNNEL_URL` | - | Cloudflare Tunnel URL |
| `CUSTOM_URL` | - | Eigene URL (WireGuard, etc.) |

### OIDC/SSO

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `OIDC_ENABLED` | false | OIDC aktivieren |
| `OIDC_ISSUER_URL` | - | OIDC Provider URL |
| `OIDC_CLIENT_ID` | - | Client ID |
| `OIDC_CLIENT_SECRET` | - | Client Secret |
| `OIDC_REDIRECT_URL` | - | Callback URL |
| `OIDC_DEFAULT_ROLE` | user | Standard-Rolle (admin/user/viewer) |

### E-Mail

| Variable | Standard | Beschreibung |
|----------|----------|--------------|
| `SMTP_HOST` | - | SMTP Server |
| `SMTP_PORT` | 587 | SMTP Port |
| `SMTP_USER` | - | SMTP Benutzer |
| `SMTP_PASSWORD` | - | SMTP Passwort |
| `SMTP_FROM` | - | Absender E-Mail |

---

## 5. Nach der Installation

### Erster Login

1. Öffne `http://[ZIMA-IP]:8787`
2. Melde dich mit `admin` / `[ADMIN_PASSWORD]` an
3. Gehe zu **Settings** um Netzwerk-Einstellungen zu konfigurieren

### Netzwerk konfigurieren

1. Öffne **Settings** → **Network**
2. Wähle **Primary Network** (für Share-URLs):
   - `Cloudflare` - Für Cloudflare Tunnel
   - `Pangolin` - Für Pangolin/Newt
   - `Tailscale` - Für Tailscale Funnel
   - `ZeroTier` - Für ZeroTier
   - `Local` - Für lokales Netzwerk
3. Deaktiviere nicht genutzte Netzwerke

### Receive Links einrichten

1. Gehe zu **Receive Links**
2. Klicke **Create Receive Link**
3. Optional: Zielverzeichnis setzen (z.B. `/DATA/Uploads`)
4. Teile den Link - andere können Dateien an dich senden

---

## 6. Backup & Restore

### Backup erstellen

```bash
cd /DATA/AppData/casadrop

# Container stoppen
docker compose stop

# Backup erstellen
tar -czvf casadrop-backup-$(date +%Y%m%d).tar.gz data/

# Container starten
docker compose start
```

### Backup wiederherstellen

```bash
cd /DATA/AppData/casadrop

# Container stoppen
docker compose stop

# Altes Datenverzeichnis sichern
mv data data.old

# Backup entpacken
tar -xzvf casadrop-backup-YYYYMMDD.tar.gz

# Container starten
docker compose start
```

---

## 7. Update

```bash
cd /DATA/AppData/casadrop

# Neues Image ziehen
docker compose pull

# Container neu starten
docker compose up -d

# Alte Images aufräumen
docker image prune -f
```

---

## 8. Troubleshooting

### Container startet nicht

```bash
# Logs prüfen
docker logs casadrop

# Berechtigungen prüfen
ls -la /DATA/AppData/casadrop/data/
```

### Web-UI nicht erreichbar

```bash
# Port prüfen
netstat -tlnp | grep 8787

# Firewall prüfen
sudo ufw status
```

### Netzwerke werden nicht erkannt

Bei Nutzung von `network_mode: host` sollte Auto-Erkennung funktionieren.
Sonst manuell setzen:

```yaml
environment:
  - LOCAL_IP=192.168.1.100
  - ZEROTIER_IP=10.147.20.50
```

### Datenbank-Fehler

```bash
# SQLite reparieren
docker exec casadrop sqlite3 /data/casadrop.db "PRAGMA integrity_check;"

# Oder komplett neu (LÖSCHT ALLE DATEN!)
docker compose stop
rm /DATA/AppData/casadrop/data/casadrop.db
docker compose start
```

---

## 9. Deinstallation

### Nur Container entfernen (Daten behalten)

```bash
cd /DATA/AppData/casadrop
docker compose down
```

### Vollständige Deinstallation

```bash
cd /DATA/AppData/casadrop
docker compose down -v --rmi all
cd ..
rm -rf casadrop
```

---

## Support

- **GitHub Issues**: [github.com/user/casadrop/issues](https://github.com/user/casadrop/issues)
- **IceWhale Community**: [community.zimaspace.com](https://community.zimaspace.com)
