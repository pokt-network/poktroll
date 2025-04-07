package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"sort"
	"strings"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/cmd/signals"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const (
	sortAscending sortDirection = iota
	sortDescending
)

// sortDirection is an enum type used to indicate the direction of sorting.
// It is intended to be used in a `Less` function when sorting.
type sortDirection int

func ValidateMorseAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-morse-accounts [msg_import_morse_claimable_accounts_json_path] [morse_src_address_to_check, ...]",
		Args:  cobra.MinimumNArgs(1),
		Short: "Validate and inspect the morse account state contained within a given MsgImportMorseClaimableAccount JSON file",
		Long: `Validate and inspect the morse account state contained within a given MsgImportMorseClaimableAccount JSON file.
Validation consists of calculating the sha256 hash of the the (serialized) morse_account_state field and comparing it to the hash given in by the morse_account_state_hash field.
If a discrepancy in the Morse account state hash is detected, a warning is printed, found Morse source accounts are printed, and a non-zero exit code is returned.
If any given morse_src_address is not found in the Morse account state, found Morse source accounts are printed, a warning is printed for each missing morse_src_address, and a non-zero exit code is returned.

The JSON serialization of each found morse_src_address is printed following the line:
> Found MorseClaimableAccount <morse_src_address>

This output is intended to be used for manual inspection of each Morse account to ensure that it matches the expected state (as of the export height).`,
		Example: `pocketd tx migration validate-morse-accounts 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288 9B4508816AC8627B364D2EA4FC1B1FEE498D5684 
pocketd tx migration validate-morse-accounts 1a0bb8623f40d2a9beac099a0bafdcae3c5d8288 9b4508816ac8627b364d2ea4fc1b1fee498d5684`,
		PreRunE: logger.PreRunESetup,
		RunE:    runValidateMorseAccounts,
		PostRun: signals.ExitWithCodeIfNonZero,
	}
	return cmd
}

// runValidateMorseAccounts performs the following sequence:
// - Load and parse the MsgImportMorseClaimableAccounts JSON.
// - Load and parse the morse account state.
// - Calculate the sha256 hash of the morse account state.
// - Check that the sha256 hash matches the morse account state hash.
// - If the sha256 hash does not match, print the expected and actual hashes and set a non-zero exit code.
// - Sort the morse claimable accounts by morseSrcAddress.
// - Check that each morseSrcAddress is present in the morse account state.
// - Print the found Morse accounts for inspection.
// - If any morseSrcAddress is not found in the morse account state, print the missing morseSrcAddresses and set a non-zero exit code.
func runValidateMorseAccounts(_ *cobra.Command, args []string) error {
	// Load and parse the MsgImportMorseClaimableAccounts JSON.
	msgImportMorseClaimableAccountsJSONPath := args[0]
	morseAddresses := args[1:]

	msgImportMorseClaimableAccountsJSONBz, err := os.ReadFile(msgImportMorseClaimableAccountsJSONPath)
	if err != nil {
		return err
	}

	msgImportMorseClaimableAccounts := new(migrationtypes.MsgImportMorseClaimableAccounts)
	if err = cmtjson.Unmarshal(msgImportMorseClaimableAccountsJSONBz, msgImportMorseClaimableAccounts); err != nil {
		return err
	}

	// Compute and validate the morse account state hash.
	morseAccountState := msgImportMorseClaimableAccounts.GetMorseAccountState()
	computedMorseAccountStateHash, err := morseAccountState.GetHash()
	if err != nil {
		return err
	}

	givenMorseAccountStateHash := msgImportMorseClaimableAccounts.GetMorseAccountStateHash()
	if !bytes.Equal(givenMorseAccountStateHash, computedMorseAccountStateHash) {
		signals.ExitCode += 1
		logger.Logger.Warn().Msg("🚨 Invalid morse account state hash! 🚨")
		logger.Logger.Warn().Msgf("Given (expected): %X", givenMorseAccountStateHash)
		logger.Logger.Warn().Msgf("Computed (actual): %X", computedMorseAccountStateHash)
	} else {
		logger.Logger.Info().Msgf("🎉 Morse account state hash matches: %X 🎉", computedMorseAccountStateHash)
	}

	// Sort the morse claimable accounts for use in binary search.
	sortedMorseClaimableAccounts := msgImportMorseClaimableAccounts.MorseAccountState.Accounts
	sortMorseClaimableAccounts(msgImportMorseClaimableAccounts.MorseAccountState.Accounts)

	// Check that each given morse address exists in the morse account state
	// and print each for inspection.
	missingMorseAddresses := make([]string, 0)
	for _, targetMorseAddress := range morseAddresses {
		// Normalize the morse address to uppercase hex.
		targetMorseAddress = strings.ToUpper(targetMorseAddress)

		// Use binary search to find the index of the target morse address efficiently.
		morseAccountIdx := sort.Search(len(sortedMorseClaimableAccounts), func(i int) bool {
			ithMorseSrcAddress := sortedMorseClaimableAccounts[i].GetMorseSrcAddress()
			return ithMorseSrcAddress >= targetMorseAddress
		})

		// DEV_NOTE: The index returned when the target is not found is the length of the slice.
		if morseAccountIdx >= len(sortedMorseClaimableAccounts) {
			missingMorseAddresses = append(missingMorseAddresses, targetMorseAddress)

			// Morse address not found, move on to the next one.
			continue
		}

		if sortedMorseClaimableAccounts[morseAccountIdx].GetMorseSrcAddress() != targetMorseAddress {
			missingMorseAddresses = append(missingMorseAddresses, targetMorseAddress)

			// Morse address found, move on to the next one.
			continue
		}

		morseClaimableAccount := sortedMorseClaimableAccounts[morseAccountIdx]
		morseClaimableAccountJSONBz, err := json.MarshalIndent(morseClaimableAccount, "", "  ")
		if err != nil {
			return err
		}
		logger.Logger.Info().Msgf(
			"Found MorseClaimableAccount %s %s",
			strings.ToUpper(targetMorseAddress),
			string(morseClaimableAccountJSONBz),
		)
	}

	if len(missingMorseAddresses) != 0 {
		signals.ExitCode += 2
		logger.Logger.Warn().Msgf("🚨 %d Morse address(es) not found: 🚨", len(missingMorseAddresses))
	}
	for _, missingMorseAddress := range missingMorseAddresses {
		logger.Logger.Warn().Msgf("  - %s", strings.ToUpper(missingMorseAddress))
	}

	return nil
}

// sortMorseClaimableAccounts sorts the given morse claimable accounts in ascending order by morseSrcAddress.
// It is intended to be used in conjunction with the `sort.Slice` function for binary searching by morseSrcAddress.
func sortMorseClaimableAccounts(morseClaimableAccounts []*migrationtypes.MorseClaimableAccount) {
	sortByMorseSrcAddress := newMorseClaimableAccountOrderByMorseSrcAddress(morseClaimableAccounts, sortAscending)
	sort.Slice(morseClaimableAccounts, sortByMorseSrcAddress)
}

// newMorseClaimableAccountOrderByMorseSrcAddress returns a function that can be
// used with the `sort.Slice` function to sort the given morse claimable accounts
// in ascending or descending order by morseSrcAddress.
func newMorseClaimableAccountOrderByMorseSrcAddress(
	morseClaimableAccounts []*migrationtypes.MorseClaimableAccount,
	direction sortDirection,
) func(i, j int) bool {
	// DEV_NOTE: this is a "less" function:
	// returns true if elem i is less than elem j.
	return func(i, j int) bool {
		bzDiff := strings.Compare(
			morseClaimableAccounts[i].MorseSrcAddress,
			morseClaimableAccounts[j].MorseSrcAddress,
		)
		switch bzDiff {
		case -1:
			return direction == sortAscending
		case 1:
			return direction == sortDescending
		default:
			// If equal, don't swap; no-op.
			return false
		}
	}
}
