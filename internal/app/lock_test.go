package app

import (
	"path/filepath"
	"testing"
	"time"

	"portpls/internal/allocations"
)

func TestLockPort(t *testing.T) {
	t.Run("locks existing unlocked allocation", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate unlocked allocation
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		absDir, _ := filepath.Abs(dir)
		allocFile.SetAllocation(20005, &allocations.Allocation{
			Directory:  absDir,
			Name:       "main",
			AssignedAt: time.Now().Add(-1 * time.Hour),
			LastUsedAt: time.Now().Add(-1 * time.Hour),
			Locked:     false,
		})
		allocFile.Save()
		allocFile.Close()

		checker := mockChecker{freePorts: map[int]bool{20005: true}}

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
			PortChecker:     checker,
		}

		port, err := LockPort(opts, "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20005 {
			t.Errorf("port = %d, want 20005", port)
		}

		// Verify it's locked now
		allocFile2, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile2.Close()
		alloc := allocFile2.Data.Allocations["20005"]
		if alloc == nil {
			t.Fatal("allocation not found")
		}
		if !alloc.Locked {
			t.Error("allocation should be locked")
		}
	})

	t.Run("creates and locks new allocation when none exists", func(t *testing.T) {
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

		port, err := LockPort(opts, "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20000 {
			t.Errorf("port = %d, want 20000", port)
		}

		// Verify allocation is created and locked
		allocFile, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile.Close()
		alloc := allocFile.Data.Allocations["20000"]
		if alloc == nil {
			t.Fatal("allocation not found")
		}
		if !alloc.Locked {
			t.Error("new allocation should be locked")
		}
	})

	t.Run("returns error when no free ports", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20002)

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

		_, err := LockPort(opts, "main")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		codeErr, ok := err.(CodeError)
		if !ok {
			t.Fatalf("expected CodeError, got %T", err)
		}
		if codeErr.Code != 1 {
			t.Errorf("exit code = %d, want 1", codeErr.Code)
		}
	})
}

func TestUnlockPort(t *testing.T) {
	t.Run("unlocks existing locked allocation", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate locked allocation
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		absDir, _ := filepath.Abs(dir)
		allocFile.SetAllocation(20005, &allocations.Allocation{
			Directory:  absDir,
			Name:       "main",
			AssignedAt: time.Now().Add(-1 * time.Hour),
			LastUsedAt: time.Now().Add(-1 * time.Hour),
			Locked:     true,
		})
		allocFile.Save()
		allocFile.Close()

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
		}

		port, err := UnlockPort(opts, "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20005 {
			t.Errorf("port = %d, want 20005", port)
		}

		// Verify it's unlocked now
		allocFile2, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile2.Close()
		alloc := allocFile2.Data.Allocations["20005"]
		if alloc == nil {
			t.Fatal("allocation not found")
		}
		if alloc.Locked {
			t.Error("allocation should be unlocked")
		}
	})

	t.Run("returns error when allocation not found", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
		}

		_, err := UnlockPort(opts, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		codeErr, ok := err.(CodeError)
		if !ok {
			t.Fatalf("expected CodeError, got %T", err)
		}
		if codeErr.Code != 1 {
			t.Errorf("exit code = %d, want 1", codeErr.Code)
		}
	})

	t.Run("unlocks allocation in different directory returns error", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate allocation for a different directory
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		allocFile.SetAllocation(20005, &allocations.Allocation{
			Directory:  "/other/project",
			Name:       "main",
			AssignedAt: time.Now(),
			LastUsedAt: time.Now(),
			Locked:     true,
		})
		allocFile.Save()
		allocFile.Close()

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       dir,
		}

		_, err := UnlockPort(opts, "main")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
