package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// SetMorseClaimableAccount set a specific morseClaimableAccount in the store from its index
func (k Keeper) SetMorseClaimableAccount(ctx context.Context, morseClaimableAccount migrationtypes.MorseClaimableAccount) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, migrationtypes.KeyPrefix(migrationtypes.MorseClaimableAccountKeyPrefix))
	morseClaimableAccountBz := k.cdc.MustMarshal(&morseClaimableAccount)
	store.Set(migrationtypes.MorseClaimableAccountKey(
		morseClaimableAccount.MorseSrcAddress,
	), morseClaimableAccountBz)
}

// GetMorseClaimableAccount returns a morseClaimableAccount from its index
func (k Keeper) GetMorseClaimableAccount(
	ctx context.Context,
	address string,

) (morseClaimableAccount migrationtypes.MorseClaimableAccount, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, migrationtypes.KeyPrefix(migrationtypes.MorseClaimableAccountKeyPrefix))

	morseClaimableAccountBz := store.Get(migrationtypes.MorseClaimableAccountKey(
		address,
	))
	if morseClaimableAccountBz == nil {
		return morseClaimableAccount, false
	}

	k.cdc.MustUnmarshal(morseClaimableAccountBz, &morseClaimableAccount)
	return morseClaimableAccount, true
}

// resetMorseClaimableAccounts removes ALL morseClaimableAccount from the store.
// SHOULD ONLY be called during (re-)import/overwrite of the MorseClaimableAccounts.
// Import overwriting SHOULD ONLY be enabled on Alpha and Beta TestNets, and is
// controlled by the `allow_morse_account_import_overwrite` migration module param.
func (k Keeper) resetMorseClaimableAccounts(
	ctx context.Context,
) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, migrationtypes.KeyPrefix(migrationtypes.MorseClaimableAccountKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		morseSrcAddressKey := iterator.Value()
		store.Delete(morseSrcAddressKey)
	}
}

// GetAllMorseClaimableAccounts returns all morseClaimableAccount
func (k Keeper) GetAllMorseClaimableAccounts(ctx context.Context) (list []migrationtypes.MorseClaimableAccount) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, migrationtypes.KeyPrefix(migrationtypes.MorseClaimableAccountKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var morseClaimableAccount migrationtypes.MorseClaimableAccount
		k.cdc.MustUnmarshal(iterator.Value(), &morseClaimableAccount)
		list = append(list, morseClaimableAccount)
	}

	return
}

// HasAnyMorseClaimableAccounts returns true if there are any MorseClaimableAccounts in the store.
func (k Keeper) HasAnyMorseClaimableAccounts(ctx context.Context) bool {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, migrationtypes.KeyPrefix(migrationtypes.MorseClaimableAccountKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	return iterator.Valid()
}

// ImportFromMorseAccountState imports the MorseClaimableAccounts from the given MorseAccountState.
// DEV_NOTE: It assumes that the MorseAccountState has already been validated.
func (k Keeper) ImportFromMorseAccountState(
	ctx context.Context,
	morseAccountState *migrationtypes.MorseAccountState,
) {
	for _, morseAccount := range morseAccountState.Accounts {
		// DEV_NOTE: Ensure all MorseClaimableAccounts are initially unclaimed.
		morseAccount.ClaimedAtHeight = 0
		k.SetMorseClaimableAccount(ctx, *morseAccount)
	}
}

// MintClaimedMorseTokens mints the given coinToMint to the given destAddress.
func (k Keeper) MintClaimedMorseTokens(
	ctx context.Context,
	destAddress cosmostypes.AccAddress,
	coinToMint cosmostypes.Coin,
) error {
	// Mint coinToMint to the migration module account.
	if err := k.bankKeeper.MintCoins(
		ctx,
		migrationtypes.ModuleName,
		cosmostypes.NewCoins(coinToMint),
	); err != nil {
		return err
	}

	// Transfer the coinToMint to the shannonDestAddress account.
	return k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		migrationtypes.ModuleName,
		destAddress,
		cosmostypes.NewCoins(coinToMint),
	)
}
