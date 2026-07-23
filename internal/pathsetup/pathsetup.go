package pathsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	beginMarker = "# >>> agentenv PATH >>>"
	endMarker   = "# <<< agentenv PATH <<<"
)

type Result struct {
	ProfilePath string
	Changed     bool
}

func EnsureWrapperBinFirst(wrapperBin string) (Result, error) {
	profile, shellKind, err := profilePath()
	if err != nil {
		return Result{}, err
	}
	if pathContainsWrapperBin(os.Getenv("PATH"), wrapperBin) {
		return Result{ProfilePath: profile, Changed: false}, nil
	}
	if err := os.MkdirAll(filepath.Dir(profile), 0o755); err != nil {
		return Result{}, err
	}
	block := managedBlock(wrapperBin, shellKind)
	current, err := os.ReadFile(profile)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	updated, changed := upsertBlock(string(current), block)
	if !changed {
		return Result{ProfilePath: profile, Changed: false}, nil
	}
	if err := os.WriteFile(profile, []byte(updated), 0o644); err != nil {
		return Result{}, err
	}
	return Result{ProfilePath: profile, Changed: true}, nil
}

func profilePath() (string, string, error) {
	shell := filepath.Base(os.Getenv("SHELL"))
	if override := os.Getenv("AGENTENV_PATH_FILE"); override != "" {
		return override, shellKind(shell), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, ".profile"), "posix", nil
	}
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc"), "posix", nil
	case "bash":
		return filepath.Join(home, ".bashrc"), "posix", nil
	case "nu":
		return nushellEnvPath(home), "nushell", nil
	case "fish":
		return fishConfigPath(home), "fish", nil
	default:
		return filepath.Join(home, ".profile"), "posix", nil
	}
}

func shellKind(shell string) string {
	switch shell {
	case "nu":
		return "nushell"
	case "fish":
		return "fish"
	default:
		return "posix"
	}
}

func nushellEnvPath(home string) string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "nushell", "env.nu")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "nushell", "env.nu")
	}
	return filepath.Join(home, ".config", "nushell", "env.nu")
}

func fishConfigPath(home string) string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "fish", "conf.d", "agentenv.fish")
	}
	return filepath.Join(home, ".config", "fish", "conf.d", "agentenv.fish")
}

func pathContainsWrapperBin(pathValue, wrapperBin string) bool {
	for _, entry := range filepath.SplitList(pathValue) {
		if samePath(entry, wrapperBin) {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aa, errA := filepath.Abs(filepath.Clean(a))
	bb, errB := filepath.Abs(filepath.Clean(b))
	if errA == nil && errB == nil {
		aaEval, evalErrA := filepath.EvalSymlinks(aa)
		bbEval, evalErrB := filepath.EvalSymlinks(bb)
		if evalErrA == nil {
			aa = aaEval
		}
		if evalErrB == nil {
			bb = bbEval
		}
	}
	return aa == bb
}

func managedBlock(wrapperBin, shellKind string) string {
	switch shellKind {
	case "nushell":
		return nushellBlock(wrapperBin)
	case "fish":
		return fishBlock(wrapperBin)
	default:
		return posixBlock(wrapperBin)
	}
}

func posixBlock(wrapperBin string) string {
	q := shellQuote(filepath.Clean(wrapperBin))
	return fmt.Sprintf(`%s
# Keep agentenv wrappers before real agent binaries.
agentenv_bin=%s
agentenv_old_path=$PATH
PATH=
while [ -n "$agentenv_old_path" ]; do
  agentenv_entry=${agentenv_old_path%%:*}
  if [ "$agentenv_old_path" = "$agentenv_entry" ]; then
    agentenv_old_path=
  else
    agentenv_old_path=${agentenv_old_path#*:}
  fi
  if [ "$agentenv_entry" != "$agentenv_bin" ]; then
    PATH=${PATH:+$PATH:}$agentenv_entry
  fi
done
export PATH="$agentenv_bin${PATH:+:$PATH}"
unset agentenv_bin agentenv_old_path agentenv_entry
%s
`, beginMarker, q, endMarker)
}

func nushellBlock(wrapperBin string) string {
	q := nuQuote(filepath.Clean(wrapperBin))
	return fmt.Sprintf(`%s
# Keep agentenv wrappers before real agent binaries.
let agentenv_bin = %s
$env.PATH = ($env.PATH | where {|entry| $entry != $agentenv_bin } | prepend $agentenv_bin)
%s
`, beginMarker, q, endMarker)
}

func fishBlock(wrapperBin string) string {
	q := fishQuote(filepath.Clean(wrapperBin))
	return fmt.Sprintf(`%s
# Keep agentenv wrappers before real agent binaries.
set -l agentenv_bin %s
set -gx PATH $agentenv_bin (string match -v -- $agentenv_bin $PATH)
%s
`, beginMarker, q, endMarker)
}

func upsertBlock(content, block string) (string, bool) {
	withoutBlock := content
	start := strings.Index(content, beginMarker)
	end := strings.Index(content, endMarker)
	if start >= 0 && end >= start {
		end += len(endMarker)
		for end < len(content) && (content[end] == '\r' || content[end] == '\n') {
			end++
		}
		withoutBlock = strings.TrimRight(content[:start], "\r\n")
		if rest := strings.TrimLeft(content[end:], "\r\n"); rest != "" {
			if withoutBlock != "" {
				withoutBlock += "\n"
			}
			withoutBlock += rest
		}
	}
	if withoutBlock == "" {
		return block, block != content
	}
	updated := withoutBlock
	if !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	updated += block
	return updated, updated != content
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func nuQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func fishQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
}
