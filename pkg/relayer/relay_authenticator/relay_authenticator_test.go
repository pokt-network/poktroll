package relay_authenticator_test

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	keyringtypes "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer/relay_authenticator"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testdelegation"
	"github.com/pokt-network/poktroll/testutil/testclient/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	testrings "github.com/pokt-network/poktroll/testutil/testcrypto/rings"
	"github.com/pokt-network/poktroll/testutil/testproxy"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

const (
	blockHeight     = int64(1)
	serviceId       = "svc1"
	supplierKeyName = "supplier1"
)

type RelayAuthenticatorTestSuite struct {
	suite.Suite

	deps depinject.Config
	ctx  context.Context

	// Test dependencies
	logger         polylog.Logger
	keyring        keyringtypes.Keyring
	sessionQuerier client.SessionQueryClient
	sharedQuerier  client.SharedQueryClient
	blockClient    client.BlockClient
	ringCache      crypto.RingCache

	// Test data
	supplierKeyName string
	supplierAddress string
	appAddress      string
	appPrivKey      cryptotypes.PrivKey
	session         *sessiontypes.Session
}

// TODO_TECHDEBT: Move all signing and verification tests from proxy_test.go to relay_authenticator_test.go
func TestRelayAuthenticatorTestSuite(t *testing.T) {
	suite.Run(t, new(RelayAuthenticatorTestSuite))
}

func (s *RelayAuthenticatorTestSuite) SetupTest() {
	// Initialize context and logger
	s.ctx = context.Background()
	s.logger = polylog.Ctx(s.ctx)

	// Generate application account details
	s.setupApplicationAccount()

	// Set up supplier keyring
	s.setupSupplierKeyring()

	// Initialize query clients
	s.setupQueryClients()

	// Set up ring cache
	s.setupRingCache()

	// Set up session
	s.setupSession()

	// Prepare dependencies
	s.deps = depinject.Supply(
		s.logger,
		s.keyring,
		s.sessionQuerier,
		s.sharedQuerier,
		s.blockClient,
		s.ringCache,
	)
}

func (s *RelayAuthenticatorTestSuite) TestNewRelayAuthenticator_NoKeyName() {
	// Create authenticator with empty key names
	auth, err := relay_authenticator.NewRelayAuthenticator(
		s.deps,
		relay_authenticator.WithSigningKeyNames([]string{}),
	)

	// Expect error and nil authenticator
	require.Error(s.T(), err)
	require.Nil(s.T(), auth)
}

func (s *RelayAuthenticatorTestSuite) TestNewRelayAuthenticator_InvalidKeyName() {
	// Create authenticator with non-existent key name
	auth, err := relay_authenticator.NewRelayAuthenticator(
		s.deps,
		relay_authenticator.WithSigningKeyNames([]string{"non_existent_key"}),
	)

	// Expect error and nil authenticator
	require.Error(s.T(), err)
	require.Nil(s.T(), auth)
}

func (s *RelayAuthenticatorTestSuite) TestVerifyRelayRequest_Success() {
	// Create authenticator with valid key name
	auth, err := relay_authenticator.NewRelayAuthenticator(
		s.deps,
		relay_authenticator.WithSigningKeyNames([]string{s.supplierKeyName}),
	)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), auth)

	// Create a mock relay request
	relayReq := &servicetypes.RelayRequest{
		Meta: servicetypes.RelayRequestMetadata{
			SupplierOperatorAddress: s.supplierAddress,
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      s.appAddress,
				SessionId:               s.session.SessionId,
				SessionStartBlockHeight: s.session.Header.SessionStartBlockHeight,
				SessionEndBlockHeight:   s.session.Header.SessionEndBlockHeight,
				ServiceId:               serviceId,
			},
		},
	}
	relayReq.Meta.Signature = testproxy.GetApplicationRingSignature(s.T(), relayReq, s.appPrivKey)

	// Verify the relay request
	err = auth.VerifyRelayRequest(s.ctx, relayReq, serviceId)
	require.NoError(s.T(), err)
}

func (s *RelayAuthenticatorTestSuite) TestSignRelayResponse_Success() {
	// Create authenticator with valid key name
	auth, err := relay_authenticator.NewRelayAuthenticator(
		s.deps,
		relay_authenticator.WithSigningKeyNames([]string{s.supplierKeyName}),
	)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), auth)

	// Create a mock relay response
	relayRes := &servicetypes.RelayResponse{
		Meta: servicetypes.RelayResponseMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      s.appAddress,
				SessionId:               s.session.SessionId,
				SessionStartBlockHeight: s.session.Header.SessionStartBlockHeight,
				SessionEndBlockHeight:   s.session.Header.SessionEndBlockHeight,
				ServiceId:               serviceId,
			},
		},
	}

	// Sign the relay response
	err = auth.SignRelayResponse(relayRes, s.supplierAddress)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), relayRes.Meta.SupplierOperatorSignature)
}

// setupApplicationAccount generates and sets up the application account details
func (s *RelayAuthenticatorTestSuite) setupApplicationAccount() {
	appAddress, appPubKey, appPrivKey := sample.AccAddressAndKeyPair()
	s.appAddress = appAddress
	s.appPrivKey = appPrivKey

	delegateeAccounts := make(map[string]cryptotypes.PubKey, 0)
	testqueryclients.AddAddressToApplicationMap(s.T(), s.appAddress, appPubKey, delegateeAccounts)
}

// setupSupplierKeyring initializes the keyring with a supplier key
func (s *RelayAuthenticatorTestSuite) setupSupplierKeyring() {
	// Set up test key names
	s.supplierKeyName = supplierKeyName

	keyring, supplierKeyringRecord := testkeyring.NewTestKeyringWithKey(s.T(), supplierKeyName)
	s.keyring = keyring

	supplierOperatorAddress, err := supplierKeyringRecord.GetAddress()
	require.NoError(s.T(), err)
	s.supplierAddress = supplierOperatorAddress.String()
}

// setupQueryClients initializes all the necessary query clients
func (s *RelayAuthenticatorTestSuite) setupQueryClients() {
	s.sessionQuerier = testqueryclients.NewTestSessionQueryClient(s.T())
	s.sharedQuerier = testqueryclients.NewTestSharedQueryClient(s.T())
	s.blockClient = testblock.NewAnyTimeLastBlockBlockClient(s.T(), []byte{}, blockHeight)
}

// setupRingCache initializes the ring cache with its dependencies
func (s *RelayAuthenticatorTestSuite) setupRingCache() {
	redelegationObs, _ := channel.NewReplayObservable[*apptypes.EventRedelegation](s.ctx, 1)
	delegationClient := testdelegation.NewAnyTimesRedelegationsSequence(s.ctx, s.T(), "", redelegationObs)

	applicationQueryClient := testqueryclients.NewTestApplicationQueryClient(s.T())
	accountQueryClient := testqueryclients.NewTestAccountQueryClient(s.T())

	ringCacheDeps := depinject.Supply(
		accountQueryClient,
		applicationQueryClient,
		delegationClient,
		s.sharedQuerier,
	)
	s.ringCache = testrings.NewRingCacheWithMockDependencies(s.ctx, s.T(), ringCacheDeps)
}

// setupSession initializes a session with a supplier
func (s *RelayAuthenticatorTestSuite) setupSession() {
	testqueryclients.AddToExistingSessions(
		s.T(),
		s.appAddress,
		serviceId,
		blockHeight,
		[]string{s.supplierAddress},
	)

	session, err := s.sessionQuerier.GetSession(
		s.ctx,
		s.appAddress,
		serviceId,
		blockHeight,
	)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), session)

	s.session = session
}
