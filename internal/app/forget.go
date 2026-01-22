package app

import (
	"fmt"
	"strconv"
)

type ForgetResult struct {
	Message string
}

// Forget removes port allocations based on the provided filter.
// - filter: determines which directories to consider
// - name: the allocation name to remove (if nameSet is true)
// - nameSet: whether the name parameter should be used
// - deleteAll: if true, deletes all allocations matching the filter
// - confirm: callback for user confirmation (required for global deletes)
func Forget(opts Options, filter DirectoryFilter, name string, nameSet bool, deleteAll bool, confirm func() bool) (ForgetResult, error) {
	if !nameSet && !deleteAll {
		return ForgetResult{}, NewCodeError(2, ErrMissingFlags)
	}
	if filter == nil {
		return ForgetResult{}, fmt.Errorf("filter cannot be nil")
	}

	var result ForgetResult
	err := withContext(opts, true, func(ctx *context) error {
		if deleteAll {
			// If confirm callback is provided, we need user confirmation
			if confirm != nil && !confirm() {
				return ErrConfirmDeclined
			}

			count := 0
			for portStr, alloc := range ctx.allocFile.Data.Allocations {
				if filter(alloc.Directory) {
					delete(ctx.allocFile.Data.Allocations, portStr)
					count++
				}
			}

			_ = ctx.logger.Event("ALLOC_DELETE_ALL", fmt.Sprintf("count=%d", count))
			if err := ctx.allocFile.Save(); err != nil {
				return err
			}
			result.Message = fmt.Sprintf("Cleared %d allocation(s)", count)
			return nil
		}

		// Delete named allocation across all matching directories
		var deleted []struct {
			port int
			dir  string
		}
		for portStr, alloc := range ctx.allocFile.Data.Allocations {
			if alloc.Name == name && filter(alloc.Directory) {
				portNum, _ := strconv.Atoi(portStr)
				ctx.allocFile.DeletePort(portNum)
				_ = ctx.logger.Event("ALLOC_DELETE", fmt.Sprintf("port=%d dir=%s name=%s", portNum, alloc.Directory, name))
				deleted = append(deleted, struct {
					port int
					dir  string
				}{portNum, alloc.Directory})
			}
		}

		if len(deleted) == 0 {
			result.Message = fmt.Sprintf("No allocation found for '%s'", name)
			return nil
		}

		if err := ctx.allocFile.Save(); err != nil {
			return err
		}

		if len(deleted) == 1 {
			result.Message = fmt.Sprintf("Cleared allocation '%s' for %s (was port %d)", name, deleted[0].dir, deleted[0].port)
		} else {
			result.Message = fmt.Sprintf("Cleared %d allocation(s) for '%s'", len(deleted), name)
		}
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
