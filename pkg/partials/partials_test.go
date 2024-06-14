package partials

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	sdkerror "cosmossdk.io/errors"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_TEST(@red-0ne): Expand coverage with more test cases when more request
// types are implemented in the partials package.
func TestPartials_GetErrorReply(t *testing.T) {
	_, logCtx := testpolylog.NewLoggerWithCtx(
		context.Background(),
		polyzero.DebugLevel,
	)

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
			name:          "invalid json - unrecognized payload",
			err:           errors.New("test error"),
			payload:       []byte(`{"invalid": "payload"}`),
			expectedReply: nil,
			expectedErr:   ErrPartialUnrecognizedRequestFormat,
		},
		{
			name:          "invalid - unrecognized payload",
			err:           errors.New("test error"),
			payload:       []byte("invalid payload"),
			expectedReply: nil,
			expectedErr:   ErrPartialUnrecognizedRequestFormat,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// build a serialized http request to detect the RPC type from
			httpReq := &http.Request{
				Method: http.MethodPost,
				Header: http.Header{},
				URL:    &url.URL{},
				Body:   io.NopCloser(bytes.NewReader(test.payload)),
			}
			httpReqBz, err := sdktypes.SerializeHTTPRequest(httpReq)
			require.NoError(t, err)

			// Generate the error reply
			replyBz, err := GetErrorReply(logCtx, httpReqBz, test.err)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
			// Unmarshal the payload to test reply equality
			partialReq, err := PartiallyUnmarshalRequest(logCtx, test.payload)
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
