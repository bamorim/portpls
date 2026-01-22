package app

import (
	"os"
	"path/filepath"
)

// DirectorySelector resolves to exactly ONE directory.
// Used by commands that need to operate on a specific directory.
type DirectorySelector interface {
	ResolveDirectory() (string, error)
}

// CurrentDirectory resolves to the current working directory.
type CurrentDirectory struct{}

func (CurrentDirectory) ResolveDirectory() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Abs(wd)
}

// SpecificDirectory resolves to a specific path.
type SpecificDirectory struct {
	Path string
}

func (s SpecificDirectory) ResolveDirectory() (string, error) {
	return filepath.Abs(s.Path)
}

// DirectoryFilter filters allocations by directory.
// Used by commands that need to filter/list allocations.
type DirectoryFilter func(allocDirectory string) bool

// NoFilter returns a filter that matches all directories.
func NoFilter() DirectoryFilter {
	return func(string) bool { return true }
}

// FilterByDirectory returns a filter that matches a specific directory.
func FilterByDirectory(path string) (DirectoryFilter, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return func(dir string) bool { return dir == resolved }, nil
}

// FilterByCurrentDirectory returns a filter that matches the current directory.
func FilterByCurrentDirectory() (DirectoryFilter, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return FilterByDirectory(wd)
}

// FilterBySelector returns a filter that matches the directory resolved by the selector.
func FilterBySelector(selector DirectorySelector) (DirectoryFilter, error) {
	dir, err := selector.ResolveDirectory()
	if err != nil {
		return nil, err
	}
	return FilterByDirectory(dir)
}
