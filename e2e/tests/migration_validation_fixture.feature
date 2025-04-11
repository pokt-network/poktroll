# TODO_MAINNET_MIGRATION(@bryanchriswhite): Finish implementing this feature and its step definitions.
Feature: Morse account validation

  Background: Prepare MsgImportMorseClaimableAccounts JSON
    Given the user has the pocketd binary installed
    And a valid MsgImportMorseClaimableAccounts JSON exists with the following morse account state:
      | morse_src_address                        | unstaked_balance | application_stake | supplier_stake |
      | 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288 | 101              | 0                 | 0              |
      | 9B4508816AC8627B364D2EA4FC1B1FEE498D5684 | 102              | 100               | 0              |
      | 44892C8AB52396BA016ADDD0221783E3BD29A400 | 103              | 0                 | 200            |
    # DEV_NOTE: Currently, claiming an account which is staked as BOTH an application AND a supplier is not supported.
    # | 44892C8AB52396BA016ADDD0221783E3BD29A400 | 103              | 200               | 300            |

  Scenario: Validate given Morse accounts are present in MsgImportClaimableAccounts
    When the user runs the `validate-morse-accounts` command with the MsgImportMorseClaimableAccounts JSON and the morse addresses with the corresponding indices:
      | index                                    |
      | 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288 |
      | 9B4508816AC8627B364D2EA4FC1B1FEE498D5684 |
      | 44892C8AB52396BA016ADDD0221783E3BD29A400 |
      | 82510FE41923685BBEE0B4844176AF0AA8EDF198 |
    Then the user should see the following morse account state:
      | index                                    | present | unstaked_balance | application_stake | supplier_stake |
      | 1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288 | true    | 101              | 0                 | 0              |
      | 9B4508816AC8627B364D2EA4FC1B1FEE498D5684 | true    | 102              | 100               | 0              |
      | 44892C8AB52396BA016ADDD0221783E3BD29A400 | true    | 103              | 0                 | 200            |
      | 82510FE41923685BBEE0B4844176AF0AA8EDF198 | false   | 104              | 0                 | 0              |

