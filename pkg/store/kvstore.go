package store

import (
	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	cosmostypes "cosmossdk.io/store/types"
	cosmosdb "github.com/cosmos/cosmos-db"
)

// TODO_IN_THIS_COMMIT: godoc...
func NewMultiStore(
	persistencePath string,
	storeTypesByStoreKey map[cosmostypes.StoreKey]cosmostypes.StoreType,
) (_ cosmostypes.MultiStore, closeFn func() error, _ error) {
	db, err := cosmosdb.NewPebbleDB("test_db", persistencePath, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO_IN_THIS_COMMIT: comment... disable logging and metrics...
	multiStore := rootmulti.NewStore(db, cosmoslog.NewNopLogger(), metrics.NewMetrics([][]string{}))

	// Mount all sub-stores.
	for storeKey, storeType := range storeTypesByStoreKey {
		multiStore.MountStoreWithDB(storeKey, storeType, db)
	}

	// Load the entire multistore.
	if err = multiStore.LoadLatestVersion(); err != nil {
		return nil, nil, err
	}

	return multiStore, db.Close, nil
}
