package fbhttp

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/asdine/storm/v3"
	"github.com/nyxlumen/nyx-browser/v2/auth"
	"github.com/nyxlumen/nyx-browser/v2/frontend"
	"github.com/nyxlumen/nyx-browser/v2/settings"
	"github.com/nyxlumen/nyx-browser/v2/storage/bolt"
)

func setupTestServer(t *testing.T, customSettings *settings.Settings, customServer *settings.Server) (http.Handler, fs.FS) {
	t.Helper()
	root := t.TempDir()

	db, err := storm.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	st, err := bolt.NewStorage(db)
	if err != nil {
		t.Fatalf("failed to get storage: %v", err)
	}

	stg := &settings.Settings{
		Key:                   []byte("test-signing-key-32-bytes-long!!"),
		Signup:                false,
		CreateUserDir:         false,
		MinimumPasswordLength: 6,
		AuthMethod:            auth.MethodJSONAuth,
	}
	if customSettings != nil {
		if customSettings.Key == nil {
			customSettings.Key = stg.Key
		}
		if customSettings.AuthMethod == "" {
			customSettings.AuthMethod = stg.AuthMethod
		}
		stg = customSettings
	}

	if err := st.Settings.Save(stg); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	rawAuth := &auth.JSONAuth{
		ReCaptcha: &auth.ReCaptcha{
			Host:   "https://recaptcha.example.com",
			Key:    "",
			Secret: "",
		},
	}
	if err := st.Auth.Save(rawAuth); err != nil {
		t.Fatalf("failed to save auth: %v", err)
	}

	server := &settings.Server{
		Root:             root,
		BaseURL:          "",
		EnableThumbnails: true,
		ResizePreview:    true,
		EnableExec:       true,
	}
	if customServer != nil {
		if customServer.Root == "" {
			customServer.Root = root
		}
		server = customServer
	}

	assetsFs, err := fs.Sub(frontend.Assets(), "dist")
	if err != nil {
		t.Fatalf("fs.Sub(frontend.Assets(), \"dist\") failed: %v", err)
	}

	handler, err := NewHandler(nil, nil, nil, st, server, assetsFs)
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}

	return handler, assetsFs
}

func TestAssetsFS_Structure(t *testing.T) {
	rawFS := frontend.Assets()

	distDir, err := fs.ReadDir(rawFS, "dist")
	if err != nil {
		t.Fatalf("expected 'dist' directory in embedded assets, got: %v", err)
	}
	if len(distDir) == 0 {
		t.Fatal("embedded 'dist' directory is empty")
	}

	assetsFs, err := fs.Sub(rawFS, "dist")
	if err != nil {
		t.Fatalf("fs.Sub failed: %v", err)
	}

	indexData, err := fs.ReadFile(assetsFs, "public/index.html")
	if err != nil {
		t.Fatalf("failed to read public/index.html from assetsFs: %v", err)
	}
	if len(indexData) == 0 {
		t.Fatal("public/index.html is empty")
	}

	manifestData, err := fs.ReadFile(assetsFs, "manifest.json")
	if err != nil {
		t.Fatalf("failed to read manifest.json from assetsFs: %v", err)
	}
	if !strings.Contains(string(manifestData), `"Nyx Browser"`) {
		t.Errorf("manifest.json does not contain 'Nyx Browser': %s", string(manifestData))
	}
}

func TestIndexHandler_DefaultNyxBranding(t *testing.T) {
	handler, _ := setupTestServer(t, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected text/html content-type, got %s", contentType)
	}

	body := rec.Body.String()

	// Verify Page Title has Nyx Browser
	if !strings.Contains(body, "<title>\n      Nyx Browser\n    </title>") &&
		!strings.Contains(body, "<title>Nyx Browser</title>") {
		t.Errorf("expected page title 'Nyx Browser', body snippet: %s", body[:500])
	}

	// Verify Apple Mobile Web App Title
	if !strings.Contains(body, `<meta name="apple-mobile-web-app-title" content="Nyx Browser" />`) {
		t.Errorf("missing apple-mobile-web-app-title meta tag")
	}

	// Verify window.Nyx injected and window.FileBrowser shimmed
	if !strings.Contains(body, `window.Nyx = {`) {
		t.Errorf("missing 'window.Nyx = {' in rendered index.html")
	}
	if !strings.Contains(body, `window.FileBrowser = window.Nyx;`) {
		t.Errorf("missing 'window.FileBrowser = window.Nyx;' shim in rendered index.html")
	}
}

func TestIndexHandler_CustomBranding(t *testing.T) {
	custom := &settings.Settings{
		Branding: settings.Branding{
			Name:  "Lumen Enterprise Vault",
			Color: "#00ffcc",
		},
	}
	handler, _ := setupTestServer(t, custom, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "Lumen Enterprise Vault") {
		t.Errorf("expected custom title 'Lumen Enterprise Vault' in body")
	}

	if !strings.Contains(body, `content="#00ffcc"`) {
		t.Errorf("expected custom color '#00ffcc' in body")
	}

	if !strings.Contains(body, `"Name":"Lumen Enterprise Vault"`) {
		t.Errorf("expected 'Lumen Enterprise Vault' inside window.Nyx JSON")
	}
}

func TestIndexHandler_AdversarialSpecialCharacters(t *testing.T) {
	// Adversarial input: quotes, HTML entities, and unicode characters
	advName := `<script>alert("xss")</script> & 'O\'Reilly' \ "Nyx"`
	custom := &settings.Settings{
		Branding: settings.Branding{
			Name:            advName,
			DisableExternal: true,
			Color:           `#123456" onfocus="alert(1)`,
		},
	}
	handler, _ := setupTestServer(t, custom, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// 1. In <title>, html/template must escape <script> tags to prevent tag breakout
	if strings.Contains(body, "<title>\n      <script>") {
		t.Errorf("XSS flaw: unescaped script tag in <title>: %s", body[:600])
	}

	// 2. Extract window.Nyx payload from script tag and verify it parses as valid JSON
	re := regexp.MustCompile(`window\.Nyx\s*=\s*(\{.*?\});`)
	match := re.FindStringSubmatch(body)
	if len(match) < 2 {
		t.Fatalf("could not extract window.Nyx JSON payload from body: %s", body)
	}

	rawJSON := match[1]
	// In static.go, single quotes are JS-escaped as \' for safe in-script evaluation
	normalizedJSON := strings.ReplaceAll(rawJSON, `\'`, `'`)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(normalizedJSON), &parsed); err != nil {
		t.Fatalf("window.Nyx payload is invalid JSON: %v, raw: %s", err, rawJSON)
	}

	if parsed["Name"] != advName {
		t.Errorf("expected Name %q, got %q", advName, parsed["Name"])
	}
	if parsed["DisableExternal"] != true {
		t.Errorf("expected DisableExternal true, got %v", parsed["DisableExternal"])
	}
}

func TestIndexHandler_SubpathBaseURL(t *testing.T) {
	customServer := &settings.Server{
		BaseURL: "/nyx/subpath",
	}
	handler, _ := setupTestServer(t, nil, customServer)

	req := httptest.NewRequest(http.MethodGet, "/nyx/subpath/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on subpath, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"BaseURL":"/nyx/subpath"`) {
		t.Errorf("expected BaseURL '/nyx/subpath' in window.Nyx, got: %s", body)
	}
	if !strings.Contains(body, `"StaticURL":"/nyx/subpath/static"`) {
		t.Errorf("expected StaticURL '/nyx/subpath/static' in window.Nyx, got: %s", body)
	}
}

func TestStaticAssets_ManifestServing(t *testing.T) {
	handler, _ := setupTestServer(t, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/static/manifest.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /static/manifest.json, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"name": "Nyx Browser"`) {
		t.Errorf("expected '\"name\": \"Nyx Browser\"' in manifest.json, got: %s", body)
	}
}

func TestStaticAssets_GzipAndRawServing(t *testing.T) {
	handler, assetsFs := setupTestServer(t, nil, nil)

	var jsRelPath string
	err := fs.WalkDir(assetsFs, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".js") {
			if _, gzErr := fs.Stat(assetsFs, p+".gz"); gzErr == nil {
				jsRelPath = p
				return fs.SkipAll
			}
		}
		return nil
	})
	if err != nil || jsRelPath == "" {
		t.Fatalf("could not find a .js file with .gz pair in assetsFs: %v", err)
	}

	urlPath := "/static/" + jsRelPath

	// 1. Request with Accept-Encoding: gzip
	reqGz := httptest.NewRequest(http.MethodGet, urlPath, nil)
	reqGz.Header.Set("Accept-Encoding", "gzip, deflate, br")
	recGz := httptest.NewRecorder()
	handler.ServeHTTP(recGz, reqGz)

	if recGz.Code != http.StatusOK {
		t.Fatalf("expected 200 with gzip, got %d", recGz.Code)
	}
	if recGz.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got %s", recGz.Header().Get("Content-Encoding"))
	}

	gzReader, err := gzip.NewReader(recGz.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader from response: %v", err)
	}
	decompressed, err := io.ReadAll(gzReader)
	_ = gzReader.Close()
	if err != nil {
		t.Fatalf("failed to decompress gzip response: %v", err)
	}
	if len(decompressed) == 0 {
		t.Errorf("decompressed content is empty")
	}

	// 2. Request without Accept-Encoding: gzip (server decompresses on the fly)
	reqRaw := httptest.NewRequest(http.MethodGet, urlPath, nil)
	recRaw := httptest.NewRecorder()
	handler.ServeHTTP(recRaw, reqRaw)

	if recRaw.Code != http.StatusOK {
		t.Fatalf("expected 200 without gzip, got %d", recRaw.Code)
	}
	if recRaw.Header().Get("Content-Encoding") != "" {
		t.Errorf("expected empty Content-Encoding, got %s", recRaw.Header().Get("Content-Encoding"))
	}
	if string(recRaw.Body.Bytes()) != string(decompressed) {
		t.Errorf("raw served body does not match decompressed body length (%d vs %d)", recRaw.Body.Len(), len(decompressed))
	}
}
