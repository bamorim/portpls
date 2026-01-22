package allocations

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

const (
	fileVersion   = 1
	lockWait      = 5 * time.Second
	lockSleepStep = 50 * time.Millisecond
)

type Allocation struct {
	Directory  string    `json:"directory"`
	Name       string    `json:"name"`
	AssignedAt time.Time `json:"assigned_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	Locked     bool      `json:"locked"`
}

type File struct {
	Version        int                    `json:"version"`
	LastIssuedPort int                    `json:"last_issued_port"`
	Allocations    map[string]*Allocation `json:"allocations"`
}

type LockedFile struct {
	Path string
	File *os.File
	Data *File
}

func DefaultFile() *File {
	return &File{
		Version:        fileVersion,
		LastIssuedPort: 0,
		Allocations:    map[string]*Allocation{},
	}
}

func OpenLocked(path string, exclusive bool) (*LockedFile, error) {
	if path == "" {
		return nil, errors.New("allocations path is empty")
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := writeFile(path, DefaultFile()); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	lockType := unix.LOCK_SH
	if exclusive {
		lockType = unix.LOCK_EX
	}
	if err := lockFile(file, lockType); err != nil {
		_ = file.Close()
		return nil, err
	}
	data, err := readFile(file)
	if err != nil {
		_ = unlockFile(file)
		_ = file.Close()
		return nil, err
	}
	return &LockedFile{Path: path, File: file, Data: data}, nil
}

func (l *LockedFile) Save() error {
	if l == nil || l.Data == nil {
		return errors.New("allocations data is nil")
	}
	return writeFile(l.Path, l.Data)
}

func (l *LockedFile) Close() error {
	if l == nil || l.File == nil {
		return nil
	}
	if err := unlockFile(l.File); err != nil {
		_ = l.File.Close()
		return err
	}
	return l.File.Close()
}

func (l *LockedFile) FindByDirectoryName(dir, name string) (int, *Allocation) {
	if l == nil || l.Data == nil {
		return 0, nil
	}
	for portStr, alloc := range l.Data.Allocations {
		if alloc.Directory == dir && alloc.Name == name {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return 0, nil
			}
			return port, alloc
		}
	}
	return 0, nil
}

func (l *LockedFile) DeletePort(port int) {
	if l == nil || l.Data == nil {
		return
	}
	delete(l.Data.Allocations, strconv.Itoa(port))
}

func (l *LockedFile) SetAllocation(port int, alloc *Allocation) {
	if l == nil || l.Data == nil {
		return
	}
	if l.Data.Allocations == nil {
		l.Data.Allocations = map[string]*Allocation{}
	}
	l.Data.Allocations[strconv.Itoa(port)] = alloc
}

func (l *LockedFile) AllPorts() []int {
	if l == nil || l.Data == nil {
		return nil
	}
	ports := make([]int, 0, len(l.Data.Allocations))
	for portStr := range l.Data.Allocations {
		port, err := strconv.Atoi(portStr)
		if err == nil {
			ports = append(ports, port)
		}
	}
	return ports
}

func readFile(file *os.File) (*File, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return DefaultFile(), nil
	}
	var out File
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse allocations: %w", err)
	}
	if out.Allocations == nil {
		out.Allocations = map[string]*Allocation{}
	}
	if out.Version == 0 {
		out.Version = fileVersion
	}
	return &out, nil
}

func writeFile(path string, data *File) error {
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".allocations-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(tmp.Name())
	}()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func lockFile(file *os.File, lockType int) error {
	deadline := time.Now().Add(lockWait)
	for {
		err := unix.Flock(int(file.Fd()), lockType|unix.LOCK_NB)
		if err == nil {
			return nil
		}
		if errors.Is(err, unix.EWOULDBLOCK) {
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out waiting for allocation lock")
			}
			time.Sleep(lockSleepStep)
			continue
		}
		return err
	}
}

func unlockFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

func ensureDir(path string) error {
	if path == "" || path == "." {
		return nil
	}
	return os.MkdirAll(path, 0o755)
}
