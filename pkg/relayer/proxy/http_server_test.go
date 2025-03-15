package proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer/config"

	"github.com/stretchr/testify/require"
)

func TestSynchronousRPCServer_Forward(t *testing.T) {
	tests := []struct {
		name                    string
		inServiceID             string
		inReqBody               []byte
		mockManagedServiceID    string
		mockSupplierHTTPHandler http.HandlerFunc
		expectedForwardBody     string
		expectedStatusCode      int
		expectedError           error
	}{
		{
			name:        "OK",
			inServiceID: "023",
			inReqBody:   []byte(`{"method": "POST", "path": "/", "data": "{\"in\": \"ping\"}"}`),
			mockSupplierHTTPHandler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				b, err := io.ReadAll(req.Body)
				require.NoError(t, err)

				require.Equal(t, `{"in": "ping"}`, string(b))

				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(`{"out": "pong"}`))
			}),
			mockManagedServiceID: "023",
			expectedStatusCode:   http.StatusOK,
			expectedForwardBody:  `{"out": "pong"}`,
		},
		{
			name:        "OK - With error on supplier",
			inServiceID: "024",
			inReqBody:   []byte(`{"method": "POST", "path": "/", "data": "{\"in\": \"ping\"}"}`),
			mockSupplierHTTPHandler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				b, err := io.ReadAll(req.Body)
				require.NoError(t, err)

				require.Equal(t, `{"in": "ping"}`, string(b))

				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(`{"error": "failed"}`))
			}),
			mockManagedServiceID: "024",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedForwardBody:  `{"error": "failed"}`,
		},
		{
			name:          "NOK - Service id not found",
			inServiceID:   "022",
			inReqBody:     []byte(`{"method": "POST", "path": "/", "data": "{\"in\": \"ping\"}"}`),
			expectedError: ErrRelayerProxyServiceIDNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := httptest.NewServer(test.mockSupplierHTTPHandler)
			defer srv.Close()

			req := httptest.NewRequest("POST", "/", io.NopCloser(bytes.NewBuffer(test.inReqBody)))

			url, err := url.Parse(srv.URL)
			require.NoError(t, err)

			sync := &relayMinerHTTPServer{
				serverConfig: &config.RelayMinerServerConfig{
					SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
						test.mockManagedServiceID: &config.RelayMinerSupplierConfig{
							ServiceConfig: &config.RelayMinerSupplierServiceConfig{
								BackendUrl: url,
							},
						},
					},
				},
				logger: polylog.DefaultContextLogger,
			}

			respRecorder := httptest.NewRecorder()

			err = sync.Forward(context.Background(), test.inServiceID, respRecorder, req)
			if test.expectedError != nil {
				require.Error(t, err)
				require.True(t, errors.Is(err, test.expectedError))
			} else {
				require.NoError(t, err)

				b, err := io.ReadAll(respRecorder.Body)
				require.NoError(t, err)

				require.JSONEq(t, test.expectedForwardBody, string(b))
				require.Equal(t, test.expectedStatusCode, respRecorder.Result().StatusCode)
			}
		})
	}
}
