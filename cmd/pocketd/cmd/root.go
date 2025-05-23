package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"

	"cosmossdk.io/client/v2/autocli"
	clientv2keyring "cosmossdk.io/client/v2/autocli/keyring"
	"cosmossdk.io/core/address"
	"cosmossdk.io/depinject"
	cosmoslog "cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/flags"
	relayercmd "github.com/pokt-network/poktroll/pkg/relayer/cmd"
)

// TODO_MAINNET: adjust chain ID to `pocket`, `pokt` or `shannon`
const DefaultChainID = "pocket"

// NewRootCmd creates a new root command for pocketd. It is called once in the main function.
func NewRootCmd() *cobra.Command {
	InitSDKConfig()

	var (
		txConfigOpts       tx.ConfigOptions
		autoCliOpts        autocli.AppOptions
		moduleBasicManager module.BasicManager
		clientCtx          client.Context
	)

	if err := depinject.Inject(
		depinject.Configs(app.AppConfig(),
			depinject.Supply(
				cosmoslog.NewNopLogger(),
			),
			depinject.Provide(
				ProvideClientContext,
				ProvideKeyring,
			),
		),
		&txConfigOpts,
		&autoCliOpts,
		&moduleBasicManager,
		&clientCtx,
	); err != nil {
		panic(err)
	}

	rootCmd := &cobra.Command{
		Use:   app.Name + "d",
		Short: "Interface with Pocket Network",
		Long: `pocketd is a binary that can be used to query, send transaction or start Pocket Network actors.

For additional documentation, see https://dev.poktroll.com/tools/user_guide/pocketd_cli
		`,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) (err error) {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			// Parse the --network flag. If set, update related flags (e.g. --chain-id, --node, --grpc-addr).
			if err = parseAndSetNetworkRelatedFlags(cmd); err != nil {
				return err
			}

			clientCtx = clientCtx.WithCmdContext(cmd.Context())
			clientCtx, err = client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			clientCtx, err = config.ReadFromClientConfig(clientCtx)
			if err != nil {
				return err
			}

			// This needs to go after ReadFromClientConfig, as that function
			// sets the RPC client needed for SIGN_MODE_TEXTUAL.
			txConfigOpts.EnabledSignModes = append(txConfigOpts.EnabledSignModes, signing.SignMode_SIGN_MODE_TEXTUAL)
			txConfigOpts.TextualCoinMetadataQueryFn = txmodule.NewGRPCCoinMetadataQueryFn(clientCtx)
			txConfigWithTextual, err := tx.NewTxConfigWithOptions(
				codec.NewProtoCodec(clientCtx.InterfaceRegistry),
				txConfigOpts,
			)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.WithTxConfig(txConfigWithTextual)

			if err = client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}

			// TODO_TECHDEBT: Investigate if the call below is duplicated intentionally
			// or if it can be deleted.
			if err = client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCMTConfig := initCometBFTConfig()

			if err = server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig); err != nil {
				return err
			}

			return nil
		},
	}

	// Since the IBC modules don't support dependency injection, we need to
	// manually register the modules on the client side.
	// This needs to be removed after IBC supports App Wiring.
	ibcModules := app.RegisterIBC(clientCtx.InterfaceRegistry)
	for name, module := range ibcModules {
		autoCliOpts.Modules[name] = module
	}
	initRootCmd(rootCmd, clientCtx.TxConfig, clientCtx.InterfaceRegistry, clientCtx.Codec, moduleBasicManager)

	if err := overwriteFlagDefaults(rootCmd, map[string]string{
		cosmosflags.FlagChainID:        DefaultChainID,
		cosmosflags.FlagKeyringBackend: "test",
	}); err != nil {
		log.Fatal(err)
	}

	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	// add relayer command
	rootCmd.AddCommand(
		relayercmd.RelayerCmd(),
	)

	rootCmd.PersistentFlags().String(flags.FlagNetwork, flags.DefaultNetwork, flags.FlagNetworkUsage)

	return rootCmd
}

// parseAndSetNetworkRelatedFlags checks if the --network flag is set (i.e. not empty-string).
// If so, set the following flags according to their hard-coded network-specific values:
// * --chain-id
// * --node
// * --grpc-addr
func parseAndSetNetworkRelatedFlags(cmd *cobra.Command) error {
	networkStr, err := cmd.Flags().GetString(flags.FlagNetwork)
	if err != nil {
		return err
	}

	switch networkStr {
	case "":
	// No network flag was provided, so we don't need to set any flags.
	case flags.LocalNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.LocalNetChainId, pocket.LocalNetRPCURL, pocket.LocalNetGRPCAddr)
	case flags.AlphaNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.AlphaTestNetChainId, pocket.AlphaTestNetRPCURL, pocket.AlphaNetGRPCAddr)
	case flags.BetaNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.BetaTestNetChainId, pocket.BetaTestNetRPCURL, pocket.BetaNetGRPCAddr)
	case flags.MainNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.MainNetChainId, pocket.MainNetRPCURL, pocket.MainNetGRPCAddr)
	default:
		return fmt.Errorf("unknown --network specified %q", networkStr)
	}

	return nil
}

// setNetworkRelatedFlags sets the following flags according to the given arguments:
// * --chain-id
// * --node
// * --grpc-addr
func setNetworkRelatedFlags(cmd *cobra.Command, chainId, nodeUrl, grpcAddr string) error {
	if err := cmd.Flags().Set(cosmosflags.FlagChainID, chainId); err != nil {
		return err
	}

	if err := cmd.Flags().Set(cosmosflags.FlagNode, nodeUrl); err != nil {
		return err
	}

	// ONLY set --grpc-addr flag if it is registered on cmd.
	if grpcFlag := cmd.Flags().Lookup(cosmosflags.FlagGRPC); grpcFlag != nil {
		if err := cmd.Flags().Set(cosmosflags.FlagGRPC, grpcAddr); err != nil {
			return err
		}
	}

	return nil
}

func overwriteFlagDefaults(c *cobra.Command, defaults map[string]string) (err error) {
	set := func(s *pflag.FlagSet, key, val string) error {
		if f := s.Lookup(key); f != nil {
			f.DefValue = val
			if err = f.Value.Set(val); err != nil {
				return err
			}
		}
		return nil
	}
	for key, val := range defaults {
		err = errors.Join(err, set(c.Flags(), key, val))
		err = errors.Join(err, set(c.PersistentFlags(), key, val))
	}
	for _, c := range c.Commands() {
		err = errors.Join(err, overwriteFlagDefaults(c, defaults))
	}
	return err
}

func ProvideClientContext(
	appCodec codec.Codec,
	interfaceRegistry codectypes.InterfaceRegistry,
	txConfig client.TxConfig,
	legacyAmino *codec.LegacyAmino,
) client.Context {
	clientCtx := client.Context{}.
		WithCodec(appCodec).
		WithInterfaceRegistry(interfaceRegistry).
		WithTxConfig(txConfig).
		WithLegacyAmino(legacyAmino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper(app.Name) // env variable prefix

	// Read the config again to overwrite the default values with the values from the config file
	clientCtx, _ = config.ReadFromClientConfig(clientCtx)

	return clientCtx
}

func ProvideKeyring(clientCtx client.Context, addressCodec address.Codec) (clientv2keyring.Keyring, error) {
	kb, err := client.NewKeyringFromBackend(clientCtx, clientCtx.Keyring.Backend())
	if err != nil {
		return nil, err
	}

	return keyring.NewAutoCLIKeyring(kb)
}
