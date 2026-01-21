package fs_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Piszmog/go-tw/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test logger that discards output
func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestWrite(t *testing.T) {
	t.Parallel()
	logger := testLogger()

	t.Run("Successful write", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.bin")
		content := []byte("test content")

		err := fs.Write(logger, bytes.NewReader(content), filePath, tmpDir, int64(len(content)))

		require.NoError(t, err)

		// Verify file contents
		//nolint:gosec // G304: Reading from test temp file, safe
		written, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, written)
	})

	t.Run("Invalid path - outside download directory", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		invalidPath := "/tmp/malicious.bin"

		err := fs.Write(logger, bytes.NewReader([]byte("data")), invalidPath, tmpDir, 4)

		assert.ErrorIs(t, err, fs.ErrInvalidPath)
	})

	t.Run("Path traversal attempt", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		maliciousPath := filepath.Join(tmpDir, "../../../etc/passwd")

		err := fs.Write(logger, bytes.NewReader([]byte("data")), maliciousPath, tmpDir, 4)

		assert.ErrorIs(t, err, fs.ErrInvalidPath)
	})

	t.Run("Size mismatch - too small", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.bin")
		content := []byte("short")

		err := fs.Write(logger, bytes.NewReader(content), filePath, tmpDir, 1000)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "incomplete download")
	})

	t.Run("Size mismatch - too large", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.bin")
		content := []byte("this is a very long string")

		err := fs.Write(logger, bytes.NewReader(content), filePath, tmpDir, 5)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "incomplete download")
	})

	t.Run("No size validation when expectedSize is 0", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.bin")
		content := []byte("any size")

		err := fs.Write(logger, bytes.NewReader(content), filePath, tmpDir, 0)

		require.NoError(t, err)
	})
}

func TestExists(t *testing.T) {
	t.Parallel()

	t.Run("File exists", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "exists.txt")
		err := os.WriteFile(filePath, []byte("content"), 0600)
		require.NoError(t, err)

		err = fs.Exists(filePath)
		assert.NoError(t, err)
	})

	t.Run("File does not exist", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nonexistent.txt")

		err := fs.Exists(filePath)
		assert.ErrorIs(t, err, fs.ErrFileNotExists)
	})
}

func TestGetCurrentVersion(t *testing.T) {
	t.Parallel()

	t.Run("Single version found", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		fileName := "tailwindcss-v4.0.0"
		err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte{}, 0600)
		require.NoError(t, err)

		version, err := fs.GetCurrentVersion(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "v4.0.0", version)
	})

	t.Run("Windows executable with .exe extension", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		fileName := "tailwindcss-v4.0.0.exe"
		err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte{}, 0600)
		require.NoError(t, err)

		version, err := fs.GetCurrentVersion(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "v4.0.0", version)
	})

	t.Run("No tailwindcss files", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "other-file.txt"), []byte{}, 0600)
		require.NoError(t, err)

		_, err = fs.GetCurrentVersion(tmpDir)
		assert.ErrorIs(t, err, fs.ErrNotInstalled)
	})

	t.Run("Empty directory", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		_, err := fs.GetCurrentVersion(tmpDir)
		assert.ErrorIs(t, err, fs.ErrNotInstalled)
	})

	t.Run("Ignores directories", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		err := os.Mkdir(filepath.Join(tmpDir, "tailwindcss-v4.0.0"), 0750)
		require.NoError(t, err)

		_, err = fs.GetCurrentVersion(tmpDir)
		assert.ErrorIs(t, err, fs.ErrNotInstalled)
	})
}

func TestGetDownloadDir(t *testing.T) {
	t.Parallel()

	t.Run("Creates download directory", func(t *testing.T) {
		t.Parallel()
		dir, err := fs.GetDownloadDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "go-tw")

		// Verify directory exists
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestDeleteOtherVersions(t *testing.T) {
	t.Parallel()

	logger := testLogger()

	t.Run("Deletes old versions", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create multiple versions
		v1 := filepath.Join(tmpDir, "tailwindcss-v3.0.0")
		v2 := filepath.Join(tmpDir, "tailwindcss-v4.0.0")
		v3 := filepath.Join(tmpDir, "tailwindcss-v5.0.0")

		for _, path := range []string{v1, v2, v3} {
			err := os.WriteFile(path, []byte{}, 0600)
			require.NoError(t, err)
		}

		err := fs.DeleteOtherVersions(logger, tmpDir, "v4.0.0")
		require.NoError(t, err)

		// v4.0.0 should exist
		assert.NoError(t, fs.Exists(v2))

		// Others should be deleted
		require.ErrorIs(t, fs.Exists(v1), fs.ErrFileNotExists)
		require.ErrorIs(t, fs.Exists(v3), fs.ErrFileNotExists)
	})

	t.Run("Handles Windows .exe files", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		v1 := filepath.Join(tmpDir, "tailwindcss-v3.0.0.exe")
		v2 := filepath.Join(tmpDir, "tailwindcss-v4.0.0.exe")

		for _, path := range []string{v1, v2} {
			err := os.WriteFile(path, []byte{}, 0600)
			require.NoError(t, err)
		}

		err := fs.DeleteOtherVersions(logger, tmpDir, "v4.0.0")
		require.NoError(t, err)

		assert.NoError(t, fs.Exists(v2))
		assert.ErrorIs(t, fs.Exists(v1), fs.ErrFileNotExists)
	})

	t.Run("Keeps non-tailwind files", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		tailwind := filepath.Join(tmpDir, "tailwindcss-v4.0.0")
		other := filepath.Join(tmpDir, "other-file.txt")

		for _, path := range []string{tailwind, other} {
			err := os.WriteFile(path, []byte{}, 0600)
			require.NoError(t, err)
		}

		err := fs.DeleteOtherVersions(logger, tmpDir, "v4.0.0")
		require.NoError(t, err)

		// Both should still exist
		assert.NoError(t, fs.Exists(tailwind))
		assert.NoError(t, fs.Exists(other))
	})

	t.Run("Ignores directories", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		file := filepath.Join(tmpDir, "tailwindcss-v4.0.0")
		dir := filepath.Join(tmpDir, "tailwindcss-v3.0.0")

		err := os.WriteFile(file, []byte{}, 0600)
		require.NoError(t, err)
		err = os.Mkdir(dir, 0750)
		require.NoError(t, err)

		err = fs.DeleteOtherVersions(logger, tmpDir, "v4.0.0")
		require.NoError(t, err)

		// Directory should still exist (not deleted)
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestMakeExecutable(t *testing.T) {
	t.Parallel()

	t.Run("Makes file executable", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.bin")
		err := os.WriteFile(filePath, []byte("content"), 0600)
		require.NoError(t, err)

		err = fs.MakeExecutable(filePath)
		require.NoError(t, err)

		// Check permissions
		info, err := os.Stat(filePath)
		require.NoError(t, err)
		mode := info.Mode()

		// Should have execute permissions for owner
		assert.NotEqual(t, 0, mode&0100, "File should be executable by owner")
	})

	t.Run("File does not exist", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nonexistent.bin")

		err := fs.MakeExecutable(filePath)
		require.Error(t, err)
	})
}
