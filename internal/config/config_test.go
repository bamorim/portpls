package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		// Standard Go durations
		{"24h", 24 * time.Hour, false},
		{"1h30m", 90 * time.Minute, false},
		{"30s", 30 * time.Second, false},
		{"1h", time.Hour, false},
		{"30m", 30 * time.Minute, false},

		// Custom day syntax
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},

		// Combined day durations
		{"1d12h", 36 * time.Hour, false},
		{"2d6h30m", 54*time.Hour + 30*time.Minute, false},
		{"1d1h1m1s", 25*time.Hour + 1*time.Minute + 1*time.Second, false},

		// Zero is valid (disables TTL)
		{"0", 0, false},

		// Whitespace handling
		{" 24h ", 24 * time.Hour, false},
		{"  1d  ", 24 * time.Hour, false},

		// Invalid formats
		{"", 0, true},
		{"   ", 0, true},
		{"abc", 0, true},
		{"24", 0, true},  // number without unit
		{"d", 0, true},   // unit without number
		{"24x", 0, true}, // invalid unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid default config",
			config: Config{
				PortStart:     20000,
				PortEnd:       22000,
				FreezePeriod:  "24h",
				AllocationTTL: "0",
			},
			wantErr: false,
		},
		{
			name: "valid config with day freeze",
			config: Config{
				PortStart:     10000,
				PortEnd:       10100,
				FreezePeriod:  "7d",
				AllocationTTL: "30d",
			},
			wantErr: false,
		},
		{
			name: "port_start zero",
			config: Config{
				PortStart:     0,
				PortEnd:       22000,
				FreezePeriod:  "24h",
				AllocationTTL: "0",
			},
			wantErr: true,
		},
		{
			name: "port_start negative",
			config: Config{
				PortStart:     -1,
				PortEnd:       22000,
				FreezePeriod:  "24h",
				AllocationTTL: "0",
			},
			wantErr: true,
		},
		{
			name: "port_end zero",
			config: Config{
				PortStart:     20000,
				PortEnd:       0,
				FreezePeriod:  "24h",
				AllocationTTL: "0",
			},
			wantErr: true,
		},
		{
			name: "port_start greater than port_end",
			config: Config{
				PortStart:     25000,
				PortEnd:       20000,
				FreezePeriod:  "24h",
				AllocationTTL: "0",
			},
			wantErr: true,
		},
		{
			name: "invalid freeze_period",
			config: Config{
				PortStart:     20000,
				PortEnd:       22000,
				FreezePeriod:  "invalid",
				AllocationTTL: "0",
			},
			wantErr: true,
		},
		{
			name: "invalid allocation_ttl",
			config: Config{
				PortStart:     20000,
				PortEnd:       22000,
				FreezePeriod:  "24h",
				AllocationTTL: "bad",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"~/config", filepath.Join(home, "config")},
		{"~/.config/portpls", filepath.Join(home, ".config/portpls")},
		{"~/", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"./relative", "./relative"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExpandPath(tt.input)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("creates default config when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "config.json")

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check defaults
		if cfg.PortStart != 20000 {
			t.Errorf("PortStart = %d, want 20000", cfg.PortStart)
		}
		if cfg.PortEnd != 22000 {
			t.Errorf("PortEnd = %d, want 22000", cfg.PortEnd)
		}
		if cfg.FreezePeriod != "24h" {
			t.Errorf("FreezePeriod = %q, want %q", cfg.FreezePeriod, "24h")
		}
		if cfg.AllocationTTL != "0" {
			t.Errorf("AllocationTTL = %q, want %q", cfg.AllocationTTL, "0")
		}

		// Verify file was created
		if _, err := os.Stat(path); err != nil {
			t.Errorf("config file was not created: %v", err)
		}
	})

	t.Run("creates nested directories if needed", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "nested", "dir", "config.json")

		_, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(path); err != nil {
			t.Errorf("config file was not created: %v", err)
		}
	})

	t.Run("loads existing config with partial values", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "config.json")

		// Write partial config - only port_end (higher than default start)
		err := os.WriteFile(path, []byte(`{"port_end": 25000}`), 0644)
		if err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.PortEnd != 25000 {
			t.Errorf("PortEnd = %d, want 25000", cfg.PortEnd)
		}
		// Other values should be defaults
		if cfg.PortStart != 20000 {
			t.Errorf("PortStart = %d, want 20000 (default)", cfg.PortStart)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "config.json")

		err := os.WriteFile(path, []byte(`{invalid json`), 0644)
		if err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		_, err = Load(path)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		_, err := Load("")
		if err == nil {
			t.Error("expected error for empty path, got nil")
		}
	})

	t.Run("validates loaded config", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "config.json")

		// Write invalid config (start > end)
		err := os.WriteFile(path, []byte(`{"port_start": 30000, "port_end": 20000}`), 0644)
		if err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		_, err = Load(path)
		if err == nil {
			t.Error("expected validation error, got nil")
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("persists config to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "config.json")

		cfg := Config{
			PortStart:     25000,
			PortEnd:       26000,
			FreezePeriod:  "7d",
			AllocationTTL: "30d",
			LogFile:       "/var/log/portpls.log",
		}

		if err := Save(path, cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Load it back
		loaded, err := Load(path)
		if err != nil {
			t.Fatalf("failed to load saved config: %v", err)
		}

		if loaded.PortStart != cfg.PortStart {
			t.Errorf("PortStart = %d, want %d", loaded.PortStart, cfg.PortStart)
		}
		if loaded.PortEnd != cfg.PortEnd {
			t.Errorf("PortEnd = %d, want %d", loaded.PortEnd, cfg.PortEnd)
		}
		if loaded.FreezePeriod != cfg.FreezePeriod {
			t.Errorf("FreezePeriod = %q, want %q", loaded.FreezePeriod, cfg.FreezePeriod)
		}
		if loaded.AllocationTTL != cfg.AllocationTTL {
			t.Errorf("AllocationTTL = %q, want %q", loaded.AllocationTTL, cfg.AllocationTTL)
		}
		if loaded.LogFile != cfg.LogFile {
			t.Errorf("LogFile = %q, want %q", loaded.LogFile, cfg.LogFile)
		}
	})

	t.Run("creates nested directories if needed", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "nested", "dir", "config.json")

		cfg := Default()
		if err := Save(path, cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(path); err != nil {
			t.Errorf("config file was not created: %v", err)
		}
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		err := Save("", Default())
		if err == nil {
			t.Error("expected error for empty path, got nil")
		}
	})
}

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.PortStart != 20000 {
		t.Errorf("PortStart = %d, want 20000", cfg.PortStart)
	}
	if cfg.PortEnd != 22000 {
		t.Errorf("PortEnd = %d, want 22000", cfg.PortEnd)
	}
	if cfg.FreezePeriod != "24h" {
		t.Errorf("FreezePeriod = %q, want %q", cfg.FreezePeriod, "24h")
	}
	if cfg.AllocationTTL != "0" {
		t.Errorf("AllocationTTL = %q, want %q", cfg.AllocationTTL, "0")
	}
	if cfg.LogFile != "" {
		t.Errorf("LogFile = %q, want empty string", cfg.LogFile)
	}
}

func TestFreezeDuration(t *testing.T) {
	cfg := Config{FreezePeriod: "7d"}
	d, err := cfg.FreezeDuration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 7*24*time.Hour {
		t.Errorf("FreezeDuration() = %v, want %v", d, 7*24*time.Hour)
	}
}

func TestTTLDuration(t *testing.T) {
	cfg := Config{AllocationTTL: "30d"}
	d, err := cfg.TTLDuration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 30*24*time.Hour {
		t.Errorf("TTLDuration() = %v, want %v", d, 30*24*time.Hour)
	}
}
