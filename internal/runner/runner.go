package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type IO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func LookupAgent(agent, wrapperBin string) (string, error) {
	if agent == "" || strings.ContainsAny(agent, `/\`) {
		return "", fmt.Errorf("invalid agent name %q", agent)
	}
	pathEnv := os.Getenv("PATH")
	wrapperBin = filepath.Clean(wrapperBin)
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		cleanDir := filepath.Clean(dir)
		if samePath(cleanDir, wrapperBin) {
			continue
		}
		candidate := filepath.Join(cleanDir, agent)
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("real agent %q not found on PATH (skipping %s)", agent, wrapperBin)
}

func RunAgent(realPath string, args []string, home string, io IO) int {
	cmd := exec.Command(realPath, args...)
	cmd.Env = withHome(os.Environ(), home)
	cmd.Stdin = io.Stdin
	cmd.Stdout = io.Stdout
	cmd.Stderr = io.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		fmt.Fprintf(io.Stderr, "agentenv: failed to run %s: %v\n", realPath, err)
		return 127
	}
	return 0
}

func withHome(env []string, home string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, "HOME=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, "HOME="+home)
}

func isExecutable(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return st.Mode()&0o111 != 0
}

func samePath(a, b string) bool {
	aa, erra := filepath.Abs(a)
	bb, errb := filepath.Abs(b)
	if erra == nil {
		a = aa
	}
	if errb == nil {
		b = bb
	}
	return a == b
}
