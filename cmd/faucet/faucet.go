package faucet

import (
	"context"
	"net"
	"net/http"

	"cosmossdk.io/math"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
)

var (

	// TODO_IN_THIS_COMMIT: make this configurable; e.g. env var...
	faucetKeyName = "faucet"
	fundCoins     = types.NewCoins(types.NewCoin(pocket.DenomuPOKT, math.NewInt(1)))
	faucetAddress types.AccAddress

	txClient client.TxClient
)

// TODO_IN_THIS_COMMIT: split server and client CLI commands...

func FaucetCmd() *cobra.Command {
	faucetCmd := &cobra.Command{
		Use:   "faucet",
		Short: "Pocket Network Faucet",
		Long:  `Pocket Network Faucet`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := logger.PreRunESetup(cmd, args); err != nil {
				return err
			}

			if err := preRunFaucet(cmd, args); err != nil {
				return err
			}
			return nil
		},
		RunE: runFaucet,
	}

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(faucetCmd)

	faucetCmd.PersistentFlags().StringVar(&logger.LogLevel, flags.FlagLogLevel, "info", flags.FlagLogLevelUsage)
	faucetCmd.PersistentFlags().StringVar(&logger.LogOutput, flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	return faucetCmd
}

func preRunFaucet(cmd *cobra.Command, _ []string) error {
	// Conventionally derive a cosmos-sdk client context from the cobra command
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	// Load the faucet key, by name, from the keyring.
	// NOTE: DOES respect the --keyring-backend and --home flags.
	record, err := clientCtx.Keyring.Key(faucetKeyName)
	if err != nil {
		return err
	}

	faucetAddress, err = record.GetAddress()
	if err != nil {
		return err
	}

	// Construct a tx client.
	signingKeyOpt := tx.WithSigningKeyName("faucet")
	if err = cosmosclient.SetCmdClientContext(cmd, clientCtx); err != nil {
		return err
	}

	txClient, err = flags.GetTxClientFromFlags(cmd.Context(), cmd, signingKeyOpt)
	return err
}

func runFaucet(cmd *cobra.Command, args []string) error {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
	}()

	logger.Logger.Info().Msgf("Listening on %s", listener.Addr())

	return http.Serve(listener, NewFaucetServer())
}

func NewFaucetServer() *http.ServeMux {
	httpSrv := http.NewServeMux()

	httpSrv.HandleFunc("/mact/", handleMactRequest)

	return httpSrv
}

func handleMactRequest(resWriter http.ResponseWriter, req *http.Request) {
	recipientAddressStr := req.URL.Path[len("/mact/"):]

	recipientAddress, err := types.AccAddressFromBech32(recipientAddressStr)
	if err != nil {
		logger.Logger.Error().Err(err).Send()

		resWriter.WriteHeader(http.StatusBadRequest)
		if _, err = resWriter.Write([]byte(err.Error())); err != nil {
			logger.Logger.Error().Err(err).Send()
		}
		return
	}

	if err = sendMact(req.Context(), recipientAddress); err != nil {
		logger.Logger.Error().Err(err).Send()

		resWriter.WriteHeader(http.StatusBadRequest)
		if _, err = resWriter.Write([]byte(err.Error())); err != nil {
			logger.Logger.Error().Err(err).Send()
		}
		return
	}

	if _, err = resWriter.Write([]byte{}); err != nil {
		logger.Logger.Error().Err(err).Send()
		return
	}
}

func sendMact(ctx context.Context, recipientAddress types.AccAddress) error {
	sendMsg := bank.NewMsgSend(faucetAddress, recipientAddress, fundCoins)
	txResponse, eitherErr := txClient.SignAndBroadcast(ctx, sendMsg)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	logger.Logger.Debug().Str("tx_hash", txResponse.TxHash).Send()

	go func(txResponse *types.TxResponse) {
		select {
		case <-ctx.Done():
			return
		case err = <-errCh:
			// TODO_IN_THIS_COMMIT: log the error
			if err != nil {
				logger.Logger.Error().Err(err).Send()
			} else {
				// TODO_INVESTIGATE: why doesn't execution reach here?
				// The errCh SHOULD close after the tx errors, times out, or is committed.
				logger.Logger.Debug().
					Str("tx_hash", txResponse.TxHash).
					Msg("transaction succeeded")
			}
			return
		}
	}(txResponse)

	return nil
}
