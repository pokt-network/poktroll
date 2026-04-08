package token_logic_module

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGetSupplierShareholderAmountMap_UniqueAddresses(t *testing.T) {
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1addr1", RevSharePercentage: 30},
		{Address: "pokt1addr2", RevSharePercentage: 70},
	}
	amountToDistribute := math.NewInt(1000)

	shareMap := GetSupplierShareholderAmountMap(revShare, amountToDistribute)

	require.Len(t, shareMap, 2)
	require.Equal(t, math.NewInt(300), shareMap["pokt1addr1"])
	require.Equal(t, math.NewInt(700), shareMap["pokt1addr2"])

	// Verify total distributed equals input
	total := math.NewInt(0)
	for _, amt := range shareMap {
		total = total.Add(amt)
	}
	require.Equal(t, amountToDistribute, total)
}

func TestGetSupplierShareholderAmountMap_DuplicateAddresses(t *testing.T) {
	// Same address appears twice — map should contain only one key.
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1duplicate", RevSharePercentage: 10},
		{Address: "pokt1duplicate", RevSharePercentage: 90},
	}
	amountToDistribute := math.NewInt(1000)

	shareMap := GetSupplierShareholderAmountMap(revShare, amountToDistribute)

	// Map deduplicates: only 1 unique address
	require.Len(t, shareMap, 1)
	require.Contains(t, shareMap, "pokt1duplicate")

	// Second entry (90%) overwrites first (10%) in the map. Final value = 900.
	require.Equal(t, math.NewInt(900), shareMap["pokt1duplicate"])
}

func TestGetSupplierShareholderAmountMap_DuplicateWithMixedAddresses(t *testing.T) {
	// Mixed unique and duplicate addresses: [{operator:15}, {owner:15}, {owner:70}]
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1operator", RevSharePercentage: 15},
		{Address: "pokt1owner", RevSharePercentage: 15},
		{Address: "pokt1owner", RevSharePercentage: 70},
	}
	amountToDistribute := math.NewInt(1000)

	shareMap := GetSupplierShareholderAmountMap(revShare, amountToDistribute)

	// 2 unique addresses
	require.Len(t, shareMap, 2)
	require.Equal(t, math.NewInt(150), shareMap["pokt1operator"])
	// owner: first write 150, overwritten to 700. Remainder = 0. Final = 700.
	require.Equal(t, math.NewInt(700), shareMap["pokt1owner"])
}

func TestGetSupplierShareholderAmountMap_Remainder(t *testing.T) {
	// 3 shareholders splitting 100 uPOKT at 33/33/34 — tests remainder allocation
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1a", RevSharePercentage: 33},
		{Address: "pokt1b", RevSharePercentage: 33},
		{Address: "pokt1c", RevSharePercentage: 34},
	}
	amountToDistribute := math.NewInt(100)

	shareMap := GetSupplierShareholderAmountMap(revShare, amountToDistribute)

	require.Len(t, shareMap, 3)

	// Verify total distributed equals input (remainder goes to first shareholder)
	total := math.NewInt(0)
	for _, amt := range shareMap {
		total = total.Add(amt)
	}
	require.Equal(t, amountToDistribute, total)
}
