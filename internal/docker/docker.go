package docker

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

type inspectResult struct {
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	Mounts []struct {
		Source string `json:"Source"`
	} `json:"Mounts"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

func FindWorkingDirByPort(port int) (string, error) {
	ids, err := dockerIDs()
	if err != nil {
		return "", err
	}
	for _, id := range ids {
		match, dir := inspectContainerForPort(id, port)
		if match {
			if dir != "" {
				return dir, nil
			}
		}
	}
	return "", errors.New("no docker container matched")
}

func dockerIDs() ([]string, error) {
	cmd := exec.Command("docker", "ps", "-q")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	ids := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ids = append(ids, line)
	}
	return ids, nil
}

func inspectContainerForPort(id string, port int) (bool, string) {
	cmd := exec.Command("docker", "inspect", id)
	out, err := cmd.Output()
	if err != nil {
		return false, ""
	}
	var results []inspectResult
	if err := json.Unmarshal(out, &results); err != nil {
		return false, ""
	}
	for _, result := range results {
		for _, bindings := range result.NetworkSettings.Ports {
			for _, binding := range bindings {
				if binding.HostPort == strconv.Itoa(port) {
					if dir := result.Config.Labels["com.docker.compose.project.working_dir"]; dir != "" {
						return true, dir
					}
					for _, mount := range result.Mounts {
						if mount.Source != "" {
							return true, mount.Source
						}
					}
					return true, ""
				}
			}
		}
	}
	return false, ""
}
