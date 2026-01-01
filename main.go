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

func main() {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)

	operatingSystem := runtime.GOOS
	arch := runtime.GOARCH

	logger.Debug("Running platform", "os", operatingSystem, "arch", arch)
	if !isSupported(operatingSystem, arch) {
		fmt.Printf("OS '%s' and arch '%s' is not supported\n", operatingSystem, arch)
		return
	}

	c := client.New(logger, 30*time.Second)

	version, args, err := getArgs()
	if err != nil {
		fmt.Println("failed to parse arguments: ", err)
		return
	}

	downloadDir, err := fs.GetDownloadDir()
	if err != nil {
		fmt.Println("failed to determine directory to download tailwind to: ", err)
		return
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
					fmt.Println("failed to check for latest version of tailwind and no version is installed: ", currErr)
					return
				}
				fmt.Println("failed to fetch latest tailwindcss version: falling back to installed version " + currVer)
				actualVersion = currVer
			} else {
				fmt.Println("failed to determine latest version", verErr)
				return
			}
		} else {
			logger.Debug("Retrieved latest version", "version", ver)
			actualVersion = ver
		}
	}

	filePath := filepath.Join(downloadDir, fs.PrefixTailwind+actualVersion)

	exists := true
	err = fs.Exists(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrFileNotExists) {
			exists = false
		} else {
			fmt.Println("failed to check if tailwind is already installed: ", err)
			return
		}
	}

	if !exists {
		fmt.Println("Downloading tailwindcss " + actualVersion)
		if err = c.Download(ctx, operatingSystem, arch, actualVersion, filePath, downloadDir); err != nil {
			fmt.Println("failed to download tailwind: ", err)
			return
		}
		if err = fs.MakeExecutable(filePath); err != nil {
			fmt.Println("failed to make tailwind executable: ", err)
			return
		}
		if err = fs.DeleteOtherVersions(logger, downloadDir, actualVersion); err != nil {
			fmt.Println("failed to delete older version: ", err)
			return
		}
	}

	if err := run(ctx, logger, filePath, args); err != nil {
		fmt.Println("failed to run tailwind: ", err)
	}
}

func isSupported(os string, arch string) bool {
	switch os {
	case "windows", "darwin", "linux":
		return arch == "amd64" || arch == "arm64"
	default:
		return false
	}
}

func getArgs() (string, []string, error) {
	args := os.Args[1:]

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
	cmd := exec.CommandContext(ctx, path, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	outStr := stdout.String()
	errStr := stderr.String()

	logger.Debug("Command", "out", outStr, "err", errStr)

	if len(outStr) > 0 && len(args) > 0 {
		fmt.Println(outStr)
	} else {
		fmt.Println(errStr)
	}

	return nil
}
