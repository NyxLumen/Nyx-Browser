package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearch_DeepHierarchies tests search traversal through deeply nested directories (100 levels).
func TestSearch_DeepHierarchies(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Build a 100-level deep directory chain: /d00/d01/d02/.../d99
	current := "/"
	for i := 0; i < 100; i++ {
		current = filepath.Join(current, fmt.Sprintf("d%02d", i))
		if i == 25 {
			require.NoError(t, afero.WriteFile(fs, filepath.Join(current, "level25_marker.txt"), []byte("25"), 0o644))
		}
		if i == 50 {
			require.NoError(t, afero.WriteFile(fs, filepath.Join(current, "level50_target.txt"), []byte("50"), 0o644))
		}
	}
	deepestFile := filepath.Join(current, "deepest_leaf.txt")
	require.NoError(t, afero.WriteFile(fs, deepestFile, []byte("leaf"), 0o644))

	t.Run("find file at depth 100 from root", func(t *testing.T) {
		var matched []string
		err := Search(context.Background(), fs, "/", "deepest_leaf", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, matched, 1)
		assert.True(t, strings.HasSuffix(matched[0], "deepest_leaf.txt"))
		// Count directory slashes in relative path
		assert.Equal(t, 100, strings.Count(matched[0], "/"))
	})

	t.Run("find intermediate file at depth 50 scoped from depth 25", func(t *testing.T) {
		// Calculate prefix path up to d25
		parts := []string{}
		for i := 0; i <= 25; i++ {
			parts = append(parts, fmt.Sprintf("d%02d", i))
		}
		scopeD25 := "/" + filepath.Join(parts...)

		var matched []string
		err := Search(context.Background(), fs, scopeD25, "level50_target", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, matched, 1)
		// Relative path from depth 25 to depth 50 should have 25 path segments
		assert.Equal(t, 25, strings.Count(matched[0], "/"))
	})
}

// TestSearch_UnicodeAndSpecialCharacters tests Unicode scripts, emojis, and special regex/shell tokens in filenames.
func TestSearch_UnicodeAndSpecialCharacters(t *testing.T) {
	fs := afero.NewMemMapFs()

	unicodeFiles := map[string]string{
		"/docs/日本語_ドキュメント.txt":               "japanese",
		"/docs/中文_测试.pdf":                     "chinese",
		"/docs/한국어_파일.md":                     "korean",
		"/docs/документ_секрет.docx":             "cyrillic",
		"/docs/résumé_café_naïve.txt":            "latin_accents",
		"/docs/RÉSUMÉ_UPPERCASE.txt":             "latin_accents_upper",
		"/docs/🎉_party_rocket_🚀.png":            "emojis",
		"/docs/עברית_שלום.txt":                   "hebrew",
		"/regex/file.*name.txt":                  "regex_dot_star",
		"/regex/file[1-3].txt":                   "regex_brackets",
		"/regex/file+name(1).tar.gz":             "regex_plus_parens",
		"/regex/^start$end.log":                  "regex_anchors",
		"/regex/foo?bar.bin":                     "regex_question",
		"/quotes/file\"quote\".txt":              "double_quote",
		"/quotes/file'single'.txt":               "single_quote",
		"/whitespace/file with spaces.txt":       "spaces",
		"/whitespace/   leading_trailing   .txt": "leading_trailing_spaces",
	}

	for path, content := range unicodeFiles {
		dir := filepath.Dir(path)
		require.NoError(t, fs.MkdirAll(dir, 0o755))
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0o644))
	}

	testCases := []struct {
		name          string
		query         string
		expectedPaths []string
	}{
		{
			name:          "Japanese search",
			query:         "日本語",
			expectedPaths: []string{"docs/日本語_ドキュメント.txt"},
		},
		{
			name:          "Chinese search",
			query:         "中文",
			expectedPaths: []string{"docs/中文_测试.pdf"},
		},
		{
			name:          "Korean search",
			query:         "한국어",
			expectedPaths: []string{"docs/한국어_파일.md"},
		},
		{
			name:          "Cyrillic search",
			query:         "документ",
			expectedPaths: []string{"docs/документ_секрет.docx"},
		},
		{
			name:          "Emoji search",
			query:         "🚀",
			expectedPaths: []string{"docs/🎉_party_rocket_🚀.png"},
		},
		{
			name:          "Hebrew search",
			query:         "שלום",
			expectedPaths: []string{"docs/עברית_שלום.txt"},
		},
		{
			name:          "Accented Latin case-insensitive",
			query:         "résumé",
			expectedPaths: []string{"docs/résumé_café_naïve.txt", "docs/RÉSUMÉ_UPPERCASE.txt"},
		},
		{
			name:          "Accented Latin case-sensitive lowercase",
			query:         "case:sensitive résumé",
			expectedPaths: []string{"docs/résumé_café_naïve.txt"},
		},
		{
			name:          "Accented Latin case-sensitive uppercase",
			query:         "case:sensitive RÉSUMÉ",
			expectedPaths: []string{"docs/RÉSUMÉ_UPPERCASE.txt"},
		},
		{
			name:          "Regex token .* treated as literal",
			query:         ".*",
			expectedPaths: []string{"regex/file.*name.txt"},
		},
		{
			name:          "Regex token [1-3] treated as literal",
			query:         "[1-3]",
			expectedPaths: []string{"regex/file[1-3].txt"},
		},
		{
			name:          "Regex token + and parens treated as literal",
			query:         "name(1)",
			expectedPaths: []string{"regex/file+name(1).tar.gz"},
		},
		{
			name:          "Regex anchors ^ and $ treated as literal",
			query:         "^start$end",
			expectedPaths: []string{"regex/^start$end.log"},
		},
		{
			name:          "Question mark treated as literal",
			query:         "foo?bar",
			expectedPaths: []string{"regex/foo?bar.bin"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var matched []string
			err := Search(context.Background(), fs, "/", tc.query, allowAllChecker{}, func(p string, f os.FileInfo) error {
				matched = append(matched, p)
				return nil
			})
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expectedPaths, matched)
		})
	}
}

// TestSearch_WhitespaceAndParsingEdgeCases tests whitespace handling, multiple terms, and empty token anomalies.
func TestSearch_WhitespaceAndParsingEdgeCases(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/alpha.txt", []byte("a"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/beta.txt", []byte("b"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/gamma.txt", []byte("g"), 0o644))

	t.Run("multiple consecutive spaces in query", func(t *testing.T) {
		opts := parseSearch("alpha    beta")
		// Document the actual parsed terms
		t.Logf("Parsed terms for 'alpha    beta': %#v", opts.Terms)

		var matched []string
		err := Search(context.Background(), fs, "/", "alpha    beta", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})
		require.NoError(t, err)

		// Verify that consecutive whitespace does NOT create empty string terms or match unrelated files (gamma.txt)
		assert.Equal(t, []string{"alpha", "beta"}, opts.Terms)
		assert.NotContains(t, opts.Terms, "")
		assert.Contains(t, matched, "alpha.txt")
		assert.Contains(t, matched, "beta.txt")
		assert.NotContains(t, matched, "gamma.txt", "consecutive spaces must not match unrelated files")
	})

	t.Run("query with double quotes around empty string", func(t *testing.T) {
		opts := parseSearch(`""`)
		t.Logf("Parsed terms for '\"\"': %#v", opts.Terms)

		var matched []string
		err := Search(context.Background(), fs, "/", `""`, allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})
		require.NoError(t, err)
		t.Logf("Matched files for '\"\"': %v", matched)
	})
}

// TestSearch_MIMEFilterAdversarial tests MIME filter edge cases.
func TestSearch_MIMEFilterAdversarial(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/photo.png", []byte("png"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/video.mp4", []byte("mp4"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/music.flac", []byte("flac"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/document.pdf", []byte("pdf"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/code.go", []byte("go"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/NO_EXTENSION", []byte("raw"), 0o644))

	t.Run("unknown type condition", func(t *testing.T) {
		var matched []string
		err := Search(context.Background(), fs, "/", "type:nonexistent_format_xyz", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})
		require.NoError(t, err)
		assert.Empty(t, matched, "unknown type should match nothing")
	})

	t.Run("multiple MIME conditions disjunction", func(t *testing.T) {
		var matched []string
		err := Search(context.Background(), fs, "/", "type:image type:video", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"photo.png", "video.mp4"}, matched)
	})

	t.Run("case variation in type token TYPE:IMAGE", func(t *testing.T) {
		opts := parseSearch("TYPE:IMAGE")
		t.Logf("Parsed terms for 'TYPE:IMAGE': Conditions=%d, Terms=%v", len(opts.Conditions), opts.Terms)
		// typeRegexp only matches lowercase 'type:' because of `type:(\w+)`
		// Verify behavior: does it get parsed as a term or a condition?
		var matched []string
		err := Search(context.Background(), fs, "/", "TYPE:IMAGE", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})
		require.NoError(t, err)
		t.Logf("Matched for TYPE:IMAGE: %v", matched)
	})

	t.Run("file without extension does not match type conditions", func(t *testing.T) {
		var matched []string
		err := Search(context.Background(), fs, "/", "type:image", allowAllChecker{}, func(p string, f os.FileInfo) error {
			matched = append(matched, p)
			return nil
		})
		require.NoError(t, err)
		assert.NotContains(t, matched, "NO_EXTENSION")
	})
}

// TestSearch_CancellationAndTimeoutStress verifies context cancellation behavior under stress.
func TestSearch_CancellationAndTimeoutStress(t *testing.T) {
	fs := afero.NewMemMapFs()
	for i := 0; i < 500; i++ {
		p := fmt.Sprintf("/dir%d/file%d.txt", i/10, i)
		_ = fs.MkdirAll(filepath.Dir(p), 0o755)
		_ = afero.WriteFile(fs, p, []byte("test"), 0o644)
	}

	t.Run("cancellation mid-traversal terminates early", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		count := 0
		err := Search(ctx, fs, "/", "file", allowAllChecker{}, func(p string, f os.FileInfo) error {
			count++
			if count == 10 {
				cancel()
			}
			return nil
		})
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.LessOrEqual(t, count, 15, "search must halt promptly when context is canceled")
	})

	t.Run("short timeout aborts before full traversal", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Microsecond)
		defer cancel()
		time.Sleep(20 * time.Microsecond)

		err := Search(ctx, fs, "/", "file", allowAllChecker{}, func(p string, f os.FileInfo) error {
			return nil
		})
		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
