package docker

import (
	"fmt"
	"strings"
	"zyp/internal/provider"

	"github.com/docker/docker/api/types/container"
)

func parseLabels(containerName, containerID string, labels map[string]string, mounts []container.MountPoint) (provider.Target, bool, error) {
	if labels["zyp.enable"] != "true" {
		return provider.Target{}, false, nil
	}

	repository := labels["zyp.repository"]
	compress := labels["zyp.compress"] == "true"
	name := labels["zyp.name"]
	if name == "" {
		name = containerName
	}

	kindStr := labels["zyp.kind"]

	if kindStr != "sqlite" && kindStr != "postgres" {
		return provider.Target{}, false, fmt.Errorf("container %s: unknown zyp.kind %q", containerName, kindStr)
	}

	if kindStr == "sqlite" {
		backupPath := labels["zyp.path"]

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
					Repository:   repository,
					Compress:     compress,
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
		Repository:   repository,
		Compress:     compress,
		ContainerRef: containerID,
		Labels:       labels,
	}
	return target, true, nil
}
