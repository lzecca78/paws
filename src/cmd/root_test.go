package cmd

import (
	"bytes"
	"fmt"
	"github.com/lzecca78/paws/src/utils"
	"github.com/spf13/afero"
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

type myShellCommand struct {
	CombinedOutputFunc func() ([]byte, error)
	RunFunc            func() error
}

func (sc myShellCommand) CombinedOutput() ([]byte, error) {
	return sc.CombinedOutputFunc()
}

func (sc myShellCommand) Run() error {
	return sc.RunFunc()
}

type execCommandFunc func(name string, arg ...string) utils.IShellCommand

func newMockShellCommanderForOutput(output string, err error) execCommandFunc {
	return func(name string, arg ...string) utils.IShellCommand {
		fmt.Printf("exec.Command() called with %v and %v\n", name, arg)
		CombinedOutputFunc := func() ([]byte, error) {
			if err == nil {
				fmt.Println("Output obtained")
			} else {
				fmt.Println("Failed to get Output")
			}
			return []byte(output), err
		}
		runFunc := func() error {
			fmt.Printf("Run called for %v with args %v\n", name, arg)
			if err != nil {
				return err
			}
			fmt.Println("Run completed successfully")
			return nil
		}
		return myShellCommand{
			CombinedOutputFunc: CombinedOutputFunc,
			RunFunc:            runFunc,
		}
	}
}

func TestDirectProfileSwitch(t *testing.T) {

	mockFs := afero.NewMemMapFs()
	mockProfile := "dev"

	// Create a mock INI file
	mockIni := ini.Empty()
	section, _ := mockIni.NewSection("profile dev")
	_, _ = section.NewKey("sso_start_url", "https://example.com/sso")

	var buf bytes.Buffer
	_, err := mockIni.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	err = afero.WriteFile(mockFs, "mock_config.ini", buf.Bytes(), 0644)

	assert.NoError(t, err)

	curShellCommander := utils.ExecuteAwsSSOCommander

	// happy path:
	curShellCommander = newMockShellCommanderForOutput("", nil)

	err = directProfileSwitch(mockFs, mockProfile, func(path string) (*ini.File, error) {
		data, err := afero.ReadFile(mockFs, "mock_config.ini")
		if err != nil {
			return nil, err
		}
		return ini.Load(data)
	})

	assert.NoError(t, err)
	// Additional checks can be added to verify the profile switch logic

}
