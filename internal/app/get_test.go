package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bamorim/portpls/internal/allocations"
)

func setupTestEnv(t *testing.T) (configPath, allocPath, dir string) {
	t.Helper()
	tmpDir := t.TempDir()

	configPath = filepath.Join(tmpDir, "config.json")
	allocPath = filepath.Join(tmpDir, "allocations.json")
	dir = filepath.Join(tmpDir, "project")

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	return configPath, allocPath, dir
}

func writeConfig(t *testing.T, path string, portStart, portEnd int) {
	t.Helper()
	cfg := map[string]interface{}{
		"port_start":     portStart,
		"port_end":       portEnd,
		"freeze_period":  "0",
		"allocation_ttl": "0",
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

func TestGetPort(t *testing.T) {
	t.Run("allocates new port when none exists", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: true,
		}}

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
			PortChecker:     checker,
		}

		port, err := GetPort(opts, "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20000 {
			t.Errorf("port = %d, want 20000", port)
		}

		// Verify allocation was saved
		allocFile, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile.Close()
		if len(allocFile.Data.Allocations) != 1 {
			t.Errorf("expected 1 allocation, got %d", len(allocFile.Data.Allocations))
		}
	})

	t.Run("reuses existing allocation when port is free", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate allocation
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		absDir, _ := filepath.Abs(dir)
		allocFile.SetAllocation(20005, &allocations.Allocation{
			Directory:  absDir,
			Name:       "main",
			AssignedAt: time.Now().Add(-1 * time.Hour),
			LastUsedAt: time.Now().Add(-1 * time.Hour),
		})
		allocFile.Save()
		allocFile.Close()

		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20005: true, // existing allocation's port is free
		}}

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
			PortChecker:     checker,
		}

		port, err := GetPort(opts, "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20005 {
			t.Errorf("port = %d, want 20005 (reused)", port)
		}
	})

	t.Run("allocates new port when existing port is busy", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate allocation
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		absDir, _ := filepath.Abs(dir)
		allocFile.SetAllocation(20005, &allocations.Allocation{
			Directory:  absDir,
			Name:       "main",
			AssignedAt: time.Now().Add(-1 * time.Hour),
			LastUsedAt: time.Now().Add(-1 * time.Hour),
		})
		allocFile.Save()
		allocFile.Close()

		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20005: false, // existing allocation's port is busy
		}}

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
			PortChecker:     checker,
		}

		port, err := GetPort(opts, "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20000 {
			t.Errorf("port = %d, want 20000 (new allocation)", port)
		}

		// Verify old allocation was replaced
		allocFile2, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile2.Close()
		if _, exists := allocFile2.Data.Allocations["20005"]; exists {
			t.Error("old allocation should have been deleted")
		}
	})

	t.Run("returns error when no free ports available", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20002)

		// All ports busy
		checker := mockChecker{freePorts: map[int]bool{
			20000: false,
			20001: false,
			20002: false,
		}}

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
			PortChecker:     checker,
		}

		_, err := GetPort(opts, "main")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Should be a CodeError
		codeErr, ok := err.(CodeError)
		if !ok {
			t.Fatalf("expected CodeError, got %T", err)
		}
		if codeErr.Code != 1 {
			t.Errorf("exit code = %d, want 1", codeErr.Code)
		}
	})

	t.Run("different names get different ports", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: true,
			20002: true,
		}}

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
			PortChecker:     checker,
		}

		port1, err := GetPort(opts, "web")
		if err != nil {
			t.Fatalf("unexpected error for web: %v", err)
		}

		port2, err := GetPort(opts, "api")
		if err != nil {
			t.Fatalf("unexpected error for api: %v", err)
		}

		if port1 == port2 {
			t.Errorf("expected different ports for different names, got %d for both", port1)
		}
	})
}
