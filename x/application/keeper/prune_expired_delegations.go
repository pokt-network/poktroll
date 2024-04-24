package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/pokt-network/poktroll/x/application/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

const archivedDelegationsRetentionInSessions = 3

func (k Keeper) EndBlockerPruneExpiredDelegations(ctx sdk.Context) error {
	logger := k.Logger().With("method", "EndBlockerPruneExpiredDelegations")

	currentBlockHeight := ctx.BlockHeight()
	if currentBlockHeight != sessionkeeper.GetSessionEndBlockHeight(currentBlockHeight) {
		return nil
	}
	endingSessionNumber := uint64(sessionkeeper.GetSessionNumber(currentBlockHeight))

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	archivedDelegationsKeyPrefix := types.KeyPrefix(types.ArchivedDelegationsKeyPrefix)
	store := prefix.NewStore(storeAdapter, archivedDelegationsKeyPrefix)

	req := &query.PageRequest{Reverse: true}
	for {
		pageRes, err := query.Paginate(store, req, func(key []byte, value []byte) error {
			var appsArchivedDelegations types.ApplicationsWithArchivedDelegations
			k.cdc.MustUnmarshal(value, &appsArchivedDelegations)
			k.deleteExpiredDelegations(ctx, &appsArchivedDelegations, endingSessionNumber)
			store.Delete(key)
			return nil
		})

		if err != nil {
			logger.Error("Error querying archived delegations", err)
			return err
		}

		if pageRes.NextKey == nil {
			break
		}

		req.Key = pageRes.NextKey
	}

	return nil
}

func (k Keeper) indexArchivedDelegations(
	ctx sdk.Context,
	sessionNumber int64,
	appsWithArchivedDelegations *types.ApplicationsWithArchivedDelegations,
) {
	logger := k.Logger().With("method", "indexArchivedDelegations")
	logger.Info("Indexing archived delegations")

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	archivedDelegationsKeyPrefix := types.KeyPrefix(types.ArchivedDelegationsKeyPrefix)
	store := prefix.NewStore(storeAdapter, archivedDelegationsKeyPrefix)

	store.Set(
		types.ArchivedDelegationsSessionKey(sessionNumber),
		k.cdc.MustMarshal(appsWithArchivedDelegations),
	)
}

func (k Keeper) deleteExpiredDelegations(
	ctx sdk.Context,
	appsWithArchivedDelegations *types.ApplicationsWithArchivedDelegations,
	endingSessionNumber uint64,
) {
	logger := k.Logger().With("method", "deleteExpiredDelegations")
	logger.Info("Deleting expired delegations")

	for _, appAddr := range appsWithArchivedDelegations.AppAddresses {
		foundApp, isAppFound := k.GetApplication(ctx, appAddr)
		if !isAppFound {
			logger.Warn(fmt.Sprintf("Application with address %q not found", appAddr))
		}

		oldestSessionNumberToRetain := endingSessionNumber - archivedDelegationsRetentionInSessions
		for i, archivedDelegation := range foundApp.ArchivedDelegations {
			if archivedDelegation.SessionNumber < oldestSessionNumberToRetain {
				logger.Info(fmt.Sprintf(
					`Deleting expired delegation for appAddress %q, sessionNumber "%d"`,
					appAddr,
					&archivedDelegation.SessionNumber,
				))
				foundApp.ArchivedDelegations = append(
					foundApp.ArchivedDelegations[:i],
					foundApp.ArchivedDelegations[i+1:]...,
				)
			}
		}

		k.SetApplication(ctx, foundApp)
	}
}
