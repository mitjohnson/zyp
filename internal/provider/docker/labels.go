package docker

import (
	"fmt"
	"strings"
	"zyp/internal/target"

	"github.com/docker/docker/api/types/container"
)

type kindResolver func(containerName string, labels map[string]string, mounts []container.MountPoint) (source string, err error)

var kindResolvers = map[target.Kind]kindResolver{
	target.KindSQLite:   resolveHostPath,
	target.KindFile:     resolveHostPath,
	target.KindPostgres: resolvePostgres,
}

func parseLabels(containerName string, labels map[string]string, mounts []container.MountPoint) (target.Target, bool, error) {
	if labels["zyp.enable"] != "true" {
		return target.Target{}, false, nil
	}

	repository := labels["zyp.repository"]
	compress := labels["zyp.compress"] == "true"
	name := labels["zyp.name"]
	if name == "" {
		name = containerName
	}

	kind := target.Kind(labels["zyp.kind"])
	resolve, ok := kindResolvers[kind]
	if !ok {
		return target.Target{}, false, fmt.Errorf("container %s: unsupported kind %s", containerName, labels["zyp.kind"])
	}

	source, err := resolve(containerName, labels, mounts)
	if err != nil {
		return target.Target{}, false, err
	}

	target := target.Target{
		Name:       name,
		Kind:       kind,
		Source:     source,
		Repository: repository,
		Compress:   compress,
	}
	return target, true, nil
}

func resolveHostPath(containerName string, labels map[string]string, mounts []container.MountPoint) (string, error) {
	backupPath := labels["zyp.file-path"]

	if backupPath == "" {
		return "", fmt.Errorf("container %s: zyp.file-path label is not set", containerName)
	}

	for _, mount := range mounts {
		if backupPath == mount.Destination || strings.HasPrefix(backupPath, mount.Destination+"/") {
			return mount.Source + strings.TrimPrefix(backupPath, mount.Destination), nil
		}
	}

	return "", fmt.Errorf("container %s: zyp.file-path %s not found", containerName, backupPath)
}

func resolvePostgres(_ string, _ map[string]string, _ []container.MountPoint) (string, error) {
	return "", nil // stub for testing, implement once collecter is implemented
}
