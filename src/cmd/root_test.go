package cmd

import (
	"fmt"
	"github.com/lzecca78/paws/src/config"
	"github.com/lzecca78/paws/src/utils"
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

type MockProfileHelper struct {
	Profiles    []string
	Chosen      string
	SSOStartURL string
	FileContent string
}

func (m *MockProfileHelper) GetProfiles() []string {
	return m.Profiles
}

func (m *MockProfileHelper) GetPromptProfiles(profiles []string) (string, error) {
	return m.Chosen, nil
}

func (m *MockProfileHelper) SetProfile(profile string) {
	m.Chosen = profile
}

func (m *MockProfileHelper) GetSSOStartURL() error {
	return nil
}

func (m *MockProfileHelper) WriteFile(loc string) error {
	m.FileContent = m.Chosen
	return nil
}

func (m *MockProfileHelper) SSOLogin() error {
	return nil
}

func (m *MockProfileHelper) PulumiSetup() error {
	return nil
}

func (m *MockProfileHelper) NewShellCommand(name string, args ...string) utils.IShellCommand {
	return nil
}

func (m *MockProfileHelper) GetCallerIdentity() (config.AwsGetCallerIdentitySpec, error) {
	return config.AwsGetCallerIdentitySpec{}, nil
}

func (m *MockProfileHelper) GetSSOUrl() string {
	return m.SSOStartURL
}

func TestRunProfileSwitcherWithPrompt(t *testing.T) {
	mock := &MockProfileHelper{
		Profiles:    []string{"dev", "staging"},
		Chosen:      "dev",
		SSOStartURL: "https://example.com/sso",
	}

	result, err := runProfileSwitcherWithPrompt(mock)
	assert.NoError(t, err)
	assert.Equal(t, "dev", result.Profile)
	assert.Equal(t, "https://example.com/sso", result.SsoStartURL)
	assert.Equal(t, "dev", mock.FileContent)
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

//func TestDirectProfileSwitch(t *testing.T) {
//
//	mockFs := afero.NewMemMapFs()
//	mockProfile := "dev"
//
//	// Create a mock INI file
//	mockIni := ini.Empty()
//	section, _ := mockIni.NewSection("profile dev")
//	_, _ = section.NewKey("sso_start_url", "https://example.com/sso")
//
//	var buf bytes.Buffer
//	_, err := mockIni.WriteTo(&buf)
//	if err != nil {
//		t.Fatal(err)
//	}
//	err = afero.WriteFile(mockFs, "mock_config.ini", buf.Bytes(), 0644)
//
//	assert.NoError(t, err)
//
//	curShellCommander := utils.ExecuteAwsSSOCommander
//
//	// happy path:
//	curShellCommander = newMockShellCommanderForOutput("", nil)
//
//	err = directProfileSwitch(mockFs, mockProfile, func(path string) (*ini.File, error) {
//		data, err := afero.ReadFile(mockFs, "mock_config.ini")
//		if err != nil {
//			return nil, err
//		}
//		return ini.Load(data)
//	})
//
//	assert.NoError(t, err)
//	// Additional checks can be added to verify the profile switch logic
//
//}
