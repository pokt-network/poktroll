package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	keyringtypes "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	ring "github.com/pokt-network/ring-go"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/signer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var (
	flagRelayFrom     string
	flagRelayPayload  string
	flagRelaySupplier string
)

// Things I want:
// - List current suppliers: given app & service ID, list all suppliers
// -

// relayCmd defines the `relay` subcommand for sending a relay as an application.
func relayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay",
		Short: "Send a relay as a staked application (for testing)",
		RunE:  runRelay,
	}

	cmd.Flags().StringVar(&flagNodeRPCURL, cosmosflags.FlagNode, "https://shannon-testnet-grove-rpc.beta.poktroll.com", "Cosmos node RPC URL (required)")
	cmd.Flags().StringVar(&flagRelayFrom, "app", "", "Name of the staked application key (required)")
	cmd.Flags().StringVar(&flagRelaySupplier, "supplier", "", "Supplier endpoint URL (e.g. http://localhost:8081/relay)")
	cmd.Flags().StringVar(&flagRelayPayload, "payload", "", "Relay payload (hex encoded)")
	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("supplier")
	_ = cmd.MarkFlagRequired("payload")

	return cmd
}

func runRelay(cmd *cobra.Command, args []string) error {
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to get Cosmos client context: %w", err)
	}

	kr := clientCtx.Keyring
	info, err := kr.Key(flagRelayFrom)
	if err != nil {
		return fmt.Errorf("failed to find key '%s' in keyring: %w", flagRelayFrom, err)
	}

	payload, err := hex.DecodeString(flagRelayPayload)
	if err != nil {
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	appAddr, err := info.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get application address: %w", err)
	}

	relayReq := &servicetypes.RelayRequest{
		Payload: payload,
		Meta: servicetypes.RelayRequestMetadata{
			SupplierOperatorAddress: appAddr.String(),
		},
	}

	resp, err := sendRelay(flagRelaySupplier, relayReq)
	if err != nil {
		return fmt.Errorf("failed to send relay: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Relay response: %s\n", resp)
	return nil
}

func sendRelay(endpoint string, relayReq *servicetypes.RelayRequest) (string, error) {
	// TODO: Implement the HTTP POST to the supplier endpoint with relayReq as the body
	// For now, just print a stub message
	return "(stub) relay sent to " + endpoint, nil
}

// GetApplicationRingSignature crafts a ring signer for test purposes and uses
// it to sign the relay request
func GetApplicationRingSignature(
	t *testing.T,
	req *servicetypes.RelayRequest,
	appPrivateKey cryptotypes.PrivKey,
) []byte {
	publicKey := appPrivateKey.PubKey()
	curve := ring_secp256k1.NewCurve()

	var point ringtypes.Point
	var err error
	if point, err = curve.DecodeToPoint(publicKey.Bytes()); err != nil {
		return nil
	}

	// At least two points are required to create a ring signer so we are reusing
	// the same key for it
	points := []ringtypes.Point{point, point}
	var pointsRing *ring.Ring
	pointsRing, err = ring.NewFixedKeyRingFromPublicKeys(curve, points)
	if err != nil {
		return nil
	}

	scalar, err := curve.DecodeToScalar(appPrivateKey.Bytes())
	if err != nil {
		return nil
	}

	signer := signer.NewRingSigner(pointsRing, scalar)

	signableBz, err := req.GetSignableBytesHash()
	if err != nil {
		return nil
	}

	signature, err := signer.Sign(signableBz)
	if err != nil {
		return nil
	}

	return signature
}

// getAddressFromKeyName returns the address of the provided keyring key name
func getAddressFromKeyName(keyName string) string {
	var keyring keyringtypes.Keyring

	// err := depinject.Inject(deps, &keyring)
	// require.NoError(err)

	var account *keyringtypes.Record
	var err error
	if account, err = keyring.Key(keyName); err != nil {
		return ""
	}

	accAddress, err := account.GetAddress()
	if err != nil {
		return ""
	}

	return accAddress.String()
}

// GenerateRelayRequest generates a relay request with the provided parameters
func GenerateRelayRequest(
	privKey *secp256k1.PrivKey,
	serviceId string,
	blockHeight int64,
	supplierOperatorKeyName string,
	payload []byte,
	session *sessiontypes.SessionHeader,
) *servicetypes.RelayRequest {
	appAddress := getAddressFromPrivateKey(privKey)
	fmt.Println("appAddress:", appAddress)
	supplierOperatorAddress := getAddressFromKeyName(supplierOperatorKeyName)
	fmt.Println("supplierOperatorAddress:", supplierOperatorAddress)

	return &servicetypes.RelayRequest{
		Meta: servicetypes.RelayRequestMetadata{
			SessionHeader:           session,
			SupplierOperatorAddress: supplierOperatorAddress,
			// The returned relay is unsigned and must be signed elsewhere for functionality
			Signature: []byte(""),
		},
		Payload: payload,
	}
}

// getAddressFromPrivateKey returns the address of the provided private key
func getAddressFromPrivateKey(privKey *secp256k1.PrivKey) string {
	addressBz := privKey.PubKey().Address()
	address, err := bech32.ConvertAndEncode("pokt", addressBz)
	if err != nil {
		return ""
	}
	return address
}
