package types

import cosmosclient "github.com/cosmos/cosmos-sdk/client"

// QueryContext is used to distinguish a cosmosclient.QueryContext intended for use in
// queries from others. This is because the same cosmosclient.Context can be
// used for both queries and transactions (as they are generated identically).
// This type is intentionally not an alias in order to make this distinction
// clear to the dependency injector (i.e. querytypes.QueryContext).
type QueryContext cosmosclient.Context
