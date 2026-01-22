package process

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Info struct {
	PID     int
	Command string
	Cwd     string
}

func FindByPort(port int) (*Info, error) {
	cmd := exec.Command("lsof", "-nP", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN", "-Fpnc")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := bytes.Split(out, []byte{'\n'})
	info := &Info{}
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			pid, err := strconv.Atoi(string(line[1:]))
			if err == nil {
				info.PID = pid
			}
		case 'c':
			info.Command = string(line[1:])
		}
		if info.PID != 0 && info.Command != "" {
			break
		}
	}
	if info.PID == 0 {
		return nil, errors.New("process not found")
	}
	cwd, _ := findCwd(info.PID)
	info.Cwd = cwd
	return info, nil
}

func findCwd(pid int) (string, error) {
	switch runtime.GOOS {
	case "linux":
		path := filepath.Join("/proc", strconv.Itoa(pid), "cwd")
		return os.Readlink(path)
	case "darwin":
		cmd := exec.Command("lsof", "-p", strconv.Itoa(pid), "-a", "-d", "cwd", "-Fn")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "n") {
				return strings.TrimPrefix(line, "n"), nil
			}
		}
		return "", errors.New("cwd not found")
	default:
		return "", errors.New("unsupported platform")
	}
}
