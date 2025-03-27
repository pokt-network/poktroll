package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	supplierconfig "github.com/pokt-network/poktroll/x/supplier/config"
	"github.com/spf13/viper"

	"github.com/pokt-network/poktroll/app/volatile"
)

type Config struct {
	DataDir              string            `yaml:"datadir" mapstructure:"datadir"`
	TxFlags              string            `yaml:"txflags" mapstructure:"txflags"`
	TxFlagsTemplate      map[string]string `yaml:"txflagstemplate" mapstructure:"txflagstemplate"`
	Applications         []Application     `yaml:"applications" mapstructure:"applications"`
	Services             []Service         `yaml:"services" mapstructure:"services"`
	Suppliers            []Supplier        `yaml:"suppliers" mapstructure:"suppliers"`
	ApplicationStakeGoal string            `yaml:"application_stake_goal" mapstructure:"application_stake_goal"` // Target stake amount for applications
	SupplierStakeGoal    string            `yaml:"supplier_stake_goal" mapstructure:"supplier_stake_goal"`       // Target stake amount for suppliers
	ServiceStakeGoal     string            `yaml:"service_stake_goal" mapstructure:"service_stake_goal"`         // Target stake amount for services
	ApplicationFundGoal  string            `yaml:"application_fund_goal" mapstructure:"application_fund_goal"`   // Target balance for funding applications
	ServiceFundGoal      string            `yaml:"service_fund_goal" mapstructure:"service_fund_goal"`           // Target balance for funding services
	GrpcEndpoint         string            `yaml:"grpc_endpoint" mapstructure:"grpc_endpoint"`                   // GRPC endpoint for querying balances
	RpcEndpoint          string            `yaml:"rpc_endpoint" mapstructure:"rpc_endpoint"`                     // RPC endpoint for broadcasting transactions
	GatewayURLs          map[string]string `yaml:"gateway_urls" mapstructure:"gateway_urls"`                     // Map of gateway IDs to their URLs
	ChainID              string            `yaml:"chain_id" mapstructure:"chain_id"`                             // Chain ID for transactions
	GasPrice             string            `yaml:"gas_price" mapstructure:"gas_price"`                           // Gas price for transactions (e.g. "0.01upokt")
}

type Application struct {
	Name           string   `yaml:"name" mapstructure:"name"`
	Address        string   `yaml:"address" mapstructure:"address"`
	Mnemonic       string   `yaml:"mnemonic" mapstructure:"mnemonic"`
	ServiceIdGoal  string   `yaml:"serviceidgoal" mapstructure:"serviceidgoal"`   // Which service ID to stake on
	DelegateesGoal []string `yaml:"delegateesgoal" mapstructure:"delegateesgoal"` // TO WHICH GATEWAY(s) that app should be delegated to
}

type Service struct {
	Name        string `yaml:"name" mapstructure:"name"`
	ServiceName string `yaml:"service_name" mapstructure:"service_name"`
	Address     string `yaml:"address" mapstructure:"address"`
	Mnemonic    string `yaml:"mnemonic" mapstructure:"mnemonic"`
	ServiceId   string `yaml:"service_id" mapstructure:"service_id"`
}

type Supplier struct {
	Name         string                         `yaml:"name" mapstructure:"name"`
	Address      string                         `yaml:"address" mapstructure:"address"`
	Mnemonic     string                         `yaml:"mnemonic" mapstructure:"mnemonic"`
	OwnerAddress string                         `yaml:"owner_address" mapstructure:"owner_address"`
	StakeConfig  supplierconfig.YAMLStakeConfig `yaml:"stake_config" mapstructure:"stake_config"`
}

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
		Services:             []Service{},
		Suppliers:            []Supplier{},
		ApplicationStakeGoal: viper.GetString("application_stake_goal"),
		ApplicationFundGoal:  viper.GetString("application_fund_goal"),
		ServiceFundGoal:      viper.GetString("service_fund_goal"),
		GrpcEndpoint:         viper.GetString("grpc_endpoint"),
		RpcEndpoint:          viper.GetString("rpc_endpoint"),
		ChainID:              viper.GetString("chain_id"),
		GasPrice:             viper.GetString("gas_price"),
		GatewayURLs:          make(map[string]string),
	}

	// Set default chain ID if not specified
	if config.ChainID == "" {
		// Check if it's in the TxFlagsTemplate
		if chainID, ok := config.TxFlagsTemplate["chain-id"]; ok {
			config.ChainID = chainID
		} else {
			// Default to "poktroll" if not specified anywhere
			config.ChainID = "poktroll"
		}
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

	// Unmarshal services
	var services []map[string]interface{}
	if err := viper.UnmarshalKey("services", &services); err == nil {
		for _, serviceMap := range services {
			service := Service{
				Name:      getString(serviceMap, "name"),
				Address:   getString(serviceMap, "address"),
				Mnemonic:  getString(serviceMap, "mnemonic"),
				ServiceId: getString(serviceMap, "service_id"),
			}
			config.Services = append(config.Services, service)
		}
	}

	// Unmarshal suppliers
	var suppliers []map[string]interface{}
	if err := viper.UnmarshalKey("suppliers", &suppliers); err == nil {
		for _, supplierMap := range suppliers {
			supplier := Supplier{
				Name:         getString(supplierMap, "name"),
				Address:      getString(supplierMap, "address"),
				Mnemonic:     getString(supplierMap, "mnemonic"),
				OwnerAddress: getString(supplierMap, "owner_address"),
				StakeConfig:  supplierconfig.YAMLStakeConfig{},
			}
			config.Suppliers = append(config.Suppliers, supplier)
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

// ParseGasPrice parses a gas price string like "0.01upokt" into sdk.DecCoins
// If the input is empty, it returns a default gas price of 0.01upokt
func ParseGasPrice(gasPrice string) (sdk.DecCoins, error) {
	if gasPrice == "" {
		// Default to 0.01upokt if not specified
		return sdk.NewDecCoins(sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 4))), nil
	}

	// Try to parse the gas price
	decCoins, err := sdk.ParseDecCoins(gasPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gas price '%s': %w", gasPrice, err)
	}

	return decCoins, nil
}
