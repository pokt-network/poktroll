package migrate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

type morseStateInputLocals struct {
	Address1Hex,
	Address2Hex,
	PublicKey1Hex,
	PublicKey2Hex string
	Account1Amount,
	Account2Amount,
	App1StakeAmount,
	App2StakeAmount,
	Supplier1StakeAmount,
	Supplier2StakeAmount uint64
}

type morseStateOutputLocals struct {
	Address1Hex,
	Address2Hex,
	PublicKey1Hex,
	PublicKey2Hex string
	Account1Amount,
	Account2Amount uint64
}

var (
	mockMorseStateExport = template.Must(template.New("morse_state_export.json").Parse(`
{
    "app_hash": "",
    "app_state": {
        "application": {
            "applications": [
                {
                    "address": "{{.Address1Hex}}",
                    "public_key": "{{.PublicKey1Hex}}",
                    "jailed": false,
                    "status": 2,
                    "staked_tokens": "{{.App1StakeAmount}}"
                },
                {
                    "address": "{{.Address2Hex}}",
                    "public_key": "{{.PublicKey2Hex}}",
                    "jailed": false,
                    "status": 2,
                    "staked_tokens": "{{.App2StakeAmount}}"
                }
            ]
        },
        "auth": {
            "accounts": [
                {
                    "type": "posmint/Account",
                    "value": {
                        "address": "{{.Address1Hex}}",
                        "coins": [
                            {
                                "denom": "upokt",
                                "amount": "{{.Account1Amount}}"
                            }
                        ],
                        "public_key": {
                            "type": "crypto/ed25519_public_key",
                            "value": "{{.PublicKey1Hex}}"
                        }
                    }
                },
                {
                    "type": "posmint/Account",
                    "value": {
                        "address": "{{.Address2Hex}}",
                        "coins": [
                            {
                                "denom": "upokt",
                                "amount": "{{.Account2Amount}}"
                            }
                        ],
                        "public_key": {
                            "type": "crypto/ed25519_public_key",
                            "value": "{{.PublicKey2Hex}}"
                        }
                    }
                }
            ]
        },
        "pos": {
            "validators": [
                {
                    "address": "{{.Address1Hex}}",
                    "public_key": "{{.PublicKey1Hex}}",
                    "jailed": false,
                    "status": 2,
                    "tokens": "{{.Supplier1StakeAmount}}"
                },
                {
                    "address": "{{.Address2Hex}}",
                    "public_key": "{{.PublicKey2Hex}}",
                    "jailed": false,
                    "status": 2,
                    "tokens": "{{.Supplier2StakeAmount}}"
                }
            ]
        }
    }
}
`))
	expectedMorseAccountStateJSONTmpl = template.Must(template.New("morse_account_state.json").Parse(`{
  "accounts": [
    {
      "address": "{{.Address1Hex}}",
      "public_key": {
        "value": "{{.PublicKey1Hex}}"
      },
      "coins": [
        {
          "denom": "upokt",
          "amount": "{{.Account1Amount}}"
        }
      ]
    },
    {
      "address": "{{.Address2Hex}}",
      "public_key": {
        "value": "{{.PublicKey2Hex}}"
      },
      "coins": [
        {
          "denom": "upokt",
          "amount": "{{.Account2Amount}}"
        }
      ]
    }
  ]
}`))

	Account1Amount       = uint64(2000000)
	Account2Amount       = uint64(2000020)
	App1StakeAmount      = uint64(10000000000000)
	App2StakeAmount      = uint64(10000000000001)
	Supplier1StakeAmount = uint64(30000000)
	Supplier2StakeAmount = uint64(30000300)
	Address1Hex          = "934066AAE79DA1E8012BACF4953985DC6BAC3371"
	Address2Hex          = "3145CF09E0E780A16E57DE7DB2A419CFEA45C830"
	PublicKey1Hex        = "f68e32d72e7f5f1c797bcd41d8d0e9a1004354c6b1c85429f2ebd7d82ccf4a70"
	PublicKey2Hex        = "0a825f4415213910f949b9081ee43cca105eae13ca44bb69e93aaad122f52c11"

	morseInputLocals = morseStateInputLocals{
		Address1Hex:          Address1Hex,
		Address2Hex:          Address2Hex,
		PublicKey1Hex:        PublicKey1Hex,
		PublicKey2Hex:        PublicKey2Hex,
		Account1Amount:       Account1Amount,
		Account2Amount:       Account2Amount,
		App1StakeAmount:      App1StakeAmount,
		App2StakeAmount:      App2StakeAmount,
		Supplier1StakeAmount: Supplier1StakeAmount,
		Supplier2StakeAmount: Supplier2StakeAmount,
	}

	morseOutputLocals = morseStateOutputLocals{
		Address1Hex:    Address1Hex,
		Address2Hex:    Address2Hex,
		PublicKey1Hex:  PublicKey1Hex,
		PublicKey2Hex:  PublicKey2Hex,
		Account1Amount: App1StakeAmount + Account1Amount + Supplier1StakeAmount,
		Account2Amount: App2StakeAmount + Account2Amount + Supplier2StakeAmount,
	}
)

func TestCollectMorseAccounts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "morse-state-output.json")
	inputFile, err := os.CreateTemp(tmpDir, "morse-state-input.json")
	require.NoError(t, err)

	err = mockMorseStateExport.Execute(inputFile, morseInputLocals)
	require.NoError(t, err)

	err = inputFile.Close()
	require.NoError(t, err)

	// Call the function under test.
	err = collectMorseAccounts(inputFile.Name(), outputPath)
	require.NoError(t, err)

	outputJSON, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	expectedJSONBuf := new(bytes.Buffer)
	err = expectedMorseAccountStateJSONTmpl.Execute(expectedJSONBuf, morseOutputLocals)
	require.NoError(t, err)

	// Strip all whitespace from the expected JSON.
	expectedJSON := expectedJSONBuf.String()
	expectedJSON = strings.ReplaceAll(expectedJSON, "\n", "")
	expectedJSON = strings.ReplaceAll(expectedJSON, " ", "")

	require.NoError(t, err)
	require.Equal(t, expectedJSON, string(outputJSON))
}
