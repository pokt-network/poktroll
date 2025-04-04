package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testmigration"
)

var (
	flagNumAccounts         = "num-accounts"
	flagShortNumAccounts    = "n"
	flagDistributionFn      = "dist-fn"
	flagShortDistributionFn = "d"

	logger = polyzero.NewLogger(polyzero.WithLevel(polyzero.InfoLevel))

	numAccounts        int
	distributionFnName string

	cmdGen = &cobra.Command{
		Use: "gen",
		// TODO_IN_THIS_COMMIT: ...
		//Short:                      "",
		//Long:                       "",
		//Example:                    "",
		Args: cobra.ExactArgs(0),
		RunE: runGen,
	}
)

const (
	roundRobinDistributionName     = "round-robin"
	allUnstakedDistributionName    = "unstaked"
	allApplicationDistributionName = "application"
	allSupplierDistributionName    = "supplier"
)

func init() {
	// TODO_IN_THIS_COMMIT: extract the following flags:
	// - distribution
	// - number of accounts
	// - output paths
	cmdGen.Flags().IntVarP(&numAccounts, flagNumAccounts, flagShortNumAccounts, 10, "The number of accounts to generate")
	cmdGen.Flags().StringVarP(&distributionFnName, flagDistributionFn, flagShortDistributionFn, roundRobinDistributionName, "The distribution function to use")
}

func main() {
	if err := cmdGen.Execute(); err != nil {
		logger.Error().Err(err).Msg("exiting due to error")
		os.Exit(1)
	}
}

func runGen(_ *cobra.Command, _ []string) error {
	distributionFn, err := getDistributionFn(distributionFnName)
	if err != nil {
		return err
	}

	morseStateExportBz, morseAccountStateBz, err := testmigration.NewMorseStateExportAndAccountStateBytes(numAccounts, distributionFn)
	if err != nil {
		return err
	}

	if err := writePrivateKeys(); err != nil {
		return err
	}

	if err = os.WriteFile("morse_state_export.json", morseStateExportBz, 0644); err != nil {
		return err
	}

	return os.WriteFile("morse_account_state.json", morseAccountStateBz, 0644)
}

// TODO_IN_THIS_COMMIT: godoc and move...
func getDistributionFn(distributionFnName string) (distributionFn testmigration.MorseAccountActorTypeDistributionFn, err error) {
	switch distributionFnName {
	case roundRobinDistributionName:
		distributionFn = testmigration.RoundRobinAllMorseAccountActorTypes
	case allUnstakedDistributionName:
		distributionFn = testmigration.AllUnstakedMorseAccountActorType
	case allApplicationDistributionName:
		distributionFn = testmigration.AllApplicationMorseAccountActorType
	case allSupplierDistributionName:
		distributionFn = testmigration.AllSupplierMorseAccountActorType
	default:
		return nil, fmt.Errorf("unknown distribution function %q", distributionFnName)
	}

	return distributionFn, nil
}

// TODO_IN_THIS_COMMIT: godoc...
func writePrivateKeys() (err error) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	defer func() {
		if err != nil {
			return
		}
		err = writer.Flush()
	}()

	// Write the header row.
	header := []string{"index", "seed", "address", "privKey (base64)"}
	if _, err := writer.Write([]byte(fmt.Sprintf("%s\n", strings.Join(header, "\t")))); err != nil {
		return err
	}

	keysDirPath := filepath.Join(".", "morse_keys")
	if err := os.MkdirAll(keysDirPath, 0755); err != nil {
		return err
	}

	for i := 0; i < numAccounts; i++ {
		//randSeed := rand.Uint64()
		randSeed := uint64(i)
		privateKey := testmigration.GenMorsePrivateKey(randSeed)
		address := privateKey.PubKey().Address().String()
		pkBase64 := base64.StdEncoding.EncodeToString(privateKey)

		keyPath := filepath.Join(keysDirPath, fmt.Sprintf("%s_%d.key", address, i))
		keyArmoredJSON, err := testmigration.EncryptArmorPrivKey(privateKey, "", "")
		if err != nil {
			return err
		}

		if err = os.WriteFile(keyPath, []byte(keyArmoredJSON), 0644); err != nil {
			return err
		}

		// Write the index.
		if _, err = writer.Write([]byte(fmt.Sprintf("%d\t", i))); err != nil {
			return err
		}

		// Write the seed.
		if _, err = writer.Write([]byte(fmt.Sprintf("%d\t", randSeed))); err != nil {
			return err
		}

		// Write the address.
		if _, err = writer.Write([]byte(fmt.Sprintf("%s\t", address))); err != nil {
			return err
		}

		// Write the private key.
		if _, err = writer.Write([]byte(fmt.Sprintf("%s\n", pkBase64))); err != nil {
			return err
		}
	}
	return nil
}
