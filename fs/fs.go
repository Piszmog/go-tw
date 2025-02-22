package fs

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	PrefixTailwind = "tailwindcss-"
)

func Write(logger *slog.Logger, reader io.Reader, path string) error {
	logger.Debug("Writing file", "path", path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	return err

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

func GetCurrentVersion(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), PrefixTailwind) {
			return strings.TrimPrefix(entry.Name(), PrefixTailwind), nil
		}
	}

	return "", errors.New("tailwindcss is not currently installed")
}

func GetDownloadDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	p := filepath.Join(cacheDir, "go-tw")

	if err = os.MkdirAll(p, 0755); err != nil {
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

		if strings.HasPrefix(entry.Name(), PrefixTailwind) && !strings.HasSuffix(entry.Name(), "-"+version) {
			logger.Debug("Deleting old version", "file", entry.Name(), "dir", downloadDir)
			if err = os.Remove(filepath.Join(downloadDir, entry.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func MakeExecutable(path string) error {
	err := os.Chmod(path, 0755)
	if err != nil {
		return err
	}
	return nil
}
