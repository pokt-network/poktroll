Feature: Morse account validation

  Background: Prepare MsgImportMorseClaimableAccounts JSON
    Given the user has the pocketd binary installed
    And a valid MsgImportMorseClaimableAccounts JSON exists with the following morse account state:
      | index | unstaked_balance | application_stake | supplier_stake |
      | 0     | 101              | 0                 | 0              |
      | 1     | 102              | 100               | 0              |
      | 2     | 103              | 0                 | 200            |

  Scenario: Validate given Morse accounts are present in MsgImportClaimableAccounts
    When the user runs the `validate-morse-accounts` command with the MsgImportMorseClaimableAccounts JSON and the morse addresses with the corresponding indices:
      | index |
      | 0     |
      | 1     |
      | 2     |
      | 3     |
    Then the user should see the following morse account state:
      | index | present | unstaked_balance | application_stake | supplier_stake |
      | 0     | true    | 101              | 0                 | 0              |
      | 1     | true    | 102              | 100               | 0              |
      | 2     | true    | 103              | 0                 | 200            |
      | 3     | false   | 104              | 0                 | 0              |

