"""Browser-Prüfung: Empfängerseite einer Ordnerfreigabe zeigt die Dateien (Fehler bis 2.5.2-dev:
„Failed to load folder contents.“ — folder.js erwartete eine Liste, /folder/{id}/contents liefert {entries: [...]}).

Erzeugt über die API zwei Ordnerfreigaben (ohne und mit Passwort) aus einem Host-Ordner und prüft als Gast:
Dateiliste sichtbar, kein Ladefehler, Unterordner öffnen und zurück, Datei-Download, Passwort-Tor.

Aufruf: python3 tests/browser_folder_share_check.py <base-url> <pwdatei> <host-pfad im Container> <ausgabeordner>
        z. B. … http://127.0.0.1:18088 /tmp/pw /DATA/Photos /tmp/out   — Passwort wird nie ausgegeben.
Erwartet im Host-Ordner mindestens eine Datei und einen Unterordner mit mindestens einer Datei. Exit 0 = alles OK."""
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright

BASE, pw, HOSTPFAD, OUT = sys.argv[1], Path(sys.argv[2]).read_text().strip(), sys.argv[3], Path(sys.argv[4])
OUT.mkdir(parents=True, exist_ok=True)
befunde = []


def pruefe(name, bedingung, detail=""):
    befunde.append((bool(bedingung), name + (f" — {detail}" if detail and not bedingung else "")))


with sync_playwright() as p:
    b = p.chromium.launch()
    admin = b.new_context().new_page()
    admin.goto(BASE + "/login")
    admin.locator("input[type=password]").first.fill(pw)
    admin.keyboard.press("Enter")
    admin.wait_for_load_state("networkidle")

    def freigabe(passwort):
        r = admin.evaluate("""async ([pfad, pw]) => {
            const r = await fetch('/api/share-folder', {method: 'POST', headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({path: pfad, password: pw, expires_in: 1, max_downloads: 0})});
            return {status: r.status, body: await r.text()};
        }""", [HOSTPFAD, passwort])
        pruefe(f"Ordnerfreigabe angelegt (Passwort={bool(passwort)})", r["status"] in (200, 201), f"HTTP {r['status']}: {r['body'][:160]}")
        import json, re
        try:
            d = json.loads(r["body"])
            return d.get("id") or (d.get("share") or {}).get("id") or re.search(r"/s/([A-Za-z0-9]+)", d.get("url", "")).group(1)
        except Exception:
            return None

    def gastseite(share_id, passwort, tag):
        g = b.new_context().new_page()
        fehler = []
        g.on("pageerror", lambda e: fehler.append(str(e)))
        g.goto(f"{BASE}/s/{share_id}")
        if passwort:
            pruefe(f"{tag}: Passwort-Tor sichtbar", g.locator("#password-gate").is_visible())
            g.locator("#folder-password").fill(passwort)
            g.locator("#folder-password-form button, #folder-password-form [type=submit]").first.click()
        g.wait_for_timeout(1500)
        g.screenshot(path=str(OUT / f"{tag}-wurzel.png"), full_page=True)
        pruefe(f"{tag}: kein Ladefehler", not g.locator("#fetch-error").is_visible(), g.locator("#fetch-error-text").inner_text())
        dateien = g.locator("#file-list-body a.file-row")
        ordner = g.locator("#file-list-body .dir-row")
        pruefe(f"{tag}: Dateien in der Wurzel sichtbar", dateien.count() >= 1, f"{dateien.count()} Dateizeilen")
        pruefe(f"{tag}: Unterordner sichtbar", ordner.count() >= 1, f"{ordner.count()} Ordnerzeilen")
        if dateien.count():
            href = dateien.first.get_attribute("href")
            st = g.evaluate("u => fetch(u).then(r => r.status)", href)
            pruefe(f"{tag}: Download der ersten Datei HTTP 200", st == 200, f"HTTP {st} für {href}")
        if ordner.count():
            name = ordner.first.locator(".file-row-label").inner_text()
            ordner.first.click()
            g.wait_for_timeout(1200)
            g.screenshot(path=str(OUT / f"{tag}-unterordner.png"), full_page=True)
            pruefe(f"{tag}: Unterordner zeigt Dateien", g.locator("#file-list-body a.file-row").count() >= 1 and not g.locator("#fetch-error").is_visible())
            pruefe(f"{tag}: Brotkrumen nennt Unterordner", name in g.locator("#breadcrumbs").inner_text())
            g.locator("#breadcrumbs [data-path='/']").first.click()
            g.wait_for_timeout(1000)
            pruefe(f"{tag}: zurück zur Wurzel", g.locator("#file-list-body .dir-row").count() >= 1)
        if passwort:      # Neuladen: gespeichertes Passwort (Cookie) öffnet die Liste ohne Eingabe
            g.reload()
            g.wait_for_timeout(1500)
            pruefe(f"{tag}: nach Neuladen Liste ohne erneute Eingabe", g.locator("#file-list-body a.file-row").count() >= 1
                   and g.locator("#file-list").is_visible(), f"{g.locator('#file-list-body a.file-row').count()} Dateizeilen")
        pruefe(f"{tag}: keine Skriptfehler", not fehler, "; ".join(fehler)[:200])
        g.context.close()

    offen = freigabe("")
    if offen:
        gastseite(offen, "", "offen")
    geschuetzt = freigabe("Ordner2026")
    if geschuetzt:
        gastseite(geschuetzt, "Ordner2026", "passwort")
    b.close()

for ok, text in befunde:
    print(("OK     " if ok else "FEHLER ") + text)
fehlt = sum(1 for ok, _ in befunde if not ok)
print(f"{len(befunde)} Prüfungen, {fehlt} Fehler")
sys.exit(1 if fehlt or not befunde else 0)
