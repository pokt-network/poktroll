//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

// cliDelegationRewardsResponse represents the response from 'query distribution rewards-by-validator' CLI command
type cliDelegationRewardsResponse struct {
	Rewards interface{} `json:"rewards"` // Can be string (empty) or array of reward objects
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
	if !exists {
		s.Logf("Skipping delegation: validator %s not found (likely skipped due to insufficient validators)", validatorName)
		// Store that this delegation was skipped
		skipKey := fmt.Sprintf("%s_to_%s_delegation_skipped", delegatorName, validatorName)
		targetAmount, err := strconv.ParseInt(amountStr, 10, 64)
		if err == nil {
			s.scenarioState[skipKey] = targetAmount
		}
		return
	}

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

// TheUserRemembersTheDelegationRewardsForFromAs stores delegation rewards (single or multiple validators)
func (s *suite) TheUserRemembersTheDelegationRewardsForFromAs(delegatorName, validatorSpec, stateKey string) {
	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	// Check if this is a single validator or comma-separated list
	if !strings.Contains(validatorSpec, ",") {
		// Single validator case
		s.querySingleValidatorRewards(delegatorAddr, delegatorName, validatorSpec, stateKey)
	} else {
		// Multiple validators case
		s.queryMultipleValidatorRewards(delegatorAddr, delegatorName, validatorSpec, stateKey)
	}
}

// querySingleValidatorRewards handles single validator reward queries
func (s *suite) querySingleValidatorRewards(delegatorAddr, delegatorName, validatorName, stateKey string) {
	validatorAddr, exists := accNameToAddrMap[validatorName]
	if !exists {
		s.scenarioState[stateKey] = int64(0)
		s.Logf("Validator %s not found, storing 0 rewards", validatorName)
		return
	}

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

// queryMultipleValidatorRewards handles multiple validator reward queries
func (s *suite) queryMultipleValidatorRewards(delegatorAddr, delegatorName, validatorSpec, stateKey string) {
	// Parse validator specification: "validator1, validator2, validator3"
	validatorNames := strings.Split(validatorSpec, ",")
	totalRewards := int64(0)
	validatorCount := 0

	for _, validatorName := range validatorNames {
		validatorName = strings.TrimSpace(validatorName)
		validatorAddr, exists := accNameToAddrMap[validatorName]
		if !exists {
			s.Logf("Skipping rewards query for validator %s (not available)", validatorName)
			continue
		}
		validatorCount++

		// Query delegation rewards with retry
		args := []string{
			"query", "distribution", "rewards-by-validator",
			delegatorAddr,
			validatorAddr,
			"--output=json",
		}

		rewardsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
		if err != nil {
			s.Logf("No rewards found for %s from %s: %v", delegatorName, validatorName, err)
			continue
		}

		var rewardsResponse cliDelegationRewardsResponse
		if err := json.Unmarshal([]byte(rewardsRes.Stdout), &rewardsResponse); err != nil {
			s.Logf("Failed to parse rewards for %s from %s: %v", delegatorName, validatorName, err)
			continue
		}

		// Extract uPOKT rewards amount
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

		totalRewards += rewardAmount
		s.Logf("Found delegation rewards for %s from %s: %d uPOKT", delegatorName, validatorName, rewardAmount)
	}

	s.scenarioState[stateKey] = totalRewards
	s.Logf("Stored total delegation rewards for %s from %d available validators: %d uPOKT", delegatorName, validatorCount, totalRewards)
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

// TheAccountBalanceOfShouldIncreaseOrRemainTheSameFrom validates that balance increases or stays the same
func (s *suite) TheAccountBalanceOfShouldIncreaseOrRemainTheSameFrom(accName, prevBalanceKey string) {
	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	currBalance := s.getAccBalance(accName)

	// Check if any multi-validator delegations were skipped
	var multiValidatorDelegationsSkipped bool
	for key := range s.scenarioState {
		if strings.HasPrefix(key, fmt.Sprintf("%s_to_validator", accName)) &&
			strings.HasSuffix(key, "_delegation_skipped") {
			multiValidatorDelegationsSkipped = true
			break
		}
	}

	if multiValidatorDelegationsSkipped {
		// If multi-validator delegations were skipped, balance should be equal or greater
		// (equal if no rewards, greater if there were rewards from the additional relay session)
		require.GreaterOrEqual(s, currBalance, prevBalance,
			"account %s balance should be greater than or equal to previous when multi-validator delegations were skipped", accName)

		if currBalance == prevBalance {
			s.Logf("Account %s balance remained the same (as expected when multi-validator delegations were skipped)", accName)
		} else {
			s.Logf("Account %s balance increased from rewards (despite multi-validator delegations being skipped)", accName)
		}
	} else {
		// Normal case: balance should increase due to rewards
		require.Greater(s, currBalance, prevBalance,
			"account %s balance should increase due to rewards", accName)
		s.Logf("Account %s balance increased from rewards", accName)
	}

	s.Logf("Account %s balance validation: previous=%d, current=%d", accName, prevBalance, currBalance)
}

// TheUserRemembersThe2ndValidatorAddressAs remembers the 2nd bonded validator's address
func (s *suite) TheUserRemembersThe2ndValidatorAddressAs(validatorName string) {
	s.theUserRemembersTheNthValidatorAddressAs(2, validatorName)
}

// TheUserRemembersThe3rdValidatorAddressAs remembers the 3rd bonded validator's address
func (s *suite) TheUserRemembersThe3rdValidatorAddressAs(validatorName string) {
	s.theUserRemembersTheNthValidatorAddressAs(3, validatorName)
}

// theUserRemembersTheNthValidatorAddressAs is a helper function for remembering the nth bonded validator's address
func (s *suite) theUserRemembersTheNthValidatorAddressAs(validatorIndex int, validatorName string) {
	// Query all validators
	validatorsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "staking", "validators", "--output=json",
	)
	require.NoError(s, err)

	var validatorsResponse cliValidatorsResponse
	err = json.Unmarshal([]byte(validatorsRes.Stdout), &validatorsResponse)
	require.NoError(s, err)

	// Filter bonded validators and sort by operator address for deterministic ordering
	var bondedValidators []cliValidatorResponse
	for _, validator := range validatorsResponse.Validators {
		if validator.Status == "BOND_STATUS_BONDED" {
			bondedValidators = append(bondedValidators, validator)
		}
	}

	if len(bondedValidators) < validatorIndex {
		s.Logf("Skipping validator selection: not enough bonded validators (found %d, need at least %d)", len(bondedValidators), validatorIndex)
		// Store a placeholder to indicate this validator was skipped
		skipKey := fmt.Sprintf("validator_%d_skipped", validatorIndex)
		s.scenarioState[skipKey] = true
		return
	}

	// Use the nth validator (1-indexed)
	selectedValidator := bondedValidators[validatorIndex-1]
	validatorOperatorAddr := selectedValidator.OperatorAddress

	// Store the validator operator address
	accNameToAddrMap[validatorName] = validatorOperatorAddr
	accAddrToNameMap[validatorOperatorAddr] = validatorName

	s.Logf("Stored validator %s (index %d) with operator address: %s", validatorName, validatorIndex, validatorOperatorAddr)
}

// TheRewardsShouldBeDistributedProportionallyAcrossValidatorsForDelegator validates proportional reward distribution
func (s *suite) TheRewardsShouldBeDistributedProportionallyAcrossValidatorsForDelegator(validatorSpec, delegatorName string) {
	delegatorAddr, exists := accNameToAddrMap[delegatorName]
	require.True(s, exists, "delegator %s not found", delegatorName)

	// Parse validator specification: "validator1, validator2, validator3"
	validatorNames := strings.Split(validatorSpec, ",")

	// Filter out validators that were skipped
	var availableValidators []string
	for _, validatorName := range validatorNames {
		validatorName = strings.TrimSpace(validatorName)
		if _, exists := accNameToAddrMap[validatorName]; exists {
			availableValidators = append(availableValidators, validatorName)
		} else {
			s.Logf("Skipping proportional validation for validator %s (not available)", validatorName)
		}
	}

	if len(availableValidators) == 0 {
		s.Logf("No validators available for proportional validation - test environment may have insufficient validators")
		return
	}

	if len(availableValidators) == 1 {
		s.Logf("Only one validator available (%s) - proportional validation not meaningful", availableValidators[0])
		return
	}

	// Query current rewards for each available validator and compare proportions
	var validatorRewards []struct {
		name   string
		addr   string
		stake  int64
		reward int64
	}

	totalStake := int64(0)
	totalRewards := int64(0)

	for _, validatorName := range availableValidators {
		validatorAddr := accNameToAddrMap[validatorName] // We know it exists from the filter above

		// Get delegation amount (stake)
		delegationAmount := s.queryExistingDelegation(delegatorAddr, validatorAddr)

		// Get current rewards
		args := []string{
			"query", "distribution", "rewards-by-validator",
			delegatorAddr,
			validatorAddr,
			"--output=json",
		}

		rewardsRes, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
		var rewardAmount int64 = 0

		if err == nil {
			var rewardsResponse cliDelegationRewardsResponse
			if err := json.Unmarshal([]byte(rewardsRes.Stdout), &rewardsResponse); err == nil {
				switch rewards := rewardsResponse.Rewards.(type) {
				case []interface{}:
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
			}
		}

		validatorRewards = append(validatorRewards, struct {
			name   string
			addr   string
			stake  int64
			reward int64
		}{validatorName, validatorAddr, delegationAmount, rewardAmount})

		totalStake += delegationAmount
		totalRewards += rewardAmount

		s.Logf("Validator %s: stake=%d, rewards=%d", validatorName, delegationAmount, rewardAmount)
	}

	// Validate proportional distribution (allow for small rounding differences)
	for _, vr := range validatorRewards {
		if totalStake > 0 && totalRewards > 0 {
			expectedProportion := float64(vr.stake) / float64(totalStake)
			actualProportion := float64(vr.reward) / float64(totalRewards)

			// Allow 5% tolerance for rounding differences in tokenomics calculations
			tolerance := 0.05
			difference := expectedProportion - actualProportion
			if difference < 0 {
				difference = -difference
			}

			require.LessOrEqual(s, difference, tolerance,
				"validator %s reward proportion (%f) should be close to stake proportion (%f), difference: %f",
				vr.name, actualProportion, expectedProportion, difference)

			s.Logf("✓ Validator %s proportional validation passed: stake proportion=%f, reward proportion=%f",
				vr.name, expectedProportion, actualProportion)
		}
	}
}
