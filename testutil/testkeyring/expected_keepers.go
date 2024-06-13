package testkeyring

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, types.AccAddress) types.AccountI
	SetAccount(context.Context, types.AccountI)
	// Return a new account with the next account number and the specified address. Does not save the new account to the store.
	NewAccountWithAddress(context.Context, types.AccAddress) types.AccountI
	// Fetch the next account number, and increment the internal counter.
	NextAccountNumber(context.Context) uint64
}
