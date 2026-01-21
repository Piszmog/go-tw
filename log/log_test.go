package log_test

import (
	"log/slog"
	"testing"

	"github.com/Piszmog/go-tw/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected log.Level
	}{
		{"debug level", "debug", log.LevelDebug},
		{"info level", "info", log.LevelInfo},
		{"warn level", "warn", log.LevelWarn},
		{"error level", "error", log.LevelError},
		{"invalid level", "invalid", log.LevelInfo},
		{"empty string", "", log.LevelInfo},
		{"uppercase", "DEBUG", log.LevelInfo}, // case sensitive
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := log.ToLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected log.Output
	}{
		{"json output", "json", log.OutputJSON},
		{"text output", "text", log.OutputText},
		{"invalid output", "invalid", log.OutputText},
		{"empty string", "", log.OutputText},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := log.ToOutput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLevelToSlog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    log.Level
		expected slog.Level
	}{
		{"debug to slog", log.LevelDebug, slog.LevelDebug},
		{"info to slog", log.LevelInfo, slog.LevelInfo},
		{"warn to slog", log.LevelWarn, slog.LevelWarn},
		{"error to slog", log.LevelError, slog.LevelError},
		{"invalid to slog", log.Level("invalid"), slog.LevelInfo},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.level.ToSlog()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		level  log.Level
		output log.Output
	}{
		{"debug json logger", log.LevelDebug, log.OutputJSON},
		{"info text logger", log.LevelInfo, log.OutputText},
		{"warn json logger", log.LevelWarn, log.OutputJSON},
		{"error text logger", log.LevelError, log.OutputText},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := log.New(tt.level, tt.output)
			require.NotNil(t, logger)
			assert.IsType(t, &slog.Logger{}, logger)
		})
	}
}

func TestGetLevel(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because subtests use t.Setenv()

	tests := []struct {
		name     string
		envValue string
		expected log.Level
	}{
		{"debug from env", "debug", log.LevelDebug},
		{"info from env", "info", log.LevelInfo},
		{"warn from env", "warn", log.LevelWarn},
		{"error from env", "error", log.LevelError},
		{"empty env", "", log.LevelInfo},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() with t.Setenv() - Go limitation
			t.Setenv("LOG_LEVEL", tt.envValue)

			result := log.GetLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetOutput(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because subtests use t.Setenv()

	tests := []struct {
		name     string
		envValue string
		expected log.Output
	}{
		{"json from env", "json", log.OutputJSON},
		{"text from env", "text", log.OutputText},
		{"empty env", "", log.OutputText},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() with t.Setenv() - Go limitation
			t.Setenv("LOG_OUTPUT", tt.envValue)

			result := log.GetOutput()
			assert.Equal(t, tt.expected, result)
		})
	}
}
