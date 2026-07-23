package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agentenv", "config.toml")
	c := Empty()
	if err := c.AddProfile("customer-a"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetProject(t.TempDir(), "customer-a"); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasProfile("customer-a") || len(got.Projects) != 1 {
		t.Fatalf("unexpected config: %+v", got)
	}
}

func TestValidateProfileName(t *testing.T) {
	valid := []string{"a", "customer-a", "foo.bar_1"}
	for _, n := range valid {
		if err := ValidateProfileName(n); err != nil {
			t.Fatalf("%s should be valid: %v", n, err)
		}
	}
	invalid := []string{"", " bad", "../x", "a/b", "a b"}
	for _, n := range invalid {
		if err := ValidateProfileName(n); err == nil {
			t.Fatalf("%s should be invalid", n)
		}
	}
}

func TestNormalizeProjectPath(t *testing.T) {
	dir := t.TempDir()
	want, err := NormalizeProjectPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := NormalizeProjectPath(filepath.Join(dir, "."))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
