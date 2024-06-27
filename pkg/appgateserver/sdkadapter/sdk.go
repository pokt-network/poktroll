package sdkadapter

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/pokt-network/shannon-sdk"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/x/service/types"
)

type ShannonSDK struct {
	blockClient   client.BlockClient
	sessionClient client.SessionQueryClient
	appClient     client.ApplicationQueryClient
	accountClient client.AccountQueryClient
	relayClient   *http.Client
	signer        *sdk.Signer
}

// NewShannonSDK creates a new ShannonSDK instance with the given signing key and dependencies.
// It initializes the necessary clients and signer for the SDK.
func NewShannonSDK(
	ctx context.Context,
	signingKey cryptotypes.PrivKey,
	deps depinject.Config,
) (*ShannonSDK, error) {
	sessionClient, err := query.NewSessionQuerier(deps)
	if err != nil {
		return nil, err
	}

	accountClient, err := query.NewAccountQuerier(deps)
	if err != nil {
		return nil, err
	}

	appClient, err := query.NewApplicationQuerier(deps)
	if err != nil {
		return nil, err
	}

	blockClient, err := NewBlockClient(deps)
	if err != nil {
		return nil, err
	}

	signer, err := NewSigner(signingKey)
	if err != nil {
		return nil, err
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
	endpoint sdk.Endpoint,
	requestBz []byte,
) (*types.RelayResponse, error) {
	relayRequest, err := sdk.BuildRelayRequest(endpoint, requestBz)
	if err != nil {
		return nil, err
	}

	application, err := shannonSDK.appClient.GetApplication(ctx, appAddress)
	if err != nil {
		return nil, err
	}

	appRing := sdk.ApplicationRing{
		PublicKeyFetcher: shannonSDK.accountClient,
		Application:      application,
	}

	shannonSDK.signer.Sign(ctx, relayRequest, appRing)

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

	return sdk.ValidateRelayResponse(
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
) (*sdk.FilteredSession, error) {
	currentHeight := shannonSDK.blockClient.LastBlock(ctx).Height()
	session, err := shannonSDK.sessionClient.GetSession(ctx, appAddress, serviceId, currentHeight)
	if err != nil {
		return nil, err
	}

	filteredSession := &sdk.FilteredSession{
		Session: session,
	}

	return filteredSession, nil
}
