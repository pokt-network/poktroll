package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func init() {
	cmd.InitSDKConfig()
}

type TestSuiteRelayMining struct {
	suite.Suite

	sdkCtx  sdk.Context
	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	// claim   prooftypes.Claim
	// proof   prooftypes.Proof
}

func (s *TestSuiteRelayMining) SetupTest() {
	// supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T())
	s.sdkCtx = sdk.UnwrapSDKContext(s.ctx)

	// Prepare a claim that can be inserted
	// s.claim = prooftypes.Claim{
	// 	SupplierAddress: supplierAddr,
	// 	SessionHeader: &sessiontypes.SessionHeader{
	// 		ApplicationAddress:      appAddr,
	// 		Service:                 &sharedtypes.Service{Id: testServiceId},
	// 		SessionId:               "session_id",
	// 		SessionStartBlockHeight: 1,
	// 		SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
	// 	},
	// 	RootHash: smstRootWithSum(69),
	// }

	// // Prepare a claim that can be inserted
	// s.proof = prooftypes.Proof{
	// 	SupplierAddress: s.claim.SupplierAddress,
	// 	SessionHeader:   s.claim.SessionHeader,
	// 	// ClosestMerkleProof
	// }

	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	app := apptypes.Application{
		Address: appAddr,
		Stake:   &appStake,
	}
	s.keepers.SetApplication(s.ctx, app)
}

func TestUpdateRelayMiningDifficulty(t *testing.T) {
	suite.Run(t, new(TestSuiteRelayMining))
}

func (s *TestSuiteRelayMining) TestUpdateRelayMiningDifficulty_NewServiceSeenForTheFirstTime() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	// sdkCtx := sdk.UnwrapSDKContext(ctx)

	// // 0. Add the claim & verify it exists
	// claim := s.claim
	// s.keepers.UpsertClaim(ctx, claim)
	// claims := s.keepers.GetAllClaims(ctx)
	// s.Require().Len(claims, 1)

	// Verify there are no relay mining difficulties
	allDifficulties := s.keepers.GetAllRelayMiningDifficulty(ctx)
	require.Len(t, allDifficulties, 0)

	relaysPerServiceMap := map[string]uint64{
		"service1": 10,
	}

	s.keepers.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)

	// Verify there are no relay mining difficulties
	allDifficulties = s.keepers.GetAllRelayMiningDifficulty(ctx)
	require.Len(t, allDifficulties, 0)

	// s.keepers.GetRelayMiningDifficulty(ctx, testServiceId)
}

func (s *TestSuiteRelayMining) UpdateRelayMiningDifficulty_UpdatingMultipleServicesAtOnce() {

}

func (s *TestSuiteRelayMining) UpdateRelayMiningDifficulty_UpdateServiceIsNotSeenForAWhile() {
}

func (s *TestSuiteRelayMining) UpdateRelayMiningDifficulty_UpdateServiceIsIncreasing() {
}

func (s *TestSuiteRelayMining) UpdateRelayMiningDifficulty_UpdateServiceIsDecreasing() {
}
