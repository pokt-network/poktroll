package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/appgateserver"
	blockclient "github.com/pokt-network/poktroll/pkg/client/block"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
)

var (
	flagSigningKey        string
	flagSelfSigning       bool
	flagListeningEndpoint string
	flagCometWebsocketUrl string
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
	cmd.Flags().StringVar(&flagCometWebsocketUrl, "comet-websocket-url", "ws://localhost:36657/websocket", "The URL of the comet websocket endpoint to communicate with the pocket blockchain")
	cmd.Flags().BoolVar(&flagSelfSigning, "self-signing", false, "Whether the server should sign all incoming requests with its own ring (for applications)")

	cmd.Flags().String(flags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:36657", "The URL of the comet tcp endpoint to communicate with the pocket blockchain")

	return cmd
}

func runAppGateServer(cmd *cobra.Command, _ []string) error {
	// Create a context that is canceled when the command is interrupted
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx()

	// Handle interrupts in a goroutine.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)

		// Block until we receive an interrupt or kill signal (OS-agnostic)
		<-sigCh
		log.Println("INFO: Interrupt signal received, shutting down...")

		// Signal goroutines to stop
		cancelCtx()
	}()

	// Parse the listening endpoint.
	listeningUrl, err := url.Parse(flagListeningEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse listening endpoint: %w", err)
	}

	// Setup the AppGate server dependencies.
	appGateServerDeps, err := setupAppGateServerDependencies(cmd, ctx, flagCometWebsocketUrl)
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

func setupAppGateServerDependencies(cmd *cobra.Command, ctx context.Context, cometWebsocketUrl string) (depinject.Config, error) {
	// Retrieve the client context for the chain interactions.
	clientCtx := cosmosclient.GetClientContextFromCmd(cmd)

	// Create the events client.
	eventsQueryClient := eventsquery.NewEventsQueryClient(cometWebsocketUrl)

	// Create the block client.
	log.Printf("INFO: Creating block client, using comet websocket URL: %s...", cometWebsocketUrl)
	deps := depinject.Supply(eventsQueryClient)
	blockClient, err := blockclient.NewBlockClient(ctx, deps, cometWebsocketUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create block client: %w", err)
	}

	// Return the dependencies config.
	return depinject.Supply(clientCtx, blockClient), nil
}
