"""2.5.2: Admin-Oberfläche in allen 14 Sprachen — jede Ansicht öffnen und sichtbare
Texte, Platzhalter, title- und aria-label-Attribute gegen I18N prüfen.
Befund = ein Text, der dem englischen Wert entspricht, obwohl die Sprache einen eigenen hat
(stiller Rückfall), ein roher Schlüssel oder ein ungefüllter {Platzhalter}.
Aufruf: python3 tests/browser_252_admin_languages.py <base-url> <pwdatei> <ausgabeordner>
Das Passwort wird nie ausgegeben."""
import json
import re
import subprocess
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright

BASE, pw, OUT = sys.argv[1], Path(sys.argv[2]).read_text().strip(), Path(sys.argv[3])
OUT.mkdir(parents=True, exist_ok=True)
ROOT = Path(__file__).resolve().parents[1]
I18N = json.loads(subprocess.run(["node", "-e", r"""
const src=require('fs').readFileSync(process.argv[1],'utf8');
const s=src.lastIndexOf('{', src.indexOf('\n        en: {'));let d=0,i=s;
for(;i<src.length;i++){if(src[i]=='{')d++;else if(src[i]=='}'){d--;if(!d)break;}}
process.stdout.write(JSON.stringify(eval('('+src.slice(s,i+1)+')')));""", str(ROOT / "web/static/js/app.js")],
    capture_output=True, text=True, check=True).stdout)
LOCALES = {"en": "en-US", "de": "de-DE", "fr": "fr-FR", "es": "es-ES", "it": "it-IT", "pt": "pt-BR", "nl": "nl-NL",
           "pl": "pl-PL", "ru": "ru-RU", "ja": "ja-JP", "zh": "zh-CN", "ko": "ko-KR", "tr": "tr-TR", "ar": "ar-EG"}
# Texte, die nicht aus I18N kommen und darum vom Rückfall-Detektor nicht erkannt
# würden: Protokolltexte des Servers und die alte de/en-Zeitangabe.
ROH = re.compile(r"Successful (JSON )?login|Failed (JSON )?login|rate limit exceeded|User logged out|"
                 r"Revoked (all|session)|setup (token|completed)|2FA (enabled|disabled|code)|\(\d+ bytes\)|"
                 r"\bjust now\b|\d+[mhd] ago\b")
VIEWS = ["upload", "hostshare", "shares", "receive", "settings"]
SAMMLE = r"""() => {
  const out = [];
  const vis = el => { const r = el.getBoundingClientRect(); const s = getComputedStyle(el);
    return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none'; };
  const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  while (w.nextNode()) { const n = w.currentNode, t = n.textContent.trim();
    if (t && n.parentElement && !['SCRIPT','STYLE'].includes(n.parentElement.tagName) && vis(n.parentElement)) out.push(t); }
  document.querySelectorAll('[placeholder],[title],[aria-label]').forEach(el => {
    if (!vis(el)) return;
    for (const a of ['placeholder','title','aria-label']) { const v = el.getAttribute(a); if (v) out.push(v.trim()); } });
  return out; }"""

befunde, zaehl = [], {}
with sync_playwright() as p:
    b = p.chromium.launch()
    # Einmal anmelden und nur das Session-Cookie weitergeben: 14 Logins pro Minute
    # löst die Login-Sperre (5/min) aus — dann prüft der Lauf leere Seiten.
    anm = b.new_context()
    a = anm.new_page()
    a.goto(BASE + "/login")
    a.locator("input[type=password]").first.fill(pw)
    a.keyboard.press("Enter")
    a.wait_for_load_state("networkidle")
    cookies = anm.cookies()
    anm.close()
    if not any(c["name"] == "casadrop_session" for c in cookies):
        sys.exit("Anmeldung fehlgeschlagen — kein Session-Cookie")
    for lang, loc in LOCALES.items():
        T, EN = I18N[lang], I18N["en"]
        # englische Werte, die in dieser Sprache anders lauten -> Rückfall-Erkennung
        rueckfall = {v: k for k, v in EN.items() if T.get(k) != v}
        schluessel = set(EN)
        c = b.new_context(locale=loc, extra_http_headers={"Accept-Language": loc})
        g = c.new_page()
        konsole = []
        g.on("console", lambda m: konsole.append(m.text) if m.type == "error" else None)
        g.on("pageerror", lambda e: konsole.append("pageerror: " + str(e)))
        c.add_cookies(cookies)
        g.goto(BASE + "/")
        g.wait_for_load_state("networkidle")
        if not g.locator("[data-view=settings]").is_visible():
            befunde.append(f"{lang}: Admin-Oberfläche nicht erreicht")
        geprueft = 0
        for v in VIEWS:
            btn = g.locator(f"[data-view={v}]")
            if not btn.is_visible():
                continue
            btn.click()
            g.wait_for_timeout(700)
            texte = g.evaluate(SAMMLE)
            geprueft += len(texte)
            for t in texte:
                if t in rueckfall:
                    befunde.append(f"{lang} {v}: englisch geblieben {t!r} ({rueckfall[t]})")
                elif t in schluessel:
                    befunde.append(f"{lang} {v}: roher Schlüssel {t!r}")
                elif lang != "en" and ROH.search(t):
                    befunde.append(f"{lang} {v}: unübersetzter Server-/Zeittext {t!r}")
                elif "{" in t and "}" in t:
                    befunde.append(f"{lang} {v}: Platzhalter {t!r}")
            g.screenshot(path=str(OUT / f"{lang}-{v}.png"), full_page=True)
        zaehl[lang] = geprueft
        if geprueft < 50:
            befunde.append(f"{lang}: nur {geprueft} Texte geprüft — Lauf ungültig")
        befunde += [f"{lang} Konsole: {k}" for k in konsole]
        c.close()
    b.close()

(OUT / "admin-sprachen.txt").write_text("\n".join(befunde) + "\n")
print("geprüfte Texte je Sprache:", zaehl)
print(len(befunde), "Befunde")
