package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bamorim/portpls/internal/allocations"
)

func TestForget(t *testing.T) {
	t.Run("forgets specific named allocation", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate allocations
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		absDir, _ := filepath.Abs(dir)
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory: absDir, Name: "web",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.SetAllocation(20002, &allocations.Allocation{
			Directory: absDir, Name: "api",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.Save()
		allocFile.Close()

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       SpecificDirectory{Path: dir},
		}

		filter, _ := FilterByDirectory(absDir)
		result, err := Forget(opts, filter, "web", true, false, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Message, "web") {
			t.Errorf("message should contain 'web', got: %s", result.Message)
		}

		// Verify web is deleted, api remains
		allocFile2, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile2.Close()
		if _, exists := allocFile2.Data.Allocations["20001"]; exists {
			t.Error("allocation 20001 (web) should be deleted")
		}
		if _, exists := allocFile2.Data.Allocations["20002"]; !exists {
			t.Error("allocation 20002 (api) should remain")
		}
	})

	t.Run("returns message when named allocation not found", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       SpecificDirectory{Path: dir},
		}

		absDir, _ := filepath.Abs(dir)
		filter, _ := FilterByDirectory(absDir)
		result, err := Forget(opts, filter, "nonexistent", true, false, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Message, "No allocation found") {
			t.Errorf("expected 'No allocation found' message, got: %s", result.Message)
		}
	})

	t.Run("forgets all allocations for current directory", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate allocations
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		absDir, _ := filepath.Abs(dir)
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory: absDir, Name: "web",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.SetAllocation(20002, &allocations.Allocation{
			Directory: absDir, Name: "api",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.SetAllocation(20003, &allocations.Allocation{
			Directory: "/other/project", Name: "main",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.Save()
		allocFile.Close()

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       SpecificDirectory{Path: dir},
		}

		filter, _ := FilterByDirectory(absDir)
		result, err := Forget(opts, filter, "", false, true, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Message, "2 allocation(s)") {
			t.Errorf("expected '2 allocation(s)' in message, got: %s", result.Message)
		}

		// Verify current directory's allocations are deleted, other remains
		allocFile2, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile2.Close()
		if len(allocFile2.Data.Allocations) != 1 {
			t.Errorf("expected 1 allocation remaining, got %d", len(allocFile2.Data.Allocations))
		}
		if _, exists := allocFile2.Data.Allocations["20003"]; !exists {
			t.Error("allocation 20003 (other project) should remain")
		}
	})

	t.Run("forgets all allocations globally with confirmation", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate allocations
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		absDir, _ := filepath.Abs(dir)
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory: absDir, Name: "web",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.SetAllocation(20002, &allocations.Allocation{
			Directory: "/other/project", Name: "main",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.Save()
		allocFile.Close()

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       SpecificDirectory{Path: dir},
		}

		// Confirm returns true
		confirmed := false
		confirm := func() bool {
			confirmed = true
			return true
		}

		filter := NoFilter() // Global delete across all directories
		result, err := Forget(opts, filter, "", false, true, confirm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !confirmed {
			t.Error("confirm callback should have been called")
		}
		if !strings.Contains(result.Message, "2 allocation(s)") {
			t.Errorf("expected '2 allocation(s)' in message, got: %s", result.Message)
		}

		// Verify all allocations are deleted
		allocFile2, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile2.Close()
		if len(allocFile2.Data.Allocations) != 0 {
			t.Errorf("expected 0 allocations, got %d", len(allocFile2.Data.Allocations))
		}
	})

	t.Run("returns error when confirmation declined", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate allocation
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory: "/some/project", Name: "main",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.Save()
		allocFile.Close()

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       SpecificDirectory{Path: dir},
		}

		// Confirm returns false
		confirm := func() bool { return false }

		filter := NoFilter() // Global delete across all directories
		_, err := Forget(opts, filter, "", false, true, confirm)
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

		// Allocation should still exist
		allocFile2, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile2.Close()
		if len(allocFile2.Data.Allocations) != 1 {
			t.Errorf("allocation should still exist after decline")
		}
	})

	t.Run("forgets named allocation across all directories", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		// Pre-populate allocations with same name in different directories
		allocFile, _ := allocations.OpenLocked(allocPath, true)
		absDir, _ := filepath.Abs(dir)
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory: absDir, Name: "web",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.SetAllocation(20002, &allocations.Allocation{
			Directory: "/other/project", Name: "web",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.SetAllocation(20003, &allocations.Allocation{
			Directory: "/another/project", Name: "api",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})
		allocFile.Save()
		allocFile.Close()

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       SpecificDirectory{Path: dir},
		}

		filter := NoFilter() // Match all directories
		result, err := Forget(opts, filter, "web", true, false, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Message, "2 allocation(s)") {
			t.Errorf("expected '2 allocation(s)' in message, got: %s", result.Message)
		}

		// Verify both "web" allocations are deleted, "api" remains
		allocFile2, _ := allocations.OpenLocked(allocPath, false)
		defer allocFile2.Close()
		if len(allocFile2.Data.Allocations) != 1 {
			t.Errorf("expected 1 allocation remaining, got %d", len(allocFile2.Data.Allocations))
		}
		if _, exists := allocFile2.Data.Allocations["20003"]; !exists {
			t.Error("allocation 20003 (api) should remain")
		}
		if _, exists := allocFile2.Data.Allocations["20001"]; exists {
			t.Error("allocation 20001 (web) should be deleted")
		}
		if _, exists := allocFile2.Data.Allocations["20002"]; exists {
			t.Error("allocation 20002 (web) should be deleted")
		}
	})

	t.Run("returns error when neither name nor all specified", func(t *testing.T) {
		configPath, allocPath, dir := setupTestEnv(t)
		writeConfig(t, configPath, 20000, 20010)

		opts := Options{
			ConfigPath:      configPath,
			AllocationsPath: allocPath,
			Directory:       SpecificDirectory{Path: dir},
		}

		absDir, _ := filepath.Abs(dir)
		filter, _ := FilterByDirectory(absDir)
		_, err := Forget(opts, filter, "", false, false, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		codeErr, ok := err.(CodeError)
		if !ok {
			t.Fatalf("expected CodeError, got %T", err)
		}
		if codeErr.Code != 2 {
			t.Errorf("exit code = %d, want 2", codeErr.Code)
		}
	})
}
