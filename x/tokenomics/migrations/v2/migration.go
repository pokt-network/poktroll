package v2

import (
	"context"
	"fmt"

	corestoretypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
)

// MigrateStore - NOOP migration to upgrade the consensus version.
func MigrateStore(ctx context.Context, storeService corestoretypes.KVStoreService, cdc codec.BinaryCodec) error {
	fmt.Println("Would migrate the store here, but not doing anything - we don't have anything to migrate.")
	return nil
}
