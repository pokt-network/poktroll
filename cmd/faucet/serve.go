package faucet

import (
	"context"
	"strings"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/faucet"
)

var faucetConfigPath string

func ServeCmd() *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start a Pocket Network faucet server.",
		Long: `Start a Pocket Network faucet server.

The faucet server is a REST API that allows users to request tokens of a given denom be sent to a recipient address.

For more information, see: https://dev.poktroll.com/operate/faucet
// TODO_UP_NEXT(@bryanchriswhite): update docs URL once known.`,
		Example: `pocketd faucet serve --listen-address 0.0.0.0:8080 --config ./faucet_config.yaml # Using a config file

# Using environment variables:
FAUCET_LISTEN_ADDRESS=0.0.0.0:8080 \
FAUCET_SUPPORTED_SEND_TOKENS_="100upokt,1mact" \
FAUCET_SIGNING_KEY_NAME=faucet \
pocketd faucet serve`,
		PreRunE: preRunServe,
		RunE:    runServe,
	}

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(serveCmd)

	serveCmd.Flags().StringVar(&faucetConfigPath, flags.FaucetConfigPath, flags.DefaultFaucetConfigPath, flags.FaucetConfigPathUsage)

	return serveCmd
}

// preRunServe performs the following setup steps:
// - retrieves the cosmos-sdk client context from the cobra command
// - unmarshals viper config values into a new FaucetConfig struct
// - constructs a tx client for use in the faucet
// - constructs a bank query client for use in the faucet
func preRunServe(cmd *cobra.Command, _ []string) (err error) {
	// Conventionally derive a cosmos-sdk client context from the cobra command
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	faucetCfg, err = parseFaucetConfigFromViper(clientCtx)
	if err != nil {
		return err
	}

	// Construct a tx client.
	var txClientOpts []client.TxClientOption
	signingKeyOpt := tx.WithSigningKeyName(faucetCfg.SigningKeyName)
	if err = cosmosclient.SetCmdClientContext(cmd, clientCtx); err != nil {
		return err
	}
	txClientOpts = append(txClientOpts, signingKeyOpt)

	unorderedOpt := tx.WithUnordered()
	txClientOpts = append(txClientOpts, unorderedOpt)

	// Construct the tx client.
	txClient, err = flags.GetTxClientFromFlags(cmd.Context(), cmd, txClientOpts...)
	if err != nil {
		return err
	}

	// Construct the account query client and dependencies.
	bankQueryClient = banktypes.NewQueryClient(clientCtx)

	return nil
}

// runServe starts the faucet server.
func runServe(cmd *cobra.Command, _ []string) error {
	logger.Logger.Info().
		Str("signing_key_name", faucetCfg.SigningKeyName).
		Str("send_coins", strings.Join(faucetCfg.SupportedSendCoins, ",")).
		Msgf("Listening on %s", faucetCfg.ListenAddress)

	cmdContext, cmdCancel := context.WithCancel(cmd.Context())
	cmd.SetContext(cmdContext)

	signals.GoOnExitSignal(func() {
		logger.Logger.Info().Msg("Shutting down...")
		cmdCancel()
	})

	faucetSrv, err := faucet.NewFaucetServer(
		cmdContext,
		faucet.WithConfig(faucetCfg),
		faucet.WithTxClient(txClient),
		faucet.WithBankQueryClient(bankQueryClient),
	)
	if err != nil {
		return err
	}

	return faucetSrv.Serve(cmdContext)
}

// parseFaucetConfigFromViper performs the following steps:
// - unmarshal the current viper config values into a new FaucetConfig struct
// - validate the resulting faucet config
// - load the faucet config's signing key (from the keyring)
func parseFaucetConfigFromViper(clientCtx cosmosclient.Context) (*faucet.Config, error) {
	config := new(faucet.Config)
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	// Ensure the faucet config is valid.
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Ensure the signing key and related config fields are ready to be used.
	if err := config.LoadSigningKey(clientCtx); err != nil {
		return nil, err
	}

	return config, nil
}
