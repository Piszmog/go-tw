//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIntegration runs cross-platform integration tests that verify go-tw can:
// 1. Build successfully on the current platform
// 2. Download the Tailwind CSS binary
// 3. Process CSS files correctly
// 4. Generate valid output
func TestIntegration(t *testing.T) {
	t.Run("BasicCompilation", testBasicCompilation)
}

func testBasicCompilation(t *testing.T) {
	// Create a temporary directory for this test
	tmpDir := t.TempDir()

	// Build the go-tw binary
	binaryPath := buildGoTw(t, tmpDir)

	// Create test directory structure
	stylesDir := filepath.Join(tmpDir, "styles")
	err := os.Mkdir(stylesDir, 0750)
	require.NoError(t, err)

	// Copy input.css to temp directory
	inputCSS := filepath.Join("testdata", "integration", "styles", "input.css")
	destInputCSS := filepath.Join(stylesDir, "input.css")
	copyFile(t, inputCSS, destInputCSS)

	// Define output path
	outputCSS := filepath.Join(tmpDir, "output.css")

	// Run go-tw to compile the CSS
	t.Logf("Running: %s -i %s -o %s", binaryPath, destInputCSS, outputCSS)
	runGoTw(t, binaryPath, "-i", destInputCSS, "-o", outputCSS)

	// Validate the output
	validateCSSOutput(t, outputCSS)
}

// buildGoTw builds the go-tw binary and returns its path
func buildGoTw(t *testing.T, outputDir string) string {
	t.Helper()

	binaryName := "go-tw"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	binaryPath := filepath.Join(outputDir, binaryName)

	t.Logf("Building go-tw binary: %s", binaryPath)
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build go-tw: %s", string(output))

	// Verify binary exists
	require.FileExists(t, binaryPath)

	return binaryPath
}

// runGoTw executes the go-tw binary with the given arguments
func runGoTw(t *testing.T, binary string, args ...string) {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), "LOG_LEVEL=debug")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("go-tw output: %s", string(output))
	}
	require.NoError(t, err, "go-tw execution failed: %s", string(output))
}

// validateCSSOutput verifies that the generated CSS file exists and has content
func validateCSSOutput(t *testing.T, outputPath string) {
	t.Helper()

	// Verify file exists
	require.FileExists(t, outputPath, "Output CSS file should exist")

	// Read the file to verify it has content
	//nolint:gosec // G304: Reading from test temp file, safe
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err, "Should be able to read output CSS file")
	require.NotEmpty(t, content, "Output CSS file should not be empty")

	t.Logf("✓ Output CSS file generated successfully (%d bytes)", len(content))
}

// copyFile copies a file from src to dst
func copyFile(t *testing.T, src, dst string) {
	t.Helper()

	//nolint:gosec // G304: Reading from test fixture file, safe
	content, err := os.ReadFile(src)
	require.NoError(t, err, "Failed to read source file: %s", src)

	err = os.WriteFile(dst, content, 0600)
	require.NoError(t, err, "Failed to write destination file: %s", dst)
}
