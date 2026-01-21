package fs

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	PrefixTailwind = "tailwindcss-"
)

func Write(logger *slog.Logger, reader io.Reader, path string, downloadDir string, expectedSize int64) error {
	logger.Debug("Writing file", "path", path, "expectedSize", expectedSize)

	// Validate path is within download directory
	cleanPath := filepath.Clean(path)
	cleanDir := filepath.Clean(downloadDir)
	if !strings.HasPrefix(cleanPath, cleanDir+string(filepath.Separator)) {
		return ErrInvalidPath
	}

	f, err := os.Create(cleanPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			logger.Error("failed to close file", "error", closeErr)
		}
	}()

	written, err := io.Copy(f, reader)
	if err != nil {
		return err
	}

	// Validate file size if Content-Length was provided
	if expectedSize > 0 && written != expectedSize {
		return fmt.Errorf("%w: expected %d bytes, got %d bytes", ErrIncompleteDownload, expectedSize, written)
	}

	logger.Debug("File written successfully", "path", path, "bytes", written)
	return nil
}

func Exists(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotExists
		}
		return err
	}
	return nil
}

var ErrFileNotExists = errors.New("file does not exist")
var ErrNotInstalled = errors.New("tailwindcss is not currently installed")
var ErrInvalidPath = errors.New("invalid path: attempting to write outside cache directory")
var ErrIncompleteDownload = errors.New("incomplete download")

func GetCurrentVersion(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if s, hasPrefix := strings.CutPrefix(entry.Name(), PrefixTailwind); hasPrefix {
			// Remove .exe extension if present (Windows)
			version := strings.TrimSuffix(s, ".exe")
			return version, nil
		}
	}

	return "", ErrNotInstalled
}

func GetDownloadDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	p := filepath.Join(cacheDir, "go-tw")

	if err = os.MkdirAll(p, 0750); err != nil {
		return "", err
	}

	return p, nil
}

func DeleteOtherVersions(logger *slog.Logger, downloadDir string, version string) error {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), PrefixTailwind) {
			// Remove .exe extension if present (Windows) for version comparison
			fileName := strings.TrimSuffix(entry.Name(), ".exe")
			if !strings.HasSuffix(fileName, "-"+version) {
				logger.Debug("Deleting old version", "file", entry.Name(), "dir", downloadDir)
				if err = os.Remove(filepath.Join(downloadDir, entry.Name())); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func MakeExecutable(path string) error {
	//nolint:gosec
	// Files needs to be exexuted
	err := os.Chmod(path, 0700)
	if err != nil {
		return err
	}
	return nil
}
