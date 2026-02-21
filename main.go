package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Piszmog/go-tw/client"
	"github.com/Piszmog/go-tw/fs"
	"github.com/Piszmog/go-tw/log"
)

var ErrMissingVersionArg = errors.New("version flag passed but missing argument")
var ErrUnsupportedPlatform = errors.New("unsupported platform")

func main() {
	if err := execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

//nolint:cyclop // linear flow with early returns; splitting would obscure the sequence
func execute() error {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)

	operatingSystem := runtime.GOOS
	arch := runtime.GOARCH

	logger.Debug("Running platform", "os", operatingSystem, "arch", arch)
	if !IsSupported(operatingSystem, arch) {
		return fmt.Errorf("%w: OS '%s' and arch '%s'", ErrUnsupportedPlatform, operatingSystem, arch)
	}

	c := client.New(logger, 3*time.Minute)

	version, args, err := GetArgs(os.Args[1:])
	if err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	downloadDir, err := fs.GetDownloadDir()
	if err != nil {
		return fmt.Errorf("failed to determine directory to download tailwind to: %w", err)
	}
	ctx := context.Background()

	actualVersion := version
	//nolint:nestif
	if version == "latest" {
		ver, verErr := c.GetLatestVersion(ctx)
		if verErr != nil {
			if errors.Is(verErr, client.ErrHTTP) {
				currVer, currErr := fs.GetCurrentVersion(downloadDir)
				if currErr != nil {
					return fmt.Errorf("failed to check for latest version of tailwind and no version is installed: %w", currErr)
				}
				fmt.Println("failed to fetch latest tailwindcss version: falling back to installed version " + currVer)
				actualVersion = currVer
			} else {
				return fmt.Errorf("failed to determine latest version: %w", verErr)
			}
		} else {
			logger.Debug("Retrieved latest version", "version", ver)
			actualVersion = ver
		}
	}

	fileName := fs.PrefixTailwind + actualVersion
	if operatingSystem == "windows" {
		fileName += ".exe"
	}
	filePath := filepath.Join(downloadDir, fileName)

	exists := true
	err = fs.Exists(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrFileNotExists) {
			exists = false
		} else {
			return fmt.Errorf("failed to check if tailwind is already installed: %w", err)
		}
	}

	if !exists {
		fmt.Println("Downloading tailwindcss " + actualVersion)
		if err = c.Download(ctx, operatingSystem, arch, actualVersion, filePath, downloadDir); err != nil {
			return fmt.Errorf("failed to download tailwind: %w", err)
		}
		if err = fs.MakeExecutable(filePath); err != nil {
			return fmt.Errorf("failed to make tailwind executable: %w", err)
		}
		if err = fs.DeleteOtherVersions(logger, downloadDir, actualVersion); err != nil {
			return fmt.Errorf("failed to delete older version: %w", err)
		}
	}

	if err := run(ctx, logger, filePath, args); err != nil {
		return fmt.Errorf("failed to run tailwind: %w", err)
	}
	return nil
}

// IsSupported checks if the given OS and architecture combination is supported
func IsSupported(os string, arch string) bool {
	switch os {
	case "windows", "darwin", "linux":
		return arch == "amd64" || arch == "arm64"
	default:
		return false
	}
}

// GetArgs parses command line arguments and extracts the version flag
func GetArgs(args []string) (string, []string, error) {
	var filteredArgs []string
	version := "latest"

	for i := 0; i < len(args); i++ {
		if args[i] == "-version" {
			if i+1 >= len(args) {
				return "", nil, ErrMissingVersionArg
			}
			version = args[i+1]
			i++
			continue
		}
		filteredArgs = append(filteredArgs, args[i])
	}
	return version, filteredArgs, nil
}

func run(ctx context.Context, logger *slog.Logger, path string, args []string) error {
	logger.Debug("Running command", "path", path, "args", args)
	cmd := exec.CommandContext(ctx, path, args...) //nolint:gosec // G204: path is the downloaded tailwindcss binary, not user input

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	outStr := stdout.String()
	errStr := stderr.String()

	logger.Debug("Command", "out", outStr, "err", errStr)

	if err != nil {
		return err
	}

	if len(outStr) > 0 && len(args) > 0 {
		fmt.Println(outStr)
	} else {
		fmt.Println(errStr)
	}

	return nil
}
