package paths

import (
	"path/filepath"
	"testing"
)

func TestResolveOverrides(t *testing.T) {
	cfg := t.TempDir()
	data := t.TempDir()
	t.Setenv("AGENTENV_CONFIG_HOME", cfg)
	t.Setenv("AGENTENV_HOME", data)
	p, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if p.ConfigFile() != filepath.Join(cfg, "agentenv", "config.toml") {
		t.Fatal(p.ConfigFile())
	}
	if p.BinDir() != filepath.Join(data, "bin") {
		t.Fatal(p.BinDir())
	}
	if p.ProfileHome("x") != filepath.Join(data, "profiles", "x", "home") {
		t.Fatal(p.ProfileHome("x"))
	}
}
