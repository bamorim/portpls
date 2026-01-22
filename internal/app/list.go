package app

import (
	"sort"
	"strconv"
	"time"
)

type AllocationEntry struct {
	Port       int
	Directory  string
	Name       string
	Status     string
	Locked     bool
	AssignedAt time.Time
	LastUsedAt time.Time
}

func ListAllocations(opts Options) ([]AllocationEntry, error) {
	entries := []AllocationEntry{}
	err := withContext(opts, true, func(ctx *context) error {
		for portStr, alloc := range ctx.allocFile.Data.Allocations {
			portNum, err := strconv.Atoi(portStr)
			if err != nil {
				continue
			}
			status := "busy"
			if ctx.portChecker.IsFree(portNum) {
				status = "free"
			}
			entries = append(entries, AllocationEntry{
				Port:       portNum,
				Directory:  alloc.Directory,
				Name:       alloc.Name,
				Status:     status,
				Locked:     alloc.Locked,
				AssignedAt: alloc.AssignedAt,
				LastUsedAt: alloc.LastUsedAt,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Port < entries[j].Port })
	return entries, nil
}
