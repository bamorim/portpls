package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogger_Event(t *testing.T) {
	t.Run("writes event to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		log := Logger{Path: logPath, Verbose: false}
		err := log.Event("TEST_EVENT", "key=value")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		if !strings.Contains(string(content), "TEST_EVENT") {
			t.Error("log should contain event name")
		}
		if !strings.Contains(string(content), "key=value") {
			t.Error("log should contain event details")
		}
	})

	t.Run("creates directory if needed", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "subdir", "nested", "test.log")

		log := Logger{Path: logPath}
		err := log.Event("EVENT", "details")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(logPath); err != nil {
			t.Errorf("log file was not created: %v", err)
		}
	})

	t.Run("no-op when path is empty", func(t *testing.T) {
		log := Logger{Path: ""}
		err := log.Event("EVENT", "details")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("appends to existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		log := Logger{Path: logPath}

		_ = log.Event("FIRST", "first event")
		_ = log.Event("SECOND", "second event")

		content, _ := os.ReadFile(logPath)
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")

		if len(lines) != 2 {
			t.Errorf("expected 2 log lines, got %d", len(lines))
		}
	})

	t.Run("includes RFC3339 timestamp", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		log := Logger{Path: logPath}
		_ = log.Event("EVENT", "details")

		content, _ := os.ReadFile(logPath)
		// RFC3339 format looks like: 2024-01-15T10:30:00Z
		if !strings.Contains(string(content), "T") || !strings.Contains(string(content), "Z") {
			t.Error("log should contain RFC3339 timestamp")
		}
	})
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~/logs/app.log", filepath.Join(home, "logs/app.log")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandPath(tt.input)
			if got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
