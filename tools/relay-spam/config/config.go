package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Home                string              `mapstructure:"home"`
	TxFlags             string              `mapstructure:"tx_flags"`
	TxFlagsTemplate     map[string]string   `mapstructure:"tx_flags_template"`
	Applications        []Application       `mapstructure:"applications"`
	ApplicationDefaults ApplicationDefaults `mapstructure:"application_defaults"`
}

type Application struct {
	Name           string   `mapstructure:"name"`
	Address        string   `mapstructure:"address"`
	Mnemonic       string   `mapstructure:"mnemonic"`
	StakeGoal      int      `mapstructure:"stake_goal"`      // How much stake to maintain. Top off to this goal if less than this
	ServiceIdGoal  string   `mapstructure:"service_id_goal"` // Which service ID to stake on
	DelegateesGoal []string `mapstructure:"delegatees_goal"` // TO WHICH GATEWAY(s) that app should be delegated to
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

type ApplicationDefaults struct {
	Stake     string   `mapstructure:"stake"`
	ServiceID string   `mapstructure:"service_id"`
	Gateways  []string `mapstructure:"gateways"`
}

func LoadConfig() (*Config, error) {
	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	// Process config (expand templates, etc.)
	if config.TxFlagsTemplate != nil {
		config.TxFlags = expandTxFlagsTemplate(config.TxFlagsTemplate)
	}

	// Set default home directory if not specified
	if config.Home == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		config.Home = filepath.Join(homeDir, ".poktroll")
	}

	return &config, nil
}

func expandTxFlagsTemplate(template map[string]string) string {
	var flags []string
	for k, v := range template {
		flags = append(flags, fmt.Sprintf("--%s=%s", k, v))
	}
	return strings.Join(flags, " ")
}
