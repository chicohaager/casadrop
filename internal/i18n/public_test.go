package i18n

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestLangFromAcceptLanguage(t *testing.T) {
	cases := []struct {
		header, want string
	}{
		{"de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7", "de"}, // typical German browser
		{"en-US,en;q=0.9", "en"},
		{"sv-SE,sv;q=0.9,de;q=0.5", "de"}, // unsupported first, supported later
		{"sv-SE,sv;q=0.9", "en"},          // nothing supported -> fallback
		{"en;q=0.4,de;q=0.8", "de"},
		{"fr-CH,fr;q=0.9,en;q=0.8", "fr"}, // any of the 14 UI languages
		{"ja", "ja"},                      // q-value beats order
		{"de;q=0", "en"},                  // q=0 means "not acceptable"
		{"", "en"},                        // no header
		{"*", "en"},                       // wildcard alone
		{"DE-at", "de"},                   // case-insensitive, region dropped
		{"de_CH", "de"},                   // underscore variant
		{"garbage;;;,,", "en"},            // malformed
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/s/x", nil)
		if c.header != "" {
			r.Header.Set("Accept-Language", c.header)
		}
		if got := Lang(r); got != c.want {
			t.Errorf("Accept-Language %q: got %q, want %q", c.header, got, c.want)
		}
	}
}

func TestLangQueryOverride(t *testing.T) {
	r := httptest.NewRequest("GET", "/s/x?lang=de", nil)
	r.Header.Set("Accept-Language", "en-US")
	if got := Lang(r); got != "de" {
		t.Fatalf("?lang=de: got %q", got)
	}
	// unsupported query value must not win over the header
	r = httptest.NewRequest("GET", "/s/x?lang=xx", nil)
	r.Header.Set("Accept-Language", "de-DE")
	if got := Lang(r); got != "de" {
		t.Fatalf("?lang=xx with de header: got %q", got)
	}
}

// Every language must cover every key of the fallback — a missing key would
// silently show English inside a German page.
func TestEveryLanguageHasEveryKey(t *testing.T) {
	for lang, table := range public {
		for k := range public[Fallback] {
			if _, ok := table[k]; !ok {
				t.Errorf("%s: missing key %q", lang, k)
			}
		}
		for k := range table {
			if _, ok := public[Fallback][k]; !ok {
				t.Errorf("%s: key %q not in fallback (typo?)", lang, k)
			}
		}
	}
}

// The German table must not fall back to ASCII transliterations (ae/oe/ue)
// — the bug fixed in 2.5.2 for the admin UI.
func TestGermanUsesUmlauts(t *testing.T) {
	for k, v := range public["de"] {
		for _, bad := range []string{"moeglich", "groesse", "Groesse", "loesch", "geloescht", "laeuft", "fuer", "waehl", "geschuetzt", "Geschuetzt"} {
			if strings.Contains(v, bad) {
				t.Errorf("de %q contains transliteration %q: %q", k, bad, v)
			}
		}
	}
}

func TestStringsFallsBackPerKey(t *testing.T) {
	en := Strings("en")
	xx := Strings("xx")
	if len(xx) != len(en) || xx["common.unlock"] != en["common.unlock"] {
		t.Fatalf("unknown language should equal the fallback table")
	}
}

// The public/login/e-mail strings must cover exactly the languages the admin
// UI offers (SUPPORTED_LANGS in app.js). 2.5.2 first shipped only en+de here
// while the UI had 14 — this test makes such a half-done state red.
func TestLanguagesMatchAdminUI(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "web", "static", "js", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`SUPPORTED_LANGS = \[([^\]]+)\]`).FindSubmatch(b)
	if m == nil {
		t.Fatal("SUPPORTED_LANGS not found in app.js")
	}
	ui := regexp.MustCompile(`'([a-z]+)'`).FindAllSubmatch(m[1], -1)
	if len(ui) < 2 {
		t.Fatalf("parsed only %d UI languages — pattern broken", len(ui))
	}
	seen := map[string]bool{}
	for _, l := range ui {
		lang := string(l[1])
		seen[lang] = true
		if !Supported(lang) {
			t.Errorf("admin UI offers %q but internal/i18n has no locales/%s.json", lang, lang)
		}
	}
	for lang := range public {
		if !seen[lang] {
			t.Errorf("locales/%s.json exists but the admin UI does not offer %q", lang, lang)
		}
	}
}

// Every translation keeps exactly the {placeholders} of the English master,
// otherwise a value (file name, size, date) silently disappears or a literal
// "{n}" shows up.
func TestPlaceholdersMatchEnglish(t *testing.T) {
	slot := regexp.MustCompile(`\{[a-z]+\}`)
	norm := func(v string) string {
		s := slot.FindAllString(v, -1)
		sort.Strings(s)
		return strings.Join(s, ",")
	}
	for lang, table := range public {
		for k, en := range public[Fallback] {
			if k == "format.dateTime" {
				continue
			}
			if got, want := norm(table[k]), norm(en); got != want {
				t.Errorf("%s %s: placeholders %q, English has %q", lang, k, got, want)
			}
			if strings.Contains(table[k], "%s") {
				t.Errorf("%s %s still uses %%s", lang, k)
			}
		}
	}
}

func TestDateLayoutPerLanguage(t *testing.T) {
	when := time.Date(2026, 10, 5, 14, 15, 0, 0, time.UTC)
	for lang := range public {
		got := FormatTime(lang, when)
		if !strings.Contains(got, "2026") || !(strings.Contains(got, "14:15") || strings.Contains(got, "2:15")) {
			t.Errorf("%s: format.dateTime gives %q — not a Go layout?", lang, got)
		}
	}
	if got := FormatTime("de", when); got != "05.10.2026, 14:15" {
		t.Errorf("de date = %q", got)
	}
}

func TestFormatFillsSlots(t *testing.T) {
	if got := Format("de", "receive.uploaded", map[string]string{"file": "a.pdf"}); got != "a.pdf wurde hochgeladen." {
		t.Errorf("got %q", got)
	}
}

// adminUIStrings parses the I18N object of app.js: language -> key -> value.
// Every line inside a language block must be a 'key': 'value' pair; a line the
// pattern cannot read fails the test instead of being skipped silently.
func adminUIStrings(t *testing.T) map[string]map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "web", "static", "js", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	start := regexp.MustCompile(`^        ([a-z]{2}): \{$`)
	pair := regexp.MustCompile(`^\s+'([A-Za-z0-9_.]+)':\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"),?\s*$`)
	out := map[string]map[string]string{}
	lang := ""
	for i, line := range strings.Split(string(b), "\n") {
		if m := start.FindStringSubmatch(line); m != nil {
			lang = m[1]
			out[lang] = map[string]string{}
			continue
		}
		if lang == "" {
			continue
		}
		if strings.HasPrefix(line, "        }") {
			lang = ""
			continue
		}
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		m := pair.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("app.js:%d: unreadable line in I18N.%s: %q", i+1, lang, line)
		}
		out[lang][m[1]] = m[2][1 : len(m[2])-1]
	}
	if len(out["en"]) < 100 {
		t.Fatalf("parsed only %d English UI strings — parser broken", len(out["en"]))
	}
	return out
}

// The admin UI in 12 of 14 languages had only 107 of 234 strings until 2.5.2;
// the rest silently fell back to English. Every language must have every key.
func TestAdminUIEveryLanguageHasEveryKey(t *testing.T) {
	ui := adminUIStrings(t)
	if len(ui) != len(public) {
		t.Errorf("app.js has %d languages, locales/ has %d", len(ui), len(public))
	}
	ph := regexp.MustCompile(`\{[a-zA-Z]+\}`)
	for lang, strs := range ui {
		for key, en := range ui["en"] {
			v, ok := strs[key]
			if !ok {
				t.Errorf("app.js I18N.%s lacks %q", lang, key)
				continue
			}
			if strings.TrimSpace(v) == "" {
				t.Errorf("app.js I18N.%s %q is empty", lang, key)
			}
			want, got := ph.FindAllString(en, -1), ph.FindAllString(v, -1)
			sort.Strings(want)
			sort.Strings(got)
			if strings.Join(want, ",") != strings.Join(got, ",") {
				t.Errorf("app.js I18N.%s %q placeholders %v, English has %v", lang, key, got, want)
			}
		}
		for key := range strs {
			if _, ok := ui["en"][key]; !ok {
				t.Errorf("app.js I18N.%s has %q which English lacks", lang, key)
			}
		}
	}
}

// Every key the admin UI asks for must exist in English, or the raw key shows.
func TestAdminUIKeysUsedExist(t *testing.T) {
	en := adminUIStrings(t)["en"]
	used := map[string]bool{}
	for _, f := range []string{"web/static/js/app.js", "web/templates/index.html"} {
		b, err := os.ReadFile(filepath.Join("..", "..", f))
		if err != nil {
			t.Fatal(err)
		}
		for _, re := range []*regexp.Regexp{
			regexp.MustCompile(`\bt\('([A-Za-z0-9_.]+)'\)`),
			regexp.MustCompile(`data-i18n(?:-[a-z]+)?="([A-Za-z0-9_.]+)"`),
		} {
			for _, m := range re.FindAllStringSubmatch(string(b), -1) {
				used[m[1]] = true
			}
		}
	}
	if len(used) < 50 {
		t.Fatalf("found only %d used keys — pattern broken", len(used))
	}
	for k := range used {
		if _, ok := en[k]; !ok {
			t.Errorf("admin UI uses %q but I18N.en lacks it", k)
		}
	}
}

// Text written straight into app.js markup bypasses I18N and stays English in
// every language — the SMTP form, nine "could not load" messages and the theme
// toggle did until 2.5.2. Language-neutral examples are allowed explicitly.
func TestAdminUINoHardcodedText(t *testing.T) {
	allowed := map[string]bool{
		"CasaDrop": true, "STARTTLS (587)": true, "SSL/TLS (465)": true, "&times;": true,
		"name@example.com": true, ".exe,.bat,.cmd,...": true, "https://...": true, "smtp.gmail.com": true,
	}
	text := regexp.MustCompile(`>([^<>]*?)<`)
	attr := regexp.MustCompile(`(?:title|placeholder|aria-label)="([^"$]*)"`)
	interp := regexp.MustCompile(`\$\{[^}]*\}`)
	word := regexp.MustCompile(`[A-Za-z]{3,}`)
	checked := 0
	for _, f := range []string{"web/static/js/app.js", "web/templates/index.html"} {
		b, err := os.ReadFile(filepath.Join("..", "..", f))
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			trim := strings.TrimSpace(line)
			if strings.Contains(line, "data-i18n") || strings.HasPrefix(trim, "//") {
				continue
			}
			var found []string
			for _, m := range text.FindAllStringSubmatch(line, -1) {
				s := strings.TrimSpace(interp.ReplaceAllString(m[1], ""))
				// JS expressions between > and < (comparisons, string concatenation)
				if strings.ContainsAny(s, "'&|=") && !allowed[s] {
					continue
				}
				found = append(found, s)
			}
			for _, m := range attr.FindAllStringSubmatch(line, -1) {
				found = append(found, strings.TrimSpace(m[1]))
			}
			for _, s := range found {
				checked++
				if word.MatchString(s) && !allowed[s] {
					t.Errorf("%s:%d: hard-coded UI text %q — use t()/data-i18n", f, i+1, s)
				}
			}
		}
	}
	if checked < 100 {
		t.Fatalf("checked only %d markup texts — pattern broken", checked)
	}
}

// Every detail text internal/middleware records for sign-in and session events
// must match a rule in app.js ACTIVITY_DETAIL_RULES, or it shows up English in
// the activity log of every other language. The texts are read from the
// middleware source, so a new aa.audit(...) call without a rule fails here.
func TestActivityDetailsAreTranslated(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join("..", "..", p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	js := read("web/static/js/app.js")
	a := strings.Index(js, "const ACTIVITY_DETAIL_RULES = [")
	if a < 0 {
		t.Fatal("ACTIVITY_DETAIL_RULES not found in app.js")
	}
	block := js[a : a+strings.Index(js[a:], "\n    ];")]
	var rules []*regexp.Regexp
	for _, m := range regexp.MustCompile(`(?m)^\s+\[/(.+?)/, `).FindAllStringSubmatch(block, -1) {
		rules = append(rules, regexp.MustCompile(m[1]))
	}
	if len(rules) < 10 {
		t.Fatalf("parsed only %d rules — pattern broken", len(rules))
	}

	// Event types without their own kind get "TYPE: " prepended by audit.AuthSink.
	mapped := map[string]bool{}
	for _, m := range regexp.MustCompile(`middleware\.(Audit\w+):`).FindAllStringSubmatch(read("internal/audit/audit.go"), -1) {
		mapped[m[1]] = true
	}
	names := map[string]string{}
	for _, m := range regexp.MustCompile(`(Audit\w+)\s+AuditEventType = "(\w+)"`).FindAllStringSubmatch(read("internal/middleware/auth.go"), -1) {
		names[m[1]] = m[2]
	}

	call := regexp.MustCompile(`aa\.audit\((Audit\w+),[^;]*?,\s*(?:fmt\.Sprintf\()?"((?:[^"\\]|\\.)*)"(\+[^)]*)?`)
	verbs := strings.NewReplacer("%d", "3", "%s", "admin")
	total, samples := 0, 0
	for _, f := range []string{"internal/middleware/auth.go", "internal/middleware/sessions.go"} {
		src := read(f)
		total += strings.Count(src, "aa.audit(")
		for _, m := range call.FindAllStringSubmatch(src, -1) {
			sample := verbs.Replace(m[2])
			if m[3] != "" { // "Revoked session "+id+" ("+email+")"
				sample += "abc123 (a@example.org)"
			}
			if !mapped[m[1]] {
				sample = names[m[1]] + ": " + sample
			}
			samples++
			ok := false
			for _, r := range rules {
				if r.MatchString(sample) {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("%s: activity detail %q has no rule in ACTIVITY_DETAIL_RULES", f, sample)
			}
		}
	}
	if samples != total || total < 15 {
		t.Fatalf("read %d of %d aa.audit calls — extraction pattern broken", samples, total)
	}
}

// A branch on one language ("LANG === 'de' ? … : …") translates for that
// language only — timeAgo did that until 2.5.2. Texts belong in I18N.
func TestAdminUINoPerLanguageBranches(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "web", "static", "js", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`LANG\s*[!=]==?\s*'[a-z]{2}'`)
	for i, line := range strings.Split(string(b), "\n") {
		if re.MatchString(line) {
			t.Errorf("app.js:%d: per-language branch %q — put the text into I18N", i+1, strings.TrimSpace(line))
		}
	}
}
