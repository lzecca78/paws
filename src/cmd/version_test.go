package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"version"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()

	assert.NoError(t, err, "Command should not error")
	// Version is set at build time via -ldflags, defaults to "dev" in tests
	assert.Contains(t, buf.String(), "paws version:")
}

func TestVersionVariable(t *testing.T) {
	// When not built with -ldflags, version defaults to "dev"
	// When built with make build or goreleaser, it will be the git tag
	//
	// Example values:
	// - "dev" (default, no ldflags)
	// - "v1.0.0" (exact tag)
	// - "0.3.2-5-gabcdef" (5 commits after v0.3.2)
	// - "0.3.2-5-gabcdef-dirty" (uncommitted changes)

	assert.NotEmpty(t, version, "version should not be empty")
}

func TestVersionCommandOutput(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectContains string
	}{
		{
			name:           "version command",
			args:           []string{"version"},
			expectContains: "paws version:",
		},
		{
			name:           "version alias v",
			args:           []string{"v"},
			expectContains: "paws version:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)

			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)

			err := rootCmd.Execute()

			assert.NoError(t, err)
			assert.Contains(t, buf.String(), tt.expectContains)
		})
	}
}

func TestVersionFormat(t *testing.T) {
	// Test that version output follows expected format
	rootCmd.SetArgs([]string{"version"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := buf.String()

	// Should have "paws version: " prefix
	assert.True(t, strings.HasPrefix(output, "paws version: "), "output should start with 'paws version: '")

	// Should end with newline
	assert.True(t, strings.HasSuffix(output, "\n"), "output should end with newline")

	// Extract version string
	versionStr := strings.TrimPrefix(output, "paws version: ")
	versionStr = strings.TrimSuffix(versionStr, "\n")

	// Version should not be empty
	assert.NotEmpty(t, versionStr, "version string should not be empty")
}
