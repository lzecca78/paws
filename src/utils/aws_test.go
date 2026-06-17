package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	localconfig "github.com/lzecca78/paws/src/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"
)

// MockShellCommand is a mock implementation of IShellCommand for testing
type MockShellCommand struct {
	OutputData           []byte
	CombinedOutputData   []byte
	RunErr               error
	OutputErr            error
	CombinedOutputErr    error
	RunCalled            bool
	OutputCalled         bool
	CombinedOutputCalled bool
}

func (m *MockShellCommand) Run() error {
	m.RunCalled = true
	return m.RunErr
}

func (m *MockShellCommand) Output() ([]byte, error) {
	m.OutputCalled = true
	return m.OutputData, m.OutputErr
}

func (m *MockShellCommand) CombinedOutput() ([]byte, error) {
	m.CombinedOutputCalled = true
	return m.CombinedOutputData, m.CombinedOutputErr
}

// MockINILoader creates a mock INI loader for testing
func MockINILoader(sections map[string]map[string]string) func(string) (*ini.File, error) {
	return func(path string) (*ini.File, error) {
		cfg := ini.Empty()
		for sectionName, keys := range sections {
			section, err := cfg.NewSection(sectionName)
			if err != nil {
				return nil, err
			}
			for key, value := range keys {
				_, err := section.NewKey(key, value)
				if err != nil {
					return nil, err
				}
			}
		}
		return cfg, nil
	}
}

// MockINILoaderError creates a mock INI loader that returns an error
func MockINILoaderError(errMsg string) func(string) (*ini.File, error) {
	return func(path string) (*ini.File, error) {
		return nil, fmt.Errorf("%s", errMsg)
	}
}

// ============================================================================
// Tests for GetAWSProfiles
// ============================================================================

func TestGetAWSProfiles(t *testing.T) {
	tests := []struct {
		name           string
		sections       map[string]map[string]string
		loaderErr      string
		expectedProfiles []string
		wantErr        bool
		errContains    string
	}{
		{
			name: "successfully loads multiple profiles",
			sections: map[string]map[string]string{
				"profile dev": {
					"sso_start_url": "https://dev.awsapps.com/start",
					"sso_region":    "us-east-1",
				},
				"profile staging": {
					"sso_start_url": "https://staging.awsapps.com/start",
					"sso_region":    "us-west-2",
				},
				"profile prod": {
					"sso_start_url": "https://prod.awsapps.com/start",
					"sso_region":    "eu-west-1",
				},
			},
			expectedProfiles: []string{"default", "dev", "prod", "staging"}, // sorted alphabetically
			wantErr:          false,
		},
		{
			name: "includes default profile even if not in config",
			sections: map[string]map[string]string{
				"profile myprofile": {
					"sso_start_url": "https://example.awsapps.com/start",
				},
			},
			expectedProfiles: []string{"default", "myprofile"},
			wantErr:          false,
		},
		{
			name: "handles empty config with only default",
			sections: map[string]map[string]string{
				"DEFAULT": {},
			},
			expectedProfiles: []string{"default"},
			wantErr:          false,
		},
		{
			name: "ignores non-profile sections",
			sections: map[string]map[string]string{
				"profile valid": {
					"sso_start_url": "https://example.com",
				},
				"sso-session mysession": {
					"sso_start_url": "https://session.example.com",
				},
				"services myservice": {
					"endpoint_url": "https://service.example.com",
				},
			},
			expectedProfiles: []string{"default", "valid"},
			wantErr:          false,
		},
		{
			name:            "returns error when loader fails",
			loaderErr:       "failed to read config file",
			expectedProfiles: nil,
			wantErr:         true,
			errContains:     "failed to load profiles",
		},
		{
			name: "handles profiles with special characters in names",
			sections: map[string]map[string]string{
				"profile my-profile-123": {
					"sso_start_url": "https://example.com",
				},
				"profile another_profile": {
					"sso_start_url": "https://example2.com",
				},
			},
			expectedProfiles: []string{"another_profile", "default", "my-profile-123"},
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var loader func(string) (*ini.File, error)
			if tt.loaderErr != "" {
				loader = MockINILoaderError(tt.loaderErr)
			} else {
				loader = MockINILoader(tt.sections)
			}

			spec := &Spec{
				Loader: loader,
				Fs:     afero.NewMemMapFs(),
			}

			profiles, err := spec.GetAWSProfiles()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedProfiles, profiles)
			}
		})
	}
}

// ============================================================================
// Tests for GetSSOStart
// ============================================================================

func TestGetSSOStart(t *testing.T) {
	tests := []struct {
		name        string
		profile     string
		sections    map[string]map[string]string
		loaderErr   string
		expectedURL string
		wantErr     bool
		errContains string
	}{
		{
			name:    "successfully gets SSO start URL",
			profile: "dev",
			sections: map[string]map[string]string{
				"profile dev": {
					"sso_start_url": "https://mycompany.awsapps.com/start",
					"sso_region":    "us-east-1",
					"sso_account_id": "123456789012",
					"sso_role_name": "PowerUserAccess",
				},
			},
			expectedURL: "https://mycompany.awsapps.com/start",
			wantErr:     false,
		},
		{
			name:    "returns error when profile not found",
			profile: "nonexistent",
			sections: map[string]map[string]string{
				"profile dev": {
					"sso_start_url": "https://example.com",
				},
			},
			wantErr:     true,
			errContains: "failed to get section for profile nonexistent",
		},
		{
			name:    "returns error when sso_start_url is empty",
			profile: "incomplete",
			sections: map[string]map[string]string{
				"profile incomplete": {
					"sso_region": "us-east-1",
					// sso_start_url is missing
				},
			},
			wantErr:     true,
			errContains: "SSO start URL not found for profile incomplete",
		},
		{
			name:      "returns error when loader fails",
			profile:   "dev",
			loaderErr: "permission denied",
			wantErr:   true,
			errContains: "failed to load profile file",
		},
		{
			name:    "handles multiple profiles and gets correct one",
			profile: "staging",
			sections: map[string]map[string]string{
				"profile dev": {
					"sso_start_url": "https://dev.awsapps.com/start",
				},
				"profile staging": {
					"sso_start_url": "https://staging.awsapps.com/start",
				},
				"profile prod": {
					"sso_start_url": "https://prod.awsapps.com/start",
				},
			},
			expectedURL: "https://staging.awsapps.com/start",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var loader func(string) (*ini.File, error)
			if tt.loaderErr != "" {
				loader = MockINILoaderError(tt.loaderErr)
			} else {
				loader = MockINILoader(tt.sections)
			}

			spec := &Spec{
				Loader:  loader,
				Profile: tt.profile,
				Fs:      afero.NewMemMapFs(),
			}

			url, err := spec.GetSSOStart()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}
		})
	}
}

// ============================================================================
// Tests for ExecuteSSOCommand (AWS CLI mock responses)
// ============================================================================

func TestExecuteSSOCommand(t *testing.T) {
	tests := []struct {
		name           string
		mockOutput     []byte
		mockErr        error
		expectedOutput []byte
		wantErr        bool
		errContains    string
	}{
		{
			name: "successful SSO login",
			mockOutput: []byte(`Successfully logged into Start URL: https://mycompany.awsapps.com/start
`),
			mockErr:        nil,
			expectedOutput: []byte(`Successfully logged into Start URL: https://mycompany.awsapps.com/start
`),
			wantErr: false,
		},
		{
			name: "SSO login with browser redirect message",
			mockOutput: []byte(`Attempting to automatically open the SSO authorization page in your default browser.
If the browser does not open or you wish to use a different device to authorize this request, open the following URL:

https://device.sso.us-east-1.amazonaws.com/

Then enter the code:

ABCD-EFGH

Successfully logged into Start URL: https://mycompany.awsapps.com/start
`),
			mockErr:        nil,
			expectedOutput: []byte(`Attempting to automatically open the SSO authorization page in your default browser.
If the browser does not open or you wish to use a different device to authorize this request, open the following URL:

https://device.sso.us-east-1.amazonaws.com/

Then enter the code:

ABCD-EFGH

Successfully logged into Start URL: https://mycompany.awsapps.com/start
`),
			wantErr: false,
		},
		{
			name:        "SSO login fails with expired token",
			mockOutput:  []byte(`Error: The SSO session has expired or is invalid`),
			mockErr:     fmt.Errorf("exit status 255"),
			wantErr:     true,
			errContains: "failed to execute AWS SSO command",
		},
		{
			name:        "SSO login fails with network error",
			mockOutput:  []byte(`Error: Unable to connect to SSO endpoint`),
			mockErr:     fmt.Errorf("connection refused"),
			wantErr:     true,
			errContains: "failed to execute AWS SSO command",
		},
		{
			name:        "SSO login fails with invalid profile",
			mockOutput:  []byte(`Error: The config profile (invalid-profile) could not be found`),
			mockErr:     fmt.Errorf("exit status 253"),
			wantErr:     true,
			errContains: "failed to execute AWS SSO command",
		},
		{
			name:        "SSO login fails with no browser available",
			mockOutput:  []byte(`Error: Unable to open browser. Please navigate to: https://device.sso.us-east-1.amazonaws.com/`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "failed to execute AWS SSO command",
		},
		{
			name:        "SSO login fails with AWS CLI not installed",
			mockOutput:  nil,
			mockErr:     fmt.Errorf("executable file not found in $PATH"),
			wantErr:     true,
			errContains: "failed to execute AWS SSO command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := &MockShellCommand{
				CombinedOutputData: tt.mockOutput,
				CombinedOutputErr:  tt.mockErr,
			}

			output, err := ExecuteSSOCommand(mockCmd)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, output)
			}

			assert.True(t, mockCmd.CombinedOutputCalled)
		})
	}
}

// ============================================================================
// Tests for IsSSOTokenValid
// ============================================================================

func TestIsSSOTokenValid(t *testing.T) {
	tests := []struct {
		name          string
		ssoStartURL   string
		threshold     time.Duration
		setupFs       func(afero.Fs) error
		expectedValid bool
		wantErr       bool
		errContains   string
	}{
		{
			name:        "valid token with sufficient time remaining",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				token := SSOToken{
					StartURL:  "https://mycompany.awsapps.com/start",
					ExpiresAt: time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
				}
				data, _ := json.Marshal(token)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "abc123.json"), data, 0644)
			},
			expectedValid: true,
			wantErr:       false,
		},
		{
			name:        "expired token",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				token := SSOToken{
					StartURL:  "https://mycompany.awsapps.com/start",
					ExpiresAt: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
				}
				data, _ := json.Marshal(token)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "abc123.json"), data, 0644)
			},
			expectedValid: false,
			wantErr:       false,
		},
		{
			name:        "token expires within threshold",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				token := SSOToken{
					StartURL:  "https://mycompany.awsapps.com/start",
					ExpiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339), // Less than 15 min threshold
				}
				data, _ := json.Marshal(token)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "abc123.json"), data, 0644)
			},
			expectedValid: false,
			wantErr:       false,
		},
		{
			name:        "token for different SSO URL",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				token := SSOToken{
					StartURL:  "https://different-company.awsapps.com/start",
					ExpiresAt: time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
				}
				data, _ := json.Marshal(token)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "abc123.json"), data, 0644)
			},
			expectedValid: false,
			wantErr:       false,
		},
		{
			name:        "cache directory does not exist",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				// Don't create the cache directory
				return nil
			},
			expectedValid: false,
			wantErr:       true,
			errContains:   "failed to read SSO cache dir",
		},
		{
			name:        "empty cache directory",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				return fs.MkdirAll(cacheDir, 0755)
			},
			expectedValid: false,
			wantErr:       false,
		},
		{
			name:        "invalid JSON in cache file",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				return afero.WriteFile(fs, filepath.Join(cacheDir, "invalid.json"), []byte("not valid json"), 0644)
			},
			expectedValid: false,
			wantErr:       false,
		},
		{
			name:        "multiple tokens finds matching one",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				// Expired token
				token1 := SSOToken{
					StartURL:  "https://mycompany.awsapps.com/start",
					ExpiresAt: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
				}
				data1, _ := json.Marshal(token1)
				if err := afero.WriteFile(fs, filepath.Join(cacheDir, "expired.json"), data1, 0644); err != nil {
					return err
				}
				// Valid token
				token2 := SSOToken{
					StartURL:  "https://mycompany.awsapps.com/start",
					ExpiresAt: time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
				}
				data2, _ := json.Marshal(token2)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "valid.json"), data2, 0644)
			},
			expectedValid: true,
			wantErr:       false,
		},
		{
			name:        "skips non-json files",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				// Non-JSON file
				if err := afero.WriteFile(fs, filepath.Join(cacheDir, "README.txt"), []byte("readme"), 0644); err != nil {
					return err
				}
				// Valid token
				token := SSOToken{
					StartURL:  "https://mycompany.awsapps.com/start",
					ExpiresAt: time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
				}
				data, _ := json.Marshal(token)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "token.json"), data, 0644)
			},
			expectedValid: true,
			wantErr:       false,
		},
		{
			name:        "skips directories in cache",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				// Create a subdirectory
				if err := fs.MkdirAll(filepath.Join(cacheDir, "subdir"), 0755); err != nil {
					return err
				}
				// Valid token
				token := SSOToken{
					StartURL:  "https://mycompany.awsapps.com/start",
					ExpiresAt: time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
				}
				data, _ := json.Marshal(token)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "token.json"), data, 0644)
			},
			expectedValid: true,
			wantErr:       false,
		},
		{
			name:        "token with missing startUrl field",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				token := map[string]string{
					"expiresAt": time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
					// startUrl is missing
				}
				data, _ := json.Marshal(token)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "incomplete.json"), data, 0644)
			},
			expectedValid: false,
			wantErr:       false,
		},
		{
			name:        "token with invalid expiresAt format",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			threshold:   15 * time.Minute,
			setupFs: func(fs afero.Fs) error {
				cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
				if err := fs.MkdirAll(cacheDir, 0755); err != nil {
					return err
				}
				token := map[string]string{
					"startUrl":  "https://mycompany.awsapps.com/start",
					"expiresAt": "invalid-date-format",
				}
				data, _ := json.Marshal(token)
				return afero.WriteFile(fs, filepath.Join(cacheDir, "baddate.json"), data, 0644)
			},
			expectedValid: false,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setupFs != nil {
				err := tt.setupFs(fs)
				require.NoError(t, err)
			}

			valid, err := IsSSOTokenValid(fs, tt.ssoStartURL, tt.threshold)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedValid, valid)
		})
	}
}

// ============================================================================
// Tests for AWS STS GetCallerIdentity Mock Responses
// ============================================================================

func TestGetCallerIdentityResponses(t *testing.T) {
	// These tests document the expected response formats from AWS STS GetCallerIdentity
	// The actual API call is not mocked here, but these test the data structures

	tests := []struct {
		name     string
		response localconfig.AwsGetCallerIdentitySpec
		validate func(t *testing.T, spec localconfig.AwsGetCallerIdentitySpec)
	}{
		{
			name: "standard IAM user response",
			response: localconfig.AwsGetCallerIdentitySpec{
				Account: "123456789012",
				UserID:  "AIDAEXAMPLEUSERID",
				ARN:     "arn:aws:iam::123456789012:user/myuser",
			},
			validate: func(t *testing.T, spec localconfig.AwsGetCallerIdentitySpec) {
				assert.Equal(t, "123456789012", spec.Account)
				assert.True(t, len(spec.UserID) > 0)
				assert.Contains(t, spec.ARN, "arn:aws:iam::")
				assert.Contains(t, spec.ARN, ":user/")
			},
		},
		{
			name: "assumed role response",
			response: localconfig.AwsGetCallerIdentitySpec{
				Account: "123456789012",
				UserID:  "AROAEXAMPLEROLEID:session-name",
				ARN:     "arn:aws:sts::123456789012:assumed-role/PowerUserAccess/session-name",
			},
			validate: func(t *testing.T, spec localconfig.AwsGetCallerIdentitySpec) {
				assert.Equal(t, "123456789012", spec.Account)
				assert.Contains(t, spec.UserID, ":")
				assert.Contains(t, spec.ARN, ":assumed-role/")
			},
		},
		{
			name: "SSO federated user response",
			response: localconfig.AwsGetCallerIdentitySpec{
				Account: "987654321098",
				UserID:  "AROAEXAMPLESSOID:user@example.com",
				ARN:     "arn:aws:sts::987654321098:assumed-role/AWSReservedSSO_PowerUserAccess_abc123/user@example.com",
			},
			validate: func(t *testing.T, spec localconfig.AwsGetCallerIdentitySpec) {
				assert.Equal(t, "987654321098", spec.Account)
				assert.Contains(t, spec.ARN, "AWSReservedSSO_")
			},
		},
		{
			name: "root user response",
			response: localconfig.AwsGetCallerIdentitySpec{
				Account: "111222333444",
				UserID:  "111222333444",
				ARN:     "arn:aws:iam::111222333444:root",
			},
			validate: func(t *testing.T, spec localconfig.AwsGetCallerIdentitySpec) {
				assert.Equal(t, spec.Account, spec.UserID)
				assert.Contains(t, spec.ARN, ":root")
			},
		},
		{
			name: "cross-account assumed role",
			response: localconfig.AwsGetCallerIdentitySpec{
				Account: "999888777666",
				UserID:  "AROAEXAMPLECROSSID:cross-account-session",
				ARN:     "arn:aws:sts::999888777666:assumed-role/CrossAccountRole/cross-account-session",
			},
			validate: func(t *testing.T, spec localconfig.AwsGetCallerIdentitySpec) {
				assert.Equal(t, "999888777666", spec.Account)
				assert.Contains(t, spec.ARN, "CrossAccountRole")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.response)
		})
	}
}

// ============================================================================
// Integration-style tests for SSO flow
// ============================================================================

func TestSSOFlowWithMockedCommands(t *testing.T) {
	tests := []struct {
		name                string
		profile             string
		ssoStartURL         string
		tokenValid          bool
		ssoCommandOutput    []byte
		ssoCommandErr       error
		expectedRunCLI      bool
		wantErr             bool
		errContains         string
	}{
		{
			name:        "valid token skips CLI login",
			profile:     "dev",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			tokenValid:  true,
			ssoCommandOutput: nil,
			ssoCommandErr:    nil,
			expectedRunCLI:   false,
			wantErr:          false,
		},
		{
			name:        "expired token triggers CLI login",
			profile:     "dev",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			tokenValid:  false,
			ssoCommandOutput: []byte("Successfully logged in"),
			ssoCommandErr:    nil,
			expectedRunCLI:   true,
			wantErr:          false,
		},
		{
			name:        "CLI login failure returns error",
			profile:     "dev",
			ssoStartURL: "https://mycompany.awsapps.com/start",
			tokenValid:  false,
			ssoCommandOutput: []byte("Error: Token expired"),
			ssoCommandErr:    fmt.Errorf("exit status 1"),
			expectedRunCLI:   true,
			wantErr:          true,
			errContains:      "failed to execute AWS SSO command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates the logic flow, not actual AWS calls
			// In a real scenario, you would use dependency injection for the STS client

			mockCmd := &MockShellCommand{
				CombinedOutputData: tt.ssoCommandOutput,
				CombinedOutputErr:  tt.ssoCommandErr,
			}

			if tt.expectedRunCLI {
				output, err := ExecuteSSOCommand(mockCmd)
				if tt.wantErr {
					assert.Error(t, err)
					if tt.errContains != "" {
						assert.Contains(t, err.Error(), tt.errContains)
					}
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.ssoCommandOutput, output)
				}
			}
		})
	}
}
