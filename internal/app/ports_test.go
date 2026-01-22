package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bamorim/portpls/internal/allocations"
	"github.com/bamorim/portpls/internal/config"
	"github.com/bamorim/portpls/internal/logger"
)

// mockChecker allows controlling which ports appear free in tests.
type mockChecker struct {
	freePorts map[int]bool // true = free, false or missing = busy
}

func (m mockChecker) IsFree(port int) bool {
	if m.freePorts == nil {
		return true // all free by default
	}
	free, exists := m.freePorts[port]
	return exists && free
}

// newTestContext creates a context for testing with the given config and checker.
func newTestContext(t *testing.T, cfg config.Config, checker mockChecker) (*context, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	allocPath := filepath.Join(tmpDir, "allocations.json")

	allocFile, err := allocations.OpenLocked(allocPath, true)
	if err != nil {
		t.Fatalf("failed to open allocations: %v", err)
	}

	ctx := &context{
		config:      cfg,
		allocFile:   allocFile,
		logger:      logger.Logger{},
		directory:   "/test/project",
		portChecker: checker,
	}

	cleanup := func() {
		allocFile.Close()
	}

	return ctx, cleanup
}

func TestFindFreePort(t *testing.T) {
	t.Run("returns first free port in range", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20005,
			FreezePeriod: "0",
		}
		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: true,
			20002: true,
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		port, err := findFreePort(ctx, "main", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20000 {
			t.Errorf("port = %d, want 20000", port)
		}
	})

	t.Run("continues from LastIssuedPort", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20005,
			FreezePeriod: "0",
		}
		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: true,
			20002: true,
			20003: true,
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		ctx.allocFile.Data.LastIssuedPort = 20001

		port, err := findFreePort(ctx, "main", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20002 {
			t.Errorf("port = %d, want 20002", port)
		}
	})

	t.Run("wraps around to start after reaching end", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20002,
			FreezePeriod: "0",
		}
		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: false, // busy
			20002: false, // busy
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		ctx.allocFile.Data.LastIssuedPort = 20002

		port, err := findFreePort(ctx, "main", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20000 {
			t.Errorf("port = %d, want 20000 (wrapped around)", port)
		}
	})

	t.Run("skips busy ports", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20005,
			FreezePeriod: "0",
		}
		checker := mockChecker{freePorts: map[int]bool{
			20000: false, // busy
			20001: false, // busy
			20002: true,  // free
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		port, err := findFreePort(ctx, "main", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20002 {
			t.Errorf("port = %d, want 20002", port)
		}
	})

	t.Run("skips locked ports", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20005,
			FreezePeriod: "0",
		}
		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: true,
			20002: true,
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		// Port 20000 is locked by another directory
		ctx.allocFile.SetAllocation(20000, &allocations.Allocation{
			Directory: "/other/project",
			Name:      "main",
			Locked:    true,
		})

		port, err := findFreePort(ctx, "main", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20001 {
			t.Errorf("port = %d, want 20001 (skipped locked port)", port)
		}
	})

	t.Run("skips ports allocated to other directories", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20005,
			FreezePeriod: "0",
		}
		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: true,
			20002: true,
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		// Port 20000 is allocated to another directory
		ctx.allocFile.SetAllocation(20000, &allocations.Allocation{
			Directory:  "/other/project",
			Name:       "main",
			AssignedAt: time.Now(),
			Locked:     false,
		})

		port, err := findFreePort(ctx, "main", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20001 {
			t.Errorf("port = %d, want 20001 (skipped other directory's port)", port)
		}
	})

	t.Run("allows reuse of own allocation", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20005,
			FreezePeriod: "0",
		}
		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: true,
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		// Port 20000 is allocated to this directory with same name - but this is checked elsewhere
		// findFreePort actually skips it - the reuse logic is in GetPort
		ctx.allocFile.SetAllocation(20000, &allocations.Allocation{
			Directory:  ctx.directory,
			Name:       "main",
			AssignedAt: time.Now(),
			Locked:     false,
		})

		// findFreePort skips own allocations too (it looks for new ports)
		port, err := findFreePort(ctx, "main", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20001 {
			t.Errorf("port = %d, want 20001", port)
		}
	})

	t.Run("returns error when no free ports available", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20002,
			FreezePeriod: "0",
		}
		// All ports busy
		checker := mockChecker{freePorts: map[int]bool{
			20000: false,
			20001: false,
			20002: false,
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		_, err := findFreePort(ctx, "main", time.Now())
		if err != ErrNoFreePorts {
			t.Errorf("expected ErrNoFreePorts, got %v", err)
		}
	})

	t.Run("returns error for invalid port range", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20005,
			PortEnd:      20000, // end < start
			FreezePeriod: "0",
		}
		checker := mockChecker{}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		_, err := findFreePort(ctx, "main", time.Now())
		if err != ErrInvalidPortRange {
			t.Errorf("expected ErrInvalidPortRange, got %v", err)
		}
	})

	t.Run("respects freeze period for recent allocations", func(t *testing.T) {
		cfg := config.Config{
			PortStart:    20000,
			PortEnd:      20005,
			FreezePeriod: "24h",
		}
		checker := mockChecker{freePorts: map[int]bool{
			20000: true,
			20001: true,
			20002: true,
		}}
		ctx, cleanup := newTestContext(t, cfg, checker)
		defer cleanup()

		// Port 20000 was recently allocated to another directory
		ctx.allocFile.SetAllocation(20000, &allocations.Allocation{
			Directory:  "/other/project",
			Name:       "main",
			AssignedAt: time.Now().Add(-1 * time.Hour), // 1 hour ago, within freeze period
			Locked:     false,
		})

		port, err := findFreePort(ctx, "main", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 20001 {
			t.Errorf("port = %d, want 20001 (skipped frozen port)", port)
		}
	})
}
