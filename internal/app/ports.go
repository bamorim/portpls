package app

import (
	"strconv"
	"time"

	"portpls/internal/port"
)

func findFreePort(ctx *context, name string, now time.Time) (int, error) {
	start := ctx.config.PortStart
	end := ctx.config.PortEnd
	if start > end {
		return 0, ErrInvalidPortRange
	}
	freeze, _ := ctx.config.FreezeDuration()
	candidate := ctx.allocFile.Data.LastIssuedPort + 1
	if candidate < start || candidate > end {
		candidate = start
	}
	maxAttempts := end - start + 1
	attempts := 0
	for attempts < maxAttempts {
		portNum := candidate
		candidate++
		if candidate > end {
			candidate = start
		}
		attempts++
		if alloc, exists := ctx.allocFile.Data.Allocations[strconv.Itoa(portNum)]; exists {
			if alloc.Directory == ctx.directory && alloc.Name == name {
				continue
			}
			if alloc.Locked {
				continue
			}
			if freeze > 0 && alloc.AssignedAt.Add(freeze).After(now) {
				continue
			}
			continue
		}
		if !port.IsFree(portNum) {
			continue
		}
		return portNum, nil
	}
	return 0, ErrNoFreePorts
}
