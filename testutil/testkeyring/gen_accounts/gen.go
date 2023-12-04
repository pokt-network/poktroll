//go:build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/testutil/testkeyring"

	"github.com/pokt-network/poktroll/app"
)

var (
	flagOut           string
	flagAccountsLimit int
	defaultOutPath    = "accounts_table.go"
)

func init() {
	flag.StringVar(&flagOut, "out", defaultOutPath, "the path to the generated file.")
	flag.IntVar(&flagAccountsLimit, "limit", 100, "the number of accounts to generate.")
}

func main() {
	flag.Parse()

	kr := keyring.NewInMemory(app.MakeEncodingConfig().Marshaler)

	preGeneratedAccountsGobHexLines := make([]string, flagAccountsLimit)
	for i := range preGeneratedAccountsGobHexLines {
		record, mnemonic, err := kr.NewMnemonic(
			fmt.Sprintf("key-%d", i),
			keyring.English,
			types.FullFundraiserPath,
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

		preGeneratedAccountsGobHexLines[i] = fmt.Sprintf(preGeneratedAccountLineFmt, preGeneratedAccountStr)
	}

	newPreGeneratedAccountIteratorArgLines := strings.Join(preGeneratedAccountsGobHexLines, "\n")
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
