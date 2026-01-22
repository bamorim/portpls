package app

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bamorim/portpls/internal/allocations"
	"github.com/bamorim/portpls/internal/config"
	"github.com/bamorim/portpls/internal/logger"
	"github.com/bamorim/portpls/internal/port"
)

type Options struct {
	ConfigPath      string
	AllocationsPath string
	Directory       DirectorySelector // resolves to one directory
	Verbose         bool
	PortChecker     port.Checker // optional, defaults to TCPChecker
}

type context struct {
	config      config.Config
	allocFile   *allocations.LockedFile
	logger      logger.Logger
	directory   string
	portChecker port.Checker
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
	checker := opts.PortChecker
	if checker == nil {
		checker = port.TCPChecker{}
	}
	ctx := &context{config: cfg, allocFile: allocFile, logger: log, portChecker: checker}
	ctx.directory, err = opts.Directory.ResolveDirectory()
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
