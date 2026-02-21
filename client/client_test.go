package client_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Piszmog/go-tw/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test logger that discards output
func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// isMuslEnvironment mirrors the production musl detection logic so TestGetName
// can compute the correct expected value for Linux regardless of whether the
// test runs on a glibc or musl host.
func isMuslEnvironment() bool {
	if data, err := os.ReadFile("/proc/self/maps"); err == nil {
		if strings.Contains(string(data), "musl") {
			return true
		}
	}
	for _, path := range []string{
		"/lib/ld-musl-x86_64.so.1",
		"/lib/ld-musl-aarch64.so.1",
		"/lib/ld-musl-armhf.so.1",
	} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func TestGetName(t *testing.T) {
	t.Parallel()
	linuxSuffix := ""
	if isMuslEnvironment() {
		linuxSuffix = "-musl"
	}
	tests := []struct {
		name     string
		os       string
		arch     string
		expected string
	}{
		{"Linux AMD64", "linux", "amd64", "tailwindcss-linux-x64" + linuxSuffix},
		{"Linux ARM64", "linux", "arm64", "tailwindcss-linux-arm64" + linuxSuffix},
		{"Darwin AMD64", "darwin", "amd64", "tailwindcss-macos-x64"},
		{"Darwin ARM64", "darwin", "arm64", "tailwindcss-macos-arm64"},
		{"Windows AMD64", "windows", "amd64", "tailwindcss-windows-x64.exe"},
		{"Windows ARM64", "windows", "arm64", "tailwindcss-windows-arm64.exe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := client.GetName(tt.os, tt.arch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	logger := testLogger()
	timeout := 30 * time.Second

	c := client.New(logger, timeout)

	require.NotNil(t, c)
	assert.IsType(t, &client.Client{}, c)
}

func TestGetLatestVersion(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			response := map[string]string{"tag_name": "v4.0.0"}
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		c := client.New(testLogger(), 30*time.Second).WithTestURLs("", server.URL)

		version, err := c.GetLatestVersion(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "v4.0.0", version)
	})

	t.Run("HTTP Error - Internal Server Error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := client.New(testLogger(), 30*time.Second).WithTestURLs("", server.URL)

		_, err := c.GetLatestVersion(context.Background())

		assert.Error(t, err)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		c := client.New(testLogger(), 30*time.Second).WithTestURLs("", server.URL)

		_, err := c.GetLatestVersion(context.Background())

		assert.Error(t, err)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		c := client.New(testLogger(), 30*time.Second).WithTestURLs("", server.URL)

		_, err := c.GetLatestVersion(ctx)

		assert.Error(t, err)
	})
}

func TestDownload(t *testing.T) {
	t.Parallel()
	t.Run("Successful download", func(t *testing.T) {
		t.Parallel()
		content := []byte("fake tailwindcss binary content here")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "tailwindcss-test")

		c := client.New(testLogger(), 30*time.Second).WithTestURLs(server.URL, "")

		err := c.Download(context.Background(), "linux", "amd64", "v4.0.0", filePath, tmpDir)

		require.NoError(t, err)

		// Verify file was written
		//nolint:gosec // G304: Reading from test temp file, safe
		written, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, written)
	})

	t.Run("HTTP Error triggers retry", func(t *testing.T) {
		t.Parallel()
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "tailwindcss-test")

		c := client.New(testLogger(), 30*time.Second).WithTestURLs(server.URL, "")

		err := c.Download(context.Background(), "linux", "amd64", "v4.0.0", filePath, tmpDir)

		require.Error(t, err)
		require.ErrorIs(t, err, client.ErrDownloadFailed)
		// Should have retried 3 times
		assert.Equal(t, 3, attemptCount)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "tailwindcss-test")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		c := client.New(testLogger(), 30*time.Second).WithTestURLs(server.URL, "")

		err := c.Download(ctx, "linux", "amd64", "v4.0.0", filePath, tmpDir)

		assert.Error(t, err)
	})

	t.Run("Invalid path - outside directory", func(t *testing.T) {
		t.Parallel()
		content := []byte("fake content")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		invalidPath := "/tmp/malicious.bin"

		c := client.New(testLogger(), 30*time.Second).WithTestURLs(server.URL, "")

		err := c.Download(context.Background(), "linux", "amd64", "v4.0.0", invalidPath, tmpDir)

		assert.Error(t, err)
	})
}
