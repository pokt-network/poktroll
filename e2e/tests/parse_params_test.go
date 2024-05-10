package e2e

import (
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	paramNameColIdx = iota
	paramValueColIdx
	paramTypeColIdx
)

// parseParamsTable parses a gocuke.DataTable into a paramsMap.
func (s *suite) parseParamsTable(table gocuke.DataTable) paramsMap {
	paramsMap := make(paramsMap)

	// NB: skip the header row.
	for rowIdx := 1; rowIdx < table.NumRows(); rowIdx++ {
		param := s.parseParam(table, rowIdx)
		paramsMap[param.name] = param
	}

	return paramsMap
}

// parseParam parses a row of a gocuke.DataTable into a paramName and a paramAny.
func (s *suite) parseParam(table gocuke.DataTable, rowIdx int) paramAny {
	var paramValue any
	paramName := table.Cell(rowIdx, paramNameColIdx).String()
	paramType := table.Cell(rowIdx, paramTypeColIdx).String()

	switch paramType {
	case "string":
		paramValue = table.Cell(rowIdx, paramValueColIdx).String()
	case "int64":
		paramValue = table.Cell(rowIdx, paramValueColIdx).Int64()
	case "bytes":
		paramValue = []byte(table.Cell(rowIdx, paramValueColIdx).String())
	default:
		s.Fatalf("unexpected param type %q", paramType)
	}

	return paramAny{
		name:    paramName,
		typeStr: paramType,
		value:   paramValue,
	}
}

// paramsMapToMsgUpdateParams converts a paramsMap into a MsgUpdateParams, which
// it returns as a proto.Message/cosmostypes.Msg interface type.
func (s *suite) paramsMapToMsgUpdateParams(moduleName string, paramsMap paramsMap) (msg cosmostypes.Msg) {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	switch moduleName {
	case tokenomicstypes.ModuleName:
		msgUpdateParams := &tokenomicstypes.MsgUpdateParams{
			Authority: authority,
			Params:    tokenomicstypes.Params{},
		}

		for paramName, paramValue := range paramsMap {
			switch paramName {
			case "compute_units_to_tokens_multiplier":
				msgUpdateParams.Params.ComputeUnitsToTokensMultiplier = uint64(paramValue.value.(int64))
			default:
				s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
			}
		}
		msg = proto.Message(msgUpdateParams)

	case prooftypes.ModuleName:
		msgUpdateParams := &prooftypes.MsgUpdateParams{
			Authority: authority,
			Params:    prooftypes.Params{},
		}

		for paramName, paramValue := range paramsMap {
			s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
			switch paramName {
			case "min_relay_difficulty_bits":
				msgUpdateParams.Params.MinRelayDifficultyBits = uint64(paramValue.value.(int64))
			default:
				s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
			}
		}
		msg = proto.Message(msgUpdateParams)

	default:
		err := fmt.Errorf("unexpected module name %q", moduleName)
		s.Fatal(err)
		panic(err)
	}

	return msg
}

// newMsgUpdateParam returns a MsgUpdateParam for the given module name, param name,
// and param type/value.
func (s *suite) newMsgUpdateParam(
	moduleName string,
	param paramAny,
) (msg cosmostypes.Msg) {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	// TODO_IMPROVE: can this be simplified?
	switch moduleName {
	case tokenomicstypes.ModuleName:
		switch param.typeStr {
		case "string":
			msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
				Authority: authority,
				Name:      param.name,
				AsType: &tokenomicstypes.MsgUpdateParam_AsString{
					AsString: param.value.(string),
				},
			})
		case "int64":
			msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
				Authority: authority,
				Name:      param.name,
				AsType: &tokenomicstypes.MsgUpdateParam_AsInt64{
					AsInt64: param.value.(int64),
				},
			})
		case "bytes":
			msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
				Authority: authority,
				Name:      param.name,
				AsType: &tokenomicstypes.MsgUpdateParam_AsBytes{
					AsBytes: param.value.([]byte),
				},
			})
		}
	case prooftypes.ModuleName:
		switch param.typeStr {
		case "string":
			msg = proto.Message(&prooftypes.MsgUpdateParam{
				Authority: authority,
				Name:      param.name,
				AsType: &prooftypes.MsgUpdateParam_AsString{
					AsString: param.value.(string),
				},
			})
		case "int64":
			msg = proto.Message(&prooftypes.MsgUpdateParam{
				Authority: authority,
				Name:      param.name,
				AsType: &prooftypes.MsgUpdateParam_AsInt64{
					AsInt64: param.value.(int64),
				},
			})
		case "bytes":
			msg = proto.Message(&prooftypes.MsgUpdateParam{
				Authority: authority,
				Name:      param.name,
				AsType: &prooftypes.MsgUpdateParam_AsBytes{
					AsBytes: param.value.([]byte),
				},
			})
		}
	default:
		err := fmt.Errorf("unexpected module name %q", moduleName)
		s.Fatal(err)
		panic(err)
	}

	return msg
}
