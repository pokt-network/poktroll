@manual
Feature: Morse Migration Success

  Scenario: Authority generates and imports MorseAccountState
    # TODO_UPNEXT(@bryanchriswhite, #1034): Print a link to the latest liquify snapshot if no local state exists.
    Given a local Morse node persisted state exists
    When the authority executes "pocket util export-genesis-for-reset" with "stdout" written to "morse_state_export.json"
    Then a MorseStateExport is written to "morse_state_export.json"

    When the authority executes "pocketd migrate collect-morse-accounts morse_state_export.json morse_account_state.json"
    Then a MorseAccountState is written to "morse_account_state.json"

    Given no MorseClaimableAccounts exist
    And the MorseAccountState in "morse_account_state.json" is valid
    When the authority executes "make import-morse-claimable-accounts morse_account_state.json"
    Then the MorseClaimableAccounts are persisted onchain

  Rule: Non-actor account claims MAY reference existing Shannon accounts
    Background:
      # TODO_INCOMPLETE: Ensure the liquify Morse snapshot includes known Morse
      # private keys such that valid claim signatures can be generated for testing.
      Given an unclaimed MorseClaimableAccount with a known private key exists
      And a Shannon destination key exists in the local keyring

    Scenario: Morse account-holder claims as an new non-actor account
      And the Shannon destination account does not exist onchain
      When the Morse private key is used to claim a MorseClaimableAccount as a non-actor account
      Then the Shannon destination account balance is increased by the sum of all MorseClaimableAccount tokens

    Scenario: Morse account-holder claims as an existing non-actor account
      And the Shannon destination account exists onchain
      And the Shannon destination account upokt balance is non-zero
      When the Morse private key is used to claim a MorseClaimableAccount as a non-actor account
      Then the Shannon destination account balance is increased by the sum of all MorseClaimableAccount tokens

  Rule: Actor re-stake claims use the Morse stake amount by default
    Background:
      # TODO_INCOMPLETE: Ensure the liquify Morse snapshot includes known Morse
      # private keys such that valid claim signatures can be generated for testing.
      Given an unclaimed MorseClaimableAccount with a known private key exists
      And a Shannon destination key exists in the local keyring

    Scenario Outline:
      And the Shannon destination account is staked as an "<actor>"
      When the Morse private key is used to claim a MorseClaimableAccount as an "<actor>" without specifying the stake amount
      Then the Shannon destination account balance is increased by the sum of "<balance_summand_1>" and "<balance_summand_2>" of the MorseClaimableAccount
      Then the Shannon destination account is staked as an "<actor>"
      And the Shannon "<actor>" stake increased by the "<stake_amount_field>" of the MorseClaimableAccount
      And the Shannon "<actor>" service config is updated, if applicable

      Examples:
        | actor       | balance_summand_1 | balance_summand_2 | stake_amount_field |
        | application | unstaked_balance  | supplier_stake    | application_stake  |
        | supplier    | unstaked_balance  | service_rev_share | supplier_stake     |
      # TODO_TEST: No default gateway stake amount - should fail.
      # | gateway     | unstaked_balance  | NA                | NA

  Rule: Actor re-stake claims MAY use custom stake amounts
    Background:
      # TODO_INCOMPLETE: Ensure the liquify Morse snapshot includes known Morse
      # private keys such that valid claim signatures can be generated for testing.
      Given an unclaimed MorseClaimableAccount with a known private key exists
      And a Shannon destination key exists in the local keyring

    Scenario Outline:
      And the Shannon destination account is staked as an "<actor>"
      When the Morse private key is used to claim a MorseClaimableAccount as an "<actor>" with a stake equal to "<total_tokens_stake_pct>"% of the total tokens of the MorseClaimableAccount
      Then the Shannon destination account balance is increased by the remaining tokens of the MorseClaimableAccount
      Then the Shannon destination account is staked as an "<actor>"
      And the Shannon "<actor>" stake equals the "<stake_amount_field>" of the MorseClaimableAccount

      Examples:
        | actor       | total_tokens_stake_pct |
        | application | 0.75                   |
        | supplier    | 0.75                   |
        | gateway     | 0.75                   |

  Rule: Actor re-stake claims MAY reference existing Shannon actors
    Background:
      # TODO_INCOMPLETE: Ensure the liquify Morse snapshot includes known Morse
      # private keys such that valid claim signatures can be generated for testing.
      Given an unclaimed MorseClaimableAccount with a known private key exists
      And a Shannon destination key exists in the local keyring

    Scenario Outline:
      And the Shannon destination account is staked as an "<actor>"
      When the Morse private key is used to claim a MorseClaimableAccount as an "<actor>" without specifying the stake amount
      Then the Shannon destination account balance is increased by the sum of "<balance_summand_1>" and "<balance_summand_2>" of the MorseClaimableAccount
      Then the Shannon destination account is staked as an "<actor>"
      And the Shannon "<actor>" stake increased by the "<stake_amount_field>" of the MorseClaimableAccount
      And the Shannon "<actor>" service config is updated, if applicable

      Examples:
        | actor       | balance_summand_1 | balance_summand_2 | stake_amount_field |
        | application | unstaked_balance  | supplier_stake    | application_stake  |
        | supplier    | unstaked_balance  | service_rev_share | supplier_stake     |
      # TODO_TEST: Existing gateway scenario; doesn't fit in this scenario outline.
      # | gateway     | unstaked_balance  | NA                | NA
