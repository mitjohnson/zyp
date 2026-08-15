package docker

import (
	"fmt"
	"strings"
	"zyp/internal/provider"

	"github.com/docker/docker/api/types/container"
)

func parseLabels(containerName, containerID string, labels map[string]string, mounts []container.MountPoint) (provider.Target, bool, error) {
	if labels["backup.enable"] != "true" {
		return provider.Target{}, false, nil
	}

	name := labels["backup.name"]
	if name == "" {
		name = containerName
	}

	kindStr := labels["backup.kind"]

	if kindStr != "sqlite" && kindStr != "postgres" {
		return provider.Target{}, false, fmt.Errorf("container %s: unknown backup.kind %q", containerName, kindStr)
	}

	if kindStr == "sqlite" {
		backupPath := labels["backup.path"]

		if backupPath == "" {
			return provider.Target{}, false, fmt.Errorf("container %s: no backup path specified", containerName)
		}

		for _, mount := range mounts {
			if backupPath == mount.Destination || strings.HasPrefix(backupPath, mount.Destination+"/") {
				hostPath := mount.Source + strings.TrimPrefix(backupPath, mount.Destination)
				target := provider.Target{
					Name:         name,
					Kind:         provider.Kind(kindStr),
					Source:       hostPath,
					ContainerRef: containerID,
					Labels:       labels,
				}
				return target, true, nil
			}
		}

		return provider.Target{}, false, fmt.Errorf("container %s: backup path %s not found in container mounts", containerName, backupPath)
	}

	target := provider.Target{
		Name:         name,
		Kind:         provider.Kind(kindStr),
		ContainerRef: containerID,
		Labels:       labels,
	}
	return target, true, nil
}
