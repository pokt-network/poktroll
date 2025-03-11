package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

// Default batch size for multi-send commands
const defaultBatchSize = 1000

// fundCmd represents the fund command
var fundCmd = &cobra.Command{
	Use:   "fund",
	Short: "Generate funding commands",
	Long:  `Generate commands to fund accounts from the faucet using bank send, only funding the difference needed to reach the target balance.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Ensure data directory exists
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
			os.Exit(1)
		}

		// Get batch size from flag or use default
		batchSize, err := cmd.Flags().GetInt("batch-size")
		if err != nil {
			batchSize = defaultBatchSize
		}

		// Validate required config settings
		if cfg.GrpcEndpoint == "" {
			fmt.Fprintf(os.Stderr, "GRPC endpoint is required for balance checking\n")
			os.Exit(1)
		}

		if cfg.ApplicationFundGoal == "" {
			fmt.Fprintf(os.Stderr, "ApplicationFundGoal is required\n")
			os.Exit(1)
		}

		// Generate funding commands only for accounts that need it
		commands, err := generateSmartFundingCommands(cfg, batchSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate smart funding commands: %v\n", err)
			os.Exit(1)
		}

		if len(commands) == 0 {
			fmt.Println("No accounts need funding. All balances are at or above the target.")
		} else {
			for _, cmd := range commands {
				fmt.Println(cmd)
			}
		}
	},
}

// generateSmartFundingCommands generates bank send commands only for accounts that need funding
func generateSmartFundingCommands(cfg *config.Config, batchSize int) ([]string, error) {
	var commands []string
	var addressesToFund []string
	var fundingAmounts []string

	// Parse the target fund goal
	targetFund, err := parseAmount(cfg.ApplicationFundGoal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ApplicationFundGoal: %w", err)
	}

	// Connect to the GRPC endpoint
	conn, err := grpc.Dial(cfg.GrpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GRPC endpoint: %w", err)
	}
	defer conn.Close()

	// Create a bank query client
	bankQueryClient := banktypes.NewQueryClient(conn)

	// Check each application's balance
	for _, app := range cfg.Applications {
		balance, err := getBalance(context.Background(), bankQueryClient, app.Address)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to get balance for %s: %v\n", app.Address, err)
			// Add to funding list with full amount if we can't check balance
			addressesToFund = append(addressesToFund, app.Address)
			fundingAmounts = append(fundingAmounts, cfg.ApplicationFundGoal)
			continue
		}

		// If balance is less than target, add to funding list
		if balance.Amount.LT(targetFund) {
			// Calculate the amount needed to reach the target
			amountNeeded := targetFund.Sub(balance.Amount)
			addressesToFund = append(addressesToFund, app.Address)
			fundingAmounts = append(fundingAmounts, fmt.Sprintf("%s%s", amountNeeded.String(), volatile.DenomuPOKT))
			fmt.Fprintf(os.Stderr, "Account %s needs funding. Current balance: %s, Target: %s, Funding: %s\n",
				app.Address, balance.Amount.String(), targetFund.String(), amountNeeded.String())
		} else {
			fmt.Fprintf(os.Stderr, "Account %s has sufficient balance. Current: %s, Target: %s\n",
				app.Address, balance.Amount.String(), targetFund.String())
		}
	}

	// If no addresses need funding, return empty list
	if len(addressesToFund) == 0 {
		return commands, nil
	}

	// Process addresses in batches
	for i := 0; i < len(addressesToFund); i += batchSize {
		end := i + batchSize
		if end > len(addressesToFund) {
			end = len(addressesToFund)
		}

		batchAddresses := addressesToFund[i:end]
		batchAmounts := fundingAmounts[i:end]

		// Create individual send commands for each address
		for j, addr := range batchAddresses {
			cmd := fmt.Sprintf("poktrolld tx bank send faucet %s %s %s",
				addr, batchAmounts[j], cfg.TxFlags)
			commands = append(commands, cmd)
		}
	}

	return commands, nil
}

// getBalance queries the balance of an address
func getBalance(ctx context.Context, bankQueryClient banktypes.QueryClient, address string) (*sdk.Coin, error) {
	req := &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   volatile.DenomuPOKT,
	}

	res, err := bankQueryClient.Balance(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Balance, nil
}

// parseAmount parses a string amount like "1000000upokt" into an sdk.Int
func parseAmount(amount string) (math.Int, error) {
	// Remove the denomination suffix
	numStr := strings.TrimSuffix(amount, volatile.DenomuPOKT)

	// Parse the numeric part
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return math.ZeroInt(), err
	}

	return math.NewInt(num), nil
}

func init() {
	rootCmd.AddCommand(fundCmd)

	// Add batch-size flag
	fundCmd.Flags().Int("batch-size", defaultBatchSize, "Number of addresses to include in each batch")
}
