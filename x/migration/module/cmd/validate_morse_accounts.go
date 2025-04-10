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
		Use:     "validate-morse-accounts [msg_import_morse_claimable_accounts_json_path] [morse_src_address_to_check, ...]",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: logger.PreRunESetup,
		RunE:    runValidateMorseAccounts,
		PostRun: signals.ExitWithCodeIfNonZero,
		Short:   "Validate and inspect the morse account state contained within a given MsgImportMorseClaimableAccount JSON file",
		Long: `Validate and inspect the morse account state contained within a given MsgImportMorseClaimableAccount JSON file.

This output is intended to be used for manual inspection of each Morse account to ensure that it matches the expected state (as of the export height).

Validation consists of:
	1. Calculating the sha256 hash of the (serialized) 'morse_account_state' field
	2. Comparing it to the hash given in by the 'morse_account_state_hash' field.

If a discrepancy in the Morse account state hash is detected:
	1. A warning is printed
	2. Found Morse source accounts are printed
	3. A non-zero exit code is returned.

If any given 'morse_src_address' is not found in the Morse account state:
	1. Found Morse source accounts are printed
	2. A warning is printed for each missing 'morse_src_address'
	3. A non-zero exit code is returned.

The JSON serialization of each found 'morse_src_address' is printed following the line:

	> Found MorseClaimableAccount <morse_src_address>`,
		Example: `## Example 1: Uppercase hex and multiple MorseSrcAddresses
	$ pocketd tx migration validate-morse-accounts ./msg_import_morse_claimable_accounts.json 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288 9B4508816AC8627B364D2EA4FC1B1FEE498D5684

🎉 Morse account state hash matches: BE89A43098CA8C37612491DA674FC26F1F4314AA82EB466A777F1E2BA6C2FBA8 🎉
        Found MorseClaimableAccount 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288 {
          "shannon_dest_address": "",
          "morse_src_address": "1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288",
          "public_key": "M2vbX4RIBJYU8Bm/R6Fz55SFixNmbImz5Oll0cf2nQs=",
          "unstaked_balance": {
            "denom": "upokt",
            "amount": "1000001"
          },
          "supplier_stake": {
            "denom": "upokt",
            "amount": "0"
          },
          "application_stake": {
            "denom": "upokt",
            "amount": "0"
          },
          "claimed_at_height": 0
        }
        Found MorseClaimableAccount 9B4508816AC8627B364D2EA4FC1B1FEE498D5684 {
          "shannon_dest_address": "",
          "morse_src_address": "9B4508816AC8627B364D2EA4FC1B1FEE498D5684",
          "public_key": "i05T5/bfYVt/KN65wyzOCnQ1exRG4F1IDsAAbTEs7fg=",
          "unstaked_balance": {
            "denom": "upokt",
            "amount": "2000002"
          },
          "supplier_stake": {
            "denom": "upokt",
            "amount": "0"
          },
          "application_stake": {
            "denom": "upokt",
            "amount": "200020"
          },
          "claimed_at_height": 0
        }

## Example 2: Lowercase hex MorseSrcAddresses
	$ pocketd tx migration validate-morse-accounts ./msg_import_morse_claimable_accounts.json 1a0bb8623f40d2a9beac099a0bafdcae3c5d8288

🎉 Morse account state hash matches: BE89A43098CA8C37612491DA674FC26F1F4314AA82EB466A777F1E2BA6C2FBA8 🎉
        Found MorseClaimableAccount 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288 {
          "shannon_dest_address": "",
          "morse_src_address": "1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288",
          "public_key": "M2vbX4RIBJYU8Bm/R6Fz55SFixNmbImz5Oll0cf2nQs=",
          "unstaked_balance": {
            "denom": "upokt",
            "amount": "1000001"
          },
          "supplier_stake": {
            "denom": "upokt",
            "amount": "0"
          },
          "application_stake": {
            "denom": "upokt",
            "amount": "0"
          },
          "claimed_at_height": 0
        }

## Example 3: Missing MorseSrcAddresses
	$ pocketd tx migration validate-morse-accounts ./msg_import_morse_claimable_accounts.json 6629E4DEAE5AAC5EFA5C6CBCFDA5A289C825EC73 C81FBB2361A72CFAF8C1FAF3A4C439EF1EA5F8E3 0256531919A13334088737667636CBB603982E46

🎉 Morse account state hash matches: BE89A43098CA8C37612491DA674FC26F1F4314AA82EB466A777F1E2BA6C2FBA8 🎉
        WRN 🚨 3 Morse address(es) not found: 🚨
        WRN   - 6629E4DEAE5AAC5EFA5C6CBCFDA5A289C825EC73
        WRN   - C81FBB2361A72CFAF8C1FAF3A4C439EF1EA5F8E3
        WRN   - 0256531919A13334088737667636CBB603982E46

## Example 4: Invalid MorseAccountStateHash
	$ pocketd tx migration validate-morse-accounts ./invalid_msg_import_morse_claimable_accounts.json 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288

WRN 🚨 Invalid morse account state hash! 🚨
        WRN Given (expected): 696E76616C69645F68617368
        WRN Computed (actual): BE89A43098CA8C37612491DA674FC26F1F4314AA82EB466A777F1E2BA6C2FBA8
        Found MorseClaimableAccount 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288 {
          "shannon_dest_address": "",
          "morse_src_address": "1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288",
          "public_key": "M2vbX4RIBJYU8Bm/R6Fz55SFixNmbImz5Oll0cf2nQs=",
          "unstaked_balance": {
            "denom": "upokt",
            "amount": "1000001"
          },
          "supplier_stake": {
            "denom": "upokt",
            "amount": "0"
          },
          "application_stake": {
            "denom": "upokt",
            "amount": "0"
          },
          "claimed_at_height": 0
        }`,
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

	// Retrieve the morse addresses to check from the command line arguments.
	morseAddresses := args[1:]

	// Read the MsgImportMorseClaimableAccounts JSON file.
	msgImportMorseClaimableAccountsJSONBz, err := os.ReadFile(msgImportMorseClaimableAccountsJSONPath)
	if err != nil {
		return err
	}

	// Unmarshal the MsgImportMorseClaimableAccounts JSON into a MsgImportMorseClaimableAccounts struct.
	msgImportMorseClaimableAccounts := new(migrationtypes.MsgImportMorseClaimableAccounts)
	if err = cmtjson.Unmarshal(msgImportMorseClaimableAccountsJSONBz, msgImportMorseClaimableAccounts); err != nil {
		return err
	}

	// Compute and validate the morse account state hash in the file provided.
	morseAccountState := msgImportMorseClaimableAccounts.GetMorseAccountState()
	givenMorseAccountStateHash := msgImportMorseClaimableAccounts.GetMorseAccountStateHash()
	computedMorseAccountStateHash, err := morseAccountState.GetHash()
	if err != nil {
		return err
	}
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
	numSortedMorseClaimableAccounts := len(sortedMorseClaimableAccounts)
	sortMorseClaimableAccounts(msgImportMorseClaimableAccounts.MorseAccountState.Accounts)

	// Check that each given morse address exists in the morse account state and print each for inspection.
	missingMorseAddresses := make([]string, 0)
	for _, targetMorseAddress := range morseAddresses {
		// Normalize the morse address to uppercase hex.
		targetMorseAddress = strings.ToUpper(targetMorseAddress)

		// Use binary search to find the index of the target morse address efficiently.
		morseAccountIdx := sort.Search(numSortedMorseClaimableAccounts, func(i int) bool {
			ithMorseSrcAddress := sortedMorseClaimableAccounts[i].GetMorseSrcAddress()
			return ithMorseSrcAddress >= targetMorseAddress
		})

		// DEV_NOTE: The index returned when the target is not found is the length of the slice.
		if morseAccountIdx >= numSortedMorseClaimableAccounts {
			missingMorseAddresses = append(missingMorseAddresses, targetMorseAddress)
			logger.Logger.Warn().Msgf("Morse address %s not found in the morse account state. Moving on to the next one.", targetMorseAddress)
			continue
		}

		if sortedMorseClaimableAccounts[morseAccountIdx].GetMorseSrcAddress() != targetMorseAddress {
			missingMorseAddresses = append(missingMorseAddresses, targetMorseAddress)
			logger.Logger.Warn().Msgf("Morse address %s not found in the morse account state. Moving on to the next one.", targetMorseAddress)
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

// newMorseClaimableAccountOrderByMorseSrcAddress returns a function that can be used with
// `sort.Slice` to sort the given morse claimable accounts in ascending or descending order by morseSrcAddress.
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
