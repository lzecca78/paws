package cmd

import (
	"gopkg.in/ini.v1"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldRunDirectProfileSwitch(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"no args", []string{"paws"}, false},
		{"valid profile", []string{"paws", "dev"}, true},
		{"invalid command", []string{"paws", "list"}, false},
		{"help flag", []string{"paws", "--help"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = tc.args
			assert.Equal(t, tc.expected, shouldRunDirectProfileSwitch())
		})
	}
}

func TestRunProfileSwitcherWithPrompt(t *testing.T) {
	mockLoadProfiles := func(path string) (*ini.File, error) {
		cfg := ini.Empty()
		_, _ = cfg.NewSection("profile dev")
		_, _ = cfg.NewSection("profile staging")
		return cfg, nil
	}

	mockLoadSSO := func(path string) (*ini.File, error) {
		cfg := ini.Empty()
		section, _ := cfg.NewSection("profile dev")
		_, _ = section.NewKey("sso_start_url", "https://example.com/sso")
		return cfg, nil
	}

	mockPrompt := func(options []string) (string, error) {
		return "dev", nil
	}

	result := runProfileSwitcherWithPrompt(mockPrompt, mockLoadProfiles, mockLoadSSO)

	assert.Equal(t, "dev", result.Profile)
	assert.Equal(t, "https://example.com/sso", result.SsoStartURL)
}
