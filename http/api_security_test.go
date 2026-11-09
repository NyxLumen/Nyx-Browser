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
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nyxlumen/nyx-browser/v2/auth"
	"github.com/nyxlumen/nyx-browser/v2/diskcache"
	"github.com/nyxlumen/nyx-browser/v2/frontend"
	"github.com/nyxlumen/nyx-browser/v2/settings"
	"github.com/nyxlumen/nyx-browser/v2/storage"
	"github.com/nyxlumen/nyx-browser/v2/storage/bolt"
	"github.com/nyxlumen/nyx-browser/v2/users"
)

type testRig struct {
	handler   http.Handler
	store     *storage.Storage
	server    *settings.Server
	settings  *settings.Settings
	jwtKey    []byte
	admin     *users.User
	regular   *users.User
	userScope string
	srvRoot   string
	plainPwd  string
}

func setupSecurityTestRig(t *testing.T) *testRig {
	t.Helper()
	root := t.TempDir()
	userHome := filepath.Join(root, "userhome")
	require.NoError(t, os.MkdirAll(userHome, 0o755))

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := storm.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	st, err := bolt.NewStorage(db)
	require.NoError(t, err)

	jwtKey := []byte("very-secure-random-jwt-secret-key-12345")
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
		EnableThumbnails: true,
		ResizePreview:    true,
		EnableExec:       false,
	}

	plainPwd := "K9#mQ2$vL8!zW5@xY3"
	hashedPwd, err := users.ValidateAndHashPwd(plainPwd, 6)
	require.NoError(t, err)

	adminUser := &users.User{
		Username: "admin",
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
	require.NoError(t, st.Users.Save(adminUser))

	regularUser := &users.User{
		Username: "regular",
		Password: hashedPwd,
		Scope:    "userhome",
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
	require.NoError(t, st.Users.Save(regularUser))

	assetsFs, err := fs.Sub(frontend.Assets(), "dist")
	require.NoError(t, err)

	uploadCache := newMemoryUploadCache()
	t.Cleanup(func() { uploadCache.Close() })
	handler, err := NewHandler(nil, diskcache.NewNoOp(), uploadCache, st, srv, assetsFs)
	require.NoError(t, err)

	return &testRig{
		handler:   handler,
		store:     st,
		server:    srv,
		settings:  stg,
		jwtKey:    jwtKey,
		admin:     adminUser,
		regular:   regularUser,
		userScope: userHome,
		srvRoot:   root,
		plainPwd:  plainPwd,
	}
}

func issueToken(t *testing.T, user *users.User, key []byte, exp time.Duration) string {
	t.Helper()
	claims := &authToken{
		User: userInfo{
			ID:       user.ID,
			Username: user.Username,
			Perm:     user.Perm,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			Issuer:    "Nyx Browser",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func TestAdminEndpointsAuthorization(t *testing.T) {
	rig := setupSecurityTestRig(t)

	adminToken := issueToken(t, rig.admin, rig.jwtKey, time.Hour)
	regularToken := issueToken(t, rig.regular, rig.jwtKey, time.Hour)

	t.Run("GET /api/users requires admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		req.Header.Set("X-Auth", regularToken)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)

		reqAdmin := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		reqAdmin.Header.Set("X-Auth", adminToken)
		recAdmin := httptest.NewRecorder()
		rig.handler.ServeHTTP(recAdmin, reqAdmin)
		assert.Equal(t, http.StatusOK, recAdmin.Code)
	})

	t.Run("POST /api/users requires admin", func(t *testing.T) {
		body := `{"what":"user","which":[],"data":{"username":"attacker","password":"password123"}}`

		req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader([]byte(body)))
		req.Header.Set("X-Auth", regularToken)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("DELETE /api/users/{id} for another user requires admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/users/%d", rig.admin.ID), bytes.NewReader([]byte(`{}`)))
		req.Header.Set("X-Auth", regularToken)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("GET and PUT /api/settings requires admin", func(t *testing.T) {
		reqGet := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		reqGet.Header.Set("X-Auth", regularToken)
		recGet := httptest.NewRecorder()
		rig.handler.ServeHTTP(recGet, reqGet)
		assert.Equal(t, http.StatusForbidden, recGet.Code)

		reqPut := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte(`{"signup":true}`)))
		reqPut.Header.Set("X-Auth", regularToken)
		recPut := httptest.NewRecorder()
		rig.handler.ServeHTTP(recPut, reqPut)
		assert.Equal(t, http.StatusForbidden, recPut.Code)

		reqGetAdmin := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		reqGetAdmin.Header.Set("X-Auth", adminToken)
		recGetAdmin := httptest.NewRecorder()
		rig.handler.ServeHTTP(recGetAdmin, reqGetAdmin)
		assert.Equal(t, http.StatusOK, recGetAdmin.Code)
	})

	t.Run("PUT /api/users/{self} privilege escalation attempt is rejected", func(t *testing.T) {
		payload := map[string]interface{}{
			"what":             "user",
			"which":            []string{"perm"},
			"current_password": rig.plainPwd,
			"data": map[string]interface{}{
				"id": rig.regular.ID,
				"perm": map[string]bool{
					"admin":    true,
					"execute":  true,
					"download": true,
				},
			},
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/users/%d", rig.regular.ID), bytes.NewReader(data))
		req.Header.Set("X-Auth", regularToken)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code, "privilege escalation attempt must return 403 Forbidden")

		scopePayload := map[string]interface{}{
			"what":             "user",
			"which":            []string{"scope"},
			"current_password": rig.plainPwd,
			"data": map[string]interface{}{
				"id":    rig.regular.ID,
				"scope": "/",
			},
		}
		dataScope, _ := json.Marshal(scopePayload)
		reqScope := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/users/%d", rig.regular.ID), bytes.NewReader(dataScope))
		reqScope.Header.Set("X-Auth", regularToken)
		recScope := httptest.NewRecorder()
		rig.handler.ServeHTTP(recScope, reqScope)
		assert.Equal(t, http.StatusForbidden, recScope.Code, "scope escape attempt must return 403 Forbidden")
	})
}

func TestJWTTokenSecurity(t *testing.T) {
	rig := setupSecurityTestRig(t)

	protectedURL := "/api/resources/"

	t.Run("valid token is accepted", func(t *testing.T) {
		token := issueToken(t, rig.regular, rig.jwtKey, time.Hour)
		req := httptest.NewRequest(http.MethodGet, protectedURL, nil)
		req.Header.Set("X-Auth", token)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("token with invalid/forged signature is rejected", func(t *testing.T) {
		wrongKey := []byte("wrong-signing-key-attacker-provided!!")
		token := issueToken(t, rig.regular, wrongKey, time.Hour)

		req := httptest.NewRequest(http.MethodGet, protectedURL, nil)
		req.Header.Set("X-Auth", token)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		token := issueToken(t, rig.regular, rig.jwtKey, -10*time.Minute)

		req := httptest.NewRequest(http.MethodGet, protectedURL, nil)
		req.Header.Set("X-Auth", token)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("token using alg none is rejected", func(t *testing.T) {
		claims := &authToken{
			User: userInfo{ID: rig.regular.ID, Username: rig.regular.Username},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
		unsignedToken := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		signed, err := unsignedToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, protectedURL, nil)
		req.Header.Set("X-Auth", signed)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("token without exp claim is rejected", func(t *testing.T) {
		claims := &authToken{
			User: userInfo{ID: rig.regular.ID, Username: rig.regular.Username},
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt: jwt.NewNumericDate(time.Now()),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString(rig.jwtKey)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, protectedURL, nil)
		req.Header.Set("X-Auth", signed)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestFileOperationsPermissionsEnforcement(t *testing.T) {
	rig := setupSecurityTestRig(t)

	targetFile := filepath.Join(rig.userScope, "existing.txt")
	require.NoError(t, os.WriteFile(targetFile, []byte("content"), 0o644))

	createUserWithPerms := func(name string, p users.Permissions) (*users.User, string) {
		u := &users.User{
			Username: name,
			Password: rig.admin.Password,
			Scope:    "userhome",
			Perm:     p,
		}
		require.NoError(t, rig.store.Users.Save(u))
		tok := issueToken(t, u, rig.jwtKey, time.Hour)
		return u, tok
	}

	t.Run("POST /api/resources without Perm.Create is rejected", func(t *testing.T) {
		_, tok := createUserWithPerms("no_create", users.Permissions{
			Create:   false,
			Modify:   true,
			Download: true,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/resources/new.txt", bytes.NewReader([]byte("new content")))
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("PUT /api/resources without Perm.Modify is rejected", func(t *testing.T) {
		_, tok := createUserWithPerms("no_modify", users.Permissions{
			Create:   true,
			Modify:   false,
			Download: true,
		})

		req := httptest.NewRequest(http.MethodPut, "/api/resources/existing.txt", bytes.NewReader([]byte("edited")))
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("DELETE /api/resources without Perm.Delete is rejected", func(t *testing.T) {
		_, tok := createUserWithPerms("no_delete", users.Permissions{
			Create:   true,
			Modify:   true,
			Delete:   false,
			Download: true,
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/resources/existing.txt", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("PATCH rename without Perm.Rename is rejected", func(t *testing.T) {
		_, tok := createUserWithPerms("no_rename", users.Permissions{
			Create:   true,
			Modify:   true,
			Rename:   false,
			Download: true,
		})

		req := httptest.NewRequest(http.MethodPatch, "/api/resources/existing.txt?action=rename&destination=/renamed.txt", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("GET /api/raw without Perm.Download is rejected", func(t *testing.T) {
		_, tok := createUserWithPerms("no_dl", users.Permissions{
			Create:   true,
			Modify:   true,
			Download: false,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/raw/existing.txt", nil)
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusAccepted, rec.Code, "raw download without Perm.Download returns 202 Accepted")
		assert.NotContains(t, rec.Body.String(), "content", "no file content should be served")
	})

	t.Run("POST /api/share without Perm.Share is rejected", func(t *testing.T) {
		_, tok := createUserWithPerms("no_share", users.Permissions{
			Create:   true,
			Modify:   true,
			Download: true,
			Share:    false,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/share/existing.txt", bytes.NewReader([]byte(`{"expires":"1","unit":"days"}`)))
		req.Header.Set("X-Auth", tok)
		rec := httptest.NewRecorder()
		rig.handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestDirectoryTraversalProtection(t *testing.T) {
	rig := setupSecurityTestRig(t)

	secretOutside := filepath.Join(rig.srvRoot, "top_secret.txt")
	require.NoError(t, os.WriteFile(secretOutside, []byte("SUPER_SECRET_DATA"), 0o644))

	inScopeFile := filepath.Join(rig.userScope, "hello.txt")
	require.NoError(t, os.WriteFile(inScopeFile, []byte("hello world"), 0o644))

	regularToken := issueToken(t, rig.regular, rig.jwtKey, time.Hour)

	traversalPaths := []string{
		"/api/resources/..%2f..%2ftop_secret.txt",
		"/api/resources/..%2ftop_secret.txt",
		"/api/resources/dir/..%2f..%2ftop_secret.txt",
		"/api/raw/..%2f..%2ftop_secret.txt",
		"/api/raw/..%2ftop_secret.txt",
	}

	for _, p := range traversalPaths {
		t.Run("traversal attempt "+p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			req.Header.Set("X-Auth", regularToken)
			rec := httptest.NewRecorder()
			rig.handler.ServeHTTP(rec, req)

			// The secret data must NEVER leak in response body
			assert.NotContains(t, rec.Body.String(), "SUPER_SECRET_DATA")
			// The status must not be 200 OK
			assert.NotEqual(t, http.StatusOK, rec.Code)
		})
	}
}
