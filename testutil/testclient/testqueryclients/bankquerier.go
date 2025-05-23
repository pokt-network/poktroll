package testqueryclients

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/mockclient"
)

// NewTestBankQueryClientWithWithBalance creates a mock of the BankQueryClient that
// uses the provided balance for its GetBalance() method implementation.
func NewTestBankQueryClientWithBalance(t *testing.T, balance int64) *mockclient.MockBankQueryClient {
	ctrl := gomock.NewController(t)
	bankQueryClientMock := mockclient.NewMockBankQueryClient(ctrl)
	bankQueryClientMock.EXPECT().
		GetBalance(gomock.Any(), gomock.Any()).
		Return(&sdk.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(balance)}, nil).
		AnyTimes()

	return bankQueryClientMock
}
