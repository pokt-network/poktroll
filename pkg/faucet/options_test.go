package faucet_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/faucet"
)

var (
	testSupportedSendCoins = []string{testSendUPOKT, testSendMACT}
)

func TestNewConfig_SupportedCoins(t *testing.T) {
	config := &faucet.Config{
		SupportedSendCoins: []string{testSendUPOKT, testSendMACT},
	}

	supportedSendCoins := config.GetSupportedSendCoins()
	require.Equal(t, 2, len(supportedSendCoins))
	require.Equal(t, testSendUPOKT, supportedSendCoins[1].String())
	require.Equal(t, testSendMACT, supportedSendCoins[0].String())
}

func TestNewConfig_LoadSigningKey(t *testing.T) {
	config := &faucet.Config{
		SigningKeyName: testSigningKeyName,
	}
	err := config.LoadSigningKey(clientCtx)
	require.NoError(t, err)
	require.NotNil(t, config.GetSigningAddress)
	require.Equal(t, testSigningAddress.String(), config.GetSigningAddress().String())
}

func TestNewConfig(t *testing.T) {
	config, err := faucet.NewConfig(
		clientCtx,
		testSigningKeyName,
		testListenAddress,
		[]string{"1mact", "100000000000upokt"},
		false,
	)
	require.NoError(t, err)

	err = config.LoadSigningKey(clientCtx)
	require.NoError(t, err)

	err = config.Validate()
	require.NoError(t, err)
}

func TestNewConfig_Validate_Error(t *testing.T) {
	testCases := []struct {
		desc        string
		config      *faucet.Config
		expectedErr error
	}{
		{
			desc: "empty signing key name",
			config: &faucet.Config{
				SigningKeyName:     "",
				ListenAddress:      testListenAddress,
				SupportedSendCoins: testSupportedSendCoins,
				CreateAccountsOnly: false,
			},
			expectedErr: fmt.Errorf("signing key name MUST be set"),
		},
		{
			desc: "empty listen address",
			config: &faucet.Config{
				SigningKeyName:     testSigningKeyName,
				ListenAddress:      "",
				SupportedSendCoins: testSupportedSendCoins,
				CreateAccountsOnly: false,
			},
			expectedErr: fmt.Errorf("listen address MUST be in the form of host:port (e.g. 127.0.0.1:42069)"),
		},
		{
			desc: "URL for listen address",
			config: &faucet.Config{
				SigningKeyName:     testSigningKeyName,
				ListenAddress:      "scheme://host:42069",
				SupportedSendCoins: testSupportedSendCoins,
				CreateAccountsOnly: false,
			},
			expectedErr: fmt.Errorf("listen address MUST be in the form of host:port (e.g. 127.0.0.1:42069)"),
		},
		{
			desc: "invalid send coins (missing denom)",
			config: &faucet.Config{
				SigningKeyName:     testSigningKeyName,
				ListenAddress:      testListenAddress,
				SupportedSendCoins: []string{"123"},
				CreateAccountsOnly: false,
			},
			expectedErr: fmt.Errorf("unable to parse send coins"),
		},
		{
			desc: "invalid send coins (missing amount)",
			config: &faucet.Config{
				SigningKeyName:     testSigningKeyName,
				ListenAddress:      testListenAddress,
				SupportedSendCoins: []string{"xyz"},
				CreateAccountsOnly: false,
			},
			expectedErr: fmt.Errorf("unable to parse send coins"),
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			err := test.config.Validate()
			require.ErrorContains(t, err, test.expectedErr.Error())
		})
	}
}
