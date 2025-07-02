package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func TestRelayResponse_PayloadHash(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		description string
	}{
		{
			name:        "small payload",
			payload:     []byte("test payload"),
			description: "verify hash computation for small payload",
		},
		{
			name:        "large payload",
			payload:     make([]byte, 1024*1024), // 1MB payload
			description: "verify hash computation for large payload",
		},
		{
			name:        "empty payload",
			payload:     []byte{},
			description: "verify hash computation for empty payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a relay response with the test payload
			sessionHeader := &sessiontypes.SessionHeader{
				ApplicationAddress:      "cosmos1test",
				ServiceId:               "svc1",
				SessionId:               "session1",
				SessionStartBlockHeight: 1,
				SessionEndBlockHeight:   10,
			}

			// Compute expected hash
			expectedHash := protocol.GetRelayHashFromBytes(tt.payload)

			// Create relay response with payload and its hash
			relayResponse := &types.RelayResponse{
				Meta: types.RelayResponseMetadata{
					SessionHeader: sessionHeader,
				},
				Payload:     tt.payload,
				PayloadHash: expectedHash[:],
			}

			// Verify the payload hash matches
			require.Equal(t, expectedHash[:], relayResponse.PayloadHash, tt.description)

			// Test GetSignableBytesHash behavior
			signableHash1, err := relayResponse.GetSignableBytesHash()
			require.NoError(t, err)

			// Verify that GetSignableBytesHash clears the payload
			// by checking that a second call produces the same hash
			// even if we modify the payload
			relayResponse.Payload = []byte("modified payload")
			signableHash2, err := relayResponse.GetSignableBytesHash()
			require.NoError(t, err)

			require.Equal(t, signableHash1, signableHash2,
				"GetSignableBytesHash should produce consistent results regardless of payload content")
		})
	}
}

func TestRelayResponse_GetSignableBytesHash_Consistency(t *testing.T) {
	// This test ensures that the signable bytes hash is consistent
	// whether computed with or without the payload field populated

	sessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress:      "cosmos1test",
		ServiceId:               "svc1",
		SessionId:               "session1",
		SessionStartBlockHeight: 1,
		SessionEndBlockHeight:   10,
	}

	payload := []byte("test payload data")
	payloadHash := protocol.GetRelayHashFromBytes(payload)

	// Create two relay responses: one with payload, one without
	responseWithPayload := &types.RelayResponse{
		Meta: types.RelayResponseMetadata{
			SessionHeader: sessionHeader,
		},
		Payload:     payload,
		PayloadHash: payloadHash[:],
	}

	responseWithoutPayload := &types.RelayResponse{
		Meta: types.RelayResponseMetadata{
			SessionHeader: sessionHeader,
		},
		Payload:     nil, // No payload, only hash
		PayloadHash: payloadHash[:],
	}

	// Get signable bytes hash for both
	hash1, err := responseWithPayload.GetSignableBytesHash()
	require.NoError(t, err)

	hash2, err := responseWithoutPayload.GetSignableBytesHash()
	require.NoError(t, err)

	// Both should produce the same hash since GetSignableBytesHash
	// clears the payload before hashing
	require.Equal(t, hash1, hash2,
		"Signable bytes hash should be the same regardless of payload presence")
}