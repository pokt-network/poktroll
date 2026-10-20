package types

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
)

func TestValidateMinStakeIncludedServices(t *testing.T) {
	require.NoError(t, ValidateMinStakeIncludedServices(uint64(1)))
	require.NoError(t, ValidateMinStakeIncludedServices(uint64(1000)))
	require.ErrorIs(t, ValidateMinStakeIncludedServices(uint64(0)), ErrSupplierParamInvalid)
	require.ErrorIs(t, ValidateMinStakeIncludedServices("1"), ErrSupplierParamInvalid)
}

func TestValidateStakePerAdditionalService(t *testing.T) {
	zero := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0)
	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 5_000_000_000)
	wrongDenom := cosmostypes.NewInt64Coin("stake", 1)
	require.NoError(t, ValidateStakePerAdditionalService(&zero))
	require.NoError(t, ValidateStakePerAdditionalService(&price))
	require.ErrorIs(t, ValidateStakePerAdditionalService(&wrongDenom), ErrSupplierParamInvalid)
	require.ErrorIs(t, ValidateStakePerAdditionalService((*cosmostypes.Coin)(nil)), ErrSupplierParamInvalid)
}

func TestDefaultParamsValidate(t *testing.T) {
	require.NoError(t, DefaultParams().Validate())
	params := DefaultParams()
	params.MinStakeIncludedServices = 0
	require.ErrorIs(t, params.Validate(), ErrSupplierParamInvalid)
}
