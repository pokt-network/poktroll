package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/appgateserver"
	"github.com/pokt-network/poktroll/pkg/deps/config"
)

const omittedDefaultFlagValue = "explicitly omitting default"

var (
	flagSigningKey        string
	flagSelfSigning       bool
	flagListeningEndpoint string
	flagQueryNodeUrl      string
)

func AppGateServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appgate-server",
		Short: "Starts the AppGate server",
		Long: `Starts the AppGate server that listens for incoming relay requests and handles
the necessary on-chain interactions (sessions, suppliers, etc) to receive the
respective relay response.

-- App Mode (Flag)- -
If the server is started with a defined '--self-signing' flag, it will behave
as an Application. Any incoming requests will be signed by using the private
key and ring associated with the '--signing-key' flag.

-- Gateway Mode (Flag)--
If the '--self-signing' flag is not provided, the server will behave as a Gateway.
It will sign relays on behalf of any Application sending it relays, provided
that the address associated with '--signing-key' has been delegated to. This is
necessary for the application<->gateway ring signature to function.

-- App Mode (HTTP) --
If an application doesn't provide the '--self-signing' flag, it can still send
relays to the AppGate server and function as an Application, provided that:
1. Each request contains the '?senderAddress=[address]' query parameter
2. The key associated with the '--signing-key' flag belongs to the address
   provided in the request, otherwise the ring signature will not be valid.`,
		Args: cobra.NoArgs,
		RunE: runAppGateServer,
	}

	cmd.Flags().StringVar(&flagSigningKey, "signing-key", "", "The name of the key that will be used to sign relays")
	cmd.Flags().StringVar(&flagListeningEndpoint, "listening-endpoint", "http://localhost:42069", "The host and port that the appgate server will listen on")
	cmd.Flags().BoolVar(&flagSelfSigning, "self-signing", false, "Whether the server should sign all incoming requests with its own ring (for applications)")
	cmd.Flags().StringVar(&flagQueryNodeUrl, "query-node", omittedDefaultFlagValue, "The URL of the pocket node to query for on-chain data")

	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().String(cosmosflags.FlagNode, omittedDefaultFlagValue, "The URL of the comet tcp endpoint to communicate with the pocket blockchain")

	return cmd
}

func runAppGateServer(cmd *cobra.Command, _ []string) error {
	// Create a context that is canceled when the command is interrupted
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx()

	// Handle interrupt and kill signals asynchronously.
	signals.GoOnExitSignal(cancelCtx)

	// Parse the listening endpoint.
	listeningUrl, err := url.Parse(flagListeningEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse listening endpoint: %w", err)
	}

	// Setup the AppGate server dependencies.
	appGateServerDeps, err := setupAppGateServerDependencies(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to setup AppGate server dependencies: %w", err)
	}

	log.Println("INFO: Creating AppGate server...")

	// Create the AppGate server.
	appGateServer, err := appgateserver.NewAppGateServer(
		appGateServerDeps,
		appgateserver.WithSigningInformation(&appgateserver.SigningInformation{
			// provide the name of the key to use for signing all incoming requests
			SigningKeyName: flagSigningKey,
			// provide whether the appgate server should sign all incoming requests
			// with its own ring (for applications) or not (for gateways)
			SelfSigning: flagSelfSigning,
		}),
		appgateserver.WithListeningUrl(listeningUrl),
	)
	if err != nil {
		return fmt.Errorf("failed to create AppGate server: %w", err)
	}

	log.Printf("INFO: Starting AppGate server, listening on %s...", listeningUrl.String())

	// Start the AppGate server.
	if err := appGateServer.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start app gate server: %w", err)
	} else if errors.Is(err, http.ErrServerClosed) {
		log.Println("INFO: AppGate server stopped")
	}

	return nil
}

func setupAppGateServerDependencies(
	ctx context.Context,
	cmd *cobra.Command,
) (depinject.Config, error) {
	pocketNodeWebsocketUrl, err := getPocketNodeWebsocketUrl()
	if err != nil {
		return nil, err
	}

	supplierFuncs := []config.SupplierFn{
		config.NewSupplyEventsQueryClientFn(pocketNodeWebsocketUrl),
		config.NewSupplyBlockClientFn(pocketNodeWebsocketUrl),
		newSupplyQueryClientContextFn(flagQueryNodeUrl),
	}

	return config.SupplyConfig(ctx, cmd, supplierFuncs)
}

// getPocketNodeWebsocketUrl returns the websocket URL of the Pocket Node to
// connect to for subscribing to on-chain events.
func getPocketNodeWebsocketUrl() (string, error) {
	if flagQueryNodeUrl == omittedDefaultFlagValue {
		return "", errors.New("missing required flag: --query-node")
	}

	pocketNodeURL, err := url.Parse(flagQueryNodeUrl)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ws://%s/websocket", pocketNodeURL.Host), nil
}

// newSupplyQueryClientContextFn returns a new depinject.Config which is supplied with
// the given deps and a new cosmos ClientCtx
func newSupplyQueryClientContextFn(pocketQueryClientUrl string) config.SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Set --node flag to the pocketQueryClientUrl for the client context
		// This flag is read by cosmosclient.GetClientQueryContext.
		err := cmd.Flags().Set(cosmosflags.FlagNode, pocketQueryClientUrl)
		if err != nil {
			return nil, err
		}

		// NB: Currently, the implementations of GetClientTxContext() and
		// GetClientQueryContext() are identical, allowing for their interchangeable
		// use in both querying and transaction operations. However, in order to support
		// independent configuration of client contexts for distinct querying and
		// transacting purposes. E.g.: transactions are dispatched to the sequencer
		// while queries are handled by a trusted full-node.
		queryClientCtx, err := cosmosclient.GetClientQueryContext(cmd)
		if err != nil {
			return nil, err
		}
		deps = depinject.Configs(deps, depinject.Supply(
			queryClientCtx,
		))
		return deps, nil
	}
}
