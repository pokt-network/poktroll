package types_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/types"
)

// panickingAccountKeeper panics on every account read, like an account keeper
// reading an account it cannot decode.
type panickingAccountKeeper struct {
	types.AccountKeeper
}

func (panickingAccountKeeper) GetAccount(context.Context, sdk.AccAddress) sdk.AccountI {
	panic("cannot decode account")
}

// A panic while reading an account is returned as an error.
func TestAccountKeeperQueryClient_AccountReadPanicReturnsError(t *testing.T) {
	accountQuerier := types.NewAccountKeeperQueryClient(panickingAccountKeeper{}, log.NewNopLogger())

	_, err := accountQuerier.GetPubKeyFromAddress(context.Background(), sample.AccAddressBech32())
	require.ErrorIs(t, err, types.ErrProofPubKeyNotFound)
	require.ErrorContains(t, err, "cannot decode account")
}
