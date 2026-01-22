package allocations

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestOpenLocked(t *testing.T) {
	t.Run("creates file when it does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "allocations.json")

		lf, err := OpenLocked(path, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer lf.Close()

		// File should exist
		if _, err := os.Stat(path); err != nil {
			t.Errorf("file was not created: %v", err)
		}

		// Data should have defaults
		if lf.Data.Version != 1 {
			t.Errorf("Version = %d, want 1", lf.Data.Version)
		}
		if lf.Data.LastIssuedPort != 0 {
			t.Errorf("LastIssuedPort = %d, want 0", lf.Data.LastIssuedPort)
		}
		if len(lf.Data.Allocations) != 0 {
			t.Errorf("Allocations should be empty, got %d", len(lf.Data.Allocations))
		}
	})

	t.Run("creates nested directories if needed", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "nested", "dir", "allocations.json")

		lf, err := OpenLocked(path, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer lf.Close()

		if _, err := os.Stat(path); err != nil {
			t.Errorf("file was not created: %v", err)
		}
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		_, err := OpenLocked("", true)
		if err == nil {
			t.Error("expected error for empty path, got nil")
		}
	})

	t.Run("loads existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "allocations.json")

		// Create initial file with some data
		initial := &File{
			Version:        1,
			LastIssuedPort: 20005,
			Allocations: map[string]*Allocation{
				"20001": {
					Directory:  "/project/foo",
					Name:       "main",
					AssignedAt: time.Now().Add(-time.Hour),
					LastUsedAt: time.Now(),
					Locked:     false,
				},
			},
		}
		if err := writeFile(path, initial); err != nil {
			t.Fatalf("failed to write initial file: %v", err)
		}

		lf, err := OpenLocked(path, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer lf.Close()

		if lf.Data.LastIssuedPort != 20005 {
			t.Errorf("LastIssuedPort = %d, want 20005", lf.Data.LastIssuedPort)
		}
		if len(lf.Data.Allocations) != 1 {
			t.Errorf("expected 1 allocation, got %d", len(lf.Data.Allocations))
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "allocations.json")

		// Create empty file
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("failed to write empty file: %v", err)
		}

		lf, err := OpenLocked(path, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer lf.Close()

		// Should get defaults
		if lf.Data.Version != 1 {
			t.Errorf("Version = %d, want 1", lf.Data.Version)
		}
	})
}

func TestLockedFile_SetAllocation(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "allocations.json")

	lf, err := OpenLocked(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lf.Close()

	now := time.Now().UTC()
	alloc := &Allocation{
		Directory:  "/project/foo",
		Name:       "main",
		AssignedAt: now,
		LastUsedAt: now,
		Locked:     false,
	}

	lf.SetAllocation(20001, alloc)

	// Verify it was set
	if len(lf.Data.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(lf.Data.Allocations))
	}

	stored := lf.Data.Allocations["20001"]
	if stored == nil {
		t.Fatal("allocation not found at key 20001")
	}
	if stored.Directory != "/project/foo" {
		t.Errorf("Directory = %q, want %q", stored.Directory, "/project/foo")
	}
	if stored.Name != "main" {
		t.Errorf("Name = %q, want %q", stored.Name, "main")
	}
}

func TestLockedFile_FindByDirectoryName(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "allocations.json")

	lf, err := OpenLocked(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lf.Close()

	now := time.Now().UTC()

	// Add some allocations
	lf.SetAllocation(20001, &Allocation{
		Directory: "/project/foo", Name: "main",
		AssignedAt: now, LastUsedAt: now,
	})
	lf.SetAllocation(20002, &Allocation{
		Directory: "/project/foo", Name: "api",
		AssignedAt: now, LastUsedAt: now,
	})
	lf.SetAllocation(20003, &Allocation{
		Directory: "/project/bar", Name: "main",
		AssignedAt: now, LastUsedAt: now,
	})

	t.Run("finds existing allocation", func(t *testing.T) {
		port, alloc := lf.FindByDirectoryName("/project/foo", "main")
		if port != 20001 {
			t.Errorf("port = %d, want 20001", port)
		}
		if alloc == nil {
			t.Fatal("allocation should not be nil")
		}
	})

	t.Run("finds different name in same directory", func(t *testing.T) {
		port, alloc := lf.FindByDirectoryName("/project/foo", "api")
		if port != 20002 {
			t.Errorf("port = %d, want 20002", port)
		}
		if alloc == nil {
			t.Fatal("allocation should not be nil")
		}
	})

	t.Run("finds same name in different directory", func(t *testing.T) {
		port, alloc := lf.FindByDirectoryName("/project/bar", "main")
		if port != 20003 {
			t.Errorf("port = %d, want 20003", port)
		}
		if alloc == nil {
			t.Fatal("allocation should not be nil")
		}
	})

	t.Run("returns zero and nil when not found", func(t *testing.T) {
		port, alloc := lf.FindByDirectoryName("/project/foo", "nonexistent")
		if port != 0 {
			t.Errorf("port = %d, want 0", port)
		}
		if alloc != nil {
			t.Error("allocation should be nil")
		}
	})
}

func TestLockedFile_DeletePort(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "allocations.json")

	lf, err := OpenLocked(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lf.Close()

	now := time.Now().UTC()
	lf.SetAllocation(20001, &Allocation{
		Directory: "/project/foo", Name: "main",
		AssignedAt: now, LastUsedAt: now,
	})
	lf.SetAllocation(20002, &Allocation{
		Directory: "/project/foo", Name: "api",
		AssignedAt: now, LastUsedAt: now,
	})

	// Delete one
	lf.DeletePort(20001)

	if len(lf.Data.Allocations) != 1 {
		t.Errorf("expected 1 allocation, got %d", len(lf.Data.Allocations))
	}

	// Verify the right one was deleted
	if _, exists := lf.Data.Allocations["20001"]; exists {
		t.Error("allocation 20001 should have been deleted")
	}
	if _, exists := lf.Data.Allocations["20002"]; !exists {
		t.Error("allocation 20002 should still exist")
	}

	// Delete non-existent port (should not error)
	lf.DeletePort(99999)
}

func TestLockedFile_AllPorts(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "allocations.json")

	lf, err := OpenLocked(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lf.Close()

	t.Run("returns empty slice for no allocations", func(t *testing.T) {
		ports := lf.AllPorts()
		if len(ports) != 0 {
			t.Errorf("expected empty slice, got %v", ports)
		}
	})

	now := time.Now().UTC()
	lf.SetAllocation(20003, &Allocation{
		Directory: "/a", Name: "x", AssignedAt: now, LastUsedAt: now,
	})
	lf.SetAllocation(20001, &Allocation{
		Directory: "/b", Name: "y", AssignedAt: now, LastUsedAt: now,
	})
	lf.SetAllocation(20002, &Allocation{
		Directory: "/c", Name: "z", AssignedAt: now, LastUsedAt: now,
	})

	t.Run("returns all port numbers", func(t *testing.T) {
		ports := lf.AllPorts()
		if len(ports) != 3 {
			t.Fatalf("expected 3 ports, got %d", len(ports))
		}

		// Sort for consistent comparison
		sort.Ints(ports)
		expected := []int{20001, 20002, 20003}
		for i, p := range ports {
			if p != expected[i] {
				t.Errorf("ports[%d] = %d, want %d", i, p, expected[i])
			}
		}
	})
}

func TestLockedFile_Save(t *testing.T) {
	t.Run("persists changes to disk", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "allocations.json")

		// First, create and modify
		lf, err := OpenLocked(path, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		now := time.Now().UTC()
		lf.SetAllocation(20001, &Allocation{
			Directory: "/project/foo", Name: "main",
			AssignedAt: now, LastUsedAt: now, Locked: true,
		})
		lf.Data.LastIssuedPort = 20001

		if err := lf.Save(); err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		lf.Close()

		// Reopen and verify
		lf2, err := OpenLocked(path, false)
		if err != nil {
			t.Fatalf("unexpected error on reopen: %v", err)
		}
		defer lf2.Close()

		if lf2.Data.LastIssuedPort != 20001 {
			t.Errorf("LastIssuedPort = %d, want 20001", lf2.Data.LastIssuedPort)
		}

		alloc := lf2.Data.Allocations["20001"]
		if alloc == nil {
			t.Fatal("allocation not found after reload")
		}
		if alloc.Directory != "/project/foo" {
			t.Errorf("Directory = %q, want %q", alloc.Directory, "/project/foo")
		}
		if !alloc.Locked {
			t.Error("Locked should be true")
		}
	})

	t.Run("uses atomic write (temp file + rename)", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "allocations.json")

		lf, err := OpenLocked(path, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer lf.Close()

		lf.SetAllocation(20001, &Allocation{
			Directory: "/test", Name: "test",
			AssignedAt: time.Now(), LastUsedAt: time.Now(),
		})

		if err := lf.Save(); err != nil {
			t.Fatalf("Save() error: %v", err)
		}

		// Verify no temp files left behind
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}

		for _, entry := range entries {
			if entry.Name() != "allocations.json" {
				t.Errorf("unexpected file in directory: %s", entry.Name())
			}
		}
	})
}

func TestLockedFile_NilSafety(t *testing.T) {
	var lf *LockedFile

	t.Run("FindByDirectoryName on nil receiver", func(t *testing.T) {
		port, alloc := lf.FindByDirectoryName("/any", "name")
		if port != 0 {
			t.Errorf("port = %d, want 0", port)
		}
		if alloc != nil {
			t.Error("alloc should be nil")
		}
	})

	t.Run("DeletePort on nil receiver does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DeletePort panicked: %v", r)
			}
		}()
		lf.DeletePort(12345)
	})

	t.Run("SetAllocation on nil receiver does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SetAllocation panicked: %v", r)
			}
		}()
		lf.SetAllocation(12345, nil)
	})

	t.Run("AllPorts on nil receiver", func(t *testing.T) {
		ports := lf.AllPorts()
		if ports != nil {
			t.Errorf("expected nil, got %v", ports)
		}
	})

	t.Run("Close on nil receiver does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Close panicked: %v", r)
			}
		}()
		err := lf.Close()
		if err != nil {
			t.Errorf("Close() error: %v", err)
		}
	})

	t.Run("Save on nil receiver returns error", func(t *testing.T) {
		err := lf.Save()
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestDefaultFile(t *testing.T) {
	f := DefaultFile()

	if f.Version != 1 {
		t.Errorf("Version = %d, want 1", f.Version)
	}
	if f.LastIssuedPort != 0 {
		t.Errorf("LastIssuedPort = %d, want 0", f.LastIssuedPort)
	}
	if f.Allocations == nil {
		t.Error("Allocations should not be nil")
	}
	if len(f.Allocations) != 0 {
		t.Errorf("Allocations should be empty, got %d", len(f.Allocations))
	}
}
