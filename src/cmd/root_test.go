package cmd

import (
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
