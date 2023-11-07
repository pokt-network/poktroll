package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// InsertClaim set a specific a claim given a sessionId & supplierAddr
func (k Keeper) InsertClaim(ctx sdk.Context, claim types.Claim) {
	claimBz := k.cdc.MustMarshal(&claim)
	parentStore := ctx.KVStore(k.storeKey)

	// Store the whole claim in the primary key store
	primaryStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	primaryKey := types.ClaimPrimaryKey(claim)
	primaryStore.Set(primaryKey, claimBz)

	// Save the claim in the main claim store with primary key
	// primaryStore := prefix.NewStore(ctx.KVStore(k.storeKey), []byte("claim"))
	// primaryStore.Set(claim.PrimaryKey(), claim.Marshal())

	// Store the param
	// heightStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimHeightPrefix))
	// heightKey := HeightKey(claim.Height) // Serialize height into a byte slice if needed
	// heightStore.Set(heightKey, claim.PrimaryKey())

	// Index by address
	addressStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimHeightPrefix))
	addressKey := types.ClaimSupplierAddressKey(claim.SupplierAddress)
	addressStoreIndex.Set(addressKey, primaryKey)

	// ClaimHeightPrefix
	// ClaimAddressPrefix
	// ClaimSessionIdPrefix

}

// GetClaim returns a claim given a sessionId & supplierAddr
func (k Keeper) GetClaim(
	ctx sdk.Context,
	sessionId, supplierAddr string,

) (val types.Claim, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimKeyPrefix))

	b := store.Get(types.ClaimKey(
		sessionId, supplierAddr,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveClaim removes a claim from the store
// func (k Keeper) RemoveClaim(
// 	ctx sdk.Context,
// 	supplierAddr string,
// 	sessionId,

// ) {
// 	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
// 	store.Delete(types.ClaimKey(
// 		sessionId, supplierAddr,
// 	))
// }

// GetAllClaims returns all claim
func (k Keeper) GetAllClaims(ctx sdk.Context) (list []types.Claim) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Claim
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// When retrieving by height:
// func (k Keeper) GetClaimsByHeight(ctx sdk.Context, height int64) []Claim {
//     heightStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimHeightPrefix))
//     heightKey := HeightKey(height) // Serialize height into a byte slice if needed

//     // Iterate over the height index store using heightKey
//     iterator := sdk.KVStorePrefixIterator(heightStore, heightKey)
//     defer iterator.Close()

//     var claims []Claim
//     for ; iterator.Valid(); iterator.Next() {
//         primaryKey := iterator.Value()
//         claim := GetClaimByPrimaryKey(ctx, claimStoreKey, primaryKey)
//         claims = append(claims, claim)
//     }

//     return claims
// }

// When retrieving by address:
func (k Keeper) GetClaimsByAddress(ctx sdk.Context, address sdk.AccAddress) []Claim {
	addressStore := prefix.NewStore(ctx.KVStore(addressIndexStoreKey), types.KeyPrefix(types.ClaimAddressPrefix))
	addressKey := AddressKey(address) // Serialize address into a byte slice if needed

	// Iterate over the address index store using addressKey
	iterator := sdk.KVStorePrefixIterator(addressStore, addressKey)
	defer iterator.Close()

	var claims []Claim
	for ; iterator.Valid(); iterator.Next() {
		primaryKey := iterator.Value()
		claim := GetClaimByPrimaryKey(ctx, claimStoreKey, primaryKey)
		claims = append(claims, claim)
	}

	return claims
}

// // Helper function to get a claim by primary key:
func GetClaimByPrimaryKey(ctx sdk.Context, claimStoreKey sdk.StoreKey, primaryKey []byte) Claim {
	primaryStore := prefix.NewStore(ctx.KVStore(claimStoreKey), []byte("claim"))
	byteClaim := primaryStore.Get(primaryKey)
	var claim Claim
	claim.Unmarshal(byteClaim) // Unmarshal byte slice into Claim object
	return claim
}

// GetOlshCount get the total number of olsh
// func (k Keeper) GetOlshCount(ctx sdk.Context) uint64 {
// 	store :=  prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
// 	byteKey := types.KeyPrefix(types.OlshCountKey)
// 	bz := store.Get(byteKey)

// 	// Count doesn't exist: no element
// 	if bz == nil {
// 		return 0
// 	}

// 	// Parse bytes
// 	return binary.BigEndian.Uint64(bz)
// }

// // SetOlshCount set the total number of olsh
// func (k Keeper) SetOlshCount(ctx sdk.Context, count uint64)  {
// 	store :=  prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
// 	byteKey := types.KeyPrefix(types.OlshCountKey)
// 	bz := make([]byte, 8)
// 	binary.BigEndian.PutUint64(bz, count)
// 	store.Set(byteKey, bz)
// }

// // AppendOlsh appends a olsh in the store with a new id and update the count
// func (k Keeper) AppendOlsh(
//     ctx sdk.Context,
//     olsh types.Olsh,
// ) uint64 {
// 	// Create the olsh
//     count := k.GetOlshCount(ctx)

//     // Set the ID of the appended value
//     olsh.Id = count

//     store :=  prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.OlshKey))
//     appendedValue := k.cdc.MustMarshal(&olsh)
//     store.Set(GetOlshIDBytes(olsh.Id), appendedValue)

//     // Update olsh count
//     k.SetOlshCount(ctx, count+1)

//     return count
// }
