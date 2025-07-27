package cmd

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mock version of runProfileListerToWriter
func mockRunProfileListerToWriter(w io.Writer) error {
	profiles := []string{"profile-a", "profile-b"}
	for _, p := range profiles {
		fmt.Fprintln(w, p)
	}
	return nil
}

func TestListCommand(t *testing.T) {
	buf := new(bytes.Buffer) // ⬅ define it first
	// Monkey-patch the logic for test
	original := runProfileLister
	runProfileLister = func() error {
		return mockRunProfileListerToWriter(buf)
	}
	defer func() { runProfileLister = original }()

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"list"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "profile-a")
	assert.Contains(t, got, "profile-b")
}
