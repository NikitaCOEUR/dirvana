package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		output *bytes.Buffer
	}{
		{
			name:   "debug level",
			level:  "debug",
			output: &bytes.Buffer{},
		},
		{
			name:   "info level",
			level:  "info",
			output: &bytes.Buffer{},
		},
		{
			name:   "warn level",
			level:  "warn",
			output: &bytes.Buffer{},
		},
		{
			name:   "error level",
			level:  "error",
			output: &bytes.Buffer{},
		},
		{
			name:   "invalid level defaults to info",
			level:  "invalid",
			output: &bytes.Buffer{},
		},
		{
			name:   "empty level defaults to info",
			level:  "",
			output: &bytes.Buffer{},
		},
		{
			name:   "uppercase level",
			level:  "DEBUG",
			output: &bytes.Buffer{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.level, tt.output)
			if logger == nil {
				t.Fatal("Expected logger to be non-nil")
				return
			}
			if logger.log == nil {
				t.Fatal("Expected internal log to be non-nil")
				return
			}
		})
	}
}

func TestNew_NilOutput(t *testing.T) {
	logger := New("info", nil)
	if logger == nil {
		t.Fatal("Expected logger to be non-nil")
		return
	}
	if logger.log == nil {
		t.Fatal("Expected internal log to be non-nil")
		return
	}
}

func TestLogger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("debug", buf)

	logger.Debug().Msg("debug message")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Errorf("Expected output to contain 'debug message', got: %s", output)
	}
}

func TestLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("info", buf)

	logger.Info().Msg("info message")

	output := buf.String()
	if !strings.Contains(output, "info message") {
		t.Errorf("Expected output to contain 'info message', got: %s", output)
	}
}

func TestLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("warn", buf)

	logger.Warn().Msg("warn message")

	output := buf.String()
	if !strings.Contains(output, "warn message") {
		t.Errorf("Expected output to contain 'warn message', got: %s", output)
	}
}

func TestLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("error", buf)

	logger.Error().Msg("error message")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("Expected output to contain 'error message', got: %s", output)
	}
}

func TestEntry_Dur(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("info", buf)

	duration := 1500 * time.Microsecond
	logger.Info().Dur("duration", duration).Msg("test duration")

	output := buf.String()
	if !strings.Contains(output, "duration") {
		t.Errorf("Expected output to contain 'duration' field")
	}
	if !strings.Contains(output, "1.5") {
		t.Errorf("Expected output to contain '1.5' milliseconds")
	}
}

func TestEntry_Float(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("info", buf)

	logger.Info().Float("value", 3.14159).Msg("test float")

	output := buf.String()
	if !strings.Contains(output, "value") {
		t.Errorf("Expected output to contain 'value' field")
	}
	if !strings.Contains(output, "3.14159") {
		t.Errorf("Expected output to contain '3.14159'")
	}
}

func TestEntry_Str(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("info", buf)

	logger.Info().Str("key", "value").Msg("message")

	output := buf.String()
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("Expected output to contain key=value, got: %s", output)
	}
}

func TestEntry_Int(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("info", buf)

	logger.Info().Int("count", 42).Msg("message")

	output := buf.String()
	if !strings.Contains(output, "count") || !strings.Contains(output, "42") {
		t.Errorf("Expected output to contain count=42, got: %s", output)
	}
}

func TestEntry_Bool(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("info", buf)

	logger.Info().Bool("enabled", true).Msg("message")

	output := buf.String()
	if !strings.Contains(output, "enabled") || !strings.Contains(output, "true") {
		t.Errorf("Expected output to contain enabled=true, got: %s", output)
	}
}

func TestEntry_Err(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("error", buf)

	testErr := errors.New("test error")
	logger.Error().Err(testErr).Msg("error occurred")

	output := buf.String()
	if !strings.Contains(output, "test error") {
		t.Errorf("Expected output to contain 'test error', got: %s", output)
	}
}

func TestEntry_Err_Nil(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("error", buf)

	logger.Error().Err(nil).Msg("no error")

	output := buf.String()
	if !strings.Contains(output, "no error") {
		t.Errorf("Expected output to contain 'no error', got: %s", output)
	}
}

func TestEntry_ChainedFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("info", buf)

	testErr := errors.New("chain error")
	logger.Info().
		Str("string", "value").
		Int("number", 123).
		Bool("flag", false).
		Err(testErr).
		Msg("chained message")

	output := buf.String()
	if !strings.Contains(output, "chained message") {
		t.Errorf("Expected output to contain 'chained message', got: %s", output)
	}
	if !strings.Contains(output, "string") {
		t.Errorf("Expected output to contain 'string', got: %s", output)
	}
	if !strings.Contains(output, "number") {
		t.Errorf("Expected output to contain 'number', got: %s", output)
	}
	if !strings.Contains(output, "flag") {
		t.Errorf("Expected output to contain 'flag', got: %s", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		logLevel    string
		messageFunc func(*Logger)
		shouldLog   bool
	}{
		{
			name:     "debug message with debug level",
			logLevel: "debug",
			messageFunc: func(l *Logger) {
				l.Debug().Msg("debug")
			},
			shouldLog: true,
		},
		{
			name:     "debug message with info level",
			logLevel: "info",
			messageFunc: func(l *Logger) {
				l.Debug().Msg("debug")
			},
			shouldLog: false,
		},
		{
			name:     "info message with warn level",
			logLevel: "warn",
			messageFunc: func(l *Logger) {
				l.Info().Msg("info")
			},
			shouldLog: false,
		},
		{
			name:     "warn message with warn level",
			logLevel: "warn",
			messageFunc: func(l *Logger) {
				l.Warn().Msg("warn")
			},
			shouldLog: true,
		},
		{
			name:     "error message with error level",
			logLevel: "error",
			messageFunc: func(l *Logger) {
				l.Error().Msg("error")
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(tt.logLevel, buf)

			tt.messageFunc(logger)

			output := buf.String()
			hasOutput := len(output) > 0

			if tt.shouldLog && !hasOutput {
				t.Errorf("Expected log output but got none")
			}
			if !tt.shouldLog && hasOutput {
				t.Errorf("Expected no log output but got: %s", output)
			}
		})
	}
}

func TestEntry_Msg_DefaultLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("info", buf)

	// Create an entry and manually remove the level field to trigger default case
	entry := logger.Info()
	delete(entry.entry.Data, "level")
	entry.Msg("default level message")

	output := buf.String()
	if !strings.Contains(output, "default level message") {
		t.Errorf("Expected output to contain 'default level message', got: %s", output)
	}
}

func TestLogger_AllLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New("debug", buf)

	// Test all log levels
	logger.Debug().Str("type", "debug").Msg("debug message")
	logger.Info().Str("type", "info").Msg("info message")
	logger.Warn().Str("type", "warn").Msg("warn message")
	logger.Error().Str("type", "error").Msg("error message")

	output := buf.String()

	expectedMessages := []string{"debug message", "info message", "warn message", "error message"}
	for _, msg := range expectedMessages {
		if !strings.Contains(output, msg) {
			t.Errorf("Expected output to contain '%s', got: %s", msg, output)
		}
	}
}
