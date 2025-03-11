package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile     string
	numRequests    int
	concurrency    int
	numAccounts    int
	rateLimit      float64
	keyringBackend string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "relay-spam",
	Short: "A tool for stress testing Pocket Network with relay requests",
	Long:  `Relay Spam is a comprehensive tool for testing Pocket Network's relay capabilities by generating high volumes of relay requests from multiple accounts.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yml", "Config file")
	rootCmd.PersistentFlags().IntVarP(&numRequests, "num-requests", "n", 10, "Number of requests per application-gateway pair")
	rootCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "p", 10, "Concurrent requests")
	rootCmd.PersistentFlags().IntVarP(&numAccounts, "num-accounts", "a", 10, "Number of accounts to create")
	rootCmd.PersistentFlags().Float64VarP(&rateLimit, "rate-limit", "r", 0, "Rate limit in requests per second (0 for no limit)")
	rootCmd.PersistentFlags().StringVar(&keyringBackend, "keyring-backend", "test", "Keyring backend to use (os, file, test, inmemory)")

	viper.BindPFlag("num_requests", rootCmd.PersistentFlags().Lookup("num-requests"))
	viper.BindPFlag("concurrency", rootCmd.PersistentFlags().Lookup("concurrency"))
	viper.BindPFlag("num_accounts", rootCmd.PersistentFlags().Lookup("num-accounts"))
	viper.BindPFlag("rate_limit", rootCmd.PersistentFlags().Lookup("rate-limit"))
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
