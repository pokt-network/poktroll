//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

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
	// txDelaySeconds is the number of seconds to wait for a tx to be committed before making assertions.
	txDelaySeconds = 3
	// txFeesCoinStr is the string representation of the amount & denom of tokens
	// which are sufficient to pay for tx fees in the test.
	txFeesCoinStr = "1000000upokt"
	// PNF is the account that acts on behalf of the DAO and is therefore the only
	// one authorized to perform certain actions such as updating params.
	pnfKeyName = "pnf"
)

// allModuleMsgUpdateParamTypes is a list of all MsgUpdateParams types for each module.
// DEV_NOTE: If you are reading this and any module has a MsgUpdateParams message which is not
// included in this list, please add it.
var allModuleMsgUpdateParamTypes = []string{
	"/pocket.application.MsgUpdateParams",
	"/pocket.gateway.MsgUpdateParams",
	"/pocket.proof.MsgUpdateParams",
	"/pocket.session.MsgUpdateParams",
	"/pocket.shared.MsgUpdateParams",
	"/pocket.supplier.MsgUpdateParams",
	"/pocket.tokenomics.MsgUpdateParams",
	"/pocket.service.MsgUpdateParams",
}

func init() {
}

// AllModuleParamsAreSetToTheirDefaultValues asserts that all module params are set to their default values.
func (s *suite) AllModuleParamsAreSetToTheirDefaultValues(moduleName string) {
	argsAndFlags := []string{
		"query",
		moduleName,
		"params",
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoError(s, err)

	switch moduleName {
	case tokenomicstypes.ModuleName:
		var tokenomicsParamsRes tokenomicstypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &tokenomicsParamsRes)
		fmt.Println("OLSH1", tokenomicsParamsRes.GetParams())
		fmt.Println("OLSH2", tokenomicstypes.DefaultParams())
		require.Equal(s, tokenomicstypes.DefaultParams(), tokenomicsParamsRes.GetParams())

	case prooftypes.ModuleName:
		var proofParamsRes prooftypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &proofParamsRes)
		require.Equal(s, prooftypes.DefaultParams(), proofParamsRes.GetParams())

	case sessiontypes.ModuleName:
		var sessionParamsRes sessiontypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &sessionParamsRes)
		require.Equal(s, sessiontypes.DefaultParams(), sessionParamsRes.GetParams())

	case apptypes.ModuleName:
		var appParamsRes apptypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &appParamsRes)
		require.Equal(s, apptypes.DefaultParams(), appParamsRes.GetParams())

	case gatewaytypes.ModuleName:
		var gatewayParamsRes gatewaytypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &gatewayParamsRes)
		require.Equal(s, gatewaytypes.DefaultParams(), gatewayParamsRes.GetParams())

	case suppliertypes.ModuleName:
		var supplierParamsRes suppliertypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &supplierParamsRes)
		require.Equal(s, suppliertypes.DefaultParams(), supplierParamsRes.GetParams())

	case servicetypes.ModuleName:
		var serviceParamsRes servicetypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &serviceParamsRes)
		require.Equal(s, servicetypes.DefaultParams(), serviceParamsRes.GetParams())

	case sharedtypes.ModuleName:
		var sharedParamsRes sharedtypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &sharedParamsRes)
		require.Equal(s, sharedtypes.DefaultParams(), sharedParamsRes.GetParams())

	default:
		s.Fatalf("ERROR: unexpected module name: (%v)", moduleName)
	}
}

// AnAuthzGrantFromTheAccountToTheAccountForEachModuleMsgupdateparamMessageExists queries the
// authz module for grants with the expected granter & grantee (authz.QueryGrantsRequest) &
// asserts that the expected grants (for each module) are found in the response.
func (s *suite) AnAuthzGrantFromTheAccountToTheAccountForEachModuleMsgupdateparamMessageExists(
	granterName string,
	granterAddrType string,
	granteeName string,
	granteeAddrType string,
) {
	for _, msgType := range allModuleMsgUpdateParamTypes {
		s.AnAuthzGrantFromTheAccountToTheAccountForTheMessageExists(
			granterName,
			granterAddrType,
			granteeName,
			granteeAddrType,
			msgType,
		)
	}
}

// AnAuthzGrantFromTheAccountToTheAccountForTheMessage queries the authz module for grants
// with the expected granter & grantee (authz.QueryGrantsRequest) & asserts that the expected
// grant is found in the response.
func (s *suite) AnAuthzGrantFromTheAccountToTheAccountForTheMessageExists(
	granterName string,
	granterAddrType string,
	granteeName string,
	granteeAddrType string,
	msgType string,
) {
	// Declare granter & grantee addresses for use in the authz CLI query sub-command.
	var granterAddr, granteeAddr string
	nameToAddrMap := map[string]*string{
		granterName: &granterAddr,
		granteeName: &granteeAddr,
	}
	nameToAddrTypeMap := map[string]string{
		granterName: granterAddrType,
		granteeName: granteeAddrType,
	}

	// Set the granter & grantee addresses based on the address type.
	for name, addr := range nameToAddrMap {
		switch nameToAddrTypeMap[name] {
		case "module":
			*addr = authtypes.NewModuleAddress(name).String()
		case "user":
			*addr = s.getKeyAddress(name)
		default:
			s.Fatal("unexpected address type")
		}
	}

	// Query the authz module for grants with the expected granter & grantee (authz.QueryGrantsRequest).
	argsAndFlags := []string{
		"query", "authz", "grants",
		granterAddr, granteeAddr, msgType,
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoError(s, err)

	// Unmarshal the JSON response into the grantsRes struct.
	// Using s.cdc.UnmarshalJSON() with *authz.QueryGrantsResponse complains about
	// the "type" field, expecting "@type" for the #Grants[n]#Authorization pb.Any.
	grantsRes := new(authzCLIGrantResponse)
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	err = json.Unmarshal(responseBz, grantsRes)
	require.NoError(s, err)

	// Check if the expected grant is found in the response.
	grantFound := false
	for _, grant := range grantsRes.Grants {
		if grant.Authorization.Value.Msg == msgType {
			grantFound = true
			break
		}
	}
	require.True(s, grantFound)

	// Update the state of the suite with the granter & grantee names.
	s.granterName = granterName
	s.granteeName = granteeName
}

// AllModuleParamsShouldBeSetToTheirDefaultValues asserts that all module params are set to their default values.
func (s *suite) AllModuleParamsShouldBeSetToTheirDefaultValues(moduleName string) {
	s.AllModuleParamsAreSetToTheirDefaultValues(moduleName)
}

// TheAccountSendsAnAuthzExecMessageToUpdateAllModuleParams sends an authz exec
// message to update all module params for the given module.
func (s *suite) TheAccountSendsAnAuthzExecMessageToUpdateAllModuleParams(accountName, moduleName string, table gocuke.DataTable) {
	paramsTableMap := s.parseParamsTable(table)

	s.sendAuthzExecToUpdateAllModuleParams(accountName, moduleName, paramsTableMap)
}

// AllModuleParamsShouldBeUpdated asserts that all module params have been updated as expected.
func (s *suite) AllModuleParamsShouldBeUpdated(moduleName string) {
	_, ok := s.expectedModuleParams[moduleName]
	require.True(s, ok, "module %q params expectation not set on the test suite", moduleName)

	s.assertExpectedModuleParamsUpdated(moduleName)
}

// TheAccountSendsAnAuthzExecMessageToUpdateTheModuleParam sends an authz exec message to update a single module param.
func (s *suite) TheAccountSendsAnAuthzExecMessageToUpdateTheModuleParam(accountName, moduleName string, table gocuke.DataTable) {
	// NB: skip the header row & only expect a single row.
	param := s.parseParam(table, 1)

	// NB: set s#moduleParamsMap for later assertion.
	s.expectedModuleParams = moduleParamsMap{
		moduleName: {
			param.name: param,
		},
	}

	// Use the map of params to populate a tx JSON template & write it to a file.
	txJSONFile := s.newTempUpdateParamTxJSONFile(s.expectedModuleParams)

	// Send the authz exec tx to update the module param.
	s.sendAuthzExecTx(accountName, txJSONFile.Name())
}

// TheModuleParamShouldBeUpdated asserts that the module param has been updated as expected.
func (s *suite) TheModuleParamShouldBeUpdated(moduleName, paramName string) {
	moduleParamsMap, ok := s.expectedModuleParams[moduleName]
	require.True(s, ok, "module %q params expectation not set on the test suite", moduleName)

	var foundExpectedParam bool
	for expectedParamName := range moduleParamsMap {
		if paramName == expectedParamName {
			foundExpectedParam = true
			break
		}
	}
	require.True(s, foundExpectedParam, "param %q expectation not set on the test suite", paramName)

	s.assertExpectedModuleParamsUpdated(moduleName)
}

// AllModuleParamsShouldBeSetToTheirDefaultValues ensures that all module params are set to their default values.
func (s *suite) AllModuleParamsAreResetToTheirDefaultValues() {
	s.resetAllModuleParamsToDefaults()
}

// TheModuleParamShouldBeSetToItsDefaultValue asserts that the given param for the
// given module has been set to its default value.
func (s *suite) TheModuleParamShouldBeSetToItsDefaultValue(moduleName, paramName string) {
	// TODO_HACK: So long as no other modules are expected to have been changed by this scenario,
	// it is more than sufficient (and less code) to re-use the existing step which asserts that
	// all modules have their params set to their respective defaults.
	_ = paramName
	s.AllModuleParamsShouldBeSetToTheirDefaultValues(moduleName)
}

// ensureAccountForKeyName ensures that an account exists for the given key name in the keychain.
func (s *suite) ensureAccountForKeyName(keyName string) {
	s.Helper()

	// Get the address of the key.
	addr := s.getKeyAddress(keyName)

	// Fund the account with minimal tokens to ensure it can afford tx fees.
	coin, err := cosmostypes.ParseCoinNormalized(txFeesCoinStr)
	require.NoError(s, err)
	s.fundAddress(addr, coin)
}

// fundAddress sends the given amount & demon of tokens to the given address.
func (s *suite) fundAddress(addr string, coin cosmostypes.Coin) {
	s.Helper()

	// pocketd tx bank send <from> <to> <amount> --keyring-backend test --chain-id <chain_id> --yes
	argsAndFlags := []string{
		"tx",
		"bank",
		"send",
		pnfKeyName,
		addr,
		coin.String(),
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}

	_, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	// TODO_IMPROVE: wait for the tx to be committed using an events query client
	// instead of sleeping for a specific amount of time.
	s.Logf("waiting %d seconds for the funding tx to be committed...", txDelaySeconds)
	time.Sleep(txDelaySeconds * time.Second)
}

// getKeyAddress uses the `keys show` CLI subcommand to get the address of a key.
func (s *suite) getKeyAddress(keyName string) string {
	s.Helper()

	argsAndFlags := []string{
		"keys",
		"show",
		keyName,
		keyRingFlag,
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHost("local", argsAndFlags...)
	require.NoError(s, err)

	keyListRes := make(map[string]any)
	err = json.Unmarshal([]byte(res.Stdout), &keyListRes)
	require.NoError(s, err)

	return keyListRes["address"].(string)
}

func (s *suite) assertExpectedModuleParamsUpdated(moduleName string) {
	s.Helper()

	argsAndFlags := []string{
		"query",
		moduleName,
		"params",
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoError(s, err)

	switch moduleName {
	case tokenomicstypes.ModuleName:
		params := tokenomicstypes.DefaultParams()
		paramsMap := s.expectedModuleParams[moduleName]

		daoRewardAddress, ok := paramsMap[tokenomicstypes.ParamDaoRewardAddress]
		if ok {
			params.DaoRewardAddress = daoRewardAddress.value.(string)
		}

		globalInflationPerClaim, ok := paramsMap[tokenomicstypes.ParamGlobalInflationPerClaim]
		if ok {
			params.GlobalInflationPerClaim = globalInflationPerClaim.value.(float64)
		}

		mintAllocationPercentages, ok := paramsMap[tokenomicstypes.ParamMintAllocationPercentages]
		if ok {
			params.MintAllocationPercentages = mintAllocationPercentages.value.(tokenomicstypes.MintAllocationPercentages)
		}

		mintEqualsBurnClaimDistribution, ok := paramsMap[tokenomicstypes.ParamMintEqualsBurnClaimDistribution]
		if ok {
			params.MintEqualsBurnClaimDistribution = mintEqualsBurnClaimDistribution.value.(tokenomicstypes.MintEqualsBurnClaimDistribution)
		}

		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&tokenomicstypes.QueryParamsResponse{
				Params: params,
			},
		)

	case prooftypes.ModuleName:
		params := prooftypes.DefaultParams()
		paramsMap := s.expectedModuleParams[moduleName]

		proofRequestProbability, ok := paramsMap[prooftypes.ParamProofRequestProbability]
		if ok {
			params.ProofRequestProbability = proofRequestProbability.value.(float64)
		}

		proofRequirementThreshold, ok := paramsMap[prooftypes.ParamProofRequirementThreshold]
		if ok {
			params.ProofRequirementThreshold = proofRequirementThreshold.value.(*cosmostypes.Coin)
		}

		proofMissingPenalty, ok := paramsMap[prooftypes.ParamProofMissingPenalty]
		if ok {
			params.ProofMissingPenalty = proofMissingPenalty.value.(*cosmostypes.Coin)
		}

		proofSubmissionFee, ok := paramsMap[prooftypes.ParamProofSubmissionFee]
		if ok {
			params.ProofSubmissionFee = proofSubmissionFee.value.(*cosmostypes.Coin)
		}

		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&prooftypes.QueryParamsResponse{
				Params: params,
			},
		)
	case sharedtypes.ModuleName:
		params := sharedtypes.DefaultParams()
		paramsMap := s.expectedModuleParams[moduleName]

		numBlocksPerSessionParam, ok := paramsMap[sharedtypes.ParamNumBlocksPerSession]
		if ok {
			params.NumBlocksPerSession = uint64(numBlocksPerSessionParam.value.(int64))
		}

		gracePeriodEndOffsetBlocksParam, ok := paramsMap[sharedtypes.ParamGracePeriodEndOffsetBlocks]
		if ok {
			params.GracePeriodEndOffsetBlocks = uint64(gracePeriodEndOffsetBlocksParam.value.(int64))
		}

		claimWindowOpenOffsetBlocksParam, ok := paramsMap[sharedtypes.ParamClaimWindowOpenOffsetBlocks]
		if ok {
			params.ClaimWindowOpenOffsetBlocks = uint64(claimWindowOpenOffsetBlocksParam.value.(int64))
		}

		claimWindowCloseOffsetBlocksParam, ok := paramsMap[sharedtypes.ParamClaimWindowCloseOffsetBlocks]
		if ok {
			params.ClaimWindowCloseOffsetBlocks = uint64(claimWindowCloseOffsetBlocksParam.value.(int64))
		}

		proofWindowOpenOffsetBlocksParam, ok := paramsMap[sharedtypes.ParamProofWindowOpenOffsetBlocks]
		if ok {
			params.ProofWindowOpenOffsetBlocks = uint64(proofWindowOpenOffsetBlocksParam.value.(int64))
		}

		proofWindowCloseOffsetBlocksParam, ok := paramsMap[sharedtypes.ParamProofWindowCloseOffsetBlocks]
		if ok {
			params.ProofWindowCloseOffsetBlocks = uint64(proofWindowCloseOffsetBlocksParam.value.(int64))
		}

		supplierUnbondingPeriodSessions, ok := paramsMap[sharedtypes.ParamSupplierUnbondingPeriodSessions]
		if ok {
			params.SupplierUnbondingPeriodSessions = uint64(supplierUnbondingPeriodSessions.value.(int64))
		}

		applicationUnbondingPeriodSessions, ok := paramsMap[sharedtypes.ParamApplicationUnbondingPeriodSessions]
		if ok {
			params.ApplicationUnbondingPeriodSessions = uint64(applicationUnbondingPeriodSessions.value.(int64))
		}

		gatewayUnbondingPeriodSessions, ok := paramsMap[sharedtypes.ParamGatewayUnbondingPeriodSessions]
		if ok {
			params.GatewayUnbondingPeriodSessions = uint64(gatewayUnbondingPeriodSessions.value.(int64))
		}

		computeUnitsToTokensMultiplier, ok := paramsMap[sharedtypes.ParamComputeUnitsToTokensMultiplier]
		if ok {
			params.ComputeUnitsToTokensMultiplier = uint64(computeUnitsToTokensMultiplier.value.(int64))
		}

		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&sharedtypes.QueryParamsResponse{
				Params: params,
			},
		)
	case apptypes.ModuleName:
		maxDelegatedGateways := s.expectedModuleParams[moduleName][apptypes.ParamMaxDelegatedGateways].value.(uint64)
		minStake := s.expectedModuleParams[moduleName][apptypes.ParamMinStake].value.(*cosmostypes.Coin)
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&apptypes.QueryParamsResponse{
				Params: apptypes.Params{
					MaxDelegatedGateways: maxDelegatedGateways,
					MinStake:             minStake,
				},
			},
		)
	case servicetypes.ModuleName:
		addServiceFee := s.expectedModuleParams[moduleName][servicetypes.ParamAddServiceFee].value.(*cosmostypes.Coin)
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&servicetypes.QueryParamsResponse{
				Params: servicetypes.Params{
					AddServiceFee: addServiceFee,
				},
			},
		)
	case suppliertypes.ModuleName:
		minStake := s.expectedModuleParams[moduleName][suppliertypes.ParamMinStake].value.(*cosmostypes.Coin)
		stakingFee := s.expectedModuleParams[moduleName][suppliertypes.ParamStakingFee].value.(*cosmostypes.Coin)
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&suppliertypes.QueryParamsResponse{
				Params: suppliertypes.Params{
					MinStake:   minStake,
					StakingFee: stakingFee,
				},
			},
		)
	default:
		s.Fatalf("ERROR: unexpected module name %q", moduleName)
	}
}

// sendAuthzExecToUpdateAllModuleParams constructs and sends an authz exec
// tx to update all params for moduleName the given params.
func (s *suite) sendAuthzExecToUpdateAllModuleParams(accountName, moduleName string, params paramsAnyMap) {
	// NB: set s#moduleParamsMap for later assertion.
	s.expectedModuleParams = moduleParamsMap{
		moduleName: params,
	}

	// Use the map of params to populate a tx JSON template & write it to a file.
	txJSONFile := s.newTempUpdateParamsTxJSONFile(s.expectedModuleParams)

	// Send the authz exec tx to update all module params.
	s.sendAuthzExecTx(accountName, txJSONFile.Name())
}

// assertUpdatedParams deserializes the param query response JSON into a
// MsgUpdateParams of type P & asserts that it matches the expected params.
func assertUpdatedParams[P cosmostypes.Msg](
	s *suite,
	queryParamsResJSON []byte,
	expectedParamsRes P,
) {
	s.Helper()

	queryParamsMsgValue := reflect.New(reflect.TypeOf(expectedParamsRes).Elem())
	queryParamsMsg := queryParamsMsgValue.Interface().(P)
	err := s.cdc.UnmarshalJSON(queryParamsResJSON, queryParamsMsg)
	require.NoError(s, err)
	require.EqualValues(s, expectedParamsRes, queryParamsMsg)
}
