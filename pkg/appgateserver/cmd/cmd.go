package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"cosmossdk.io/depinject"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/appgateserver"
	appgateconfig "github.com/pokt-network/poktroll/pkg/appgateserver/config"
	"github.com/pokt-network/poktroll/pkg/deps/config"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

// We're `explicitly omitting default` so that the appgateserver crashes if these aren't specified.
const omittedDefaultFlagValue = "explicitly omitting default"

var (
	// flagAppGateConfig is the variable containing the AppGate config filepath
	// sourced from the `--config` flag.
	flagAppGateConfig string
	// flagNodeRPCURL is the variable containing the Cosmos node RPC URL flag value.
	flagNodeRPCURL string
	// flagNodeGRPCURL is the variable containing the Cosmos node GRPC URL flag value.
	flagNodeGRPCURL string
	// flagLogLevel is the variable to set a log level (used by cosmos and polylog).
	flagLogLevel string
)

// AppGateServerCmd returns the Cobra command for running the AppGate server.
func AppGateServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appgate-server",
		Short: "Starts the AppGate server",
		Long: `Starts the AppGate server that listens for incoming relay requests and handles
the necessary on-chain interactions (sessions, suppliers, etc) to receive the
respective relay response.

-- App Mode --
If the server is started with a defined 'self-signing' configuration directive,
it will behave as an Application. Any incoming requests will be signed by using
the private key and ring associated with the 'signing_key' configuration directive.

-- Gateway Mode --
If the 'self_signing' configuration directive is not provided, the server will
behave as a Gateway.
It will sign relays on behalf of any Application sending it relays, provided
that the address associated with 'signing_key' has been delegated to. This is
necessary for the application<->gateway ring signature to function.

-- App Mode (HTTP) --
If an application doesn't provide the 'self_signing' configuration directive,
it can still send relays to the AppGate server and function as an Application,
provided that:
1. Each request contains the '?applicationAddr=[address]' query parameter
2. The key associated with the 'signing_key' configuration directive belongs
   to the address provided in the request, otherwise the ring signature will not be valid.`,
		Args: cobra.NoArgs,
		RunE: runAppGateServer,
	}

	// Custom flags
	cmd.Flags().StringVar(&flagAppGateConfig, "config", "", "The path to the appgate config file")

	// Cosmos flags
	// TODO_TECHDEBT(#256): Remove unneeded cosmos flags.
	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().StringVar(&flagNodeRPCURL, cosmosflags.FlagNode, omittedDefaultFlagValue, "Register the default Cosmos node flag, which is needed to initialize the Cosmos query context correctly. It can be used to override the `QueryNodeUrl` field in the config file if specified.")
	cmd.Flags().StringVar(&flagNodeGRPCURL, cosmosflags.FlagGRPC, omittedDefaultFlagValue, "Register the default Cosmos node grpc flag, which is needed to initialize the Cosmos query context with grpc correctly. It can be used to override the `QueryNodeGRPCUrl` field in the config file if specified.")
	cmd.Flags().Bool(cosmosflags.FlagGRPCInsecure, true, "Used to initialize the Cosmos query context with grpc security options. It can be used to override the `QueryNodeGRPCInsecure` field in the config file if specified.")
	cmd.Flags().StringVar(&flagLogLevel, cosmosflags.FlagLogLevel, "debug", "The logging level (debug|info|warn|error)")

	return cmd
}

func runAppGateServer(cmd *cobra.Command, _ []string) error {
	// Create a context that is canceled when the command is interrupted
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx()

	// Handle interrupt and kill signals asynchronously.
	signals.GoOnExitSignal(cancelCtx)

	configContent, err := os.ReadFile(flagAppGateConfig)
	if err != nil {
		return err
	}

	// TODO_TECHDEBT: add logger level and output options to the config.
	appGateConfigs, err := appgateconfig.ParseAppGateServerConfigs(configContent)
	if err != nil {
		return err
	}

	// TODO_TECHDEBT: populate logger from the config (ideally, from viper).
	loggerOpts := []polylog.LoggerOption{
		polyzero.WithLevel(polyzero.ParseLevel(flagLogLevel)),
		polyzero.WithOutput(os.Stderr),
	}

	// Construct a logger and associate it with the command context.
	logger := polyzero.NewLogger(loggerOpts...)
	ctx = logger.WithContext(ctx)
	cmd.SetContext(ctx)

	// Setup the AppGate server dependencies.
	appGateServerDeps, err := setupAppGateServerDependencies(ctx, cmd, appGateConfigs)
	if err != nil {
		return fmt.Errorf("failed to setup AppGate server dependencies: %w", err)
	}

	logger.Info().Msg("Creating AppGate server...")

	// Create the AppGate server.
	appGateServer, err := appgateserver.NewAppGateServer(
		appGateServerDeps,
		appgateserver.WithSigningInformation(&appgateserver.SigningInformation{
			// provide the name of the key to use for signing all incoming requests
			SigningKeyName: appGateConfigs.SigningKey,
			// provide whether the appgate server should sign all incoming requests
			// with its own ring (for applications) or not (for gateways)
			SelfSigning: appGateConfigs.SelfSigning,
		}),
		appgateserver.WithListeningUrl(appGateConfigs.ListeningEndpoint),
	)
	if err != nil {
		return fmt.Errorf("failed to create AppGate server: %w", err)
	}

	logger.Info().
		Str("listening_endpoint", appGateConfigs.ListeningEndpoint.String()).
		Msg("Starting AppGate server...")

	if appGateConfigs.Metrics.Enabled {
		err = appGateServer.ServeMetrics(appGateConfigs.Metrics.Addr)
		if err != nil {
			return fmt.Errorf("failed to start metrics endpoint: %w", err)
		}
	}

	if appGateConfigs.Pprof.Enabled {
		err = appGateServer.ServePprof(ctx, appGateConfigs.Pprof.Addr)
		if err != nil {
			return fmt.Errorf("failed to start pprof endpoint: %w", err)
		}
	}

	// Start the AppGate server.
	if err := appGateServer.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start app gate server: %w", err)
	} else if errors.Is(err, http.ErrServerClosed) {
		logger.Info().Msg("AppGate server stopped")
	}

	return nil
}

func setupAppGateServerDependencies(
	ctx context.Context,
	cmd *cobra.Command,
	appGateConfig *appgateconfig.AppGateServerConfig,
) (_ depinject.Config, err error) {
	queryNodeRPCURL := appGateConfig.QueryNodeRPCUrl
	queryNodeGRPCURL := appGateConfig.QueryNodeGRPCUrl

	// Override the config file's `QueryNodeGRPCUrl` field
	// with the `--grpc-addr` flag if it was specified.
	// TODO_TECHDEBT(#223) Remove this check once viper is used as SoT for overridable config values.
	if flagNodeGRPCURL != omittedDefaultFlagValue {
		queryNodeGRPCURL, err = url.Parse(flagNodeGRPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse grpc query URL: %w", err)
		}
	}

	// Override the config file's `QueryNodeRPCURL` field
	// with the `--node` flag if it was specified.
	// TODO_TECHDEBT(#223) Remove this check once viper is used as SoT for overridable config values.
	if flagNodeRPCURL != omittedDefaultFlagValue {
		queryNodeRPCURL, err = url.Parse(flagNodeRPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rpc query URL: %w", err)
		}
	}

	supplierFuncs := []config.SupplierFn{
		config.NewSupplyLoggerFromCtx(ctx),
		config.NewSupplyEventsQueryClientFn(queryNodeRPCURL),   // leaf
		config.NewSupplyBlockQueryClientFn(queryNodeRPCURL),    // leaf
		config.NewSupplyBlockClientFn(queryNodeRPCURL),         // leaf
		config.NewSupplyQueryClientContextFn(queryNodeGRPCURL), // leaf
		config.NewSupplyAccountQuerierFn(),                     // leaf
		config.NewSupplyApplicationQuerierFn(),                 // leaf
		config.NewSupplySessionQuerierFn(),                     // leaf
		config.NewSupplySharedQueryClientFn(),                  // leaf

		config.NewSupplyShannonSDKFn(appGateConfig.SigningKey),
	}

	return config.SupplyConfig(ctx, cmd, supplierFuncs)
}
