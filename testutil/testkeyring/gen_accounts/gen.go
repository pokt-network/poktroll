//go:build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"cosmossdk.io/depinject"
	sdklog "cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
)

var (
	flagOut           string
	flagAccountsLimit int
	defaultOutPath    = "accounts_table.go"
)

func init() {
	flag.StringVar(&flagOut, "out", defaultOutPath, "the path to the generated go source of pre-generated accounts.")
	flag.IntVar(&flagAccountsLimit, "limit", 100, "the number of accounts to generate.")

	cmd.InitSDKConfig()
}

func main() {
	flag.Parse()

	var marshaler codec.Codec
	deps := depinject.Configs(
		app.AppConfig(),
		depinject.Supply(sdklog.NewNopLogger()),
	)
	if err := depinject.Inject(deps, &marshaler); err != nil {
		log.Fatal(err)
	}
	kr := keyring.NewInMemory(marshaler)

	preGeneratedAccountLines := make([]string, flagAccountsLimit)
	for i := range preGeneratedAccountLines {
		record, mnemonic, err := kr.NewMnemonic(
			fmt.Sprintf("key-%d", i),
			keyring.English,
			cosmostypes.FullFundraiserPath,
			keyring.DefaultBIP39Passphrase,
			hd.Secp256k1,
		)
		addr, err := record.GetAddress()
		if err != nil {
			log.Fatal(err)
		}

		preGeneratedAccount := &testkeyring.PreGeneratedAccount{
			Address:  addr,
			Mnemonic: mnemonic,
		}

		preGeneratedAccountStr, err := preGeneratedAccount.Marshal()
		if err != nil {
			log.Fatal(err)
		}

		preGeneratedAccountLines[i] = fmt.Sprintf(preGeneratedAccountLineFmt, preGeneratedAccountStr)
	}

	newPreGeneratedAccountIteratorArgLines := strings.Join(preGeneratedAccountLines, "\n")
	outputBuffer := new(bytes.Buffer)
	if err := accountsTableTemplate.Execute(
		outputBuffer,
		map[string]any{
			"newPreGeneratedAccountIteratorArgLines": newPreGeneratedAccountIteratorArgLines,
		},
	); err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(flagOut, outputBuffer.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
}
