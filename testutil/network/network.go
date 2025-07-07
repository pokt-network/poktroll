package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	appmodule "github.com/pokt-network/poktroll/x/application/module"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

type (
	Network = network.Network
	Config  = network.Config
)

var (
	addrCodec = addresscodec.NewBech32Codec(app.AccountAddressPrefix)
	// Global mutex to ensure only one network test runs at a time within a single process
	networkMutex sync.Mutex
)

func init() {
	cmd.InitSDKConfig()
}

// acquireGlobalTestLock creates a file-based lock to coordinate tests across processes
func acquireGlobalTestLock(t *testing.T) *os.File {
	tempDir := os.TempDir()
	lockPath := filepath.Join(tempDir, "poktroll_network_test.lock")

	// Try to create/open the lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0666)
	require.NoError(t, err, "Failed to create global test lock file")

	// Acquire exclusive lock (will block until available)
	for {
		err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// Successfully acquired lock
			break
		}
		if err == syscall.EWOULDBLOCK {
			// Lock held by another process, wait and retry
			time.Sleep(100 * time.Millisecond)
			continue
		}
		// Some other error
		lockFile.Close()
		require.NoError(t, err, "Failed to acquire global test lock")
	}

	return lockFile
}

// releaseGlobalTestLock releases the file-based lock
func releaseGlobalTestLock(lockFile *os.File) {
	if lockFile != nil {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
	}
}

// New creates instance with fully configured cosmos network.
// Accepts optional config, that will be used in place of the DefaultConfig() if provided.
// Uses both in-process mutex and cross-process file lock to prevent all race conditions.
func New(t *testing.T, configs ...Config) *Network {
	t.Helper()

	// First acquire in-process mutex
	networkMutex.Lock()
	defer networkMutex.Unlock()

	// Then acquire cross-process file lock
	lockFile := acquireGlobalTestLock(t)

	// Lock will be released in the custom cleanup function below

	if len(configs) > 1 {
		panic("at most one config should be provided")
	}
	var cfg network.Config
	if len(configs) == 0 {
		cfg = DefaultConfig()
	} else {
		cfg = configs[0]
	}
	net, err := network.New(t, t.TempDir(), cfg)
	require.NoError(t, err, "TODO_FLAKY: This config setup is periodically flaky")
	_, err = net.WaitForHeight(1)
	require.NoError(t, err)

	// Custom cleanup with enhanced CometBFT shutdown handling
	t.Cleanup(func() {
		// Record initial goroutine count
		initialGoroutines := runtime.NumGoroutine()

		// Step 1: Give consensus reactors time to finish current operations
		// This helps prevent the panic where consensus reactor tries to access closed leveldb
		time.Sleep(1 * time.Second)

		// Step 2: Call standard cleanup
		net.Cleanup()

		// Step 3: Wait for goroutines to finish with extended timeout
		maxWait := 10 * time.Second
		start := time.Now()
		for time.Since(start) < maxWait {
			currentGoroutines := runtime.NumGoroutine()
			// If goroutine count has stabilized (reduced to 1/3 or less), we can proceed
			if currentGoroutines <= initialGoroutines/3 || currentGoroutines < 50 {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		// Step 4: Final safety delay to ensure leveldb is fully closed
		time.Sleep(2 * time.Second)
		
		// Step 5: Release the global lock
		releaseGlobalTestLock(lockFile)
	})

	return net
}

// DefaultConfig will initialize config for the network with custom application,
// genesis and single validator. All other parameters are inherited from cosmos-sdk/testutil/network.DefaultConfig
func DefaultConfig() network.Config {
	cfg, err := network.DefaultConfigWithAppConfig(app.AppConfig())
	if err != nil {
		panic(err)
	}
	ports, err := freePorts(3)
	if err != nil {
		panic(err)
	}
	if cfg.APIAddress == "" {
		cfg.APIAddress = fmt.Sprintf("tcp://0.0.0.0:%s", ports[0])
	}
	if cfg.RPCAddress == "" {
		cfg.RPCAddress = fmt.Sprintf("tcp://0.0.0.0:%s", ports[1])
	}
	if cfg.GRPCAddress == "" {
		cfg.GRPCAddress = fmt.Sprintf("0.0.0.0:%s", ports[2])
	}
	return cfg
}

// TODO_CLEANUP: Refactor the genesis state helpers below to consolidate usage
// and reduce the code footprint.

// ApplicationModuleGenesisStateWithAccount generates a GenesisState object with
// a single application for each of the given addresses.
func ApplicationModuleGenesisStateWithAddresses(t *testing.T, addresses []string) *apptypes.GenesisState {
	t.Helper()
	state := apptypes.DefaultGenesis()
	for _, addr := range addresses {
		application := apptypes.Application{
			Address: addr,
			Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(10000)},
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
				{
					ServiceId: "svc1",
				},
			},
		}
		state.ApplicationList = append(state.ApplicationList, application)
	}

	return state
}

// DefaultApplicationModuleGenesisState generates a GenesisState object with a given number of applications.
// It returns the populated GenesisState object.
func DefaultApplicationModuleGenesisState(t *testing.T, n int) *apptypes.GenesisState {
	t.Helper()
	state := apptypes.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", math.NewInt(int64(i+1)))
		application := apptypes.Application{
			Address: sample.AccAddress(),
			Stake:   &stake,
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
				{
					ServiceId: fmt.Sprintf("svc%d", i),
				},
				{
					ServiceId: fmt.Sprintf("svc%d%d", i, i),
				},
			},
			PendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{},
		}
		// TODO_CONSIDERATION: Evaluate whether we need `nullify.Fill` or if we should enforce `(gogoproto.nullable) = false` everywhere
		// nullify.Fill(&application)
		state.ApplicationList = append(state.ApplicationList, application)
	}
	return state
}

// DefaultSupplierModuleGenesisState generates a GenesisState object with a given number of suppliers.
// It returns the populated GenesisState object.
func DefaultSupplierModuleGenesisState(t *testing.T, n int) *suppliertypes.GenesisState {
	t.Helper()
	state := suppliertypes.DefaultGenesis()
	for i := 0; i < n; i++ {
		svcId := fmt.Sprintf("svc%d", i)
		stake := sdk.NewCoin("upokt", math.NewInt(int64(i)))
		services := []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: svcId,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     fmt.Sprintf("http://localhost:%d", i),
						RpcType: sharedtypes.RPCType_JSON_RPC,
					},
				},
			},
		}
		operatorAddr := sample.AccAddress()
		serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(operatorAddr, services, 1, 0)
		supplier := sharedtypes.Supplier{
			OwnerAddress:         sample.AccAddress(),
			OperatorAddress:      operatorAddr,
			Stake:                &stake,
			Services:             services,
			ServiceConfigHistory: serviceConfigHistory,
		}
		// TODO_CONSIDERATION: Evaluate whether we need `nullify.Fill` or if we should enforce `(gogoproto.nullable) = false` everywhere
		// nullify.Fill(&supplier)
		state.SupplierList = append(state.SupplierList, supplier)
	}
	return state
}

// SupplierModuleGenesisStateWithAddresses generates a GenesisState object with
// a single supplier for each of the given addresses.
func SupplierModuleGenesisStateWithAddresses(t *testing.T, addresses []string) *suppliertypes.GenesisState {
	t.Helper()
	state := suppliertypes.DefaultGenesis()
	services := []*sharedtypes.SupplierServiceConfig{
		{
			ServiceId: "svc1",
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     "http://localhost:1",
					RpcType: sharedtypes.RPCType_JSON_RPC,
				},
			},
		},
	}
	for _, addr := range addresses {
		serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(addr, services, 1, 0)
		supplier := sharedtypes.Supplier{
			OwnerAddress:         sample.AccAddress(),
			OperatorAddress:      addr,
			Stake:                &sdk.Coin{Denom: "upokt", Amount: math.NewInt(10000)},
			Services:             services,
			ServiceConfigHistory: serviceConfigHistory,
		}
		state.SupplierList = append(state.SupplierList, supplier)
	}
	return state
}

func DefaultTokenomicsModuleGenesisState(t *testing.T) *tokenomicstypes.GenesisState {
	t.Helper()
	state := tokenomicstypes.DefaultGenesis()
	return state
}

// DefaultGatewayModuleGenesisState generates a GenesisState object with a given
// number of gateways. It returns the populated GenesisState object.
func DefaultGatewayModuleGenesisState(t *testing.T, n int) *gatewaytypes.GenesisState {
	t.Helper()
	state := gatewaytypes.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", math.NewInt(int64(i)))
		gateway := gatewaytypes.Gateway{
			Address: sample.AccAddress(),
			Stake:   &stake,
		}
		// TODO_CONSIDERATION: Evaluate whether we need `nullify.Fill` or if we should enforce `(gogoproto.nullable) = false` everywhere
		// nullify.Fill(&gateway)
		state.GatewayList = append(state.GatewayList, gateway)
	}
	return state
}

// GatewayModuleGenesisStateWithAddresses generates a GenesisState object with
// a gateway list full of gateways with the given addresses.
// It returns the populated GenesisState object.
func GatewayModuleGenesisStateWithAddresses(t *testing.T, addresses []string) *gatewaytypes.GenesisState {
	t.Helper()
	state := gatewaytypes.DefaultGenesis()
	for _, addr := range addresses {
		gateway := gatewaytypes.Gateway{
			Address: addr,
			Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(10000)},
		}
		state.GatewayList = append(state.GatewayList, gateway)
	}
	return state
}

// ProofModuleGenesisStateWithClaims generates a GenesisState object with the
// given claims. It returns the populated GenesisState object.
func ProofModuleGenesisStateWithClaims(t *testing.T, claims []prooftypes.Claim) *prooftypes.GenesisState {
	t.Helper()

	state := prooftypes.DefaultGenesis()
	state.ClaimList = claims

	return state
}

// TODO_CLEANUP: Consolidate all of the helpers below to use shared business
// logic and move into its own helpers file.

// InitAccount initializes an Account by sending it some funds from the validator
// in the network to the address provided
func InitAccount(t *testing.T, net *Network, addr sdk.AccAddress) {
	t.Helper()
	val := net.Validators[0]
	ctx := val.ClientCtx
	args := []string{
		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
		fmt.Sprintf("--%s=%t", flags.FlagUnordered, true),
		fmt.Sprintf("--%s=%s", flags.TimeoutDuration, 5*time.Second),
	}
	amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(200)))
	responseRaw, err := clitestutil.MsgSendExec(ctx, val.Address, addr, amount, addrCodec, args...)
	require.NoError(t, err)
	var responseJSON map[string]interface{}
	err = json.Unmarshal(responseRaw.Bytes(), &responseJSON)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJSON["code"], "code is not 0 in the response: %v", responseJSON)
}

// InitAccountWithSequence initializes an Account by sending it some funds from
// the validator in the network to the address provided
func InitAccountWithSequence(
	t *testing.T,
	net *Network,
	addr sdk.AccAddress,
	signatureSequencerNumber int,
) {
	t.Helper()
	val := net.Validators[0]
	signerAccountNumber := 0
	ctx := val.ClientCtx
	args := []string{
		fmt.Sprintf("--%s=true", flags.FlagOffline),
		fmt.Sprintf("--%s=%d", flags.FlagAccountNumber, signerAccountNumber),
		fmt.Sprintf("--%s=%d", flags.FlagSequence, signatureSequencerNumber),

		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}
	amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(200)))
	responseRaw, err := clitestutil.MsgSendExec(ctx, val.Address, addr, amount, addrCodec, args...)
	require.NoError(t, err)
	var responseJSON map[string]interface{}
	err = json.Unmarshal(responseRaw.Bytes(), &responseJSON)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJSON["code"], "code is not 0 in the response: %v", responseJSON)
}

// DelegateAppToGateway delegates the provided application to the provided gateway
func DelegateAppToGateway(
	t *testing.T,
	net *Network,
	appAddr string,
	gatewayAddr string,
) {
	t.Helper()
	val := net.Validators[0]
	ctx := val.ClientCtx
	args := []string{
		gatewayAddr,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, appAddr),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}
	responseRaw, err := clitestutil.ExecTestCLICmd(ctx, appmodule.CmdDelegateToGateway(), args)
	require.NoError(t, err)
	var resp sdk.TxResponse
	require.NoError(t, net.Config.Codec.UnmarshalJSON(responseRaw.Bytes(), &resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.TxHash)
	require.Equal(t, uint32(0), resp.Code)
}

// UndelegateAppFromGateway undelegates the provided application from the provided gateway
func UndelegateAppFromGateway(
	t *testing.T,
	net *Network,
	appAddr string,
	gatewayAddr string,
) {
	t.Helper()
	val := net.Validators[0]
	ctx := val.ClientCtx
	args := []string{
		gatewayAddr,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, appAddr),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}
	responseRaw, err := clitestutil.ExecTestCLICmd(ctx, appmodule.CmdUndelegateFromGateway(), args)
	require.NoError(t, err)
	var resp sdk.TxResponse
	require.NoError(t, net.Config.Codec.UnmarshalJSON(responseRaw.Bytes(), &resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.TxHash)
	require.Equal(t, uint32(0), resp.Code)
}

// freePorts return the available ports based on the number of requested ports.
func freePorts(n int) ([]string, error) {
	closeFns := make([]func() error, n)
	ports := make([]string, n)
	for i := 0; i < n; i++ {
		_, port, closeFn, err := network.FreeTCPAddr()
		if err != nil {
			return nil, err
		}
		ports[i] = port
		closeFns[i] = closeFn
	}
	for _, closeFn := range closeFns {
		if err := closeFn(); err != nil {
			return nil, err
		}
	}
	return ports, nil
}

// TODO_TECHDEBT: Reuse this helper in all test helpers where appropriate.
func NewBondDenomCoins(t *testing.T, net *network.Network, numCoins int64) sdk.Coins {
	t.Helper()

	return sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(numCoins)))
}
