package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DataDir              string            `yaml:"datadir" mapstructure:"datadir"`
	TxFlags              string            `yaml:"txflags" mapstructure:"txflags"`
	TxFlagsTemplate      map[string]string `yaml:"txflagstemplate" mapstructure:"txflagstemplate"`
	Applications         []Application     `yaml:"applications" mapstructure:"applications"`
	ApplicationStakeGoal string            `yaml:"application_stake_goal" mapstructure:"application_stake_goal"` // Target stake amount for applications
	ApplicationFundGoal  string            `yaml:"application_fund_goal" mapstructure:"application_fund_goal"`   // Target balance for funding applications
	GrpcEndpoint         string            `yaml:"grpc_endpoint" mapstructure:"grpc_endpoint"`                   // GRPC endpoint for querying balances
	RpcEndpoint          string            `yaml:"rpc_endpoint" mapstructure:"rpc_endpoint"`                     // RPC endpoint for broadcasting transactions
	GatewayURLs          map[string]string `yaml:"gateway_urls" mapstructure:"gateway_urls"`                     // Map of gateway IDs to their URLs
}

type Application struct {
	Name           string   `yaml:"name" mapstructure:"name"`
	Address        string   `yaml:"address" mapstructure:"address"`
	Mnemonic       string   `yaml:"mnemonic" mapstructure:"mnemonic"`
	ServiceIdGoal  string   `yaml:"serviceidgoal" mapstructure:"serviceidgoal"`   // Which service ID to stake on
	DelegateesGoal []string `yaml:"delegateesgoal" mapstructure:"delegateesgoal"` // TO WHICH GATEWAY(s) that app should be delegated to
}

// Example of application data returned from the API:
// {
// 	"application": {
// 	  "address": "pokt100ta9phah2dfupn25ast25zv4q3rvyva6c9ckq",
// 	  "stake": {
// 		"denom": "upokt",
// 		"amount": "99992378"
// 	  },
// 	  "service_configs": [
// 		{
// 		  "service_id": "proto-anvil"
// 		}
// 	  ],
// 	  "delegatee_gateway_addresses": [
// 		"pokt1tgfhrtpxa4afeh70fk2aj6ca4mw84xqrkfgrdl"
// 	  ],
// 	  "pending_undelegations": {},
// 	  "unstake_session_end_height": "0",
// 	  "pending_transfer": null
// 	}
//   }

func LoadConfig(configFile string) (*Config, error) {
	// If a config file is provided, use it
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// Otherwise, look for config.yml in the current directory
		viper.SetConfigName("config")
		viper.SetConfigType("yml")
		viper.AddConfigPath(".")
	}

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Create a new config with default values
	config := &Config{
		DataDir:              viper.GetString("datadir"),
		TxFlags:              viper.GetString("txflags"),
		TxFlagsTemplate:      viper.GetStringMapString("txflagstemplate"),
		Applications:         []Application{},
		ApplicationStakeGoal: viper.GetString("application_stake_goal"),
		ApplicationFundGoal:  viper.GetString("application_fund_goal"),
		GrpcEndpoint:         viper.GetString("grpc_endpoint"),
		RpcEndpoint:          viper.GetString("rpc_endpoint"),
		GatewayURLs:          make(map[string]string),
	}

	// Debug output
	fmt.Printf("Viper keys: %v\n", viper.AllKeys())
	fmt.Printf("Manual config: %+v\n", config)

	// Load gateway URLs
	if err := viper.UnmarshalKey("gateway_urls", &config.GatewayURLs); err != nil {
		fmt.Printf("Warning: failed to unmarshal gateway_urls: %v\n", err)
	}

	// Unmarshal applications
	var apps []map[string]interface{}
	if err := viper.UnmarshalKey("applications", &apps); err == nil {
		for _, appMap := range apps {
			app := Application{
				Name:           getString(appMap, "name"),
				Address:        getString(appMap, "address"),
				Mnemonic:       getString(appMap, "mnemonic"),
				ServiceIdGoal:  getString(appMap, "serviceidgoal"),
				DelegateesGoal: getStringSlice(appMap, "delegateesgoal"),
			}
			config.Applications = append(config.Applications, app)
		}
	}

	// Process config (expand templates, etc.)
	if config.TxFlagsTemplate != nil {
		config.TxFlags = expandTxFlagsTemplate(config.TxFlagsTemplate)
	}

	// Set default data directory if not specified
	if config.DataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		config.DataDir = filepath.Join(homeDir, ".poktroll")
	}

	return config, nil
}

// Helper functions to safely get values from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if val, ok := m[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

func expandTxFlagsTemplate(template map[string]string) string {
	var flags []string
	for k, v := range template {
		flags = append(flags, fmt.Sprintf("--%s=%s", k, v))
	}
	return strings.Join(flags, " ")
}

// ParseAmount parses a string amount like "1000000upokt" into an integer
func ParseAmount(amount string) (int64, error) {
	// Remove the denomination suffix
	numStr := strings.TrimSuffix(amount, "upokt")

	// Parse the numeric part
	var result int64
	_, err := fmt.Sscanf(numStr, "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("failed to parse amount: %s", amount)
	}

	return result, nil
}
