package keeper_test

import (
	"strconv"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestMorseAccountClaimMsgServerCreate(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	shannonDestAddress := "A"
	for i := 0; i < 5; i++ {
		expected := &types.MsgCreateMorseAccountClaim{ShannonDestAddress: shannonDestAddress,
			MorseSrcAddress: strconv.Itoa(i),
		}
		_, err := srv.CreateMorseAccountClaim(ctx, expected)
		require.NoError(t, err)
		rst, found := k.GetMorseAccountClaim(ctx,
			expected.MorseSrcAddress,
		)
		require.True(t, found)
		require.Equal(t, expected.ShannonDestAddress, rst.ShannonDestAddress)
	}
}

func TestMorseAccountClaimMsgServerUpdate(t *testing.T) {
	shannonDestAddress := "A"

	tests := []struct {
		desc    string
		request *types.MsgUpdateMorseAccountClaim
		err     error
	}{
		{
			desc: "Completed",
			request: &types.MsgUpdateMorseAccountClaim{ShannonDestAddress: shannonDestAddress,
				MorseSrcAddress: strconv.Itoa(0),
			},
		},
		{
			desc: "Unauthorized",
			request: &types.MsgUpdateMorseAccountClaim{ShannonDestAddress: "B",
				MorseSrcAddress: strconv.Itoa(0),
			},
			err: sdkerrors.ErrUnauthorized,
		},
		{
			desc: "KeyNotFound",
			request: &types.MsgUpdateMorseAccountClaim{ShannonDestAddress: shannonDestAddress,
				MorseSrcAddress: strconv.Itoa(100000),
			},
			err: sdkerrors.ErrKeyNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := keepertest.MigrationKeeper(t)
			srv := keeper.NewMsgServerImpl(k)
			expected := &types.MsgCreateMorseAccountClaim{ShannonDestAddress: shannonDestAddress,
				MorseSrcAddress: strconv.Itoa(0),
			}
			_, err := srv.CreateMorseAccountClaim(ctx, expected)
			require.NoError(t, err)

			_, err = srv.UpdateMorseAccountClaim(ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				rst, found := k.GetMorseAccountClaim(ctx,
					expected.MorseSrcAddress,
				)
				require.True(t, found)
				require.Equal(t, expected.ShannonDestAddress, rst.ShannonDestAddress)
			}
		})
	}
}

func TestMorseAccountClaimMsgServerDelete(t *testing.T) {
	shannonDestAddress := "A"

	tests := []struct {
		desc    string
		request *types.MsgDeleteMorseAccountClaim
		err     error
	}{
		{
			desc: "Completed",
			request: &types.MsgDeleteMorseAccountClaim{ShannonDestAddress: shannonDestAddress,
				MorseSrcAddress: strconv.Itoa(0),
			},
		},
		{
			desc: "Unauthorized",
			request: &types.MsgDeleteMorseAccountClaim{ShannonDestAddress: "B",
				MorseSrcAddress: strconv.Itoa(0),
			},
			err: sdkerrors.ErrUnauthorized,
		},
		{
			desc: "KeyNotFound",
			request: &types.MsgDeleteMorseAccountClaim{ShannonDestAddress: shannonDestAddress,
				MorseSrcAddress: strconv.Itoa(100000),
			},
			err: sdkerrors.ErrKeyNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := keepertest.MigrationKeeper(t)
			srv := keeper.NewMsgServerImpl(k)

			_, err := srv.CreateMorseAccountClaim(ctx, &types.MsgCreateMorseAccountClaim{ShannonDestAddress: shannonDestAddress,
				MorseSrcAddress: strconv.Itoa(0),
			})
			require.NoError(t, err)
			_, err = srv.DeleteMorseAccountClaim(ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				_, found := k.GetMorseAccountClaim(ctx,
					tc.request.MorseSrcAddress,
				)
				require.False(t, found)
			}
		})
	}
}
