package sdkadapter

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	shannonsdk "github.com/pokt-network/shannon-sdk"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/auth"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/session"
	"github.com/pokt-network/poktroll/proto/types/service"
)

// ShannonSDK is a wrapper around the Shannon SDK that is used by the AppGateServer
// to encapsulate the SDK's functionality and dependencies.
type ShannonSDK struct {
	blockClient   client.BlockClient
	sessionClient client.SessionQueryClient
	appClient     client.ApplicationQueryClient
	accountClient client.AccountQueryClient
	relayClient   *http.Client
	signer        *shannonsdk.Signer
}

// NewShannonSDK creates a new ShannonSDK instance with the given signing key and dependencies.
// It initializes the necessary clients and signer for the SDK.
func NewShannonSDK(
	ctx context.Context,
	signingKey cryptotypes.PrivKey,
	deps depinject.Config,
) (*ShannonSDK, error) {
	sessionClient, sessionClientErr := session.NewSessionQueryClient(deps)
	if sessionClientErr != nil {
		return nil, sessionClientErr
	}

	accountClient, accountClientErr := auth.NewAccountQuerier(deps)
	if accountClientErr != nil {
		return nil, accountClientErr
	}

	appClient, appClientErr := query.NewApplicationQuerier(deps)
	if appClientErr != nil {
		return nil, appClientErr
	}

	blockClient := client.BlockClient(nil)
	if depsErr := depinject.Inject(deps, &blockClient); depsErr != nil {
		return nil, depsErr
	}

	signer, signerErr := NewSigner(signingKey)
	if signerErr != nil {
		return nil, signerErr
	}

	shannonSDK := &ShannonSDK{
		blockClient:   blockClient,
		sessionClient: sessionClient,
		accountClient: accountClient,
		appClient:     appClient,
		relayClient:   http.DefaultClient,
		signer:        signer,
	}

	return shannonSDK, nil
}

// SendRelay builds a relay request from the given requestBz, signs it with the
// application address, then sends it to the given endpoint.
func (shannonSDK *ShannonSDK) SendRelay(
	ctx context.Context,
	appAddress string,
	endpoint shannonsdk.Endpoint,
	requestBz []byte,
) (*service.RelayResponse, error) {
	relayRequest, err := shannonsdk.BuildRelayRequest(endpoint, requestBz)
	if err != nil {
		return nil, err
	}

	application, err := shannonSDK.appClient.GetApplication(ctx, appAddress)
	if err != nil {
		return nil, err
	}

	appRing := shannonsdk.ApplicationRing{
		PublicKeyFetcher: shannonSDK.accountClient,
		Application:      application,
	}

	if _, err = shannonSDK.signer.Sign(ctx, relayRequest, appRing); err != nil {
		return nil, err
	}

	relayRequestBz, err := relayRequest.Marshal()
	if err != nil {
		return nil, err
	}

	response, err := shannonSDK.relayClient.Post(
		endpoint.Endpoint().Url,
		"application/json",
		bytes.NewReader(relayRequestBz),
	)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBodyBz, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return shannonsdk.ValidateRelayResponse(
		ctx,
		endpoint.Supplier(),
		responseBodyBz,
		shannonSDK.accountClient,
	)
}

// GetSessionSupplierEndpoints returns the current session's supplier endpoints
// for the given appAddress and serviceId.
func (shannonSDK *ShannonSDK) GetSessionSupplierEndpoints(
	ctx context.Context,
	appAddress, serviceId string,
) (*shannonsdk.SessionFilter, error) {
	currentHeight := shannonSDK.blockClient.LastBlock(ctx).Height()
	session, err := shannonSDK.sessionClient.GetSession(ctx, appAddress, serviceId, currentHeight)
	if err != nil {
		return nil, err
	}

	filteredSession := &shannonsdk.SessionFilter{
		Session: session,
	}

	return filteredSession, nil
}
