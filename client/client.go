package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Piszmog/go-tw/fs"
)

const (
	urlDownload      = "https://github.com/tailwindlabs/tailwindcss/releases/download"
	urlLatestVersion = "https://api.github.com/repos/tailwindlabs/tailwindcss/releases/latest"
	maxRetries       = 3
	retryDelay       = 2 * time.Second
)

type Client struct {
	logger           *slog.Logger
	c                *http.Client
	downloadURL      string
	latestVersionURL string
}

func New(logger *slog.Logger, timeout time.Duration) *Client {
	return &Client{
		logger:           logger,
		c:                &http.Client{Timeout: timeout},
		downloadURL:      urlDownload,
		latestVersionURL: urlLatestVersion,
	}
}

// WithTestURLs allows injecting custom URLs for testing purposes
func (c *Client) WithTestURLs(downloadURL, latestVersionURL string) *Client {
	c.downloadURL = downloadURL
	c.latestVersionURL = latestVersionURL
	return c
}

func (c *Client) Download(ctx context.Context, operatingSystem string, arch string, version string, path string, downloadDir string) error {
	fileName := GetName(operatingSystem, arch)
	url := c.downloadURL + "/" + version + "/" + fileName

	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			c.logger.Info("Download failed, retrying", "attempt", attempt, "max", maxRetries)
			time.Sleep(retryDelay)
		}

		err := c.downloadAttempt(ctx, url, path, downloadDir)
		if err == nil {
			return nil // Success!
		}

		lastErr = err
		c.logger.Info("Download attempt failed", "attempt", attempt, "error", err)

		// Clean up partial file
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			c.logger.Error("Failed to clean up partial download", "path", path, "error", removeErr)
		}
	}

	return fmt.Errorf("%w: %w", ErrDownloadFailed, lastErr)
}

// FileReader is an interface for reading file contents and checking file existence
type FileReader interface {
	ReadFile(path string) ([]byte, error)
	FileExists(path string) bool
}

// osFileReader implements FileReader using os package functions
type osFileReader struct{}

func (o osFileReader) ReadFile(path string) ([]byte, error) {
	//nolint:gosec // G304: path is not user-controlled, only called with hardcoded paths
	return os.ReadFile(path)
}

func (o osFileReader) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// defaultFileReader is used in production
var defaultFileReader FileReader = osFileReader{}

// muslLinkers are the well-known paths for the musl dynamic linker on Linux
var muslLinkers = []string{
	"/lib/ld-musl-x86_64.so.1",
	"/lib/ld-musl-aarch64.so.1",
	"/lib/ld-musl-armhf.so.1",
}

func isMusl(reader FileReader) bool {
	// Strategy 1: check /proc/self/maps for "musl".
	// Works when the binary is dynamically linked (CGO_ENABLED=1).
	if data, err := reader.ReadFile("/proc/self/maps"); err == nil {
		if strings.Contains(string(data), "musl") {
			return true
		}
	}

	// Strategy 2: check for the musl dynamic linker at well-known paths.
	// Works for statically compiled Go binaries (CGO_ENABLED=0) on musl systems.
	return slices.ContainsFunc(muslLinkers, reader.FileExists)
}

// GetName generates the tailwindcss binary filename for the given OS and architecture
func GetName(os string, arch string) string {
	return GetNameWithReader(os, arch, defaultFileReader)
}

// GetNameWithReader generates the tailwindcss binary filename, using the provided FileReader
// for musl detection. Useful for testing.
func GetNameWithReader(os string, arch string, reader FileReader) string {
	muslPostfix := ""
	if os == "linux" && isMusl(reader) {
		muslPostfix = "-musl"
	}

	osName := os
	if osName == "darwin" {
		osName = "macos"
	}

	archName := arch
	if archName == "amd64" {
		archName = "x64"
	}

	executablePostfix := ""
	if os == "windows" {
		executablePostfix = ".exe"
	}

	return "tailwindcss-" + osName + "-" + archName + muslPostfix + executablePostfix
}

func (c *Client) GetLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.latestVersionURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.c.Do(req) //nolint:gosec // G704: URL is a hardcoded GitHub API constant, not user input
	if err != nil {
		return "", ErrHTTP
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("failed to close body", "error", err)
		}
	}()

	var release release
	if err = json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func (c *Client) downloadAttempt(ctx context.Context, url string, path string, downloadDir string) error {
	c.logger.Debug("Downloading file", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.c.Do(req) //nolint:gosec // G704: URL is derived from a hardcoded GitHub releases constant, not user input
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("failed to close body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("failed to download file", "status_code", resp.StatusCode)
		return ErrHTTP
	}

	// Pass Content-Length for size validation
	expectedSize := resp.ContentLength
	return fs.Write(c.logger, resp.Body, path, downloadDir, expectedSize)
}

var ErrHTTP = errors.New("failed to get the resource")
var ErrDownloadFailed = errors.New("failed to download after multiple attempts")

type release struct {
	TagName string `json:"tag_name"`
}
