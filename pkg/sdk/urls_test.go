package sdk_test

import (
	"net/url"
	"testing"

	"github.com/pokt-network/poktroll/pkg/sdk"
	"github.com/stretchr/testify/require"
)

func TestRPCToWebsocketURL(t *testing.T) {
	tests := []struct {
		desc    string
		hostUrl string
		expectedUrl    string
	}{
		{
			desc:    "https results in wss",
			hostUrl: "https://poktroll.com",
			want:    "wss://poktroll.com/websocket",
		},
		{
			desc:    "wss stays wss",
			hostUrl: "wss://poktroll.com",
			want:    "wss://poktroll.com/websocket",
		},
		{
			desc:    "http results in ws",
			hostUrl: "http://poktroll.com",
			want:    "ws://poktroll.com/websocket",
		},
		{
			desc:    "default is wss",
			hostUrl: "other://poktroll.com/websocket",
			want:    "wss://poktroll.com/websocket",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.hostUrl)
			require.NoError(t, err)
			got := sdk.RPCToWebsocketURL(u)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestConstructGRPCUrl(t *testing.T) {
	tests := []struct {
		desc    string
		hostUrl string
		want    string
	}{
		{
			desc:    "https results in grpcs",
			hostUrl: "https://poktroll.com",
			want:    "grpcs://poktroll.com",
		},
		{
			desc:    "https with port results in grpcs",
			hostUrl: "https://poktroll.com:443",
			want:    "grpcs://poktroll.com:443",
		},
		{
			desc:    "grpcs stays grpcs",
			hostUrl: "grpcs://poktroll.com",
			want:    "grpcs://poktroll.com",
		},
		{
			desc:    "grpc stays grpc",
			hostUrl: "grpc://poktroll.com",
			want:    "grpc://poktroll.com",
		},
		{
			desc:    "tcp stays tcp",
			hostUrl: "tcp://poktroll.com",
			want:    "tcp://poktroll.com",
		},
		{
			desc:    "http results in grpc",
			hostUrl: "http://poktroll.com",
			want:    "grpc://poktroll.com",
		},
		{
			desc:    "default is grpcs",
			hostUrl: "other://poktroll.com",
			want:    "grpcs://poktroll.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.hostUrl)
			require.Nil(t, err)
			got := sdk.ConstructGRPCUrl(u)
			require.Equal(t, tc.want, got)
		})
	}
}
