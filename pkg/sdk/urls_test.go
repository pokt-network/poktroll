package sdk_test

import (
	"net/url"
	"testing"

	"github.com/pokt-network/poktroll/pkg/sdk"
)

func TestHostToWebsocketURL(t *testing.T) {
	tests := []struct {
		name    string
		hostUrl string
		want    string
	}{
		{
			name:    "Test HTTPS",
			hostUrl: "https://poktroll.com",
			want:    "wss://poktroll.com/websocket",
		},
		{
			name:    "Test HTTP",
			hostUrl: "http://poktroll.com",
			want:    "ws://poktroll.com/websocket",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, _ := url.Parse(tc.hostUrl)
			got := sdk.HostToWebsocketURL(u)
			if got != tc.want {
				t.Fatalf("expected %s; got %s", tc.want, got)
			}
		})
	}
}

func TestHostToGRPCUrl(t *testing.T) {
	tests := []struct {
		name    string
		hostUrl string
		want    string
	}{
		{
			name:    "Test HTTPS",
			hostUrl: "https://poktroll.com",
			want:    "grpcs://poktroll.com",
		},
		{
			name:    "Test with port",
			hostUrl: "https://poktroll.com:443",
			want:    "grpcs://poktroll.com:443",
		},
		{
			name:    "Test gRPCs",
			hostUrl: "grpcs://poktroll.com",
			want:    "grpcs://poktroll.com",
		},
		{
			name:    "Test HTTP",
			hostUrl: "http://poktroll.com",
			want:    "grpc://poktroll.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, _ := url.Parse(tc.hostUrl)
			got := sdk.HostToGRPCUrl(u)
			if got != tc.want {
				t.Fatalf("expected %s; got %s", tc.want, got)
			}
		})
	}
}
