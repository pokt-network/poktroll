package upgrades_test

import (
	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
)

// init configures the SDK's bech32 prefixes for every test in this package.
//
// Without it, `sdk.GetConfig()` keeps the SDK default ("cosmos") and any test that
// validates a pokt-prefixed address fails with:
//
//	invalid Bech32 prefix; expected cosmos, got pokt
//
// which is what `Params.ValidateBasic` hits via `ValidateDaoRewardAddress`.
//
// This lives in its own tracked file on purpose. The prefixes used to be set by an
// `init()` in a v0.1.31 test file that is excluded from version control on some
// machines, so the upgrade tests passed locally and failed in CI, where that file
// does not exist. Package-level setup that every test depends on must not hang off
// a file that may or may not be checked out.
func init() {
	cmd.InitSDKConfig()
}
