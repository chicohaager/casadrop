"""Browser-Prüfung für 2.5.2 gegen eine laufende Instanz: Logo, deutsche Umlaute, Copy/Protected, Empfängerseiten de/en.
Aufruf: python3 tests/browser_252_check.py <base-url> <pwdatei> <ausgabeordner>  — Passwort wird nie ausgegeben."""
import re
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright

BASE, pw, OUT = sys.argv[1], Path(sys.argv[2]).read_text().strip(), Path(sys.argv[3])
OUT.mkdir(parents=True, exist_ok=True)
DEMO = Path(__file__).resolve().parents[2] / "video-studio/videos/casadrop/capture/demo/Summer-Party-2026.jpg"
befunde, konsole = [], []


def pruefe(name, bedingung, detail=""):
    befunde.append(("OK " if bedingung else "FEHLER ") + name + (f" — {detail}" if detail and not bedingung else ""))


with sync_playwright() as p:
    b = p.chromium.launch()

    def kontext(locale, lang=None):
        c = b.new_context(viewport={"width": 1440, "height": 900}, locale=locale)
        if lang:
            c.add_init_script(f"localStorage.setItem('casadrop_lang','{lang}')")
        return c

    # --- Admin-Oberfläche auf Deutsch ---
    ctx = kontext("de-DE", "de")
    pg = ctx.new_page()
    pg.on("console", lambda m: konsole.append(f"admin {m.type}: {m.text}") if m.type in ("error", "warning") else None)
    pg.on("pageerror", lambda e: konsole.append(f"admin pageerror: {e}"))
    pg.goto(BASE + "/login")
    pg.screenshot(path=str(OUT / "01-login-de.png"))
    pruefe("Login zeigt neues Icon", pg.locator('img[src="/static/icon.png"]').count() >= 1)
    login = pg.locator("body").inner_text()
    pruefe("Login auf Deutsch", "admin-anmeldung" in login.lower() and "passwort" in login.lower() and "admin login" not in login.lower(), login[:120].replace("\n", " | "))
    pg.locator("input[type=password]").first.fill(pw)
    pg.keyboard.press("Enter")
    pg.wait_for_load_state("networkidle")
    pruefe("Seitenleiste: neues Icon", pg.locator(".sidebar-brand img.brand-icon[src='/static/icon.png']").count() == 1)
    pruefe("Seitenleiste: Name CasaDrop", pg.locator(".sidebar-brand .brand-name").inner_text().strip() == "CasaDrop")
    pruefe("Favicon = icon.png", pg.locator('link[rel=icon]').get_attribute("href") == "/static/icon.png")
    hinweis = pg.locator("[data-i18n='upload.hint']").first.inner_text()
    pruefe("Umlaut 'möglich'", "möglich" in hinweis, hinweis)
    pg.screenshot(path=str(OUT / "02-start-de.png"))

    pg.locator("#file-input").set_input_files(str(DEMO))
    pg.locator("#upload-options").wait_for(state="visible")
    pg.locator("#upload-password").fill("Sommer2026")
    pg.locator("#upload-expiry").select_option("1")          # 1 Stunde -> Abzeichen „Läuft bald ab“
    pg.locator("#upload-btn").click()
    pg.locator("#upload-results").wait_for(state="visible", timeout=30000)
    pg.wait_for_timeout(800)
    ergebnis = pg.locator("#upload-results").inner_text()
    pruefe("Ergebnis-Knopf 'Kopieren'", "Kopieren" in ergebnis and "Copy" not in ergebnis, ergebnis.replace("\n", " | ")[:120])
    pg.screenshot(path=str(OUT / "03-ergebnis-de.png"))
    m = re.search(r"https?://\S+?(/s/[A-Za-z0-9]+)", ergebnis)
    share_pfad = m.group(1) if m else None

    pg.locator("[data-view=shares]").click()
    pg.wait_for_timeout(1000)
    liste = pg.locator("#shares-view").inner_text()
    pruefe("Abzeichen 'Geschützt'", "geschützt" in liste.lower() and "protected" not in liste.lower())   # CSS setzt Großbuchstaben
    pruefe("Umlaut 'läuft ab'", "läuft ab" in liste.lower() and "laeuft" not in liste.lower(), liste[:160].replace("\n", " | "))
    titel = pg.locator("#shares-view .btn-icon[data-copy]").first.get_attribute("title")
    pruefe("Tooltip 'Link kopieren'", titel == "Link kopieren", titel)
    pruefe("Abzeichen 'Läuft bald ab'", "läuft bald ab" in liste.lower() and "expiring" not in liste.lower())
    loeschen = pg.locator("#shares-view .btn-icon.danger").first.get_attribute("title")
    pruefe("Tooltip 'Löschen'", loeschen == "Löschen", loeschen)
    aktual = pg.locator("#refresh-shares").get_attribute("title")
    pruefe("Tooltip 'Aktualisieren'", aktual == "Aktualisieren", aktual)
    pg.screenshot(path=str(OUT / "04-freigaben-de.png"))

    pg.locator("[data-view=receive]").click()
    pg.get_by_role("button", name="Neuer Link").click()
    pg.locator("#receive-name").fill("Fotos vom Sommerfest")
    pg.locator("#receive-extensions").fill(".pdf")            # nur PDF -> ein JPG muss mit Grund abgelehnt werden
    pg.get_by_role("button", name="Erstellen", exact=True).click()
    pg.wait_for_timeout(1200)
    links = pg.evaluate("fetch('/api/receive-links').then(r => r.json())")
    eintraege = links if isinstance(links, list) else (links.get("links") or links.get("data") or [])
    recv_pfad = "/r/" + eintraege[0]["id"] if eintraege else None

    # Dunkelmodus der Seitenleiste
    pg.evaluate("document.documentElement.setAttribute('data-theme','dark')")
    pg.wait_for_timeout(300)
    pg.locator(".sidebar-brand").screenshot(path=str(OUT / "05-brand-dunkel.png"))
    ctx.close()

    # --- Empfängerseiten: Deutsch und Englisch per Browser-Sprache ---
    for loc, lang, erwartet, fremd in [("de-DE", "de", ["Passwortgeschützt", "Entsperren"], ["Password Protected", "Unlock"]),
                                       ("en-US", "en", ["Password Protected", "Unlock"], ["Passwortgeschützt", "Entsperren"])]:
        c = kontext(loc)
        g = c.new_page()
        g.on("console", lambda m, lang=lang: konsole.append(f"gast-{lang} {m.type}: {m.text}") if m.type in ("error", "warning") else None)
        g.on("pageerror", lambda e, lang=lang: konsole.append(f"gast-{lang} pageerror: {e}"))
        g.goto(BASE + share_pfad)
        text = g.locator("body").inner_text()
        pruefe(f"Share-Seite {lang}: Sprache", all(e in text for e in erwartet) and not any(f in text for f in fremd), text[:120].replace("\n", " | "))
        pruefe(f"Share-Seite {lang}: html lang", g.locator("html").get_attribute("lang") == lang)
        pruefe(f"Share-Seite {lang}: neues Icon", g.locator('img[src="/static/icon.png"]').count() == 1)
        g.screenshot(path=str(OUT / f"06-share-passwort-{lang}.png"))
        g.locator("input[type=password]").first.fill("Sommer2026")
        g.keyboard.press("Enter")
        g.wait_for_timeout(1500)
        dl = g.locator("#download-btn").inner_text().strip()
        pruefe(f"Share-Seite {lang}: Download-Knopf", dl == ("Herunterladen" if lang == "de" else "Download"), dl)
        g.screenshot(path=str(OUT / f"07-share-vorschau-{lang}.png"))

        g.goto(BASE + recv_pfad)
        text = g.locator("body").inner_text()
        pruefe(f"Empfangsseite {lang}", ("Datei hierher ziehen" in text) if lang == "de" else ("Drag & drop a file here" in text), text[:150].replace("\n", " | "))
        g.locator("input[type=file]").first.set_input_files(str(DEMO))
        g.wait_for_timeout(500)
        g.locator("#btn-upload").click()
        g.wait_for_timeout(2500)
        text = g.locator("body").inner_text()
        pruefe(f"Empfangsseite {lang}: Ablehnung mit Grund", ("Dateityp .jpg ist nicht erlaubt" in text) if lang == "de"
               else ("File type .jpg not allowed" in text), text[-200:].replace("\n", " | "))
        g.screenshot(path=str(OUT / f"08-empfang-fertig-{lang}.png"))

        g.goto(BASE + "/s/gibtsnicht")
        text = g.locator("body").inner_text()
        pruefe(f"Nicht-gefunden {lang}", ("Freigabe nicht gefunden" in text) if lang == "de" else ("Share Not Found" in text))
        c.close()
    b.close()

(OUT / "befunde.txt").write_text("\n".join(befunde + ["", "Konsole:"] + (konsole or ["(keine Fehler/Warnungen)"])) + "\n")
print("\n".join(befunde))
print("Konsole:", len(konsole), "Einträge")
