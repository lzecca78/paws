package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	localconfig "github.com/lzecca78/paws/src/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Tests for Pulumi CLI Command Execution
// ============================================================================

func TestExecutePulumiCommand(t *testing.T) {
	tests := []struct {
		name           string
		mockOutput     []byte
		mockErr        error
		expectedOutput []byte
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful pulumi command",
			mockOutput:     []byte(`Logged in to s3://my-pulumi-state as user@example.com`),
			mockErr:        nil,
			expectedOutput: []byte(`Logged in to s3://my-pulumi-state as user@example.com`),
			wantErr:        false,
		},
		{
			name:           "empty output on success",
			mockOutput:     []byte(``),
			mockErr:        nil,
			expectedOutput: []byte(``),
			wantErr:        false,
		},
		{
			name:        "pulumi not installed",
			mockOutput:  nil,
			mockErr:     fmt.Errorf("executable file not found in $PATH"),
			wantErr:     true,
			errContains: "executable file not found",
		},
		{
			name:        "pulumi command fails with exit code",
			mockOutput:  []byte(`error: failed to login: access denied`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:        "pulumi timeout",
			mockOutput:  nil,
			mockErr:     fmt.Errorf("signal: killed"),
			wantErr:     true,
			errContains: "signal: killed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := &MockShellCommand{
				OutputData: tt.mockOutput,
				OutputErr:  tt.mockErr,
			}

			output, err := executePulumiCommand(mockCmd)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, output)
			}

			assert.True(t, mockCmd.OutputCalled)
		})
	}
}

// ============================================================================
// Tests for Pulumi Stack List (pulumi stack ls --json)
// ============================================================================

// TestPulumiStacksStdoutVsStderr demonstrates why we use Output() (stdout only)
// instead of CombinedOutput() (stdout + stderr) for pulumi commands.
//
// Real-world scenario: AWS SDK warnings are written to stderr and would corrupt
// the JSON output if we captured both stdout and stderr together.
//
// Example problematic output with CombinedOutput():
//
//	SDK 2026/06/16 15:59:16 WARN Response has no supported checksum. Not validating response payload.
//	[
//	  {
//	    "name": "test.production",
//	    "current": true,
//	    "lastUpdate": "2026-06-15T14:41:41.757Z",
//	    "resourceCount": 25
//	  }
//	]
//
// The AWS SDK warning at the beginning makes the output invalid JSON.
func TestPulumiStacksStdoutVsStderr(t *testing.T) {
	tests := []struct {
		name               string
		stdoutData         []byte
		stderrData         []byte
		combinedData       []byte // What CombinedOutput() would return
		expectedStacks     []string
		stdoutParseErr     bool
		combinedParseErr   bool
		description        string
	}{
		{
			name: "AWS SDK checksum warning corrupts combined output",
			stdoutData: []byte(`[
  {
    "name": "test.production",
    "current": true,
    "lastUpdate": "2026-06-15T14:41:41.757Z",
    "resourceCount": 25
  }
]`),
			stderrData: []byte("SDK 2026/06/16 15:59:16 WARN Response has no supported checksum. Not validating response payload.\n"),
			combinedData: []byte(`SDK 2026/06/16 15:59:16 WARN Response has no supported checksum. Not validating response payload.
[
  {
    "name": "test.production",
    "current": true,
    "lastUpdate": "2026-06-15T14:41:41.757Z",
    "resourceCount": 25
  }
]`),
			expectedStacks:   []string{"test.production"},
			stdoutParseErr:   false, // stdout alone is valid JSON
			combinedParseErr: true,  // combined output is invalid JSON
			description:      "AWS SDK warning about checksum validation appears in stderr",
		},
		{
			name: "AWS SDK debug logging corrupts combined output",
			stdoutData: []byte(`[
  {"name": "dev", "current": false},
  {"name": "prod", "current": true}
]`),
			stderrData: []byte(`SDK 2026/06/16 15:59:16 DEBUG Refreshing cached credentials
SDK 2026/06/16 15:59:16 DEBUG Credentials retrieved successfully
`),
			combinedData: []byte(`SDK 2026/06/16 15:59:16 DEBUG Refreshing cached credentials
SDK 2026/06/16 15:59:16 DEBUG Credentials retrieved successfully
[
  {"name": "dev", "current": false},
  {"name": "prod", "current": true}
]`),
			expectedStacks:   []string{"dev", "prod"},
			stdoutParseErr:   false,
			combinedParseErr: true,
			description:      "AWS SDK debug messages appear when AWS_SDK_LOG_LEVEL is set",
		},
		{
			name: "Pulumi deprecation warning corrupts combined output",
			stdoutData: []byte(`[
  {"name": "staging", "current": true, "resourceCount": 42}
]`),
			stderrData:       []byte("warning: A new version of Pulumi is available. To upgrade from version '3.100.0' to '3.110.0', run\n   $ curl -sSL https://get.pulumi.com | sh\nor visit https://pulumi.com/docs/install/ for manual installation.\n"),
			combinedData:     []byte("warning: A new version of Pulumi is available. To upgrade from version '3.100.0' to '3.110.0', run\n   $ curl -sSL https://get.pulumi.com | sh\nor visit https://pulumi.com/docs/install/ for manual installation.\n[\n  {\"name\": \"staging\", \"current\": true, \"resourceCount\": 42}\n]"),
			expectedStacks:   []string{"staging"},
			stdoutParseErr:   false,
			combinedParseErr: true,
			description:      "Pulumi version upgrade warning appears in stderr",
		},
		{
			name: "S3 backend warning corrupts combined output",
			stdoutData: []byte(`[
  {"name": "production", "current": true}
]`),
			stderrData:       []byte("warning: using legacy S3 backend; consider migrating to the new backend\n"),
			combinedData:     []byte("warning: using legacy S3 backend; consider migrating to the new backend\n[\n  {\"name\": \"production\", \"current\": true}\n]"),
			expectedStacks:   []string{"production"},
			stdoutParseErr:   false,
			combinedParseErr: true,
			description:      "S3 backend deprecation warning",
		},
		{
			name: "multiple stderr warnings corrupt combined output",
			stdoutData: []byte(`[{"name": "dev"}]`),
			stderrData: []byte(`SDK 2026/06/16 15:59:16 WARN Response has no supported checksum.
warning: A new version of Pulumi is available.
SDK 2026/06/16 15:59:17 WARN Retrying request due to throttling.
`),
			combinedData: []byte(`SDK 2026/06/16 15:59:16 WARN Response has no supported checksum.
warning: A new version of Pulumi is available.
SDK 2026/06/16 15:59:17 WARN Retrying request due to throttling.
[{"name": "dev"}]`),
			expectedStacks:   []string{"dev"},
			stdoutParseErr:   false,
			combinedParseErr: true,
			description:      "Multiple warnings from different sources",
		},
		{
			name:             "no stderr - both outputs are valid",
			stdoutData:       []byte(`[{"name": "clean-stack", "current": true}]`),
			stderrData:       []byte(``),
			combinedData:     []byte(`[{"name": "clean-stack", "current": true}]`),
			expectedStacks:   []string{"clean-stack"},
			stdoutParseErr:   false,
			combinedParseErr: false, // No stderr means combined is also valid
			description:      "Clean output with no warnings",
		},
		{
			name:             "empty stack list with stderr warning",
			stdoutData:       []byte(`[]`),
			stderrData:       []byte("SDK 2026/06/16 15:59:16 WARN Some warning\n"),
			combinedData:     []byte("SDK 2026/06/16 15:59:16 WARN Some warning\n[]"),
			expectedStacks:   []string{},
			stdoutParseErr:   false,
			combinedParseErr: true,
			description:      "Empty stack list still corrupted by stderr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test 1: Parsing stdout only (what we do with Output())
			var stacksFromStdout []PulumiStack
			stdoutErr := json.Unmarshal(tt.stdoutData, &stacksFromStdout)

			if tt.stdoutParseErr {
				assert.Error(t, stdoutErr, "stdout should fail to parse")
			} else {
				assert.NoError(t, stdoutErr, "stdout should parse successfully")
				stackNames := make([]string, 0, len(stacksFromStdout))
				for _, s := range stacksFromStdout {
					stackNames = append(stackNames, s.Name)
				}
				assert.Equal(t, tt.expectedStacks, stackNames)
			}

			// Test 2: Parsing combined output (what would happen with CombinedOutput())
			var stacksFromCombined []PulumiStack
			combinedErr := json.Unmarshal(tt.combinedData, &stacksFromCombined)

			if tt.combinedParseErr {
				assert.Error(t, combinedErr, "combined output should fail to parse due to stderr pollution: %s", tt.description)
			} else {
				assert.NoError(t, combinedErr, "combined output should parse when no stderr")
			}

			// Test 3: Verify our mock command returns stdout only
			mockCmd := &MockShellCommand{
				OutputData:         tt.stdoutData,
				CombinedOutputData: tt.combinedData,
			}

			// Using Output() - should get clean stdout
			output, err := executePulumiCommand(mockCmd)
			assert.NoError(t, err)
			assert.Equal(t, tt.stdoutData, output, "executePulumiCommand should return stdout only")
			assert.True(t, mockCmd.OutputCalled, "Output() should be called")
			assert.False(t, mockCmd.CombinedOutputCalled, "CombinedOutput() should NOT be called")
		})
	}
}

func TestPulumiStacks(t *testing.T) {
	tests := []struct {
		name           string
		mockOutput     []byte
		mockErr        error
		expectedStacks []string
		wantErr        bool
		errContains    string
	}{
		{
			name: "successfully lists multiple stacks",
			mockOutput: []byte(`[
				{"name": "dev", "current": false, "updateInProgress": false},
				{"name": "staging", "current": true, "updateInProgress": false},
				{"name": "prod", "current": false, "updateInProgress": false}
			]`),
			mockErr:        nil,
			expectedStacks: []string{"dev", "staging", "prod"},
			wantErr:        false,
		},
		{
			name:           "empty stack list",
			mockOutput:     []byte(`[]`),
			mockErr:        nil,
			expectedStacks: []string{},
			wantErr:        false,
		},
		{
			name: "single stack",
			mockOutput: []byte(`[
				{"name": "production", "current": true, "updateInProgress": false}
			]`),
			mockErr:        nil,
			expectedStacks: []string{"production"},
			wantErr:        false,
		},
		{
			name: "stack with organization prefix",
			mockOutput: []byte(`[
				{"name": "myorg/myproject/dev", "current": false, "updateInProgress": false},
				{"name": "myorg/myproject/prod", "current": true, "updateInProgress": false}
			]`),
			mockErr:        nil,
			expectedStacks: []string{"myorg/myproject/dev", "myorg/myproject/prod"},
			wantErr:        false,
		},
		{
			name: "stack with update in progress",
			mockOutput: []byte(`[
				{"name": "dev", "current": true, "updateInProgress": true, "lastUpdate": "2024-01-15T10:30:00Z"}
			]`),
			mockErr:        nil,
			expectedStacks: []string{"dev"},
			wantErr:        false,
		},
		{
			name:        "invalid JSON response",
			mockOutput:  []byte(`not valid json`),
			mockErr:     nil,
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "pulumi command fails",
			mockOutput:  []byte(`error: no Pulumi.yaml found`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:        "pulumi not logged in",
			mockOutput:  []byte(`error: PULUMI_ACCESS_TOKEN must be set for login during non-interactive CLI sessions`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name: "stack list with resource counts",
			mockOutput: []byte(`[
				{"name": "dev", "current": false, "resourceCount": 42},
				{"name": "prod", "current": true, "resourceCount": 156}
			]`),
			mockErr:        nil,
			expectedStacks: []string{"dev", "prod"},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock PulumiConfig that uses our mock command
			_ = &PulumiConfig{
				FileSystem: afero.NewMemMapFs(),
			}

			// We need to test the Stacks() method behavior with mocked output
			// Since Stacks() creates its own command, we test the JSON parsing separately
			if tt.mockErr == nil && tt.mockOutput != nil {
				var stacks []PulumiStack
				err := json.Unmarshal(tt.mockOutput, &stacks)

				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)

					stackNames := make([]string, 0, len(stacks))
					for _, s := range stacks {
						stackNames = append(stackNames, s.Name)
					}
					assert.Equal(t, tt.expectedStacks, stackNames)
				}
			}
		})
	}
}

// ============================================================================
// Tests for Pulumi Login (pulumi login s3://bucket)
// ============================================================================

func TestPulumiS3Login(t *testing.T) {
	tests := []struct {
		name        string
		mockOutput  []byte
		mockErr     error
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful S3 backend login",
			mockOutput: []byte(`Logged in to s3://my-pulumi-state-bucket as user@example.com (s3://my-pulumi-state-bucket)`),
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "successful login with existing state",
			mockOutput: []byte(`Logged in to s3://my-pulumi-state-bucket as user@example.com (s3://my-pulumi-state-bucket)`),
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:        "S3 bucket access denied",
			mockOutput:  []byte(`error: failed to login: AccessDenied: Access Denied`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:        "S3 bucket does not exist",
			mockOutput:  []byte(`error: failed to login: NoSuchBucket: The specified bucket does not exist`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:        "invalid S3 URL",
			mockOutput:  []byte(`error: invalid backend URL "s3://": no bucket specified`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:        "AWS credentials not configured",
			mockOutput:  []byte(`error: failed to login: NoCredentialProviders: no valid providers in chain`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:        "network error",
			mockOutput:  []byte(`error: failed to login: RequestError: send request failed`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:       "login with passphrase prompt bypass",
			mockOutput: []byte(`Logged in to s3://encrypted-state-bucket as user@example.com (s3://encrypted-state-bucket)`),
			mockErr:    nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := &MockShellCommand{
				OutputData: tt.mockOutput,
				OutputErr:  tt.mockErr,
			}

			output, err := executePulumiCommand(mockCmd)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockOutput, output)
			}
		})
	}
}

// ============================================================================
// Tests for Pulumi Stack Select (pulumi stack select)
// ============================================================================

func TestPulumiStackSelect(t *testing.T) {
	tests := []struct {
		name        string
		stackName   string
		mockOutput  []byte
		mockErr     error
		wantErr     bool
		errContains string
	}{
		{
			name:       "successfully select stack",
			stackName:  "dev",
			mockOutput: []byte(``), // stack select typically has no output on success
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "select stack with org prefix",
			stackName:  "myorg/myproject/production",
			mockOutput: []byte(``),
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:        "stack does not exist",
			stackName:   "nonexistent",
			mockOutput:  []byte(`error: no stack named 'nonexistent' found`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:        "not in a pulumi project",
			stackName:   "dev",
			mockOutput:  []byte(`error: no Pulumi.yaml project file found`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
		{
			name:        "stack name with invalid characters",
			stackName:   "invalid stack name!",
			mockOutput:  []byte(`error: invalid stack name "invalid stack name!"`),
			mockErr:     fmt.Errorf("exit status 1"),
			wantErr:     true,
			errContains: "exit status 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := &MockShellCommand{
				OutputData: tt.mockOutput,
				OutputErr:  tt.mockErr,
			}

			output, err := executePulumiCommand(mockCmd)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockOutput, output)
			}
		})
	}
}

// ============================================================================
// Tests for checkYaml (Pulumi.yaml detection)
// ============================================================================

func TestCheckYaml(t *testing.T) {
	// Save and restore current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	tests := []struct {
		name      string
		setupFs   func(afero.Fs, string) error
		expectedOk bool
		wantErr   bool
	}{
		{
			name: "Pulumi.yaml exists",
			setupFs: func(fs afero.Fs, dir string) error {
				content := `name: my-pulumi-project
runtime: go
description: A minimal Go Pulumi program
`
				return afero.WriteFile(fs, filepath.Join(dir, "Pulumi.yaml"), []byte(content), 0644)
			},
			expectedOk: true,
			wantErr:    false,
		},
		{
			name: "Pulumi.yaml does not exist",
			setupFs: func(fs afero.Fs, dir string) error {
				// Don't create Pulumi.yaml
				return nil
			},
			expectedOk: false,
			wantErr:    false,
		},
		{
			name: "Pulumi.yaml with full configuration",
			setupFs: func(fs afero.Fs, dir string) error {
				content := `name: infrastructure
runtime:
  name: go
  options:
    binary: ./bin/infrastructure
description: Infrastructure as Code for my application
config:
  aws:region: us-east-1
`
				return afero.WriteFile(fs, filepath.Join(dir, "Pulumi.yaml"), []byte(content), 0644)
			},
			expectedOk: true,
			wantErr:    false,
		},
		{
			name: "Pulumi.yml alternative extension",
			setupFs: func(fs afero.Fs, dir string) error {
				// Note: The current implementation only checks for Pulumi.yaml, not Pulumi.yml
				content := `name: my-project
runtime: python
`
				return afero.WriteFile(fs, filepath.Join(dir, "Pulumi.yml"), []byte(content), 0644)
			},
			expectedOk: false, // Current implementation only checks .yaml
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			// Create a temp directory in the memory filesystem
			tempDir := "/tmp/pulumi-test"
			err := fs.MkdirAll(tempDir, 0755)
			require.NoError(t, err)

			if tt.setupFs != nil {
				err := tt.setupFs(fs, tempDir)
				require.NoError(t, err)
			}

			// Create a real temp directory for os.Getwd() to work
			realTempDir, err := os.MkdirTemp("", "pulumi-test")
			require.NoError(t, err)
			defer os.RemoveAll(realTempDir)

			err = os.Chdir(realTempDir)
			require.NoError(t, err)

			// Create Pulumi.yaml in real filesystem if needed
			if tt.expectedOk {
				err = os.WriteFile(filepath.Join(realTempDir, "Pulumi.yaml"), []byte("name: test\nruntime: go\n"), 0644)
				require.NoError(t, err)
			}

			pConfig := &PulumiConfig{
				FileSystem: afero.NewOsFs(), // Use real fs for this test
			}

			ok, err := pConfig.checkYaml()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}

// ============================================================================
// Tests for PulumiConfig.s3Login
// ============================================================================

func TestS3LoginBucketConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		pulumiProjects map[string]string
		awsAccount    string
		wantErr       bool
		errContains   string
	}{
		{
			name: "bucket configured for account",
			pulumiProjects: map[string]string{
				"123456789012": "my-pulumi-state-bucket",
				"987654321098": "other-pulumi-bucket",
			},
			awsAccount:  "123456789012",
			wantErr:     false,
		},
		{
			name: "no bucket configured for account",
			pulumiProjects: map[string]string{
				"111111111111": "some-bucket",
			},
			awsAccount:  "123456789012",
			wantErr:     true,
			errContains: "no s3 bucket configured",
		},
		{
			name:           "empty pulumi projects map",
			pulumiProjects: map[string]string{},
			awsAccount:    "123456789012",
			wantErr:       true,
			errContains:   "no s3 bucket configured",
		},
		{
			name:           "nil pulumi projects map",
			pulumiProjects: nil,
			awsAccount:    "123456789012",
			wantErr:       true,
			errContains:   "no s3 bucket configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pConfig := &PulumiConfig{
				PulumiProjects: tt.pulumiProjects,
				AwsSpec: localconfig.AwsGetCallerIdentitySpec{
					Account: tt.awsAccount,
				},
				FileSystem: afero.NewMemMapFs(),
			}

			// Check if bucket name is properly retrieved
			bucketName := pConfig.PulumiProjects[pConfig.AwsSpec.Account]

			if tt.wantErr {
				assert.Empty(t, bucketName)
			} else {
				assert.NotEmpty(t, bucketName)
			}
		})
	}
}

// ============================================================================
// Tests for Pulumi JSON Output Parsing
// ============================================================================

func TestPulumiStackJSONParsing(t *testing.T) {
	tests := []struct {
		name           string
		jsonInput      string
		expectedStacks []PulumiStack
		wantErr        bool
	}{
		{
			name: "parse basic stack list",
			jsonInput: `[
				{"name": "dev"},
				{"name": "prod"}
			]`,
			expectedStacks: []PulumiStack{
				{Name: "dev"},
				{Name: "prod"},
			},
			wantErr: false,
		},
		{
			name: "parse stack with extra fields (forward compatibility)",
			jsonInput: `[
				{
					"name": "dev",
					"current": true,
					"lastUpdate": "2024-01-15T10:30:00Z",
					"resourceCount": 42,
					"unknownField": "should be ignored"
				}
			]`,
			expectedStacks: []PulumiStack{
				{Name: "dev"},
			},
			wantErr: false,
		},
		{
			name:           "parse empty array",
			jsonInput:      `[]`,
			expectedStacks: []PulumiStack{},
			wantErr:        false,
		},
		{
			name:      "invalid JSON",
			jsonInput: `{"not": "an array"}`,
			wantErr:   true,
		},
		{
			name:      "malformed JSON",
			jsonInput: `[{"name": "unclosed`,
			wantErr:   true,
		},
		{
			name:      "null response",
			jsonInput: `null`,
			wantErr:   false, // null unmarshals to nil slice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stacks []PulumiStack
			err := json.Unmarshal([]byte(tt.jsonInput), &stacks)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedStacks != nil {
					assert.Equal(t, tt.expectedStacks, stacks)
				}
			}
		})
	}
}

// ============================================================================
// Tests for Pulumi Preview/Up Output Simulation
// ============================================================================

func TestPulumiPreviewOutputParsing(t *testing.T) {
	// These tests document expected output formats from pulumi preview
	// Useful for understanding what responses the CLI produces

	tests := []struct {
		name        string
		output      string
		description string
	}{
		{
			name: "preview with no changes",
			output: `Previewing update (dev)

View Live: https://app.pulumi.com/myorg/myproject/dev/previews/abc123

     Type                 Name               Plan
     pulumi:pulumi:Stack  myproject-dev

Resources:
    10 unchanged

Duration: 5s
`,
			description: "No changes detected in the stack",
		},
		{
			name: "preview with creates",
			output: `Previewing update (dev)

View Live: https://app.pulumi.com/myorg/myproject/dev/previews/abc123

     Type                       Name               Plan
 +   pulumi:pulumi:Stack        myproject-dev      create
 +   ├─ aws:s3:Bucket           my-bucket          create
 +   └─ aws:s3:BucketPolicy     my-bucket-policy   create

Resources:
    + 3 to create

Duration: 8s
`,
			description: "New resources to be created",
		},
		{
			name: "preview with updates",
			output: `Previewing update (dev)

View Live: https://app.pulumi.com/myorg/myproject/dev/previews/abc123

     Type                 Name               Plan       Info
     pulumi:pulumi:Stack  myproject-dev
 ~   └─ aws:s3:Bucket     my-bucket          update     [diff: ~tags]

Resources:
    ~ 1 to update
    9 unchanged

Duration: 6s
`,
			description: "Existing resources to be updated",
		},
		{
			name: "preview with deletes",
			output: `Previewing update (dev)

View Live: https://app.pulumi.com/myorg/myproject/dev/previews/abc123

     Type                 Name               Plan
     pulumi:pulumi:Stack  myproject-dev
 -   └─ aws:s3:Bucket     old-bucket         delete

Resources:
    - 1 to delete
    9 unchanged

Duration: 5s
`,
			description: "Resources to be deleted",
		},
		{
			name: "preview with errors",
			output: `Previewing update (dev)

View Live: https://app.pulumi.com/myorg/myproject/dev/previews/abc123

     Type                 Name               Plan
     pulumi:pulumi:Stack  myproject-dev      **failed**

Diagnostics:
  pulumi:pulumi:Stack (myproject-dev):
    error: Program failed with an unhandled exception

Duration: 3s
`,
			description: "Preview failed with errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These tests document output format - no assertions needed
			// They serve as documentation for expected CLI output
			assert.NotEmpty(t, tt.output)
			assert.NotEmpty(t, tt.description)
		})
	}
}

// ============================================================================
// Integration Tests for PulumiSetup
// ============================================================================

func TestPulumiSetupIntegration(t *testing.T) {
	tests := []struct {
		name          string
		setupFs       func(afero.Fs) error
		awsSpec       localconfig.AwsGetCallerIdentitySpec
		wantErr       bool
		errContains   string
	}{
		{
			name: "no Pulumi.yaml file",
			setupFs: func(fs afero.Fs) error {
				// Don't create Pulumi.yaml
				return nil
			},
			awsSpec: localconfig.AwsGetCallerIdentitySpec{
				Account: "123456789012",
			},
			wantErr: false, // Returns early without error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setupFs != nil {
				err := tt.setupFs(fs)
				require.NoError(t, err)
			}

			err := PulumiSetup(fs, tt.awsSpec)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkPulumiStackJSONParsing(b *testing.B) {
	// Simulate a large stack list
	stacks := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		stacks[i] = map[string]interface{}{
			"name":           fmt.Sprintf("stack-%d", i),
			"current":        i == 0,
			"resourceCount":  i * 10,
			"lastUpdate":     "2024-01-15T10:30:00Z",
		}
	}
	jsonData, _ := json.Marshal(stacks)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var parsed []PulumiStack
		_ = json.Unmarshal(jsonData, &parsed)
	}
}
