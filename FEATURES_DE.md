# Zima-Share Funktionsübersicht

## Architektur-Diagramm

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ZIMA-SHARE                                     │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │   Benutzer   │
                              └──────┬───────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
              ▼                      ▼                      ▼
    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
    │ Lokales Netzwerk│    │    ZeroTier     │    │   Cloudflare    │
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
                    │    ┌──────────┴──────────┐    │
                    │    │                     │    │
                    │    ▼                     ▼    │
                    │ ┌──────────┐    ┌──────────┐  │
                    │ │ Handler  │    │ Speicher │  │
                    │ └──────────┘    └──────────┘  │
                    │                              │
                    └───────────────────────────────┘
                                    │
                                    ▼
                          ┌─────────────────┐
                          │  Docker Volume  │
                          │   /data         │
                          │  ├─ uploads/    │
                          │  ├─ shares.json │
                          │  └─ config.json │
                          └─────────────────┘
```

---

## Upload-Ablauf

```
┌────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│Benutzer│────▶│   Web UI    │────▶│ /api/upload  │────▶│  Speicher   │
│        │     │ (Drag&Drop) │     │              │     │             │
└────────┘     └─────────────┘     └──────────────┘     └─────────────┘
                                          │
                     ┌────────────────────┼────────────────────┐
                     │                    │                    │
                     ▼                    ▼                    ▼
              ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
              │  Dateityp   │     │  Passwort   │     │   Datei     │
              │  prüfen     │     │  hashen     │     │  speichern  │
              │ (Sicherheit)│     │  (bcrypt)   │     │ + Metadaten │
              └─────────────┘     └─────────────┘     └─────────────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │  Share-URL  │
                                   │  generieren │
                                   │  + QR-Code  │
                                   └─────────────┘
```

---

## Download-Ablauf

```
┌────────┐     ┌─────────────┐     ┌──────────────┐
│Benutzer│────▶│  /s/{id}    │────▶│   Freigabe   │
│        │     │ Share-Seite │     │  vorhanden?  │
└────────┘     └─────────────┘     └──────┬───────┘
                                          │
                          ┌───────────────┴───────────────┐
                          │                               │
                          ▼                               ▼
                   ┌─────────────┐                 ┌─────────────┐
                   │ Abgelaufen? │                 │  Nicht      │
                   │ Max. DL?    │                 │  gefunden   │
                   └──────┬──────┘                 └─────────────┘
                          │
              ┌───────────┴───────────┐
              │                       │
              ▼                       ▼
       ┌─────────────┐         ┌─────────────┐
       │  Passwort   │         │Kein Passwort│
       │  benötigt   │         │ Direkter DL │
       └──────┬──────┘         └─────────────┘
              │
              ▼
       ┌─────────────┐
       │  Passwort   │────▶ Erfolg ────▶ Download
       │  prüfen     │
       └─────────────┘────▶ Fehler ────▶ Erneut (Rate Limited)
```

---

## Netzwerk-Erkennung

```
┌─────────────────────────────────────────────────────────────────────┐
│                         start.sh                                    │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │  1. ZeroTier-IP erkennen (Interface: zt*)                   │   │
│   │  2. Lokale IP erkennen (erste nicht-Docker, nicht-Loopback) │   │
│   │  3. Als Umgebungsvariablen exportieren                      │   │
│   └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Docker Container                               │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │  Umgebungsvariablen:                                        │   │
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
│   │  Priorität für ZeroTier-IP:                                 │   │
│   │  1. Benutzer-Konfiguration (tunnel_config.json)             │   │
│   │  2. Umgebungsvariable (ZEROTIER_IP)                         │   │
│   │  3. Automatische Erkennung aus Netzwerk-Interfaces          │   │
│   └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Web UI                                     │
│   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐       │
│   │ Cloudflare URL  │ │  ZeroTier URL   │ │   Lokale URL    │       │
│   │ https://...     │ │ http://10.x:80  │ │ http://192.x:80 │       │
│   └─────────────────┘ └─────────────────┘ └─────────────────┘       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Kernfunktionen

### Datei-Sharing
- **Drag & Drop Upload** - Dateien einfach auf den Upload-Bereich ziehen
- **Große Dateien** - Upload bis zu 10 GB
- **QR-Code Generierung** - Automatischer QR-Code für einfaches mobiles Teilen
- **Ein-Klick Kopieren** - Share-Links sofort in die Zwischenablage kopieren

### Sicherheit
- **Passwortschutz** - Optionales Passwort für jede Freigabe
- **Passwort-Generator** - Eingebauter sicherer Passwort-Generator
- **Blockierte Dateitypen** - Ausführbare Dateien (.exe, .bat, .ps1, etc.) werden automatisch blockiert
- **bcrypt Hashing** - Sichere Passwortspeicherung
- **Rate-Limiting** - Schutz gegen Brute-Force-Angriffe
- **Security-Header** - CSP, X-Frame-Options, HSTS, X-Content-Type-Options

### Ablauf & Limits
- **Flexible Gültigkeit** - 1 Stunde, 6 Stunden, 24 Stunden, 3 Tage, 7 Tage oder 30 Tage
- **Download-Limits** - Maximale Anzahl Downloads festlegen (0 = unbegrenzt)
- **Auto-Bereinigung** - Abgelaufene Dateien werden automatisch entfernt (stündliche Prüfung)

---

## Netzwerk-Zugangsoptionen

### Drei Zugangswege

| Methode | URL-Format | Anwendungsfall |
|---------|------------|----------------|
| **Lokales Netzwerk** | `http://192.168.x.x:8080` | Gleiches WLAN/LAN |
| **ZeroTier VPN** | `http://10.147.x.x:8080` | Fernzugriff über VPN |
| **Cloudflare Tunnel** | `https://xxx.trycloudflare.com` | Öffentlicher Internetzugang |

### Automatische IP-Erkennung

```bash
# start.sh erkennt automatisch:
./start.sh up -d

# Ausgabe:
# Detected IPs:
#   ZeroTier: 10.147.19.1
#   Local:    192.168.1.100
```

### Netzwerk-Einstellungen UI
- **Zahnrad-Symbol** - Netzwerk-Einstellungen im Header öffnen
- **ZeroTier IP-Eingabe** - Manuelle Eingabe falls Auto-Erkennung fehlschlägt
- **Cloudflare Checkbox** - Externen Tunnel aktivieren
- **Persistente Konfiguration** - Einstellungen werden in `tunnel_config.json` gespeichert

---

## Cloudflare Tunnel Optionen

### Drei Tunnel-Optionen

| Option | Beschreibung | Anwendungsfall |
|--------|--------------|----------------|
| **Quick Tunnel** | Temporäre zufällige URL | Testen, schnelles Teilen |
| **Token Tunnel** | Permanente eigene Domain | Produktivbetrieb |
| **Externer Tunnel** | Vorhandenes Cloudflared-Web nutzen | ZimaOS Integration |

### Quick Tunnel (Ohne Konfiguration)
```bash
docker compose --profile tunnel up -d
# URL: https://zufaellige-worte.trycloudflare.com
```

### Token Tunnel (Permanente Domain)
```bash
CLOUDFLARE_TUNNEL_TOKEN=xxx docker compose --profile tunnel up -d
# URL: https://share.deinedomain.com
```

### Externer Tunnel (Cloudflared-Web)
1. Tunnel in Cloudflared-Web UI konfigurieren
2. Auf `http://zima-share:8080` verweisen
3. Checkbox in Zima-Share Einstellungen aktivieren
4. Öffentliche URL eingeben

---

## Benutzeroberfläche

### Themes
- **Dunkelmodus** - Standard dunkles Theme, augenschonend
- **Hellmodus** - Helles Theme für Tageslicht
- **Auto-Erkennung** - Folgt Systemeinstellung
- **Persistent** - Theme-Wahl wird im Browser gespeichert

### Sprachen
- **Englisch** - Standardsprache
- **Deutsch** - Vollständige deutsche Übersetzung
- **Sprachwechsler** - EN/DE Umschalt-Buttons
- **Persistent** - Sprachwahl wird im Browser gespeichert

### Responsives Design
- **Mobilfreundlich** - Funktioniert auf Handys und Tablets
- **Desktop-Optimiert** - Übersichtliches Layout auf großen Bildschirmen

---

## Administration

### Freigaben-Verwaltung
- **Aktive Freigaben Liste** - Alle aktuellen Freigaben anzeigen
- **Datei-Informationen** - Name, Größe, Download-Zähler
- **Schnellaktionen** - Link kopieren, QR anzeigen, löschen
- **Passwort-Indikator** - Schloss-Symbol für geschützte Freigaben

### Optionale Admin-Authentifizierung
- **Passwortschutz** - Upload/Verwaltungs-Funktionen schützen
- **Session-basiert** - Sichere HttpOnly Cookies
- **Rate-Limited Login** - 5 Versuche pro Minute

---

## Technische Features

### API-Endpunkte
```
POST   /api/upload       - Datei hochladen
GET    /api/shares       - Alle Freigaben auflisten
GET    /api/shares/{id}  - Freigabe-Info abrufen
DELETE /api/shares/{id}  - Freigabe löschen
GET    /api/tunnel       - Tunnel-Konfiguration abrufen
POST   /api/tunnel       - Tunnel/Netzwerk-Konfiguration speichern
GET    /api/network      - Alle Netzwerk-URLs abrufen
GET    /s/{id}           - Freigabe-Seite
GET    /d/{id}           - Direkter Download
```

### Docker-Unterstützung
- **Multi-Stage Build** - Minimale Image-Größe (~11 MB komprimiert)
- **Docker Compose** - Einfaches Deployment
- **Volume-Persistenz** - Daten überleben Neustarts
- **Health Checks** - Container-Gesundheitsüberwachung

### ZimaOS Integration
- **x-casaos Metadaten** - Native ZimaOS App-Unterstützung
- **Cloudflared-Web kompatibel** - Funktioniert mit vorhandenen Tunnels
- **Auto IP-Erkennung** - Funktioniert mit ZeroTier out-of-the-box
- **start.sh Skript** - Automatische Netzwerk-Konfiguration

---

## Dateistruktur

```
zima-share/
├── cmd/server/main.go        # Anwendungs-Einstiegspunkt
├── internal/
│   ├── auth/                 # Passwort-Hashing (bcrypt)
│   ├── handlers/             # HTTP-Handler + Netzwerk-Konfig
│   ├── middleware/           # Auth, Rate-Limit, Sicherheit
│   ├── models/               # Datenstrukturen
│   └── storage/              # JSON-Datei-Persistenz
├── web/
│   ├── static/
│   │   ├── css/style.css     # Dunkel/Hell Themes
│   │   ├── js/app.js         # Frontend-Logik + i18n
│   │   └── img/              # Logo & Icons
│   └── templates/            # HTML-Templates
├── scripts/
│   └── tunnel-wrapper.sh     # Quick Tunnel URL-Extraktion
├── start.sh                  # Auto IP-Erkennungs-Skript
├── Dockerfile                # Haupt-App Container
├── Dockerfile.tunnel         # Tunnel Container
├── docker-compose.yaml       # Deployment-Konfiguration
└── .env.example              # Konfigurations-Vorlage
```

---

## Datenspeicherung

```
/data/                        # Docker Volume
├── uploads/                  # Hochgeladene Dateien (UUID-benannt)
├── shares.json               # Freigabe-Metadaten
└── tunnel_config.json        # Netzwerk-Einstellungen
```

### shares.json Struktur
```json
{
  "abc12345": {
    "id": "abc12345",
    "file_name": "abc12345.pdf",
    "original_name": "dokument.pdf",
    "file_size": 1048576,
    "mime_type": "application/pdf",
    "password": "$2a$10$...",
    "has_password": true,
    "expires_at": "2024-12-06T12:00:00Z",
    "created_at": "2024-12-05T12:00:00Z",
    "downloads": 5,
    "max_downloads": 10
  }
}
```

---

## Schnellstart

```bash
# Nur lokal
docker compose up -d

# Mit Auto IP-Erkennung
./start.sh up -d

# Mit öffentlichem Tunnel
docker compose --profile tunnel up -d

# Eigener Port
WEBUI_PORT=9000 ./start.sh up -d
```

Zugriff unter: `http://localhost:8080`
