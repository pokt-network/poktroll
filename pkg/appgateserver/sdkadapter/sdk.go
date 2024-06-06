package sdkadapter

import (
	"context"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/pokt-network/shannon-sdk/sdk"

	"github.com/pokt-network/poktroll/pkg/client/query"
)

// NewShannonSDKAdapter creates a new ShannonSDK instance with the given
// signing key and dependencies.
// It initializes the necessary clients and signer for the SDK.
func NewShannonSDKAdapter(
	ctx context.Context,
	signingKey cryptotypes.PrivKey,
	deps depinject.Config,
) (*sdk.ShannonSDK, error) {
	applicationClient, err := query.NewApplicationQuerier(deps)
	if err != nil {
		return nil, err
	}

	sessionClient, err := query.NewSessionQuerier(deps)
	if err != nil {
		return nil, err
	}

	accountClient, err := query.NewAccountQuerier(deps)
	if err != nil {
		return nil, err
	}

	sharedParamsClient, err := query.NewSharedQuerier(deps)
	if err != nil {
		return nil, err
	}

	blockClient, err := NewBlockClient(ctx, deps)
	if err != nil {
		return nil, err
	}

	relayClient, err := NewRelayClient(ctx, deps)
	if err != nil {
		return nil, err
	}

	signer, err := NewSigner(signingKey)
	if err != nil {
		return nil, err
	}

	return sdk.NewShannonSDK(
		applicationClient,
		sessionClient,
		accountClient,
		sharedParamsClient,
		blockClient,
		relayClient,
		signer,
	)
}
