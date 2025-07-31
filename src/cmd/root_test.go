package cmd

import (
	"fmt"
	"github.com/lzecca78/paws/src/config"
	"github.com/lzecca78/paws/src/utils"
	"os"
	"os/exec"
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

type MockShellCommand struct {
	Name                 string
	Args                 []string
	Output               []byte
	Err                  error
	RunCalled            bool
	CombinedOutputCalled bool
}

func (m *MockShellCommand) Run() error {
	m.RunCalled = true
	return m.Err
}

func (m *MockShellCommand) CombinedOutput() ([]byte, error) {
	m.CombinedOutputCalled = true
	if m.Err != nil {
		return m.Output, m.Err
	}
	return m.Output, nil
}

type MockProfileHelper struct {
	Profiles         []string
	Chosen           string
	SSOStartURL      string
	FileContent      string
	SSOStartURLError error
	SSOLoginError    error
	PulumiError      error
	WriteFileError   error
	MockCmd          *MockShellCommand
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
	return m.SSOStartURLError
}

func (m *MockProfileHelper) WriteFile(loc string) error {
	if m.WriteFileError != nil {
		return m.WriteFileError
	}
	m.FileContent = m.Chosen
	return nil
}

func (m *MockProfileHelper) SSOLogin() error {
	return m.SSOLoginError
}

func (m *MockProfileHelper) PulumiSetup() error {
	return m.PulumiError
}

func (m *MockProfileHelper) NewShellCommand(name string, args ...string) utils.IShellCommand {
	if m.MockCmd != nil {
		m.MockCmd.Name = name
		m.MockCmd.Args = args
		return m.MockCmd
	}
	return &MockShellCommand{Name: name, Args: args}
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

func TestDirectProfileSwitch(t *testing.T) {
	tests := []struct {
		name            string
		desiredProfile  string
		mock            *MockProfileHelper
		mockShellOutput []byte
		mockShellError  error
		wantErr         bool
		expectedErrMsg  string
		wantProfileSet  string
		wantFile        string
	}{
		{
			name:           "successfully switches to dev",
			desiredProfile: "dev",
			mock: &MockProfileHelper{
				Profiles:    []string{"dev", "staging"},
				SSOStartURL: "https://example.com/sso",
			},
			mockShellOutput: []byte("OK"),
			mockShellError:  nil,
			wantErr:         false,
			wantProfileSet:  "dev",
			wantFile:        "dev",
		},
		{
			name:           "profile not found",
			desiredProfile: "prod",
			mock: &MockProfileHelper{
				Profiles: []string{"dev", "staging"},
			},
			mockShellOutput: nil,
			mockShellError:  nil,
			wantErr:         false,
			wantProfileSet:  "",
			wantFile:        "",
		},
		{
			name:           "SSO start URL fails",
			desiredProfile: "dev",
			mock: &MockProfileHelper{
				Profiles:         []string{"dev"},
				SSOStartURLError: fmt.Errorf("sso error"),
			},
			wantErr:        true,
			expectedErrMsg: "sso error",
			wantProfileSet: "dev",
			wantFile:       "",
		},
		{
			name:           "SSO login fails",
			desiredProfile: "dev",
			mock: &MockProfileHelper{
				Profiles:      []string{"dev"},
				SSOLoginError: fmt.Errorf("login error"),
			},
			wantErr:        true,
			expectedErrMsg: "login error",
			wantProfileSet: "dev",
			wantFile:       "",
		},
		{
			name:           "Pulumi setup fails",
			desiredProfile: "dev",
			mock: &MockProfileHelper{
				Profiles:    []string{"dev"},
				PulumiError: fmt.Errorf("pulumi error"),
			},
			wantErr:        true,
			expectedErrMsg: "pulumi error",
			wantProfileSet: "dev",
			wantFile:       "",
		},
		{
			name:           "WriteFile fails",
			desiredProfile: "dev",
			mock: &MockProfileHelper{
				Profiles:       []string{"dev"},
				WriteFileError: fmt.Errorf("write error"),
			},
			wantErr:        true,
			expectedErrMsg: "write error",
			wantProfileSet: "dev",
			wantFile:       "",
		},
		{
			name:           "shell command fails due to missing binary",
			desiredProfile: "dev",
			mock: &MockProfileHelper{
				Profiles:      []string{"dev"},
				SSOLoginError: exec.ErrNotFound, // trigger the real error path
			},
			wantErr:        true,
			expectedErrMsg: "file not found", // depends on OS
			wantProfileSet: "dev",
			wantFile:       "",
		},
		{
			name:           "SSO login fails due to exec error",
			desiredProfile: "dev",
			mock: &MockProfileHelper{
				Profiles:      []string{"dev"},
				SSOLoginError: fmt.Errorf("exec failure: permission denied"),
			},
			wantErr:        true,
			expectedErrMsg: "permission denied",
			wantProfileSet: "dev",
			wantFile:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != nil && tt.mockShellOutput != nil || tt.mockShellError != nil {
				tt.mock.MockCmd = &MockShellCommand{
					Output: tt.mockShellOutput,
					Err:    tt.mockShellError,
				}
			}

			err := directProfileSwitch(tt.desiredProfile, tt.mock)

			if tt.wantErr {
				assert.Error(t, err)
				if err != nil && tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantProfileSet, tt.mock.Chosen)
			assert.Equal(t, tt.wantFile, tt.mock.FileContent)
		})
	}
}
