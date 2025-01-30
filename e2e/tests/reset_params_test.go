//go:build e2e

package e2e

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// resetAllModuleParamsToDefaults resets all module params to their default values using
// a single authz exec message. It blocks until the resulting tx has been committed.
func (s *suite) resetAllModuleParamsToDefaults() {
	s.Helper()

	s.Log("resetting all module params to their default values")

	msgUpdateParamsAnys := s.allModulesMsgUpdateParamsToDefaultsAny()
	resetTxJSONFile := s.newTempTxJSONFile(msgUpdateParamsAnys)
	s.sendAuthzExecTx(s.granteeName, resetTxJSONFile.Name())
}

// allModulesMsgUpdateParamsToDefaultsAny returns a slice of Any messages, each corresponding
// to a MsgUpdateParams for a module, populated with the respective default values.
func (s *suite) allModulesMsgUpdateParamsToDefaultsAny() []*codectypes.Any {
	s.Helper()

	return []*codectypes.Any{
		s.msgUpdateParamsToDefaultsAny(gatewaytypes.ModuleName),
		s.msgUpdateParamsToDefaultsAny(apptypes.ModuleName),
		s.msgUpdateParamsToDefaultsAny(suppliertypes.ModuleName),
		s.msgUpdateParamsToDefaultsAny(prooftypes.ModuleName),
		s.msgUpdateParamsToDefaultsAny(tokenomicstypes.ModuleName),
		s.msgUpdateParamsToDefaultsAny(sharedtypes.ModuleName),
		s.msgUpdateParamsToDefaultsAny(servicetypes.ModuleName),
	}
}

// msgUpdateParamsToDefaultsAny returns an Any corresponding to a MsgUpdateParams
// for the given module name, populated with the respective default values.
func (s *suite) msgUpdateParamsToDefaultsAny(moduleName string) *codectypes.Any {
	s.Helper()

	var (
		anyMsg *codectypes.Any
		err    error
	)

	switch moduleName {
	case gatewaytypes.ModuleName:
		anyMsg, err = codectypes.NewAnyWithValue(
			&gatewaytypes.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(s.granterName).String(),
				Params:    gatewaytypes.DefaultParams(),
			},
		)
	case apptypes.ModuleName:
		anyMsg, err = codectypes.NewAnyWithValue(
			&apptypes.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(s.granterName).String(),
				Params:    apptypes.DefaultParams(),
			},
		)
	case suppliertypes.ModuleName:
		anyMsg, err = codectypes.NewAnyWithValue(
			&suppliertypes.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(s.granterName).String(),
				Params:    suppliertypes.DefaultParams(),
			},
		)
	case prooftypes.ModuleName:
		anyMsg, err = codectypes.NewAnyWithValue(
			&prooftypes.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(s.granterName).String(),
				Params:    prooftypes.DefaultParams(),
			},
		)
	case tokenomicstypes.ModuleName:
		anyMsg, err = codectypes.NewAnyWithValue(
			&tokenomicstypes.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(s.granterName).String(),
				Params:    tokenomicstypes.DefaultParams(),
			},
		)
	case sharedtypes.ModuleName:
		anyMsg, err = codectypes.NewAnyWithValue(
			&sharedtypes.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(s.granterName).String(),
				Params:    sharedtypes.DefaultParams(),
			},
		)
	case servicetypes.ModuleName:
		anyMsg, err = codectypes.NewAnyWithValue(
			&servicetypes.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(s.granterName).String(),
				Params:    servicetypes.DefaultParams(),
			},
		)
	default:
		s.Fatalf("ERROR: unknown module name: %s", moduleName)
	}
	require.NoError(s, err)

	return anyMsg
}
