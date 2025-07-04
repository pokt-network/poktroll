package relay_authenticator

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
)

var _ relayer.RelayAuthenticator = (*relayAuthenticator)(nil)

// relayAuthenticator is a relayer.RelayerAuthenticator implementation that is
// responsible for authenticating relay requests and responses.
// It verifies the relay request signature and session validity, and signs relay responses.
type relayAuthenticator struct {
	logger polylog.Logger

	// signingKeyNames are the supplier operator key names in the Cosmos's keybase.
	// They are used along with the keyring to get the supplier operator addresses
	// and sign relay responses.
	signingKeyNames []string
	keyring         keyring.Keyring

	// sessionQuerier is the query client used to get the current session & session
	// params from the blockchain, which are needed to check if the relay proxy
	// should be serving an incoming relay request.
	sessionQuerier client.SessionQueryClient

	// sharedQuerier is the query client used to get the current shared params from
	// the blockchain, which are needed to check if the relay proxy should be serving
	// an incoming relay request.
	sharedQuerier client.SharedQueryClient

	// blockClient is the client used to get the block at the latest height from
	// the blockchain/ and be notified of new incoming blocks.
	// It is used to update the current session data.
	blockClient client.BlockClient

	// ringClient stores and manages rings for signing/verifying relay requests using
	// gateway and application public keys.
	ringClient crypto.RingClient

	// operatorAddressToSigningKeyNameMap is a map with a CosmoSDK address as a key,
	// and the keyring signing key name as a value.
	//
	// It is used to:
	// 1. Check if an incoming relay request matches a supplier hosted by the relay miner
	// 2. Get the corresponding keyring signing key name to sign the relay response
	operatorAddressToSigningKeyNameMap map[string]string

	// nodeStatus is the node status client used to get the current node status from
	// the blockchain, which is needed to check if the relay proxy should be serving
	// an incoming relay request.
	nodeStatus client.NodeStatusClient
}

// NewRelayAuthenticator creates a new relay authenticator with the given dependencies and options.
//
// Required dependencies:
//   - polylog.Logger
//   - keyring.Keyring
//   - client.SessionQueryClient
//   - client.SharedQueryClient
//   - client.BlockClient
//   - crypto.RingClient
func NewRelayAuthenticator(
	deps depinject.Config,
	opts ...relayer.RelayAuthenticatorOption,
) (relayer.RelayAuthenticator, error) {
	ra := &relayAuthenticator{}

	if err := depinject.Inject(
		deps,
		&ra.logger,
		&ra.keyring,
		&ra.sessionQuerier,
		&ra.sharedQuerier,
		&ra.blockClient,
		&ra.ringClient,
		&ra.nodeStatus,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(ra)
	}

	if err := ra.validateConfig(); err != nil {
		return nil, err
	}

	if err := ra.populateOperatorAddressToSigningKeyNameMap(); err != nil {
		return nil, err
	}

	if err := ra.populateNodeStatus(); err != nil {
		return nil, err
	}

	return ra, nil
}

// GetSupplierOperatorAddresses returns the supplier operator addresses that
// the relay authenticator can use to sign relay responses.
func (ra *relayAuthenticator) GetSupplierOperatorAddresses() []string {
	addresses := make([]string, 0, len(ra.operatorAddressToSigningKeyNameMap))
	for address := range ra.operatorAddressToSigningKeyNameMap {
		addresses = append(addresses, address)
	}

	return addresses
}

// validateConfig validates the relayer proxy's configuration options and returns an error if it is invalid.
// TODO_TEST: Add tests for validating these configurations.
func (ra *relayAuthenticator) validateConfig() error {
	if len(ra.signingKeyNames) == 0 || ra.signingKeyNames[0] == "" {
		return ErrRelayAuthenticatorUndefinedSigningKeyNames
	}

	return nil
}

// populateOperatorAddressToSigningKeyNameMap populates the operatorAddressToSigningKeyNameMap
// with the supplier operator addresses as keys and the keyring signing key names as values.
func (ra *relayAuthenticator) populateOperatorAddressToSigningKeyNameMap() error {
	ra.operatorAddressToSigningKeyNameMap = make(map[string]string, len(ra.signingKeyNames))
	for _, operatorSigningKeyName := range ra.signingKeyNames {
		supplierOperatorKey, err := ra.keyring.Key(operatorSigningKeyName)
		if err != nil {
			return err
		}

		supplierOperatorAddress, err := supplierOperatorKey.GetAddress()
		if err != nil {
			return err
		}

		ra.operatorAddressToSigningKeyNameMap[supplierOperatorAddress.String()] = operatorSigningKeyName
	}

	return nil
}

func (ra *relayAuthenticator) populateNodeStatus() error {
	ctx := context.Background()
	queryClientCtx := context.Background()

	// Get the chain ID from the configured query client context.
	nodeStatus, err := cmtservice.GetNodeStatus(ctx, ra.quer)
	if err != nil {
		return err
	}

	ra.nodeStatus = nodeStatus
	return nil
}
