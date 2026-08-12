Feature: Params Namespace

    # Governance param updates are validated onchain. This suite pins the validations whose
    # entire purpose is to be UNREACHABLE through governance, i.e. the ones whose failure
    # mode is a chain halt rather than a failed tx.
    #
    # DEV_NOTE: A rejected `MsgUpdateParams` is rejected at DELIVERY, not at broadcast, so
    # the CLI still exits 0. `sendAuthzExecTxExpectingError` asserts on the committed tx
    # result; without it a rejection is silent & only surfaces later as a confusing
    # "params were not updated" diff.

    Scenario: Shared params which would halt settlement are rejected
        Given the user has the pocketd binary installed

        # All four claim/proof window offsets at zero make GetNumPendingSessions() return 0.
        # Settlement divides the application stake by that value to derive the per-session
        # budget, and math.Int.Quo panics on a zero divisor. That runs in the EndBlocker, so
        # the panic would HALT THE CHAIN rather than fail a tx. Each offset is individually
        # valid at zero, so nothing but this cross-field check rejects the combination.
        #
        # DEV_NOTE: grace_period_end_offset_blocks MUST be zeroed alongside them, otherwise
        # validateClaimWindowOpenOffsetIsAtLeastGracePeriodEndOffset rejects the param set
        # first & this scenario would assert on the wrong guard.
        Then the "shared" module parameter update is rejected with the error "pending sessions"
            | name                             | value | type  |
            | grace_period_end_offset_blocks   | 0     | int64 |
            | claim_window_open_offset_blocks  | 0     | int64 |
            | claim_window_close_offset_blocks | 0     | int64 |
            | proof_window_open_offset_blocks  | 0     | int64 |
            | proof_window_close_offset_blocks | 0     | int64 |

    # overservicing_bonus_multiplier is new in v0.1.35 & ships as a no-op (1). Its
    # REDISTRIBUTION behaviour needs two suppliers that both submit claims for the same
    # application session (the budget divisor counts claimants, see
    # settlementContext.GetActualSupplierCount), which LocalNet cannot produce with
    # relayminers.count = 1 — that stays covered at the integration level. What is coverable
    # here is the governance path: until this suite could set the param, a new param could
    # ship without the E2E param plumbing ever touching it.
    Scenario: The settlement budget redistribution multiplier can be set by governance
        Given the user has the pocketd binary installed
        When the "tokenomics" module parameters are set as follows
            | name                           | value | type   |
            | overservicing_bonus_multiplier | 2     | uint64 |
        Then all "tokenomics" module params should be updated

    # The cap is a typo-catcher, not a safety bound: a multiplier at or above the session's
    # supplier count is already equivalent to "cap removed" because the application's
    # committed budget binds first.
    Scenario: A settlement budget redistribution multiplier above the cap is rejected
        Given the user has the pocketd binary installed
        Then the "tokenomics" module parameter update is rejected with the error "overservicing_bonus_multiplier"
            | name                           | value | type   |
            | overservicing_bonus_multiplier | 1001  | uint64 |

    # Every at-height consumer resolves shared params through the history store: x/proof,
    # settlement pricing, the settlement budget divisor & the RelayMiner. The query that
    # exposes it is the only way to check what a past height actually resolved to, and its
    # autocli registration is not covered by any unit or integration test.
    Scenario: Shared params are queryable at a past height
        Given the user has the pocketd binary installed
        When the user runs the query "query shared params-at-height 1"
        Then the user should be able to see standard output containing "num_blocks_per_session"
