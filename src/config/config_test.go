package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// Tests for InitConfig
// ============================================================================

func TestInitConfig(t *testing.T) {
	// Reset viper before each test
	setupTest := func() {
		viper.Reset()
	}

	t.Run("loads config from specified file path", func(t *testing.T) {
		setupTest()

		// Create a temporary config file
		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "test-config.yaml")
		configContent := `
pulumi_projects:
  "123456789012": "my-pulumi-bucket"
  "987654321098": "other-bucket"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)
		assert.Equal(t, configPath, viper.ConfigFileUsed())

		// Verify config was loaded correctly
		projects := viper.GetStringMapString("pulumi_projects")
		assert.Equal(t, "my-pulumi-bucket", projects["123456789012"])
	})

	t.Run("returns error for non-existent config file", func(t *testing.T) {
		setupTest()

		err := InitConfig("/nonexistent/path/config.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config")
	})

	t.Run("returns error for invalid yaml content", func(t *testing.T) {
		setupTest()

		// Create a temporary invalid config file
		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "invalid-config.yaml")
		invalidContent := `
this is not valid yaml: [
  unclosed bracket
`
		err = os.WriteFile(configPath, []byte(invalidContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config")
	})

	t.Run("loads config with all supported fields", func(t *testing.T) {
		setupTest()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "full-config.yaml")
		configContent := `
pulumi_projects:
  "111111111111": "bucket-1"
  "222222222222": "bucket-2"
  "333333333333": "bucket-3"

aws_spec:
  account: "123456789012"
  arn: "arn:aws:iam::123456789012:user/testuser"
  user_id: "AIDAEXAMPLE123"

custom_setting: "custom_value"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)

		// Verify all fields loaded
		projects := viper.GetStringMapString("pulumi_projects")
		assert.Len(t, projects, 3)
		assert.Equal(t, "bucket-1", projects["111111111111"])

		assert.Equal(t, "custom_value", viper.GetString("custom_setting"))
	})

	t.Run("loads config from default location when no path specified", func(t *testing.T) {
		setupTest()

		// Save original HOME and restore after test
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)

		// Create a temporary home directory with config
		tempHome, err := os.MkdirTemp("", "paws-home-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempHome)

		os.Setenv("HOME", tempHome)

		configPath := filepath.Join(tempHome, ".pulumi_config.yaml")
		configContent := `
pulumi_projects:
  "999999999999": "default-bucket"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig("")
		assert.NoError(t, err)

		projects := viper.GetStringMapString("pulumi_projects")
		assert.Equal(t, "default-bucket", projects["999999999999"])
	})

	t.Run("returns error when default config does not exist", func(t *testing.T) {
		setupTest()

		// Save original HOME and restore after test
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)

		// Create a temporary home directory WITHOUT config
		tempHome, err := os.MkdirTemp("", "paws-home-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempHome)

		os.Setenv("HOME", tempHome)

		err = InitConfig("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config")
	})

	t.Run("handles empty config file", func(t *testing.T) {
		setupTest()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "empty-config.yaml")
		err = os.WriteFile(configPath, []byte(""), 0644)
		require.NoError(t, err)

		// Empty yaml file should be valid
		err = InitConfig(configPath)
		assert.NoError(t, err)
	})

	t.Run("handles config with comments", func(t *testing.T) {
		setupTest()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "commented-config.yaml")
		configContent := `
# This is a comment
pulumi_projects:
  # Account for development
  "123456789012": "dev-bucket"
  # Account for production
  "987654321098": "prod-bucket"

# Another comment at the end
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)

		projects := viper.GetStringMapString("pulumi_projects")
		assert.Equal(t, "dev-bucket", projects["123456789012"])
		assert.Equal(t, "prod-bucket", projects["987654321098"])
	})

	t.Run("environment variables override config file", func(t *testing.T) {
		setupTest()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "env-config.yaml")
		configContent := `
test_value: "from_file"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Set environment variable (viper uses uppercase with underscores)
		os.Setenv("TEST_VALUE", "from_env")
		defer os.Unsetenv("TEST_VALUE")

		err = InitConfig(configPath)
		assert.NoError(t, err)

		// Environment variable should take precedence
		assert.Equal(t, "from_env", viper.GetString("test_value"))
	})
}

// ============================================================================
// Tests for AwsGetCallerIdentitySpec
// ============================================================================

func TestAwsGetCallerIdentitySpec(t *testing.T) {
	t.Run("struct has correct json tags", func(t *testing.T) {
		spec := AwsGetCallerIdentitySpec{
			Account: "123456789012",
			ARN:     "arn:aws:iam::123456789012:user/testuser",
			UserID:  "AIDAEXAMPLE123",
		}

		jsonBytes, err := json.Marshal(spec)
		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(jsonBytes, &result)
		require.NoError(t, err)

		assert.Equal(t, "123456789012", result["account"])
		assert.Equal(t, "arn:aws:iam::123456789012:user/testuser", result["arn"])
		assert.Equal(t, "AIDAEXAMPLE123", result["user_id"])
	})

	t.Run("struct has correct yaml tags", func(t *testing.T) {
		spec := AwsGetCallerIdentitySpec{
			Account: "987654321098",
			ARN:     "arn:aws:sts::987654321098:assumed-role/MyRole/session",
			UserID:  "AROAEXAMPLE456:session",
		}

		yamlBytes, err := yaml.Marshal(spec)
		require.NoError(t, err)

		var result map[string]string
		err = yaml.Unmarshal(yamlBytes, &result)
		require.NoError(t, err)

		assert.Equal(t, "987654321098", result["account"])
		assert.Equal(t, "arn:aws:sts::987654321098:assumed-role/MyRole/session", result["arn"])
		assert.Equal(t, "AROAEXAMPLE456:session", result["user_id"])
	})

	t.Run("json unmarshal works correctly", func(t *testing.T) {
		jsonInput := `{
			"account": "111222333444",
			"arn": "arn:aws:iam::111222333444:root",
			"user_id": "111222333444"
		}`

		var spec AwsGetCallerIdentitySpec
		err := json.Unmarshal([]byte(jsonInput), &spec)
		require.NoError(t, err)

		assert.Equal(t, "111222333444", spec.Account)
		assert.Equal(t, "arn:aws:iam::111222333444:root", spec.ARN)
		assert.Equal(t, "111222333444", spec.UserID)
	})

	t.Run("yaml unmarshal works correctly", func(t *testing.T) {
		yamlInput := `
account: "555666777888"
arn: "arn:aws:sts::555666777888:assumed-role/AdminRole/admin"
user_id: "AROAADMINEXAMPLE:admin"
`
		var spec AwsGetCallerIdentitySpec
		err := yaml.Unmarshal([]byte(yamlInput), &spec)
		require.NoError(t, err)

		assert.Equal(t, "555666777888", spec.Account)
		assert.Equal(t, "arn:aws:sts::555666777888:assumed-role/AdminRole/admin", spec.ARN)
		assert.Equal(t, "AROAADMINEXAMPLE:admin", spec.UserID)
	})

	t.Run("handles empty struct", func(t *testing.T) {
		spec := AwsGetCallerIdentitySpec{}

		jsonBytes, err := json.Marshal(spec)
		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(jsonBytes, &result)
		require.NoError(t, err)

		assert.Equal(t, "", result["account"])
		assert.Equal(t, "", result["arn"])
		assert.Equal(t, "", result["user_id"])
	})

	t.Run("handles partial data", func(t *testing.T) {
		jsonInput := `{"account": "123456789012"}`

		var spec AwsGetCallerIdentitySpec
		err := json.Unmarshal([]byte(jsonInput), &spec)
		require.NoError(t, err)

		assert.Equal(t, "123456789012", spec.Account)
		assert.Empty(t, spec.ARN)
		assert.Empty(t, spec.UserID)
	})

	t.Run("mapstructure tags work with viper", func(t *testing.T) {
		viper.Reset()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "mapstructure-config.yaml")
		configContent := `
aws_spec:
  account: "mapstructure_account"
  arn: "mapstructure_arn"
  user_id: "mapstructure_user_id"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		require.NoError(t, err)

		var spec AwsGetCallerIdentitySpec
		err = viper.UnmarshalKey("aws_spec", &spec)
		require.NoError(t, err)

		assert.Equal(t, "mapstructure_account", spec.Account)
		assert.Equal(t, "mapstructure_arn", spec.ARN)
		assert.Equal(t, "mapstructure_user_id", spec.UserID)
	})
}

// ============================================================================
// Tests for config file formats
// ============================================================================

func TestConfigFileFormats(t *testing.T) {
	t.Run("loads yaml with .yaml extension", func(t *testing.T) {
		viper.Reset()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "config.yaml")
		configContent := `key: "yaml_value"`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)
		assert.Equal(t, "yaml_value", viper.GetString("key"))
	})

	t.Run("loads yaml with .yml extension", func(t *testing.T) {
		viper.Reset()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "config.yml")
		configContent := `key: "yml_value"`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)
		assert.Equal(t, "yml_value", viper.GetString("key"))
	})
}

// ============================================================================
// Tests for edge cases
// ============================================================================

func TestConfigEdgeCases(t *testing.T) {
	t.Run("handles unicode in config values", func(t *testing.T) {
		viper.Reset()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "unicode-config.yaml")
		configContent := `
description: "Configuration with émojis 🚀 and ünïcödé"
bucket_name: "my-bucket-名前"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)
		assert.Equal(t, "Configuration with émojis 🚀 and ünïcödé", viper.GetString("description"))
		assert.Equal(t, "my-bucket-名前", viper.GetString("bucket_name"))
	})

	t.Run("handles deeply nested config", func(t *testing.T) {
		viper.Reset()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "nested-config.yaml")
		configContent := `
level1:
  level2:
    level3:
      level4:
        value: "deep_value"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)
		assert.Equal(t, "deep_value", viper.GetString("level1.level2.level3.level4.value"))
	})

	t.Run("handles config with arrays", func(t *testing.T) {
		viper.Reset()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "array-config.yaml")
		configContent := `
profiles:
  - dev
  - staging
  - prod
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)

		profiles := viper.GetStringSlice("profiles")
		assert.Equal(t, []string{"dev", "staging", "prod"}, profiles)
	})

	t.Run("handles numeric account IDs as strings", func(t *testing.T) {
		viper.Reset()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "numeric-config.yaml")
		// Account IDs should be quoted to preserve leading zeros
		configContent := `
pulumi_projects:
  "012345678901": "bucket-with-leading-zero"
  "123456789012": "normal-bucket"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)

		projects := viper.GetStringMapString("pulumi_projects")
		assert.Equal(t, "bucket-with-leading-zero", projects["012345678901"])
		assert.Equal(t, "normal-bucket", projects["123456789012"])
	})

	t.Run("handles special characters in bucket names", func(t *testing.T) {
		viper.Reset()

		tempDir, err := os.MkdirTemp("", "paws-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := filepath.Join(tempDir, "special-config.yaml")
		configContent := `
pulumi_projects:
  "123456789012": "my-bucket-with-dashes"
  "234567890123": "my.bucket.with.dots"
  "345678901234": "mybucket123"
`
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		err = InitConfig(configPath)
		assert.NoError(t, err)

		projects := viper.GetStringMapString("pulumi_projects")
		assert.Equal(t, "my-bucket-with-dashes", projects["123456789012"])
		assert.Equal(t, "my.bucket.with.dots", projects["234567890123"])
		assert.Equal(t, "mybucket123", projects["345678901234"])
	})
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkInitConfig(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "paws-config-bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "bench-config.yaml")
	configContent := `
pulumi_projects:
  "123456789012": "bucket-1"
  "234567890123": "bucket-2"
  "345678901234": "bucket-3"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		viper.Reset()
		_ = InitConfig(configPath)
	}
}

func BenchmarkAwsGetCallerIdentitySpecMarshal(b *testing.B) {
	spec := AwsGetCallerIdentitySpec{
		Account: "123456789012",
		ARN:     "arn:aws:sts::123456789012:assumed-role/MyRole/session",
		UserID:  "AROAEXAMPLE:session",
	}

	b.Run("json marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(spec)
		}
	})

	b.Run("yaml marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = yaml.Marshal(spec)
		}
	})
}
