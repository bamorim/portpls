package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	Path    string
	Verbose bool
}

func (l Logger) Debugf(format string, args ...any) {
	if !l.Verbose {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func (l Logger) Event(event string, details string) error {
	if l.Path == "" {
		return nil
	}
	path := expandPath(l.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	stamp := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("%s %s %s\n", stamp, event, details)
	_, err = f.WriteString(line)
	return err
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
