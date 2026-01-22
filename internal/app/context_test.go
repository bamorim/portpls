package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bamorim/portpls/internal/allocations"
	"github.com/bamorim/portpls/internal/config"
	"github.com/bamorim/portpls/internal/logger"
)

func TestApplyTTL(t *testing.T) {
	t.Run("removes expired allocations", func(t *testing.T) {
		tmpDir := t.TempDir()
		allocPath := filepath.Join(tmpDir, "allocations.json")

		allocFile, err := allocations.OpenLocked(allocPath, true)
		if err != nil {
			t.Fatalf("failed to open allocations: %v", err)
		}
		defer allocFile.Close()

		now := time.Now().UTC()
		// Add allocation that expired 2 hours ago (TTL is 1 hour)
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory:  "/old/project",
			Name:       "main",
			AssignedAt: now.Add(-3 * time.Hour),
			LastUsedAt: now.Add(-2 * time.Hour), // Last used 2 hours ago
			Locked:     false,
		})

		ctx := &context{
			config:    config.Config{AllocationTTL: "1h"},
			allocFile: allocFile,
			logger:    logger.Logger{},
		}

		changed, err := applyTTL(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed to be true")
		}

		// Allocation should be deleted
		if len(allocFile.Data.Allocations) != 0 {
			t.Errorf("expected 0 allocations, got %d", len(allocFile.Data.Allocations))
		}
	})

	t.Run("keeps non-expired allocations", func(t *testing.T) {
		tmpDir := t.TempDir()
		allocPath := filepath.Join(tmpDir, "allocations.json")

		allocFile, err := allocations.OpenLocked(allocPath, true)
		if err != nil {
			t.Fatalf("failed to open allocations: %v", err)
		}
		defer allocFile.Close()

		now := time.Now().UTC()
		// Add allocation used 30 minutes ago (TTL is 1 hour)
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory:  "/active/project",
			Name:       "main",
			AssignedAt: now.Add(-2 * time.Hour),
			LastUsedAt: now.Add(-30 * time.Minute), // Last used 30 min ago
			Locked:     false,
		})

		ctx := &context{
			config:    config.Config{AllocationTTL: "1h"},
			allocFile: allocFile,
			logger:    logger.Logger{},
		}

		changed, err := applyTTL(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed to be false")
		}

		// Allocation should still exist
		if len(allocFile.Data.Allocations) != 1 {
			t.Errorf("expected 1 allocation, got %d", len(allocFile.Data.Allocations))
		}
	})

	t.Run("returns false when TTL is 0 (disabled)", func(t *testing.T) {
		tmpDir := t.TempDir()
		allocPath := filepath.Join(tmpDir, "allocations.json")

		allocFile, err := allocations.OpenLocked(allocPath, true)
		if err != nil {
			t.Fatalf("failed to open allocations: %v", err)
		}
		defer allocFile.Close()

		now := time.Now().UTC()
		// Add old allocation
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory:  "/old/project",
			Name:       "main",
			AssignedAt: now.Add(-100 * 24 * time.Hour),
			LastUsedAt: now.Add(-100 * 24 * time.Hour), // 100 days ago
			Locked:     false,
		})

		ctx := &context{
			config:    config.Config{AllocationTTL: "0"},
			allocFile: allocFile,
			logger:    logger.Logger{},
		}

		changed, err := applyTTL(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed to be false when TTL is disabled")
		}

		// Allocation should still exist
		if len(allocFile.Data.Allocations) != 1 {
			t.Errorf("expected 1 allocation, got %d", len(allocFile.Data.Allocations))
		}
	})

	t.Run("handles nil context gracefully", func(t *testing.T) {
		changed, err := applyTTL(nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed to be false for nil context")
		}
	})

	t.Run("expires multiple allocations in one pass", func(t *testing.T) {
		tmpDir := t.TempDir()
		allocPath := filepath.Join(tmpDir, "allocations.json")

		allocFile, err := allocations.OpenLocked(allocPath, true)
		if err != nil {
			t.Fatalf("failed to open allocations: %v", err)
		}
		defer allocFile.Close()

		now := time.Now().UTC()
		// Add 2 expired and 1 active allocation
		allocFile.SetAllocation(20001, &allocations.Allocation{
			Directory:  "/old1",
			Name:       "main",
			LastUsedAt: now.Add(-2 * time.Hour), // expired
		})
		allocFile.SetAllocation(20002, &allocations.Allocation{
			Directory:  "/old2",
			Name:       "main",
			LastUsedAt: now.Add(-3 * time.Hour), // expired
		})
		allocFile.SetAllocation(20003, &allocations.Allocation{
			Directory:  "/active",
			Name:       "main",
			LastUsedAt: now.Add(-10 * time.Minute), // still active
		})

		ctx := &context{
			config:    config.Config{AllocationTTL: "1h"},
			allocFile: allocFile,
			logger:    logger.Logger{},
		}

		changed, err := applyTTL(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed to be true")
		}

		// Only the active allocation should remain
		if len(allocFile.Data.Allocations) != 1 {
			t.Errorf("expected 1 allocation, got %d", len(allocFile.Data.Allocations))
		}
		if _, exists := allocFile.Data.Allocations["20003"]; !exists {
			t.Error("expected allocation 20003 to remain")
		}
	})
}

func TestResolveOptions(t *testing.T) {
	t.Run("fills in default config path when empty", func(t *testing.T) {
		opts := resolveOptions(Options{})

		if opts.ConfigPath == "" {
			t.Error("ConfigPath should not be empty")
		}
		if opts.AllocationsPath == "" {
			t.Error("AllocationsPath should not be empty")
		}
	})

	t.Run("expands tilde in paths", func(t *testing.T) {
		opts := resolveOptions(Options{
			ConfigPath:      "~/.config/portpls/config.json",
			AllocationsPath: "~/.local/share/portpls/allocations.json",
		})

		if opts.ConfigPath[0] == '~' {
			t.Error("ConfigPath should have tilde expanded")
		}
		if opts.AllocationsPath[0] == '~' {
			t.Error("AllocationsPath should have tilde expanded")
		}
	})

	t.Run("preserves provided paths", func(t *testing.T) {
		opts := resolveOptions(Options{
			ConfigPath:      "/custom/config.json",
			AllocationsPath: "/custom/allocations.json",
		})

		if opts.ConfigPath != "/custom/config.json" {
			t.Errorf("ConfigPath = %q, want /custom/config.json", opts.ConfigPath)
		}
		if opts.AllocationsPath != "/custom/allocations.json" {
			t.Errorf("AllocationsPath = %q, want /custom/allocations.json", opts.AllocationsPath)
		}
	})
}

func TestResolveDirectory(t *testing.T) {
	t.Run("uses override when provided", func(t *testing.T) {
		dir, err := resolveDirectory("/custom/dir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != "/custom/dir" {
			t.Errorf("dir = %q, want /custom/dir", dir)
		}
	})

	t.Run("uses current working directory when override is empty", func(t *testing.T) {
		dir, err := resolveDirectory("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir == "" {
			t.Error("dir should not be empty")
		}
	})

	t.Run("returns absolute path", func(t *testing.T) {
		dir, err := resolveDirectory("relative/path")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir[0] != '/' {
			t.Errorf("dir should be absolute, got %q", dir)
		}
	})
}

func TestOptionsResolvedDirectory(t *testing.T) {
	t.Run("uses directory override when set", func(t *testing.T) {
		opts := Options{
			Directory:          "/custom/dir",
			DirectorySet:       true,
			ParentDirectory:    "/parent/dir",
			ParentDirectorySet: true,
		}
		dir, err := opts.ResolvedDirectory()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != "/custom/dir" {
			t.Errorf("dir = %q, want /custom/dir", dir)
		}
	})

	t.Run("falls back to parent directory when override not set", func(t *testing.T) {
		opts := Options{
			ParentDirectory:    "/parent/dir",
			ParentDirectorySet: true,
		}
		dir, err := opts.ResolvedDirectory()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != "/parent/dir" {
			t.Errorf("dir = %q, want /parent/dir", dir)
		}
	})

	t.Run("uses directory value when set without flag metadata", func(t *testing.T) {
		opts := Options{
			Directory: "/custom/dir",
		}
		dir, err := opts.ResolvedDirectory()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != "/custom/dir" {
			t.Errorf("dir = %q, want /custom/dir", dir)
		}
	})
}
