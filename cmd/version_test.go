package cmd

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nyxlumen/nyx-browser/v2/version"
)

// versionTestLock ensures sequential execution of tests modifying rootCmd global state.
var versionTestLock sync.Mutex

func captureStdoutAndBuffer(fn func(buf *bytes.Buffer) error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	outChan := make(chan string)
	go func() {
		var pipeBuf bytes.Buffer
		_, _ = io.Copy(&pipeBuf, r)
		_ = r.Close()
		outChan <- pipeBuf.String()
	}()

	execErr := fn(buf)

	_ = w.Close()
	os.Stdout = oldStdout
	pipeOutput := <-outChan

	// Return combined output of Cobra output buffer and stdout pipe
	combined := buf.String() + pipeOutput
	return combined, execErr
}

func TestVersionFieldConfigured(t *testing.T) {
	assert.NotEmpty(t, rootCmd.Version, "rootCmd.Version must be set")
	assert.Equal(t, version.Version, rootCmd.Version, "rootCmd.Version must match version.Version")
	assert.NotEmpty(t, version.Version, "version.Version must not be empty")
}

func TestVersionFlag(t *testing.T) {
	versionTestLock.Lock()
	defer versionTestLock.Unlock()

	out, err := captureStdoutAndBuffer(func(_ *bytes.Buffer) error {
		rootCmd.SetArgs([]string{"--version"})
		return rootCmd.Execute()
	})

	require.NoError(t, err, "nyx --version must succeed")
	assert.Contains(t, out, "Nyx Browser version")
	assert.Contains(t, out, version.Version)
}

func TestVersionShorthandFlag(t *testing.T) {
	versionTestLock.Lock()
	defer versionTestLock.Unlock()

	out, err := captureStdoutAndBuffer(func(_ *bytes.Buffer) error {
		rootCmd.SetArgs([]string{"-v"})
		return rootCmd.Execute()
	})

	require.NoError(t, err, "nyx -v must succeed")
	assert.Contains(t, out, "Nyx Browser version")
	assert.Contains(t, out, version.Version)
}

func TestVersionSubcommand(t *testing.T) {
	versionTestLock.Lock()
	defer versionTestLock.Unlock()

	out, err := captureStdoutAndBuffer(func(_ *bytes.Buffer) error {
		rootCmd.SetArgs([]string{"version"})
		return rootCmd.Execute()
	})

	require.NoError(t, err, "nyx version must succeed")
	assert.Contains(t, out, "Nyx Browser v")
	assert.Contains(t, out, version.Version)
	assert.Contains(t, out, version.CommitSHA)
}
