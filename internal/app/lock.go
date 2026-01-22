package app

import (
	"fmt"
	"time"

	"portpls/internal/allocations"
)

func LockPort(opts Options, name string) (int, error) {
	var result int
	err := withContext(opts, true, func(ctx *context) error {
		now := time.Now().UTC()
		portNum, alloc := ctx.allocFile.FindByDirectoryName(ctx.directory, name)
		if alloc == nil {
			var err error
			portNum, err = findFreePort(ctx, name, now)
			if err != nil {
				return err
			}
			alloc = &allocations.Allocation{
				Directory:  ctx.directory,
				Name:       name,
				AssignedAt: now,
				LastUsedAt: now,
				Locked:     true,
			}
			ctx.allocFile.SetAllocation(portNum, alloc)
			ctx.allocFile.Data.LastIssuedPort = portNum
			_ = ctx.logger.Event("ALLOC_ADD", fmt.Sprintf("port=%d dir=%s name=%s", portNum, ctx.directory, name))
		} else {
			alloc.Locked = true
			alloc.LastUsedAt = now
			ctx.allocFile.SetAllocation(portNum, alloc)
		}
		_ = ctx.logger.Event("ALLOC_LOCK", fmt.Sprintf("port=%d locked=true", portNum))
		if err := ctx.allocFile.Save(); err != nil {
			return err
		}
		result = portNum
		return nil
	})
	if err != nil {
		if err == ErrNoFreePorts {
			return 0, NewCodeError(1, err)
		}
		return 0, err
	}
	return result, nil
}

func UnlockPort(opts Options, name string) (int, error) {
	var result int
	err := withContext(opts, true, func(ctx *context) error {
		portNum, alloc := ctx.allocFile.FindByDirectoryName(ctx.directory, name)
		if alloc == nil {
			return ErrAllocationNotFound
		}
		alloc.Locked = false
		ctx.allocFile.SetAllocation(portNum, alloc)
		_ = ctx.logger.Event("ALLOC_LOCK", fmt.Sprintf("port=%d locked=false", portNum))
		if err := ctx.allocFile.Save(); err != nil {
			return err
		}
		result = portNum
		return nil
	})
	if err != nil {
		if err == ErrAllocationNotFound {
			return 0, NewCodeError(1, err)
		}
		return 0, err
	}
	return result, nil
}
