package cmd

import (
	"errors"
	"log"
	"os"

	"cosmossdk.io/client/v2/autocli"
	clientv2keyring "cosmossdk.io/client/v2/autocli/keyring"
	"cosmossdk.io/core/address"
	"cosmossdk.io/depinject"
	cosmoslog "cosmossdk.io/log"
	cosmostypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
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
	flags2 "github.com/pokt-network/poktroll/cmd/flags"
	relayercmd "github.com/pokt-network/poktroll/pkg/relayer/cmd"
	"github.com/pokt-network/poktroll/pkg/store"
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

	// TODO_IN_THIS_COMMIT: godoc...
	cleanupFns := make([]func() error, 0)

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

			if err := server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig); err != nil {
				return err
			}

			// TODO_IN_THIS_COMMIT: godoc & consider extracting...
			offchainMultiStorePath, err := cmd.Flags().GetString(flags2.FlagOffchainMultiStorePath)
			if err != nil {
				return err
			}

			// TODO_IN_THIS_COMMIT: extract to somewhere...
			storeTypesByStoreKey := map[cosmostypes.StoreKey]cosmostypes.StoreType{
				// TODO_IN_THIS_COMMIT: extract to a const.
				cosmostypes.NewKVStoreKey("account_sequence_cache"): cosmostypes.StoreTypeDB,
			}

			offchainMultiStore, closeDB, err := store.NewMultiStore(offchainMultiStorePath, storeTypesByStoreKey)
			if err != nil {
				return err
			}
			// --- END extract ---

			// Initially set the cleanupFns to only close the DB.
			cleanupFns = append(cleanupFns, closeDB)
			accountSequenceCacheStore := offchainMultiStore.GetKVStore(cosmostypes.NewKVStoreKey("account_sequence_cache"))

			return flags2.CheckAutoSequenceFlag(
				cmd, clientCtx,
				accountSequenceCacheStore,
				func(updateAccountSequenceCache func() error) {
					cleanupFns = append(cleanupFns, updateAccountSequenceCache)
				},
			)
		},
		// TODO_IN_THIS_COMMIT: godoc...
		PersistentPostRunE: func(cmd *cobra.Command, _ []string) error {
			// Collect any errors from all cleanupFns;
			// i.e. ALL cleanup fns are ALWAYS called.
			errs := make([]error, 0)
			if len(cleanupFns) > 0 {
				for _, cleanupFn := range cleanupFns {
					if err := cleanupFn(); err != nil {
						errs = append(errs, err)
					}
				}
			}
			return errors.Join(errs...)
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
		flags.FlagChainID:        DefaultChainID,
		flags.FlagKeyringBackend: "test",
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

	rootCmd.PersistentFlags().Bool(flags2.FlagAutoSequence, flags2.DefaultAutoSequence, flags2.FlagAutoSequenceUsage)

	return rootCmd
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
