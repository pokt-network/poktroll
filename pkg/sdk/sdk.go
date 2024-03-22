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
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ POKTRollSDK = (*poktrollSDK)(nil)

// POKTRollSDKConfig is the configuration for the POKTRollSDK.
// It contains the Pocket Node URL to be used by the queriers and the private key
// to be used for signing relay requests.
// Deps is an optional field that can be used to provide the needed dependencies
// for the SDK. If it is not provided, the SDK will build the dependencies itself.
type POKTRollSDKConfig struct {
	QueryNodeGRPCUrl *url.URL
	QueryNodeUrl     *url.URL
	PrivateKey       cryptotypes.PrivKey
	Deps             depinject.Config
}

// poktrollSDK is the implementation of the POKTRollSDK.
type poktrollSDK struct {
	logger polylog.Logger
	config *POKTRollSDKConfig

	// signingKey is the scalar representation of the private key to be used
	// for signing relay requests.
	signingKey ringtypes.Scalar

	// ringCache is used to obtain and store the ring for the application.
	ringCache crypto.RingCache

	// sessionQuerier is the querier for the session module.
	// It used to get the current session for the application given a requested service.
	sessionQuerier client.SessionQueryClient

	// serviceSessionSuppliersMu is a mutex to protect latestSessions map reads and updates.
	serviceSessionSuppliersMu sync.RWMutex

	// serviceSessionSuppliers is a map of serviceId -> {appAddress -> SessionSuppliers}
	// for a specific session
	serviceSessionSuppliers map[string]map[string]*SessionSuppliers

	// applicationQuerier is the querier for the application module.
	// It is used to query a specific application or all applications
	applicationQuerier client.ApplicationQueryClient

	// accountQuerier is the querier for the account module.
	// It retrieves on-chain accounts provided the address.
	accountQuerier client.AccountQueryClient

	// blockClient is the client for the block module.
	// It is used to get the current block height to query for the current session.
	blockClient client.BlockClient
}

// NewPOKTRollSDK creates a new POKTRollSDK instance with the given configuration.
func NewPOKTRollSDK(ctx context.Context, config *POKTRollSDKConfig) (POKTRollSDK, error) {
	sdk := &poktrollSDK{
		config:                  config,
		serviceSessionSuppliers: make(map[string]map[string]*SessionSuppliers),
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
		&sdk.logger,
		&sdk.ringCache,
		&sdk.sessionQuerier,
		&sdk.accountQuerier,
		&sdk.applicationQuerier,
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

	// Start the ring cache, when the context is canceled, the ring cache
	// will stop. And clear any cached rings.
	sdk.ringCache.Start(ctx)

	return sdk, nil
}
