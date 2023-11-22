package partials

import (
	"encoding/json"
	"errors"
	"testing"

	sdkerror "cosmossdk.io/errors"
	"github.com/stretchr/testify/require"
)

// TODO(@h5law): Expand coverage with more test cases when more request types
// are implemented in the partials package.
func TestPartials_GetErrorReply(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		payload     []byte
		payloadType string
		expectedErr *sdkerror.Error
	}{
		{
			name: "valid json - properly formatted payload",
			err:  errors.New("test error"),
			payload: []byte(
				`{"id": 1, "jsonrpc": "2.0", "method": "eth_getBlockNumber", "params": {"these": "are", "ignored": 0}}`,
			),
			payloadType: "json",
			expectedErr: nil,
		},
		{
			name:        "invalid json - unrecognised payload",
			err:         errors.New("test error"),
			payload:     []byte(`{"invalid": "payload"}`),
			payloadType: "json",
			expectedErr: ErrPartialUnrecognisedRequestFormat,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			switch test.payloadType {
			case "json":
				testPayload := map[string]any{}
				require.NoError(t, json.Unmarshal(test.payload, &testPayload))
				replyBz, err := GetErrorReply(test.payload, test.err)
				if test.expectedErr != nil {
					require.ErrorIs(t, err, test.expectedErr)
					return
				}
				require.NoError(t, err)
				unmarshalledReply := map[string]any{}
				require.NoError(t, json.Unmarshal(replyBz, &unmarshalledReply))
				require.Equal(t, unmarshalledReply["jsonrpc"], testPayload["jsonrpc"])
				require.Equal(t, unmarshalledReply["id"], testPayload["id"])
				require.Equal(
					t,
					unmarshalledReply["error"].(map[string]any)["message"],
					test.err.Error(),
				)
			}
		})
	}
}
