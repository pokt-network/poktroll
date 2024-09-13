package params

import (
	"reflect"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	msgUpdateParamsName = "MsgUpdateParams"
	msgUpdateParamName  = "MsgUpdateParam"
)

var (
	authorityAddr,
	authorizedAddr,
	unauthorizedAddr cosmostypes.AccAddress

	validSharedParams = sharedtypes.Params{
		NumBlocksPerSession:                5,
		GracePeriodEndOffsetBlocks:         2,
		ClaimWindowOpenOffsetBlocks:        5,
		ClaimWindowCloseOffsetBlocks:       5,
		ProofWindowOpenOffsetBlocks:        2,
		ProofWindowCloseOffsetBlocks:       5,
		SupplierUnbondingPeriodSessions:    9,
		ApplicationUnbondingPeriodSessions: 9,
	}

	validSessionParams = sessiontypes.Params{}

	validServiceFeeCoin = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 42000000000)
	validServiceParams  = servicetypes.Params{
		AddServiceFee: &validServiceFeeCoin,
	}

	validApplicationParams = apptypes.Params{
		MaxDelegatedGateways: 10,
	}

	validGatewayParams  = gatewaytypes.Params{}
	validSupplierParams = suppliertypes.Params{}

	validMissingPenaltyCoin = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 4200)
	validSubmissionFeeCoin  = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 42000000)
	validProofParams        = prooftypes.Params{
		RelayDifficultyTargetHash: prooftypes.DefaultRelayDifficultyTargetHash,
		ProofRequestProbability:   1,
		ProofRequirementThreshold: 0,
		ProofMissingPenalty:       &validMissingPenaltyCoin,
		ProofSubmissionFee:        &validSubmissionFeeCoin,
	}

	validTokenomicsParams = tokenomicstypes.Params{
		ComputeUnitsToTokensMultiplier: 5,
	}

	// NB: Authority fields are intentionally omitted.
	// To be added to a **copy** by the test.
	msgUpdateParamsByModule = map[string]cosmostypes.Msg{
		sharedtypes.ModuleName: &sharedtypes.MsgUpdateParams{
			Params: validSharedParams,
		},
		sessiontypes.ModuleName: &sessiontypes.MsgUpdateParams{
			Params: validSessionParams,
		},
		servicetypes.ModuleName: &servicetypes.MsgUpdateParams{
			Params: validServiceParams,
		},
		apptypes.ModuleName: &apptypes.MsgUpdateParams{
			Params: validApplicationParams,
		},
		gatewaytypes.ModuleName: &gatewaytypes.MsgUpdateParams{
			Params: validGatewayParams,
		},
		suppliertypes.ModuleName: &suppliertypes.MsgUpdateParams{
			Params: validSupplierParams,
		},
		prooftypes.ModuleName: &prooftypes.MsgUpdateParams{
			Params: validProofParams,
		},
		tokenomicstypes.ModuleName: &tokenomicstypes.MsgUpdateParams{
			Params: validTokenomicsParams,
		},
	}

	_ suites.IntegrationSuite = (*UpdateParamsSuite)(nil)
)

type UpdateParamsSuite struct {
	suites.AuthzIntegrationSuite
}

// SetupTest runs before each test in the suite.
func (s *UpdateParamsSuite) SetupTest() {
	// Construct a fresh integration app for each test.
	// Set the authority, authorized, and unauthorized addresses.
	authorityAddr = cosmostypes.MustAccAddressFromBech32(s.GetApp().GetAuthority())

	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	authorizedAddr = nextAcct.Address

	nextAcct, ok = s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	unauthorizedAddr = nextAcct.Address

	// Create authz grants for all poktroll modules' MsgUpdateParams messages.
	s.SendAuthzGrantMsgForPoktrollModules(
		authorityAddr,
		authorizedAddr,
		msgUpdateParamsName,
		s.GetModuleNames()...,
	)
}

func (s *UpdateParamsSuite) TestUnauthorizedMsgUpdateParamsFails() {
	for _, moduleName := range s.GetModuleNames() {
		s.T().Run(moduleName, func(t *testing.T) {
			// Assert that the module's params are set to their default values.
			// TODO_IN_THIS_COMMIT: consider whether/how to do this. Seems
			// to require a query client instance for each module.

			//msgStruct, isMsgTypeFound := msgUpdateParamsByModule[moduleName]
			msgIface, isMsgTypeFound := msgUpdateParamsByModule[moduleName]
			require.Truef(s.T(), isMsgTypeFound, "unknown message type for module %q", moduleName)

			msgValue := reflect.ValueOf(msgIface)
			msgType := msgValue.Elem().Type()

			// Copy the message and set the authority field.
			msgValueCopy := reflect.New(msgType)
			msgValueCopy.Elem().Set(msgValue.Elem())
			msgValueCopy.Elem().
				FieldByName("Authority").
				SetString(authorityAddr.String())

			msgUpdateParams := msgValueCopy.Interface().(cosmostypes.Msg)

			msgTypeName := proto.MessageName(msgUpdateParams)
			s.T().Logf("msgType: %s", msgType.Name())
			s.T().Logf("msgTypeName: %s", msgTypeName)

			// Set up assertion that the MsgExec will fail.
			errAssertionOpt := integration.WithErrorAssertion(
				func(err error) {
					require.ErrorIs(t, err, authz.ErrNoAuthorizationFound)
				},
			)

			// Send an authz MsgExec from an unauthorized address.
			runOpts := integration.RunUntilNextBlockOpts.Append(errAssertionOpt)
			execMsg := authz.NewMsgExec(unauthorizedAddr, []cosmostypes.Msg{msgUpdateParams})
			anyRes := s.GetApp().RunMsg(s.T(), &execMsg, runOpts...)
			require.Nil(s.T(), anyRes)
		})
	}
}

func TestUpdateParamsSuite(t *testing.T) {
	suite.Run(t, &UpdateParamsSuite{})
}
