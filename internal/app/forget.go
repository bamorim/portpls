package app

import (
	"fmt"

	"portpls/internal/allocations"
)

type ForgetResult struct {
	Message string
}

func Forget(opts Options, name string, nameSet bool, all bool, allDirs bool, confirm func() bool) (ForgetResult, error) {
	if !nameSet && !all {
		return ForgetResult{}, NewCodeError(2, ErrMissingFlags)
	}
	var result ForgetResult
	err := withContext(opts, true, func(ctx *context) error {
		if all && allDirs {
			if confirm != nil && !confirm() {
				return ErrConfirmDeclined
			}
			count := len(ctx.allocFile.Data.Allocations)
			ctx.allocFile.Data.Allocations = map[string]*allocations.Allocation{}
			_ = ctx.logger.Event("ALLOC_DELETE_ALL", fmt.Sprintf("count=%d", count))
			if err := ctx.allocFile.Save(); err != nil {
				return err
			}
			result.Message = fmt.Sprintf("Cleared %d allocation(s)", count)
			return nil
		}

		if all {
			count := 0
			for portStr, alloc := range ctx.allocFile.Data.Allocations {
				if alloc.Directory == ctx.directory {
					delete(ctx.allocFile.Data.Allocations, portStr)
					count++
				}
			}
			_ = ctx.logger.Event("ALLOC_DELETE_ALL", fmt.Sprintf("count=%d", count))
			if err := ctx.allocFile.Save(); err != nil {
				return err
			}
			result.Message = fmt.Sprintf("Cleared %d allocation(s) for %s", count, ctx.directory)
			return nil
		}

		portNum, alloc := ctx.allocFile.FindByDirectoryName(ctx.directory, name)
		if alloc == nil {
			result.Message = fmt.Sprintf("No allocation found for %s (%s)", name, ctx.directory)
			return nil
		}
		ctx.allocFile.DeletePort(portNum)
		_ = ctx.logger.Event("ALLOC_DELETE", fmt.Sprintf("port=%d dir=%s name=%s", portNum, ctx.directory, name))
		if err := ctx.allocFile.Save(); err != nil {
			return err
		}
		result.Message = fmt.Sprintf("Cleared allocation '%s' for %s (was port %d)", name, ctx.directory, portNum)
		return nil
	})
	if err != nil {
		if err == ErrConfirmDeclined {
			return ForgetResult{}, NewCodeError(1, err)
		}
		return ForgetResult{}, err
	}
	return result, nil
}
