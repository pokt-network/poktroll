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
Feature: MorseAccountState Import

  Scenario: Authority generates and imports MorseAccountState
    # TODO_UPNEXT(@bryanchriswhite, #1034): Print a link to the latest liquify snapshot if no local state exists.
    Given a local Morse node persisted state exists
    # TODO_MAINNET: Replace 1000 with the published "canonical" export/migration/cutover height.
    When the authority executes "pocket util export-genesis-for-reset 1000 poktroll" with stdout written to "morse_state_export.json"
    Then a MorseStateExport is written to "morse_state_export.json"

    When the authority executes "poktrolld tx migration collect-morse-accounts morse_state_export.json morse_account_state.json"
    Then a MorseAccountState is written to "morse_account_state.json"

    Given no MorseClaimableAccounts exist
    And the MorseAccountState in "morse_account_state.json" is valid
    When the authority executes "poktrolld tx migration import-morse-accounts --from=pnf --grpc-addr=localhost:9090 morse_account_state.json"
    Then the MorseClaimableAccounts are persisted onchain

  Scenario:
    Given a Morse node snapshot is available
    And the authority sucessfully imports MorseAccountState generated from the snapshot state
    # TODO_INCOMPLETE: Ensure the liquify Morse snapshot includes known Morse
    # private keys such that valid claim signatures can be generated for testing.
    And Morse private keys are available in the following actor type distribution:
      | non-actor | application | supplier |
      | 2         | 2           | 2        |

    When a Morse account-holder claims as a new non-actor account

    When a Morse account-holder claims as an existing non-actor account

    When a Morse account-holder claims as a new application

    Given an application is staked
    When a Morse account-holder claims as an existing application

    When a Morse account-holder claims as a new supplier

    Given a supplier is staked
    When a Morse account-holder claims as an existing supplier

  # TODO_UPNEXT(@bryanchriswhite, #1034): Enumerate and implement error scenarios.
