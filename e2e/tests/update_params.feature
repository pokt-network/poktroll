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
