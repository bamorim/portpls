package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"portpls/internal/allocations"
	"portpls/internal/config"
	"portpls/internal/logger"
)

type Options struct {
	ConfigPath      string
	AllocationsPath string
	Directory       string
	Verbose         bool
}

type context struct {
	config    config.Config
	allocFile *allocations.LockedFile
	logger    logger.Logger
	directory string
}

func withContext(opts Options, exclusive bool, fn func(*context) error) error {
	resolved := resolveOptions(opts)
	cfg, err := config.Load(resolved.ConfigPath)
	if err != nil {
		return err
	}
	allocFile, err := allocations.OpenLocked(resolved.AllocationsPath, exclusive)
	if err != nil {
		return err
	}
	defer allocFile.Close()
	log := logger.Logger{Path: cfg.LogFile, Verbose: resolved.Verbose}
	ctx := &context{config: cfg, allocFile: allocFile, logger: log}
	ctx.directory, err = resolveDirectory(resolved.Directory)
	if err != nil {
		return err
	}
	changed, err := applyTTL(ctx)
	if err != nil {
		return err
	}
	if changed {
		if err := ctx.allocFile.Save(); err != nil {
			return err
		}
	}
	return fn(ctx)
}

func resolveOptions(opts Options) Options {
	if opts.ConfigPath == "" {
		opts.ConfigPath = DefaultConfigPath()
	}
	if opts.AllocationsPath == "" {
		opts.AllocationsPath = DefaultAllocationsPath()
	}
	opts.ConfigPath = config.ExpandPath(opts.ConfigPath)
	opts.AllocationsPath = config.ExpandPath(opts.AllocationsPath)
	return opts
}

func resolveDirectory(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Abs(wd)
}

func applyTTL(ctx *context) (bool, error) {
	if ctx == nil {
		return false, nil
	}
	ttl, err := ctx.config.TTLDuration()
	if err != nil || ttl == 0 {
		return false, nil
	}
	now := time.Now().UTC()
	changed := false
	for portStr, alloc := range ctx.allocFile.Data.Allocations {
		if alloc.LastUsedAt.Add(ttl).Before(now) {
			portNum, _ := strconv.Atoi(portStr)
			delete(ctx.allocFile.Data.Allocations, portStr)
			_ = ctx.logger.Event("ALLOC_EXPIRE", fmt.Sprintf("port=%d dir=%s name=%s ttl=%s", portNum, alloc.Directory, alloc.Name, ctx.config.AllocationTTL))
			changed = true
		}
	}
	return changed, nil
}
