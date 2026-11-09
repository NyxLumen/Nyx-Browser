package fbhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nyxlumen/nyx-browser/v2/auth"
	"github.com/nyxlumen/nyx-browser/v2/diskcache"
	"github.com/nyxlumen/nyx-browser/v2/frontend"
	"github.com/nyxlumen/nyx-browser/v2/settings"
	"github.com/nyxlumen/nyx-browser/v2/share"
	"github.com/nyxlumen/nyx-browser/v2/storage"
	"github.com/nyxlumen/nyx-browser/v2/storage/bolt"
	"github.com/nyxlumen/nyx-browser/v2/users"
)

type permTestRig struct {
	handler   http.Handler
	store     *storage.Storage
	server    *settings.Server
	settings  *settings.Settings
	jwtKey    []byte
	admin     *users.User
	user1     *users.User
	user2     *users.User
	root      string
	user1Home string
	user2Home string
}

func setupPermTestRig(t *testing.T) *permTestRig {
	t.Helper()
	root := t.TempDir()
	user1Home := filepath.Join(root, "user1")
	user2Home := filepath.Join(root, "user2")
	require.NoError(t, os.MkdirAll(user1Home, 0o755))
	require.NoError(t, os.MkdirAll(user2Home, 0o755))

	dbPath := filepath.Join(t.TempDir(), "perm_test.db")
	db, err := storm.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	st, err := bolt.NewStorage(db)
	require.NoError(t, err)

	jwtKey := []byte("perm-test-secret-key-32-bytes-ok")
	stg := &settings.Settings{
		Key:                   jwtKey,
		AuthMethod:            auth.MethodJSONAuth,
		MinimumPasswordLength: 6,
		FileMode:              0o644,
		DirMode:               0o755,
	}
	require.NoError(t, st.Settings.Save(stg))

	srv := &settings.Server{
		Root:             root,
		BaseURL:          "",
		EnableThumbnails: false,
		ResizePreview:    false,
		EnableExec:       false,
	}

	hashedPwd, err := users.ValidateAndHashPwd("SecretPassword123!", 6)
	require.NoError(t, err)

	admin := &users.User{
		Username: "admin_user",
		Password: hashedPwd,
		Scope:    "",
		Perm: users.Permissions{
			Admin:    true,
			Execute:  true,
			Create:   true,
			Rename:   true,
			Modify:   true,
			Delete:   true,
			Share:    true,
			Download: true,
		},
	}
	require.NoError(t, st.Users.Save(admin))

	user1 := &users.User{
		Username: "alice",
		Password: hashedPwd,
		Scope:    "user1",
		Perm: users.Permissions{
			Admin:    false,
			Execute:  false,
			Create:   true,
			Rename:   true,
			Modify:   true,
			Delete:   true,
			Share:    true,
			Download: true,
		},
	}
	require.NoError(t, st.Users.Save(user1))

	user2 := &users.User{
		Username: "bob",
		Password: hashedPwd,
		Scope:    "user2",
		Perm: users.Permissions{
			Admin:    false,
			Execute:  false,
			Create:   true,
			Rename:   true,
			Modify:   true,
			Delete:   true,
			Share:    true,
			Download: true,
		},
	}
	require.NoError(t, st.Users.Save(user2))

	assetsFs, err := fs.Sub(frontend.Assets(), "dist")
	require.NoError(t, err)

	uploadCache := newMemoryUploadCache()
	t.Cleanup(func() { uploadCache.Close() })

	handler, err := NewHandler(nil, diskcache.NewNoOp(), uploadCache, st, srv, assetsFs)
	require.NoError(t, err)

	return &permTestRig{
		handler:   handler,
		store:     st,
		server:    srv,
		settings:  stg,
		jwtKey:    jwtKey,
		admin:     admin,
		user1:     user1,
		user2:     user2,
		root:      root,
		user1Home: user1Home,
		user2Home: user2Home,
	}
}

// TestAdversarial_NonAdminCannotAccessAdminEndpoints comprehensively attacks admin endpoints.
func TestAdversarial_NonAdminCannotAccessAdminEndpoints(t *testing.T) {
	rig := setupPermTestRig(t)
	userTok := issueToken(t, rig.user1, rig.jwtKey, time.Hour)

	adminEndpoints := []struct {
		name   string
		method string
		url    string
		body   string
	}{
		{"GET /api/users", http.MethodGet, "/api/users", ""},
		{"POST /api/users", http.MethodPost, "/api/users", `{"what":"user","which":[],"data":{"username":"evil","password":"password123"}}`},
		{"GET /api/users/{other}", http.MethodGet, fmt.Sprintf("/api/users/%d", rig.user2.ID), ""},
		{"PUT /api/users/{other}", http.MethodPut, fmt.Sprintf("/api/users/%d", rig.user2.ID), `{"what":"user","which":["scope"],"current_password":"SecretPassword123!","data":{"id":3,"scope":"/"}}`},
		{"DELETE /api/users/{other}", http.MethodDelete, fmt.Sprintf("/api/users/%d", rig.user2.ID), `{"current_password":"SecretPassword123!"}`},
		{"GET /api/settings", http.MethodGet, "/api/settings", ""},
		{"PUT /api/settings", http.MethodPut, "/api/settings", `{"signup":true}`},
	}

	for _, ep := range adminEndpoints {
		t.Run("non-admin rejected on "+ep.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if ep.body != "" {
				bodyReader = bytes.NewReader([]byte(ep.body))
			} else {
				bodyReader = bytes.NewReader(nil)
			}
			req := httptest.NewRequest(ep.method, ep.url, bodyReader)
			req.Header.Set("X-Auth", userTok)
			rec := httptest.NewRecorder()
			rig.handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code, "endpoint %s must return 403 Forbidden for non-admin", ep.name)
		})
	}
}

// TestAdversarial_GranularPermissionsDisabled tests individual disabled permission flags.
func TestAdversarial_GranularPermissionsDisabled(t *testing.T) {
	rig := setupPermTestRig(t)

	// Create test files in user1's scope
	existingFile := filepath.Join(rig.user1Home, "existing.txt")
	require.NoError(t, os.WriteFile(existingFile, []byte("original content"), 0o644))
	subDir := filepath.Join(rig.user1Home, "subfolder")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	saveUserWithPerms := func(name string, p users.Permissions) string {
		u := &users.User{
			Username: name,
			Password: rig.user1.Password,
			Scope:    "user1",
			Perm:     p,
		}
		require.NoError(t, rig.store.Users.Save(u))
		return issueToken(t, u, rig.jwtKey, time.Hour)
	}

	t.Run("Create=false denies POST file creation", func(t *testing.T) {
		tok := saveUserWithPerms("no_create_file", users.Permissions{
			Create: false, Modify: true, Delete: true, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/resources/newfile.txt", bytes.NewReader([]byte("new content")))
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Create=false denies directory creation", func(t *testing.T) {
		tok := saveUserWithPerms("no_create_dir", users.Permissions{
			Create: false, Modify: true, Delete: true, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/resources/newdir/", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Create=false denies TUS upload initialization", func(t *testing.T) {
		tok := saveUserWithPerms("no_create_tus", users.Permissions{
			Create: false, Modify: true, Delete: true, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/tus/tus_file.txt", nil)
		req.Header.Set("X-Auth", tok)
		req.Header.Set("Upload-Length", "100")
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Create=false denies copy action", func(t *testing.T) {
		tok := saveUserWithPerms("no_create_copy", users.Permissions{
			Create: false, Modify: true, Delete: true, Rename: true, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodPatch, "/api/resources/existing.txt?action=copy&destination=/copy.txt", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Modify=false denies PUT file overwrite", func(t *testing.T) {
		tok := saveUserWithPerms("no_modify_put", users.Permissions{
			Create: true, Modify: false, Delete: true, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodPut, "/api/resources/existing.txt", bytes.NewReader([]byte("modified content")))
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Modify=false denies TUS override on existing file", func(t *testing.T) {
		tok := saveUserWithPerms("no_modify_tus", users.Permissions{
			Create: true, Modify: false, Delete: true, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/tus/existing.txt?override=true", nil)
		req.Header.Set("X-Auth", tok)
		req.Header.Set("Upload-Length", "10")
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Delete=false denies DELETE on file", func(t *testing.T) {
		tok := saveUserWithPerms("no_delete_file", users.Permissions{
			Create: true, Modify: true, Delete: false, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodDelete, "/api/resources/existing.txt", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.FileExists(t, existingFile, "file must not have been deleted")
	})

	t.Run("Delete=false denies DELETE on directory", func(t *testing.T) {
		tok := saveUserWithPerms("no_delete_dir", users.Permissions{
			Create: true, Modify: true, Delete: false, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodDelete, "/api/resources/subfolder/", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Rename=false denies PATCH rename action", func(t *testing.T) {
		tok := saveUserWithPerms("no_rename_patch", users.Permissions{
			Create: true, Modify: true, Delete: true, Rename: false, Download: true, Share: true,
		})
		req := httptest.NewRequest(http.MethodPatch, "/api/resources/existing.txt?action=rename&destination=/renamed.txt", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Download=false denies GET /api/raw", func(t *testing.T) {
		tok := saveUserWithPerms("no_download_raw", users.Permissions{
			Create: true, Modify: true, Delete: true, Download: false, Share: false,
		})
		req := httptest.NewRequest(http.MethodGet, "/api/raw/existing.txt", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.Contains(t, rec.Body.String(), "202 Accepted")
		assert.NotContains(t, rec.Body.String(), "original content")
	})

	t.Run("Download=false denies content in GET /api/resources with X-Encoding: true", func(t *testing.T) {
		tok := saveUserWithPerms("no_download_content", users.Permissions{
			Create: true, Modify: true, Delete: true, Download: false, Share: false,
		})
		req := httptest.NewRequest(http.MethodGet, "/api/resources/existing.txt", nil)
		req.Header.Set("X-Auth", tok)
		req.Header.Set("X-Encoding", "true")
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.Contains(t, rec.Body.String(), "202 Accepted")
		assert.NotContains(t, rec.Body.String(), "original content")
	})

	t.Run("Share=false denies POST /api/share", func(t *testing.T) {
		tok := saveUserWithPerms("no_share_post", users.Permissions{
			Create: true, Modify: true, Delete: true, Download: true, Share: false,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/share/existing.txt", bytes.NewReader([]byte(`{"expires":"1","unit":"hours"}`)))
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Share=false denies GET /api/shares listing", func(t *testing.T) {
		tok := saveUserWithPerms("no_share_list", users.Permissions{
			Create: true, Modify: true, Delete: true, Download: true, Share: false,
		})
		req := httptest.NewRequest(http.MethodGet, "/api/shares", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Download=false denies POST /api/share even if Perm.Share=true", func(t *testing.T) {
		tok := saveUserWithPerms("share_without_dl", users.Permissions{
			Create: true, Modify: true, Delete: true, Download: false, Share: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/share/existing.txt", bytes.NewReader([]byte(`{"expires":"1","unit":"hours"}`)))
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code, "sharing without download permission must be forbidden")
	})
}

// TestAdversarial_CrossUserIsolation verifies that users cannot see or manipulate other users' shares or data.
func TestAdversarial_CrossUserIsolation(t *testing.T) {
	rig := setupPermTestRig(t)

	// Create share links for user1 and user2
	link1 := &share.Link{
		Hash:   "alice_share_hash_1",
		Path:   "/existing.txt",
		UserID: rig.user1.ID,
		Expire: time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, rig.store.Share.Save(link1))

	link2 := &share.Link{
		Hash:   "bob_share_hash_2",
		Path:   "/bob_secret.txt",
		UserID: rig.user2.ID,
		Expire: time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, rig.store.Share.Save(link2))

	user1Tok := issueToken(t, rig.user1, rig.jwtKey, time.Hour)
	adminTok := issueToken(t, rig.admin, rig.jwtKey, time.Hour)

	t.Run("Alice GET /api/shares only sees Alice shares", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/shares", nil)
		req.Header.Set("X-Auth", user1Tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var shares []*shareResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &shares))
		assert.Len(t, shares, 1)
		assert.Equal(t, "alice_share_hash_1", shares[0].Hash)
	})

	t.Run("Admin GET /api/shares sees all shares", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/shares", nil)
		req.Header.Set("X-Auth", adminTok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var shares []*shareResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &shares))
		assert.Len(t, shares, 2)
	})
}


// TestAdversarial_HealthAndPublicEndpoints verifies health check accessibility and unauthenticated rejections.
func TestAdversarial_HealthAndPublicEndpoints(t *testing.T) {
	rig := setupPermTestRig(t)

	t.Run("GET /health is public and returns 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, `{"status":"OK"}`, rec.Body.String())
	})

	t.Run("GET /api/resources without auth returns 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/resources/", nil)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("GET /api/settings without auth returns 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
