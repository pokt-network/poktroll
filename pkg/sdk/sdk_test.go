package sdk_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/sdk"
	testsdk "github.com/pokt-network/poktroll/testutil/sdk"
	"github.com/pokt-network/poktroll/x/service/types"
)

func TestSDK_Dependencies(t *testing.T) {
	tests := []struct {
		desc                 string
		sdkBehavior          *testsdk.TestBehavior
		inputScenario        func(behavior *testsdk.TestBehavior) error
		expectedError        error
		expectedErrorMessage string
	}{
		{
			desc:                 "Successful initialization",
			sdkBehavior:          testsdk.NewTestBehavior(t),
			inputScenario:        initializeSDK,
			expectedError:        nil,
			expectedErrorMessage: "can't resolve type",
		},
		{
			desc:                 "Invalid dependencies",
			sdkBehavior:          testsdk.NewTestBehavior(t).WithDependencies(testsdk.InvalidDependencies),
			inputScenario:        initializeSDK,
			expectedError:        sdk.ErrSDKInvalidConfig,
			expectedErrorMessage: "can't resolve type",
		},
		{
			desc:                 "Missing private key",
			sdkBehavior:          testsdk.NewTestBehavior(t).WithPrivateKey(testsdk.MissingPrivateKey),
			inputScenario:        initializeSDK,
			expectedError:        sdk.ErrSDKInvalidConfig,
			expectedErrorMessage: "missing PrivateKey in config",
		},
		{
			desc:                 "Missing QueryNodeGRPCURL",
			sdkBehavior:          testsdk.NewTestBehavior(t).WithQueryNodeGRPCURL(testsdk.MissingGRPCURL),
			inputScenario:        initializeSDK,
			expectedError:        sdk.ErrSDKInvalidConfig,
			expectedErrorMessage: "missing QueryNodeGRPCURL in config",
		},
		{
			desc:                 "Missing QueryNodeRPCURL",
			sdkBehavior:          testsdk.NewTestBehavior(t).WithQueryNodeRPCURL(testsdk.MissingRPCURL),
			inputScenario:        initializeSDK,
			expectedError:        sdk.ErrSDKInvalidConfig,
			expectedErrorMessage: "missing QueryNodeRPCURL in config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.expectedError == nil {
				require.NoError(t, tt.inputScenario(tt.sdkBehavior))
				return
			}

			err := tt.inputScenario(tt.sdkBehavior)
			require.ErrorIs(t, err, tt.expectedError)
			require.ErrorContains(t, err, tt.expectedErrorMessage)
		})
	}
}

func TestSDK_GetSessionSupplierEndpoints(t *testing.T) {
	tests := []struct {
		desc                 string
		sdkBehavior          *testsdk.TestBehavior
		inputScenario        func(behavior *testsdk.TestBehavior) error
		expectedError        error
		expectedErrorMessage string
	}{
		{
			desc:        "Invalid application address",
			sdkBehavior: testsdk.NewTestBehavior(t),
			inputScenario: callGetSessionSupplierEndpointsWith(
				testsdk.InvalidAppAddress,
				testsdk.ValidServiceID,
			),
			expectedError: sdk.ErrSDKInvalidSession,
		},
		{
			desc:        "Invalid serviceId",
			sdkBehavior: testsdk.NewTestBehavior(t),
			inputScenario: callGetSessionSupplierEndpointsWith(
				testsdk.ValidAppAddress,
				testsdk.InvalidServiceID,
			),
			expectedError: sdk.ErrSDKInvalidSession,
		},
		{
			desc:        "Invalid session",
			sdkBehavior: testsdk.NewTestBehavior(t).WithDependencies(testsdk.NonDefaultLatestBlockHeight),
			inputScenario: callGetSessionSupplierEndpointsWith(
				testsdk.ValidAppAddress,
				testsdk.ValidServiceID,
			),
			expectedError: sdk.ErrSDKInvalidSession,
		},
		{
			desc:          "Successful session retrieval",
			sdkBehavior:   testsdk.NewTestBehavior(t),
			inputScenario: callGetSessionSupplierEndpointsWith(testsdk.ValidAppAddress, testsdk.ValidServiceID),
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.expectedError == nil {
				require.NoError(t, tt.inputScenario(tt.sdkBehavior))
				return
			}

			err := tt.inputScenario(tt.sdkBehavior)
			require.ErrorIs(t, err, tt.expectedError)
			require.ErrorContains(t, err, tt.expectedErrorMessage)
		})
	}
}

func TestSDK_SendRelay(t *testing.T) {
	tests := []struct {
		desc                 string
		SDKBehavior          *testsdk.TestBehavior
		inputScenario        func(behavior *testsdk.TestBehavior) error
		expectedError        error
		expectedErrorMessage string
	}{
		// {
		// 	desc:                 "Invalid request body",
		// 	SDKBehavior:          testsdk.NewTestBehavior(t),
		// 	inputScenario:        callSendRelayWithInvalidBody,
		// 	expectedError:        sdk.ErrSDKHandleRelay,
		// 	expectedErrorMessage: "reading request body",
		// },
		// {
		// 	desc:                 "Invalid app ring",
		// 	SDKBehavior:          testsdk.NewTestBehavior(t),
		// 	inputScenario:        callSendRelayWithInvalidAddress,
		// 	expectedError:        sdk.ErrSDKHandleRelay,
		// 	expectedErrorMessage: "getting app ring",
		// },
		// // TODO could not trigger error in GetSignableBytesHash
		// {
		// 	desc:          "Invalid relay request signable bytes hash",
		// 	SDKBehavior:   testsdk.NewTestBehavior(t),
		// 	inputScenario: callSendRelayWithInvalidSignableBytes,
		// 	expectedError: nil,
		// },
		// // TODO could not trigger error in signer.Sign
		// {
		// 	desc:          "Error signing relay request",
		// 	SDKBehavior:   testsdk.NewTestBehavior(t),
		// 	inputScenario: func(behavior *testsdk.TestBehavior) error { return nil },
		// 	expectedError: nil,
		// },
		// {
		// 	// TODO How to trigger an unmarshal error?
		// 	desc:          "Error marshaling relay request",
		// 	SDKBehavior:   testsdk.NewTestBehavior(t),
		// 	inputScenario: func(behavior *testsdk.TestBehavior) error { return nil },
		// 	expectedError: nil,
		// },
		{
			desc:          "Error sending request",
			SDKBehavior:   testsdk.NewTestBehavior(t),
			inputScenario: callSendRelayWith(&http.Request{Method: "POST"}),
			// inputScenario: func(behavior *testsdk.TestBehavior) error { return nil },
			expectedError: nil,
		},
		// {
		// 	desc:          "Error reading response body",
		// 	SDKBehavior:   testsdk.NewTestBehavior(t),
		// 	inputScenario: func(behavior *testsdk.TestBehavior) error { return nil },
		// 	expectedError: nil,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.expectedError == nil {
				require.NoError(t, tt.inputScenario(tt.SDKBehavior))
				return
			}

			err := tt.inputScenario(tt.SDKBehavior)
			require.ErrorIs(t, err, tt.expectedError)
			require.ErrorContains(t, err, tt.expectedErrorMessage)
		})
	}
}

func TestSDK_VerifyResponse(t *testing.T) {
	tests := []struct {
		desc          string
		SDKBehavior   *testsdk.TestBehavior
		inputScenario func(behavior *testsdk.TestBehavior) error
		expectedError error
	}{
		{
			desc:        "Error getting supplier public key",
			SDKBehavior: testsdk.NewTestBehavior(t),
			// inputScenario: callVerifyResponseWith(
			// 	testsdk.InvalidAppAddress,
			// 	testsdk.RelayResponse,
			// ),
			expectedError: nil,
		},
		{
			desc:          "Missing relay response meta",
			SDKBehavior:   testsdk.NewTestBehavior(t),
			inputScenario: func(behavior *testsdk.TestBehavior) error { return nil },
			expectedError: nil,
		},
		{
			desc:          "Missing relay response signature",
			SDKBehavior:   testsdk.NewTestBehavior(t),
			inputScenario: func(behavior *testsdk.TestBehavior) error { return nil },
			expectedError: nil,
		},
		{
			desc:          "Invalid signable bytes hash",
			SDKBehavior:   testsdk.NewTestBehavior(t),
			inputScenario: func(behavior *testsdk.TestBehavior) error { return nil },
			expectedError: nil,
		},
		{
			desc:          "Invalid signature",
			SDKBehavior:   testsdk.NewTestBehavior(t),
			inputScenario: func(behavior *testsdk.TestBehavior) error { return nil },
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.expectedError == nil {
				require.NoError(t, tt.inputScenario(tt.SDKBehavior))
				return
			}

			err := tt.inputScenario(tt.SDKBehavior)
			require.ErrorIs(t, err, tt.expectedError)
		})
	}
}

func TestSDK_SuccessfulRelay(t *testing.T) {
	//
}

func callGetSessionSupplierEndpointsWith(appAddress, serviceID string) func(*testsdk.TestBehavior) error {
	return func(testBehavior *testsdk.TestBehavior) error {
		testBehavior.SdkConfig.Deps = testBehavior.BuildDeps()
		sdk, err := sdk.NewPOKTRollSDK(testBehavior.Ctx, testBehavior.SdkConfig)
		require.NoError(testBehavior.T, err)

		_, err = sdk.GetSessionSupplierEndpoints(testBehavior.Ctx, appAddress, serviceID)

		return err
	}
}

func callSendRelayWith(request *http.Request) func(*testsdk.TestBehavior) error {
	return func(testBehavior *testsdk.TestBehavior) error {
		testBehavior.SdkConfig.Deps = testBehavior.BuildDeps()
		sdk, err := sdk.NewPOKTRollSDK(testBehavior.Ctx, testBehavior.SdkConfig)
		require.NoError(testBehavior.T, err)

		suppliers, err := sdk.GetSessionSupplierEndpoints(
			testBehavior.Ctx,
			testsdk.ValidAppAddress,
			testsdk.ValidServiceID,
		)
		require.NoError(testBehavior.T, err)

		request.Body = io.NopCloser(bytes.NewBuffer([]byte("asd")))
		_, err = sdk.SendRelay(
			testBehavior.Ctx,
			suppliers.SuppliersEndpoints[0],
			request,
		)
		return err
	}
}

// func callSendRelayWithInvalidBody(testBehavior *testsdk.TestBehavior) error {
// 	testBehavior.SdkConfig.Deps = testBehavior.BuildDeps()
// 	sdk, err := sdk.NewPOKTRollSDK(testBehavior.Ctx, testBehavior.SdkConfig)
// 	require.NoError(testBehavior.T, err)

// 	suppliers, err := sdk.GetSessionSupplierEndpoints(
// 		testBehavior.Ctx,
// 		testsdk.ValidAppAddress,
// 		testsdk.ValidServiceID,
// 	)
// 	require.NoError(testBehavior.T, err)

// 	requestWithInvalidBody := &http.Request{}

// 	// TODO find a better way to trigger an error in io.ReadAll
// 	file, _ := os.Open("nonexistent.txt")
// 	requestWithInvalidBody.Body = file
// 	_, err = sdk.SendRelay(
// 		testBehavior.Ctx,
// 		suppliers.SuppliersEndpoints[0],
// 		requestWithInvalidBody,
// 	)

// 	return err
// }

// func callSendRelayWithInvalidAddress(testBehavior *testsdk.TestBehavior) error {
// 	testBehavior.SdkConfig.Deps = testBehavior.BuildDeps()
// 	sdk, err := sdk.NewPOKTRollSDK(testBehavior.Ctx, testBehavior.SdkConfig)
// 	require.NoError(testBehavior.T, err)

// 	suppliers, err := sdk.GetSessionSupplierEndpoints(
// 		testBehavior.Ctx,
// 		testsdk.ValidAppAddress,
// 		testsdk.ValidServiceID,
// 	)
// 	require.NoError(testBehavior.T, err)

// 	suppliers.SuppliersEndpoints[0].Header.ApplicationAddress = "Asd"
// 	request, err := http.NewRequest("POST", "http://localhost", bytes.NewBuffer([]byte("data")))
// 	if err != nil {
// 		return err
// 	}
// 	_, err = sdk.SendRelay(
// 		testBehavior.Ctx,
// 		suppliers.SuppliersEndpoints[0],
// 		request,
// 	)

// 	return err
// }

// func callSendRelayWithInvalidSignableBytes(testBehavior *testsdk.TestBehavior) error {
// 	testBehavior.SdkConfig.Deps = testBehavior.BuildDeps()
// 	sdk, err := sdk.NewPOKTRollSDK(testBehavior.Ctx, testBehavior.SdkConfig)
// 	require.NoError(testBehavior.T, err)

// 	suppliers, err := sdk.GetSessionSupplierEndpoints(
// 		testBehavior.Ctx,
// 		testsdk.ValidAppAddress,
// 		testsdk.ValidServiceID,
// 	)
// 	require.NoError(testBehavior.T, err)

// 	suppliers.SuppliersEndpoints[0].Header.ApplicationAddress = testsdk.ValidAppAddress
// 	request, err := http.NewRequest("POST", "http://localhost", bytes.NewBuffer([]byte("{")))
// 	if err != nil {
// 		return err
// 	}
// 	_, err = sdk.SendRelay(
// 		testBehavior.Ctx,
// 		suppliers.SuppliersEndpoints[0],
// 		request,
// 	)

// 	return err
// }

func callVerifyResponseWith(supplierAddress string, relayResponse *types.RelayResponse) func(*testsdk.TestBehavior) error {
	return func(testBehavior *testsdk.TestBehavior) error {
		// testBehavior.SdkConfig.Deps = testBehavior.BuildDeps()
		// sdk, err := sdk.NewPOKTRollSDK(testBehavior.Ctx, testBehavior.SdkConfig)
		// require.NoError(testBehavior.T, err)

		// sdk := &poktrollSDK{}

		// sdk.verifyResponse()
		return nil
	}
}

func initializeSDK(testBehavior *testsdk.TestBehavior) error {
	testBehavior.SdkConfig.Deps = testBehavior.BuildDeps()
	_, err := sdk.NewPOKTRollSDK(testBehavior.Ctx, testBehavior.SdkConfig)
	return err
}
