# CasaDrop 2.3.0 — Installation (Deutsch)

Sicheres, selbst-gehostetes File-Sharing für das Homelab.
Dieses Paket enthält alles für die Installation auf **ZimaOS / CasaOS** (Ein-Klick)
oder auf **jedem Linux-Server mit Docker**.

## Paket-Inhalt

| Datei | Zweck |
|-------|-------|
| `casadrop-zimaos-app.yaml` | App-Definition für den ZimaOS/CasaOS App-Store **und** als Docker-Compose-Datei nutzbar |
| `casadrop-2.3.0-amd64.tar.gz` | Offline-Image (nur nötig, wenn kein Internet/Docker-Hub verfügbar) |
| `.env.example` | Vorlage für optionale Einstellungen |
| `INSTALL_DE.md` / `INSTALL_EN.md` | Diese Anleitung |
| `NOTICE.txt` | Urheber-/Lizenzhinweis |
| `SHA256SUMS` | Prüfsummen zur Integritätskontrolle |

**Architektur:** amd64 (x86-64). **Standard-Port:** 8080.

---

## Variante A — ZimaOS / CasaOS (Ein-Klick, empfohlen)

1. Im ZimaOS-Dashboard: **App Store → Eigene App installieren / Import**.
2. Die Datei `casadrop-zimaos-app.yaml` hochladen bzw. ihren Inhalt einfügen.
3. Optional den Port (`WEBUI_PORT`) anpassen, dann **Installieren**.
   Das Image wird automatisch von Docker Hub (`chicohaager/casadrop:2.3.0`) geladen.
4. Nach dem Start die App-Kachel öffnen → der Setup-Assistent erscheint.

---

## Variante B — Beliebiger Linux-Server mit Docker

```bash
# 1) (nur offline) Image lokal laden – sonst überspringen, es wird gepullt:
gunzip -c casadrop-2.3.0-amd64.tar.gz | docker load

# 2) Starten:
docker compose -f casadrop-zimaos-app.yaml up -d

# 3) Anderer Port (optional):
WEBUI_PORT=8085 docker compose -f casadrop-zimaos-app.yaml up -d
```

Aufruf danach: `http://<server-ip>:8080` (bzw. der gewählte Port).

---

## Erster Start: Admin-Passwort setzen

Beim ersten Aufruf führt ein **Setup-Assistent** durch die Vergabe des
Admin-Passworts. Aus Sicherheitsgründen verlangt er einen **einmaligen
Setup-Token**, der nur im Container-Log steht:

```bash
docker logs casadrop 2>&1 | grep -i "setup token"
```

Alternativ den Assistenten überspringen, indem vor dem Start ein Passwort
gesetzt wird — in `.env`:

```
ADMIN_PASSWORD=EinSicheresPasswort
```

---

## Optionale Einstellungen (`.env`)

`.env.example` nach `.env` kopieren und anpassen. Wichtigste Schalter:

| Variable | Standard | Bedeutung |
|----------|----------|-----------|
| `WEBUI_PORT` | 8080 | Host-Port der WebUI |
| `TZ` | Europe/Berlin | Zeitzone |
| `ADMIN_PASSWORD` | – | Admin-Passwort (überspringt den Assistenten) |
| `TRUSTED_PROXY` | – | IP/CIDR des Reverse-Proxys (für echte Client-IP hinter Proxy) |
| `SHARE_ALLOWED_PATHS` | /DATA,/media | Wurzeln des Datei-Browsers (nur Admin) |
| `OIDC_ENABLED` | false | SSO/OIDC aktivieren |

> **Reverse-Proxy:** Läuft CasaDrop hinter nginx/Traefik/Caddy, unbedingt
> `TRUSTED_PROXY` auf die Proxy-IP setzen — sonst werden `X-Forwarded-For`-
> Header (fail-closed) ignoriert und die Rate-Limits sehen nur die Proxy-IP.

---

## Datenablage & Backup

Alle Nutzdaten (SQLite-DB, Uploads, Thumbnails) liegen im Volume
`/data` im Container. Auf ZimaOS gebunden an `/DATA/AppData/casadrop/data`.
Für ein Backup einfach dieses Verzeichnis sichern.

## Update

```bash
docker compose -f casadrop-zimaos-app.yaml pull
docker compose -f casadrop-zimaos-app.yaml up -d
```

## Integrität prüfen

```bash
sha256sum -c SHA256SUMS
```

## Support

Quellcode & Issues: https://github.com/chicohaager/casadrop
