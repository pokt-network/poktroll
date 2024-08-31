Feature: Params Namespace

  # Why do we have this scenario?
  #   During development, if one of these tests fails along the way, we get into a
  #   state where LocalNet is inconsistent w/ expectations and needs to be restarted.
  #   Rather than using a `Background` set of commands that rerun on every scenario,
  #   we add one to prepare for everything downstream.
  Scenario: All params are reset to their default values
    Given the user has the pocketd binary installed
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for each module MsgUpdateParam message exists
    Then all module params are reset to their default values

  Scenario: An unauthorized user cannot update a module params
    Given the user has the pocketd binary installed
    And all "tokenomics" module params are set to their default values
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.tokenomics.MsgUpdateParams" message exists
    When the "unauthorized" account sends an authz exec message to update all "tokenomics" module params
      | name                               | value | type  |
      | compute_units_to_tokens_multiplier | 666   | int64 |
    Then all "tokenomics" module params should be set to their default values

  # NB: If you are reading this and the tokenomics module has parameters
  # that are not being updated in this test, please update the test.
  Scenario: An authorized user updates all "tokenomics" module params
    Given the user has the pocketd binary installed
    And all "tokenomics" module params are set to their default values
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.tokenomics.MsgUpdateParams" message exists
    When the "pnf" account sends an authz exec message to update all "tokenomics" module params
      | name                               | value | type  |
      | compute_units_to_tokens_multiplier | 420   | int64 |
    Then all "tokenomics" module params should be updated

  # NB: If you are reading this and the proof module has parameters
  # that are not being updated in this test, please update the test.
  Scenario: An authorized user updates all "proof" module params
    Given the user has the pocketd binary installed
    And all "proof" module params are set to their default values
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.proof.MsgUpdateParams" message exists
    When the "pnf" account sends an authz exec message to update all "proof" module params
      | name                         | value                                                            | type  |
      | relay_difficulty_target_hash | 00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff | bytes |
      | proof_request_probability    | 0.1                                                              | float |
      | proof_requirement_threshold  | 100                                                              | int64 |
      | proof_missing_penalty        | 500                                                              | coin  |
      | proof_submission_fee         | 5000000                                                          | coin  |
    Then all "proof" module params should be updated

  # NB: If you are reading this and the proof module has parameters
  # that are not being updated in this test, please update the test.
  Scenario: An authorized user updates all "shared" module params
    Given the user has the pocketd binary installed
    And all "shared" module params are set to their default values
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.shared.MsgUpdateParams" message exists
    When the "pnf" account sends an authz exec message to update all "shared" module params
      | name                                  | value | type  |
      | num_blocks_per_session                | 5     | int64 |
      | grace_period_end_offset_blocks        | 2     | int64 |
      | claim_window_open_offset_blocks       | 2     | int64 |
      | claim_window_close_offset_blocks      | 3     | int64 |
      | proof_window_open_offset_blocks       | 1     | int64 |
      | proof_window_close_offset_blocks      | 5     | int64 |
      | supplier_unbonding_period_sessions    | 5     | int64 |
      | application_unbonding_period_sessions | 5     | int64 |
    Then all "shared" module params should be updated


  # NB: If you are reading this and any module has parameters that
  # are not being updated in this test, please update the test.
  Scenario: An authorized user updates all "service" module params
    Given the user has the pocketd binary installed
    And all "service" module params are set to their default values
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.service.MsgUpdateParams" message exists
    When the "pnf" account sends an authz exec message to update all "service" module params
      | name            | value        | type  |
      | add_service_fee | 1000000001   | coin  |
    Then all "service" module params should be updated

  # NB: If you are reading this and any module has parameters that
  # are not being updated in this test, please update the test.
  Scenario Outline: An authorized user updates individual <module> module params
    Given the user has the pocketd binary installed
    And all "<module>" module params are set to their default values
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "<message_type>" message exists
    When the "pnf" account sends an authz exec message to update "<module>" the module param
      | name         | value         | type         |
      | <param_name> | <param_value> | <param_type> |
    Then the "<module>" module param "<param_name>" should be updated

    Examples:
      | module     | message_type                        | param_name                            | param_value | param_type |
      | tokenomics | /poktroll.tokenomics.MsgUpdateParam | compute_units_to_tokens_multiplier    | 68          | int64      |
      | proof      | /poktroll.proof.MsgUpdateParam      | min_relay_difficulty_bits             | 12          | int64      |
      | proof      | /poktroll.proof.MsgUpdateParam      | proof_request_probability             | 0.1         | float      |
      | proof      | /poktroll.proof.MsgUpdateParam      | proof_requirement_threshold           | 100         | int64      |
      | proof      | /poktroll.proof.MsgUpdateParam      | proof_missing_penalty                 | 500         | coin       |
      | proof      | /poktroll.proof.MsgUpdateParam      | proof_submission_fee                  | 5000000     | coin       |
      | shared     | /poktroll.shared.MsgUpdateParam     | num_blocks_per_session                | 5           | int64      |
      | shared     | /poktroll.shared.MsgUpdateParam     | grace_period_end_offset_blocks        | 2           | int64      |
      | shared     | /poktroll.shared.MsgUpdateParam     | claim_window_open_offset_blocks       | 2           | int64      |
      | shared     | /poktroll.shared.MsgUpdateParam     | claim_window_close_offset_blocks      | 3           | int64      |
      | shared     | /poktroll.shared.MsgUpdateParam     | proof_window_open_offset_blocks       | 1           | int64      |
      | shared     | /poktroll.shared.MsgUpdateParam     | proof_window_close_offset_blocks      | 5           | int64      |
      | shared     | /poktroll.shared.MsgUpdateParam     | supplier_unbonding_period_sessions    | 5           | int64      |
      | shared     | /poktroll.shared.MsgUpdateParam     | application_unbonding_period_sessions | 5           | int64      |
      | service    | /poktroll.service.MsgUpdateParam    | add_service_fee                       | 1000000001  | coin       |

  Scenario: An unauthorized user cannot update individual module params
    Given the user has the pocketd binary installed
    And all "proof" module params are set to their default values
    And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.proof.MsgUpdateParams" message exists
    When the "unauthorized" account sends an authz exec message to update "proof" the module param
      | name                      | value | type  |
      | proof_request_probability | 0.1   | float |
    Then the "proof" module param "min_relay_difficulty_bits" should be set to its default value
