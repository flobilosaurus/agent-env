package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Profiles []Profile         `toml:"profiles"`
	Projects map[string]string `toml:"projects"`
}

type Profile struct {
	Name string `toml:"name"`
}

var profileNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func Empty() Config { return Config{Projects: map[string]string{}} }

func NormalizeProjectPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("project path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = real
	}
	return filepath.Clean(abs), nil
}

func ValidateProfileName(name string) error {
	if strings.TrimSpace(name) != name || name == "" {
		return fmt.Errorf("profile name must be non-empty and contain no surrounding whitespace")
	}
	if !profileNameRE.MatchString(name) || strings.Contains(name, "..") {
		return fmt.Errorf("profile name %q is invalid; use letters, numbers, dot, underscore, or dash", name)
	}
	return nil
}

func Load(path string) (Config, error) {
	c := Empty()
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return c, err
	}
	if c.Projects == nil {
		c.Projects = map[string]string{}
	}
	return c, nil
}

func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.toml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	enc := toml.NewEncoder(tmp)
	if err := enc.Encode(c); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (c *Config) HasProfile(name string) bool {
	for _, p := range c.Profiles {
		if p.Name == name {
			return true
		}
	}
	return false
}

func (c *Config) AddProfile(name string) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	if c.HasProfile(name) {
		return fmt.Errorf("profile %q already exists", name)
	}
	c.Profiles = append(c.Profiles, Profile{Name: name})
	sort.Slice(c.Profiles, func(i, j int) bool { return c.Profiles[i].Name < c.Profiles[j].Name })
	if c.Projects == nil {
		c.Projects = map[string]string{}
	}
	return nil
}

func (c *Config) SetProject(project, profile string) error {
	if !c.HasProfile(profile) {
		return fmt.Errorf("profile %q does not exist", profile)
	}
	key, err := NormalizeProjectPath(project)
	if err != nil {
		return err
	}
	if c.Projects == nil {
		c.Projects = map[string]string{}
	}
	c.Projects[key] = profile
	return nil
}
