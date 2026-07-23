package pathsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureWrapperBinFirstDoesNotWriteWhenCurrentPathAlreadyContainsWrapper(t *testing.T) {
	wrapper := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(wrapper, 0o755); err != nil {
		t.Fatal(err)
	}
	pathFile := filepath.Join(t.TempDir(), ".profile")
	t.Setenv("AGENTENV_PATH_FILE", pathFile)
	t.Setenv("PATH", "/real/bin"+string(os.PathListSeparator)+wrapper)

	got, err := EnsureWrapperBinFirst(wrapper + string(os.PathSeparator))
	if err != nil {
		t.Fatal(err)
	}
	if got.Changed {
		t.Fatal("expected no change")
	}
	if _, err := os.Stat(pathFile); !os.IsNotExist(err) {
		t.Fatalf("expected path file not to be written, stat err=%v", err)
	}
}

func TestUpsertBlockMovesManagedBlockToEnd(t *testing.T) {
	oldBlock := posixBlock("/old/agentenv/bin")
	newBlock := posixBlock("/new/agentenv/bin")
	content := oldBlock + "export PATH=/real/bin:$PATH\n"

	got, changed := upsertBlock(content, newBlock)
	if !changed {
		t.Fatal("expected change")
	}
	if strings.Contains(got, "/old/agentenv/bin") {
		t.Fatalf("old block not removed:\n%s", got)
	}
	if !strings.HasSuffix(got, newBlock) {
		t.Fatalf("managed block should be last so it wins PATH ordering:\n%s", got)
	}
	if strings.Index(got, "export PATH=/real/bin:$PATH") > strings.Index(got, beginMarker) {
		t.Fatalf("PATH changes should remain before managed block:\n%s", got)
	}
}

func TestPosixBlockPutsWrapperBinFirstWhenSourced(t *testing.T) {
	wrapper := filepath.Clean("/tmp/agentenv/bin")
	block := posixBlock(wrapper)
	if !strings.Contains(block, "export PATH=\"$agentenv_bin${PATH:+:$PATH}\"") {
		t.Fatalf("block does not prefix wrapper bin:\n%s", block)
	}
}
