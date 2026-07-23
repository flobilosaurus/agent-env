package tui

import (
	"strings"
	"testing"

	"github.com/flobilosaurus/agent-env/internal/config"
)

func TestBanner(t *testing.T) {
	want := "┌─ agentenv ───────────────────────────────────┐\n" +
		"│ work • pi                                    │\n" +
		"└──────────────────────────────────────────────┘"
	if got := Banner("work", "pi"); got != want {
		t.Fatalf("banner mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestProfileSelectionView(t *testing.T) {
	m := newModel("pi", []config.Profile{{Name: "customer-a"}, {Name: "customer-b"}, {Name: "personal"}})
	want := "╭─ agentenv ───────────────────────────────────────────────╮\n" +
		"│ customer-a • pi                                          │\n" +
		"├──────────────────────────────────────────────────────────┤\n" +
		"│                                                          │\n" +
		"│  Select a Profile                                        │\n" +
		"│  Choose an isolated HOME for this project                │\n" +
		"│                                                          │\n" +
		"│  ▸ customer-a                                            │\n" +
		"│    customer-b                                            │\n" +
		"│    personal                                              │\n" +
		"│                                                          │\n" +
		"│    ＋ Create new profile                                 │\n" +
		"│                                                          │\n" +
		"│  ↑/↓/j/k move • enter select • esc/ctrl+c cancel         │\n" +
		"╰──────────────────────────────────────────────────────────╯"
	if got := m.View(); got != want {
		t.Fatalf("selection view mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestProfileCreateView(t *testing.T) {
	m := newModel("pi", []config.Profile{{Name: "customer-a"}})
	m.mode = modeCreate
	want := "╭─ agentenv ───────────────────────────────────────────────╮\n" +
		"│  • pi                                                    │\n" +
		"├──────────────────────────────────────────────────────────┤\n" +
		"│                                                          │\n" +
		"│  Create a Profile                                        │\n" +
		"│  Allowed: lowercase, numbers, dot, dash, underscore      │\n" +
		"│                                                          │\n" +
		"│  > profile-name                                          │\n" +
		"│                                                          │\n" +
		"│  enter create • esc/ctrl+c cancel                        │\n" +
		"╰──────────────────────────────────────────────────────────╯"
	if got := m.View(); got != want {
		t.Fatalf("create view mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestProfileCreateViewShowsTypedProfile(t *testing.T) {
	m := newModel("pi", []config.Profile{{Name: "customer-a"}})
	m.mode = modeCreate
	m.input.SetValue("new-profile")
	if got := m.View(); !strings.Contains(got, "│ new-profile • pi") {
		t.Fatalf("create view should show typed profile in header\ngot:\n%s", got)
	}
}

func TestProfileSelectionCreateRowShowsBlankProfile(t *testing.T) {
	m := newModel("pi", []config.Profile{{Name: "customer-a"}})
	m.cursor = len(m.profiles)
	if got := m.View(); !strings.Contains(got, "│  • pi") {
		t.Fatalf("selection create row should show blank profile in header\ngot:\n%s", got)
	}
}

func TestProfileRemoveView(t *testing.T) {
	m := newRemoveModel([]config.Profile{{Name: "customer-a"}, {Name: "customer-b"}})
	want := "╭─ agentenv ───────────────────────────────────────────────╮\n" +
		"│                                                          │\n" +
		"│  Remove a Profile                                        │\n" +
		"│  Select a profile to delete with its folder              │\n" +
		"│                                                          │\n" +
		"│  ▸ customer-a                                            │\n" +
		"│    customer-b                                            │\n" +
		"│                                                          │\n" +
		"│  ↑/↓/j/k move • enter remove • esc/ctrl+c cancel         │\n" +
		"╰──────────────────────────────────────────────────────────╯"
	if got := m.View(); got != want {
		t.Fatalf("remove view mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}
