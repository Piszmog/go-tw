package client_test

import (
	"os"
	"testing"

	"github.com/Piszmog/go-tw/client"
	"github.com/stretchr/testify/assert"
)

// mockFileReader is a test double for client.FileReader
type mockFileReader struct {
	files  map[string][]byte
	exists map[string]bool
}

func (m *mockFileReader) ReadFile(path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileReader) FileExists(path string) bool {
	return m.exists[path]
}

func TestGetNameWithReader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		os       string
		arch     string
		reader   *mockFileReader
		expected string
	}{
		{
			name: "Linux AMD64 musl via /proc/self/maps",
			os:   "linux",
			arch: "amd64",
			reader: &mockFileReader{
				files: map[string][]byte{
					"/proc/self/maps": []byte("7f00 r--p 00000000 08:01 /lib/ld-musl-x86_64.so.1"),
				},
			},
			expected: "tailwindcss-linux-x64-musl",
		},
		{
			name: "Linux AMD64 musl via linker file (static binary)",
			os:   "linux",
			arch: "amd64",
			reader: &mockFileReader{
				exists: map[string]bool{
					"/lib/ld-musl-x86_64.so.1": true,
				},
			},
			expected: "tailwindcss-linux-x64-musl",
		},
		{
			name: "Linux ARM64 musl via linker file",
			os:   "linux",
			arch: "arm64",
			reader: &mockFileReader{
				exists: map[string]bool{
					"/lib/ld-musl-aarch64.so.1": true,
				},
			},
			expected: "tailwindcss-linux-arm64-musl",
		},
		{
			name: "Linux AMD64 glibc in maps no musl linker",
			os:   "linux",
			arch: "amd64",
			reader: &mockFileReader{
				files: map[string][]byte{
					"/proc/self/maps": []byte("7f00 r--p 00000000 08:01 /lib/x86_64-linux-gnu/libc.so.6"),
				},
			},
			expected: "tailwindcss-linux-x64",
		},
		{
			name:     "Linux AMD64 no files (static binary on glibc)",
			os:       "linux",
			arch:     "amd64",
			reader:   &mockFileReader{},
			expected: "tailwindcss-linux-x64",
		},
		{
			name: "Darwin ignores musl check",
			os:   "darwin",
			arch: "arm64",
			reader: &mockFileReader{
				files: map[string][]byte{
					"/proc/self/maps": []byte("7f00 r--p 00000000 08:01 /lib/ld-musl-x86_64.so.1"),
				},
			},
			expected: "tailwindcss-macos-arm64",
		},
		{
			name: "Windows ignores musl check",
			os:   "windows",
			arch: "amd64",
			reader: &mockFileReader{
				exists: map[string]bool{
					"/lib/ld-musl-x86_64.so.1": true,
				},
			},
			expected: "tailwindcss-windows-x64.exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, client.GetNameWithReader(tt.os, tt.arch, tt.reader))
		})
	}
}
