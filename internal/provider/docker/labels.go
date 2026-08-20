package docker

import (
	"fmt"
	"strings"
	"zyp/internal/target"

	"github.com/docker/docker/api/types/container"
)

func parseLabels(containerName, containerID string, labels map[string]string, mounts []container.MountPoint) (target.Target, bool, error) {
	if labels["zyp.enable"] != "true" {
		return target.Target{}, false, nil
	}

	repository := labels["zyp.repository"]
	compress := labels["zyp.compress"] == "true"
	name := labels["zyp.name"]
	if name == "" {
		name = containerName
	}

	kindStr := labels["zyp.kind"]

	if kindStr != "sqlite" && kindStr != "postgres" {
		return target.Target{}, false, fmt.Errorf("container %s: unknown zyp.kind %q", containerName, kindStr)
	}

	if kindStr == "sqlite" {
		backupPath := labels["zyp.path"]

		if backupPath == "" {
			return target.Target{}, false, fmt.Errorf("container %s: no backup path specified", containerName)
		}

		for _, mount := range mounts {
			if backupPath == mount.Destination || strings.HasPrefix(backupPath, mount.Destination+"/") {
				hostPath := mount.Source + strings.TrimPrefix(backupPath, mount.Destination)
				target := target.Target{
					Name:       name,
					Kind:       target.Kind(kindStr),
					Source:     hostPath,
					Repository: repository,
					Compress:   compress,
				}
				return target, true, nil
			}
		}

		return target.Target{}, false, fmt.Errorf("container %s: backup path %s not found in container mounts", containerName, backupPath)
	}

	target := target.Target{
		Name:       name,
		Kind:       target.Kind(kindStr),
		Repository: repository,
		Compress:   compress,
	}
	return target, true, nil
}
