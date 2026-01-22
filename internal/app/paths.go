package app

import (
	"os"
	"path/filepath"
)

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/portpls/config.json"
	}
	return filepath.Join(home, ".config", "portpls", "config.json")
}

func DefaultAllocationsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".local/share/portpls/allocations.json"
	}
	return filepath.Join(home, ".local", "share", "portpls", "allocations.json")
}
