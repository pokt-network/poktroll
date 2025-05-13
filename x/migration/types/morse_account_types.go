package types

const (
	// MorseExternallyOwnedAccountType is the type string of an externally owned account in the Morse state export.
	MorseExternallyOwnedAccountType = "posmint/Account"
	MorseMultiSigAccountType        = "posmint/MultiSigAccount"
	// MorseModuleAccountType is the type string of a module account in the Morse state export.
	MorseModuleAccountType = "posmint/ModuleAccount"
)

// Morse module account names.
const (
	MorseModuleAccountNameDao                        = "dao"
	MorseModuleAccountNameFeeCollector               = "fee_collector"
	MorseModuleAccountNameApplicationStakeTokensPool = "application_stake_tokens_pool"
	MorseModuleAccountNameStakedTokensPool           = "staked_tokens_pool"
)

// morseModuleAccountNames is the list of module account names that are present
// in the canonical Morse state export.
var (
	// MorseModuleAccountNames is the list of all module account names that are
	// expected to be present in the canonical Morse state export.
	MorseModuleAccountNames = []string{
		MorseModuleAccountNameDao,
		MorseModuleAccountNameFeeCollector,
		MorseModuleAccountNameApplicationStakeTokensPool,
		MorseModuleAccountNameStakedTokensPool,
	}

	// MorseStakePoolModuleAccountNames is the list of module account names which
	// SHOULD be EXCLUDED from the MorseAccountState because they are accounted for elsewhere.
	MorseStakePoolModuleAccountNames = []string{
		MorseModuleAccountNameApplicationStakeTokensPool,
		MorseModuleAccountNameStakedTokensPool,
	}
)
