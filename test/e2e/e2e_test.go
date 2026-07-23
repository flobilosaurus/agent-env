package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/flobilosaurus/agent-env/internal/config"
)

func buildAgentenv(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "agentenv")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/agentenv")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	return filepath.Clean(filepath.Join(wd, "../.."))
}

func writeConfig(t *testing.T, cfgHome, project, profile string) {
	t.Helper()
	c := config.Empty()
	if err := c.AddProfile(profile); err != nil {
		t.Fatal(err)
	}
	if err := c.SetProject(project, profile); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(filepath.Join(cfgHome, "agentenv", "config.toml"), c); err != nil {
		t.Fatal(err)
	}
}

func fakeAgent(t *testing.T, dir, name, script string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func baseEnv(cfgHome, dataHome, path string) []string {
	return append(os.Environ(), "AGENTENV_CONFIG_HOME="+cfgHome, "AGENTENV_HOME="+dataHome, "PATH="+path, "AGENTENV_NONINTERACTIVE=1")
}
