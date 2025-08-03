package sample

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
)

func init() {
	cmd.InitSDKConfig()
}

func TestConsAddress(t *testing.T) {
	base64Addr := "FO/Fr8EaBW/Zi1l8pTcqkhr4WwQ="
	addrBytes, _ := base64.StdEncoding.DecodeString(base64Addr)
	fmt.Println(types.ConsAddress(addrBytes).String())
}

func TestDebugAddr(t *testing.T) {

}

// pocketd debug addr poktvaloper18kk3aqe2pjz7x7993qp2pjt95ghurra9c5ef0
// Address: [61 173 30 131 42 12 133 227 120 165 136 2 160 201 101 162 47 193 143 165]
// Address (hex): 3DAD1E832A0C85E378A58802A0C965A22FC18FA5
// Bech32 Acc: pokt18kk3aqe2pjz7x7993qp2pjt95ghurra9682tyn
// Bech32 Val: poktvaloper18kk3aqe2pjz7x7993qp2pjt95ghurra9c5ef0t
// Bech32 Con: poktvalcons18kk3aqe2pjz7x7993qp2pjt95ghurra9v824r2
