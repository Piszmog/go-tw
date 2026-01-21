package main_test

import (
	"os"
	"testing"

	main "github.com/Piszmog/go-tw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		os       string
		arch     string
		expected bool
	}{
		{"Linux AMD64", "linux", "amd64", true},
		{"Linux ARM64", "linux", "arm64", true},
		{"Darwin AMD64", "darwin", "amd64", true},
		{"Darwin ARM64", "darwin", "arm64", true},
		{"Windows AMD64", "windows", "amd64", true},
		{"Windows ARM64", "windows", "arm64", true},
		{"FreeBSD AMD64", "freebsd", "amd64", false},
		{"Linux 386", "linux", "386", false},
		{"Linux ARM", "linux", "arm", false},
		{"Unsupported OS", "plan9", "amd64", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := main.IsSupported(tt.os, tt.arch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantVersion string
		wantArgs    []string
		wantErr     error
	}{
		{
			name:        "No arguments",
			args:        []string{},
			wantVersion: "latest",
			wantArgs:    nil,
			wantErr:     nil,
		},
		{
			name:        "No version flag",
			args:        []string{"-i", "input.css", "-o", "output.css"},
			wantVersion: "latest",
			wantArgs:    []string{"-i", "input.css", "-o", "output.css"},
			wantErr:     nil,
		},
		{
			name:        "With version flag",
			args:        []string{"-version", "v4.0.0", "-i", "input.css"},
			wantVersion: "v4.0.0",
			wantArgs:    []string{"-i", "input.css"},
			wantErr:     nil,
		},
		{
			name:        "Version flag at end",
			args:        []string{"-i", "input.css", "-version", "v3.0.0"},
			wantVersion: "v3.0.0",
			wantArgs:    []string{"-i", "input.css"},
			wantErr:     nil,
		},
		{
			name:        "Version flag without argument",
			args:        []string{"-version"},
			wantVersion: "",
			wantArgs:    nil,
			wantErr:     main.ErrMissingVersionArg,
		},
		{
			name:        "Version flag at end without argument",
			args:        []string{"-i", "input.css", "-version"},
			wantVersion: "",
			wantArgs:    nil,
			wantErr:     main.ErrMissingVersionArg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Save and restore os.Args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = append([]string{"go-tw"}, tt.args...)

			version, args, err := main.GetArgs()

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantVersion, version)
				assert.Equal(t, tt.wantArgs, args)
			}
		})
	}
}
