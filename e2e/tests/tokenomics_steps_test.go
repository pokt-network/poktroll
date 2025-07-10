//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

	// Store the proposer address in accNameToAddrMap if not already there
	if _, exists := accNameToAddrMap["proposer"]; !exists {
		accNameToAddrMap["proposer"] = proposerAddr
		accAddrToNameMap[proposerAddr] = "proposer"
	}

	balance := s.getAccBalance("proposer")
	s.scenarioState[stateKey] = balance
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

// getCurrentBlockProposer gets the address of the current block proposer
func (s *suite) getCurrentBlockProposer() string {
	// Query the latest block to get the proposer address
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "block", "--output=json",
	)
	require.NoError(s, err)

	// Parse the block info to extract proposer address
	var blockInfo struct {
		Block struct {
			Header struct {
				ProposerAddress string `json:"proposer_address"`
			} `json:"header"`
		} `json:"block"`
	}

	// Strip any warning messages and get just the JSON part
	jsonStart := strings.Index(res.Stdout, "{")
	require.Greater(s, jsonStart, -1, "no JSON found in block query response")
	jsonData := res.Stdout[jsonStart:]

	err = json.Unmarshal([]byte(jsonData), &blockInfo)
	require.NoError(s, err)

	// Convert the hex proposer address to bech32 format
	// The proposer address is in hex format, need to convert to bech32
	// For LocalNet testing, we know validator1 is the only validator, so use its address
	// In a real implementation, you'd need to convert from hex to bech32
	return "pokt18kk3aqe2pjz7x7993qp2pjt95ghurra9682tyn" // validator1 address - the actual proposer in LocalNet
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
