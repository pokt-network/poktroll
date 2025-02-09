//go:build e2e

package e2e

import (
	"reflect"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/cases"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (s *suite) TheUnbondingPeriodParamIsSuccessfullySetToSessionsOfBlocks(
	_ string,
	unbondingPeriodSessions,
	numBlocksPerSession int64,
) {
	require.GreaterOrEqualf(s, numBlocksPerSession, int64(2),
		"num_blocks_per_session MUST be at least 2 to satisfy parameter validation requirements")

	paramModuleName := "shared"
	granter := "gov"
	grantee := "pnf"

	// Ensure an authz grant is present such that this step may update parameters.
	s.AnAuthzGrantFromTheAccountToTheAccountForEachModuleMsgupdateparamMessageExists(
		granter, "module",
		grantee, "user",
	)

	// Get current params first
	currentParams := s.queryAllModuleParams(paramModuleName)
	currentSharedParams := currentParams.Params

	// Only update the specific parameters we care about
	sharedParams := sharedtypes.Params{
		NumBlocksPerSession:                uint64(numBlocksPerSession),
		GracePeriodEndOffsetBlocks:         currentSharedParams.GracePeriodEndOffsetBlocks,
		ClaimWindowOpenOffsetBlocks:        currentSharedParams.ClaimWindowOpenOffsetBlocks,
		ClaimWindowCloseOffsetBlocks:       currentSharedParams.ClaimWindowCloseOffsetBlocks,
		ProofWindowOpenOffsetBlocks:        currentSharedParams.ProofWindowOpenOffsetBlocks,
		ProofWindowCloseOffsetBlocks:       currentSharedParams.ProofWindowCloseOffsetBlocks,
		SupplierUnbondingPeriodSessions:    uint64(unbondingPeriodSessions),
		ApplicationUnbondingPeriodSessions: uint64(unbondingPeriodSessions),
		ComputeUnitsToTokensMultiplier:     currentSharedParams.ComputeUnitsToTokensMultiplier,
	}

	// Convert params struct to the map type expected by
	// s.sendAuthzExecToUpdateAllModuleParams().
	paramsMap := paramsAnyMapFromParamsStruct(sharedParams)
	s.sendAuthzExecToUpdateAllModuleParams(grantee, paramModuleName, paramsMap)

	// Assert that the parameter values were updated.
	s.AllModuleParamsShouldBeUpdated(paramModuleName)
}

// paramsAnyMapFromParamStruct construct a paramsAnyMap from any
// protobuf Param message type (tx.proto) using reflection.
func paramsAnyMapFromParamsStruct(paramStruct any) paramsAnyMap {
	paramsMap := make(paramsAnyMap)
	paramsReflectValue := reflect.ValueOf(paramStruct)
	for i := 0; i < paramsReflectValue.NumField(); i++ {
		fieldValue := paramsReflectValue.Field(i)
		fieldStruct := paramsReflectValue.Type().Field(i)
		paramName := cases.ToSnakeCase(fieldStruct.Name)

		fieldTypeName := fieldStruct.Type.Name()
		// TODO_IMPROVE: MsgUpdateParam currently only supports int64 and not uint64 value types.
		if fieldTypeName == "uint64" {
			fieldTypeName = "int64"
			fieldValue = reflect.ValueOf(int64(fieldValue.Interface().(uint64)))
		}

		paramsMap[paramName] = paramAny{
			name:    paramName,
			typeStr: fieldTypeName,
			value:   fieldValue.Interface(),
		}
	}
	return paramsMap
}

// queryAllModuleParams queries all parameters for the given module and returns them
// as a QueryParamsResponse.
func (s *suite) queryAllModuleParams(moduleName string) *sharedtypes.QueryParamsResponse {
	argsAndFlags := []string{
		"query",
		moduleName,
		"params",
		"--output=json",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoError(s, err)

	var paramsRes sharedtypes.QueryParamsResponse
	s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &paramsRes)
	return &paramsRes
}
