package doctor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/flobilosaurus/agent-env/internal/config"
	"github.com/flobilosaurus/agent-env/internal/paths"
	"github.com/flobilosaurus/agent-env/internal/runner"
)

type Severity int

const (
	OK Severity = iota
	Warn
	Fail
)

type Check struct {
	Severity           Severity
	Label, Detail, Fix string
}

type Report struct{ Checks []Check }

func (r Report) ExitCode() int {
	for _, c := range r.Checks {
		if c.Severity == Fail {
			return 1
		}
	}
	return 0
}

func Run(projectDir, agent string, p paths.Paths) Report {
	var out Report
	cfg, err := config.Load(p.ConfigFile())
	if err != nil {
		out.add(Fail, "config readable", p.ConfigFile(), err.Error())
	} else {
		out.add(OK, "config readable", p.ConfigFile(), "")
	}
	proj, _ := config.NormalizeProjectPath(projectDir)
	profile := ""
	if err == nil {
		profile = cfg.Projects[proj]
		if profile == "" {
			out.add(Fail, "project mapping", proj, "run `agentenv run <agent>` interactively to select a profile")
		} else if !cfg.HasProfile(profile) {
			out.add(Fail, "project mapping", proj+" -> "+profile, "add missing profile or update mapping")
		} else {
			out.add(OK, "project mapping", proj+" -> "+profile, "")
		}
	}
	if profile != "" {
		h := p.ProfileHome(profile)
		if st, err := os.Stat(h); err != nil || !st.IsDir() {
			out.add(Fail, "profile home", h, "create the profile home by running `agentenv run <agent>`")
		} else if f, err := os.CreateTemp(h, ".agentenv-write-*"); err != nil {
			out.add(Fail, "profile home writable", h, err.Error())
		} else {
			name := f.Name()
			f.Close()
			os.Remove(name)
			out.add(OK, "profile home", h, "")
		}
	}
	if agent != "" {
		out.agentChecks(agent, p)
	}
	return out
}

func (r *Report) agentChecks(agent string, p paths.Paths) {
	wrapperPath := filepath.Join(p.BinDir(), agent)
	if st, err := os.Stat(wrapperPath); err != nil || st.IsDir() {
		r.add(Warn, "wrapper", wrapperPath, "run `agentenv wrap "+agent+"`")
	} else {
		r.add(OK, "wrapper", wrapperPath, "")
	}
	real, err := runner.LookupAgent(agent, p.BinDir())
	if err != nil {
		r.add(Fail, "real agent", agent, err.Error())
		return
	}
	r.add(OK, "real agent", real, "")
	r.checkPathOrder(real, p.BinDir())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, real, "--version")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	detail := strings.TrimSpace(buf.String())
	if detail == "" {
		detail = real + " --version"
	}
	if ctx.Err() == context.DeadlineExceeded {
		r.add(Warn, "probe", detail, "version probe timed out")
	} else if err != nil {
		r.add(Warn, "probe", detail, err.Error())
	} else {
		r.add(OK, "probe", detail, "")
	}
}

func (r *Report) checkPathOrder(real, bin string) {
	realDir := filepath.Dir(real)
	bin = filepath.Clean(bin)
	binIdx, realIdx := -1, -1
	for i, d := range filepath.SplitList(os.Getenv("PATH")) {
		cd := filepath.Clean(d)
		if cd == bin && binIdx < 0 {
			binIdx = i
		}
		if cd == realDir && realIdx < 0 {
			realIdx = i
		}
	}
	if binIdx < 0 {
		r.add(Warn, "wrapper PATH", bin+" is not on PATH", "add "+bin+" before the real agent directory in PATH")
		return
	}
	if realIdx >= 0 && binIdx > realIdx {
		r.add(Warn, "wrapper PATH", bin+" is not before "+realDir, "add "+bin+" before the real agent directory in PATH")
		return
	}
	r.add(OK, "wrapper PATH", bin+" before "+realDir, "")
}

func (r *Report) add(s Severity, label, detail, fix string) {
	r.Checks = append(r.Checks, Check{s, label, detail, fix})
}

func Format(r Report) string {
	var b strings.Builder
	for _, c := range r.Checks {
		mark := "✓"
		if c.Severity == Warn {
			mark = "!"
		} else if c.Severity == Fail {
			mark = "✗"
		}
		fmt.Fprintf(&b, "%s %s: %s\n", mark, c.Label, c.Detail)
		if c.Fix != "" {
			fmt.Fprintf(&b, "  fix: %s\n", c.Fix)
		}
	}
	return b.String()
}
