package sdk

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
)

var _ POKTRollSDK = (*poktrollSDK)(nil)

// POKTRollSDKConfig is the configuration for the POKTRollSDK.
// It contains the Pocket Node URL to be used by the queriers and the private key
// to be used for signing relay requests.
// Deps is an optional field that can be used to provide the needed dependencies
// for the SDK. If it is not provided, the SDK will build the dependencies itself.
type POKTRollSDKConfig struct {
	PocketNodeUrl *url.URL
	PrivateKey    cryptotypes.PrivKey
	Deps          depinject.Config
}

// poktrollSDK is the implementation of the POKTRollSDK.
type poktrollSDK struct {
	config *POKTRollSDKConfig

	// signingKey is the scalar representation of the private key to be used
	// for signing relay requests.
	signingKey ringtypes.Scalar

	// ringCache is used to obtain and store the ring for the application.
	ringCache crypto.RingCache

	// sessionQuerier is the querier for the session module.
	// It used to get the current session for the application given a requested service.
	sessionQuerier client.SessionQueryClient

	// sessionMu is a mutex to protect latestSessions map reads and updates.
	sessionMu sync.RWMutex

	// latestSessions is a latest sessions map of serviceId -> {appAddress -> SessionSuppliers}
	// based on the latest block data available.
	latestSessions map[string]map[string]*sessionSuppliers

	// accountQuerier is the querier for the account module.
	// It is used to get the the supplier's public key to verify the relay response signature.
	accountQuerier client.AccountQueryClient

	// blockClient is the client for the block module.
	// It is used to get the current block height to query for the current session.
	blockClient client.BlockClient

	// accountCache is a cache of the supplier accounts that has been queried
	// TODO_TECHDEBT: Add a size limit to the cache.
	supplierAccountCache map[string]cryptotypes.PubKey
}

func NewPOKTRollSDK(ctx context.Context, config *POKTRollSDKConfig) (POKTRollSDK, error) {
	sdk := &poktrollSDK{
		config:               config,
		latestSessions:       make(map[string]map[string]*sessionSuppliers),
		supplierAccountCache: make(map[string]cryptotypes.PubKey),
	}

	var err error
	var deps depinject.Config

	// Build the dependencies if they are not provided in the config.
	if config.Deps != nil {
		deps = config.Deps
	} else if deps, err = sdk.buildDeps(ctx, config); err != nil {
		return nil, err
	}

	if err := depinject.Inject(
		deps,
		&sdk.ringCache,
		&sdk.sessionQuerier,
		&sdk.accountQuerier,
		&sdk.blockClient,
	); err != nil {
		return nil, err
	}

	// Store the private key as a ring scalar to be used for ring signatures.
	crv := ring_secp256k1.NewCurve()
	sdk.signingKey, err = crv.DecodeToScalar(config.PrivateKey.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	return sdk, nil
}
