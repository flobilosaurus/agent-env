package e2e

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallScriptInstallsLatestReleaseAsset(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is POSIX shell")
	}

	root := repoRoot(t)
	fixtureDir := t.TempDir()
	installDir := t.TempDir()
	fakeBinDir := t.TempDir()

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goarch != "amd64" && goarch != "arm64" {
		t.Skipf("unsupported test architecture: %s", goarch)
	}

	asset := fmt.Sprintf("agentenv-%s-%s.tar.gz", goos, goarch)
	payloadDir := filepath.Join(t.TempDir(), "payload")
	if err := os.MkdirAll(payloadDir, 0755); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(payloadDir, "agentenv")
	if err := os.WriteFile(payload, []byte("#!/bin/sh\necho installed-agentenv\n"), 0755); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(fixtureDir, asset)
	tarCmd := exec.Command("tar", "-czf", archive, "agentenv")
	tarCmd.Dir = payloadDir
	if out, err := tarCmd.CombinedOutput(); err != nil {
		t.Fatalf("create fixture archive: %v\n%s", err, out)
	}

	archiveBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveBytes)
	checksums := fmt.Sprintf("%x  %s\n", sum, asset)
	if err := os.WriteFile(filepath.Join(fixtureDir, "checksums.txt"), []byte(checksums), 0644); err != nil {
		t.Fatal(err)
	}

	fakeCurl := filepath.Join(fakeBinDir, "curl")
	if err := os.WriteFile(fakeCurl, []byte(`#!/bin/sh
set -eu
out=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) shift; out="$1" ;;
    http*) url="$1" ;;
  esac
  shift
done
[ -n "$out" ] || exit 2
[ -n "$url" ] || exit 3
case "$url" in
  */releases/latest/download/*) ;;
  *) echo "unexpected url: $url" >&2; exit 4 ;;
esac
cp "$FIXTURE_DIR/$(basename "$url")" "$out"
`), 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sh", filepath.Join(root, "install.sh"))
	cmd.Env = append(os.Environ(),
		"PATH="+fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FIXTURE_DIR="+fixtureDir,
		"AGENTENV_INSTALL_DIR="+installDir,
		"AGENTENV_REPO=example/agentenv",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Installed agentenv") {
		t.Fatalf("missing success output:\n%s", out)
	}

	installed := filepath.Join(installDir, "agentenv")
	run := exec.Command(installed)
	runOut, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("installed binary failed: %v\n%s", err, runOut)
	}
	if got := strings.TrimSpace(string(runOut)); got != "installed-agentenv" {
		t.Fatalf("installed binary output = %q", got)
	}
}
