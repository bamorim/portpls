package app

import (
	"fmt"
	"strconv"

	"github.com/bamorim/portpls/internal/config"
)

func ConfigShow(opts Options) ([]string, error) {
	cfg, err := config.Load(resolveOptions(opts).ConfigPath)
	if err != nil {
		return nil, err
	}
	lines := []string{
		fmt.Sprintf("port_start: %d", cfg.PortStart),
		fmt.Sprintf("port_end: %d", cfg.PortEnd),
		fmt.Sprintf("freeze_period: %s", cfg.FreezePeriod),
		fmt.Sprintf("allocation_ttl: %s", cfg.AllocationTTL),
	}
	if cfg.LogFile != "" {
		lines = append(lines, fmt.Sprintf("log_file: %s", cfg.LogFile))
	} else {
		lines = append(lines, "log_file: ")
	}
	return lines, nil
}

func ConfigGet(opts Options, key string) (string, error) {
	cfg, err := config.Load(resolveOptions(opts).ConfigPath)
	if err != nil {
		return "", err
	}
	switch key {
	case "port_start":
		return fmt.Sprintf("%d", cfg.PortStart), nil
	case "port_end":
		return fmt.Sprintf("%d", cfg.PortEnd), nil
	case "freeze_period":
		return cfg.FreezePeriod, nil
	case "allocation_ttl":
		return cfg.AllocationTTL, nil
	case "log_file":
		return cfg.LogFile, nil
	default:
		return "", NewCodeError(1, ErrInvalidConfigKey)
	}
}

func ConfigSet(opts Options, key, value string) (string, error) {
	cfgPath := resolveOptions(opts).ConfigPath
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return "", err
	}
	updated, err := setConfigValue(cfg, key, value)
	if err != nil {
		return "", NewCodeError(1, err)
	}
	if err := config.Save(cfgPath, updated); err != nil {
		return "", err
	}
	return fmt.Sprintf("Set %s to %s", key, value), nil
}

func setConfigValue(cfg config.Config, key, value string) (config.Config, error) {
	switch key {
	case "port_start":
		val, err := strconv.Atoi(value)
		if err != nil {
			return cfg, ErrInvalidConfigValue
		}
		cfg.PortStart = val
	case "port_end":
		val, err := strconv.Atoi(value)
		if err != nil {
			return cfg, ErrInvalidConfigValue
		}
		cfg.PortEnd = val
	case "freeze_period":
		if _, err := config.ParseDuration(value); err != nil {
			return cfg, ErrInvalidConfigValue
		}
		cfg.FreezePeriod = value
	case "allocation_ttl":
		if _, err := config.ParseDuration(value); err != nil {
			return cfg, ErrInvalidConfigValue
		}
		cfg.AllocationTTL = value
	case "log_file":
		cfg.LogFile = value
	default:
		return cfg, ErrInvalidConfigKey
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}
