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
	signingKeyName    string
	selfSigning       bool
	listeningEndpoint string
	cometWebsocketUrl string
)

func AppGateServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appgate-server",
		Short: "Starts the AppGate server",
		Long: `Starts the AppGate server that will listen for incoming relays requests and will handle
the interaction with the chain, sessions and suppliers in order to receive the correct
response for the request.

If the server is started with a defined --self-signing flag, it will behave
as an application and sign any incoming requests with the private key associated with
the --signing-key-name flag. If however, this flag is not provided, the server will
behave as a gateway and will sign relays on behalf of any application sending it relays provided
that the address recieved in the query parameters of a request has been delegated to by the
gateway, this is so that it can sign relays using the ring of the application with the
key associated with the --signing-key-name flag.

If an application doesn't provide the --self-signing flag, it will be able to send relays
to the AppGate server and it will still function as an application, however each request
will have to contain the "?senderAddress=[address]" query parameter, where [address] is
the address of the application that is sending the request. This is so that the server
can generate the correct ring for the application and sign the request.`,
		Args: cobra.NoArgs,
		RunE: runAppGateServer,
	}

	cmd.Flags().StringVar(&signingKeyName, "signing-key-name", "", "The name of the key that will be used to sign relays")
	cmd.Flags().StringVar(&listeningEndpoint, "listening-endpoint", "http://localhost:42069", "The host and port that the server will listen on")
	cmd.Flags().StringVar(&cometWebsocketUrl, "comet-websocket-url", "ws://localhost:36657/websocket", "The URL of the tendermint websocket endpoint to interact with the chain")
	cmd.Flags().BoolVar(&selfSigning, "self-signing", false, "Whether the server should sign all incoming requests with its own ring (for applications)")

	cmd.Flags().String(flags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:36657", "tcp://<host>:<port> to tendermint rpc interface for this chain")

	return cmd
}

func runAppGateServer(cmd *cobra.Command, _ []string) error {
	// Create a context that is cancelled when the command is interrupted
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx()

	// Retrieve the client context for the chain interactions.
	clientCtx := cosmosclient.GetClientContextFromCmd(cmd)

	// Parse the listening endpoint.
	listeningUrl, err := url.Parse(listeningEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse listening endpoint: %w", err)
	}

	// Obtain the tendermint websocket endpoint from the client context.
	cometWSUrl, err := url.Parse(clientCtx.NodeURI + "/websocket")
	if err != nil {
		return fmt.Errorf("failed to parse block query URL: %w", err)
	}
	cometWSUrl.Scheme = "ws"
	// If the comet websocket URL is not provided, use the one from the client context.
	if cometWebsocketUrl == "" {
		cometWebsocketUrl = cometWSUrl.String()
	}

	log.Printf("INFO: Creating block client, using websocket URL: %s...", cometWebsocketUrl)

	// Create the block client with its dependency on the events client.
	eventsQueryClient := eventsquery.NewEventsQueryClient(cometWebsocketUrl)
	deps := depinject.Supply(eventsQueryClient)
	blockClient, err := blockclient.NewBlockClient(ctx, deps, cometWebsocketUrl)
	if err != nil {
		return fmt.Errorf("failed to create block client: %w", err)
	}

	log.Println("INFO: Creating AppGate server...")

	key, err := clientCtx.Keyring.Key(signingKeyName)
	if err != nil {
		return fmt.Errorf("failed to get key from keyring: %w", err)
	}

	appAddress, err := key.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get address from key: %w", err)
	}
	signingAddress := ""
	if selfSigning {
		signingAddress = appAddress.String()
	}

	signingKey, err := recordLocalToScalar(key.GetLocal())
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
