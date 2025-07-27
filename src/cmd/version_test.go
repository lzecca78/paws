package cmd

import (
	"bytes"
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
	assert.Equal(t, "paws version: v0.1.3\n", buf.String())
}
