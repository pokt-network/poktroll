//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"regexp"
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


// TheUserRemembersTheBalanceOfValidatorAs stores the current balance of a validator in the scenario state
func (s *suite) TheUserRemembersTheBalanceOfValidatorAs(validatorName, stateKey string) {
	validatorAddr, exists := accNameToAddrMap[validatorName]
	require.True(s, exists, "validator %s not found", validatorName)
	
	// Convert validator operator address (poktvaloper...) to regular account address (pokt...)
	valAddr, err := cosmostypes.ValAddressFromBech32(validatorAddr)
	require.NoError(s, err, "invalid validator operator address %s", validatorAddr)
	
	// Validator addresses and account addresses have the same underlying bytes
	accAddr := cosmostypes.AccAddress(valAddr.Bytes())
	accAddrStr := accAddr.String()
	
	// Query balance directly using the account address
	args := []string{
		"query",
		"bank",
		"balances",
		accAddrStr,
	}
	
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error querying validator balance")
	
	// Parse the balance using regex (same as getAccBalance)
	amountRe := regexp.MustCompile(`amount:\s+"(.+?)"\s+denom:\s+upokt`)
	match := amountRe.FindStringSubmatch(res.Stdout)
	
	balance := int64(0)
	if len(match) >= 2 {
		accBalance, err := strconv.Atoi(match[1])
		require.NoError(s, err)
		balance = int64(accBalance)
	}
	
	s.scenarioState[stateKey] = balance
	s.Logf("Stored validator %s balance: %d uPOKT (account address: %s)", validatorName, balance, accAddrStr)
}

// TheAccountBalanceOfShouldBeThan validates account balance changes in any direction
func (s *suite) TheAccountBalanceOfShouldBeThan(accName, direction, prevBalanceKey string) {
	prevBalance, ok := s.scenarioState[prevBalanceKey].(int64)
	require.True(s, ok, "previous balance %s not found or not an int64", prevBalanceKey)

	// Special handling for validator accounts - need to convert operator address to account address
	var currBalance int64
	if accName == "validator1" || strings.HasPrefix(accNameToAddrMap[accName], "poktvaloper") {
		validatorAddr := accNameToAddrMap[accName]
		
		// Convert validator operator address to account address
		valAddr, err := cosmostypes.ValAddressFromBech32(validatorAddr)
		require.NoError(s, err)
		accAddr := cosmostypes.AccAddress(valAddr.Bytes())
		
		// Query balance directly
		args := []string{
			"query",
			"bank",
			"balances",
			accAddr.String(),
		}
		res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
		require.NoError(s, err, "error getting balance")
		
		// Parse the balance using regex
		amountRe := regexp.MustCompile(`amount:\s+"(.+?)"\s+denom:\s+upokt`)
		match := amountRe.FindStringSubmatch(res.Stdout)
		
		if len(match) >= 2 {
			accBalance, err := strconv.Atoi(match[1])
			require.NoError(s, err)
			currBalance = int64(accBalance)
		}
	} else {
		currBalance = s.getAccBalance(accName)
	}

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
