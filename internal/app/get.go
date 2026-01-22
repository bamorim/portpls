package app

import (
	"fmt"
	"time"

	"portpls/internal/allocations"
)

func GetPort(opts Options, name string) (int, error) {
	var result int
	err := withContext(opts, true, func(ctx *context) error {
		now := time.Now().UTC()
		if portNum, alloc := ctx.allocFile.FindByDirectoryName(ctx.directory, name); alloc != nil {
			if ctx.portChecker.IsFree(portNum) {
				alloc.LastUsedAt = now
				ctx.allocFile.SetAllocation(portNum, alloc)
				_ = ctx.logger.Event("ALLOC_UPDATE", fmt.Sprintf("port=%d (reused)", portNum))
				if err := ctx.allocFile.Save(); err != nil {
					return err
				}
				result = portNum
				return nil
			}
			ctx.allocFile.DeletePort(portNum)
			_ = ctx.logger.Event("ALLOC_DELETE", fmt.Sprintf("port=%d dir=%s name=%s", portNum, ctx.directory, name))
		}

		portNum, err := findFreePort(ctx, name, now)
		if err != nil {
			return err
		}
		alloc := &allocations.Allocation{
			Directory:  ctx.directory,
			Name:       name,
			AssignedAt: now,
			LastUsedAt: now,
			Locked:     false,
		}
		ctx.allocFile.SetAllocation(portNum, alloc)
		ctx.allocFile.Data.LastIssuedPort = portNum
		_ = ctx.logger.Event("ALLOC_ADD", fmt.Sprintf("port=%d dir=%s name=%s", portNum, ctx.directory, name))
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
