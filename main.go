package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Piszmog/go-tw/log"
)

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

	version, args, err := getArgs()
	if err != nil {
		fmt.Println("failed to parse arguments: ", err)
		return
	}

	actualVersion := version
	if version == "latest" {
		ver, verErr := getLatestVersion()
		if verErr != nil {
			fmt.Println("failed to determine latest version", verErr)
			return
		}
		logger.Debug("Retrieved latest version", "version", ver)
		actualVersion = ver
	}

	downloadDir, err := getDownloadDir()
	if err != nil {
		fmt.Println("failed to determine directory to download tailwind to: ", err)
		return
	}

	filePath := filepath.Join(downloadDir, "tailwindcss-"+actualVersion)

	if installed, err := isInstalled(filePath); err != nil {
		fmt.Println("failed to check if tailwind is already installed: ", err)
		return
	} else if !installed {
		downloadErr := download(logger, operatingSystem, arch, filePath, actualVersion)
		if downloadErr != nil {
			fmt.Println("failed to download tailwind: ", downloadErr)
			return
		}

		if exeErr := makeExecutable(filePath); exeErr != nil {
			fmt.Println("failed to make tailwind executable: ", exeErr)
			return
		}

		if delErr := deleteOtherVersions(logger, downloadDir, actualVersion); delErr != nil {
			fmt.Println("failed to delete older version: ", delErr)
			return
		}
	}

	if err := run(logger, filePath, args); err != nil {
		fmt.Println("failed to run tailwind: ", err)
	}
}

func isSupported(os string, arch string) bool {
	switch os {
	case "windows":
		if arch != "amd64" {
			return true
		}
	case "darwin":
		if arch == "amd64" || arch == "arm64" {
			return true
		}
	case "linux":
		if arch == "amd64" || arch == "arm64" {
			return true
		}
	default:
		return false
	}
	return false
}

func getArgs() (string, []string, error) {
	args := os.Args[1:]

	var filteredArgs []string
	version := "latest"

	for i := 0; i < len(args); i++ {
		if args[i] == "-version" {
			if i+1 >= len(args) {
				return "", nil, errors.New("version flag passed but missing argument")
			}
			version = args[i+1]
			i++
			continue
		}
		filteredArgs = append(filteredArgs, args[i])
	}
	return version, filteredArgs, nil
}

func isInstalled(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func download(logger *slog.Logger, operatingSystem string, arch string, path string, version string) error {
	fileName := getName(operatingSystem, arch)
	url := "https://github.com/tailwindlabs/tailwindcss/releases/download/" + version + "/" + fileName

	logger.Debug("Downloading file", "url", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to download: " + resp.Status)
	}

	logger.Debug("Writing file", "path", path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func getLatestVersion() (string, error) {
	url := "https://api.github.com/repos/tailwindlabs/tailwindcss/releases/latest"

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release release
	if err = json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

type release struct {
	TagName string `json:"tag_name"`
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

func makeExecutable(path string) error {
	err := os.Chmod(path, 0755)
	if err != nil {
		return err
	}
	return nil
}

func deleteOtherVersions(logger *slog.Logger, downloadDir string, version string) error {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "tailwindcss-") && !strings.HasSuffix(entry.Name(), "-"+version) {
			logger.Debug("Deleting old version", "file", entry.Name(), "dir", downloadDir)
			if err = os.Remove(filepath.Join(downloadDir, entry.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func run(logger *slog.Logger, path string, args []string) error {
	logger.Debug("Running command", "path", path, "args", args)
	cmd := exec.Command(path, args...)

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

func getDownloadDir() (string, error) {
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
