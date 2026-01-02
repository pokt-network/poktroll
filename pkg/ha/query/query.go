package query

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	// defaultQueryTimeout is the default timeout for chain queries.
	defaultQueryTimeout = 5 * time.Second
)

// QueryClientConfig contains configuration for query clients.
type QueryClientConfig struct {
	// GRPCEndpoint is the gRPC endpoint for the full node.
	// Example: "localhost:9090"
	GRPCEndpoint string

	// QueryTimeout is the timeout for chain queries.
	// Default: 5 seconds
	QueryTimeout time.Duration

	// UseTLS enables TLS for the gRPC connection.
	// Set to true when connecting to endpoints on port 443 or with TLS enabled.
	// Default: false (insecure connection)
	UseTLS bool
}

// QueryClients provides access to all on-chain query clients.
type QueryClients struct {
	logger polylog.Logger
	config QueryClientConfig

	// gRPC connection
	grpcConn *grpc.ClientConn

	// Individual query clients
	sharedClient      *sharedQueryClient
	sessionClient     *sessionQueryClient
	applicationClient *applicationQueryClient
	supplierClient    *supplierQueryClient
	proofClient       *proofQueryClient
	serviceClient     *serviceQueryClient

	// Lifecycle
	mu     sync.RWMutex
	closed bool
}

// NewQueryClients creates a new QueryClients instance.
func NewQueryClients(
	logger polylog.Logger,
	config QueryClientConfig,
) (*QueryClients, error) {
	if config.GRPCEndpoint == "" {
		return nil, fmt.Errorf("gRPC endpoint is required")
	}
	if config.QueryTimeout == 0 {
		config.QueryTimeout = defaultQueryTimeout
	}

	// Establish gRPC connection with appropriate credentials
	var transportCreds credentials.TransportCredentials
	if config.UseTLS {
		transportCreds = credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})
	} else {
		transportCreds = insecure.NewCredentials()
	}

	grpcConn, err := grpc.NewClient(
		config.GRPCEndpoint,
		grpc.WithTransportCredentials(transportCreds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	qc := &QueryClients{
		logger:   logging.ForComponent(logger, logging.ComponentQueryClients),
		config:   config,
		grpcConn: grpcConn,
	}

	// Initialize individual clients
	qc.sharedClient = newSharedQueryClient(logger, grpcConn, config.QueryTimeout)
	qc.sessionClient = newSessionQueryClient(logger, grpcConn, qc.sharedClient, config.QueryTimeout)
	qc.applicationClient = newApplicationQueryClient(logger, grpcConn, config.QueryTimeout)
	qc.supplierClient = newSupplierQueryClient(logger, grpcConn, config.QueryTimeout)
	qc.proofClient = newProofQueryClient(logger, grpcConn, config.QueryTimeout)
	qc.serviceClient = newServiceQueryClient(logger, grpcConn, config.QueryTimeout)

	qc.logger.Info().
		Str("endpoint", config.GRPCEndpoint).
		Msg("query clients initialized")

	return qc, nil
}

// Shared returns the shared module query client.
func (qc *QueryClients) Shared() client.SharedQueryClient {
	return qc.sharedClient
}

// Session returns the session module query client.
func (qc *QueryClients) Session() client.SessionQueryClient {
	return qc.sessionClient
}

// Application returns the application module query client.
func (qc *QueryClients) Application() client.ApplicationQueryClient {
	return qc.applicationClient
}

// Supplier returns the supplier module query client.
func (qc *QueryClients) Supplier() client.SupplierQueryClient {
	return qc.supplierClient
}

// Proof returns the proof module query client.
func (qc *QueryClients) Proof() client.ProofQueryClient {
	return qc.proofClient
}

// Service returns the service module query client.
func (qc *QueryClients) Service() client.ServiceQueryClient {
	return qc.serviceClient
}

// GRPCConnection returns the underlying gRPC connection.
// This allows sharing the connection with other clients (e.g., TxClient).
func (qc *QueryClients) GRPCConnection() *grpc.ClientConn {
	return qc.grpcConn
}

// Close closes all query clients and the underlying gRPC connection.
func (qc *QueryClients) Close() error {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if qc.closed {
		return nil
	}
	qc.closed = true

	if qc.grpcConn != nil {
		if err := qc.grpcConn.Close(); err != nil {
			return fmt.Errorf("failed to close gRPC connection: %w", err)
		}
	}

	qc.logger.Info().Msg("query clients closed")
	return nil
}

// =============================================================================
// Shared Query Client
// =============================================================================

type sharedQueryClient struct {
	logger       polylog.Logger
	queryClient  sharedtypes.QueryClient
	queryTimeout time.Duration

	// Simple in-memory cache for params
	paramsCache   *sharedtypes.Params
	paramsCacheMu sync.RWMutex
}

var _ client.SharedQueryClient = (*sharedQueryClient)(nil)

func newSharedQueryClient(logger polylog.Logger, conn *grpc.ClientConn, timeout time.Duration) *sharedQueryClient {
	return &sharedQueryClient{
		logger:       logger.With("query_client", "shared"),
		queryClient:  sharedtypes.NewQueryClient(conn),
		queryTimeout: timeout,
	}
}

func (c *sharedQueryClient) GetParams(ctx context.Context) (*sharedtypes.Params, error) {
	// Check cache first
	c.paramsCacheMu.RLock()
	if c.paramsCache != nil {
		cached := c.paramsCache
		c.paramsCacheMu.RUnlock()
		return cached, nil
	}
	c.paramsCacheMu.RUnlock()

	// Query chain
	c.paramsCacheMu.Lock()
	defer c.paramsCacheMu.Unlock()

	// Double-check after acquiring lock
	if c.paramsCache != nil {
		return c.paramsCache, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Params(queryCtx, &sharedtypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query shared params: %w", err)
	}

	c.paramsCache = &res.Params
	return &res.Params, nil
}

func (c *sharedQueryClient) GetSessionGracePeriodEndHeight(ctx context.Context, queryHeight int64) (int64, error) {
	params, err := c.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetSessionGracePeriodEndHeight(params, queryHeight), nil
}

func (c *sharedQueryClient) GetClaimWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	params, err := c.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetClaimWindowOpenHeight(params, queryHeight), nil
}

func (c *sharedQueryClient) GetEarliestSupplierClaimCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	params, err := c.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	// TODO: For now, use a simple calculation without block hash
	// In production, we should query the block hash at claim window open height
	// claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(params, queryHeight)
	return sharedtypes.GetEarliestSupplierClaimCommitHeight(
		params,
		queryHeight,
		nil, // Block hash - simplified for now
		supplierOperatorAddr,
	), nil
}

func (c *sharedQueryClient) GetProofWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	params, err := c.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetProofWindowOpenHeight(params, queryHeight), nil
}

func (c *sharedQueryClient) GetEarliestSupplierProofCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	params, err := c.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	// TODO: For now, use a simple calculation without block hash
	// In production, we should query the block hash at proof window open height
	return sharedtypes.GetEarliestSupplierProofCommitHeight(
		params,
		queryHeight,
		nil, // Block hash - simplified for now
		supplierOperatorAddr,
	), nil
}

// InvalidateCache clears the cached params.
func (c *sharedQueryClient) InvalidateCache() {
	c.paramsCacheMu.Lock()
	c.paramsCache = nil
	c.paramsCacheMu.Unlock()
}

// =============================================================================
// Session Query Client
// =============================================================================

type sessionQueryClient struct {
	logger       polylog.Logger
	queryClient  sessiontypes.QueryClient
	sharedClient *sharedQueryClient
	queryTimeout time.Duration

	// Simple in-memory cache for sessions
	sessionCache   map[string]*sessiontypes.Session
	sessionCacheMu sync.RWMutex

	// Params cache
	paramsCache   *sessiontypes.Params
	paramsCacheMu sync.RWMutex
}

var _ client.SessionQueryClient = (*sessionQueryClient)(nil)

func newSessionQueryClient(
	logger polylog.Logger,
	conn *grpc.ClientConn,
	sharedClient *sharedQueryClient,
	timeout time.Duration,
) *sessionQueryClient {
	return &sessionQueryClient{
		logger:       logger.With("query_client", "session"),
		queryClient:  sessiontypes.NewQueryClient(conn),
		sharedClient: sharedClient,
		queryTimeout: timeout,
		sessionCache: make(map[string]*sessiontypes.Session),
	}
}

func (c *sessionQueryClient) GetSession(
	ctx context.Context,
	appAddress string,
	serviceId string,
	blockHeight int64,
) (*sessiontypes.Session, error) {
	// Get shared params for cache key calculation
	sharedParams, err := c.sharedClient.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate session start height for consistent caching
	sessionStartHeight := sharedtypes.GetSessionStartHeight(sharedParams, blockHeight)
	cacheKey := fmt.Sprintf("%s/%s/%d", appAddress, serviceId, sessionStartHeight)

	// Check cache
	c.sessionCacheMu.RLock()
	if session, ok := c.sessionCache[cacheKey]; ok {
		c.sessionCacheMu.RUnlock()
		return session, nil
	}
	c.sessionCacheMu.RUnlock()

	// Query chain
	c.sessionCacheMu.Lock()
	defer c.sessionCacheMu.Unlock()

	// Double-check after acquiring lock
	if session, ok := c.sessionCache[cacheKey]; ok {
		return session, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.GetSession(queryCtx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          serviceId,
		BlockHeight:        blockHeight,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	c.sessionCache[cacheKey] = res.Session
	return res.Session, nil
}

func (c *sessionQueryClient) GetParams(ctx context.Context) (*sessiontypes.Params, error) {
	// Check cache first
	c.paramsCacheMu.RLock()
	if c.paramsCache != nil {
		cached := c.paramsCache
		c.paramsCacheMu.RUnlock()
		return cached, nil
	}
	c.paramsCacheMu.RUnlock()

	// Query chain
	c.paramsCacheMu.Lock()
	defer c.paramsCacheMu.Unlock()

	// Double-check after acquiring lock
	if c.paramsCache != nil {
		return c.paramsCache, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Params(queryCtx, &sessiontypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query session params: %w", err)
	}

	c.paramsCache = &res.Params
	return &res.Params, nil
}

// InvalidateCache clears all cached data.
func (c *sessionQueryClient) InvalidateCache() {
	c.sessionCacheMu.Lock()
	c.sessionCache = make(map[string]*sessiontypes.Session)
	c.sessionCacheMu.Unlock()

	c.paramsCacheMu.Lock()
	c.paramsCache = nil
	c.paramsCacheMu.Unlock()
}

// =============================================================================
// Application Query Client
// =============================================================================

type applicationQueryClient struct {
	logger       polylog.Logger
	queryClient  apptypes.QueryClient
	queryTimeout time.Duration

	// Simple in-memory cache
	appCache   map[string]apptypes.Application
	appCacheMu sync.RWMutex

	paramsCache   *apptypes.Params
	paramsCacheMu sync.RWMutex
}

var _ client.ApplicationQueryClient = (*applicationQueryClient)(nil)

func newApplicationQueryClient(logger polylog.Logger, conn *grpc.ClientConn, timeout time.Duration) *applicationQueryClient {
	return &applicationQueryClient{
		logger:       logger.With("query_client", "application"),
		queryClient:  apptypes.NewQueryClient(conn),
		queryTimeout: timeout,
		appCache:     make(map[string]apptypes.Application),
	}
}

func (c *applicationQueryClient) GetApplication(ctx context.Context, appAddress string) (apptypes.Application, error) {
	// Check cache
	c.appCacheMu.RLock()
	if app, ok := c.appCache[appAddress]; ok {
		c.appCacheMu.RUnlock()
		return app, nil
	}
	c.appCacheMu.RUnlock()

	// Query chain
	c.appCacheMu.Lock()
	defer c.appCacheMu.Unlock()

	// Double-check after acquiring lock
	if app, ok := c.appCache[appAddress]; ok {
		return app, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Application(queryCtx, &apptypes.QueryGetApplicationRequest{
		Address: appAddress,
	})
	if err != nil {
		return apptypes.Application{}, fmt.Errorf("failed to query application: %w", err)
	}

	c.appCache[appAddress] = res.Application
	return res.Application, nil
}

func (c *applicationQueryClient) GetAllApplications(ctx context.Context) ([]apptypes.Application, error) {
	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.AllApplications(queryCtx, &apptypes.QueryAllApplicationsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query all applications: %w", err)
	}

	return res.Applications, nil
}

func (c *applicationQueryClient) GetParams(ctx context.Context) (*apptypes.Params, error) {
	// Check cache first
	c.paramsCacheMu.RLock()
	if c.paramsCache != nil {
		cached := c.paramsCache
		c.paramsCacheMu.RUnlock()
		return cached, nil
	}
	c.paramsCacheMu.RUnlock()

	// Query chain
	c.paramsCacheMu.Lock()
	defer c.paramsCacheMu.Unlock()

	if c.paramsCache != nil {
		return c.paramsCache, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Params(queryCtx, &apptypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query application params: %w", err)
	}

	c.paramsCache = &res.Params
	return &res.Params, nil
}

// =============================================================================
// Supplier Query Client
// =============================================================================

type supplierQueryClient struct {
	logger       polylog.Logger
	queryClient  suppliertypes.QueryClient
	queryTimeout time.Duration

	// Simple in-memory cache
	supplierCache   map[string]sharedtypes.Supplier
	supplierCacheMu sync.RWMutex

	paramsCache   *suppliertypes.Params
	paramsCacheMu sync.RWMutex
}

var _ client.SupplierQueryClient = (*supplierQueryClient)(nil)

func newSupplierQueryClient(logger polylog.Logger, conn *grpc.ClientConn, timeout time.Duration) *supplierQueryClient {
	return &supplierQueryClient{
		logger:        logger.With("query_client", "supplier"),
		queryClient:   suppliertypes.NewQueryClient(conn),
		queryTimeout:  timeout,
		supplierCache: make(map[string]sharedtypes.Supplier),
	}
}

func (c *supplierQueryClient) GetSupplier(ctx context.Context, supplierOperatorAddress string) (sharedtypes.Supplier, error) {
	// Check cache
	c.supplierCacheMu.RLock()
	if supplier, ok := c.supplierCache[supplierOperatorAddress]; ok {
		c.supplierCacheMu.RUnlock()
		return supplier, nil
	}
	c.supplierCacheMu.RUnlock()

	// Query chain
	c.supplierCacheMu.Lock()
	defer c.supplierCacheMu.Unlock()

	// Double-check after acquiring lock
	if supplier, ok := c.supplierCache[supplierOperatorAddress]; ok {
		return supplier, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Supplier(queryCtx, &suppliertypes.QueryGetSupplierRequest{
		OperatorAddress: supplierOperatorAddress,
	})
	if err != nil {
		return sharedtypes.Supplier{}, fmt.Errorf("failed to query supplier: %w", err)
	}

	c.supplierCache[supplierOperatorAddress] = res.Supplier
	return res.Supplier, nil
}

func (c *supplierQueryClient) GetParams(ctx context.Context) (*suppliertypes.Params, error) {
	// Check cache first
	c.paramsCacheMu.RLock()
	if c.paramsCache != nil {
		cached := c.paramsCache
		c.paramsCacheMu.RUnlock()
		return cached, nil
	}
	c.paramsCacheMu.RUnlock()

	// Query chain
	c.paramsCacheMu.Lock()
	defer c.paramsCacheMu.Unlock()

	if c.paramsCache != nil {
		return c.paramsCache, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Params(queryCtx, &suppliertypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query supplier params: %w", err)
	}

	c.paramsCache = &res.Params
	return &res.Params, nil
}

// =============================================================================
// Proof Query Client
// =============================================================================

type proofQueryClient struct {
	logger       polylog.Logger
	queryClient  prooftypes.QueryClient
	queryTimeout time.Duration

	// Simple in-memory cache
	claimCache   map[string]*prooftypes.Claim
	claimCacheMu sync.RWMutex

	paramsCache   *prooftypes.Params
	paramsCacheMu sync.RWMutex
}

var _ client.ProofQueryClient = (*proofQueryClient)(nil)

func newProofQueryClient(logger polylog.Logger, conn *grpc.ClientConn, timeout time.Duration) *proofQueryClient {
	return &proofQueryClient{
		logger:       logger.With("query_client", "proof"),
		queryClient:  prooftypes.NewQueryClient(conn),
		queryTimeout: timeout,
		claimCache:   make(map[string]*prooftypes.Claim),
	}
}

func (c *proofQueryClient) GetParams(ctx context.Context) (client.ProofParams, error) {
	// Check cache first
	c.paramsCacheMu.RLock()
	if c.paramsCache != nil {
		cached := c.paramsCache
		c.paramsCacheMu.RUnlock()
		return cached, nil
	}
	c.paramsCacheMu.RUnlock()

	// Query chain
	c.paramsCacheMu.Lock()
	defer c.paramsCacheMu.Unlock()

	if c.paramsCache != nil {
		return c.paramsCache, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Params(queryCtx, &prooftypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query proof params: %w", err)
	}

	c.paramsCache = &res.Params
	return &res.Params, nil
}

func (c *proofQueryClient) GetClaim(ctx context.Context, supplierOperatorAddress string, sessionId string) (client.Claim, error) {
	cacheKey := fmt.Sprintf("%s/%s", supplierOperatorAddress, sessionId)

	// Check cache
	c.claimCacheMu.RLock()
	if claim, ok := c.claimCache[cacheKey]; ok {
		c.claimCacheMu.RUnlock()
		return claim, nil
	}
	c.claimCacheMu.RUnlock()

	// Query chain
	c.claimCacheMu.Lock()
	defer c.claimCacheMu.Unlock()

	// Double-check after acquiring lock
	if claim, ok := c.claimCache[cacheKey]; ok {
		return claim, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Claim(queryCtx, &prooftypes.QueryGetClaimRequest{
		SupplierOperatorAddress: supplierOperatorAddress,
		SessionId:               sessionId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query claim: %w", err)
	}

	c.claimCache[cacheKey] = &res.Claim
	return &res.Claim, nil
}

// =============================================================================
// Service Query Client
// =============================================================================

type serviceQueryClient struct {
	logger       polylog.Logger
	queryClient  servicetypes.QueryClient
	queryTimeout time.Duration

	// Simple in-memory cache
	serviceCache   map[string]sharedtypes.Service
	serviceCacheMu sync.RWMutex

	difficultyCache   map[string]servicetypes.RelayMiningDifficulty
	difficultyCacheMu sync.RWMutex

	paramsCache   *servicetypes.Params
	paramsCacheMu sync.RWMutex
}

var _ client.ServiceQueryClient = (*serviceQueryClient)(nil)

func newServiceQueryClient(logger polylog.Logger, conn *grpc.ClientConn, timeout time.Duration) *serviceQueryClient {
	return &serviceQueryClient{
		logger:          logger.With("query_client", "service"),
		queryClient:     servicetypes.NewQueryClient(conn),
		queryTimeout:    timeout,
		serviceCache:    make(map[string]sharedtypes.Service),
		difficultyCache: make(map[string]servicetypes.RelayMiningDifficulty),
	}
}

func (c *serviceQueryClient) GetService(ctx context.Context, serviceId string) (sharedtypes.Service, error) {
	// Check cache
	c.serviceCacheMu.RLock()
	if service, ok := c.serviceCache[serviceId]; ok {
		c.serviceCacheMu.RUnlock()
		return service, nil
	}
	c.serviceCacheMu.RUnlock()

	// Query chain
	c.serviceCacheMu.Lock()
	defer c.serviceCacheMu.Unlock()

	// Double-check after acquiring lock
	if service, ok := c.serviceCache[serviceId]; ok {
		return service, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Service(queryCtx, &servicetypes.QueryGetServiceRequest{
		Id: serviceId,
	})
	if err != nil {
		return sharedtypes.Service{}, fmt.Errorf("failed to query service: %w", err)
	}

	c.serviceCache[serviceId] = res.Service
	return res.Service, nil
}

func (c *serviceQueryClient) GetServiceRelayDifficulty(ctx context.Context, serviceId string) (servicetypes.RelayMiningDifficulty, error) {
	// Check cache
	c.difficultyCacheMu.RLock()
	if difficulty, ok := c.difficultyCache[serviceId]; ok {
		c.difficultyCacheMu.RUnlock()
		return difficulty, nil
	}
	c.difficultyCacheMu.RUnlock()

	// Query chain
	c.difficultyCacheMu.Lock()
	defer c.difficultyCacheMu.Unlock()

	// Double-check after acquiring lock
	if difficulty, ok := c.difficultyCache[serviceId]; ok {
		return difficulty, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.RelayMiningDifficulty(queryCtx, &servicetypes.QueryGetRelayMiningDifficultyRequest{
		ServiceId: serviceId,
	})
	if err != nil {
		return servicetypes.RelayMiningDifficulty{}, fmt.Errorf("failed to query relay mining difficulty: %w", err)
	}

	c.difficultyCache[serviceId] = res.RelayMiningDifficulty
	return res.RelayMiningDifficulty, nil
}

func (c *serviceQueryClient) GetParams(ctx context.Context) (*servicetypes.Params, error) {
	// Check cache first
	c.paramsCacheMu.RLock()
	if c.paramsCache != nil {
		cached := c.paramsCache
		c.paramsCacheMu.RUnlock()
		return cached, nil
	}
	c.paramsCacheMu.RUnlock()

	// Query chain
	c.paramsCacheMu.Lock()
	defer c.paramsCacheMu.Unlock()

	if c.paramsCache != nil {
		return c.paramsCache, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	res, err := c.queryClient.Params(queryCtx, &servicetypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query service params: %w", err)
	}

	c.paramsCache = &res.Params
	return &res.Params, nil
}

// =============================================================================
// Block Query Client (using CometBFT RPC)
// =============================================================================

// BlockQueryClient provides block queries using CometBFT RPC.
type BlockQueryClient interface {
	client.BlockQueryClient
	Close() error
}

type blockQueryClient struct {
	logger       polylog.Logger
	rpcEndpoint  string
	queryTimeout time.Duration
}

// NewBlockQueryClient creates a new block query client.
// Note: This requires a CometBFT RPC endpoint, not gRPC.
func NewBlockQueryClient(
	logger polylog.Logger,
	rpcEndpoint string,
	queryTimeout time.Duration,
) BlockQueryClient {
	if queryTimeout == 0 {
		queryTimeout = defaultQueryTimeout
	}
	return &blockQueryClient{
		logger:       logger.With("query_client", "block"),
		rpcEndpoint:  rpcEndpoint,
		queryTimeout: queryTimeout,
	}
}

func (c *blockQueryClient) Block(ctx context.Context, height *int64) (*cometrpctypes.ResultBlock, error) {
	// This is a simplified implementation
	// For full functionality, use the cometbft/rpc/client package
	return nil, fmt.Errorf("block query requires CometBFT RPC client - use pkg/client/block for full implementation")
}

func (c *blockQueryClient) Close() error {
	return nil
}
