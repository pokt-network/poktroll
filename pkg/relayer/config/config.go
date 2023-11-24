package config

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	KeyRelayMinerConfigPath = "relay_miner_config_path"
	KeyQueryNodeUrl         = "query_node_url"
	KeyNetworkNodeUrl       = "network_node_url"
	KeySigningKeyName       = "signing_key_name"
	KeySmtStorePath         = "smt_store_path"
)

func ReadConfig(configPath string) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		//// TODO_IN_THIS_COMMIT: what exactly does this do?
		//viper.SetConfigName("config")
		//viper.AddConfigPath(".")

		// TODO_IN_THIS_COMMIT: establish a default...
		panic("config path is required")
	}

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}
}
