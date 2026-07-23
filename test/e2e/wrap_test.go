package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrapCreatesExecutableAndRunsRealAgent(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	project := t.TempDir()
	realBin := t.TempDir()
	pathFile := filepath.Join(t.TempDir(), ".profile")
	rec := filepath.Join(t.TempDir(), "rec")
	writeConfig(t, cfg, project, "p")
	fakeAgent(t, realBin, "pi", "#!/bin/sh\necho real:$* > \""+rec+"\"\n")
	cmd := exec.Command(bin, "wrap", "pi")
	cmd.Env = append(baseEnv(cfg, data, realBin), "AGENTENV_PATH_FILE="+pathFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrap: %v\n%s", err, out)
	}
	wrapperPath := filepath.Join(data, "bin", "pi")
	st, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&0111 == 0 {
		t.Fatal("not executable")
	}
	pathSetup, err := os.ReadFile(pathFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pathSetup), filepath.Join(data, "bin")) {
		t.Fatalf("PATH setup does not mention wrapper dir: %s", pathSetup)
	}
	run := exec.Command(wrapperPath, "--version")
	run.Dir = project
	run.Env = baseEnv(cfg, data, filepath.Join(data, "bin")+string(os.PathListSeparator)+realBin)
	out, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("wrapped run: %v\n%s", err, out)
	}
	got, _ := os.ReadFile(rec)
	if !strings.Contains(string(got), "real:--version") {
		t.Fatalf("bad rec %s", got)
	}
}

func TestWrapAcceptsAgentNamesContainingNOrT(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	cmd := exec.Command(bin, "wrap", "opencode")
	cmd.Env = baseEnv(cfg, data, "")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrap: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(data, "bin", "opencode")); err != nil {
		t.Fatal(err)
	}
}

func TestWrapWritesNushellPathSetup(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	pathFile := filepath.Join(t.TempDir(), "env.nu")
	cmd := exec.Command(bin, "wrap", "pi")
	cmd.Env = append(baseEnv(cfg, data, ""), "AGENTENV_PATH_FILE="+pathFile, "SHELL=/usr/bin/nu")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrap: %v\n%s", err, out)
	}
	pathSetup, err := os.ReadFile(pathFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(pathSetup)
	if !strings.Contains(got, "$env.PATH") || !strings.Contains(got, "prepend $agentenv_bin") {
		t.Fatalf("not nushell PATH setup: %s", got)
	}
	if strings.Contains(got, "export PATH") {
		t.Fatalf("unexpected posix PATH setup in nushell file: %s", got)
	}
}

func TestWrapWritesFishPathSetup(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	pathFile := filepath.Join(t.TempDir(), "agentenv.fish")
	cmd := exec.Command(bin, "wrap", "pi")
	cmd.Env = append(baseEnv(cfg, data, ""), "AGENTENV_PATH_FILE="+pathFile, "SHELL=/usr/bin/fish")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrap: %v\n%s", err, out)
	}
	pathSetup, err := os.ReadFile(pathFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(pathSetup)
	if !strings.Contains(got, "set -gx PATH") || !strings.Contains(got, "string match -v") {
		t.Fatalf("not fish PATH setup: %s", got)
	}
	if strings.Contains(got, "export PATH") || strings.Contains(got, "$env.PATH") {
		t.Fatalf("unexpected non-fish PATH setup in fish file: %s", got)
	}
}

func TestWrapSucceedsWithHintsWhenPathUpdateFails(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	pathDir := t.TempDir()
	cmd := exec.Command(bin, "wrap", "pi")
	cmd.Env = append(baseEnv(cfg, data, ""), "AGENTENV_PATH_FILE="+pathDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrap should not fail when PATH update fails: %v\n%s", err, out)
	}
	got := string(out)
	if !strings.Contains(got, "installed wrapper:") || !strings.Contains(got, "could not update PATH") || !strings.Contains(got, "hint: add") {
		t.Fatalf("missing wrapper success/path hints: %s", got)
	}
	if _, err := os.Stat(filepath.Join(data, "bin", "pi")); err != nil {
		t.Fatal(err)
	}
}

func TestWrapRefusesNonAgentenvFile(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	os.MkdirAll(filepath.Join(data, "bin"), 0755)
	os.WriteFile(filepath.Join(data, "bin", "pi"), []byte("real"), 0755)
	cmd := exec.Command(bin, "wrap", "pi")
	cmd.Env = baseEnv(cfg, data, "")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected fail")
	}
	if !strings.Contains(string(out), "refusing to overwrite") {
		t.Fatalf("unexpected %s", out)
	}
}
