package client

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/Piszmog/go-tw/fs"
)

const (
	urlDownload      = "https://github.com/tailwindlabs/tailwindcss/releases/download"
	urlLatestVersion = "https://api.github.com/repos/tailwindlabs/tailwindcss/releases/latest"
)

type Client struct {
	logger *slog.Logger
	c      *http.Client
}

func New(logger *slog.Logger) *Client {
	return &Client{
		logger: logger,
		c:      &http.Client{},
	}
}

func (c *Client) Download(ctx context.Context, operatingSystem string, arch string, version string, path string, downloadDir string) error {
	fileName := getName(operatingSystem, arch)
	url := urlDownload + "/" + version + "/" + fileName

	c.logger.Debug("Downloading file", "url", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.c.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("failed to close body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("failed to download file", "status_code", resp.StatusCode)
		return ErrHTTP
	}

	if resp.ContentLength <= 0 {
		return ErrInvalidContentLength
	}

	if err := fs.Write(
		c.logger,
		resp.Body,
		path,
		downloadDir,
		resp.ContentLength,
	); err != nil {
		_ = os.Remove(path)
		return err
	}

	return nil
}

func getName(os string, arch string) string {
	muslPostfix := ""
	if os == "linux" && isMusl() {
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

func isMusl() bool {
	data, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return false // Cannot determine, assume not musl
	}
	return strings.Contains(string(data), "musl")
}

func (c *Client) GetLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlLatestVersion, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.c.Do(req)
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

var ErrHTTP = errors.New("failed to get the resource")
var ErrInvalidContentLength = errors.New("invalid content length")

type release struct {
	TagName string `json:"tag_name"`
}
