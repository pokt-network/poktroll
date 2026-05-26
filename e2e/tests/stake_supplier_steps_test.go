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

	// NB: If new parameters are added to the shared module, they
	// MUST be included here; otherwise, this step will fail.
	sharedParams := sharedtypes.Params{
		NumBlocksPerSession:                uint64(numBlocksPerSession),
		GracePeriodEndOffsetBlocks:         0,
		ClaimWindowOpenOffsetBlocks:        0,
		ClaimWindowCloseOffsetBlocks:       1,
		ProofWindowOpenOffsetBlocks:        0,
		ProofWindowCloseOffsetBlocks:       1,
		SupplierUnbondingPeriodSessions:    uint64(unbondingPeriodSessions),
		ApplicationUnbondingPeriodSessions: uint64(unbondingPeriodSessions),
		GatewayUnbondingPeriodSessions:     uint64(unbondingPeriodSessions),
		ComputeUnitsToTokensMultiplier:     sharedtypes.DefaultComputeUnitsToTokensMultiplier,
		ComputeUnitCostGranularity:         sharedtypes.DefaultComputeUnitCostGranularity,
	}

	// Convert params struct to the map type expected by
	// s.sendAuthzExecToUpdateAllModuleParams().
	paramsMap := paramsAnyMapFromParamsStruct(sharedParams)
	s.sendAuthzExecToUpdateAllModuleParams(grantee, paramModuleName, paramsMap)

	// Assert that the parameter values were updated.
	s.AllModuleParamsShouldBeUpdated(paramModuleName)
}

// derivedSharedParamFieldNames lists shared module Params fields that are NOT
// governance-settable — they are derived runtime metadata stamped per epoch by
// the shared keeper (#543 anchored grid). They have no corresponding ParamX
// constant and no case in buildSharedMsgUpdateParams' switch, so including them
// in a params update map would fatally fail. Skip them in the reflection helper.
var derivedSharedParamFieldNames = map[string]struct{}{
	"session_grid_anchor_height": {},
	"session_number_at_anchor":   {},
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

		// Skip derived (non-governance-settable) fields. The shared module's
		// anchor metadata fields are stamped per-epoch by the keeper, not set by
		// MsgUpdateParam(s), and have no corresponding ParamX constant.
		if _, isDerived := derivedSharedParamFieldNames[paramName]; isDerived {
			continue
		}

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
