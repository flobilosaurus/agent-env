package paths

import (
	"os"
	"path/filepath"
)

type Paths struct {
	ConfigHome string
	DataRoot   string
}

func Resolve() (Paths, error) {
	cfg := os.Getenv("AGENTENV_CONFIG_HOME")
	if cfg == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return Paths{}, err
		}
		cfg = base
	}
	data := os.Getenv("AGENTENV_HOME")
	if data == "" {
		base, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
		data = filepath.Join(base, ".local", "share", "agentenv")
	}
	return Paths{ConfigHome: cfg, DataRoot: data}, nil
}

func (p Paths) ConfigFile() string { return filepath.Join(p.ConfigHome, "agentenv", "config.toml") }
func (p Paths) BinDir() string     { return filepath.Join(p.DataRoot, "bin") }
func (p Paths) ProfileHome(profile string) string {
	return filepath.Join(p.DataRoot, "profiles", profile, "home")
}

func EnsureProfileHome(p Paths, profile string) (string, error) {
	h := p.ProfileHome(profile)
	return h, os.MkdirAll(h, 0o700)
}
