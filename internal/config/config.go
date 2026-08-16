package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultRepository string                `yaml:"defaultRepository"`
	Repositories      map[string]Repository `yaml:"repositories"`
	Providers         Providers             `yaml:"providers"`
}

type Repository struct {
	Engine string            `yaml:"engine"`
	Remote string            `yaml:"repo"`
	Env    map[string]string `yaml:"env"`
}

type Providers struct {
	Docker DockerProviderConfig `yaml:"docker"`
	File   FileProviderConfig   `yaml:"file"`
}

type DockerProviderConfig struct {
	Enabled bool `yaml:"enabled"`
}

type FileProviderConfig struct {
	Enabled bool               `yaml:"enabled"`
	Targets []FileTargetConfig `yaml:"targets"`
}

type FileTargetConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

func Load(path string) (Config, error) {
	rawConf, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var conf Config
	err = yaml.Unmarshal(rawConf, &conf)
	if err != nil {
		return Config{}, err
	}

	if err := Validate(conf); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return conf, nil
}

func Validate(conf Config) error {
	if len(conf.Repositories) == 0 {
		return fmt.Errorf("no repositories defined in configuration")
	}

	defaultRepo := conf.DefaultRepository
	if defaultRepo != "" {
		if _, ok := conf.Repositories[defaultRepo]; !ok {
			return fmt.Errorf("default repository '%s' is not defined in repositories", defaultRepo)
		}
	}

	for name, repo := range conf.Repositories {
		if repo.Engine != "restic" && repo.Engine != "rclone" {
			return fmt.Errorf("repository %q has unsupported engine %q", name, repo.Engine)
		}

		if repo.Remote == "" {
			return fmt.Errorf("repository %q has an undefined remote", name)
		}
	}

	return nil
}
