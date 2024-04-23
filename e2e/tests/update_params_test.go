//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"strings"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func (s *suite) AllModuleParamsAreSetToTheirDefaultValues(moduleName string) {
	args := strings.Split(fmt.Sprintf("query %s params", moduleName), " ")
	args = append(args, "--output", "json")
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	s.Log(res.Stdout)

	switch moduleName {
	case tokenomicstypes.ModuleName:
		var tokenomicsParamsRes tokenomicstypes.QueryParamsResponse
		err = s.cdc.UnmarshalJSON([]byte(res.Stdout), &tokenomicsParamsRes)
		require.NoError(s, err)

		require.Equal(s, tokenomicstypes.DefaultParams(), tokenomicsParamsRes.GetParams())
	case prooftypes.ModuleName:
		var proofParamsRes prooftypes.QueryParamsResponse
		err = s.cdc.UnmarshalJSON([]byte(res.Stdout), &proofParamsRes)
		require.NoError(s, err)

		require.Equal(s, prooftypes.DefaultParams(), proofParamsRes.GetParams())
	default:
		s.Fatal("unexpected module name")
	}
}

func (s *suite) AnAuthzGrantFromTheAccountToTheAccountForTheMessage(granterName, granterAddrType, granteeName, granteeAddrType string, msgType string) {
	var granterAddr, granteeAddr string
	nameToAddrMap := map[string]*string{
		granterName: &granterAddr,
		granteeName: &granteeAddr,
	}
	nameToAddrTypeMap := map[string]string{
		granterName: granterAddrType,
		granteeName: granteeAddrType,
	}

	for name, addr := range nameToAddrMap {
		switch nameToAddrTypeMap[name] {
		case "module":
			*addr = authtypes.NewModuleAddress(name).String()
		case "user":
			*addr = s.getKeyAddress(name)
		default:
			s.Fatal("unexpected address type")
		}
	}

	// Query the authz module for grants with the expected granter & grantee (authz.QueryGrantsRequest).
	args := strings.Split(
		fmt.Sprintf(
			"query authz grants %s %s %s --output json",
			granterAddr, granteeAddr, msgType,
		), " ",
	)
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	// TODO_IN_THIS_COMMIT: is there a better way to unmarshal this?
	// Using s.cdc.UnmarshalJSON() with *authz.QueryGrantsResponse complains about
	// the "type" field, expecting "@type" for the #Grants[n]#Authorization pb.Any.
	var grantsRes struct {
		Grants []struct {
			Authorization struct {
				Type  string `json:"type"`
				Value struct {
					Msg string `json:"msg"`
				} `json:"value"`
			} `json:"authorization"`
		} `json:"grants"`
	}

	err = json.Unmarshal([]byte(res.Stdout), &grantsRes)
	require.NoError(s, err)

	grantFound := false
	for _, grant := range grantsRes.Grants {
		if grant.Authorization.Value.Msg == msgType {
			grantFound = true
			break
		}
	}
	require.True(s, grantFound)
}

func (s *suite) TheUserSendsAnAuthzExecMessageToUpdateAllModuleParams(moduleName string, table gocuke.DataTable) {
	//panic("PENDING")
}

func (s *suite) AllModuleParamsShouldBeUpdated(moduleName string, table gocuke.DataTable) {
	//panic("PENDING")
}

func (s *suite) TheUserSendsAnAuthzExecMessageToUpdateModuleParam(moduleName, paramName string, table gocuke.DataTable) {
	//panic("PENDING")
}

func (s *suite) TheModuleParamShouldBeUpdated(moduleName, paramName string, table gocuke.DataTable) {
	//panic("PENDING")
}

func (s *suite) getKeyAddress(keyName string) string {
	cmd := "keys show %s --keyring-backend test --output json"
	args := strings.Split(fmt.Sprintf(cmd, keyName), " ")
	res, err := s.pocketd.RunCommand(args...)
	require.NoError(s, err)

	keyListRes := make(map[string]any)
	err = json.Unmarshal([]byte(res.Stdout), &keyListRes)
	require.NoError(s, err)

	return keyListRes["address"].(string)
}
