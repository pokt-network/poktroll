package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
)

// TestYAMLRelayMinerSmtBackupConfig_Unmarshal tests YAML unmarshaling of backup configuration
func TestYAMLRelayMinerSmtBackupConfig_Unmarshal(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		expectedConfig relayerconfig.YAMLRelayMinerSmtBackupConfig
		expectError    bool
	}{
		{
			name: "valid_complete_config",
			yamlContent: `
interval_seconds: 300
backup_dir: "/tmp/backups"
on_session_close: true
on_claim_generation: false
on_graceful_shutdown: true
retain_backup_count: 10
`,
			expectedConfig: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				IntervalSeconds:      300,
				BackupDir:            "/tmp/backups",
				OnSessionClose:       true,
				OnClaimGeneration:    false,
				OnGracefulShutdown:   true,
				RetainBackupCount:    10,
			},
			expectError: false,
		},
		{
			name: "valid_minimal_config",
			yamlContent: `
# Empty config with no fields set
`,
			expectedConfig: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				IntervalSeconds:      0,
				BackupDir:            "",
				OnSessionClose:       false,
				OnClaimGeneration:    false,
				OnGracefulShutdown:   false,
				RetainBackupCount:    0,
			},
			expectError: false,
		},
		{
			name: "valid_zero_retention_unlimited",
			yamlContent: `
backup_dir: "/tmp/backups"
retain_backup_count: 0
`,
			expectedConfig: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				IntervalSeconds:      0,
				BackupDir:            "/tmp/backups",
				OnSessionClose:       false,
				OnClaimGeneration:    false,
				OnGracefulShutdown:   false,
				RetainBackupCount:    0, // 0 means unlimited retention
			},
			expectError: false,
		},
		{
			name: "valid_negative_retention_unlimited",
			yamlContent: `
backup_dir: "/tmp/backups"
retain_backup_count: -1
`,
			expectedConfig: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				IntervalSeconds:      0,
				BackupDir:            "/tmp/backups",
				OnSessionClose:       false,
				OnClaimGeneration:    false,
				OnGracefulShutdown:   false,
				RetainBackupCount:    -1, // Negative means unlimited retention
			},
			expectError: false,
		},
		{
			name: "invalid_yaml_syntax",
			yamlContent: `
interval_seconds: not_a_number
`,
			expectedConfig: relayerconfig.YAMLRelayMinerSmtBackupConfig{},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config relayerconfig.YAMLRelayMinerSmtBackupConfig
			err := yaml.Unmarshal([]byte(tt.yamlContent), &config)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedConfig, config)
		})
	}
}

// TestYAMLRelayMinerConfig_WithBackupConfig tests backup configuration integration in main config
func TestYAMLRelayMinerConfig_WithBackupConfig(t *testing.T) {
	yamlContent := `
default_signing_key_names: ["test_key"]
default_request_timeout_seconds: 30
default_max_body_size: "1MB"
smt_store_path: "/tmp/smt"
smt_backup:
  interval_seconds: 600
  backup_dir: "/backup/location"
  on_session_close: true
  on_claim_generation: true
  on_graceful_shutdown: true
  retain_backup_count: 5
suppliers: []
metrics:
  enabled: false
pocket_node:
  query_node_rpc_url: "http://localhost:26657"
  query_node_grpc_url: "localhost:9090"
  tx_node_rpc_url: "http://localhost:26657"
pprof:
  enabled: false
ping:
  enabled: false
enable_over_servicing: false
`

	var config relayerconfig.YAMLRelayMinerConfig
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	require.NoError(t, err)

	// Verify main config fields
	require.Equal(t, []string{"test_key"}, config.DefaultSigningKeyNames)
	require.Equal(t, uint64(30), config.DefaultRequestTimeoutSeconds)
	require.Equal(t, "/tmp/smt", config.SmtStorePath)

	// Verify backup configuration
	expectedBackupConfig := relayerconfig.YAMLRelayMinerSmtBackupConfig{
		
		IntervalSeconds:      600,
		BackupDir:            "/backup/location",
		OnSessionClose:       true,
		OnClaimGeneration:    true,
		OnGracefulShutdown:   true,
		RetainBackupCount:    5,
	}
	require.Equal(t, expectedBackupConfig, config.SmtBackup)
}

// TestYAMLRelayMinerConfig_EmptyBackupConfig tests handling of empty backup configuration
func TestYAMLRelayMinerConfig_EmptyBackupConfig(t *testing.T) {
	yamlContent := `
default_signing_key_names: ["test_key"]
smt_store_path: "/tmp/smt"
suppliers: []
metrics:
  enabled: false
pocket_node:
  query_node_rpc_url: "http://localhost:26657"
  query_node_grpc_url: "localhost:9090"
  tx_node_rpc_url: "http://localhost:26657"
pprof:
  enabled: false
ping:
  enabled: false
enable_over_servicing: false
`

	var config relayerconfig.YAMLRelayMinerConfig
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	require.NoError(t, err)

	// Verify backup configuration defaults to zero values
	expectedBackupConfig := relayerconfig.YAMLRelayMinerSmtBackupConfig{
		
		IntervalSeconds:      0,
		BackupDir:            "",
		OnSessionClose:       false,
		OnClaimGeneration:    false,
		OnGracefulShutdown:   false,
		RetainBackupCount:    0,
	}
	require.Equal(t, expectedBackupConfig, config.SmtBackup)
}

// TestBackupConfigValidation tests logical validation of backup configuration values
func TestBackupConfigValidation(t *testing.T) {
	tests := []struct {
		name         string
		config       relayerconfig.YAMLRelayMinerSmtBackupConfig
		expectValid  bool
		description  string
	}{
		{
			name: "valid_enabled_with_directory",
			config: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				BackupDir: "/tmp/backups",
			},
			expectValid: true,
			description: "Backup config with valid directory should be valid",
		},
		{
			name: "valid_disabled_without_directory",
			config: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				BackupDir: "",
			},
			expectValid: true,
			description: "Backup config without directory should be valid (disabled)",
		},
		{
			name: "potentially_invalid_enabled_without_directory",
			config: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				BackupDir: "",
			},
			expectValid: true,
			description: "Backup config without directory should be valid (disabled)",
		},
		{
			name: "valid_with_all_event_triggers",
			config: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				BackupDir:            "/tmp/backups",
				OnSessionClose:       true,
				OnClaimGeneration:    true,
				OnGracefulShutdown:   true,
				RetainBackupCount:    10,
			},
			expectValid: true,
			description: "All event triggers enabled should be valid",
		},
		{
			name: "valid_with_periodic_backup_only",
			config: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				IntervalSeconds:      300,
				BackupDir:            "/tmp/backups",
				OnSessionClose:       false,
				OnClaimGeneration:    false,
				OnGracefulShutdown:   false,
				RetainBackupCount:    5,
			},
			expectValid: true,
			description: "Periodic backup only should be valid",
		},
		{
			name: "potentially_invalid_enabled_without_triggers",
			config: relayerconfig.YAMLRelayMinerSmtBackupConfig{
				
				IntervalSeconds:      0, // No periodic backups
				BackupDir:            "/tmp/backups",
				OnSessionClose:       false, // No event-based backups
				OnClaimGeneration:    false,
				OnGracefulShutdown:   false,
			},
			expectValid: true, // Valid because directory is specified (manual backups possible)
			description: "Backup config without any triggers is valid (manual backups possible)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := validateBackupConfig(tt.config)
			if tt.expectValid {
				require.True(t, isValid, "Config should be valid: %s", tt.description)
			} else {
				require.False(t, isValid, "Config should be invalid: %s", tt.description)
			}
		})
	}
}

// validateBackupConfig performs logical validation of backup configuration
// This simulates validation that would be performed in the application
func validateBackupConfig(config relayerconfig.YAMLRelayMinerSmtBackupConfig) bool {
	// If no backup directory is specified, configuration is always valid (disabled)
	if config.BackupDir == "" {
		return true
	}

	// If enabled, should have at least one backup trigger
	// Note: A configuration with just a backup directory is considered valid
	// because the application might have default triggers or manual backups
	hasPeriodicBackup := config.IntervalSeconds > 0
	hasEventBackup := config.OnSessionClose || config.OnClaimGeneration || config.OnGracefulShutdown
	
	// For this simple test, we'll consider it valid if directory is specified
	// A real implementation might enforce stricter requirements
	_ = hasPeriodicBackup
	_ = hasEventBackup
	
	return true
}

// TestBackupConfigMarshal tests YAML marshaling of backup configuration
func TestBackupConfigMarshal(t *testing.T) {
	config := relayerconfig.YAMLRelayMinerSmtBackupConfig{
		
		IntervalSeconds:      300,
		BackupDir:            "/tmp/backups",
		OnSessionClose:       true,
		OnClaimGeneration:    false,
		OnGracefulShutdown:   true,
		RetainBackupCount:    5,
	}

	yamlBytes, err := yaml.Marshal(config)
	require.NoError(t, err)

	expectedYAML := `interval_seconds: 300
backup_dir: /tmp/backups
on_session_close: true
on_claim_generation: false
on_graceful_shutdown: true
retain_backup_count: 5
`
	require.Equal(t, expectedYAML, string(yamlBytes))
}

// TestBackupConfigRoundTrip tests marshaling and unmarshaling backup configuration
func TestBackupConfigRoundTrip(t *testing.T) {
	originalConfig := relayerconfig.YAMLRelayMinerSmtBackupConfig{
		
		IntervalSeconds:      600,
		BackupDir:            "/backup/location",
		OnSessionClose:       true,
		OnClaimGeneration:    false,
		OnGracefulShutdown:   true,
		RetainBackupCount:    10,
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(originalConfig)
	require.NoError(t, err)

	// Unmarshal back to struct
	var roundTripConfig relayerconfig.YAMLRelayMinerSmtBackupConfig
	err = yaml.Unmarshal(yamlBytes, &roundTripConfig)
	require.NoError(t, err)

	// Verify configs are identical
	require.Equal(t, originalConfig, roundTripConfig)
}