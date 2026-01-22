package app

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bamorim/portpls/internal/allocations"
	"github.com/bamorim/portpls/internal/docker"
	"github.com/bamorim/portpls/internal/process"
)

type ScanResult struct {
	Lines []string
	Added int
	Start int
	End   int
}

func Scan(opts Options) (ScanResult, error) {
	result := ScanResult{}
	err := withContext(opts, true, func(ctx *context) error {
		start := ctx.config.PortStart
		end := ctx.config.PortEnd
		result.Start = start
		result.End = end
		now := time.Now().UTC()
		added := 0
		for portNum := start; portNum <= end; portNum++ {
			if ctx.portChecker.IsFree(portNum) {
				continue
			}
			if _, exists := ctx.allocFile.Data.Allocations[strconv.Itoa(portNum)]; exists {
				result.Lines = append(result.Lines, fmt.Sprintf("Port %d: already allocated", portNum))
				continue
			}
			procInfo, err := process.FindByPort(portNum)
			dir := ""
			procLabel := "unknown"
			if err == nil && procInfo != nil {
				procLabel = procInfo.Command
				if procInfo.Command == "docker-proxy" {
					if dockerDir, derr := docker.FindWorkingDirByPort(portNum); derr == nil {
						dir = dockerDir
					}
				}
				if dir == "" && procInfo.Cwd != "" {
					dir = procInfo.Cwd
				}
			}
			if dir == "" {
				dir = fmt.Sprintf("(unknown:%d)", portNum)
			}
			alloc := &allocations.Allocation{
				Directory:  dir,
				Name:       "main",
				AssignedAt: now,
				LastUsedAt: now,
				Locked:     false,
			}
			ctx.allocFile.SetAllocation(portNum, alloc)
			if portNum > ctx.allocFile.Data.LastIssuedPort {
				ctx.allocFile.Data.LastIssuedPort = portNum
			}
			_ = ctx.logger.Event("ALLOC_ADD", fmt.Sprintf("port=%d dir=%s name=main", portNum, dir))
			result.Lines = append(result.Lines, fmt.Sprintf("Port %d: used by %s - recorded", portNum, procLabel))
			added++
		}
		if err := ctx.allocFile.Save(); err != nil {
			return err
		}
		result.Added = added
		return nil
	})
	if err != nil {
		return ScanResult{}, NewCodeError(1, err)
	}
	return result, nil
}
