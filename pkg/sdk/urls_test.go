package sdk_test

import (
	"net/url"
	"testing"

	"github.com/pokt-network/poktroll/pkg/sdk"
	"github.com/stretchr/testify/require"
)

func TestHostToWebsocketURL(t *testing.T) {
	tests := []struct {
		desc    string
		hostUrl string
		want    string
	}{
		{
			desc:    "Test HTTPS",
			hostUrl: "https://poktroll.com",
			want:    "wss://poktroll.com/websocket",
		},
		{
			desc:    "Test HTTP",
			hostUrl: "http://poktroll.com",
			want:    "ws://poktroll.com/websocket",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, _ := url.Parse(tc.hostUrl)
			got := sdk.HostToWebsocketURL(u)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestHostToGRPCUrl(t *testing.T) {
	tests := []struct {
		desc    string
		hostUrl string
		want    string
	}{
		{
			desc:    "Test HTTPS",
			hostUrl: "https://poktroll.com",
			want:    "grpcs://poktroll.com",
		},
		{
			desc:    "Test with port",
			hostUrl: "https://poktroll.com:443",
			want:    "grpcs://poktroll.com:443",
		},
		{
			desc:    "Test gRPCs",
			hostUrl: "grpcs://poktroll.com",
			want:    "grpcs://poktroll.com",
		},
		{
			desc:    "Test HTTP",
			hostUrl: "http://poktroll.com",
			want:    "grpc://poktroll.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, _ := url.Parse(tc.hostUrl)
			got := sdk.HostToGRPCUrl(u)
			require.Equal(t, tc.want, got)
		})
	}
}
