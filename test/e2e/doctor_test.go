package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorHealthyWithProbe(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	project := t.TempDir()
	realBin := t.TempDir()
	os.MkdirAll(filepath.Join(data, "bin"), 0755)
	writeConfig(t, cfg, project, "p")
	os.MkdirAll(filepath.Join(data, "profiles", "p", "home"), 0700)
	fakeAgent(t, filepath.Join(data, "bin"), "pi", "#!/bin/sh\nexec \""+bin+"\" run pi \"$@\"\n")
	fakeAgent(t, realBin, "pi", "#!/bin/sh\necho pi 1.2.3\n")
	cmd := exec.Command(bin, "doctor", "pi")
	cmd.Dir = project
	cmd.Env = baseEnv(cfg, data, filepath.Join(data, "bin")+string(os.PathListSeparator)+realBin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, out)
	}
	s := string(out)
	for _, sub := range []string{"✓ config readable", "✓ project mapping", "✓ profile home", "✓ real agent", "✓ probe: pi 1.2.3"} {
		if !strings.Contains(s, sub) {
			t.Fatalf("missing %q in\n%s", sub, s)
		}
	}
}

func TestDoctorMissingMappingFails(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	project := t.TempDir()
	cmd := exec.Command(bin, "doctor")
	cmd.Dir = project
	cmd.Env = baseEnv(cfg, data, "")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected fail")
	}
	if !strings.Contains(string(out), "✗ project mapping") {
		t.Fatalf("unexpected %s", out)
	}
}

func TestDoctorBadPathOrderingWarns(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	project := t.TempDir()
	realBin := t.TempDir()
	os.MkdirAll(filepath.Join(data, "bin"), 0755)
	writeConfig(t, cfg, project, "p")
	os.MkdirAll(filepath.Join(data, "profiles", "p", "home"), 0700)
	fakeAgent(t, filepath.Join(data, "bin"), "pi", "#!/bin/sh\nexit 0\n")
	fakeAgent(t, realBin, "pi", "#!/bin/sh\necho v\n")
	cmd := exec.Command(bin, "doctor", "pi")
	cmd.Dir = project
	cmd.Env = baseEnv(cfg, data, realBin+string(os.PathListSeparator)+filepath.Join(data, "bin"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("warnings should not fail: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "! wrapper PATH") {
		t.Fatalf("unexpected %s", out)
	}
}

func TestDoctorMissingRealAgentFails(t *testing.T) {
	bin := buildAgentenv(t)
	cfg := t.TempDir()
	data := t.TempDir()
	project := t.TempDir()
	writeConfig(t, cfg, project, "p")
	os.MkdirAll(filepath.Join(data, "profiles", "p", "home"), 0700)
	cmd := exec.Command(bin, "doctor", "pi")
	cmd.Dir = project
	cmd.Env = baseEnv(cfg, data, filepath.Join(data, "bin"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected fail")
	}
	if !strings.Contains(string(out), "✗ real agent") {
		t.Fatalf("unexpected %s", out)
	}
}
