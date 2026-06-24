package routes_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"casadrop/internal/auth"
	"casadrop/internal/handlers"
	"casadrop/internal/middleware"
	"casadrop/internal/models"
	"casadrop/internal/routes"
	"casadrop/internal/storage"
)

// TestSharePasswordHeadValidation guards a routing regression: the share page
// validates a share password with a HEAD probe to /d/{id}?password=... (see
// web/static/js/share.js) and only accepts the password when the probe returns
// 2xx. gorilla/mux does NOT route HEAD to a GET-only handler — it falls through
// to the 404 NotFound handler — so when /d/{id} was registered with
// Methods("GET") only, every HEAD probe returned 404 and the correct password
// was rejected. The route now lists "GET", "HEAD" and DownloadFile answers HEAD
// once the password check passes, without consuming a download.
//
// This test drives the REAL router so it exercises the actual method matching.
func TestSharePasswordHeadValidation(t *testing.T) {
	srv, store, cleanup := newDownloadTestServer(t)
	defer cleanup()

	const password = "s3cret-share-pw!"

	// Create a real on-disk file and a password-protected share pointing at it.
	fileName := "head-test-file.bin"
	if err := os.WriteFile(filepath.Join(store.UploadsDir(), fileName), []byte("hello world"), 0644); err != nil {
		t.Fatalf("write upload file: %v", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	share := &models.Share{
		ID:           "headshare1",
		FileName:     fileName,
		OriginalName: "report.bin",
		FileSize:     11,
		MimeType:     "application/octet-stream",
		Password:     hash,
		HasPassword:  true,
		ExpiresAt:    time.Now().Add(time.Hour),
		CreatedAt:    time.Now(),
		MaxDownloads: 1, // proves the HEAD probe must NOT consume a download
	}
	if err := store.Save(share); err != nil {
		t.Fatalf("save share: %v", err)
	}

	client := newClientWithJar(t, srv)
	base := srv.URL + "/d/" + share.ID

	cases := []struct {
		name   string
		method string
		query  string
		want   int
	}{
		{"HEAD wrong password -> 401", http.MethodHead, "?password=nope", http.StatusUnauthorized},
		{"HEAD no password -> 401", http.MethodHead, "", http.StatusUnauthorized},
		{"HEAD correct password -> 200", http.MethodHead, "?password=" + password, http.StatusOK},
		{"GET correct password -> 200", http.MethodGet, "?password=" + password, http.StatusOK},
	}
	for _, tc := range cases {
		res := do(t, client, tc.method, base+tc.query, nil, nil)
		res.Body.Close()
		if res.StatusCode != tc.want {
			t.Errorf("%s: want %d, got %d", tc.name, tc.want, res.StatusCode)
		}
	}

	// The HEAD probes above must not have spent the single allowed download;
	// the real GET download is what counts against MaxDownloads.
	got, ok := store.Get(share.ID)
	if !ok {
		t.Fatalf("share vanished")
	}
	if got.Downloads != 1 {
		t.Errorf("download counter: want 1 (only the GET), got %d — HEAD probes leaked downloads", got.Downloads)
	}
}

func newDownloadTestServer(t *testing.T) (*httptest.Server, *storage.Storage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "casadrop-dl-head-*")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	must(t, os.MkdirAll(filepath.Join(tmpDir, "uploads"), 0755))
	templatesDir := filepath.Join(tmpDir, "templates")
	must(t, os.MkdirAll(templatesDir, 0755))
	for _, name := range []string{"index.html", "share.html", "folder.html", "receive.html", "login.html", "setup.html", "error.html"} {
		must(t, os.WriteFile(filepath.Join(templatesDir, name), []byte(`<html><body>{{.}}</body></html>`), 0644))
	}
	t.Setenv("DATA_DIR", tmpDir)

	store, err := storage.New(tmpDir)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	h, err := handlers.New(store, templatesDir)
	if err != nil {
		store.Close()
		t.Fatalf("handlers: %v", err)
	}

	adminAuth := middleware.NewAdminAuth("integration-test-pass-123!", tmpDir)
	oidcProvider, _ := auth.NewProvider(tmpDir)
	oidcHandlers := auth.NewHandlers(oidcProvider, adminAuth)
	oidcHandlers.SetUserService(auth.NewUserService(store))
	emailHandler := handlers.NewEmailHandler(store)
	h.SetEmailHandler(emailHandler)
	downloadLimiter := middleware.NewRateLimiter(1000, time.Minute)

	router := routes.New(routes.Deps{
		Handler:         h,
		AdminAuth:       adminAuth,
		OIDC:            oidcHandlers,
		EmailHandler:    emailHandler,
		DownloadLimiter: downloadLimiter,
		StaticDir:       tmpDir,
	})
	srv := httptest.NewServer(router)

	cleanup := func() {
		srv.Close()
		adminAuth.Stop()
		downloadLimiter.Stop()
		emailHandler.Stop()
		if oidcProvider != nil {
			oidcProvider.Stop()
		}
		store.Close()
		os.RemoveAll(tmpDir)
	}
	return srv, store, cleanup
}
