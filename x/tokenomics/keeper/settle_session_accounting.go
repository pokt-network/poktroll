package keeper

import sessiontypes "github.com/pokt-network/poktroll/x/session/types"

func (k TokenomicsKeeper) SettleSessionAccounting(sessionHeader *sessiontypes.SessionHeader, smstRoot []byte) {
	// TODO_UPNEXT(@Olshansk): POC implementation
	// 1. Retrieve the sum from smstRoot and name it `computeUnits`
	// 2. Retrieve the `ComputeUnitsToTokensMultiplier` parameter
	// 3. Compute `uPOKT` of work done by multiplying (1) by (2)
	// 4. Mint (3) `uPOKT` in the supplier module account
	// 5. Send (3) `uPOKT` from the supplier module account to the `Supplier` who did the work
	// 6. Send (3) `uPOKT` from the `Application` who paid for the work to the application module account
	// 7. Burn (3) `uPOKT` from the application module account
}
