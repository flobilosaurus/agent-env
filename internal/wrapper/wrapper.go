package wrapper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const marker = "# agentenv generated wrapper v1"

func ValidateAgentName(agent string) error {
	if agent == "" || strings.ContainsAny(agent, "/\\;&|$><'\"(){}[]!*?~` \t\n") {
		return fmt.Errorf("unsafe agent name %q", agent)
	}
	return nil
}

func Install(binDir, agentenvPath, agent string) (string, error) {
	if err := ValidateAgentName(agent); err != nil {
		return "", err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(binDir, agent)
	content := fmt.Sprintf("#!/bin/sh\n%s\nexec %q run %q \"$@\"\n", marker, agentenvPath, agent)
	if existing, err := os.ReadFile(target); err == nil {
		if !strings.Contains(string(existing), marker) {
			return "", fmt.Errorf("refusing to overwrite non-agentenv file: %s", target)
		}
	}
	if err := os.WriteFile(target, []byte(content), 0o755); err != nil {
		return "", err
	}
	return target, nil
}
