package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// TODO_IN_THIS_PR:
// - Need to trigger EndBlocker
// - Need to verify its idempotent
// - Investigate how to get stats
// - Look into the guage
// - Look into adding a dashboard
// - Look into updating the SMT for this

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
	// 	RootHash: testproof.SmstRootWithSum(69),
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
}

func (s *TestSuiteRelayMining) UpdateRelayMiningDifficulty_UpdatingMultipleServicesAtOnce() {

}

func (s *TestSuiteRelayMining) UpdateRelayMiningDifficulty_UpdateServiceIsNotSeenForAWhile() {
}

func (s *TestSuiteRelayMining) UpdateRelayMiningDifficulty_UpdateServiceIsIncreasing() {
}

func (s *TestSuiteRelayMining) UpdateRelayMiningDifficulty_UpdateServiceIsDecreasing() {
}
