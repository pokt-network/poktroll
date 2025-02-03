package migrate

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func init() {
	logger = polyzero.NewLogger(polyzero.WithLevel(polyzero.DebugLevel))
	flagDebugAccountsPerLog = 1
}

func TestCollectMorseAccounts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "morse-state-output.json")
	inputFile, err := os.CreateTemp(tmpDir, "morse-state-input.json")
	require.NoError(t, err)

	morseStateExportBz, morseAccountStateBz := testmigration.NewMorseStateExportAndAccountStateBytes(t, 10)
	_, err = inputFile.Write(morseStateExportBz)
	require.NoError(t, err)

	err = inputFile.Close()
	require.NoError(t, err)

	// Call the function under test.
	_, err = collectMorseAccounts(inputFile.Name(), outputPath)
	require.NoError(t, err)

	outputJSON, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	expectedJSON := string(morseAccountStateBz)
	require.NoError(t, err)

	// Strip all whitespace from the expected JSON.
	expectedJSON = strings.ReplaceAll(expectedJSON, "\n", "")
	expectedJSON = strings.ReplaceAll(expectedJSON, " ", "")

	require.NoError(t, err)
	require.Equal(t, expectedJSON, string(outputJSON))
}

func TestNewTestMorseStateExport(t *testing.T) {
	for i := 1; i < 10; i++ {
		t.Run(fmt.Sprintf("num_accounts=%d", i), func(t *testing.T) {
			morseStateExport := new(migrationtypes.MorseStateExport)
			stateExportBz, _ := testmigration.NewMorseStateExportAndAccountStateBytes(t, i)
			err := cmtjson.Unmarshal(stateExportBz, morseStateExport)
			require.NoError(t, err)

			exportAccounts := morseStateExport.AppState.Auth.Accounts
			require.Equal(t, i, len(exportAccounts))

			expectedShannonBalance := fmt.Sprintf("%d%d%d0%d%d%d", i, i, i, i, i, i)
			morseWorkspace := newMorseImportWorkspace()
			err = transformMorseState(morseStateExport, morseWorkspace)
			require.NoError(t, err)

			morseAccounts := morseWorkspace.accountState.Accounts[i-1]
			require.Equal(t, expectedShannonBalance, morseAccounts.Coins[0].Amount.String())
		})
	}
}

func BenchmarkTransformMorseState(b *testing.B) {
	for i := 0; i < 5; i++ {
		numAccounts := int(math.Pow10(i + 1))
		morseStateExport := new(migrationtypes.MorseStateExport)
		morseStateExportBz, _ := testmigration.NewMorseStateExportAndAccountStateBytes(b, numAccounts)
		err := cmtjson.Unmarshal(morseStateExportBz, morseStateExport)
		require.NoError(b, err)

		b.Run(fmt.Sprintf("num_accounts=%d", numAccounts), func(b *testing.B) {
			morseWorkspace := newMorseImportWorkspace()

			// Call the function under test.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err = transformMorseState(morseStateExport, morseWorkspace)
				require.NoError(b, err)
			}
		})
	}
}

// TODO_CONSIDERATION: Test/benchmark execution speed can be optimized by refactoring this to a pre-generate fixture.
func newMorseStateExportAndAccountState(
	t gocuke.TestingT,
	numAccounts int,
) (morseStateExportBz []byte, morseAccountStateBz []byte) {
	morseStateExport := &migrationtypes.MorseStateExport{
		AppHash: "",
		AppState: &migrationtypes.MorseAppState{
			Application: &migrationtypes.MorseApplications{},
			Auth:        &migrationtypes.MorseAuth{},
			Pos:         &migrationtypes.MorsePos{},
		},
	}

	morseAccountState := &migrationtypes.MorseAccountState{
		Accounts: make([]*migrationtypes.MorseAccount, numAccounts),
	}

	for i := 1; i < numAccounts+1; i++ {
		seedUint := rand.Uint64()
		seedBz := make([]byte, 8)
		binary.LittleEndian.PutUint64(seedBz, seedUint)
		privKey := cometcrypto.GenPrivKeyFromSecret(seedBz)
		pubKey := privKey.PubKey()
		balanceAmount := int64(1e6*i + i)                                 // i_000_00i
		appStakeAmount := int64(1e5*i + (i * 10))                         //   i00_0i0
		supplierStakeAmount := int64(1e4*i + (i * 100))                   //    i0_i00
		sumAmount := balanceAmount + appStakeAmount + supplierStakeAmount // i_ii0_iii

		// Add an account.
		morseStateExport.AppState.Auth.Accounts = append(
			morseStateExport.AppState.Auth.Accounts,
			&migrationtypes.MorseAuthAccount{
				Type: "posmint/Account",
				Value: &migrationtypes.MorseAccount{
					Address: pubKey.Address(),
					Coins:   cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, balanceAmount)),
					PubKey: &migrationtypes.MorsePublicKey{
						Value: pubKey.Bytes(),
					},
				},
			},
		)

		// Add an application.
		morseStateExport.AppState.Application.Applications = append(
			morseStateExport.AppState.Application.Applications,
			&migrationtypes.MorseApplication{
				Address:      pubKey.Address(),
				PublicKey:    pubKey.Bytes(),
				Jailed:       false,
				Status:       2,
				StakedTokens: fmt.Sprintf("%d", appStakeAmount),
			},
		)

		// Add a supplier.
		morseStateExport.AppState.Pos.Validators = append(
			morseStateExport.AppState.Pos.Validators,
			&migrationtypes.MorseValidator{
				Address:      pubKey.Address(),
				PublicKey:    pubKey.Bytes(),
				Jailed:       false,
				Status:       2,
				StakedTokens: fmt.Sprintf("%d", supplierStakeAmount),
			},
		)

		// Add the account to the morseAccountState.
		morseAccountState.Accounts[i-1] = &migrationtypes.MorseAccount{
			Address: pubKey.Address(),
			Coins:   cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, sumAmount)),
			PubKey: &migrationtypes.MorsePublicKey{
				Value: pubKey.Bytes(),
			},
		}
	}

	var err error
	morseStateExportBz, err = cmtjson.Marshal(morseStateExport)
	require.NoError(t, err)

	morseAccountStateBz, err = cmtjson.Marshal(morseAccountState)
	require.NoError(t, err)

	return morseStateExportBz, morseAccountStateBz
}
