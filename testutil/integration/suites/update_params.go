//go:build integration

package suites

import (
	"encoding/hex"
	"reflect"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/cases"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_IN_THIS_COMMIT: godoc...
type ParamType = string

const (
	ParamTypeInt64   ParamType = "int64"
	ParamTypeUint64  ParamType = "uint64"
	ParamTypeFloat32 ParamType = "float32"
	ParamTypeString  ParamType = "string"
	ParamTypeBytes   ParamType = "uint8"
	ParamTypeCoin    ParamType = "Coin"
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

	// TODO_IN_THIS_COMMIT: godoc...
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

	// TODO_IN_THIS_COMMIT: ... each module defines its own MsgUpdateParam_As* structs
	// ... not every module has all types...
	MsgUpdateParamTypesByModuleName = map[string]map[ParamType]any{
		sharedtypes.ModuleName: {
			ParamTypeUint64: sharedtypes.MsgUpdateParam_AsInt64{},
			ParamTypeInt64:  sharedtypes.MsgUpdateParam_AsInt64{},
			ParamTypeString: sharedtypes.MsgUpdateParam_AsString{},
			ParamTypeBytes:  sharedtypes.MsgUpdateParam_AsBytes{},
		},
		servicetypes.ModuleName: {
			ParamTypeCoin: servicetypes.MsgUpdateParam_AsCoin{},
		},
		prooftypes.ModuleName: {
			ParamTypeUint64:  prooftypes.MsgUpdateParam_AsInt64{},
			ParamTypeInt64:   prooftypes.MsgUpdateParam_AsInt64{},
			ParamTypeString:  prooftypes.MsgUpdateParam_AsString{},
			ParamTypeBytes:   prooftypes.MsgUpdateParam_AsBytes{},
			ParamTypeFloat32: prooftypes.MsgUpdateParam_AsFloat{},
			ParamTypeCoin:    prooftypes.MsgUpdateParam_AsCoin{},
		},
		tokenomicstypes.ModuleName: {
			ParamTypeUint64: tokenomicstypes.MsgUpdateParam_AsInt64{},
			ParamTypeInt64:  tokenomicstypes.MsgUpdateParam_AsInt64{},
			ParamTypeString: tokenomicstypes.MsgUpdateParam_AsString{},
			ParamTypeBytes:  tokenomicstypes.MsgUpdateParam_AsBytes{},
		},
	}

	// TODO_IN_THIS_COMMIT: godoc...
	MsgUpdateParamEnabledModuleNames []string

	// TODO_IN_THIS_COMMIT: godoc...
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

	// TODO_IN_THIS_COMMIT: godoc...
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

	// TODO_IN_THIS_COMMIT: godoc...
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
	// TODO_IN_THIS_COMMIT: godoc...
	for moduleName := range MsgUpdateParamByModule {
		MsgUpdateParamEnabledModuleNames = append(MsgUpdateParamEnabledModuleNames, moduleName)
	}
}

type UpdateParamsSuite struct {
	AuthzIntegrationSuite
}

// TODO_IN_THIS_COMMIT: godoc
// SetupTestAccounts ... expected to be called after s.NewApp() ... accounts ... and module names...
func (s *UpdateParamsSuite) SetupTestAccounts() {
	// Set the authority, authorized, and unauthorized addresses.
	AuthorityAddr = cosmostypes.MustAccAddressFromBech32(s.GetApp().GetAuthority())

	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	AuthorizedAddr = nextAcct.Address
}

// TODO_IN_THIS_COMMIT: godoc
// SetupTestAuthzGrants ... expected to be called after s.NewApp() ... authority and authorized addresses...
func (s *UpdateParamsSuite) SetupTestAuthzGrants() {
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

func (s *UpdateParamsSuite) RunUpdateParam(
	t *testing.T,
	moduleName string,
	paramName string,
	paramValue any,
) (tx.MsgResponse, error) {
	return s.RunUpdateParamAsSigner(t,
		moduleName,
		paramName,
		paramValue,
		AuthorizedAddr,
	)
}

func (s *UpdateParamsSuite) RunUpdateParamAsSigner(
	t *testing.T,
	moduleName string,
	paramName string,
	paramValue any,
	signerAddr cosmostypes.AccAddress,
) (tx.MsgResponse, error) {
	paramReflectValue := reflect.ValueOf(paramValue)
	paramType := paramReflectValue.Type().Name()
	if paramReflectValue.Kind() == reflect.Pointer {
		paramType = paramReflectValue.Elem().Type().Name()
	}

	msgIface, isMsgTypeFound := MsgUpdateParamByModule[moduleName]
	require.Truef(t, isMsgTypeFound, "unknown message type for module %q: %T", moduleName, msgIface)

	msgValue := reflect.ValueOf(msgIface)
	msgType := msgValue.Type()

	// Copy the message and set the authority field.
	msgValueCopy := reflect.New(msgType)
	msgValueCopy.Elem().Set(msgValue)
	msgValueCopy.Elem().
		FieldByName("Authority").
		SetString(AuthorityAddr.String())

	msgValueCopy.Elem().FieldByName("Name").SetString(cases.ToSnakeCase(paramName))

	msgAsTypeStruct := MsgUpdateParamTypesByModuleName[moduleName][paramType]
	msgAsTypeType := reflect.TypeOf(msgAsTypeStruct)
	msgAsTypeValue := reflect.New(msgAsTypeType)
	switch paramType {
	case ParamTypeUint64:
		// NB: MsgUpdateParam doesn't currently support uint64 param type.
		msgAsTypeValue.Elem().FieldByName("AsInt64").SetInt(int64(paramReflectValue.Interface().(uint64)))
	case ParamTypeInt64:
		msgAsTypeValue.Elem().FieldByName("AsInt64").Set(paramReflectValue)
	case ParamTypeFloat32:
		msgAsTypeValue.Elem().FieldByName("AsFloat").Set(paramReflectValue)
	case ParamTypeString:
		msgAsTypeValue.Elem().FieldByName("AsString").Set(paramReflectValue)
	case ParamTypeBytes:
		msgAsTypeValue.Elem().FieldByName("AsBytes").Set(paramReflectValue)
	case ParamTypeCoin:
		msgAsTypeValue.Elem().FieldByName("AsCoin").Set(paramReflectValue)
	default:
		t.Fatalf("ERROR: unknown field type %q", paramType)
	}

	msgValueCopy.Elem().FieldByName("AsType").Set(msgAsTypeValue)

	msgUpdateParam := msgValueCopy.Interface().(cosmostypes.Msg)

	// Send an authz MsgExec from the authority address.
	execMsg := authz.NewMsgExec(signerAddr, []cosmostypes.Msg{msgUpdateParam})
	execResps, err := s.RunAuthzExecMsg(t, signerAddr, &execMsg)

	require.Equal(t, 1, len(execResps), "expected exactly one MsgResponse")

	return execResps[0], err
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
