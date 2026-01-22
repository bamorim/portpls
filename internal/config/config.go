package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPortStart     = 20000
	defaultPortEnd       = 22000
	defaultFreezePeriod  = "24h"
	defaultAllocationTTL = "0"
)

// Config represents user configuration on disk.
type Config struct {
	PortStart     int    `json:"port_start"`
	PortEnd       int    `json:"port_end"`
	FreezePeriod  string `json:"freeze_period"`
	AllocationTTL string `json:"allocation_ttl"`
	LogFile       string `json:"log_file"`
}

type configOnDisk struct {
	PortStart     *int    `json:"port_start"`
	PortEnd       *int    `json:"port_end"`
	FreezePeriod  *string `json:"freeze_period"`
	AllocationTTL *string `json:"allocation_ttl"`
	LogFile       *string `json:"log_file"`
}

func Default() Config {
	return Config{
		PortStart:     defaultPortStart,
		PortEnd:       defaultPortEnd,
		FreezePeriod:  defaultFreezePeriod,
		AllocationTTL: defaultAllocationTTL,
		LogFile:       "",
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		return Config{}, errors.New("config path is empty")
	}
	path = ExpandPath(path)
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return Config{}, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		if err := Save(path, cfg); err != nil {
			return Config{}, err
		}
		return cfg, nil
	} else if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var raw configOnDisk
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	cfg := Default()
	if raw.PortStart != nil {
		cfg.PortStart = *raw.PortStart
	}
	if raw.PortEnd != nil {
		cfg.PortEnd = *raw.PortEnd
	}
	if raw.FreezePeriod != nil {
		cfg.FreezePeriod = strings.TrimSpace(*raw.FreezePeriod)
	}
	if raw.AllocationTTL != nil {
		cfg.AllocationTTL = strings.TrimSpace(*raw.AllocationTTL)
	}
	if raw.LogFile != nil {
		cfg.LogFile = strings.TrimSpace(*raw.LogFile)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if path == "" {
		return errors.New("config path is empty")
	}
	path = ExpandPath(path)
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func (c Config) Validate() error {
	if c.PortStart <= 0 {
		return fmt.Errorf("port_start must be > 0")
	}
	if c.PortEnd <= 0 {
		return fmt.Errorf("port_end must be > 0")
	}
	if c.PortStart > c.PortEnd {
		return fmt.Errorf("port_start must be <= port_end")
	}
	if _, err := ParseDuration(c.FreezePeriod); err != nil {
		return fmt.Errorf("invalid freeze_period: %w", err)
	}
	if _, err := ParseDuration(c.AllocationTTL); err != nil {
		return fmt.Errorf("invalid allocation_ttl: %w", err)
	}
	return nil
}

func (c Config) FreezeDuration() (time.Duration, error) {
	return ParseDuration(c.FreezePeriod)
}

func (c Config) TTLDuration() (time.Duration, error) {
	return ParseDuration(c.AllocationTTL)
}

func ExpandPath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}

func ParseDuration(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, errors.New("duration is empty")
	}
	if trimmed == "0" {
		return 0, nil
	}
	if strings.Contains(trimmed, "d") {
		return parseDayDuration(trimmed)
	}
	return time.ParseDuration(trimmed)
}

func parseDayDuration(input string) (time.Duration, error) {
	var total time.Duration
	remaining := strings.TrimSpace(input)
	for len(remaining) > 0 {
		numEnd := 0
		for numEnd < len(remaining) && remaining[numEnd] >= '0' && remaining[numEnd] <= '9' {
			numEnd++
		}
		if numEnd == 0 || numEnd == len(remaining) {
			return 0, fmt.Errorf("invalid duration segment: %s", remaining)
		}
		valueStr := remaining[:numEnd]
		unit := remaining[numEnd]
		value, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration value: %s", valueStr)
		}
		switch unit {
		case 'd':
			total += time.Hour * 24 * time.Duration(value)
		case 'h':
			total += time.Hour * time.Duration(value)
		case 'm':
			total += time.Minute * time.Duration(value)
		case 's':
			total += time.Second * time.Duration(value)
		default:
			return 0, fmt.Errorf("invalid duration unit: %c", unit)
		}
		remaining = strings.TrimSpace(remaining[numEnd+1:])
	}
	return total, nil
}

func ensureDir(path string) error {
	if path == "" || path == "." {
		return nil
	}
	return os.MkdirAll(path, 0o755)
}
