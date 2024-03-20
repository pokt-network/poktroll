package sdk

import (
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGetTransportCreds(t *testing.T) {

	systemRoots, err := x509.SystemCertPool()
	require.NoError(t, err)

	tlsCreds := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})

	tests := []struct {
		desc    string
		hostUrl string
		want    credentials.TransportCredentials
	}{
		{
			desc:    "Test grpcs scheme",
			hostUrl: "grpcs://poktroll.com",
			want:    tlsCreds,
		},
		{
			desc:    "Test any non-grpcs scheme",
			hostUrl: "http://poktroll.com",
			want:    insecure.NewCredentials(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, _ := url.Parse(tc.hostUrl)

			got, err := getTransportCreds(u)
			require.Nil(t, err)

			// Comparing tc.want and got directly resulted in an error due to unexported fields
			require.Equal(t, tc.want.Info(), got.Info())
		})
	}
}
