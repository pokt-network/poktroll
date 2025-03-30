//go:build e2e

package e2e

type (
	// moduleNameKey is the key for a module name in the module params map.
	moduleNameKey = string
	// paramNameKey is the key for a param name in the params map.
	paramNameKey = string
)

// paramsAnyMap is a map of param names to param any values.
type paramsAnyMap map[paramNameKey]paramAny

// moduleParamsMap is a map of module names to params maps.
type moduleParamsMap map[moduleNameKey]paramsAnyMap

// paramAny is a struct that holds a param type and a param value.
type paramAny struct {
	name    string
	typeStr string
	value   any
}

// authzCLIGrantResponse is the JSON response struct for the authz grants query
// CLI subcommand: `pocketd query authz grants <granter_addr> <grantee_addr>`.
// NB: `authz.QueryGrantsResponse` is not used because it seems to be incompatible
// with the JSON response format of the authz CLI query subcommand.
type authzCLIGrantResponse struct {
	Grants []struct {
		Authorization struct {
			Type  string `json:"type"`
			Value struct {
				Msg string `json:"msg"`
			} `json:"value"`
		} `json:"authorization"`
	} `json:"grants"`
}
