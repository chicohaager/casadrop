"""Stichprobe 2.5.2 in mehreren Sprachen: Vorschauseite (Datum), Empfangsseite (Ablehnung mit Grund), Login.
Aufruf: python3 tests/browser_252_languages.py <base-url> <pwdatei> <ausgabeordner> — Passwort wird nie ausgegeben."""
import json
import re
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright

BASE, pw, OUT = sys.argv[1], Path(sys.argv[2]).read_text().strip(), Path(sys.argv[3])
OUT.mkdir(parents=True, exist_ok=True)
DEMO = Path(__file__).resolve().parents[2] / "video-studio/videos/casadrop/capture/demo/Summer-Party-2026.jpg"
LOC = Path(__file__).resolve().parents[1] / "internal/i18n/locales"
SPRACHEN = {"fr": "fr-FR", "ja": "ja-JP", "ru": "ru-RU", "ar": "ar-EG"}
erg = []

with sync_playwright() as p:
    b = p.chromium.launch()
    admin = b.new_context(locale="en-US")
    pg = admin.new_page()
    pg.goto(BASE + "/login")
    pg.locator("input[type=password]").first.fill(pw)
    pg.keyboard.press("Enter")
    pg.wait_for_load_state("networkidle")
    # ungeschützte Freigabe (Datum sichtbar) und Empfangslink nur für .pdf
    pg.locator("#file-input").set_input_files(str(DEMO))
    pg.locator("#upload-options").wait_for(state="visible")
    pg.locator("#upload-password").fill("")
    pg.locator("#upload-expiry").select_option("168")
    pg.locator("#upload-btn").click()
    pg.locator("#upload-results").wait_for(state="visible", timeout=30000)
    share = re.search(r"(/s/[A-Za-z0-9]+)", pg.locator("#upload-results").inner_text()).group(1)
    pg.locator("[data-view=receive]").click()
    pg.locator("#receive-form-wrapper").evaluate("e => e.style.display = 'block'")
    pg.locator("#receive-name").fill("Probe")
    pg.locator("#receive-extensions").fill(".pdf")
    pg.locator("#receive-form button[type=submit]").click()
    pg.wait_for_timeout(1200)
    links = pg.evaluate("fetch('/api/receive-links').then(r => r.json())")
    links = links if isinstance(links, list) else (links.get("links") or links.get("data") or [])
    recv = "/r/" + links[0]["id"]
    admin.close()

    for lang, loc in SPRACHEN.items():
        T = json.loads((LOC / f"{lang}.json").read_text())
        c = b.new_context(locale=loc, extra_http_headers={"Accept-Language": loc + "," + lang + ";q=0.9"})
        g = c.new_page()
        g.goto(BASE + share)
        txt = g.locator("body").inner_text()
        dl = T["common.download"]
        erg.append((lang, "Vorschau: html lang", g.locator("html").get_attribute("lang") == lang))
        erg.append((lang, "Vorschau: Download-Knopf", dl.lower() in txt.lower()))
        datum_teil = T["common.expires"].split("{date}")[0].strip()
        erg.append((lang, "Vorschau: Ablauftext", (datum_teil.lower() in txt.lower()) if datum_teil else True))
        erg.append((lang, "Vorschau: kein {Platzhalter}", "{" not in txt))
        g.screenshot(path=str(OUT / f"vorschau-{lang}.png"))
        g.goto(BASE + recv)
        g.locator("input[type=file]").first.set_input_files(str(DEMO))
        g.wait_for_timeout(400)
        g.locator("#btn-upload").click()
        g.wait_for_timeout(2000)
        txt = g.locator("body").inner_text()
        grund = T["receive.err.fileType"].replace("{ext}", ".jpg")
        erg.append((lang, "Empfang: Ablehnung mit Grund", grund in txt))
        g.screenshot(path=str(OUT / f"empfang-{lang}.png"))
        g.goto(BASE + "/login")
        txt = g.locator("body").inner_text()
        erg.append((lang, "Login: Untertitel", T["login.subtitle"].lower() in txt.lower()))
        c.close()
    b.close()

zeilen = [f"{'OK ' if ok else 'FEHLER '}{lang} {name}" for lang, name, ok in erg]
(OUT / "sprachen.txt").write_text("\n".join(zeilen) + "\n")
print("\n".join(zeilen))
