package sdk

import (
	"crypto/tls"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGetTransportCreds(t *testing.T) {
	tests := []struct {
		desc                string
		hostUrl             string
		expectedCredentials credentials.TransportCredentials
	}{
		{
			desc:                "Test http results in insecure",
			hostUrl:             "http://poktroll.com",
			expectedCredentials: insecure.NewCredentials(),
		},
		{
			desc:                "Test tcp results in insecure",
			hostUrl:             "tcp://poktroll.com",
			expectedCredentials: insecure.NewCredentials(),
		},
		{
			desc:                "Test default is tls credentials",
			hostUrl:             "other://poktroll.com",
			expectedCredentials: credentials.NewTLS(&tls.Config{}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.hostUrl)
			require.NoError(t, err)

			got, err := getTransportCreds(u)
			require.Nil(t, err)

			// Comparing tc.want and got directly resulted in an error due to unexported fields
			require.Equal(t, tc.expectedCredentials.Info(), got.Info())
		})
	}
}
