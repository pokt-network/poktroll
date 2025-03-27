package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

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

		// Get parameters from flags
		numRequests := viper.GetInt("num_requests")
		concurrency := viper.GetInt("concurrency")
		rateLimit := viper.GetFloat64("rate_limit")
		mode := viper.GetString("mode")
		duration := viper.GetDuration("duration")
		distribution := viper.GetString("distribution")
		timeout := viper.GetDuration("timeout")

		// Create spammer options
		var options []relay.SpammerOption

		// Set mode
		switch mode {
		case "fixed":
			options = append(options, relay.WithMode(relay.FixedRequestsMode))
		case "time":
			options = append(options, relay.WithMode(relay.TimeBasedMode))
			options = append(options, relay.WithDuration(duration))
		case "infinite":
			options = append(options, relay.WithMode(relay.InfiniteMode))
		default:
			// Default to fixed mode
			options = append(options, relay.WithMode(relay.FixedRequestsMode))
		}

		// Set distribution strategy
		if distribution != "" {
			options = append(options, relay.WithDistributionStrategy(distribution))
		}

		// Set request timeout
		if timeout > 0 {
			options = append(options, relay.WithRequestTimeout(timeout))
		}

		// Create relay spammer
		spammer := relay.NewSpammer(cfg, numRequests, concurrency, rateLimit, options...)

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
		fmt.Printf("Starting relay spam with mode=%s, concurrency=%d, and rate limit=%.2f req/s\n",
			mode, concurrency, rateLimit)

		if mode == "fixed" {
			fmt.Printf("Will send %d requests per app-gateway pair\n", numRequests)
		} else if mode == "time" {
			fmt.Printf("Will run for %s\n", duration)
		} else {
			fmt.Printf("Will run until interrupted\n")
		}

		relayMetrics, err := spammer.Run(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Relay spam failed: %v\n", err)
			os.Exit(1)
		}

		// Print metrics
		fmt.Println("\n=== Relay Spam Results ===")
		testDuration := relayMetrics.EndTime.Sub(relayMetrics.StartTime)
		successRate := float64(relayMetrics.SuccessfulRequests) / float64(relayMetrics.TotalRequests) * 100
		requestsPerSecond := float64(relayMetrics.TotalRequests) / testDuration.Seconds()

		fmt.Printf("Total Requests:      %d\n", relayMetrics.TotalRequests)
		fmt.Printf("Successful Requests: %d (%.2f%%)\n", relayMetrics.SuccessfulRequests, successRate)
		fmt.Printf("Failed Requests:     %d (%.2f%%)\n", relayMetrics.FailedRequests, 100-successRate)
		fmt.Printf("Duration:            %.2f seconds\n", testDuration.Seconds())
		fmt.Printf("Requests Per Second: %.2f\n", requestsPerSecond)

		// Print response time metrics
		if relayMetrics.TotalRequests > 0 {
			fmt.Printf("Min Response Time:   %s\n", relayMetrics.ResponseTimeMin)
			fmt.Printf("Max Response Time:   %s\n", relayMetrics.ResponseTimeMax)
			fmt.Printf("Avg Response Time:   %s\n", relayMetrics.ResponseTimeAvg)
		}

		// Print per-application metrics
		fmt.Println("\n=== Per-Application Metrics ===")

		// Sort applications by total requests
		type appMetricsPair struct {
			address string
			metrics *relay.AppMetrics
		}

		var appMetricsList []appMetricsPair
		for addr, metrics := range relayMetrics.AppMetrics {
			if metrics.TotalRequests > 0 {
				appMetricsList = append(appMetricsList, appMetricsPair{addr, metrics})
			}
		}

		sort.Slice(appMetricsList, func(i, j int) bool {
			return appMetricsList[i].metrics.TotalRequests > appMetricsList[j].metrics.TotalRequests
		})

		for _, pair := range appMetricsList {
			metrics := pair.metrics
			appSuccessRate := float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests) * 100
			fmt.Printf("App %s:\n", pair.address)
			fmt.Printf("  Requests:      %d (%.2f%% of total)\n",
				metrics.TotalRequests,
				float64(metrics.TotalRequests)/float64(relayMetrics.TotalRequests)*100)
			fmt.Printf("  Success Rate:  %.2f%%\n", appSuccessRate)
			fmt.Printf("  Avg Response:  %s\n", metrics.ResponseTimeAvg)
		}

		// Print per-gateway metrics
		fmt.Println("\n=== Per-Gateway Metrics ===")

		// Sort gateways by total requests
		type gatewayMetricsPair struct {
			url     string
			metrics *relay.GatewayMetrics
		}

		var gatewayMetricsList []gatewayMetricsPair
		for url, metrics := range relayMetrics.GatewayMetrics {
			if metrics.TotalRequests > 0 {
				gatewayMetricsList = append(gatewayMetricsList, gatewayMetricsPair{url, metrics})
			}
		}

		sort.Slice(gatewayMetricsList, func(i, j int) bool {
			return gatewayMetricsList[i].metrics.TotalRequests > gatewayMetricsList[j].metrics.TotalRequests
		})

		for _, pair := range gatewayMetricsList {
			metrics := pair.metrics
			gatewaySuccessRate := float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests) * 100
			fmt.Printf("Gateway %s:\n", pair.url)
			fmt.Printf("  Requests:      %d (%.2f%% of total)\n",
				metrics.TotalRequests,
				float64(metrics.TotalRequests)/float64(relayMetrics.TotalRequests)*100)
			fmt.Printf("  Success Rate:  %.2f%%\n", gatewaySuccessRate)
			fmt.Printf("  Avg Response:  %s\n", metrics.ResponseTimeAvg)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Add config flag
	runCmd.Flags().String("config", "", "Path to the config file")

	// Add mode flag
	runCmd.Flags().String("mode", "fixed", "Spam mode: fixed, time, or infinite")
	viper.BindPFlag("mode", runCmd.Flags().Lookup("mode"))

	// Add duration flag
	runCmd.Flags().Duration("duration", 10*time.Minute, "Duration for time-based mode")
	viper.BindPFlag("duration", runCmd.Flags().Lookup("duration"))

	// Add distribution strategy flag
	runCmd.Flags().String("distribution", "even", "Distribution strategy: even, weighted, or random")
	viper.BindPFlag("distribution", runCmd.Flags().Lookup("distribution"))

	// Add timeout flag
	runCmd.Flags().Duration("timeout", 10*time.Second, "Request timeout")
	viper.BindPFlag("timeout", runCmd.Flags().Lookup("timeout"))
}
