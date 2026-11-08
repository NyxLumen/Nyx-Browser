package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/asdine/storm/v3"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvVarPrecedence_NyxOverridesFb(t *testing.T) {
	t.Setenv("NYX_PORT", "9090")
	t.Setenv("FB_PORT", "8080")
	t.Setenv("NYX_DISABLE_PREVIEW_RESIZE", "true")
	t.Setenv("FB_DISABLE_PREVIEW_RESIZE", "false")

	v, err := initViper(rootCmd)
	require.NoError(t, err)

	assert.Equal(t, "9090", v.GetString("port"), "NYX_PORT must override FB_PORT")
	assert.True(t, v.GetBool("disablePreviewResize"), "NYX_DISABLE_PREVIEW_RESIZE must override FB_DISABLE_PREVIEW_RESIZE")
}

func TestEnvVarFallback_FbWhenNyxAbsent(t *testing.T) {
	_ = os.Unsetenv("NYX_PORT")
	_ = os.Unsetenv("NYX_DISABLE_PREVIEW_RESIZE")
	t.Setenv("FB_PORT", "8085")
	t.Setenv("FB_DISABLE_PREVIEW_RESIZE", "true")

	v, err := initViper(rootCmd)
	require.NoError(t, err)

	assert.Equal(t, "8085", v.GetString("port"), "FB_PORT must be used when NYX_PORT is unset")
	assert.True(t, v.GetBool("disablePreviewResize"), "FB_DISABLE_PREVIEW_RESIZE must be used when NYX_ is unset")
}

func TestLegacyDatabaseFallback(t *testing.T) {
	tempDir := t.TempDir()
	legacyDbPath := filepath.Join(tempDir, "filebrowser.db")
	targetNyxDbPath := filepath.Join(tempDir, "nyx.db")

	// Create legacy database with storm
	db, err := storm.Open(legacyDbPath)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	// Confirm legacy database exists and nyx.db does NOT exist
	assert.FileExists(t, legacyDbPath)
	assert.NoFileExists(t, targetNyxDbPath)

	v := viper.New()
	v.Set("database", targetNyxDbPath)

	resolvedPath, exists, err := resolveDatabase(v)
	require.NoError(t, err)

	// Must resolve to legacy database with exists = true
	assert.Equal(t, legacyDbPath, resolvedPath, "resolveDatabase should fallback to legacy filebrowser.db")
	assert.True(t, exists, "resolved database should be marked as existing")

	// Ensure no silent creation, rename, or deletion
	assert.FileExists(t, legacyDbPath, "legacy database must not be deleted or renamed")
	assert.NoFileExists(t, targetNyxDbPath, "nyx.db must not be silently created during resolution")
}

func TestDatabasePreference_NyxWhenBothExist(t *testing.T) {
	tempDir := t.TempDir()
	legacyDbPath := filepath.Join(tempDir, "filebrowser.db")
	nyxDbPath := filepath.Join(tempDir, "nyx.db")

	// Create both databases
	db1, err := storm.Open(legacyDbPath)
	require.NoError(t, err)
	require.NoError(t, db1.Close())

	db2, err := storm.Open(nyxDbPath)
	require.NoError(t, err)
	require.NoError(t, db2.Close())

	v := viper.New()
	v.Set("database", nyxDbPath)

	resolvedPath, exists, err := resolveDatabase(v)
	require.NoError(t, err)

	assert.Equal(t, nyxDbPath, resolvedPath, "nyx.db must take precedence over filebrowser.db when both exist")
	assert.True(t, exists)
	assert.FileExists(t, legacyDbPath, "legacy filebrowser.db must remain untouched")
}

func TestDatabaseCustomPathNoFallback(t *testing.T) {
	tempDir := t.TempDir()
	customDbPath := filepath.Join(tempDir, "custom.db")
	legacyDbPath := filepath.Join(tempDir, "filebrowser.db")

	// Create legacy database
	db, err := storm.Open(legacyDbPath)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	v := viper.New()
	v.Set("database", customDbPath)

	resolvedPath, exists, err := resolveDatabase(v)
	require.NoError(t, err)

	assert.Equal(t, customDbPath, resolvedPath, "custom path must not fallback to filebrowser.db")
	assert.False(t, exists, "custom database does not exist")
}

func TestConfigFileResolutionPrecedence(t *testing.T) {
	tempDir := t.TempDir()

	// Write legacy .filebrowser.json
	legacyConfig := `{"port": "7777"}`
	err := os.WriteFile(filepath.Join(tempDir, ".filebrowser.json"), []byte(legacyConfig), 0644)
	require.NoError(t, err)

	// 1. With only legacy config present:
	v1 := viper.New()
	v1.AddConfigPath(tempDir)
	v1.SetConfigName(".nyx")
	err = v1.ReadInConfig()
	var notFoundErr viper.ConfigFileNotFoundError
	if errorsAs(err, &notFoundErr) {
		v1.SetConfigName(".filebrowser")
		err = v1.ReadInConfig()
	}
	require.NoError(t, err)
	assert.Equal(t, "7777", v1.GetString("port"))

	// 2. Now write .nyx.json in the same directory:
	nyxConfig := `{"port": "9999"}`
	err = os.WriteFile(filepath.Join(tempDir, ".nyx.json"), []byte(nyxConfig), 0644)
	require.NoError(t, err)

	v2 := viper.New()
	v2.AddConfigPath(tempDir)
	v2.SetConfigName(".nyx")
	err = v2.ReadInConfig()
	if errorsAs(err, &notFoundErr) {
		v2.SetConfigName(".filebrowser")
		err = v2.ReadInConfig()
	}
	require.NoError(t, err)
	assert.Equal(t, "9999", v2.GetString("port"), ".nyx.* config must take precedence over .filebrowser.*")
}

func errorsAs(err error, target any) bool {
	if err == nil {
		return false
	}
	return errors.As(err, target)
}
