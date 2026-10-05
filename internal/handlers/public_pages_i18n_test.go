package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"casadrop/internal/i18n"
	"casadrop/internal/models"
)

// The recipient/guest pages were English-only until 2.5.2. These tests render
// the REAL templates (setupRealTemplateHandler) and assert on what a visitor
// sees, so reverting either the templates or renderPublic turns them red.

type publicPage struct {
	body string
	hdr  http.Header
}

func renderPublicPage(t *testing.T, h *Handler, handler func(http.ResponseWriter, *http.Request), path, id, acceptLang string) publicPage {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	if acceptLang != "" {
		req.Header.Set("Accept-Language", acceptLang)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status %d", path, rec.Code)
	}
	return publicPage{rec.Body.String(), rec.Header()}
}

var i18nBlock = regexp.MustCompile(`(?s)<script type="application/json" id="i18n-data">(.*?)</script>`)

// i18nJSON returns the string table the page script will see — parsed exactly
// as the browser's JSON.parse would, so an escaping mistake in html/template
// shows up here, not only in a browser.
func i18nJSON(t *testing.T, body string) map[string]string {
	t.Helper()
	m := i18nBlock.FindStringSubmatch(body)
	if m == nil {
		t.Fatal("no i18n-data JSON block in page")
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(m[1]), &out); err != nil {
		t.Fatalf("i18n-data is not valid JSON: %v\n%.300s", err, m[1])
	}
	return out
}

func TestPublicPagesFollowAcceptLanguage(t *testing.T) {
	h, store, cleanup := setupRealTemplateHandler(t)
	defer cleanup()

	if err := store.Save(&models.Share{
		ID: "pw0001", FileName: "pw0001.jpg", OriginalName: "Photo.jpg", FileSize: 10,
		MimeType: "image/jpeg", HasPassword: true, Password: "x",
		ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed share: %v", err)
	}
	if err := store.SaveReceiveLink(&models.ReceiveLink{
		ID: "rcv001", Name: "Fotos", MaxUploads: 20, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed receive link: %v", err)
	}

	t.Run("share page in German", func(t *testing.T) {
		p := renderPublicPage(t, h, h.SharePage, "/s/pw0001", "pw0001", "de-DE,de;q=0.9,en;q=0.8")
		for _, want := range []string{`<html lang="de">`, "Passwortgeschützt", "Entsperren", "Herunterladen"} {
			if !strings.Contains(p.body, want) {
				t.Errorf("German share page lacks %q", want)
			}
		}
		for _, notWant := range []string{"Password Protected", ">Unlock<", "Enter the password to access this file"} {
			if strings.Contains(p.body, notWant) {
				t.Errorf("German share page still contains English %q", notWant)
			}
		}
		if got := p.hdr.Get("Content-Language"); got != "de" {
			t.Errorf("Content-Language = %q, want de", got)
		}
		if !strings.Contains(strings.Join(p.hdr.Values("Vary"), ","), "Accept-Language") {
			t.Error("missing Vary: Accept-Language — a cache would serve one language to everyone")
		}
	})

	t.Run("share page defaults to English", func(t *testing.T) {
		p := renderPublicPage(t, h, h.SharePage, "/s/pw0001", "pw0001", "")
		for _, want := range []string{`<html lang="en">`, "Password Protected", ">Unlock<"} {
			if !strings.Contains(p.body, want) {
				t.Errorf("English share page lacks %q", want)
			}
		}
	})

	t.Run("receive page in German, script strings included", func(t *testing.T) {
		p := renderPublicPage(t, h, h.ReceivePage, "/r/rcv001", "rcv001", "de")
		for _, want := range []string{"Datei hierher ziehen", "oder klicken zum Auswählen", "Uploads genutzt"} {
			if !strings.Contains(p.body, want) {
				t.Errorf("German receive page lacks %q", want)
			}
		}
		if strings.Contains(p.body, "Drag &amp; drop a file here") || strings.Contains(p.body, "Drag & drop a file here") {
			t.Error("German receive page still shows the English drop text")
		}
		js := i18nJSON(t, p.body)
		if js["receive.success"] != "Datei erfolgreich hochgeladen" {
			t.Errorf("receive.success for the script = %q", js["receive.success"])
		}
	})

	t.Run("folder page in German, script strings included", func(t *testing.T) {
		if err := store.Save(&models.Share{
			ID: "dir001", FileName: "dir001", OriginalName: "Fotos", IsDirectory: true, TotalFiles: 3,
			ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("seed folder share: %v", err)
		}
		p := renderPublicPage(t, h, h.SharePage, "/s/dir001", "dir001", "de-DE")
		for _, want := range []string{`<html lang="de">`, "Alles als ZIP herunterladen", "3 Dateien", "Größe"} {
			if !strings.Contains(p.body, want) {
				t.Errorf("German folder page lacks %q", want)
			}
		}
		if js := i18nJSON(t, p.body); js["folder.empty"] != "Dieser Ordner ist leer" {
			t.Errorf("folder.empty for the script = %q", js["folder.empty"])
		}
	})

	t.Run("not found page in German", func(t *testing.T) {
		p := renderPublicPage(t, h, h.SharePage, "/s/none00", "none00", "de-AT")
		if !strings.Contains(p.body, "Freigabe nicht gefunden") {
			t.Error("German not-found page lacks its title")
		}
	})

	t.Run("?lang= overrides the header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/r/rcv001?lang=de", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "rcv001"})
		req.Header.Set("Accept-Language", "en-US")
		rec := httptest.NewRecorder()
		h.ReceivePage(rec, req)
		if !strings.Contains(rec.Body.String(), "Datei hierher ziehen") {
			t.Error("?lang=de did not switch the receive page to German")
		}
	})
}

// A key typo in a template ({{index .T "x"}}) or a script (t('x', ...)) renders
// silently empty or English. Every key the pages use must exist in the table.
func TestEveryUsedKeyExists(t *testing.T) {
	table := i18n.Strings("en")
	tplKey := regexp.MustCompile(`(?:index|tf|tfh) \.T "([^"]+)"`)
	jsKey := regexp.MustCompile(`\btf?\('([^']+)'`)
	files := map[string]*regexp.Regexp{
		"web/templates/share.html": tplKey, "web/templates/folder.html": tplKey,
		"web/templates/receive.html": tplKey, "web/templates/not_found.html": tplKey,
		"web/static/js/receive.js": jsKey, "web/static/js/folder.js": jsKey,
	}
	used := 0
	for f, re := range files {
		b, err := os.ReadFile(filepath.Join("..", "..", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			used++
			if _, ok := table[m[1]]; !ok {
				t.Errorf("%s uses unknown key %q", f, m[1])
			}
		}
	}
	if used < 40 { // positive control: the scan must actually find the keys
		t.Fatalf("only %d key uses found — the scan pattern no longer matches the files", used)
	}
}

// Receive-upload errors used to be text/plain, which receive.js could not parse,
// so every guest saw only the generic "Upload failed.". They are JSON now, in
// the visitor's language.
func TestReceiveUploadErrorIsTranslatedJSON(t *testing.T) {
	h, _, cleanup := setupRealTemplateHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/r/none00/upload", strings.NewReader(""))
	req = mux.SetURLVars(req, map[string]string{"id": "none00"})
	req.Header.Set("Accept-Language", "de-DE")
	rec := httptest.NewRecorder()
	h.ReceiveUpload(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type %q — receive.js only reads JSON", ct)
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body is not JSON: %v (%q)", err, rec.Body.String())
	}
	if out["error"] != "Diesen Empfangslink gibt es nicht oder er ist abgelaufen" {
		t.Errorf("error = %q", out["error"])
	}
}
