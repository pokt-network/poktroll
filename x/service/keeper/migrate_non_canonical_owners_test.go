package keeper_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestCanonicalizeServiceOwnerAddresses(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)

	owner, otherOwner := sample.AccAddressBech32(), sample.AccAddressBech32()
	k.SetService(ctx, sharedtypes.Service{Id: "svc1", Name: "svc1", ComputeUnitsPerRelay: 1, OwnerAddress: strings.ToUpper(owner)})
	k.SetService(ctx, sharedtypes.Service{Id: "svc2", Name: "svc2", ComputeUnitsPerRelay: 1, OwnerAddress: otherOwner})

	require.Equal(t, 1, k.CanonicalizeServiceOwnerAddresses(ctx))

	service, found := k.GetService(ctx, "svc1")
	require.True(t, found)
	require.Equal(t, owner, service.OwnerAddress)
	service, found = k.GetService(ctx, "svc2")
	require.True(t, found)
	require.Equal(t, otherOwner, service.OwnerAddress)

	// The owner can transfer the service again.
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.TransferService(ctx, &servicetypes.MsgTransferService{OwnerAddress: owner, ServiceId: "svc1", NewOwnerAddress: otherOwner})
	require.NoError(t, err)
}
