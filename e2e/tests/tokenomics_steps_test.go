//go:build e2e

package e2e

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TheUserRemembersTheBalanceOfAs stores the current balance of an account in the scenario state
func (s *suite) TheUserRemembersTheBalanceOfAs(accName, stateKey string) {
	balance := s.getAccBalance(accName)
	s.scenarioState[stateKey] = balance
}

// TheUserRemembersTheBalanceOfTheDaoAs stores the current balance of the DAO in the scenario state
func (s *suite) TheUserRemembersTheBalanceOfTheDaoAs(stateKey string) {
	// Get the DAO reward address from tokenomics params
	tokenomicsParams := s.getTokenomicsParams()
	daoAddress := tokenomicsParams.DaoRewardAddress

	// Store the DAO address in accNameToAddrMap if not already there
	if _, exists := accNameToAddrMap["dao"]; !exists {
		accNameToAddrMap["dao"] = daoAddress
		accAddrToNameMap[daoAddress] = "dao"
	}

	balance := s.getAccBalance("dao")
	s.scenarioState[stateKey] = balance
}

// TheUserRemembersTheBalanceOfTheProposerAs stores the current balance of the block proposer in the scenario state
func (s *suite) TheUserRemembersTheBalanceOfTheProposerAs(stateKey string) {
	// Get the current block proposer address
	proposerAddr := s.getCurrentBlockProposer()

	// DEV_NOTE: fund the proposer with a MACT so that the s.getAccBalance check doesn't fail the test.
	// s.fundAddress(proposerAddr, cosmostypes.NewInt64Coin(pocket.DenomMACT, 1))

	// Store the proposer address in accNameToAddrMap if not already there
	if _, exists := accNameToAddrMap["proposer"]; !exists {
		accNameToAddrMap["proposer"] = proposerAddr
		accAddrToNameMap[proposerAddr] = "proposer"
	}

	balance := s.getAccBalance("proposer")
	s.scenarioState[stateKey] = balance
	require.GreaterOrEqual(s, balance, int64(0))
}

// TheUserRemembersTheBalanceOfTheServiceOwnerForAs stores the current balance of a service owner in the scenario state
func (s *suite) TheUserRemembersTheBalanceOfTheServiceOwnerForAs(serviceId, stateKey string) {
	// Get the service owner address
	service := s.getService(serviceId)
	serviceOwnerAddr := service.OwnerAddress

	// Create a unique name for this service owner
	serviceOwnerName := fmt.Sprintf("service_owner_%s", serviceId)

	// Store the service owner address in accNameToAddrMap if not already there
	if _, exists := accNameToAddrMap[serviceOwnerName]; !exists {
		accNameToAddrMap[serviceOwnerName] = serviceOwnerAddr
		accAddrToNameMap[serviceOwnerAddr] = serviceOwnerName
	}

	balance := s.getAccBalance(serviceOwnerName)
	s.scenarioState[stateKey] = balance
}

// TheDaoBalanceShouldBeUpoktMoreThan checks if the DAO balance increased by the expected amount
func (s *suite) TheDaoBalanceShouldBeUpoktMoreThan(expectedIncreaseStr, prevBalanceKey string) {
	expectedIncrease, err := strconv.ParseInt(expectedIncreaseStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance("dao")

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedIncrease, "dao", "more", "balance")
}

// TheDaoBalanceShouldBeUpoktThan checks if the DAO balance changed by the expected amount in the specified direction
func (s *suite) TheDaoBalanceShouldBeUpoktThan(expectedChangeStr, direction, prevBalanceKey string) {
	expectedChange, err := strconv.ParseInt(expectedChangeStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance("dao")

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedChange, "dao", direction, "balance")
}

// TheProposerBalanceShouldBeUpoktMoreThan checks if the proposer balance increased by the expected amount
func (s *suite) TheProposerBalanceShouldBeUpoktMoreThan(expectedIncreaseStr, prevBalanceKey string) {
	expectedIncrease, err := strconv.ParseInt(expectedIncreaseStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance("proposer")

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedIncrease, "proposer", "more", "balance")
}

// TheProposerBalanceShouldBeUpoktThan checks if the proposer balance changed by the expected amount in the specified direction
func (s *suite) TheProposerBalanceShouldBeUpoktThan(expectedChangeStr, direction, prevBalanceKey string) {
	expectedChange, err := strconv.ParseInt(expectedChangeStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance("proposer")

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedChange, "proposer", direction, "balance")
}

// TheServiceOwnerBalanceForShouldBeUpoktMoreThan checks if the service owner balance increased by the expected amount
func (s *suite) TheServiceOwnerBalanceForShouldBeUpoktMoreThan(serviceId, expectedIncreaseStr, prevBalanceKey string) {
	expectedIncrease, err := strconv.ParseInt(expectedIncreaseStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	serviceOwnerName := fmt.Sprintf("service_owner_%s", serviceId)
	currBalance := s.getAccBalance(serviceOwnerName)

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedIncrease, serviceOwnerName, "more", "balance")
}

// TheServiceOwnerBalanceForShouldBeUpoktThan checks if the service owner balance changed by the expected amount in the specified direction
func (s *suite) TheServiceOwnerBalanceForShouldBeUpoktThan(serviceId, expectedChangeStr, direction, prevBalanceKey string) {
	expectedChange, err := strconv.ParseInt(expectedChangeStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	serviceOwnerName := fmt.Sprintf("service_owner_%s", serviceId)
	currBalance := s.getAccBalance(serviceOwnerName)

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedChange, serviceOwnerName, direction, "balance")
}

// TheAccountBalanceOfShouldBeUpoktMoreThan validates that an account balance increased compared to a remembered balance
func (s *suite) TheAccountBalanceOfShouldBeUpoktMoreThan(accName, expectedIncreaseStr, prevBalanceKey string) {
	expectedIncrease, err := strconv.ParseInt(expectedIncreaseStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance(accName)

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedIncrease, accName, "more", "balance")
}

// TheAccountBalanceOfShouldBeUpoktThan validates that an account balance changed by the expected amount in the specified direction
func (s *suite) TheAccountBalanceOfShouldBeUpoktThan(accName, expectedChangeStr, direction, prevBalanceKey string) {
	expectedChange, err := strconv.ParseInt(expectedChangeStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance(accName)

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedChange, accName, direction, "balance")
}

// Helper methods

// getTokenomicsParams queries and returns the current tokenomics module parameters
func (s *suite) getTokenomicsParams() tokenomicstypes.Params {
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "tokenomics", "params", "--output=json",
	)
	require.NoError(s, err)

	var paramsRes tokenomicstypes.QueryParamsResponse
	err = s.cdc.UnmarshalJSON([]byte(res.Stdout), &paramsRes)
	require.NoError(s, err)

	return paramsRes.Params
}

// getCurrentBlockProposer gets the operator address of the current block proposer
func (s *suite) getCurrentBlockProposer() string {
	// Step 1: Query the latest block to get the proposer address
	blockRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "block", "--output=json",
	)
	require.NoError(s, err)

	// Parse the block info
	var blockInfo struct {
		Header struct {
			ProposerAddress string `json:"proposer_address"`
		} `json:"header"`
	}
	jsonBytes := s.stdOutToJSONBytes(blockRes.Stdout)
	err = json.Unmarshal(jsonBytes, &blockInfo)
	require.NoError(s, err)
	require.NotEmpty(s, blockInfo.Header.ProposerAddress)

	// Step 2: Query all validators
	validatorsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "staking", "validators", "--output=json",
	)
	require.NoError(s, err)

	// Parse validators
	var validatorsInfo struct {
		Validators []struct {
			OperatorAddress string `json:"operator_address"`
			ConsensusPubkey struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"consensus_pubkey"`
		} `json:"validators"`
	}
	jsonBytes = s.stdOutToJSONBytes(validatorsRes.Stdout)
	err = json.Unmarshal(jsonBytes, &validatorsInfo)
	require.NoError(s, err)

	// Step 3: Decode proposer address
	proposerAddrBytes, err := base64.StdEncoding.DecodeString(blockInfo.Header.ProposerAddress)
	require.NoError(s, err)

	// Step 4: Find matching validator
	for _, validator := range validatorsInfo.Validators {
		// Decode validator's consensus pubkey
		pubkeyBytes, err := base64.StdEncoding.DecodeString(validator.ConsensusPubkey.Value)
		if err != nil {
			continue
		}
		// Compute address from pubkey (SHA256 then take first 20 bytes)
		hash := sha256.Sum256(pubkeyBytes)
		validatorConsAddr := hash[:20]

		// Compare with proposer address
		if bytes.Equal(validatorConsAddr, proposerAddrBytes) {
			valAddr, err := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
			require.NoError(s, err)

			// Convert validator address to account address (same bytes, different prefix)
			accAddr := cosmostypes.AccAddress(valAddr)
			return accAddr.String()
		}
	}

	require.Fail(s, "could not find validator for proposer address")
	return ""
}

func (s *suite) stdOutToJSONBytes(stdOut string) []byte {
	jsonStart := strings.Index(stdOut, "{")
	require.Greater(s, jsonStart, -1)
	jsonData := stdOut[jsonStart:]
	return []byte(jsonData)
}

// getService queries and returns the service with the given ID
func (s *suite) getService(serviceId string) *sharedtypes.Service {
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "service", "show-service", serviceId, "--output=json",
	)
	require.NoError(s, err)

	var response servicetypes.QueryGetServiceResponse
	err = s.cdc.UnmarshalJSON([]byte(res.Stdout), &response)
	require.NoError(s, err)

	return &response.Service
}
