package partials

import (
	"encoding/json"
	"errors"
	"testing"

	sdkerror "cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO(@h5law): Expand coverage with more test cases when more request types
// are implemented in the partials package.
func TestPartials_GetErrorReply(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		payload       []byte
		expectedReply []byte
		expectedErr   *sdkerror.Error
	}{
		{
			name: "valid json - properly formatted payload",
			err:  errors.New("test error"),
			payload: []byte(
				`{"id":1,"jsonrpc":"2.0","method":"eth_getBlockNumber","params":{"these":"are","ignored":0}}`,
			),
			expectedReply: []byte(
				`{"id":1,"jsonrpc":"2.0","error":{"code":-32000,"data":null,"message":"test error"}}`,
			),
			expectedErr: nil,
		},
		{
			name:          "invalid json - unrecognised payload",
			err:           errors.New("test error"),
			payload:       []byte(`{"invalid": "payload"}`),
			expectedReply: nil,
			expectedErr:   ErrPartialUnrecognisedRequestFormat,
		},
		{
			name:          "invalid - unrecognised payload",
			err:           errors.New("test error"),
			payload:       []byte("invalid payload"),
			expectedReply: nil,
			expectedErr:   ErrPartialUnrecognisedRequestFormat,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Generate the error reply
			replyBz, err := GetErrorReply(test.payload, test.err)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
			// Unmarshal the payload to test reply equality
			partialReq, err := PartiallyUnmarshalRequest(test.payload)
			require.NoError(t, err)
			require.NotNil(t, partialReq)
			switch partialReq.GetRPCType() {
			case sharedtypes.RPCType_JSON_RPC:
				reply := make(map[string]any)
				err = json.Unmarshal(replyBz, &reply)
				require.NoError(t, err)
				expectedReply := make(map[string]any)
				err = json.Unmarshal(test.expectedReply, &expectedReply)
				require.NoError(t, err)
				for k, v := range expectedReply {
					require.Equal(t, v, reply[k])
				}
				require.Equal(t, len(reply), len(expectedReply))
			}
		})
	}
}
