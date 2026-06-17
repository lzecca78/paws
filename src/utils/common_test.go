package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Tests for TouchFile
// ============================================================================

func TestTouchFile(t *testing.T) {
	tests := []struct {
		name      string
		setupFs   func(afero.Fs) error
		filePath  string
		wantErr   bool
		checkFile bool
	}{
		{
			name:      "creates new file",
			setupFs:   nil,
			filePath:  "/tmp/newfile.txt",
			wantErr:   false,
			checkFile: true,
		},
		{
			name: "touches existing file",
			setupFs: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/tmp", 0755); err != nil {
					return err
				}
				return afero.WriteFile(fs, "/tmp/existing.txt", []byte("content"), 0644)
			},
			filePath:  "/tmp/existing.txt",
			wantErr:   false,
			checkFile: true,
		},
		{
			name: "creates file in nested directory",
			setupFs: func(fs afero.Fs) error {
				return fs.MkdirAll("/tmp/nested/deep/dir", 0755)
			},
			filePath:  "/tmp/nested/deep/dir/file.txt",
			wantErr:   false,
			checkFile: true,
		},
		{
			name: "preserves existing file content",
			setupFs: func(fs afero.Fs) error {
				if err := fs.MkdirAll("/tmp", 0755); err != nil {
					return err
				}
				return afero.WriteFile(fs, "/tmp/withcontent.txt", []byte("original content"), 0644)
			},
			filePath:  "/tmp/withcontent.txt",
			wantErr:   false,
			checkFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			if tt.setupFs != nil {
				err := tt.setupFs(fs)
				require.NoError(t, err)
			} else {
				// Create parent directory for cases without setup
				parentDir := filepath.Dir(tt.filePath)
				err := fs.MkdirAll(parentDir, 0755)
				require.NoError(t, err)
			}

			err := TouchFile(fs, tt.filePath)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFile {
					exists, err := afero.Exists(fs, tt.filePath)
					assert.NoError(t, err)
					assert.True(t, exists, "file should exist after TouchFile")
				}
			}
		})
	}
}

// TestTouchFilePreservesContent verifies that TouchFile doesn't modify existing content
func TestTouchFilePreservesContent(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/tmp", 0755))

	originalContent := []byte("this is the original content")
	require.NoError(t, afero.WriteFile(fs, "/tmp/preserve.txt", originalContent, 0644))

	err := TouchFile(fs, "/tmp/preserve.txt")
	assert.NoError(t, err)

	content, err := afero.ReadFile(fs, "/tmp/preserve.txt")
	assert.NoError(t, err)
	assert.Equal(t, originalContent, content, "TouchFile should preserve existing content")
}

// ============================================================================
// Tests for WriteFile
// ============================================================================

func TestWriteFile(t *testing.T) {
	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	tests := []struct {
		name            string
		profile         string
		location        string
		homeDir         string
		setupFs         func(afero.Fs, string) error
		expectedContent string
		wantErr         bool
	}{
		{
			name:     "writes non-default profile",
			profile:  "dev",
			location: "/home/testuser",
			homeDir:  "/home/testuser",
			setupFs: func(fs afero.Fs, home string) error {
				return fs.MkdirAll(home, 0755)
			},
			expectedContent: "dev",
			wantErr:         false,
		},
		{
			name:     "writes empty content for default profile",
			profile:  "default",
			location: "/home/testuser",
			homeDir:  "/home/testuser",
			setupFs: func(fs afero.Fs, home string) error {
				return fs.MkdirAll(home, 0755)
			},
			expectedContent: "",
			wantErr:         false,
		},
		{
			name:     "writes staging profile",
			profile:  "staging",
			location: "/home/testuser",
			homeDir:  "/home/testuser",
			setupFs: func(fs afero.Fs, home string) error {
				return fs.MkdirAll(home, 0755)
			},
			expectedContent: "staging",
			wantErr:         false,
		},
		{
			name:     "writes production profile",
			profile:  "production",
			location: "/home/testuser",
			homeDir:  "/home/testuser",
			setupFs: func(fs afero.Fs, home string) error {
				return fs.MkdirAll(home, 0755)
			},
			expectedContent: "production",
			wantErr:         false,
		},
		{
			name:     "overwrites existing .paws file",
			profile:  "newprofile",
			location: "/home/testuser",
			homeDir:  "/home/testuser",
			setupFs: func(fs afero.Fs, home string) error {
				if err := fs.MkdirAll(home, 0755); err != nil {
					return err
				}
				return afero.WriteFile(fs, filepath.Join(home, ".paws"), []byte("oldprofile"), 0644)
			},
			expectedContent: "newprofile",
			wantErr:         false,
		},
		{
			name:     "handles profile with special characters",
			profile:  "my-profile_123",
			location: "/home/testuser",
			homeDir:  "/home/testuser",
			setupFs: func(fs afero.Fs, home string) error {
				return fs.MkdirAll(home, 0755)
			},
			expectedContent: "my-profile_123",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			// Set HOME environment variable
			_ = os.Setenv("HOME", tt.homeDir)

			if tt.setupFs != nil {
				err := tt.setupFs(fs, tt.homeDir)
				require.NoError(t, err)
			}

			err := WriteFile(fs, tt.profile, tt.location)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify file content
				content, err := afero.ReadFile(fs, filepath.Join(tt.location, ".paws"))
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedContent, string(content))

				// Verify file also created in home directory
				exists, err := afero.Exists(fs, filepath.Join(tt.homeDir, ".paws"))
				assert.NoError(t, err)
				assert.True(t, exists)
			}
		})
	}
}

// ============================================================================
// Tests for GetEnv
// ============================================================================

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		fallback    string
		envValue    string
		setEnv      bool
		expected    string
	}{
		{
			name:     "returns env value when set",
			key:      "TEST_VAR_1",
			fallback: "default_value",
			envValue: "actual_value",
			setEnv:   true,
			expected: "actual_value",
		},
		{
			name:     "returns fallback when env not set",
			key:      "TEST_VAR_2",
			fallback: "fallback_value",
			envValue: "",
			setEnv:   false,
			expected: "fallback_value",
		},
		{
			name:     "returns fallback when env is empty string",
			key:      "TEST_VAR_3",
			fallback: "fallback_for_empty",
			envValue: "",
			setEnv:   true,
			expected: "fallback_for_empty",
		},
		{
			name:     "returns env value with special characters",
			key:      "TEST_VAR_4",
			fallback: "default",
			envValue: "value-with_special.chars/and:more",
			setEnv:   true,
			expected: "value-with_special.chars/and:more",
		},
		{
			name:     "returns env value with spaces",
			key:      "TEST_VAR_5",
			fallback: "no spaces",
			envValue: "value with spaces",
			setEnv:   true,
			expected: "value with spaces",
		},
		{
			name:     "returns empty fallback when appropriate",
			key:      "TEST_VAR_6",
			fallback: "",
			envValue: "",
			setEnv:   false,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up env before and after test
			_ = os.Unsetenv(tt.key)
			defer func() { _ = os.Unsetenv(tt.key) }()

			if tt.setEnv {
				_ = os.Setenv(tt.key, tt.envValue)
			}

			result := GetEnv(tt.key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// Tests for GetHomeDir
// ============================================================================

func TestGetHomeDir(t *testing.T) {
	t.Run("returns home directory", func(t *testing.T) {
		homeDir, err := GetHomeDir()
		assert.NoError(t, err)
		assert.NotEmpty(t, homeDir)
		// Verify it's a valid path (exists on the system)
		_, err = os.Stat(homeDir)
		assert.NoError(t, err)
	})

	t.Run("home directory matches os.UserHomeDir", func(t *testing.T) {
		expected, err := os.UserHomeDir()
		require.NoError(t, err)

		actual, err := GetHomeDir()
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

// ============================================================================
// Tests for GetCurrentProfileFile
// ============================================================================

func TestGetCurrentProfileFile(t *testing.T) {
	// Save original env vars and restore after test
	originalHome := os.Getenv("HOME")
	originalConfigFile := os.Getenv("AWS_CONFIG_FILE")
	defer func() {
		_ = os.Setenv("HOME", originalHome)
		if originalConfigFile != "" {
			_ = os.Setenv("AWS_CONFIG_FILE", originalConfigFile)
		} else {
			_ = os.Unsetenv("AWS_CONFIG_FILE")
		}
	}()

	tests := []struct {
		name           string
		homeDir        string
		awsConfigFile  string
		setConfigFile  bool
		expectedSuffix string
		exactMatch     string
	}{
		{
			name:           "returns default path when AWS_CONFIG_FILE not set",
			homeDir:        "/home/testuser",
			awsConfigFile:  "",
			setConfigFile:  false,
			expectedSuffix: ".aws/config",
		},
		{
			name:          "returns AWS_CONFIG_FILE when set",
			homeDir:       "/home/testuser",
			awsConfigFile: "/custom/path/to/config",
			setConfigFile: true,
			exactMatch:    "/custom/path/to/config",
		},
		{
			name:          "AWS_CONFIG_FILE takes precedence over default",
			homeDir:       "/home/testuser",
			awsConfigFile: "/override/aws/config",
			setConfigFile: true,
			exactMatch:    "/override/aws/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("HOME", tt.homeDir)
			if tt.setConfigFile {
				_ = os.Setenv("AWS_CONFIG_FILE", tt.awsConfigFile)
			} else {
				_ = os.Unsetenv("AWS_CONFIG_FILE")
			}

			result := GetCurrentProfileFile()

			if tt.exactMatch != "" {
				assert.Equal(t, tt.exactMatch, result)
			} else if tt.expectedSuffix != "" {
				assert.True(t, filepath.IsAbs(result) || result == "", "should return absolute path or empty string")
				if result != "" {
					assert.Contains(t, result, tt.expectedSuffix)
				}
			}
		})
	}
}

// ============================================================================
// Tests for AppendIfNotExists
// ============================================================================

func TestAppendIfNotExists(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected []string
	}{
		{
			name:     "appends to empty slice",
			slice:    []string{},
			item:     "new",
			expected: []string{"new"},
		},
		{
			name:     "appends when item does not exist",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "does not append when item exists at beginning",
			slice:    []string{"existing", "b", "c"},
			item:     "existing",
			expected: []string{"existing", "b", "c"},
		},
		{
			name:     "does not append when item exists at end",
			slice:    []string{"a", "b", "existing"},
			item:     "existing",
			expected: []string{"a", "b", "existing"},
		},
		{
			name:     "does not append when item exists in middle",
			slice:    []string{"a", "existing", "c"},
			item:     "existing",
			expected: []string{"a", "existing", "c"},
		},
		{
			name:     "handles single element slice - exists",
			slice:    []string{"only"},
			item:     "only",
			expected: []string{"only"},
		},
		{
			name:     "handles single element slice - not exists",
			slice:    []string{"only"},
			item:     "new",
			expected: []string{"only", "new"},
		},
		{
			name:     "handles empty string item",
			slice:    []string{"a", "b"},
			item:     "",
			expected: []string{"a", "b", ""},
		},
		{
			name:     "does not duplicate empty string",
			slice:    []string{"a", "", "b"},
			item:     "",
			expected: []string{"a", "", "b"},
		},
		{
			name:     "handles special characters",
			slice:    []string{"profile-1", "profile_2"},
			item:     "profile-3",
			expected: []string{"profile-1", "profile_2", "profile-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendIfNotExists(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// Tests for Contains
// ============================================================================

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "empty slice returns false",
			slice:    []string{},
			item:     "anything",
			expected: false,
		},
		{
			name:     "finds item at beginning",
			slice:    []string{"first", "second", "third"},
			item:     "first",
			expected: true,
		},
		{
			name:     "finds item at end",
			slice:    []string{"first", "second", "third"},
			item:     "third",
			expected: true,
		},
		{
			name:     "finds item in middle",
			slice:    []string{"first", "second", "third"},
			item:     "second",
			expected: true,
		},
		{
			name:     "returns false when item not found",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "handles single element slice - found",
			slice:    []string{"only"},
			item:     "only",
			expected: true,
		},
		{
			name:     "handles single element slice - not found",
			slice:    []string{"only"},
			item:     "other",
			expected: false,
		},
		{
			name:     "is case sensitive",
			slice:    []string{"Dev", "Staging", "Prod"},
			item:     "dev",
			expected: false,
		},
		{
			name:     "finds empty string",
			slice:    []string{"a", "", "b"},
			item:     "",
			expected: true,
		},
		{
			name:     "handles special characters",
			slice:    []string{"profile-1", "profile_2", "profile.3"},
			item:     "profile_2",
			expected: true,
		},
		{
			name:     "partial match returns false",
			slice:    []string{"development", "staging", "production"},
			item:     "dev",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkContains(b *testing.B) {
	slice := make([]string, 100)
	for i := 0; i < 100; i++ {
		slice[i] = fmt.Sprintf("profile-%d", i)
	}

	b.Run("item at beginning", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Contains(slice, "profile-0")
		}
	})

	b.Run("item at end", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Contains(slice, "profile-99")
		}
	})

	b.Run("item not found", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Contains(slice, "nonexistent")
		}
	})
}

func BenchmarkAppendIfNotExists(b *testing.B) {
	b.Run("append to small slice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := []string{"a", "b", "c"}
			AppendIfNotExists(slice, "d")
		}
	})

	b.Run("item exists in small slice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := []string{"a", "b", "c"}
			AppendIfNotExists(slice, "b")
		}
	})
}
