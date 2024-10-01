package application

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	cosmoslog "cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testtree"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type applicationMinStakeTestSuite struct {
	suite.Suite

	ctx        context.Context
	keepers    keeper.TokenomicsModuleKeepers
	keyRing    keyring.Keyring
	ringClient crypto.RingClient

	serviceId,
	appKeyUid,
	appBech32,
	supplierKeyUid,
	supplierBech32 string
}

func TestApplicationMinStakeTestSuite(t *testing.T) {
	cmd.InitSDKConfig()

	suite.Run(t, new(applicationMinStakeTestSuite))
}

func (s *applicationMinStakeTestSuite) SetupTest() {
	s.serviceId = "svc1"
	s.appKeyUid = "app"
	s.supplierKeyUid = "supplier"
	s.keepers, s.ctx = keeper.NewTokenomicsModuleKeepers(s.T(), cosmoslog.NewNopLogger())

	s.setupRingClient()
	s.setupKeyring()
	s.setupAccounts()

	// Set block height to 1.
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(1)
}

func (s *applicationMinStakeTestSuite) TestAppCannotStakeLessThanMinStake() {
	s.T().Skip("this case is well covered in x/application/keeper/msg_server_stake_application_test.go")
}

func (s *applicationMinStakeTestSuite) TestAppIsUnbondedIfBelowMinStakeWhenSettling() {
	// Assert that the application's initial bank balance is 0.
	appBalance := s.getAppBalance()
	require.Equal(s.T(), int64(0), appBalance.Amount.Int64())

	// Add service 1
	s.addService()

	// Stake an application for service 1 with min stake.
	s.stakeApp()

	// Stake a supplier for service 1.
	s.stakeSupplier()

	// Get the session header.
	sessionHeader := s.getSessionHeader()

	// Create a claim whose settlement amount drops the application below min stake
	claim := s.getClaim(sessionHeader)

	// Process TLMs for the claim.
	err := s.keepers.Keeper.ProcessTokenLogicModules(s.ctx, claim)
	require.NoError(s.T(), err)

	// Assert that the application was unbonded.
	_, isAppFound := s.keepers.ApplicationKeeper.GetApplication(s.ctx, s.appBech32)
	require.False(s.T(), isAppFound)

	// Assert that the application's stake was returned to its bank balance.
	appBalance = s.getAppBalance()
	require.Greater(s.T(), appBalance.Amount.Int64(), int64(0))
	require.Less(s.T(), appBalance.Amount.Int64(), apptypes.DefaultMinStake.Amount.Int64())

}

// setupRingClient initializes the suite's ring client.
func (s *applicationMinStakeTestSuite) setupRingClient() {
	var err error
	deps := depinject.Supply(
		polylog.Ctx(s.ctx),
		prooftypes.NewAppKeeperQueryClient(s.keepers.ApplicationKeeper),
		prooftypes.NewAccountKeeperQueryClient(s.keepers.AccountKeeper),
		prooftypes.NewSharedKeeperQueryClient(s.keepers.SharedKeeper, s.keepers.SessionKeeper),
	)
	s.ringClient, err = rings.NewRingClient(deps)
	require.NoError(s.T(), err)
}

// setupKeyring initializes the suite's keyring.
func (s *applicationMinStakeTestSuite) setupKeyring() {
	registry := codectypes.NewInterfaceRegistry()
	registry.RegisterImplementations((*cryptotypes.PubKey)(nil), &secp256k1.PubKey{}, &ed25519.PubKey{})
	registry.RegisterImplementations((*cryptotypes.PrivKey)(nil), &secp256k1.PrivKey{}, &ed25519.PrivKey{})
	cdc := codec.NewProtoCodec(registry)
	s.keyRing = keyring.NewInMemory(cdc)
}

// setupAccounts uses pre-generated accounts to populate the necessary
// accounts in the keyring and account module state. It also populates
// the suite's addresses for the application and supplier.
func (s *applicationMinStakeTestSuite) setupAccounts() {
	acctIterator := testkeyring.PreGeneratedAccounts()
	s.appBech32 = testkeyring.CreateOnChainAccount(
		s.ctx, s.T(),
		s.appKeyUid,
		s.keyRing,
		s.keepers.AccountKeeper,
		acctIterator,
	).String()
	s.supplierBech32 = testkeyring.CreateOnChainAccount(
		s.ctx, s.T(),
		s.supplierKeyUid,
		s.keyRing,
		s.keepers.AccountKeeper,
		acctIterator,
	).String()
}

// addService adds the test service to the service module state.
func (s *applicationMinStakeTestSuite) addService() {
	s.keepers.ServiceKeeper.SetService(s.ctx, sharedtypes.Service{
		Id:                   s.serviceId,
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(), // random address.
	})
}

// stakeApp stakes an application for service 1 with min stake.
func (s *applicationMinStakeTestSuite) stakeApp() {
	s.keepers.ApplicationKeeper.SetApplication(s.ctx, apptypes.Application{
		Address:        s.appBech32,
		Stake:          &apptypes.DefaultMinStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: s.serviceId}},
	})
}

// stakeSupplier stakes a supplier for service 1.
func (s *applicationMinStakeTestSuite) stakeSupplier() {
	// TODO_NEXT(@bryanchriswhite, #612): Replace supplierStake with suppleirtypes.DefaultMinStake.
	supplierStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000) // 1 POKT.
	s.keepers.SupplierKeeper.SetSupplier(s.ctx, sharedtypes.Supplier{
		OwnerAddress:    s.supplierBech32,
		OperatorAddress: s.supplierBech32,
		Stake:           &supplierStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: s.serviceId,
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            s.supplierBech32,
						RevSharePercentage: 1,
					},
				},
			},
		},
	})
}

// getSessionHeader gets the session header for the test session.
func (s *applicationMinStakeTestSuite) getSessionHeader() *sessiontypes.SessionHeader {
	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionRes, err := s.keepers.SessionKeeper.GetSession(s.ctx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: s.appBech32,
		ServiceId:          s.serviceId,
		BlockHeight:        currentHeight,
	})
	require.NoError(s.T(), err)

	return sessionRes.GetSession().GetHeader()
}

// getClaim creates a claim whose settlement amount drops the application below min stake.
func (s *applicationMinStakeTestSuite) getClaim(
	sessionHeader *sessiontypes.SessionHeader,
) *prooftypes.Claim {
	sessionTree := testtree.NewFilledSessionTree(
		s.ctx, s.T(),
		1, 100,
		s.supplierKeyUid, s.supplierBech32,
		sessionHeader, sessionHeader, sessionHeader,
		s.keyRing, s.ringClient,
	)
	claimRoot, err := sessionTree.Flush()
	require.NoError(s.T(), err)

	return &prooftypes.Claim{
		SupplierOperatorAddress: s.supplierBech32,
		SessionHeader:           sessionHeader,
		RootHash:                claimRoot,
	}
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *applicationMinStakeTestSuite) getAppBalance() *cosmostypes.Coin {
	appBalRes, err := s.keepers.BankKeeper.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: s.appBech32, Denom: volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)

	return appBalRes.GetBalance()
}
