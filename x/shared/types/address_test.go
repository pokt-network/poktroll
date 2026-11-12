package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestValidateCanonicalAddress(t *testing.T) {
	address := sample.AccAddressBech32()
	require.NoError(t, ValidateCanonicalAddress(address))
	require.Error(t, ValidateCanonicalAddress(strings.ToUpper(address)))
	require.Error(t, ValidateCanonicalAddress("invalid_address"))

	revShare := []*ServiceRevenueShare{{Address: strings.ToUpper(address), RevSharePercentage: 100}}
	require.ErrorIs(t, ValidateServiceRevShare(revShare), ErrSharedInvalidRevShare)

	service := Service{Id: "svc", Name: "svc", ComputeUnitsPerRelay: 1, OwnerAddress: strings.ToUpper(address)}
	require.ErrorIs(t, service.ValidateBasic(), ErrSharedInvalidService)
	service.OwnerAddress = address
	require.NoError(t, service.ValidateBasic())
}
