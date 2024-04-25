Feature: Params Namespace
  #  TODO_DOCUMENT(@Olshansk): Document all of the on-chain governance parameters.

  # TODO_TEST_IN_THIS_PR:
  # Scenario: An unauthorized user cannot update an module params

  Scenario: An authorized user updates all "tokenomics" module params
    Given the user has the pocketd binary installed
    And all "tokenomics" module params are set to their default values
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.tokenomics.MsgUpdateParams" message
    When the user sends an authz exec message to update all "tokenomics" module params
      | name                                 | value | type    |
      | "compute_units_to_tokens_multiplier" | "420" | "int64" |
    # Then all "tokenomics" module params should be updated
    #   | name                                 | value | type    |
    #   | "compute_units_to_tokens_multiplier" | "420" | "int64" |

  # Scenario: An authorized user updates all "proof" module params
  #   Given the user has the pocketd binary installed
  #   And all "proof" module params are set to their default values
  #   And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.proof.MsgUpdateParams" message
  #   When the user sends an authz exec message to update all "tokenomics" module params
  #     | name                        | value | type    |
  #     | "min_relay_difficulty_bits" | "8"   | "int64" |
  #   Then all "proof" module params should be updated
  #     | name                        | value | type    |
  #     | "min_relay_difficulty_bits" | "8"   | "int64" |

  # Scenario Outline: An authorized user updates individual <module> module params
  #   Given the user has the pocketd binary installed
  #   And all <module> module params are set to their default values
  #   And an authz grant from the <granter> "module" account to the <grantee> "user" account for the <message_type> message
  #   When the user sends an authz exec message to update <module> module param <param_name>
  #     | value         | type         |
  #     | <param_value> | <param_type> |
  #   Then the <module> module param <param_name> should be updated
  #     | value         | type         |
  #     | <param_value> | <param_type> |

  #   Examples:
  #     | module       | granter | grantee | message_type                          | param_name                           | param_value | param_type |
  #     | "tokenomics" | "gov"   | "pnf"   | "/poktroll.tokenomics.MsgUpdateParam" | "compute_units_to_tokens_multiplier" | "69"        | "int64"    |
  #     | "proof"      | "gov"   | "pnf"   | "/poktroll.proof.MsgUpdateParam"      | "min_relay_difficulty_bits"          | "12"        | "int64"    |
