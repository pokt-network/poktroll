# NOTE: The @oneshot & @manual tags allows this feature to be
# excluded from any wildcard feature file execution (e.g. *.feature).
#
# The @manual tag indicates that a given feature depends on some non-automated
# setup which MUST be performed manually, prior to running the feature.
#
# The @oneshot tag indicates that a given feature is non-idempotent with respect
# to its impact on the network state. In such cases, a complete network reset
# is required before running these features again.
@oneshot @manual
Feature: Morse account import and claim all account types (with snapshot data)

  # TODO_MAINNET_CRITICAL(@bryanchriswhite, #1034): The snapshot based Morse account import feature is incomplete.

  Scenario: Authority generates and imports MorseAccountState
    # TODO_MAINNET_CRITICAL(@bryanchriswhite, #1034): Print a link to the latest liquify snapshot if no local state exists.
    Given a local Morse node persisted state exists
    # TODO_POST_MAINNET: Replace current height with the published "canonical" export/migration/cutover height.
    When the authority exports the Morse Account State at height "130000" to "morse_state_export.json"
    Then a MorseStateExport is written to "morse_state_export.json"

    When the authority executes "pocketd tx migration collect-morse-accounts morse_state_export.json morse_account_state.json"
    Then a MorseAccountState is written to "morse_account_state.json"

    Given no MorseClaimableAccounts exist
    And the MorseAccountState in "morse_account_state.json" is valid
    When the authority executes "pocketd tx migration import-morse-accounts morse_account_state.json"
    Then the MorseClaimableAccounts are persisted onchain

  Scenario:
    Given a Morse node snapshot is available
    And the authority successfully imports MorseAccountState generated from the snapshot state
    # TODO_MAINNET_CRITICAL(@bryanchriswhite): Ensure the liquify Morse snapshot includes known Morse
    # private keys such that valid claim signatures can be generated for testing.
    #
    # Use a distinct SECRET random number to seed each private key needed.
    # The implementation could read these out of a single new-line delimited
    # env var (e.g. MORSE_KEY_SEED_0).
    And "6" Morse private keys are available in a "round-robin" actor type distribution

    When a Morse account-holder claims as a new non-actor account
    Then the Shannon destination account balance is increased by the sum of all MorseClaimableAccount tokens
    And the Morse claimable account is marked as claimed by the shannon account at a recent block height

    When a Morse account-holder claims as an existing non-actor account
    Then the Shannon destination account balance is increased by the MorseClaimableAccount unstaked balance
    And the Morse claimable account is marked as claimed by the shannon account at a recent block height

    When a Morse account-holder claims as a new application
    Then the Shannon destination account balance is increased by the MorseClaimableAccount unstaked balance
    And the Morse claimable account is marked as claimed by the shannon account at a recent block height
    And the Shannon destination account is staked as an application with the stake equal to the onchain MorseClaimableAccount

    Given an application is staked
    When a Morse account-holder claims as an existing application
    Then the Shannon destination account balance is increased by the MorseClaimableAccount unstaked balance
    And the Morse claimable account is marked as claimed by the shannon account at a recent block height
    And the Shannon destination account application stake is increased by the MorseClaimableAccount application stake

    When a Morse account-holder claims as a new supplier
    Then the Shannon destination account balance is increased by the MorseClaimableAccount unstaked balance
    And the Morse claimable account is marked as claimed by the shannon account at a recent block height
    And the Shannon destination account is staked as an supplier with the stake equal to the onchain MorseClaimableAccount

    Given a supplier is staked
    When a Morse account-holder claims as an existing supplier
    Then the Shannon destination account balance is increased by the MorseClaimableAccount unstaked balance
    And the Morse claimable account is marked as claimed by the shannon account at a recent block height
    And the Shannon destination account supplier stake is increased by the MorseClaimableAccount supplier stake

  # TODO_MAINNET_CRITICAL(@bryanchriswhite, #1034): Enumerate and implement error scenarios.
