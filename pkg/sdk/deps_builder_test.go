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
		desc    string
		hostUrl string
		want    credentials.TransportCredentials
	}{
		{
			desc:    "Test grpcs scheme",
			hostUrl: "grpcs://poktroll.com",
			want:    credentials.NewTLS(&tls.Config{}),
		},
		{
			desc:    "Test any non-grpcs scheme",
			hostUrl: "http://poktroll.com",
			want:    insecure.NewCredentials(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.hostUrl)
			require.NoError(t, err)

			got, err := getTransportCreds(u)
			require.Nil(t, err)

			// Comparing tc.want and got directly resulted in an error due to unexported fields
			require.Equal(t, tc.want.Info(), got.Info())
		})
	}
}
