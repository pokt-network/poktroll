package token_logic_module

import (
	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)



// createTestTLMContext creates a TLMContext for testing validator reward distribution
func createTestTLMContext(
	settlementAmount int64,
	globalInflationPercent float64,
	proposerAllocationPercent float64,
	stakingKeeper tokenomicstypes.StakingKeeper,
) TLMContext {
	// Create test objects
	service := &sharedtypes.Service{
		Id:                   "test-service",
		Name:                 "Test Service",
		ComputeUnitsPerRelay: 100,
		OwnerAddress:         sample.AccAddressBech32(),
	}

	application := &apptypes.Application{
		Address: sample.AccAddressBech32(),
		Stake: func() *cosmostypes.Coin {
			c := cosmostypes.NewCoin(pocket.DenomuPOKT, cosmosmath.NewInt(1000000))
			return &c
		}(),
	}

	supplier := &sharedtypes.Supplier{
		OperatorAddress: sample.AccAddressBech32(),
		Stake: func() *cosmostypes.Coin {
			c := cosmostypes.NewCoin(pocket.DenomuPOKT, cosmosmath.NewInt(1000000))
			return &c
		}(),
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: service.Id,
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            sample.AccAddressBech32(),
						RevSharePercentage: 100,
					},
				},
			},
		},
	}

	sessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress: application.Address,
		ServiceId:          service.Id,
		SessionId:          "test-session",
	}

	relayMiningDifficulty := &servicetypes.RelayMiningDifficulty{
		ServiceId:    service.Id,
		BlockHeight:  1000,
		NumRelaysEma: 1000,
		TargetHash:   []byte("test-target-hash"),
	}

	// Create tokenomics parameters
	tokenomicsParams := tokenomicstypes.Params{
		GlobalInflationPerClaim: globalInflationPercent,
		MintAllocationPercentages: tokenomicstypes.MintAllocationPercentages{
			Dao:         0.1,
			Proposer:    proposerAllocationPercent,
			Supplier:    0.6,
			SourceOwner: 0.2,
			Application: 0.0,
		},
		DaoRewardAddress: sample.AccAddressBech32(),
	}

	// Create settlement coin
	settlementCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, cosmosmath.NewInt(settlementAmount))

	// Create settlement result
	result := &tokenomicstypes.ClaimSettlementResult{}

	return TLMContext{
		TokenomicsParams:      tokenomicsParams,
		SettlementCoin:        settlementCoin,
		SessionHeader:         sessionHeader,
		Result:                result,
		Service:               service,
		Application:           application,
		Supplier:              supplier,
		RelayMiningDifficulty: relayMiningDifficulty,
		StakingKeeper:         stakingKeeper,
	}
}

