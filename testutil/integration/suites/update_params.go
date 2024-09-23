//go:build integration

package suites

import (
	"encoding/hex"
	"reflect"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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

// ParamType is a type alias for a module parameter type. It is the string that
// is returned when calling reflect.Value#Type()#Name() on a module parameter.
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

// ModuleParamConfig holds a set of valid parameters and type information for a
// given module. It uses any instead of cosmostypes.Msg to mitigate easy mistakes
// that could result from using pointers instead (unintended mutation). It is still
// possible to mutate this global variable; however, it is less likely to happen
// unintentionally.
type ModuleParamConfig struct {
	ValidParams             any
	MsgUpdateParams         any
	MsgUpdateParamsResponse any
	MsgUpdateParam          any
	MsgUpdateParamResponse  any
	ParamTypes              map[ParamType]any
	QueryParamsRequest      any
	QueryParamsResponse     any
	DefaultParams           any
	NewParamClientFn        any
}

var (
	// AuthorityAddr is the cosmos account address of the authority for the integration app.
	AuthorityAddr cosmostypes.AccAddress
	// AuthorizedAddr is the cosmos account address which is the grantee of authz
	// grants for parameter update messages.
	AuthorizedAddr cosmostypes.AccAddress

	ValidServiceFeeCoin                = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000001)
	ValidMissingPenaltyCoin            = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 500)
	ValidSubmissionFeeCoin             = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 5000000)
	ValidRelayDifficultyTargetHash, _  = hex.DecodeString("00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	ValidProofRequirementThresholdCoin = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)

	// ModuleParamConfigMap is a map of module names to their respective parameter
	// configurations. It is used by the UpdateParamsSuite, mostly via reflection,
	// to construct and send parameter update messages and assert on their results.
	ModuleParamConfigMap = map[string]ModuleParamConfig{
		sharedtypes.ModuleName: {
			ValidParams: sharedtypes.Params{
				NumBlocksPerSession:                12,
				GracePeriodEndOffsetBlocks:         0,
				ClaimWindowOpenOffsetBlocks:        2,
				ClaimWindowCloseOffsetBlocks:       3,
				ProofWindowOpenOffsetBlocks:        1,
				ProofWindowCloseOffsetBlocks:       3,
				SupplierUnbondingPeriodSessions:    9,
				ApplicationUnbondingPeriodSessions: 9,
				ComputeUnitsToTokensMultiplier:     420,
			},
			MsgUpdateParams:         sharedtypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: sharedtypes.MsgUpdateParamsResponse{},
			MsgUpdateParam:          sharedtypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  sharedtypes.MsgUpdateParamResponse{},
			ParamTypes: map[ParamType]any{
				ParamTypeUint64: sharedtypes.MsgUpdateParam_AsInt64{},
				ParamTypeInt64:  sharedtypes.MsgUpdateParam_AsInt64{},
				ParamTypeString: sharedtypes.MsgUpdateParam_AsString{},
				ParamTypeBytes:  sharedtypes.MsgUpdateParam_AsBytes{},
			},
			QueryParamsRequest:  sharedtypes.QueryParamsRequest{},
			QueryParamsResponse: sharedtypes.QueryParamsResponse{},
			DefaultParams:       sharedtypes.DefaultParams(),
			NewParamClientFn:    sharedtypes.NewQueryClient,
		},
		sessiontypes.ModuleName: {
			ValidParams:             sessiontypes.Params{},
			MsgUpdateParams:         sessiontypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: sessiontypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      sessiontypes.QueryParamsRequest{},
			QueryParamsResponse:     sessiontypes.QueryParamsResponse{},
			DefaultParams:           sessiontypes.DefaultParams(),
			NewParamClientFn:        sessiontypes.NewQueryClient,
		},
		servicetypes.ModuleName: {
			ValidParams: servicetypes.Params{
				AddServiceFee: &ValidServiceFeeCoin,
			},
			MsgUpdateParams:         servicetypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: servicetypes.MsgUpdateParamsResponse{},
			MsgUpdateParam:          servicetypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  servicetypes.MsgUpdateParamResponse{},
			ParamTypes: map[ParamType]any{
				ParamTypeCoin: servicetypes.MsgUpdateParam_AsCoin{},
			},
			QueryParamsRequest:  servicetypes.QueryParamsRequest{},
			QueryParamsResponse: servicetypes.QueryParamsResponse{},
			DefaultParams:       servicetypes.DefaultParams(),
			NewParamClientFn:    servicetypes.NewQueryClient,
		},
		apptypes.ModuleName: {
			ValidParams: apptypes.Params{
				MaxDelegatedGateways: 999,
			},
			MsgUpdateParams:         apptypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: apptypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      apptypes.QueryParamsRequest{},
			QueryParamsResponse:     apptypes.QueryParamsResponse{},
			DefaultParams:           apptypes.DefaultParams(),
			NewParamClientFn:        apptypes.NewQueryClient,
		},
		gatewaytypes.ModuleName: {
			ValidParams:             gatewaytypes.Params{},
			MsgUpdateParams:         gatewaytypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: gatewaytypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      gatewaytypes.QueryParamsRequest{},
			QueryParamsResponse:     gatewaytypes.QueryParamsResponse{},
			DefaultParams:           gatewaytypes.DefaultParams(),
			NewParamClientFn:        gatewaytypes.NewQueryClient,
		},
		suppliertypes.ModuleName: {
			ValidParams:             suppliertypes.Params{},
			MsgUpdateParams:         suppliertypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: suppliertypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      suppliertypes.QueryParamsRequest{},
			QueryParamsResponse:     suppliertypes.QueryParamsResponse{},
			DefaultParams:           suppliertypes.DefaultParams(),
			NewParamClientFn:        suppliertypes.NewQueryClient,
		},
		prooftypes.ModuleName: {
			ValidParams: prooftypes.Params{
				RelayDifficultyTargetHash: ValidRelayDifficultyTargetHash,
				ProofRequestProbability:   0.1,
				ProofRequirementThreshold: &ValidProofRequirementThresholdCoin,
				ProofMissingPenalty:       &ValidMissingPenaltyCoin,
				ProofSubmissionFee:        &ValidSubmissionFeeCoin,
			},
			MsgUpdateParams:         prooftypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: prooftypes.MsgUpdateParamsResponse{},
			MsgUpdateParam:          prooftypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  prooftypes.MsgUpdateParamResponse{},
			ParamTypes: map[ParamType]any{
				ParamTypeUint64:  prooftypes.MsgUpdateParam_AsInt64{},
				ParamTypeInt64:   prooftypes.MsgUpdateParam_AsInt64{},
				ParamTypeString:  prooftypes.MsgUpdateParam_AsString{},
				ParamTypeBytes:   prooftypes.MsgUpdateParam_AsBytes{},
				ParamTypeFloat32: prooftypes.MsgUpdateParam_AsFloat{},
				ParamTypeCoin:    prooftypes.MsgUpdateParam_AsCoin{},
			},
			QueryParamsRequest:  prooftypes.QueryParamsRequest{},
			QueryParamsResponse: prooftypes.QueryParamsResponse{},
			DefaultParams:       prooftypes.DefaultParams(),
			NewParamClientFn:    prooftypes.NewQueryClient,
		},
		tokenomicstypes.ModuleName: {
			ValidParams:             tokenomicstypes.Params{},
			MsgUpdateParams:         tokenomicstypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: tokenomicstypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      tokenomicstypes.QueryParamsRequest{},
			QueryParamsResponse:     tokenomicstypes.QueryParamsResponse{},
			DefaultParams:           tokenomicstypes.DefaultParams(),
			NewParamClientFn:        tokenomicstypes.NewQueryClient,
		},
	}

	// MsgUpdateParamEnabledModuleNames is a list of module names which support
	// individual parameter updates (i.e. MsgUpdateParam). It is initialized in
	// init().
	MsgUpdateParamEnabledModuleNames []string

	_ IntegrationSuite = (*UpdateParamsSuite)(nil)
)

func init() {
	for moduleName, moduleThing := range ModuleParamConfigMap {
		if moduleThing.MsgUpdateParam != nil {
			MsgUpdateParamEnabledModuleNames = append(MsgUpdateParamEnabledModuleNames, moduleName)
		}
	}
}

// UpdateParamsSuite is an integration test suite that provides helper functions for
// running parameter update messages. It is intended to be embedded in other integration
// test suites which are dependent on parameter updates.
type UpdateParamsSuite struct {
	AuthzIntegrationSuite
}

// SetupTestAuthzAccounts sets AuthorityAddr for the suite by getting the authority
// from the integration app. It also assigns a new pre-generated identity to be used
// as the AuthorizedAddr for the suite. It is expected to be called after s.NewApp()
// as it depends on the integration app and its pre-generated account iterator.
func (s *UpdateParamsSuite) SetupTestAuthzAccounts() {
	// Set the authority, authorized, and unauthorized addresses.
	AuthorityAddr = cosmostypes.MustAccAddressFromBech32(s.GetApp().GetAuthority())

	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	AuthorizedAddr = nextAcct.Address
}

// SetupTestAuthzGrants creates on-chain authz grants for the MsgUpdateUpdateParam and
// MsgUpdateParams message for each module. It is expected to be called after s.NewApp()
// as it depends on the authority and authorized addresses having been set.
func (s *UpdateParamsSuite) SetupTestAuthzGrants() {
	// Create authz grants for all poktroll modules' MsgUpdateParams messages.
	s.RunAuthzGrantMsgForPoktrollModules(s.T(),
		AuthorityAddr,
		AuthorizedAddr,
		MsgUpdateParamsName,
		s.GetPoktrollModuleNames()...,
	)

	// Create authz grants for all poktroll modules' MsgUpdateParam messages.
	s.RunAuthzGrantMsgForPoktrollModules(s.T(),
		AuthorityAddr,
		AuthorizedAddr,
		MsgUpdateParamName,
		// NB: only modules with params are expected to support MsgUpdateParam.
		MsgUpdateParamEnabledModuleNames...,
	)
}

// RunUpdateParams runs the given MsgUpdateParams message via an authz exec as the
// AuthorizedAddr and returns the response bytes and error. It is expected to be called
// after s.SetupTestAuthzGrants() as it depends on an on-chain authz grant to AuthorizedAddr
// for MsgUpdateParams for the given module.
func (s *UpdateParamsSuite) RunUpdateParams(
	t *testing.T,
	msgUpdateParams cosmostypes.Msg,
) (msgResponseBz []byte, err error) {
	t.Helper()

	return s.RunUpdateParamsAsSigner(t, msgUpdateParams, AuthorizedAddr)
}

// RunUpdateParamsAsSigner runs the given MsgUpdateParams message via an authz exec
// as signerAddr and returns the response bytes and error. It depends on an on-chain
// authz grant to signerAddr for MsgUpdateParams for the given module.
func (s *UpdateParamsSuite) RunUpdateParamsAsSigner(
	t *testing.T,
	msgUpdateParams cosmostypes.Msg,
	signerAddr cosmostypes.AccAddress,
) (msgResponseBz []byte, err error) {
	t.Helper()

	// Send an authz MsgExec from an unauthorized address.
	execMsg := authz.NewMsgExec(AuthorizedAddr, []cosmostypes.Msg{msgUpdateParams})
	msgRespsBz, err := s.RunAuthzExecMsg(t, signerAddr, &execMsg)
	if err != nil {
		return nil, err
	}

	require.Equal(t, 1, len(msgRespsBz), "expected exactly 1 message response")
	return msgRespsBz[0], err
}

// RunUpdateParam constructs and runs an MsgUpdateParam message via an authz exec
// as the AuthorizedAddr for the given module, parameter name, and value. It returns
// the response bytes and error. It is expected to be called after s.SetupTestAuthzGrants()
// as it depends on an on-chain authz grant to AuthorizedAddr for MsgUpdateParam for the given module.
func (s *UpdateParamsSuite) RunUpdateParam(
	t *testing.T,
	moduleName string,
	paramName string,
	paramValue any,
) (msgResponseBz []byte, err error) {
	t.Helper()

	return s.RunUpdateParamAsSigner(t,
		moduleName,
		paramName,
		paramValue,
		AuthorizedAddr,
	)
}

// RunUpdateParamAsSigner constructs and runs an MsgUpdateParam message via an authz exec
// as the given signerAddr for the given module, parameter name, and value. It returns
// the response bytes and error. It depends on an on-chain authz grant to signerAddr for
// MsgUpdateParam for the given module.
func (s *UpdateParamsSuite) RunUpdateParamAsSigner(
	t *testing.T,
	moduleName string,
	paramName string,
	paramValue any,
	signerAddr cosmostypes.AccAddress,
) (msgResponseBz []byte, err error) {
	t.Helper()

	moduleCfg := ModuleParamConfigMap[moduleName]

	paramReflectValue := reflect.ValueOf(paramValue)
	paramType := paramReflectValue.Type().Name()
	switch paramReflectValue.Kind() {
	case reflect.Pointer:
		paramType = paramReflectValue.Elem().Type().Name()
	case reflect.Slice:
		paramType = paramReflectValue.Type().Elem().Name()
	}

	msgIface := moduleCfg.MsgUpdateParam
	msgValue := reflect.ValueOf(msgIface)
	msgType := msgValue.Type()

	// Copy the message and set the authority field.
	msgUpdateParamValue := reflect.New(msgType)
	msgUpdateParamValue.Elem().
		FieldByName("Authority").
		SetString(AuthorityAddr.String())

	msgUpdateParamValue.Elem().FieldByName("Name").SetString(cases.ToSnakeCase(paramName))

	msgAsTypeStruct := moduleCfg.ParamTypes[paramType]
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

	msgUpdateParamValue.Elem().FieldByName("AsType").Set(msgAsTypeValue)

	msgUpdateParam := msgUpdateParamValue.Interface().(cosmostypes.Msg)

	// Send an authz MsgExec from the authority address.
	execMsg := authz.NewMsgExec(signerAddr, []cosmostypes.Msg{msgUpdateParam})
	execResps, err := s.RunAuthzExecMsg(t, signerAddr, &execMsg)
	if err != nil {
		return nil, err
	}

	require.Equal(t, 1, len(execResps), "expected exactly 1 message response")
	return execResps[0], err
}

// RequireModuleHasDefaultParams asserts that the given module's parameters are set
// to their default values.
func (s *UpdateParamsSuite) RequireModuleHasDefaultParams(t *testing.T, moduleName string) {
	t.Helper()

	params, err := s.QueryModuleParams(t, moduleName)
	require.NoError(t, err)

	moduleCfg := ModuleParamConfigMap[moduleName]
	require.EqualValues(t, moduleCfg.DefaultParams, params)
}

// QueryModuleParams queries the given module's parameters and returns them. It is
// expected to be called after s.NewApp() as it depends on the app's query helper.
func (s *UpdateParamsSuite) QueryModuleParams(t *testing.T, moduleName string) (params any, err error) {
	t.Helper()

	moduleCfg := ModuleParamConfigMap[moduleName]

	// Construct a new param client.
	newParamClientFn := reflect.ValueOf(moduleCfg.NewParamClientFn)
	newParamClientFnArgs := []reflect.Value{
		reflect.ValueOf(s.GetApp().QueryHelper()),
	}
	paramClient := newParamClientFn.Call(newParamClientFnArgs)[0]

	// Query for the module's params.
	paramsQueryReqValue := reflect.New(reflect.TypeOf(moduleCfg.QueryParamsRequest))
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
