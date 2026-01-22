package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentDirectory_ResolveDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	expected, err := filepath.Abs(cwd)
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}

	selector := CurrentDirectory{}
	got, err := selector.ResolveDirectory()
	if err != nil {
		t.Fatalf("ResolveDirectory() error = %v", err)
	}
	if got != expected {
		t.Errorf("ResolveDirectory() = %v, want %v", got, expected)
	}
}

func TestSpecificDirectory_ResolveDirectory(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantErr  bool
	}{
		{
			name:    "absolute path",
			path:    "/tmp/test",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "relative/path",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := SpecificDirectory{Path: tt.path}
			got, err := selector.ResolveDirectory()
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				expected, _ := filepath.Abs(tt.path)
				if got != expected {
					t.Errorf("ResolveDirectory() = %v, want %v", got, expected)
				}
			}
		})
	}
}

func TestNoFilter(t *testing.T) {
	filter := NoFilter()

	testDirs := []string{
		"/some/path",
		"/another/path",
		"",
		"/tmp",
	}

	for _, dir := range testDirs {
		if !filter(dir) {
			t.Errorf("NoFilter() should return true for %q", dir)
		}
	}
}

func TestFilterByDirectory(t *testing.T) {
	// Use a temp directory to ensure consistent absolute path resolution
	tmpDir := t.TempDir()

	filter, err := FilterByDirectory(tmpDir)
	if err != nil {
		t.Fatalf("FilterByDirectory() error = %v", err)
	}

	absPath, _ := filepath.Abs(tmpDir)

	tests := []struct {
		name     string
		dir      string
		expected bool
	}{
		{
			name:     "matching directory",
			dir:      absPath,
			expected: true,
		},
		{
			name:     "non-matching directory",
			dir:      "/some/other/path",
			expected: false,
		},
		{
			name:     "empty directory",
			dir:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter(tt.dir); got != tt.expected {
				t.Errorf("FilterByDirectory()(%q) = %v, want %v", tt.dir, got, tt.expected)
			}
		})
	}
}

func TestFilterByCurrentDirectory(t *testing.T) {
	filter, err := FilterByCurrentDirectory()
	if err != nil {
		t.Fatalf("FilterByCurrentDirectory() error = %v", err)
	}

	cwd, _ := os.Getwd()
	absCwd, _ := filepath.Abs(cwd)

	if !filter(absCwd) {
		t.Errorf("FilterByCurrentDirectory() should match current directory %q", absCwd)
	}

	if filter("/some/other/path") {
		t.Errorf("FilterByCurrentDirectory() should not match /some/other/path")
	}
}
