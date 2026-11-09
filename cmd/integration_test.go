package cmd

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/asdine/storm/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nyxlumen/nyx-browser/v2/auth"
	"github.com/nyxlumen/nyx-browser/v2/diskcache"
	"github.com/nyxlumen/nyx-browser/v2/frontend"
	fbhttp "github.com/nyxlumen/nyx-browser/v2/http"
	"github.com/nyxlumen/nyx-browser/v2/img"
	"github.com/nyxlumen/nyx-browser/v2/settings"
	"github.com/nyxlumen/nyx-browser/v2/storage/bolt"
	"github.com/nyxlumen/nyx-browser/v2/users"
)

var (
	buildBinaryOnce sync.Once
	cachedBinary    string
	buildBinaryErr  error
)

// getOrBuildNyxBinary compiles the nyx binary into a temporary location once for CLI integration tests.
func getOrBuildNyxBinary(t *testing.T) string {
	t.Helper()
	buildBinaryOnce.Do(func() {
		binDir, err := os.MkdirTemp("", "nyx-bin-test-*")
		if err != nil {
			buildBinaryErr = err
			return
		}
		target := filepath.Join(binDir, "nyx")
		cmd := exec.Command("go", "build", "-o", target, "..")
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildBinaryErr = fmt.Errorf("go build failed: %w, output: %s", err, string(out))
			return
		}
		cachedBinary = target
	})

	if buildBinaryErr != nil {
		t.Fatalf("failed to build nyx test binary: %v", buildBinaryErr)
	}
	return cachedBinary
}

// createTestLegacyDatabase creates a real legacy filebrowser.db containing pre-migration users and settings.
func createTestLegacyDatabase(t *testing.T, dbPath string) {
	t.Helper()
	db, err := storm.Open(dbPath, storm.BoltOptions(databasePermissions, nil))
	require.NoError(t, err)
	defer db.Close()

	st, err := bolt.NewStorage(db)
	require.NoError(t, err)

	appSettings := &settings.Settings{
		Key:                   generateKey(),
		Signup:                false,
		HideLoginButton:       true,
		CreateUserDir:         false,
		MinimumPasswordLength: settings.DefaultMinimumPasswordLength,
		Defaults: settings.UserDefaults{
			Scope:       ".",
			Locale:      "en",
			SingleClick: false,
			Perm: users.Permissions{
				Admin:    false,
				Execute:  true,
				Create:   true,
				Rename:   true,
				Modify:   true,
				Delete:   true,
				Share:    true,
				Download: true,
			},
		},
		AuthMethod: auth.MethodJSONAuth,
		Branding: settings.Branding{
			Name:  "Legacy File Browser Corp",
			Theme: "dark",
			Color: "#334455",
		},
		Tus: settings.Tus{
			ChunkSize:  settings.DefaultTusChunkSize,
			RetryCount: settings.DefaultTusRetryCount,
		},
	}
	require.NoError(t, st.Settings.Save(appSettings))

	serverSettings := &settings.Server{
		Address:          "127.0.0.1",
		Port:             "8080",
		Root:             ".",
		EnableThumbnails: true,
		ResizePreview:    true,
		EnableExec:       false,
	}
	require.NoError(t, st.Settings.SaveServer(serverSettings))

	require.NoError(t, st.Auth.Save(&auth.JSONAuth{
		ReCaptcha: &auth.ReCaptcha{
			Host:   "https://recaptcha.net",
			Key:    "legacy-key",
			Secret: "legacy-secret",
		},
	}))

	adminHash, err := users.ValidateAndHashPwd("LegacyAdmin123!", appSettings.MinimumPasswordLength)
	require.NoError(t, err)
	adminUser := &users.User{
		Username:     "legacy_admin",
		Password:     adminHash,
		LockPassword: false,
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

	operatorHash, err := users.ValidateAndHashPwd("OperatorPass456!", appSettings.MinimumPasswordLength)
	require.NoError(t, err)
	operatorUser := &users.User{
		Username:     "legacy_operator",
		Password:     operatorHash,
		Scope:        "/srv/data",
		Locale:       "fr",
		LockPassword: true,
		Perm: users.Permissions{
			Admin:    false,
			Execute:  false,
			Create:   true,
			Rename:   true,
			Modify:   true,
			Delete:   false,
			Share:    true,
			Download: true,
		},
	}
	require.NoError(t, st.Users.Save(operatorUser))
}

// computeFileSHA256 returns hex-encoded SHA256 checksum of a file.
func computeFileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// newIsolatedTestCommand creates a fresh cobra.Command with the exact flags of rootCmd.
func newIsolatedTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nyx",
		Short: "Test command",
	}

	persistent := cmd.PersistentFlags()
	persistent.StringP("config", "c", "", "config file path")
	persistent.StringP("database", "d", "./nyx.db", "database path")

	flags := cmd.Flags()
	flags.Bool("noauth", false, "use noauth auther")
	flags.String("username", "admin", "first user username")
	flags.String("password", "", "first user password")
	flags.Uint32("socketPerm", 0666, "unix socket permissions")
	flags.String("cacheDir", "", "file cache directory")
	flags.String("redisCacheUrl", "", "redis cache URL")
	flags.Int("imageProcessors", 4, "image processors count")
	addServerFlags(flags)

	return cmd
}

// -----------------------------------------------------------------------------
// 1. CLEAN INSTALLATION INTEGRATION TESTS
// -----------------------------------------------------------------------------

func TestIntegration_CleanInstallation(t *testing.T) {
	t.Run("HeadlessQuickSetup_ExplicitCredentials", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "nyx.db")

		// Confirm clean environment: no database exists yet
		assert.NoFileExists(t, dbPath)

		hashedPass, err := users.HashPwd("NyxSuperSecure2026!")
		require.NoError(t, err)

		v := viper.New()
		v.Set("database", dbPath)
		v.Set("username", "sysadmin")
		v.Set("password", hashedPass)
		v.Set("port", "8899")
		v.Set("address", "127.0.0.1")
		v.Set("log", "stdout")
		v.Set("root", tempDir)
		v.Set("tokenExpirationTime", "4h")
		v.Set("disableThumbnails", false)
		v.Set("disablePreviewResize", false)
		v.Set("disableExec", true)

		// Open storage on the target database path
		db, err := storm.Open(dbPath, storm.BoltOptions(databasePermissions, nil))
		require.NoError(t, err)

		st, err := bolt.NewStorage(db)
		require.NoError(t, err)

		// Run quick setup
		err = quickSetup(v, st)
		require.NoError(t, err)
		require.NoError(t, db.Close())

		// Verify database file was created on disk with correct permissions
		info, err := os.Stat(dbPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(databasePermissions), info.Mode().Perm())

		// Reopen database with fresh storage to verify persistence
		reopenedDB, err := storm.Open(dbPath, storm.BoltOptions(databasePermissions, nil))
		require.NoError(t, err)
		defer reopenedDB.Close()

		reopenedSt, err := bolt.NewStorage(reopenedDB)
		require.NoError(t, err)

		// Verify user "sysadmin"
		adminUser, err := reopenedSt.Users.Get("", false, "sysadmin")
		require.NoError(t, err)
		assert.Equal(t, uint(1), adminUser.ID)
		assert.Equal(t, "sysadmin", adminUser.Username)
		assert.True(t, adminUser.Perm.Admin, "quick setup user must have admin privileges")
		assert.True(t, users.CheckPwd("NyxSuperSecure2026!", adminUser.Password), "password must be valid bcrypt hash")

		// Verify server settings
		srv, err := reopenedSt.Settings.GetServer()
		require.NoError(t, err)
		assert.Equal(t, "8899", srv.Port)
		assert.Equal(t, "127.0.0.1", srv.Address)
		assert.Equal(t, tempDir, srv.Root)
		assert.Equal(t, "4h", srv.TokenExpirationTime)
		assert.True(t, srv.EnableThumbnails)
		assert.True(t, srv.ResizePreview)
		assert.False(t, srv.EnableExec)

		// Verify application settings
		appSet, err := reopenedSt.Settings.Get()
		require.NoError(t, err)
		assert.False(t, appSet.Signup)
		assert.Equal(t, auth.MethodJSONAuth, appSet.AuthMethod)
		assert.Equal(t, uint(settings.DefaultMinimumPasswordLength), appSet.MinimumPasswordLength)
		assert.Len(t, appSet.Key, 64, "signing key must be 64 bytes")
	})

	t.Run("HeadlessQuickSetup_RandomPasswordGeneration", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "nyx.db")

		v := viper.New()
		v.Set("database", dbPath)
		v.Set("username", "admin")
		v.Set("password", "") // Empty: instructs quickSetup to generate a random password

		db, err := storm.Open(dbPath, storm.BoltOptions(databasePermissions, nil))
		require.NoError(t, err)

		st, err := bolt.NewStorage(db)
		require.NoError(t, err)

		err = quickSetup(v, st)
		require.NoError(t, err)
		require.NoError(t, db.Close())

		// Reopen and verify user has a valid bcrypt hash
		reopenedDB, err := storm.Open(dbPath, storm.BoltOptions(databasePermissions, nil))
		require.NoError(t, err)
		defer reopenedDB.Close()

		reopenedSt, err := bolt.NewStorage(reopenedDB)
		require.NoError(t, err)

		user, err := reopenedSt.Users.Get("", false, "admin")
		require.NoError(t, err)
		assert.Equal(t, "admin", user.Username)
		assert.NotEmpty(t, user.Password)
		assert.True(t, strings.HasPrefix(user.Password, "$2a$") || strings.HasPrefix(user.Password, "$2y$"),
			"generated password must be stored as bcrypt hash")
	})

	t.Run("DefaultDatabaseNameResolution_IsNyxDb", func(t *testing.T) {
		tempDir := t.TempDir()
		defaultTarget := filepath.Join(tempDir, "nyx.db")

		v := viper.New()
		v.Set("database", defaultTarget)

		resolvedPath, exists, err := resolveDatabase(v)
		require.NoError(t, err)
		assert.Equal(t, defaultTarget, resolvedPath)
		assert.False(t, exists, "new database must not exist yet")
		assert.NoFileExists(t, defaultTarget, "resolveDatabase must not create files")
	})
}

// -----------------------------------------------------------------------------
// 2. LEGACY FILE BROWSER DATABASE NON-DESTRUCTIVE FALLBACK TESTS
// -----------------------------------------------------------------------------

func TestIntegration_LegacyDatabaseFallback(t *testing.T) {
	t.Run("FallbackToLegacyDatabase_WhenNyxDbAbsent", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyDbPath := filepath.Join(tempDir, "filebrowser.db")
		nyxDbPath := filepath.Join(tempDir, "nyx.db")

		// Populate legacy database
		createTestLegacyDatabase(t, legacyDbPath)
		assert.FileExists(t, legacyDbPath)
		assert.NoFileExists(t, nyxDbPath)

		// Record initial checksum and size
		initialChecksum := computeFileSHA256(t, legacyDbPath)
		initialStat, err := os.Stat(legacyDbPath)
		require.NoError(t, err)
		assert.Greater(t, initialStat.Size(), int64(0))

		// Nyx configured with default target ./nyx.db
		v := viper.New()
		v.Set("database", nyxDbPath)

		resolvedPath, exists, err := resolveDatabase(v)
		require.NoError(t, err)

		// Must detect filebrowser.db and report exists == true
		assert.Equal(t, legacyDbPath, resolvedPath, "must fall back to filebrowser.db when nyx.db is absent")
		assert.True(t, exists, "detected legacy database must be flagged as existing")

		// Verify non-destructive invariants:
		// 1. filebrowser.db was NOT renamed or removed
		assert.FileExists(t, legacyDbPath, "legacy database file must remain in place")
		// 2. nyx.db was NOT silently created
		assert.NoFileExists(t, nyxDbPath, "nyx.db must not be created during database resolution")

		// Open storage on the resolved legacy path and verify full data fidelity
		db, err := storm.Open(resolvedPath, storm.BoltOptions(databasePermissions, nil))
		require.NoError(t, err)
		defer db.Close()

		st, err := bolt.NewStorage(db)
		require.NoError(t, err)

		// Verify users loaded from legacy DB
		admin, err := st.Users.Get("", false, "legacy_admin")
		require.NoError(t, err)
		assert.Equal(t, "legacy_admin", admin.Username)
		assert.True(t, admin.Perm.Admin)
		assert.True(t, users.CheckPwd("LegacyAdmin123!", admin.Password))

		operator, err := st.Users.Get("", false, "legacy_operator")
		require.NoError(t, err)
		assert.Equal(t, "legacy_operator", operator.Username)
		assert.Equal(t, "/srv/data", operator.Scope)
		assert.Equal(t, "fr", operator.Locale)
		assert.False(t, operator.Perm.Admin)
		assert.False(t, operator.Perm.Delete)
		assert.True(t, operator.Perm.Create)
		assert.True(t, users.CheckPwd("OperatorPass456!", operator.Password))

		// Verify branding and server settings
		set, err := st.Settings.Get()
		require.NoError(t, err)
		assert.Equal(t, "Legacy File Browser Corp", set.Branding.Name)
		assert.Equal(t, "dark", set.Branding.Theme)
		assert.Equal(t, "#334455", set.Branding.Color)

		// Perform a non-destructive write (add a new user) to confirm read-write compatibility
		newUserHash, err := users.ValidateAndHashPwd("NewUserPass789!", set.MinimumPasswordLength)
		require.NoError(t, err)
		newUser := &users.User{
			Username: "migrated_user",
			Password: newUserHash,
		}
		require.NoError(t, st.Users.Save(newUser))

		// Verify newly added user is retrievable alongside legacy users
		retrievedNew, err := st.Users.Get("", false, "migrated_user")
		require.NoError(t, err)
		assert.Equal(t, "migrated_user", retrievedNew.Username)

		// Assert filebrowser.db remains in place and nyx.db was never created
		assert.FileExists(t, legacyDbPath)
		assert.NoFileExists(t, nyxDbPath)
		_ = initialChecksum // Checksum changed as expected due to added user
	})

	t.Run("NyxDbPreferred_WhenBothDatabasesExist", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyDbPath := filepath.Join(tempDir, "filebrowser.db")
		nyxDbPath := filepath.Join(tempDir, "nyx.db")

		// Create legacy database with user "legacy_user"
		createTestLegacyDatabase(t, legacyDbPath)

		// Create native nyx database with distinct user "native_nyx_admin"
		nyxDB, err := storm.Open(nyxDbPath, storm.BoltOptions(databasePermissions, nil))
		require.NoError(t, err)

		nyxSt, err := bolt.NewStorage(nyxDB)
		require.NoError(t, err)

		nyxSet := &settings.Settings{
			Key:        generateKey(),
			AuthMethod: auth.MethodJSONAuth,
			Branding:   settings.Branding{Name: "Native Nyx Instance"},
		}
		require.NoError(t, nyxSt.Settings.Save(nyxSet))

		nyxAdminHash, err := users.ValidateAndHashPwd("NyxNative123!", 6)
		require.NoError(t, err)
		require.NoError(t, nyxSt.Users.Save(&users.User{
			Username: "native_nyx_admin",
			Password: nyxAdminHash,
			Perm:     users.Permissions{Admin: true},
		}))
		require.NoError(t, nyxDB.Close())

		// Both databases exist on disk
		assert.FileExists(t, legacyDbPath)
		assert.FileExists(t, nyxDbPath)

		v := viper.New()
		v.Set("database", nyxDbPath)

		resolvedPath, exists, err := resolveDatabase(v)
		require.NoError(t, err)

		// Must prefer nyx.db
		assert.Equal(t, nyxDbPath, resolvedPath, "nyx.db must take precedence over filebrowser.db when both exist")
		assert.True(t, exists)

		// Open resolved path and confirm it is the native nyx DB
		db, err := storm.Open(resolvedPath, storm.BoltOptions(databasePermissions, nil))
		require.NoError(t, err)
		defer db.Close()

		st, err := bolt.NewStorage(db)
		require.NoError(t, err)

		set, err := st.Settings.Get()
		require.NoError(t, err)
		assert.Equal(t, "Native Nyx Instance", set.Branding.Name)

		// Legacy filebrowser.db must remain untouched
		assert.FileExists(t, legacyDbPath)
	})

	t.Run("CustomDatabasePath_DoesNotFallbackToLegacy", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyDbPath := filepath.Join(tempDir, "filebrowser.db")
		customDbPath := filepath.Join(tempDir, "custom_name.db")

		// Create legacy DB in the directory
		createTestLegacyDatabase(t, legacyDbPath)

		v := viper.New()
		v.Set("database", customDbPath)

		resolvedPath, exists, err := resolveDatabase(v)
		require.NoError(t, err)

		// Custom path should not fall back to filebrowser.db
		assert.Equal(t, customDbPath, resolvedPath)
		assert.False(t, exists, "custom database does not exist yet")
	})
}

// -----------------------------------------------------------------------------
// 3. CONFIGURATION FILE PRECEDENCE TESTS
// -----------------------------------------------------------------------------

func TestIntegration_ConfigFilePrecedence(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)

	t.Run("NyxConfigOverridesFilebrowserConfig_YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		// Write legacy .filebrowser.yaml
		legacyContent := "port: \"8001\"\naddress: \"127.0.0.10\"\nbaseURL: \"/legacy\"\n"
		err := os.WriteFile(filepath.Join(tempDir, ".filebrowser.yaml"), []byte(legacyContent), 0644)
		require.NoError(t, err)

		// Write modern .nyx.yaml
		nyxContent := "port: \"9001\"\naddress: \"127.0.0.20\"\nbaseURL: \"/nyx\"\n"
		err = os.WriteFile(filepath.Join(tempDir, ".nyx.yaml"), []byte(nyxContent), 0644)
		require.NoError(t, err)

		err = os.Chdir(tempDir)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		cmd := newIsolatedTestCommand()
		v, err := initViper(cmd)
		require.NoError(t, err)

		assert.Equal(t, "9001", v.GetString("port"), ".nyx.yaml port must take precedence")
		assert.Equal(t, "127.0.0.20", v.GetString("address"), ".nyx.yaml address must take precedence")
		assert.Equal(t, "/nyx", v.GetString("baseURL"), ".nyx.yaml baseURL must take precedence")
		assert.Contains(t, v.ConfigFileUsed(), ".nyx.yaml")
	})

	t.Run("FallbackToFilebrowserConfig_WhenNyxAbsent_JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		// Write only legacy .filebrowser.json
		legacyJSON := `{"port": "8002", "address": "127.0.0.30", "baseURL": "/fb-legacy"}`
		err := os.WriteFile(filepath.Join(tempDir, ".filebrowser.json"), []byte(legacyJSON), 0644)
		require.NoError(t, err)

		err = os.Chdir(tempDir)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		cmd := newIsolatedTestCommand()
		v, err := initViper(cmd)
		require.NoError(t, err)

		assert.Equal(t, "8002", v.GetString("port"), "must load port from .filebrowser.json fallback")
		assert.Equal(t, "127.0.0.30", v.GetString("address"))
		assert.Equal(t, "/fb-legacy", v.GetString("baseURL"))
		assert.Contains(t, v.ConfigFileUsed(), ".filebrowser.json")
	})

	t.Run("SupportMultipleFormats_TOML", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		tomlContent := "port = \"8003\"\naddress = \"127.0.0.40\"\n"
		err := os.WriteFile(filepath.Join(tempDir, ".nyx.toml"), []byte(tomlContent), 0644)
		require.NoError(t, err)

		err = os.Chdir(tempDir)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		cmd := newIsolatedTestCommand()
		v, err := initViper(cmd)
		require.NoError(t, err)

		assert.Equal(t, "8003", v.GetString("port"))
		assert.Equal(t, "127.0.0.40", v.GetString("address"))
		assert.Contains(t, v.ConfigFileUsed(), ".nyx.toml")
	})

	t.Run("ExplicitConfigFlag_OverridesAutomaticDiscovery", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		// Place .nyx.yaml in current dir
		err := os.WriteFile(filepath.Join(tempDir, ".nyx.yaml"), []byte("port: \"9005\"\n"), 0644)
		require.NoError(t, err)

		// Place custom config file elsewhere
		customDir := t.TempDir()
		customPath := filepath.Join(customDir, "custom-config.json")
		err = os.WriteFile(customPath, []byte(`{"port": "9999", "address": "10.10.10.10"}`), 0644)
		require.NoError(t, err)

		err = os.Chdir(tempDir)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		cmd := newIsolatedTestCommand()
		err = cmd.ParseFlags([]string{"--config", customPath})
		require.NoError(t, err)

		v, err := initViper(cmd)
		require.NoError(t, err)

		assert.Equal(t, "9999", v.GetString("port"), "--config flag must override directory discovery")
		assert.Equal(t, "10.10.10.10", v.GetString("address"))
		assert.Equal(t, customPath, v.ConfigFileUsed())
	})
}

// -----------------------------------------------------------------------------
// 4. DUAL-ENVIRONMENT VARIABLE AND FLAG PRECEDENCE TESTS
// -----------------------------------------------------------------------------

func TestIntegration_DualEnvAndFlagPrecedence(t *testing.T) {
	t.Run("NYX_StrictlyOverrides_FB", func(t *testing.T) {
		t.Setenv("NYX_PORT", "9100")
		t.Setenv("FB_PORT", "8100")
		t.Setenv("NYX_ADDRESS", "192.168.1.10")
		t.Setenv("FB_ADDRESS", "10.0.0.1")
		t.Setenv("NYX_DISABLE_PREVIEW_RESIZE", "true")
		t.Setenv("FB_DISABLE_PREVIEW_RESIZE", "false")
		t.Setenv("NYX_TOKEN_EXPIRATION_TIME", "6h")
		t.Setenv("FB_TOKEN_EXPIRATION_TIME", "1h")

		cmd := newIsolatedTestCommand()
		v, err := initViper(cmd)
		require.NoError(t, err)

		assert.Equal(t, "9100", v.GetString("port"), "NYX_PORT must override FB_PORT")
		assert.Equal(t, "192.168.1.10", v.GetString("address"), "NYX_ADDRESS must override FB_ADDRESS")
		assert.True(t, v.GetBool("disablePreviewResize"), "NYX_DISABLE_PREVIEW_RESIZE must override FB_DISABLE_PREVIEW_RESIZE")
		assert.Equal(t, "6h", v.GetString("tokenExpirationTime"), "NYX_TOKEN_EXPIRATION_TIME must override FB_")
	})

	t.Run("FB_ActsAsFallback_WhenNYXUnset", func(t *testing.T) {
		_ = os.Unsetenv("NYX_PORT")
		_ = os.Unsetenv("NYX_ADDRESS")
		_ = os.Unsetenv("NYX_DISABLE_PREVIEW_RESIZE")
		_ = os.Unsetenv("NYX_TOKEN_EXPIRATION_TIME")

		t.Setenv("FB_PORT", "8200")
		t.Setenv("FB_ADDRESS", "10.20.30.40")
		t.Setenv("FB_DISABLE_PREVIEW_RESIZE", "true")
		t.Setenv("FB_TOKEN_EXPIRATION_TIME", "3h")

		cmd := newIsolatedTestCommand()
		v, err := initViper(cmd)
		require.NoError(t, err)

		assert.Equal(t, "8200", v.GetString("port"), "FB_PORT must serve as fallback")
		assert.Equal(t, "10.20.30.40", v.GetString("address"), "FB_ADDRESS must serve as fallback")
		assert.True(t, v.GetBool("disablePreviewResize"), "FB_DISABLE_PREVIEW_RESIZE must serve as fallback")
		assert.Equal(t, "3h", v.GetString("tokenExpirationTime"), "FB_TOKEN_EXPIRATION_TIME must serve as fallback")
	})

	t.Run("CLIFlags_StrictlyOverride_Both_NYX_and_FB", func(t *testing.T) {
		t.Setenv("NYX_PORT", "9300")
		t.Setenv("FB_PORT", "8300")
		t.Setenv("NYX_ADDRESS", "192.168.1.99")
		t.Setenv("FB_ADDRESS", "10.99.99.99")

		cmd := newIsolatedTestCommand()
		// Parse CLI flags as if user passed --port 7777 --address 127.0.0.1
		err := cmd.ParseFlags([]string{"--port", "7777", "--address", "127.0.0.1"})
		require.NoError(t, err)

		v, err := initViper(cmd)
		require.NoError(t, err)

		assert.Equal(t, "7777", v.GetString("port"), "CLI flag --port must override both NYX_PORT and FB_PORT")
		assert.Equal(t, "127.0.0.1", v.GetString("address"), "CLI flag --address must override both NYX_ADDRESS and FB_ADDRESS")
	})

	t.Run("FullPrecedenceChain_Flags_Over_NYX_Over_FB_Over_Config_Over_Defaults", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)
		origDir, err := os.Getwd()
		require.NoError(t, err)

		// Config file sets port: 6000, address: 127.0.0.6, tokenExpirationTime: 1h
		configFile := `{"port": "6000", "address": "127.0.0.6", "tokenExpirationTime": "1h"}`
		err = os.WriteFile(filepath.Join(tempDir, ".nyx.json"), []byte(configFile), 0644)
		require.NoError(t, err)

		err = os.Chdir(tempDir)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		// Layer 1: Config file overrides defaults (address)
		// Layer 2: FB_ overrides config (tokenExpirationTime)
		t.Setenv("FB_TOKEN_EXPIRATION_TIME", "2h")
		// Layer 3: NYX_ overrides FB_ and config (port)
		t.Setenv("FB_PORT", "7000")
		t.Setenv("NYX_PORT", "8000")
		// Layer 4: CLI flag overrides NYX_ (log)
		t.Setenv("NYX_LOG", "stdout")
		t.Setenv("FB_LOG", "stderr")

		cmd := newIsolatedTestCommand()
		err = cmd.ParseFlags([]string{"--log", "/tmp/nyx.log"})
		require.NoError(t, err)

		v, err := initViper(cmd)
		require.NoError(t, err)

		// Flag > Env
		assert.Equal(t, "/tmp/nyx.log", v.GetString("log"), "Flag must override Env")
		// NYX_ > FB_ > Config
		assert.Equal(t, "8000", v.GetString("port"), "NYX_ must override FB_ and config")
		// FB_ > Config
		assert.Equal(t, "2h", v.GetString("tokenExpirationTime"), "FB_ must override config when NYX_ unset")
		// Config > Default
		assert.Equal(t, "127.0.0.6", v.GetString("address"), "Config must override default when env unset")
	})
}

// -----------------------------------------------------------------------------
// 5. EMBEDDED FRONTEND ASSET SERVING INTEGRATION TESTS
// -----------------------------------------------------------------------------

func TestIntegration_EmbeddedFrontendAssetServing(t *testing.T) {
	// Setup test storage and HTTP handler with real embedded assets
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "nyx.db")

	db, err := storm.Open(dbPath, storm.BoltOptions(databasePermissions, nil))
	require.NoError(t, err)
	defer db.Close()

	st, err := bolt.NewStorage(db)
	require.NoError(t, err)

	appSettings := &settings.Settings{
		Key:                   []byte("test-signing-key-32-bytes-long!!"),
		Signup:                true,
		CreateUserDir:         false,
		MinimumPasswordLength: 6,
		AuthMethod:            auth.MethodJSONAuth,
		Branding: settings.Branding{
			Name:  "Nyx Browser Enterprise",
			Theme: "dark",
			Color: "#00ddbb",
		},
	}
	require.NoError(t, st.Settings.Save(appSettings))

	serverSettings := &settings.Server{
		Root:             tempDir,
		BaseURL:          "",
		EnableThumbnails: true,
		ResizePreview:    true,
		EnableExec:       false,
	}
	require.NoError(t, st.Settings.SaveServer(serverSettings))

	require.NoError(t, st.Auth.Save(&auth.JSONAuth{
		ReCaptcha: &auth.ReCaptcha{
			Host:   "https://recaptcha.example.com",
			Key:    "test-recaptcha-key",
			Secret: "test-recaptcha-secret",
		},
	}))

	assetsFs, err := fs.Sub(frontend.Assets(), "dist")
	require.NoError(t, err)

	imgService := img.New(1)
	fileCache := diskcache.NewNoOp()
	uploadCache, err := fbhttp.NewUploadCache("")
	require.NoError(t, err)

	handler, err := fbhttp.NewHandler(imgService, fileCache, uploadCache, st, serverSettings, assetsFs)
	require.NoError(t, err)

	t.Run("AssetsFS_StructureAndFiles", func(t *testing.T) {
		rawFS := frontend.Assets()
		require.NotNil(t, rawFS)

		distEntries, err := fs.ReadDir(rawFS, "dist")
		require.NoError(t, err)
		assert.NotEmpty(t, distEntries)

		// Check index.html exists
		indexBytes, err := fs.ReadFile(assetsFs, "public/index.html")
		require.NoError(t, err)
		assert.NotEmpty(t, indexBytes)
		assert.Contains(t, string(indexBytes), "window.Nyx")

		// Check manifest.json contains product branding
		manifestBytes, err := fs.ReadFile(assetsFs, "manifest.json")
		require.NoError(t, err)
		assert.Contains(t, string(manifestBytes), `"name": "Nyx Browser"`)
	})

	t.Run("IndexHTML_RuntimeConfigAndHeaders", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "1; mode=block", rec.Header().Get("x-xss-protection"))

		body := rec.Body.String()

		// Verify page title
		assert.True(t, strings.Contains(body, "<title>\n      Nyx Browser Enterprise\n    </title>") ||
			strings.Contains(body, "<title>Nyx Browser Enterprise</title>"), "title must match branding name")

		// Verify Apple Mobile Web App title
		assert.Contains(t, body, `<meta name="apple-mobile-web-app-title" content="Nyx Browser" />`)

		// Verify window.Nyx injection and window.FileBrowser shim
		assert.Contains(t, body, "window.Nyx = {", "must inject window.Nyx")
		assert.Contains(t, body, "window.FileBrowser = window.Nyx;", "must provide backward-compatible shim")

		// Extract window.Nyx JSON and verify runtime fields
		re := regexp.MustCompile(`window\.Nyx\s*=\s*(\{.*?\});`)
		matches := re.FindStringSubmatch(body)
		require.Len(t, matches, 2, "must extract window.Nyx JSON payload")

		normalizedJSON := strings.ReplaceAll(matches[1], `\'`, `'`)
		var nyxConfig map[string]interface{}
		err = json.Unmarshal([]byte(normalizedJSON), &nyxConfig)
		require.NoError(t, err, "window.Nyx payload must be valid JSON")

		assert.Equal(t, "Nyx Browser Enterprise", nyxConfig["Name"])
		assert.Equal(t, "#00ddbb", nyxConfig["Color"])
		assert.Equal(t, "/static", nyxConfig["StaticURL"])
		assert.Equal(t, "json", nyxConfig["AuthMethod"])
		assert.Equal(t, true, nyxConfig["Signup"])
		assert.Equal(t, true, nyxConfig["EnableThumbs"])
		assert.Equal(t, false, nyxConfig["EnableExec"])
	})

	t.Run("StaticAssetServing_ManifestAndCaching", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/manifest.json", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "public, max-age=86400", rec.Header().Get("Cache-Control"))
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Contains(t, rec.Body.String(), `"name": "Nyx Browser"`)
	})

	t.Run("StaticAssetServing_JavaScript_GzipAndRaw", func(t *testing.T) {
		// Discover an actual JS file with .gz pair in assetsFs
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
		require.NoError(t, err)
		require.NotEmpty(t, jsRelPath, "compiled assets must contain at least one .js and .js.gz pair")

		urlPath := "/static/" + jsRelPath

		// 1. Request with Accept-Encoding: gzip
		reqGz := httptest.NewRequest(http.MethodGet, urlPath, nil)
		reqGz.Header.Set("Accept-Encoding", "gzip, deflate, br")
		recGz := httptest.NewRecorder()
		handler.ServeHTTP(recGz, reqGz)

		assert.Equal(t, http.StatusOK, recGz.Code)
		assert.Equal(t, "gzip", recGz.Header().Get("Content-Encoding"))
		assert.Equal(t, "application/javascript; charset=utf-8", recGz.Header().Get("Content-Type"))
		assert.Equal(t, "public, max-age=86400", recGz.Header().Get("Cache-Control"))

		gzReader, err := gzip.NewReader(recGz.Body)
		require.NoError(t, err)
		decompressedBytes, err := io.ReadAll(gzReader)
		_ = gzReader.Close()
		require.NoError(t, err)
		assert.NotEmpty(t, decompressedBytes)

		// 2. Request without Accept-Encoding (on-the-fly decompressed)
		reqRaw := httptest.NewRequest(http.MethodGet, urlPath, nil)
		recRaw := httptest.NewRecorder()
		handler.ServeHTTP(recRaw, reqRaw)

		assert.Equal(t, http.StatusOK, recRaw.Code)
		assert.Empty(t, recRaw.Header().Get("Content-Encoding"), "raw response must not specify Content-Encoding: gzip")
		assert.Equal(t, "application/javascript; charset=utf-8", recRaw.Header().Get("Content-Type"))
		assert.Equal(t, decompressedBytes, recRaw.Body.Bytes(), "raw response content must match decompressed gzip bytes")
	})

	t.Run("StaticAssetServing_ErrorCases", func(t *testing.T) {
		// Non-existent static asset
		reqNotFound := httptest.NewRequest(http.MethodGet, "/static/nonexistent_asset.xyz", nil)
		recNotFound := httptest.NewRecorder()
		handler.ServeHTTP(recNotFound, reqNotFound)
		assert.Equal(t, http.StatusNotFound, recNotFound.Code)

		// POST method to static file must return 404
		reqPost := httptest.NewRequest(http.MethodPost, "/static/manifest.json", nil)
		recPost := httptest.NewRecorder()
		handler.ServeHTTP(recPost, reqPost)
		assert.Equal(t, http.StatusNotFound, recPost.Code)
	})
}

// -----------------------------------------------------------------------------
// 6. END-TO-END CLI BINARY WORKFLOW INTEGRATION TESTS
// -----------------------------------------------------------------------------

func TestIntegration_EndToEndCLI(t *testing.T) {
	binaryPath := getOrBuildNyxBinary(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "cli_integration.db")

	t.Run("VersionSubcommand", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "version")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "nyx version must exit cleanly")
		assert.Contains(t, string(out), "Nyx Browser v")
	})

	t.Run("ConfigInit_And_UsersAdd_Workflow", func(t *testing.T) {
		// 1. Initialize configuration and database
		initCmd := exec.Command(binaryPath, "config", "init",
			"--database", dbPath,
			"--address", "127.0.0.1",
			"--port", "9876",
		)
		initOut, err := initCmd.CombinedOutput()
		require.NoError(t, err, "nyx config init must succeed: %s", string(initOut))
		assert.FileExists(t, dbPath, "database file must be created by config init")

		// 2. Add an admin user via CLI
		addCmd := exec.Command(binaryPath, "users", "add", "cli_admin", "CliAdminPass123!",
			"--database", dbPath,
			"--perm.admin",
		)
		addOut, err := addCmd.CombinedOutput()
		require.NoError(t, err, "nyx users add must succeed: %s", string(addOut))
		assert.Contains(t, string(addOut), "cli_admin")

		// 3. List users via CLI
		lsCmd := exec.Command(binaryPath, "users", "ls", "--database", dbPath)
		lsOut, err := lsCmd.CombinedOutput()
		require.NoError(t, err, "nyx users ls must succeed: %s", string(lsOut))
		assert.Contains(t, string(lsOut), "cli_admin")

		// 4. Export configuration to JSON
		exportPath := filepath.Join(tempDir, "exported_config.json")
		exportCmd := exec.Command(binaryPath, "config", "export", exportPath, "--database", dbPath)
		exportOut, err := exportCmd.CombinedOutput()
		require.NoError(t, err, "nyx config export must succeed: %s", string(exportOut))
		assert.FileExists(t, exportPath)

		exportedData, err := os.ReadFile(exportPath)
		require.NoError(t, err)
		assert.Contains(t, string(exportedData), `"port": "9876"`)
		assert.Contains(t, string(exportedData), `"address": "127.0.0.1"`)
	})
}
