package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flobilosaurus/agent-env/internal/config"
)

type fakePrompter struct {
	profile string
	create  bool
	calls   int
}

func (f *fakePrompter) ChooseProfile(agent string, profiles []config.Profile) (string, bool, error) {
	f.calls++
	if f.profile == "" {
		return "new-profile", true, nil
	}
	return f.profile, f.create, nil
}

func TestMissingArgs(t *testing.T) {
	var err bytes.Buffer
	code := App{Err: &err}.Run(nil)
	if code == 0 || err.String() == "" {
		t.Fatalf("expected usage error")
	}
}

func TestRunMissingMappingWithFakePrompterCreatesProfileAndMapping(t *testing.T) {
	cfgHome, dataHome, realBin, project, record := setupRunTest(t)
	prompter := &fakePrompter{}
	var out, errBuf bytes.Buffer
	code := App{Out: &out, Err: &errBuf, Prompter: prompter}.Run([]string{"run", "pi"})
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, errBuf.String())
	}
	cfg := loadTestConfig(t, cfgHome)
	key, _ := config.NormalizeProjectPath(project)
	if cfg.Projects[key] != "new-profile" || !cfg.HasProfile("new-profile") {
		t.Fatalf("mapping/profile not saved: %+v", cfg)
	}
	got, _ := os.ReadFile(record)
	if !strings.Contains(string(got), filepath.Join(dataHome, "profiles", "new-profile", "home")) {
		t.Fatalf("wrong home: %s", got)
	}
	_ = realBin
}

func TestRunForceSelectReplacesMappingAndLaunchesWithSelectedProfile(t *testing.T) {
	cfgHome, dataHome, _, project, record := setupRunTest(t)
	writeTestConfig(t, cfgHome, project, "old-profile", "new-profile")
	prompter := &fakePrompter{profile: "new-profile"}
	var out, errBuf bytes.Buffer
	code := App{Out: &out, Err: &errBuf, Prompter: prompter}.Run([]string{"run", "--select", "pi"})
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, errBuf.String())
	}
	if prompter.calls != 1 {
		t.Fatalf("prompter calls=%d", prompter.calls)
	}
	cfg := loadTestConfig(t, cfgHome)
	key, _ := config.NormalizeProjectPath(project)
	if cfg.Projects[key] != "new-profile" {
		t.Fatalf("mapping not replaced: %+v", cfg.Projects)
	}
	got, _ := os.ReadFile(record)
	wantHome := filepath.Join(dataHome, "profiles", "new-profile", "home")
	if !strings.Contains(string(got), "HOME="+wantHome) {
		t.Fatalf("wrong launch home: %s", got)
	}
}

func TestRunSelectAfterAgentIsPassthrough(t *testing.T) {
	cfgHome, _, _, project, record := setupRunTest(t)
	writeTestConfig(t, cfgHome, project, "old-profile")
	prompter := &fakePrompter{profile: "new-profile"}
	var errBuf bytes.Buffer
	code := App{Err: &errBuf, Prompter: prompter}.Run([]string{"run", "pi", "--select"})
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, errBuf.String())
	}
	if prompter.calls != 0 {
		t.Fatalf("prompter calls=%d", prompter.calls)
	}
	got, _ := os.ReadFile(record)
	if !strings.Contains(string(got), "ARGS=--select") {
		t.Fatalf("arg not passed through: %s", got)
	}
}

func TestRunSelectWithoutAgentShowsUsage(t *testing.T) {
	var errBuf bytes.Buffer
	code := App{Err: &errBuf}.Run([]string{"run", "--select"})
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(errBuf.String(), "Usage: agentenv run [--select] <agent> [args...]") {
		t.Fatalf("missing usage: %s", errBuf.String())
	}
}

func TestRunForceSelectNonInteractiveDoesNotChangeMapping(t *testing.T) {
	cfgHome, dataHome, _, project, record := setupRunTest(t)
	writeTestConfig(t, cfgHome, project, "old-profile", "new-profile")
	t.Setenv("AGENTENV_NONINTERACTIVE", "1")
	prompter := &fakePrompter{profile: "new-profile"}
	var errBuf bytes.Buffer
	code := App{Err: &errBuf, Prompter: prompter}.Run([]string{"run", "--select", "pi"})
	if code == 0 {
		t.Fatal("expected failure")
	}
	if !strings.Contains(errBuf.String(), "cannot select a profile in non-interactive mode") {
		t.Fatalf("unexpected error: %s", errBuf.String())
	}
	if prompter.calls != 0 {
		t.Fatalf("prompter calls=%d", prompter.calls)
	}
	cfg := loadTestConfig(t, cfgHome)
	key, _ := config.NormalizeProjectPath(project)
	if cfg.Projects[key] != "old-profile" {
		t.Fatalf("mapping changed: %+v", cfg.Projects)
	}
	if _, err := os.Stat(filepath.Join(dataHome, "profiles", "old-profile", "home")); !os.IsNotExist(err) {
		t.Fatalf("profile home was created or stat failed: %v", err)
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("agent launched or stat failed: %v", err)
	}
}

func TestRunUnknownLeadingOptionRemainsAgentName(t *testing.T) {
	cfgHome, _, realBin, project, record := setupRunTest(t)
	writeTestConfig(t, cfgHome, project, "old-profile")
	if err := os.WriteFile(filepath.Join(realBin, "--foo"), []byte("#!/bin/sh\nprintf 'AGENT=--foo ARGS=%s' \"$*\" > \""+record+"\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var errBuf bytes.Buffer
	code := App{Err: &errBuf}.Run([]string{"run", "--foo"})
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, errBuf.String())
	}
	got, _ := os.ReadFile(record)
	if !strings.Contains(string(got), "AGENT=--foo") {
		t.Fatalf("did not execute --foo agent: %s", got)
	}
}

func setupRunTest(t *testing.T) (cfgHome, dataHome, realBin, project, record string) {
	t.Helper()
	cfgHome = t.TempDir()
	dataHome = t.TempDir()
	realBin = t.TempDir()
	project = t.TempDir()
	record = filepath.Join(t.TempDir(), "record")
	t.Setenv("AGENTENV_CONFIG_HOME", cfgHome)
	t.Setenv("AGENTENV_HOME", dataHome)
	t.Setenv("PATH", realBin)
	if err := os.WriteFile(filepath.Join(realBin, "pi"), []byte("#!/bin/sh\nprintf 'HOME=%s\nARGS=%s\n' \"$HOME\" \"$*\" > \""+record+"\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	return cfgHome, dataHome, realBin, project, record
}

func writeTestConfig(t *testing.T, cfgHome, project string, profiles ...string) {
	t.Helper()
	c := config.Empty()
	for _, profile := range profiles {
		if err := c.AddProfile(profile); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.SetProject(project, profiles[0]); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(filepath.Join(cfgHome, "agentenv", "config.toml"), c); err != nil {
		t.Fatal(err)
	}
}

func loadTestConfig(t *testing.T, cfgHome string) config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(cfgHome, "agentenv", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
