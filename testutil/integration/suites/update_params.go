package suites

import (
	"encoding/hex"
	"reflect"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
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
	MsgUpdateParamsName = "MsgUpdateParams"
	MsgUpdateParamName  = "MsgUpdateParam"
)

var (
	AuthorityAddr cosmostypes.AccAddress

	AuthorizedAddr cosmostypes.AccAddress

	ValidSharedParams = sharedtypes.Params{
		NumBlocksPerSession:                5,
		GracePeriodEndOffsetBlocks:         2,
		ClaimWindowOpenOffsetBlocks:        5,
		ClaimWindowCloseOffsetBlocks:       5,
		ProofWindowOpenOffsetBlocks:        2,
		ProofWindowCloseOffsetBlocks:       5,
		SupplierUnbondingPeriodSessions:    9,
		ApplicationUnbondingPeriodSessions: 9,
		ComputeUnitsToTokensMultiplier:     420,
	}

	ValidSessionParams = sessiontypes.Params{}

	ValidServiceFeeCoin = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000001)
	ValidServiceParams  = servicetypes.Params{
		AddServiceFee: &ValidServiceFeeCoin,
	}

	ValidApplicationParams = apptypes.Params{
		MaxDelegatedGateways: 999,
	}

	ValidGatewayParams    = gatewaytypes.Params{}
	ValidSupplierParams   = suppliertypes.Params{}
	ValidTokenomicsParams = tokenomicstypes.Params{}

	ValidMissingPenaltyCoin           = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 500)
	ValidSubmissionFeeCoin            = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 5000000)
	ValidRelayDifficultyTargetHash, _ = hex.DecodeString("00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	ValidProofRequirementThresholdCoin = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)
	ValidProofParams                   = prooftypes.Params{
		RelayDifficultyTargetHash: ValidRelayDifficultyTargetHash,
		ProofRequestProbability:   0.1,
		ProofRequirementThreshold: &ValidProofRequirementThresholdCoin,
		ProofMissingPenalty:       &ValidMissingPenaltyCoin,
		ProofSubmissionFee:        &ValidSubmissionFeeCoin,
	}

	// TODO_IN_THIS_COMMIT: godoc...
	// NB: Authority fields are intentionally omitted and expected to be added
	// to a **copy** of the respective message by the test.
	MsgUpdateParamsByModule = map[string]any{
		sharedtypes.ModuleName: sharedtypes.MsgUpdateParams{
			Params: ValidSharedParams,
		},
		sessiontypes.ModuleName: sessiontypes.MsgUpdateParams{
			Params: ValidSessionParams,
		},
		servicetypes.ModuleName: servicetypes.MsgUpdateParams{
			Params: ValidServiceParams,
		},
		apptypes.ModuleName: apptypes.MsgUpdateParams{
			Params: ValidApplicationParams,
		},
		gatewaytypes.ModuleName: gatewaytypes.MsgUpdateParams{
			Params: ValidGatewayParams,
		},
		suppliertypes.ModuleName: suppliertypes.MsgUpdateParams{
			Params: ValidSupplierParams,
		},
		prooftypes.ModuleName: prooftypes.MsgUpdateParams{
			Params: ValidProofParams,
		},
		tokenomicstypes.ModuleName: tokenomicstypes.MsgUpdateParams{
			Params: ValidTokenomicsParams,
		},
	}

	MsgUpdateParamByModule = map[string]any{
		sharedtypes.ModuleName:     sharedtypes.MsgUpdateParam{},
		servicetypes.ModuleName:    servicetypes.MsgUpdateParam{},
		prooftypes.ModuleName:      prooftypes.MsgUpdateParam{},
		tokenomicstypes.ModuleName: tokenomicstypes.MsgUpdateParam{},
		//sessiontypes.ModuleName:    sessiontypes.MsgUpdateParam{},
		//apptypes.ModuleName:        apptypes.MsgUpdateParam{},
		//gatewaytypes.ModuleName:    gatewaytypes.MsgUpdateParam{},
		//suppliertypes.ModuleName:   suppliertypes.MsgUpdateParam{},
	}

	MsgUpdateParamTypesByModuleName = map[string]map[string]any{
		sharedtypes.ModuleName: {
			"uint64": sharedtypes.MsgUpdateParam_AsInt64{},
			"int64":  sharedtypes.MsgUpdateParam_AsInt64{},
			"string": sharedtypes.MsgUpdateParam_AsString{},
			"uint8":  sharedtypes.MsgUpdateParam_AsBytes{},
		},
		servicetypes.ModuleName: {
			"Coin": servicetypes.MsgUpdateParam_AsCoin{},
		},
		prooftypes.ModuleName: {
			"uint64":  prooftypes.MsgUpdateParam_AsInt64{},
			"int64":   prooftypes.MsgUpdateParam_AsInt64{},
			"string":  prooftypes.MsgUpdateParam_AsString{},
			"uint8":   prooftypes.MsgUpdateParam_AsBytes{},
			"float32": prooftypes.MsgUpdateParam_AsFloat{},
			//"float64": prooftypes.MsgUpdateParam_AsFloat{},
			"Coin": prooftypes.MsgUpdateParam_AsCoin{},
		},
		tokenomicstypes.ModuleName: {
			"uint64": tokenomicstypes.MsgUpdateParam_AsInt64{},
			"int64":  tokenomicstypes.MsgUpdateParam_AsInt64{},
			"string": tokenomicstypes.MsgUpdateParam_AsString{},
			"uint8":  tokenomicstypes.MsgUpdateParam_AsBytes{},
		},
	}

	MsgUpdateParamEnabledModuleNames []string

	NewParamClientFns = map[string]any{
		sharedtypes.ModuleName:     sharedtypes.NewQueryClient,
		sessiontypes.ModuleName:    sessiontypes.NewQueryClient,
		servicetypes.ModuleName:    servicetypes.NewQueryClient,
		apptypes.ModuleName:        apptypes.NewQueryClient,
		gatewaytypes.ModuleName:    gatewaytypes.NewQueryClient,
		suppliertypes.ModuleName:   suppliertypes.NewQueryClient,
		prooftypes.ModuleName:      prooftypes.NewQueryClient,
		tokenomicstypes.ModuleName: tokenomicstypes.NewQueryClient,
	}

	QueryParamsRequestByModule = map[string]any{
		sharedtypes.ModuleName:     sharedtypes.QueryParamsRequest{},
		sessiontypes.ModuleName:    sessiontypes.QueryParamsRequest{},
		servicetypes.ModuleName:    servicetypes.QueryParamsRequest{},
		apptypes.ModuleName:        apptypes.QueryParamsRequest{},
		gatewaytypes.ModuleName:    gatewaytypes.QueryParamsRequest{},
		suppliertypes.ModuleName:   suppliertypes.QueryParamsRequest{},
		prooftypes.ModuleName:      prooftypes.QueryParamsRequest{},
		tokenomicstypes.ModuleName: tokenomicstypes.QueryParamsRequest{},
	}

	DefaultParamsByModule = map[string]any{
		sharedtypes.ModuleName:     sharedtypes.DefaultParams(),
		sessiontypes.ModuleName:    sessiontypes.DefaultParams(),
		servicetypes.ModuleName:    servicetypes.DefaultParams(),
		apptypes.ModuleName:        apptypes.DefaultParams(),
		gatewaytypes.ModuleName:    gatewaytypes.DefaultParams(),
		suppliertypes.ModuleName:   suppliertypes.DefaultParams(),
		prooftypes.ModuleName:      prooftypes.DefaultParams(),
		tokenomicstypes.ModuleName: tokenomicstypes.DefaultParams(),
	}

	_ IntegrationSuite = (*UpdateParamsSuite)(nil)
)

func init() {
	for moduleName := range MsgUpdateParamByModule {
		MsgUpdateParamEnabledModuleNames = append(MsgUpdateParamEnabledModuleNames, moduleName)
	}
}

type UpdateParamsSuite struct {
	AuthzIntegrationSuite
}

// SetupTest runs before each test in the suite.
func (s *UpdateParamsSuite) SetupTest() {
	// Construct a fresh integration app for each test.
	s.NewApp(s.T())

	// Set the authority, authorized, and unauthorized addresses.
	AuthorityAddr = cosmostypes.MustAccAddressFromBech32(s.GetApp().GetAuthority())

	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	AuthorizedAddr = nextAcct.Address

	// Create authz grants for all poktroll modules' MsgUpdateParams messages.
	s.SendAuthzGrantMsgForPoktrollModules(s.T(),
		AuthorityAddr,
		AuthorizedAddr,
		MsgUpdateParamsName,
		s.GetPoktrollModuleNames()...,
	)

	// Create authz grants for all poktroll modules' MsgUpdateParam messages.
	s.SendAuthzGrantMsgForPoktrollModules(s.T(),
		AuthorityAddr,
		AuthorizedAddr,
		MsgUpdateParamName,
		MsgUpdateParamEnabledModuleNames...,
	)
}

// TODO_IN_THIS_COMMIT: godoc
func (s *UpdateParamsSuite) RequireModuleHasDefaultParams(t *testing.T, moduleName string) {
	t.Helper()

	params, err := s.QueryModuleParams(t, moduleName)
	require.NoError(t, err)

	defaultParams := DefaultParamsByModule[moduleName]
	require.EqualValues(t, defaultParams, params)
}

// TODO_IN_THIS_COMMIT: godoc
func (s *UpdateParamsSuite) QueryModuleParams(t *testing.T, moduleName string) (params any, err error) {
	t.Helper()

	// Construct a new param client.
	newParamClientFn := reflect.ValueOf(NewParamClientFns[moduleName])
	newParamClientFnArgs := []reflect.Value{
		reflect.ValueOf(s.GetApp().QueryHelper()),
	}
	paramClient := newParamClientFn.Call(newParamClientFnArgs)[0]

	// Query for the module's params.
	paramsQueryReqValue := reflect.New(reflect.TypeOf(QueryParamsRequestByModule[moduleName]))
	callParamsArgs := []reflect.Value{
		reflect.ValueOf(s.GetApp().GetSdkCtx()),
		paramsQueryReqValue,
	}
	callParamsReturnValues := paramClient.MethodByName("Params").Call(callParamsArgs)
	paramsResParamsValue := callParamsReturnValues[0]
	paramResErrValue := callParamsReturnValues[1].Interface()

	isErr := false
	err, isErr = paramResErrValue.(error)
	if !isErr {
		require.Nil(t, callParamsReturnValues[1].Interface())
	}

	paramsValue := paramsResParamsValue.Elem().FieldByName("Params")
	return paramsValue.Interface(), err
}
