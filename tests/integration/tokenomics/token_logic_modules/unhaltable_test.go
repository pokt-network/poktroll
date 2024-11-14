package token_logic_modules

import (
	"testing"
)

// TODO_TEST(@bryanchriswhite): Settlement proceeds in the face of errors

// TestSettlePendingClaims_HaltingError asserts that the chain halts when claim
// settlement results in an unexpected error.
func (s *tokenLogicModuleTestSuite) TestSettlePendingClaims_HaltingError() {
	tests := []struct {
		desc             string
		asyncStateChange func(*testing.T)
		getExpectedErr   func() error
	}{
		{desc: "the application is unbonded prematurely"},
		{desc: "the supplier is unbonded prematurely"},
		{desc: "the service is removed prematurely"},
		{desc: "compute units per relay is updated mid-session"},
		{desc: "supplier service config is invalid (missing RevShares)"},
		{desc: "supplier service config is invalid (invalid RevShare address)"},
		{desc: "application module has insufficient funds"},
		{desc: "tokenomics module has insufficient funds"},
		{desc: "supplier module has insufficient funds"},
	}

	for _, test := range tests {
		s.T().Run(test.desc, func(t *testing.T) {
			// TODO_TEST:
			// set up keepers
			// assert 0 claims exist
			// get the session
			// store the proof path seed
			// create claim(s)
			// create proof(s)
			// simulate asynchronous state modifications
			// attempt claim settlement
			// assert the error is the expected one
		})
	}
}

// TestSettlePendingClaims_NonHaltingError asserts that the chain does NOT halt
// when claim settlement results in certain anticipated and non-halting errors.
func (s *tokenLogicModuleTestSuite) TestSettlePendingClaims_NonHaltingError() {
	tests := []struct {
		desc  string
		setup func(*testing.T)
	}{
		{desc: "supplier operator pubkey not on-chain"},
		{desc: "closest merkle proof is invalid (mangled)"},
		{desc: "closest merkle proof is invalid (non-compact)"},
		{desc: "closest merkle proof leaf is not a relay"},
		{desc: "the application is overserviced"},
	}

	for _, test := range tests {
		s.T().Run(test.desc, func(t *testing.T) {
			// TODO_TEST:
			// set up keepers
			// assert 0 claims exist
			// case-specific setup
			// settle pending claims
			// assert results (states and balances)
		})
	}
}
