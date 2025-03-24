# NOTE: The @oneshot tag allows this feature to be
# excluded from any wildcard feature file execution (e.g. *.feature).
#
# The @oneshot tag indicates that a given feature is non-idempotent with respect to its impact on the network state.
# In such cases, a complete network reset is required before running these features again.
@oneshot
Feature: Morse account import and claim all account types (with fixture data)

  Background:
    Given the user has the pocketd binary installed
    And a MorseAccountState with "20" accounts in a "round-robin" distribution has successfully been imported
    And an unclaimed MorseClaimableAccount with a known private key exists
    And a Shannon destination key exists in the local keyring

  Rule: Non-actor account claims MAY reference existing Shannon accounts
    Scenario: Morse account-holder claims as a new non-actor account
      Given the Shannon destination account does not exist onchain
      # TODO_MAINNET: Use a new token denomination.
      And the Shannon account is funded with "1upokt"
      When the Morse private key is used to claim a MorseClaimableAccount as a non-actor account
      Then the Shannon destination account balance is increased by the unstaked balance amount of the MorseClaimableAccount
      And the Morse claimable account is marked as claimed by the shannon account at a recent block height

    Scenario: Morse account-holder claims as an existing non-actor account
      And the Shannon account is funded with "100upokt"
      And the Shannon destination account upokt balance is non-zero
      When the Morse private key is used to claim a MorseClaimableAccount as a non-actor account
      Then the Shannon destination account balance is increased by the unstaked balance amount of the MorseClaimableAccount
      And the Morse claimable account is marked as claimed by the shannon account at a recent block height

  Rule: Actor (re-)stake claims MAY reference existing Shannon actors
    Scenario Outline: Morse account-holder claims as a new staked actor
      Given the Shannon destination account is not staked as an "<actor>"
      # TODO_MAINNET: Use a new token denomination.
      And the Shannon account is funded with "1upokt"
      When the Morse private key is used to claim a MorseClaimableAccount as an "<actor>" for "anvil" service
      Then the Shannon destination account balance is increased by the unstaked balance amount of the MorseClaimableAccount
      And the Shannon destination account is staked as an "<actor>"
      And the Shannon "<actor>" stake increased by the corresponding actor stake amount of the MorseClaimableAccount
      And the Shannon "<actor>" service config matches the one provided when claiming the MorseClaimableAccount

      Examples:
        | actor       |
        | application |
      # TODO_MAINNET(@bryanchriswhite, #1034: Uncomment the following example once supplier Morse account claiming is available.
      # | supplier    |

    Scenario Outline: Morse account-holder claims as an existing staked actor
      Given the Shannon account is funded with "1234567upokt"
      And the Shannon destination account is staked as an "<actor>" with "1234567" uPOKT for "anvil" service
      When the Morse private key is used to claim a MorseClaimableAccount as an "<actor>" for "ollama" service
      Then the Shannon destination account balance is increased by the unstaked balance amount of the MorseClaimableAccount
      And the Shannon destination account is staked as an "<actor>"
      And the Shannon "<actor>" stake increased by the corresponding actor stake amount of the MorseClaimableAccount
      And the Shannon "<actor>" service config matches the one provided when claiming the MorseClaimableAccount

      Examples:
        | actor       |
        | application |
      # TODO_MAINNET(@bryanchriswhite, #1034: Uncomment the following example once supplier Morse account claiming is available.
      # | supplier    |

# TODO_MAINNET(@bryanchriswhite, #1034): Enumerate and implement error scenarios.
# TODO_POST_MAINNET(@bryanchriswhite, #1034): Scenario: Morse account-holder claims with a stake below the minimum
