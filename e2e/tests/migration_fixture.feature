# NOTE: The @oneshot tag allows this feature to be
# excluded from any wildcard feature file execution (e.g. *.feature).
#
# The @oneshot tag indicates that a given feature is non-idempotent with respect
# to its impact on the network state. In such cases, a complete network reset
# is required before running these features again.
@oneshot
Feature: Morse Migration Success

  Background:
    Given a MorseAccountState has successfully been imported with the following claimable accounts type distribution:
      | non-actor | application | supplier |
      | 2         | 2           | 2        |
    And an unclaimed MorseClaimableAccount with a known private key exists
    And a Shannon destination key exists in the local keyring

  Rule: Non-actor account claims MAY reference existing Shannon accounts
    Scenario: Morse account-holder claims as a new non-actor account
      Given the Shannon destination account does not exist onchain
      When the Morse private key is used to claim a MorseClaimableAccount as a non-actor account
      Then the Shannon destination account balance is increased by the sum of all MorseClaimableAccount tokens

#    Scenario: Morse account-holder claims as an existing non-actor account
#      Given the Shannon destination account exists onchain
#      And the Shannon destination account upokt balance is non-zero
#      When the Morse private key is used to claim a MorseClaimableAccount as a non-actor account
#      Then the Shannon destination account balance is increased by the sum of all MorseClaimableAccount tokens
#
#  Rule: Actor (re-)stake claims MAY reference existing Shannon actors
#    Scenario Outline: Morse account-holder claims as a new staked actor
#      Given the Shannon destination account is not staked as an "<actor>"
#      When the Morse private key is used to claim a MorseClaimableAccount as an "<actor>"
#      Then the Shannon destination account balance is increased by the unstaked balance amount of the MorseClaimableAccount
#      And the Shannon destination account is staked as an "<actor>"
#      And the Shannon "<actor>" stake increased by the "<stake_amount_field>" of the MorseClaimableAccount
#      And the Shannon "<actor>" service config is updated, if applicable
#
#      Examples:
#        | actor       | stake_amount_field |
#        | application | application_stake  |
#        | supplier    | supplier_stake     |
#
#    Scenario Outline: Morse account-holder claims as an existing staked actor
#      Given the Shannon destination account is staked as an "<actor>"
#      When the Morse private key is used to claim a MorseClaimableAccount as an "<actor>"
#      Then the Shannon destination account balance is increased by the unstaked balance amount of the MorseClaimableAccount
#      And the Shannon destination account is staked as an "<actor>"
#      And the Shannon "<actor>" stake increased by the "<stake_amount_field>" of the MorseClaimableAccount
#      And the Shannon "<actor>" service config is updated, if applicable
#
#      Examples:
#        | actor       | stake_amount_field |
#        | application | application_stake  |
#        | supplier    | supplier_stake     |

# TODO_UPNEXT(@bryanchriswhite, #1034): Enumerate and implement error scenarios.
