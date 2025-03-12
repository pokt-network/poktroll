package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pokt-network/poktroll/tools/relay-spam/config"
	"github.com/pokt-network/poktroll/tools/relay-spam/relay"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run relay spam",
	Long:  `Run relay spam with the configured applications and gateways.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get config file from flag
		configFile, err := cmd.Flags().GetString("config")
		if err != nil || configFile == "" {
			configFile = "config.yml"
		}

		// Load config
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Ensure data directory exists
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
			os.Exit(1)
		}

		// Create relay spammer
		numRequests := viper.GetInt("num_requests")
		concurrency := viper.GetInt("concurrency")
		rateLimit := viper.GetFloat64("rate_limit")

		spammer := relay.NewSpammer(cfg, numRequests, concurrency, rateLimit)

		// Create context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle signals for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nReceived signal, shutting down...")
			cancel()
		}()

		// Run relay spam
		fmt.Printf("Starting relay spam with %d requests per app-gateway pair, %d concurrent workers, and rate limit of %.2f req/s\n",
			numRequests, concurrency, rateLimit)

		relayMetrics, err := spammer.Run(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Relay spam failed: %v\n", err)
			os.Exit(1)
		}

		// Print metrics
		fmt.Println("=== Relay Spam Results ===")
		duration := relayMetrics.EndTime.Sub(relayMetrics.StartTime)
		successRate := float64(relayMetrics.SuccessfulRequests) / float64(relayMetrics.TotalRequests) * 100
		requestsPerSecond := float64(relayMetrics.TotalRequests) / duration.Seconds()

		fmt.Printf("Total Requests:      %d\n", relayMetrics.TotalRequests)
		fmt.Printf("Successful Requests: %d (%.2f%%)\n", relayMetrics.SuccessfulRequests, successRate)
		fmt.Printf("Failed Requests:     %d (%.2f%%)\n", relayMetrics.FailedRequests, 100-successRate)
		fmt.Printf("Duration:            %.2f seconds\n", duration.Seconds())
		fmt.Printf("Requests Per Second: %.2f\n", requestsPerSecond)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Add config flag
	runCmd.Flags().String("config", "", "Path to the config file")
}
