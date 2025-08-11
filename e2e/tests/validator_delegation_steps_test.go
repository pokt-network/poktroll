//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
)

// TheUserGetsTheCurrentBlockProposerValidatorAddressAs gets the current block proposer's validator address
func (s *suite) TheUserGetsTheCurrentBlockProposerValidatorAddressAs(validatorName string) {
	// Reuse existing function to get proposer account address
	proposerAccAddr := s.getCurrentBlockProposer()
	require.NotEmpty(s, proposerAccAddr)

	// Convert account address to validator operator address by querying all validators
	validatorsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "staking", "validators", "--output=json",
	)
	require.NoError(s, err)

	var validatorsResponse stakingtypes.QueryValidatorsResponse
	err = json.Unmarshal([]byte(validatorsRes.Stdout), &validatorsResponse)
	require.NoError(s, err)

	// Find the validator whose account address matches the proposer
	var validatorOperatorAddr string
	proposerAccAddrBytes, err := cosmostypes.AccAddressFromBech32(proposerAccAddr)
	require.NoError(s, err)

	for _, validator := range validatorsResponse.Validators {
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

// TheAccountDelegatesUpoktToValidator performs a delegation transaction
func (s *suite) TheAccountDelegatesUpoktToValidator(delegatorName, amountStr, validatorName string) {
	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	require.NoError(s, err)

	// Construct delegation command
	amountCoin := fmt.Sprintf("%d%s", amount, pocket.DenomuPOKT)
	
	s.Logf("Delegating %s from %s (%s) to validator %s (%s)", 
		amountCoin, delegatorName, delegatorAddr, validatorName, validatorAddr)

	// Execute delegation transaction with proper retry handling
	args := []string{
		"tx", "staking", "delegate",
		validatorAddr,
		amountCoin,
		"--from", delegatorAddr,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error executing delegation transaction: %v", err)
	s.pocketd.result = res

	require.Contains(s, res.Stdout, "code: 0", "delegation transaction failed")
	s.Logf("Delegation transaction successful")
}

// TheAccountWithdrawsDelegationRewardsFrom withdraws delegation rewards from a validator
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
		"--yes",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error executing withdrawal transaction: %v", err)
	s.pocketd.result = res

	require.Contains(s, res.Stdout, "code: 0", "withdrawal transaction failed")
	s.Logf("Withdrawal transaction successful")
}

// TheUserRemembersTheDelegationRewardsForFromAs stores current delegation rewards in scenario state
func (s *suite) TheUserRemembersTheDelegationRewardsForFromAs(delegatorName, validatorName, stateKey string) {
	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	// Query delegation rewards with retry
	args := []string{
		"query", "distribution", "rewards",
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

	var rewardsResponse distrtypes.QueryDelegationRewardsResponse
	if err := json.Unmarshal([]byte(rewardsRes.Stdout), &rewardsResponse); err != nil {
		// If unmarshal fails, assume zero rewards
		s.scenarioState[stateKey] = int64(0)
		s.Logf("Failed to parse rewards for %s from %s, storing 0: %v", delegatorName, validatorName, err)
		return
	}

	// Extract uPOKT rewards amount
	var rewardAmount int64 = 0
	for _, reward := range rewardsResponse.Rewards {
		if reward.Denom == pocket.DenomuPOKT {
			rewardAmount = reward.Amount.TruncateInt64()
			break
		}
	}

	s.scenarioState[stateKey] = rewardAmount
	s.Logf("Stored delegation rewards for %s from %s: %d uPOKT", delegatorName, validatorName, rewardAmount)
}

// TheUserRemembersTheDistributionModuleBalanceAs stores the distribution module balance
func (s *suite) TheUserRemembersTheDistributionModuleBalanceAs(stateKey string) {
	// Create distribution module address
	distModuleAddr := authtypes.NewModuleAddress(distrtypes.ModuleName).String()
	
	// Store in account maps for reuse
	if _, exists := accNameToAddrMap["distribution_module"]; !exists {
		accNameToAddrMap["distribution_module"] = distModuleAddr
		accAddrToNameMap[distModuleAddr] = "distribution_module"
	}

	balance := s.getAccBalance("distribution_module")
	s.scenarioState[stateKey] = balance
	s.Logf("Stored distribution module balance: %d uPOKT", balance)
}

// TheDistributionModuleBalanceShouldBeUpoktMoreThan validates distribution module balance change
func (s *suite) TheDistributionModuleBalanceShouldBeUpoktMoreThan(expectedIncreaseStr, prevBalanceKey string) {
	expectedIncrease, err := strconv.ParseInt(expectedIncreaseStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance("distribution_module")

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedIncrease, "distribution_module", "more", "balance")
}

// TheDistributionModuleBalanceShouldBeUpoktThan validates distribution module balance change in any direction
func (s *suite) TheDistributionModuleBalanceShouldBeUpoktThan(expectedChangeStr, direction, prevBalanceKey string) {
	expectedChange, err := strconv.ParseInt(expectedChangeStr, 10, 64)
	require.NoError(s, err)

	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance("distribution_module")

	// Validate the change in balance
	s.validateAmountChange(prevBalance, currBalance, expectedChange, "distribution_module", direction, "balance")
}

// TheDelegationRewardsForFromShouldBeGreaterThan validates that rewards have increased
func (s *suite) TheDelegationRewardsForFromShouldBeGreaterThan(delegatorName, validatorName, prevRewardsKey string) {
	prevRewards, ok := s.scenarioState[prevRewardsKey].(int64)
	require.True(s, ok, "previous rewards %s not found or not an int64", prevRewardsKey)

	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	// Query current delegation rewards with retry
	args := []string{
		"q", "distribution", "rewards",
		delegatorAddr,
		validatorAddr,
		"--output=json",
	}

	rewardsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error querying delegation rewards: %v", err)

	var rewardsResponse distrtypes.QueryDelegationRewardsResponse
	err = json.Unmarshal([]byte(rewardsRes.Stdout), &rewardsResponse)
	require.NoError(s, err)

	// Extract current uPOKT rewards amount
	var currentRewardAmount int64 = 0
	for _, reward := range rewardsResponse.Rewards {
		if reward.Denom == pocket.DenomuPOKT {
			currentRewardAmount = reward.Amount.TruncateInt64()
			break
		}
	}

	s.Logf("Comparing delegation rewards for %s from %s: previous=%d, current=%d", 
		delegatorName, validatorName, prevRewards, currentRewardAmount)
	
	require.Greater(s, currentRewardAmount, prevRewards, 
		"delegation rewards should have increased for %s from %s", delegatorName, validatorName)
}

// TheDelegationRewardsForFromShouldBeUpokt validates exact reward amounts
func (s *suite) TheDelegationRewardsForFromShouldBeUpokt(delegatorName, validatorName, expectedAmountStr string) {
	expectedAmount, err := strconv.ParseInt(expectedAmountStr, 10, 64)
	require.NoError(s, err)

	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)

	// Query delegation rewards with retry
	args := []string{
		"q", "distribution", "rewards",
		delegatorAddr,
		validatorAddr,
		"--output=json",
	}

	rewardsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error querying delegation rewards: %v", err)

	var rewardsResponse distrtypes.QueryDelegationRewardsResponse
	err = json.Unmarshal([]byte(rewardsRes.Stdout), &rewardsResponse)
	require.NoError(s, err)

	// Extract uPOKT rewards amount
	var rewardAmount int64 = 0
	for _, reward := range rewardsResponse.Rewards {
		if reward.Denom == pocket.DenomuPOKT {
			rewardAmount = reward.Amount.TruncateInt64()
			break
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

	var validator stakingtypes.Validator
	err = json.Unmarshal([]byte(validatorRes.Stdout), &validator)
	require.NoError(s, err)

	commissionRate := validator.Commission.CommissionRates.Rate
	s.scenarioState[stateKey] = commissionRate.String()
	
	s.Logf("Stored commission rate for validator %s: %s", validatorName, commissionRate.String())
}

// TheAccountBalanceOfShouldBeThan validates account balance changes in any direction
func (s *suite) TheAccountBalanceOfShouldBeThan(accName, direction, prevBalanceKey string) {
	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance(accName)

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
	blockRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, "q", "block")
	require.NoError(s, err)
	require.NotEmpty(s, blockRes.Stdout)

	var blockResponse cliBlockQueryResponse
	err = json.Unmarshal([]byte(blockRes.Stdout), &blockResponse)
	require.NoError(s, err)

	return blockResponse.Header.Height
}

