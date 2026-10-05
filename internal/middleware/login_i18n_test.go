package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The /login and /setup pages are built in Go, not from templates, and were
// English-only until 2.5.2. Assert on what the visitor gets.
func TestLoginPageFollowsAcceptLanguage(t *testing.T) {
	aa := NewAdminAuth("pw", t.TempDir())

	get := func(lang string) (*httptest.ResponseRecorder, string) {
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		if lang != "" {
			req.Header.Set("Accept-Language", lang)
		}
		rec := httptest.NewRecorder()
		aa.LoginHandler(rec, req)
		return rec, rec.Body.String()
	}

	rec, body := get("de-DE,de;q=0.9")
	for _, want := range []string{`<html lang="de">`, "Admin-Anmeldung", "Passwort", "(für den Admin leer lassen)", ">Anmelden<"} {
		if !strings.Contains(body, want) {
			t.Errorf("German login page lacks %q", want)
		}
	}
	for _, notWant := range []string{"Admin Login", "(leave blank for admin)"} {
		if strings.Contains(body, notWant) {
			t.Errorf("German login page still contains %q", notWant)
		}
	}
	if rec.Header().Get("Content-Language") != "de" {
		t.Errorf("Content-Language = %q", rec.Header().Get("Content-Language"))
	}

	_, body = get("")
	for _, want := range []string{`<html lang="en">`, "Admin Login", ">Login<"} {
		if !strings.Contains(body, want) {
			t.Errorf("English login page lacks %q", want)
		}
	}
}

// A wrong password on the form login shows the translated error, not a raw key.
func TestLoginErrorIsTranslatedNotAKey(t *testing.T) {
	aa := NewAdminAuth("pw", t.TempDir())
	token, err := aa.GenerateCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	form := "csrf_token=" + token + "&password=wrong"
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "de")
	rec := httptest.NewRecorder()
	aa.LoginHandler(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, "login.err.") {
		t.Fatalf("raw i18n key leaked into the page: %.200s", body)
	}
	if !strings.Contains(body, "Anmeldedaten ungültig") && !strings.Contains(body, "Falsches Passwort") {
		t.Errorf("German error message missing")
	}
}
