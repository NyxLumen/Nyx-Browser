package search

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type allowAllChecker struct{}

func (allowAllChecker) Check(string) bool {
	return true
}

type denySpecificChecker struct {
	deniedPath string
}

func (d denySpecificChecker) Check(p string) bool {
	return p != d.deniedPath
}

func TestParseSearch(t *testing.T) {
	t.Run("default case insensitive and multiple terms", func(t *testing.T) {
		opts := parseSearch("Hello World")
		assert.False(t, opts.CaseSensitive)
		assert.Equal(t, []string{"hello", "world"}, opts.Terms)
		assert.Empty(t, opts.Conditions)
	})

	t.Run("case sensitive flag", func(t *testing.T) {
		opts := parseSearch("case:sensitive Hello World")
		assert.True(t, opts.CaseSensitive)
		assert.Equal(t, []string{"Hello", "World"}, opts.Terms)
		assert.Empty(t, opts.Conditions)
	})

	t.Run("case insensitive flag explicitly passed", func(t *testing.T) {
		opts := parseSearch("case:insensitive MyFile")
		assert.False(t, opts.CaseSensitive)
		assert.Equal(t, []string{"myfile"}, opts.Terms)
	})

	t.Run("exact quoted phrase", func(t *testing.T) {
		opts := parseSearch(`"Exact Phrase Match"`)
		assert.False(t, opts.CaseSensitive)
		assert.Equal(t, []string{"exact phrase match"}, opts.Terms)
	})

	t.Run("exact quoted phrase case sensitive", func(t *testing.T) {
		opts := parseSearch(`case:sensitive "Exact Phrase Match"`)
		assert.True(t, opts.CaseSensitive)
		assert.Equal(t, []string{"Exact Phrase Match"}, opts.Terms)
	})

	t.Run("type image condition", func(t *testing.T) {
		opts := parseSearch("type:image vacation")
		assert.Equal(t, []string{"vacation"}, opts.Terms)
		require.Len(t, opts.Conditions, 1)

		assert.True(t, opts.Conditions[0]("/photo.jpg"))
		assert.True(t, opts.Conditions[0]("/photo.png"))
		assert.False(t, opts.Conditions[0]("/document.txt"))
	})

	t.Run("type audio and music conditions", func(t *testing.T) {
		optsAudio := parseSearch("type:audio track")
		require.Len(t, optsAudio.Conditions, 1)
		assert.True(t, optsAudio.Conditions[0]("/song.mp3"))
		assert.False(t, optsAudio.Conditions[0]("/video.mp4"))

		optsMusic := parseSearch("type:music melody")
		require.Len(t, optsMusic.Conditions, 1)
		assert.True(t, optsMusic.Conditions[0]("/track.flac"))
	})

	t.Run("type video condition", func(t *testing.T) {
		opts := parseSearch("type:video clip")
		require.Len(t, opts.Conditions, 1)
		assert.True(t, opts.Conditions[0]("/movie.mp4"))
		assert.False(t, opts.Conditions[0]("/photo.png"))
	})

	t.Run("type custom extension condition", func(t *testing.T) {
		opts := parseSearch("type:pdf report")
		assert.Equal(t, []string{"report"}, opts.Terms)
		require.Len(t, opts.Conditions, 1)
		assert.True(t, opts.Conditions[0]("/doc.pdf"))
		assert.False(t, opts.Conditions[0]("/doc.docx"))
	})

	t.Run("empty or whitespace search value", func(t *testing.T) {
		opts1 := parseSearch("")
		assert.Empty(t, opts1.Terms)
		assert.Empty(t, opts1.Conditions)

		opts2 := parseSearch("   ")
		assert.Empty(t, opts2.Terms)
		assert.Empty(t, opts2.Conditions)

		opts3 := parseSearch("case:sensitive")
		assert.True(t, opts3.CaseSensitive)
		assert.Empty(t, opts3.Terms)
	})

	t.Run("consecutive and leading trailing whitespace in terms", func(t *testing.T) {
		opts := parseSearch("   alpha    beta   gamma   ")
		assert.False(t, opts.CaseSensitive)
		assert.Equal(t, []string{"alpha", "beta", "gamma"}, opts.Terms)
		assert.NotContains(t, opts.Terms, "")
	})
}

func TestSearch(t *testing.T) {
	setupFs := func() afero.Fs {
		fs := afero.NewMemMapFs()
		_ = fs.MkdirAll("/docs/work", 0o755)
		_ = fs.MkdirAll("/media/images", 0o755)
		_ = fs.MkdirAll("/media/audio", 0o755)

		_ = afero.WriteFile(fs, "/docs/work/report.pdf", []byte("report"), 0o644)
		_ = afero.WriteFile(fs, "/docs/notes.txt", []byte("notes"), 0o644)
		_ = afero.WriteFile(fs, "/docs/Readme.MD", []byte("readme"), 0o644)
		_ = afero.WriteFile(fs, "/media/images/sunset.jpg", []byte("jpeg"), 0o644)
		_ = afero.WriteFile(fs, "/media/images/Sunset_HighRes.PNG", []byte("png"), 0o644)
		_ = afero.WriteFile(fs, "/media/audio/track01.mp3", []byte("mp3"), 0o644)
		return fs
	}

	t.Run("search by term case insensitive", func(t *testing.T) {
		fs := setupFs()
		ctx := context.Background()

		var matched []string
		err := Search(ctx, fs, "/", "sunset", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{
			"media/images/sunset.jpg",
			"media/images/Sunset_HighRes.PNG",
		}, matched)
	})

	t.Run("search by term case sensitive", func(t *testing.T) {
		fs := setupFs()
		ctx := context.Background()

		var matched []string
		err := Search(ctx, fs, "/", "case:sensitive Sunset", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{
			"media/images/Sunset_HighRes.PNG",
		}, matched)
	})

	t.Run("search within specific scope subfolder", func(t *testing.T) {
		fs := setupFs()
		ctx := context.Background()

		var matched []string
		err := Search(ctx, fs, "/docs", "report", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		// Relative path from /docs should be work/report.pdf
		assert.Equal(t, []string{"work/report.pdf"}, matched)
	})

	t.Run("search with type condition", func(t *testing.T) {
		fs := setupFs()
		ctx := context.Background()

		var matched []string
		err := Search(ctx, fs, "/", "type:pdf", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"docs/work/report.pdf"}, matched)
	})

	t.Run("search with combined terms and type conditions", func(t *testing.T) {
		fs := setupFs()
		ctx := context.Background()

		var matched []string
		err := Search(ctx, fs, "/", "sunset type:image", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{
			"media/images/sunset.jpg",
			"media/images/Sunset_HighRes.PNG",
		}, matched)
	})

	t.Run("checker denies specific paths", func(t *testing.T) {
		fs := setupFs()
		ctx := context.Background()

		checker := denySpecificChecker{deniedPath: "/docs/work/report.pdf"}
		var matched []string
		err := Search(ctx, fs, "/docs", "report", checker, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		assert.Empty(t, matched)
	})

	t.Run("context cancellation aborts search", func(t *testing.T) {
		fs := setupFs()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := Search(ctx, fs, "/", "anything", allowAllChecker{}, func(p string, f os.FileInfo) error {
			return nil
		})

		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("context timeout aborts search", func(t *testing.T) {
		fs := setupFs()
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(1 * time.Millisecond)

		err := Search(ctx, fs, "/", "anything", allowAllChecker{}, func(p string, f os.FileInfo) error {
			return nil
		})

		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("empty fs or no matches returns empty and no error", func(t *testing.T) {
		emptyFs := afero.NewMemMapFs()
		_ = emptyFs.MkdirAll("/root", 0o755)

		var matched []string
		err := Search(context.Background(), emptyFs, "/root", "nonexistent", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		assert.Empty(t, matched)
	})

	t.Run("scope itself is not matched", func(t *testing.T) {
		fs := setupFs()
		var matched []string
		// Search query matching folder name "work"
		err := Search(context.Background(), fs, "/docs/work", "work", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		// Should only find children inside /docs/work, not /docs/work itself
		assert.NotContains(t, matched, "")
		assert.NotContains(t, matched, ".")
	})

	t.Run("consecutive whitespace query does not match unrelated files", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_ = afero.WriteFile(fs, "/alpha_file.txt", []byte("a"), 0o644)
		_ = afero.WriteFile(fs, "/beta_file.txt", []byte("b"), 0o644)
		_ = afero.WriteFile(fs, "/unrelated.txt", []byte("u"), 0o644)

		var matched []string
		err := Search(context.Background(), fs, "/", "   alpha     beta   ", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})

		require.NoError(t, err)
		assert.Contains(t, matched, "alpha_file.txt")
		assert.Contains(t, matched, "beta_file.txt")
		assert.NotContains(t, matched, "unrelated.txt", "consecutive spaces must not generate empty tokens matching unrelated files")
	})
}
