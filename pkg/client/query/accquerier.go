package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	grpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
)

var _ client.AccountQueryClient = (*accQuerier)(nil)

// accQuerier is a wrapper around the accounttypes.QueryClient that enables the
// querying of on-chain account information through a single exposed method
// which returns an accounttypes.AccountI interface
type accQuerier struct {
	clientConn     grpc.ClientConn
	accountQuerier accounttypes.QueryClient

	// accountCache is a cache of accounts that have already been queried.
	// TODO_TECHDEBT: Add a size limit to the cache and consider an LRU cache.
	accountCache   map[string]types.AccountI
	accountCacheMu sync.Mutex
}

// NewAccountQuerier returns a new instance of a client.AccountQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx
func NewAccountQuerier(deps depinject.Config) (client.AccountQueryClient, error) {
	aq := &accQuerier{accountCache: make(map[string]types.AccountI)}

	if err := depinject.Inject(
		deps,
		&aq.clientConn,
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
	aq.accountCacheMu.Lock()
	defer aq.accountCacheMu.Unlock()

	if foundAccount, isAccountFound := aq.accountCache[address]; isAccountFound {
		return foundAccount, nil
	}

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
	// got their public key recorded on-chain.
	if fetchedAccount.GetPubKey() == nil {
		return nil, ErrQueryPubKeyNotFound
	}

	aq.accountCache[address] = fetchedAccount

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

func (aq *accQuerier) ClearCache() {
}
