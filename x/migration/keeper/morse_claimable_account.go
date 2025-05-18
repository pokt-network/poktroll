package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// Set a specific MorseClaimableAccount in the store by its index
// - Overwrites any existing account with the same index
func (k Keeper) SetMorseClaimableAccount(ctx context.Context, morseClaimableAccount migrationtypes.MorseClaimableAccount) {
	// Store the MorseClaimableAccount.
	mcaStore := k.getMorseClaimableAccountStore(ctx)
	morseClaimableAccountBz := k.cdc.MustMarshal(&morseClaimableAccount)
	mcaStore.Set(migrationtypes.MorseClaimableAccountKey(
		morseClaimableAccount.MorseSrcAddress,
	), morseClaimableAccountBz)

	// Index the MorseClaimableAccount by relevant fields.
	k.indexMorseClaimableAccountMorseOutputAddress(ctx, morseClaimableAccount)
	k.indexMorseClaimableAccountShannonDestAddress(ctx, morseClaimableAccount)
}

// Get a MorseClaimableAccount by its index
// - Returns (account, true) if found
// - Returns (zero value, false) if not found
func (k Keeper) GetMorseClaimableAccount(
	ctx context.Context,
	address string,

) (morseClaimableAccount migrationtypes.MorseClaimableAccount, found bool) {
	mcaStore := k.getMorseClaimableAccountStore(ctx)
	morseClaimableAccountBz := mcaStore.Get(migrationtypes.MorseClaimableAccountKey(
		address,
	))
	if morseClaimableAccountBz == nil {
		return morseClaimableAccount, false
	}

	k.cdc.MustUnmarshal(morseClaimableAccountBz, &morseClaimableAccount)
	return morseClaimableAccount, true
}

// Remove ALL MorseClaimableAccounts from the store
// - ONLY call during (re-)import/overwrite
// - Overwrite import should ONLY be enabled on Alpha/Beta TestNets
// - Controlled by `allow_morse_account_import_overwrite` migration param
func (k Keeper) resetMorseClaimableAccounts(
	ctx context.Context,
) {
	mcaStore := k.getMorseClaimableAccountStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(mcaStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		morseSrcAddressKey := iterator.Value()
		mcaStore.Delete(morseSrcAddressKey)
	}

	k.resetMorseClaimableAccountMorseOutputAddressIndex(ctx)
	k.resetMorseClaimableAccountShannonDestAddressIndex(ctx)
}

// Get all MorseClaimableAccounts in the store
// - Returns a slice of all accounts
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

// Check if there are ANY MorseClaimableAccounts in the store
// - Returns true if at least one account exists
func (k Keeper) HasAnyMorseClaimableAccounts(ctx context.Context) bool {
	mcaStore := k.getMorseClaimableAccountStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(mcaStore, []byte{})

	defer iterator.Close()

	return iterator.Valid()
}

// Import MorseClaimableAccounts from the given MorseAccountState
// - Assumes MorseAccountState is already validated
// DEV_NOTE: All imported accounts are set to unclaimed
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

// Mint the given coinToMint to destAddress
// - Mints to migration module account
// - Sends minted coins to destAddress
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

// getMorseClaimableAccountStore returns a prefix.Store for the MorseClaimableAccounts.
func (k Keeper) getMorseClaimableAccountStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, migrationtypes.KeyPrefix(migrationtypes.MorseClaimableAccountKeyPrefix))
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (k Keeper) getMorseClaimableAccountMorseOutputAddressStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, migrationtypes.KeyPrefix(migrationtypes.MorseClaimableAccountMorseOutputAddressKeyPrefix))
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (k Keeper) getMorseClaimableAccountShannonDestAddressStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, migrationtypes.KeyPrefix(migrationtypes.MorseClaimableAccountShannonDestAddressKeyPrefix))
}

// TODO_IN_THIS_COMMIT: godoc...
func (k Keeper) indexMorseClaimableAccountMorseOutputAddress(ctx context.Context, morseClaimableAccount migrationtypes.MorseClaimableAccount) {
	mcaMorseOutputAddressStore := k.getMorseClaimableAccountMorseOutputAddressStore(ctx)

	morseSrcAddressKey := migrationtypes.MorseClaimableAccountKey(morseClaimableAccount.GetMorseSrcAddress())
	morseOutputAddressKey := migrationtypes.MorseClaimableAccountMorseOutputAddressKey(morseClaimableAccount)
	mcaMorseOutputAddressStore.Set(morseOutputAddressKey, morseSrcAddressKey)
}

// TODO_IN_THIS_COMMIT: godoc...
func (k Keeper) indexMorseClaimableAccountShannonDestAddress(ctx context.Context, morseClaimableAccount migrationtypes.MorseClaimableAccount) {
	shannonDestAddressStore := k.getMorseClaimableAccountShannonDestAddressStore(ctx)

	morseSrcAddressKey := migrationtypes.MorseClaimableAccountKey(morseClaimableAccount.GetMorseSrcAddress())
	shannonDestAddressKey := migrationtypes.MorseClaimableAccountShannonDestAddressKey(morseClaimableAccount)
	shannonDestAddressStore.Set(shannonDestAddressKey, morseSrcAddressKey)
}

// TODO_IN_THIS_COMMIT: godoc...
func (k Keeper) resetMorseClaimableAccountMorseOutputAddressIndex(ctx context.Context) {
	mcaMorseOutputAddressStore := k.getMorseClaimableAccountMorseOutputAddressStore(ctx)

	iterator := storetypes.KVStorePrefixIterator(mcaMorseOutputAddressStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		morseSrcAddressKey := iterator.Value()
		mcaMorseOutputAddressStore.Delete(morseSrcAddressKey)
	}
}

// TODO_IN_THIS_COMMIT: godoc...
func (k Keeper) resetMorseClaimableAccountShannonDestAddressIndex(ctx context.Context) {
	mcaShannonDestAddressStore := k.getMorseClaimableAccountShannonDestAddressStore(ctx)

	iterator := storetypes.KVStorePrefixIterator(mcaShannonDestAddressStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		morseSrcAddressKey := iterator.Value()
		mcaShannonDestAddressStore.Delete(morseSrcAddressKey)
	}
}
