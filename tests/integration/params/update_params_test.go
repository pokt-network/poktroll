package params

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration"
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
	// poktrollMsgTypeFormat is the format for a poktroll module's message type.
	// The first %s is the module name, and the second %s is the message name.
	poktrollMsgTypeFormat = "/poktroll.%s.%s"
	msgUpdateParamsName   = "MsgUpdateParams"
	msgUpdateParamName    = "MsgUpdateParam"
)

var (
	authorityAddr,
	authorizedAddr,
	unauthorizedAddr cosmostypes.AccAddress
	allPoktrollModuleNames = []string{
		sharedtypes.ModuleName,
		sessiontypes.ModuleName,
		servicetypes.ModuleName,
		apptypes.ModuleName,
		gatewaytypes.ModuleName,
		suppliertypes.ModuleName,
		prooftypes.ModuleName,
		tokenomicstypes.ModuleName,
	}
	authzGrantExpiration = time.Now().Add(time.Hour)

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

	runUntilNextBlockOpts = []integration.RunOption{
		integration.WithAutomaticCommit(),
		integration.WithAutomaticFinalizeBlock(),
	}
)

type UpdateParamsSuite struct {
	suite.Suite
	app *integration.App
}

// SetupTest runs before each test in the suite.
func (s *UpdateParamsSuite) SetupTest() {
	// Construct a fresh integration app for each test.
	s.app = integration.NewCompleteIntegrationApp(s.T())

	// Set the authority, authorized, and unauthorized addresses.
	authorityAddr = cosmostypes.MustAccAddressFromBech32(s.app.GetAuthority())

	nextAcct, ok := s.app.GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	authorizedAddr = nextAcct.Address

	nextAcct, ok = s.app.GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	unauthorizedAddr = nextAcct.Address

	// Create authz grants for all poktroll modules' MsgUpdateParams messages.
	s.sendAuthzGrantMsgForPoktrollModules(
		authorityAddr,
		authorizedAddr,
		msgUpdateParamsName,
		allPoktrollModuleNames...,
	)
}

func (s *UpdateParamsSuite) TestUnauthorizedMsgUpdateParamsFails() {
	for _, moduleName := range allPoktrollModuleNames {
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
			errAssertionOpt := integration.WithErrorAssertion(func(err error) {
				require.ErrorIs(t, err, authz.ErrNoAuthorizationFound)
			})

			// Send an authz MsgExec from an unauthorized address.
			runOpts := append(runUntilNextBlockOpts, errAssertionOpt)
			execMsg := authz.NewMsgExec(unauthorizedAddr, []cosmostypes.Msg{msgUpdateParams})
			anyRes := s.app.RunMsg(s.T(), &execMsg, runOpts...)
			require.Nil(s.T(), anyRes)
		})
	}
}

func TestUpdateParamsSuite(t *testing.T) {
	suite.Run(t, &UpdateParamsSuite{})
}

func (s *UpdateParamsSuite) sendAuthzGrantMsgForPoktrollModules(
	granterAddr, granteeAddr cosmostypes.AccAddress,
	msgName string,
	moduleNames ...string,
) {
	var runOpts []integration.RunOption
	for moduleIdx, moduleName := range moduleNames {
		// Commit and finalize the block after the last module's grant.
		if moduleIdx == len(moduleNames)-1 {
			runOpts = append(runOpts, runUntilNextBlockOpts...)
		}

		msgType := fmt.Sprintf(poktrollMsgTypeFormat, moduleName, msgName)
		authorization := &authz.GenericAuthorization{Msg: msgType}
		s.runAuthzGrantMsg(granterAddr, granteeAddr, authorization, runOpts...)
	}

	authzQueryClient := authz.NewQueryClient(s.app.QueryHelper())
	grantsQueryRes, err := authzQueryClient.GranteeGrants(s.app.GetSdkCtx(), &authz.QueryGranteeGrantsRequest{
		Grantee: granteeAddr.String(),
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), grantsQueryRes)

	require.Equalf(s.T(),
		len(allPoktrollModuleNames),
		len(grantsQueryRes.Grants),
		"expected %d grants but got %d: %+v",
		len(allPoktrollModuleNames),
		len(grantsQueryRes.Grants),
		grantsQueryRes.Grants,
	)

	foundModuleGrants := make(map[string]int)
	for _, grant := range grantsQueryRes.GetGrants() {
		require.Equal(s.T(), granterAddr.String(), grant.Granter)
		require.Equal(s.T(), granteeAddr.String(), grant.Grantee)

		for _, moduleName := range allPoktrollModuleNames {
			if strings.Contains(grant.Authorization.GetTypeUrl(), moduleName) {
				foundModuleGrants[moduleName]++
			}
		}
	}

	for _, foundTimes := range foundModuleGrants {
		require.Equal(s.T(), 1, foundTimes)
	}
}

func (s *UpdateParamsSuite) runAuthzGrantMsg(
	granterAddr,
	granteeAddr cosmostypes.AccAddress,
	authorization authz.Authorization,
	runOpts ...integration.RunOption,
) {
	grantMsg, err := authz.NewMsgGrant(granterAddr, granteeAddr, authorization, &authzGrantExpiration)
	require.NoError(s.T(), err)

	anyRes := s.app.RunMsg(s.T(), grantMsg, runOpts...)
	require.NotNil(s.T(), anyRes)
}

func (s *UpdateParamsSuite) runAuthzExecMsg(
	fromAddr cosmostypes.AccAddress,
	msgs ...cosmostypes.Msg,
) {
	execMsg := authz.NewMsgExec(fromAddr, msgs)
	anyRes := s.app.RunMsg(s.T(), &execMsg, runUntilNextBlockOpts...)
	require.NotNil(s.T(), anyRes)

	execRes := new(authz.MsgExecResponse)
	err := s.app.GetCodec().UnpackAny(anyRes, &execRes)
	require.NoError(s.T(), err)
}
