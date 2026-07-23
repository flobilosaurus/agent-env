package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupAgentSkipsWrapperBin(t *testing.T) {
	wrap := t.TempDir()
	real := t.TempDir()
	makeExe(t, filepath.Join(wrap, "pi"), "#!/bin/sh\nexit 99\n")
	realPath := filepath.Join(real, "pi")
	makeExe(t, realPath, "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", wrap+string(os.PathListSeparator)+real)
	got, err := LookupAgent("pi", wrap)
	if err != nil {
		t.Fatal(err)
	}
	if got != realPath {
		t.Fatalf("got %s want %s", got, realPath)
	}
}

func makeExe(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}
