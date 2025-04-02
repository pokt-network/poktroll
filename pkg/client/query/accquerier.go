package query

import (
	"context"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	grpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ client.AccountQueryClient = (*accQuerier)(nil)

// accQuerier is a wrapper around the accounttypes.QueryClient that enables the
// querying of onchain account information through a single exposed method
// which returns an accounttypes.AccountI interface
type accQuerier struct {
	clientConn     grpc.ClientConn
	accountQuerier accounttypes.QueryClient
	logger         polylog.Logger

	// accountsCache caches accountQueryClient.Account requests
	accountsCache cache.KeyValueCache[types.AccountI]
}

// NewAccountQuerier returns a new instance of a client.AccountQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx
func NewAccountQuerier(deps depinject.Config) (client.AccountQueryClient, error) {
	aq := &accQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientConn,
		&aq.logger,
		&aq.accountsCache,
	); err != nil {
		return nil, err
	}

	aq.accountQuerier = accounttypes.NewQueryClient(aq.clientConn)

	return aq, nil
}

// GetAccount returns an accounttypes.AccountI interface for a given address
func (aq *accQuerier) GetAccount(
	ctx context.Context,
	address string,
) (types.AccountI, error) {
	logger := aq.logger.With("query_client", "account", "method", "GetAccount")

	// Check if the account is present in the cache.
	if account, found := aq.accountsCache.Get(address); found {
		logger.Debug().Msgf("cache hit for account address key: %s", address)
		return account, nil
	}

	logger.Debug().Msgf("cache miss for account address key: %s", address)

	// Query the blockchain for the account record
	req := &accounttypes.QueryAccountRequest{Address: address}
	res, err := aq.accountQuerier.Account(ctx, req)
	if err != nil {
		return nil, ErrQueryAccountNotFound.Wrapf("address: %s [%v]", address, err)
	}

	// Unpack and cache the account object
	var fetchedAccount types.AccountI
	if err = queryCodec.UnpackAny(res.Account, &fetchedAccount); err != nil {
		return nil, ErrQueryUnableToDeserializeAccount.Wrapf("address: %s [%v]", address, err)
	}

	// Fetched accounts must have their public key set. Do not cache accounts
	// that do not have a public key set, such as the ones resulting from genesis
	// as they may continue failing due to the caching mechanism, even after they
	// got their public key recorded onchain.
	if fetchedAccount.GetPubKey() == nil {
		return nil, ErrQueryPubKeyNotFound
	}

	// Cache the fetched account for future queries.
	aq.accountsCache.Set(address, fetchedAccount)
	return fetchedAccount, nil
}

// GetPubKeyFromAddress returns the public key of the given address.
// It uses the accountQuerier to get the account and then returns its public key.
func (aq *accQuerier) GetPubKeyFromAddress(ctx context.Context, address string) (cryptotypes.PubKey, error) {
	acc, err := aq.GetAccount(ctx, address)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, ErrQueryAccountNotFound.Wrapf("address: %s", address)
	}

	// If the account's public key is nil, then return an error.
	pubKey := acc.GetPubKey()
	if pubKey == nil {
		return nil, ErrQueryPubKeyNotFound
	}

	return pubKey, nil
}
