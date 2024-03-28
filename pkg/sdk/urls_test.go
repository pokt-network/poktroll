package sdk_test

import (
	"net/url"
	"testing"

	"github.com/pokt-network/poktroll/pkg/sdk"
	"github.com/stretchr/testify/require"
)

func TestRPCToWebsocketURL(t *testing.T) {
	tests := []struct {
		desc        string
		hostUrl     string
		expectedUrl string
	}{
		{
			desc:        "https results in wss",
			hostUrl:     "https://poktroll.com",
			expectedUrl: "wss://poktroll.com/websocket",
		},
		{
			desc:        "wss stays wss",
			hostUrl:     "wss://poktroll.com",
			expectedUrl: "wss://poktroll.com/websocket",
		},
		{
			desc:        "http results in ws",
			hostUrl:     "http://poktroll.com",
			expectedUrl: "ws://poktroll.com/websocket",
		},
		{
			desc:        "default is wss",
			hostUrl:     "other://poktroll.com/websocket",
			expectedUrl: "wss://poktroll.com/websocket",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.hostUrl)
			require.NoError(t, err)
			got := sdk.RPCToWebsocketURL(u)
			require.Equal(t, tc.expectedUrl, got)
		})
	}
}

func TestConstructGRPCUrl(t *testing.T) {
	tests := []struct {
		desc        string
		hostUrl     string
		expectedUrl string
	}{
		{
			desc:        "https results in grpcs",
			hostUrl:     "https://poktroll.com",
			expectedUrl: "grpcs://poktroll.com",
		},
		{
			desc:        "https with port results in grpcs",
			hostUrl:     "https://poktroll.com:443",
			expectedUrl: "grpcs://poktroll.com:443",
		},
		{
			desc:        "grpcs stays grpcs",
			hostUrl:     "grpcs://poktroll.com",
			expectedUrl: "grpcs://poktroll.com",
		},
		{
			desc:        "grpc stays grpc",
			hostUrl:     "grpc://poktroll.com",
			expectedUrl: "grpc://poktroll.com",
		},
		{
			desc:        "tcp stays tcp",
			hostUrl:     "tcp://poktroll.com",
			expectedUrl: "tcp://poktroll.com",
		},
		{
			desc:        "http results in grpc",
			hostUrl:     "http://poktroll.com",
			expectedUrl: "grpc://poktroll.com",
		},
		{
			desc:        "default is grpcs",
			hostUrl:     "other://poktroll.com",
			expectedUrl: "grpcs://poktroll.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.hostUrl)
			require.NoError(t, err)
			got := sdk.ConstructGRPCUrl(u)
			require.Equal(t, tc.expectedUrl, got)
		})
	}
}
