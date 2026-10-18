package query

import (
	"context"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/retry"
	"github.com/pokt-network/poktroll/testutil/sample"
)

// erroringAccountQueryClient fails every Account query with the given error.
type erroringAccountQueryClient struct {
	accounttypes.QueryClient
	err error
}

func (c erroringAccountQueryClient) Account(context.Context, *accounttypes.QueryAccountRequest, ...grpc.CallOption) (*accounttypes.QueryAccountResponse, error) {
	return nil, c.err
}

// Only a NotFound query status is reported as a missing account; any other
// query failure is reported as a failed query.
func TestAccQuerier_GetAccount_QueryErrors(t *testing.T) {
	tests := []struct {
		desc      string
		queryErr  error
		expectErr error
	}{
		{desc: "not found", queryErr: status.Error(codes.NotFound, "account not found"), expectErr: ErrQueryAccountNotFound},
		{desc: "unavailable", queryErr: status.Error(codes.Unavailable, "connection refused"), expectErr: ErrQueryAccountQueryFailed},
		{desc: "deadline exceeded", queryErr: context.DeadlineExceeded, expectErr: ErrQueryAccountQueryFailed},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			accountsCache, err := memory.NewKeyValueCache[cosmostypes.AccountI]()
			require.NoError(t, err)
			aq := &accQuerier{
				accountQuerier: erroringAccountQueryClient{err: test.queryErr},
				logger:         polyzero.NewLogger(),
				accountsCache:  accountsCache,
			}

			noRetry := retry.RetryStrategyFunc(func(context.Context, int) bool { return false })
			ctx := context.WithValue(context.Background(), retry.RetryStrategyCtxKey, noRetry) //nolint:staticcheck // key type is defined by pkg/retry

			_, err = aq.GetAccount(ctx, sample.AccAddressBech32())
			require.ErrorIs(t, err, test.expectErr)
		})
	}
}
