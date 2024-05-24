package keeper_test

import (
	"context"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	// tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	// tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	testServiceId = "svc1"
	testSessionId = "mock_session_id"
)
const minExecutionPeriod = 5 * time.Second

func init() {
	cmd.InitSDKConfig()
}

type TestSuite struct {
	suite.Suite

	sdkCtx  sdk.Context
	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	claim   prooftypes.Claim
	proof   prooftypes.Proof
}

func (s *TestSuite) SetupTest() {
	supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T())
	s.sdkCtx = sdk.UnwrapSDKContext(s.ctx)

	// Prepare a claim that can be inserted
	s.claim = prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			Service:                 &sharedtypes.Service{Id: testServiceId},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   sessionkeeper.GetSessionEndBlockHeight(1),
		},
		RootHash: smstRootWithSum(69),
	}

	// Prepare a claim that can be inserted
	s.proof = prooftypes.Proof{
		SupplierAddress: s.claim.SupplierAddress,
		SessionHeader:   s.claim.SessionHeader,
		// ClosestMerkleProof
	}

	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	app := apptypes.Application{
		Address: appAddr,
		Stake:   &appStake,
	}
	s.keepers.SetApplication(s.ctx, app)
}

// getClaimEvent verifies that there is exactly one event of type protoType in
// the given events and returns it. If there are 0 or more than 1 events of the
// given type, it fails the test.
func (s *TestSuite) getClaimEvent(events sdk.Events, protoType string) proto.Message {
	var parsedEvent proto.Message
	numExpectedEvents := 0
	for _, event := range events {
		switch event.Type {
		case protoType:
			var err error
			parsedEvent, err = sdk.ParseTypedEvent(abci.Event(event))
			s.Require().NoError(err)
			numExpectedEvents++
		default:
			continue
		}
	}
	if numExpectedEvents == 1 {
		return parsedEvent
	}
	require.NotEqual(s.T(), 1, numExpectedEvents, "Expected exactly one claim event")
	return nil
}
