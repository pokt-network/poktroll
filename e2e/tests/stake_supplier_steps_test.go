//go:build e2e

package e2e

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/stretchr/testify/require"

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
		ComputeUnitsToTokensMultiplier:     sharedtypes.DefaultComputeUnitsToTokensMultiplier,
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
		paramName := toSnakeCase(fieldStruct.Name)

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

func toSnakeCase(str string) string {
	var result strings.Builder

	for i, runeValue := range str {
		if unicode.IsUpper(runeValue) {
			// If it's not the first letter, add an underscore
			if i > 0 {
				result.WriteRune('_')
			}
			// Convert to lowercase
			result.WriteRune(unicode.ToLower(runeValue))
		} else {
			// Otherwise, just append the rune as-is
			result.WriteRune(runeValue)
		}
	}

	return result.String()
}
