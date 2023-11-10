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
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
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

	// Retrieve the client context for the chain interactions.
	clientCtx := cosmosclient.GetClientContextFromCmd(cmd)

	// Parse the listening endpoint.
	listeningUrl, err := url.Parse(flagListeningEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse listening endpoint: %w", err)
	}

	log.Printf("INFO: Creating block client, using comet websocket URL: %s...", flagCometWebsocketUrl)

	// Create the block client with its dependency on the events client.
	eventsQueryClient := eventsquery.NewEventsQueryClient(flagCometWebsocketUrl)
	deps := depinject.Supply(eventsQueryClient)
	blockClient, err := blockclient.NewBlockClient(ctx, deps, flagCometWebsocketUrl)
	if err != nil {
		return fmt.Errorf("failed to create block client: %w", err)
	}

	log.Println("INFO: Creating AppGate server...")

	keyRecord, err := clientCtx.Keyring.Key(flagSigningKey)
	if err != nil {
		return fmt.Errorf("failed to get key from keyring: %w", err)
	}

	appAddress, err := keyRecord.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get address from key: %w", err)
	}
	signingAddress := ""
	if flagSelfSigning {
		signingAddress = appAddress.String()
	}

	// Convert the key record to a private key and return the scalar
	// point on the secp256k1 curve that it corresponds to.
	// If the key is not a secp256k1 key, this will return an error.
	signingKey, err := recordLocalToScalar(keyRecord.GetLocal())
	if err != nil {
		return fmt.Errorf("failed to convert private key to scalar: %w", err)
	}
	signingInfo := appgateserver.SigningInformation{
		SigningKey: signingKey,
		AppAddress: signingAddress,
	}

	// Create the AppGate server.
	appGateServerDeps := depinject.Supply(
		clientCtx,
		blockClient,
	)

	appGateServer, err := appgateserver.NewAppGateServer(
		appGateServerDeps,
		appgateserver.WithSigningInformation(&signingInfo),
		appgateserver.WithListeningUrl(listeningUrl),
	)
	if err != nil {
		return fmt.Errorf("failed to create AppGate server: %w", err)
	}

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

	log.Printf("INFO: Starting AppGate server, listening on %s...", listeningUrl.String())

	// Start the AppGate server.
	if err := appGateServer.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start app gate server: %w", err)
	} else if errors.Is(err, http.ErrServerClosed) {
		log.Println("INFO: AppGate server stopped")
	}

	return nil
}

// recordLocalToScalar converts the private key obtained from a
// key record to a scalar point on the secp256k1 curve
func recordLocalToScalar(local *keyring.Record_Local) (ringtypes.Scalar, error) {
	if local == nil {
		return nil, fmt.Errorf("cannot extract private key from key record: nil")
	}
	priv, ok := local.PrivKey.GetCachedValue().(cryptotypes.PrivKey)
	if !ok {
		return nil, fmt.Errorf("cannot extract private key from key record: %T", local.PrivKey.GetCachedValue())
	}
	if _, ok := priv.(*secp256k1.PrivKey); !ok {
		return nil, fmt.Errorf("unexpected private key type: %T, want %T", priv, &secp256k1.PrivKey{})
	}
	crv := ring_secp256k1.NewCurve()
	privKey, err := crv.DecodeToScalar(priv.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	return privKey, nil
}
