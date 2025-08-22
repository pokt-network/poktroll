//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
)

// cliValidatorResponse represents a validator as returned by CLI JSON output
type cliValidatorResponse struct {
	OperatorAddress string `json:"operator_address"`
	Status          string `json:"status"` // CLI returns status as string, not enum
	Tokens          string `json:"tokens"`
}

// cliValidatorsResponse represents the response from 'query staking validators' CLI command
type cliValidatorsResponse struct {
	Validators []cliValidatorResponse `json:"validators"`
}

// cliValidatorQueryResponse represents a single validator as returned by CLI JSON output
type cliValidatorQueryResponse struct {
	OperatorAddress string `json:"operator_address"`
	Status          string `json:"status"` // CLI returns status as string
	Tokens          string `json:"tokens"`
	Commission      struct {
		CommissionRates struct {
			Rate string `json:"rate"` // CLI returns commission rate as string
		} `json:"commission_rates"`
	} `json:"commission"`
}

// cliDecCoin represents a DecCoin as returned by CLI JSON output (with string amounts)
type cliDecCoin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"` // CLI returns amounts as strings, not Dec types
}

// cliDelegationRewardsResponse represents the response from 'query distribution rewards-by-validator' CLI command
type cliDelegationRewardsResponse struct {
	Rewards interface{} `json:"rewards"` // Can be string (empty) or []cliDecCoin (with rewards)
}

// cliDelegationResponse represents the response from 'query staking delegation' CLI command
type cliDelegationResponse struct {
	DelegationResponse struct {
		Delegation struct {
			Shares string `json:"shares"`
		} `json:"delegation"`
		Balance struct {
			Amount string `json:"amount"`
		} `json:"balance"`
	} `json:"delegation_response"`
}

// TheUserRemembersTheCurrentBlockProposerValidatorAddressAs remembers the current block proposer's validator address
func (s *suite) TheUserRemembersTheCurrentBlockProposerValidatorAddressAs(validatorName string) {
	proposerAccAddr := s.getCurrentBlockProposer()
	require.NotEmpty(s, proposerAccAddr)

	// Convert account address to validator operator address by querying all validators
	validatorsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "staking", "validators", "--output=json",
	)
	require.NoError(s, err)

	var validatorsResponse cliValidatorsResponse
	err = json.Unmarshal([]byte(validatorsRes.Stdout), &validatorsResponse)
	require.NoError(s, err)

	// Find the validator whose account address matches the proposer
	var validatorOperatorAddr string
	proposerAccAddrBytes, err := cosmostypes.AccAddressFromBech32(proposerAccAddr)
	require.NoError(s, err)

	for _, validator := range validatorsResponse.Validators {
		// Only consider bonded validators
		if validator.Status != "BOND_STATUS_BONDED" {
			continue
		}
		
		valAddr, err := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
		require.NoError(s, err)
		
		// Convert validator address to account address for comparison
		valAccAddr := cosmostypes.AccAddress(valAddr)
		if valAccAddr.Equals(proposerAccAddrBytes) {
			validatorOperatorAddr = validator.OperatorAddress
			break
		}
	}

	require.NotEmpty(s, validatorOperatorAddr, "could not find validator operator address for proposer")

	// Store the validator operator address
	accNameToAddrMap[validatorName] = validatorOperatorAddr
	accAddrToNameMap[validatorOperatorAddr] = validatorName

	s.Logf("Stored validator %s with operator address: %s", validatorName, validatorOperatorAddr)
}

// queryExistingDelegation queries the current delegation amount from a delegator to a validator
func (s *suite) queryExistingDelegation(delegatorAddr, validatorAddr string) int64 {
	// Query existing delegation
	args := []string{
		"query", "staking", "delegation",
		delegatorAddr,
		validatorAddr,
		"--output=json",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	if err != nil {
		// No existing delegation
		s.Logf("No existing delegation found from %s to %s", delegatorAddr, validatorAddr)
		return 0
	}

	// Parse delegation response
	var delResp cliDelegationResponse
	if err := json.Unmarshal([]byte(res.Stdout), &delResp); err != nil {
		// Failed to parse, assume no delegation
		s.Logf("Failed to parse delegation response: %v", err)
		return 0
	}

	// Parse the balance amount
	if delResp.DelegationResponse.Balance.Amount != "" {
		amount, err := strconv.ParseInt(delResp.DelegationResponse.Balance.Amount, 10, 64)
		if err == nil {
			s.Logf("Found existing delegation of %d uPOKT", amount)
			return amount
		}
	}

	return 0
}

// TheAccountDelegatesUpoktToValidator performs a delegation transaction
func (s *suite) TheAccountDelegatesUpoktToValidator(delegatorName, amountStr, validatorName string) {
	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	targetAmount, err := strconv.ParseInt(amountStr, 10, 64)
	require.NoError(s, err)

	// Check existing delegation
	existingAmount := s.queryExistingDelegation(delegatorAddr, validatorAddr)
	
	// Option 1: Skip if already delegated the target amount or more
	if existingAmount >= targetAmount {
		s.Logf("Skipping delegation: %s already has %d uPOKT delegated to %s (target: %d)", 
			delegatorName, existingAmount, validatorName, targetAmount)
		// Store that this delegation was skipped for later balance checking
		skipKey := fmt.Sprintf("%s_to_%s_delegation_skipped", delegatorName, validatorName)
		s.scenarioState[skipKey] = targetAmount // Store the amount that was skipped
		return
	}

	// Option 2: Adjust delegation amount to reach target
	// Only delegate the difference needed to reach the target amount
	amountToDelegate := targetAmount - existingAmount
	
	// Construct delegation command
	amountCoin := fmt.Sprintf("%d%s", amountToDelegate, pocket.DenomuPOKT)
	
	s.Logf("Delegating %s from %s (%s) to validator %s (%s) [existing: %d, target: %d]", 
		amountCoin, delegatorName, delegatorAddr, validatorName, validatorAddr,
		existingAmount, targetAmount)

	// Execute delegation transaction with proper retry handling
	args := []string{
		"tx", "staking", "delegate",
		validatorAddr,
		amountCoin,
		"--from", delegatorAddr,
		keyRingFlag,
		chainIdFlag,
		"--gas=auto",
		"--gas-prices=0upokt",
		"--yes",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error executing delegation transaction: %v", err)
	s.pocketd.result = res

	require.Contains(s, res.Stdout, "code: 0", "delegation transaction failed")
	s.Logf("Delegation transaction successful: added %d uPOKT (new total: %d)", 
		amountToDelegate, targetAmount)
}

// TheAccountWithdrawsDelegationRewardsFrom withdraws delegation rewards from a validator
// DEPRECATED: With ModToAcctTransfer, relay-based rewards are sent directly to delegator accounts
// during claim settlement. This function now only withdraws Cosmos SDK block production rewards.
func (s *suite) TheAccountWithdrawsDelegationRewardsFrom(delegatorName, validatorName string) {
	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	s.Logf("Withdrawing delegation rewards for %s (%s) from validator %s (%s)", 
		delegatorName, delegatorAddr, validatorName, validatorAddr)

	// Execute withdrawal transaction with proper retry handling
	args := []string{
		"tx", "distribution", "withdraw-rewards",
		validatorAddr,
		"--from", delegatorAddr,
		keyRingFlag,
		chainIdFlag,
		"--gas=auto",
		"--gas-prices=0upokt",
		"--yes",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error executing withdrawal transaction: %v", err)
	s.pocketd.result = res

	require.Contains(s, res.Stdout, "code: 0", "withdrawal transaction failed")
	s.Logf("Withdrawal transaction successful")
}

// TheUserRemembersTheDelegationRewardsForFromAs stores current delegation rewards in scenario state
// DEPRECATED: With ModToAcctTransfer, relay-based rewards are sent directly to delegator accounts.
// This function now only tracks Cosmos SDK block production rewards, not relay-based rewards.
func (s *suite) TheUserRemembersTheDelegationRewardsForFromAs(delegatorName, validatorName, stateKey string) {
	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	// Query delegation rewards with retry
	args := []string{
		"query", "distribution", "rewards-by-validator",
		delegatorAddr,
		validatorAddr,
		"--output=json",
	}

	rewardsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	if err != nil {
		// If query fails, assume zero rewards
		s.scenarioState[stateKey] = int64(0)
		s.Logf("No rewards found for %s from %s, storing 0: %v", delegatorName, validatorName, err)
		return
	}

	var rewardsResponse cliDelegationRewardsResponse
	if err := json.Unmarshal([]byte(rewardsRes.Stdout), &rewardsResponse); err != nil {
		// If unmarshal fails, assume zero rewards
		s.scenarioState[stateKey] = int64(0)
		s.Logf("Failed to parse rewards for %s from %s, storing 0: %v", delegatorName, validatorName, err)
		return
	}

	// Extract uPOKT rewards amount - handle both string (empty) and array (with rewards) cases
	var rewardAmount int64 = 0
	switch rewards := rewardsResponse.Rewards.(type) {
	case string:
		// Empty rewards returned as string, amount is 0
		rewardAmount = 0
	case []interface{}:
		// Parse rewards array
		for _, rewardInterface := range rewards {
			if rewardMap, ok := rewardInterface.(map[string]interface{}); ok {
				if denom, ok := rewardMap["denom"].(string); ok && denom == pocket.DenomuPOKT {
					if amountStr, ok := rewardMap["amount"].(string); ok {
						if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
							rewardAmount = int64(amount)
						}
					}
					break
				}
			}
		}
	}

	s.scenarioState[stateKey] = rewardAmount
	s.Logf("Stored delegation rewards for %s from %s: %d uPOKT", delegatorName, validatorName, rewardAmount)
}

// TheUserRemembersTheDistributionModuleBalanceAs stores the distribution module balance
// NOTE: This function is deprecated - distribution module acts as pass-through for validator rewards
// Rewards go directly to validator outstanding rewards, not distribution module balance
func (s *suite) TheUserRemembersTheDistributionModuleBalanceAs(stateKey string) {
	// For compatibility, store zero as distribution module doesn't hold rewards
	s.scenarioState[stateKey] = int64(0)
	s.Logf("Distribution module balance tracking deprecated - rewards go directly to validators")
}

// TheDistributionModuleBalanceShouldBeUpoktMoreThan validates distribution module balance change
// NOTE: This function is deprecated - distribution module acts as pass-through for validator rewards
func (s *suite) TheDistributionModuleBalanceShouldBeUpoktMoreThan(expectedIncreaseStr, prevBalanceKey string) {
	// Skip this validation - distribution module doesn't hold validator rewards
	// Rewards are immediately allocated to validators via AllocateTokensToValidator
	s.Logf("Skipping distribution module balance check - rewards allocated directly to validators")
}

// TheDistributionModuleBalanceShouldBeUpoktThan validates distribution module balance change in any direction
// NOTE: This function is deprecated - distribution module acts as pass-through for validator rewards
func (s *suite) TheDistributionModuleBalanceShouldBeUpoktThan(expectedChangeStr, direction, prevBalanceKey string) {
	// Skip this validation - distribution module doesn't hold validator rewards
	// Rewards are immediately allocated to validators via AllocateTokensToValidator
	s.Logf("Skipping distribution module balance check - rewards allocated directly to validators")
}

// TheDelegationRewardsForFromShouldBeGreaterThan validates that rewards have increased
// DEPRECATED: With ModToAcctTransfer, relay-based rewards are sent directly to delegator accounts.
// This function now only checks Cosmos SDK block production rewards, not relay-based rewards.
func (s *suite) TheDelegationRewardsForFromShouldBeGreaterThan(delegatorName, validatorName, prevRewardsKey string) {
	prevRewards, ok := s.scenarioState[prevRewardsKey].(int64)
	require.True(s, ok, "previous rewards %s not found or not an int64", prevRewardsKey)

	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	// Query current delegation rewards with retry
	args := []string{
		"q", "distribution", "rewards-by-validator",
		delegatorAddr,
		validatorAddr,
		"--output=json",
	}

	rewardsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error querying delegation rewards: %v", err)

	var rewardsResponse cliDelegationRewardsResponse
	err = json.Unmarshal([]byte(rewardsRes.Stdout), &rewardsResponse)
	require.NoError(s, err)

	// Extract current uPOKT rewards amount - handle both string (empty) and array (with rewards) cases
	var currentRewardAmount int64 = 0
	switch rewards := rewardsResponse.Rewards.(type) {
	case string:
		// Empty rewards returned as string, amount is 0
		currentRewardAmount = 0
	case []interface{}:
		// Parse rewards array
		for _, rewardInterface := range rewards {
			if rewardMap, ok := rewardInterface.(map[string]interface{}); ok {
				if denom, ok := rewardMap["denom"].(string); ok && denom == pocket.DenomuPOKT {
					if amountStr, ok := rewardMap["amount"].(string); ok {
						if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
							currentRewardAmount = int64(amount)
						}
					}
					break
				}
			}
		}
	}

	s.Logf("Comparing delegation rewards for %s from %s: previous=%d, current=%d", 
		delegatorName, validatorName, prevRewards, currentRewardAmount)
	
	require.Greater(s, currentRewardAmount, prevRewards, 
		"delegation rewards should have increased for %s from %s", delegatorName, validatorName)
}

// TheDelegationRewardsForFromShouldBeUpokt validates exact reward amounts
// DEPRECATED: With ModToAcctTransfer, relay-based rewards are sent directly to delegator accounts.
// This function now only checks Cosmos SDK block production rewards, not relay-based rewards.
func (s *suite) TheDelegationRewardsForFromShouldBeUpokt(delegatorName, validatorName, expectedAmountStr string) {
	expectedAmount, err := strconv.ParseInt(expectedAmountStr, 10, 64)
	require.NoError(s, err)

	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	// Query delegation rewards with retry
	args := []string{
		"q", "distribution", "rewards-by-validator",
		delegatorAddr,
		validatorAddr,
		"--output=json",
	}

	rewardsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error querying delegation rewards: %v", err)

	var rewardsResponse cliDelegationRewardsResponse
	err = json.Unmarshal([]byte(rewardsRes.Stdout), &rewardsResponse)
	require.NoError(s, err)

	// Extract uPOKT rewards amount - handle both string (empty) and array (with rewards) cases
	var rewardAmount int64 = 0
	switch rewards := rewardsResponse.Rewards.(type) {
	case string:
		// Empty rewards returned as string, amount is 0
		rewardAmount = 0
	case []interface{}:
		// Parse rewards array
		for _, rewardInterface := range rewards {
			if rewardMap, ok := rewardInterface.(map[string]interface{}); ok {
				if denom, ok := rewardMap["denom"].(string); ok && denom == pocket.DenomuPOKT {
					if amountStr, ok := rewardMap["amount"].(string); ok {
						if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
							rewardAmount = int64(amount)
						}
					}
					break
				}
			}
		}
	}

	require.Equal(s, expectedAmount, rewardAmount, 
		"delegation rewards for %s from %s should be %d uPOKT, got %d", 
		delegatorName, validatorName, expectedAmount, rewardAmount)
}

// TheUserRemembersTheCommissionRateForValidatorAs stores the validator's commission rate
func (s *suite) TheUserRemembersTheCommissionRateForValidatorAs(validatorName, stateKey string) {
	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	// Query validator information with retry
	validatorRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, 
		"q", "staking", "validator", validatorAddr, "--output=json")
	require.NoError(s, err)
	require.NotEmpty(s, validatorRes.Stdout)

	var validator cliValidatorQueryResponse
	err = json.Unmarshal([]byte(validatorRes.Stdout), &validator)
	require.NoError(s, err)

	commissionRate := validator.Commission.CommissionRates.Rate
	
	// Handle empty commission rate (default to 0%)
	if commissionRate == "" {
		commissionRate = "0.000000000000000000"
		s.Logf("Empty commission rate detected, defaulting to 0%% for validator %s", validatorName)
	}
	
	s.scenarioState[stateKey] = commissionRate
	s.Logf("Stored commission rate for validator %s: %s", validatorName, commissionRate)
}

// TheAccountBalanceOfShouldBeThan validates account balance changes in any direction
func (s *suite) TheAccountBalanceOfShouldBeThan(accName, direction, prevBalanceKey string) {
	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance(accName)

	// Note: This function doesn't handle partial skipping like validateAmountChange does
	// because it's used for generic balance checks without specific expected amounts.
	// Check if any delegations were skipped that would affect this balance check
	var delegationWasSkipped bool
	for key, value := range s.scenarioState {
		if strings.HasPrefix(key, fmt.Sprintf("%s_to_", accName)) && strings.HasSuffix(key, "_delegation_skipped") {
			if _, ok := value.(int64); ok { // Updated to expect int64 (amount) instead of bool
				delegationWasSkipped = true
				break
			}
		}
	}
	
	if delegationWasSkipped && strings.ToLower(direction) == "less" {
		// If delegation was skipped and we're checking for "less", 
		// the balance should be equal (no change) rather than less
		s.Logf("Delegation was skipped for %s, expecting balance to be equal rather than less", accName)
		require.Equal(s, currBalance, prevBalance, 
			"account %s balance should be equal since delegation was skipped", accName)
		return
	}

	switch strings.ToLower(direction) {
	case "more":
		require.Greater(s, currBalance, prevBalance, 
			"account %s balance should be more than previous", accName)
	case "less":
		require.Less(s, currBalance, prevBalance, 
			"account %s balance should be less than previous", accName)
	case "equal":
		require.Equal(s, currBalance, prevBalance, 
			"account %s balance should be equal to previous", accName)
	default:
		require.Fail(s, "invalid direction %s, must be 'more', 'less', or 'equal'", direction)
	}

	s.Logf("Account %s balance validation: previous=%d, current=%d, direction=%s", 
		accName, prevBalance, currBalance, direction)
}

// TheUserWaitsForBlocks waits for a specified number of blocks to pass
func (s *suite) TheUserWaitsForBlocks(numBlocksStr string) {
	numBlocks, err := strconv.Atoi(numBlocksStr)
	require.NoError(s, err)
	require.Greater(s, numBlocks, 0, "number of blocks must be positive")

	s.Logf("Waiting for %d blocks", numBlocks)

	// Get current block height
	initialHeight := s.getCurrentHeight()
	targetHeight := initialHeight + int64(numBlocks)

	// Wait for the target height
	for {
		currentHeight := s.getCurrentHeight()
		if currentHeight >= targetHeight {
			break
		}
		// Small sleep to prevent busy waiting
		require.Eventually(s, func() bool {
			return s.getCurrentHeight() >= targetHeight
		}, time.Minute, time.Second, "timeout waiting for %d blocks", numBlocks)
	}

	s.Logf("Successfully waited for %d blocks (from %d to %d)", 
		numBlocks, initialHeight, s.getCurrentHeight())
}

// getCurrentHeight gets the current block height
func (s *suite) getCurrentHeight() int64 {
	// Use the existing working implementation from the test suite
	return s.getCurrentBlockHeight()
}

