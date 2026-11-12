//go:build e2e

package e2e

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"strings"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/stretchr/testify/require"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TheUserCreatesAServiceWithNameAndComputeUnitsFromAccountWithMetadataFromFile creates a service with metadata from a file
func (s *suite) TheUserCreatesAServiceWithNameAndComputeUnitsFromAccountWithMetadataFromFile(
	serviceId, serviceName, computeUnits, accName, metadataFile string,
) {
	// Read the metadata file
	metadataBytes, err := os.ReadFile(metadataFile)
	require.NoError(s, err, "failed to read metadata file %s", metadataFile)

	// Encode to base64
	metadataBase64 := base64.StdEncoding.EncodeToString(metadataBytes)

	// Run the add-service command with metadata
	args := []string{
		"tx", "service", "add-service",
		serviceId,
		serviceName,
		computeUnits,
		"--card-base64", metadataBase64,
		"--from", accName,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}

	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "failed to create service with metadata: %v", err)
	s.pocketd.result = res
}

// TheUserUpdatesServiceWithMetadataFromFileFromAccount updates a service with metadata from a file
func (s *suite) TheUserUpdatesServiceWithMetadataFromFileFromAccount(
	serviceId, metadataFile, accName string,
) {
	// Get the existing service to retrieve its current name and compute units
	service := s.getService(serviceId)
	require.NotNil(s, service, "service %s does not exist", serviceId)

	// Read the metadata file
	metadataBytes, err := os.ReadFile(metadataFile)
	require.NoError(s, err, "failed to read metadata file %s", metadataFile)

	// Encode to base64
	metadataBase64 := base64.StdEncoding.EncodeToString(metadataBytes)

	// Run the add-service command with metadata (which also serves as update)
	args := []string{
		"tx", "service", "add-service",
		serviceId,
		service.Name,
		fmt.Sprintf("%d", service.ComputeUnitsPerRelay),
		"--card-base64", metadataBase64,
		"--from", accName,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}

	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "failed to update service with metadata: %v", err)
	s.pocketd.result = res
}

// TheServiceComputeUnitsPerRelayIsUpdatedToBy updates a service's compute_units_per_relay
// via MsgAddService, which doubles as an update for a service that already exists.
//
// The previous value is restored via s.Cleanup rather than a trailing Gherkin step: a
// mid-scenario failure would otherwise leave the mutated cupr behind, and every other
// feature which asserts settlement amounts for that service (0_tokenomics, session, relay)
// computes its expected values from it.
func (s *suite) TheServiceComputeUnitsPerRelayIsUpdatedToBy(serviceId, cuprStr, ownerAccName string) {
	s.Helper()

	previousService := s.getService(serviceId)
	require.NotNil(s, previousService, "service %s does not exist", serviceId)
	previousCUPR := previousService.ComputeUnitsPerRelay
	serviceName := previousService.Name

	s.setServiceComputeUnitsPerRelay(serviceId, serviceName, cuprStr, ownerAccName)

	s.Cleanup(func() {
		s.Logf("restoring compute units per relay of service %q to %d", serviceId, previousCUPR)
		s.setServiceComputeUnitsPerRelay(serviceId, serviceName, fmt.Sprintf("%d", previousCUPR), ownerAccName)

		// Drain the session that is in flight when the restore lands.
		//
		// A claim is priced at the cupr effective at its SESSION START, so a session which
		// began while the mutated value was live stays priced at it even after the restore.
		// Without this wait the next feature serves its relays inside that session and
		// settles at the mutated cupr: `session.feature` was observed settling 5 relays as
		// 1000 compute units (5 * 200) instead of 500.
		s.TheUserWaitsForTheNextSessionToStart()
	})
}

// setServiceComputeUnitsPerRelay re-registers a service with the given compute units per
// relay & blocks until the tx is observed onchain.
func (s *suite) setServiceComputeUnitsPerRelay(serviceId, serviceName, cuprStr, ownerAccName string) {
	s.Helper()

	args := []string{
		"tx", "service", "add-service",
		serviceId,
		serviceName,
		cuprStr,
		"--from", ownerAccName,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}

	// DEV_NOTE: Confirmed by tx hash rather than by a tx-result event: the cleanup which
	// restores the previous value sends a second AddService, which would match the first
	// one's replayed event & return before the restore is actually committed.
	s.broadcastTxAndRequireCommitted(args...)
}

// TheServiceShouldHaveAComputeUnitsPerRelayOfAtHeightOfTheClaim asserts the cupr recorded in
// the service's history for the session in which the last claim was created. It reads the
// history query rather than the live service so that a value which was changed after the
// session started is still observable at its original height.
func (s *suite) TheComputeUnitsPerRelayHistoryForShouldRecord(serviceId, expectedCUPRStr string) {
	s.Helper()

	args := []string{
		"query", "service", "compute-units-per-relay-history",
		serviceId,
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "failed to query compute units per relay history for service %s", serviceId)

	require.Containsf(s, res.Stdout, "computeUnitsPerRelayHistory",
		"expected a compute units per relay history for service %s, got: %s", serviceId, res.Stdout)
	require.Containsf(s, res.Stdout, fmt.Sprintf("%q", expectedCUPRStr),
		"expected compute units per relay history for service %s to record %s, got: %s", serviceId, expectedCUPRStr, res.Stdout)
}

// AServiceExistsWithComputeUnitsAndOwner creates a service without metadata
func (s *suite) AServiceExistsWithComputeUnitsAndOwner(
	serviceId, computeUnits, ownerAccName string,
) {
	// Run the add-service command without metadata
	args := []string{
		"tx", "service", "add-service",
		serviceId,
		fmt.Sprintf("Test service %s", serviceId),
		computeUnits,
		"--from", ownerAccName,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}

	_, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "failed to create service %s: %v", serviceId, err)

	// Wait for the service to be created
	s.waitForTxResultEvent(newEventMsgTypeMatchFn("service", "AddService"))
}

// TheServiceShouldExistWithMetadata verifies that a service exists and has metadata
func (s *suite) TheServiceShouldExistWithMetadata(serviceId string) {
	service := s.getService(serviceId)
	require.NotNil(s, service, "service %s does not exist", serviceId)
	require.NotNil(s, service.Metadata, "service %s has no metadata", serviceId)
	require.NotEmpty(s, service.Metadata.Card, "service %s metadata is empty", serviceId)

	// Verify the metadata size is reasonable
	require.LessOrEqual(s, len(service.Metadata.Card), 262144,
		"service %s metadata exceeds 256 KiB limit", serviceId)

	s.Logf("Service %s exists with %d bytes of metadata", serviceId, len(service.Metadata.Card))
}

// supplierAllowlistTeardownKey is the scenarioState key of the supplier allowlist
// scenario's *supplierAllowlistTeardown.
const supplierAllowlistTeardownKey = "supplierAllowlistTeardown"

// supplierAllowlistTeardown records what the supplier allowlist scenario changed, so
// its teardown can restore it.
type supplierAllowlistTeardown struct {
	serviceId    string
	ownerAccName string
	// stakedActors lists [actorType, accName] pairs, each once, in staking order.
	stakedActors [][2]string
	done         bool
}

// TheAccountStakesUpoktForService stakes the account as an actor of the given type for
// the service only, blocks until the tx is committed & refreshes the actor maps.
//
// DEV_NOTE: The tx is confirmed by hash, not by a tx-result event: a scenario staking
// several actors would match the replayed event of an earlier stake.
func (s *suite) TheAccountStakesUpoktForService(actorType, accName string, amount int64, serviceId string) {
	s.Helper()

	teardown := s.supplierAllowlistTeardownState()
	if !slices.Contains(teardown.stakedActors, [2]string{actorType, accName}) {
		teardown.stakedActors = append(teardown.stakedActors, [2]string{actorType, accName})
	}

	args, configPath := s.stakeForServiceTxArgs(actorType, accName, amount, serviceId)
	defer os.Remove(configPath)
	s.broadcastTxAndRequireCommitted(args...)

	s.buildAppMap()
	s.buildSupplierMap()
}

// TheAccountIsNotAllowedToStakeUpoktForService asserts the chain rejects the account's
// stake for the service because the service's supplier allowlist lists neither its owner
// nor its operator.
func (s *suite) TheAccountIsNotAllowedToStakeUpoktForService(actorType, accName string, amount int64, serviceId string) {
	s.Helper()

	args, configPath := s.stakeForServiceTxArgs(actorType, accName, amount, serviceId)
	defer os.Remove(configPath)
	txRes := s.broadcastTxAndRequireRejected(args...)
	require.Contains(s, txRes.RawLog, suppliertypes.ErrSupplierServiceNotAllowed.Error())
}

// stakeForServiceTxArgs writes a config staking the account (as its own owner & operator)
// for the service only, and returns the stake tx args using it with the config's path,
// which the caller removes.
func (s *suite) stakeForServiceTxArgs(actorType, accName string, amount int64, serviceId string) (args []string, configPath string) {
	s.Helper()

	configFile, err := os.CreateTemp("", fmt.Sprintf("%s_stake_config_*.yaml", accName))
	require.NoError(s, err)

	accAddress := accNameToAddrMap[accName]
	_, err = configFile.WriteString(s.getConfigFileContent(amount, accAddress, accAddress, actorType, serviceId))
	require.NoError(s, err)
	require.NoError(s, configFile.Close())

	return []string{
		"tx", actorType, fmt.Sprintf("stake-%s", actorType),
		"--config", configFile.Name(),
		"--from", accName,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}, configFile.Name()
}

// TheAccountAddsToTheSupplierAllowlistOfService adds the supplier's operator address to the
// service's supplier allowlist, signed by the service owner, & blocks until committed.
func (s *suite) TheAccountAddsToTheSupplierAllowlistOfService(ownerAccName, supplierAccName, serviceId string) {
	s.Helper()

	teardown := s.supplierAllowlistTeardownState()
	teardown.serviceId = serviceId
	teardown.ownerAccName = ownerAccName

	s.broadcastTxAndRequireCommitted(
		"tx", "service", "update-supplier-allowlist", serviceId,
		"--add", accNameToAddrMap[supplierAccName],
		"--from", ownerAccName,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	)
}

// TheSupplierAllowlistOfServiceListsButNot asserts the service is permissioned, lists the
// first supplier and does not list the second, through both allowlist queries.
func (s *suite) TheSupplierAllowlistOfServiceListsButNot(serviceId, listedAccName, unlistedAccName string) {
	s.Helper()

	allowlist := s.getSupplierAllowlist(serviceId)
	require.True(s, allowlist.Permissioned, "service %s should be permissioned", serviceId)
	require.Contains(s, allowlist.Addresses, accNameToAddrMap[listedAccName])
	require.NotContains(s, allowlist.Addresses, accNameToAddrMap[unlistedAccName])

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries,
		"query", "service", "is-supplier-allowed", serviceId, accNameToAddrMap[unlistedAccName],
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	)
	require.NoError(s, err)
	var entry servicetypes.QuerySupplierAllowlistEntryResponse
	s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &entry)
	require.True(s, entry.Permissioned, "service %s should be permissioned", serviceId)
	require.False(s, entry.Listed, "%s should not be listed on service %s", unlistedAccName, serviceId)
}

// TheSupplierAllowlistOfServiceIsEmpty asserts the service is open to every supplier.
func (s *suite) TheSupplierAllowlistOfServiceIsEmpty(serviceId string) {
	s.Helper()

	allowlist := s.getSupplierAllowlist(serviceId)
	require.False(s, allowlist.Permissioned, "service %s should not be permissioned", serviceId)
	require.Empty(s, allowlist.Addresses)
}

// TheUserWaitsForSupplierToBecomeInactiveForService waits for the height at which the
// supplier's config for the service is deactivated. The deactivation height is only set
// once the sweep has run, so a step observing the sweep's event must come first. It is
// one session after the sweep, well within eventTimeout on LocalNet.
func (s *suite) TheUserWaitsForSupplierToBecomeInactiveForService(supplierOperatorName, serviceId string) {
	s.Helper()

	deactivationHeight := int64(0)
	for _, config := range s.getSupplierInfo(supplierOperatorName).ServiceConfigHistory {
		if config.GetService().GetServiceId() == serviceId {
			deactivationHeight = max(deactivationHeight, config.DeactivationHeight)
		}
	}
	require.Positivef(s, deactivationHeight,
		"supplier %s has no deactivation scheduled for service %s", supplierOperatorName, serviceId)

	s.waitForBlockHeight(deactivationHeight)
}

// TheUserTearsDownTheSupplierAllowlistScenario reopens the allowlist the scenario changed,
// unstakes every actor it staked & waits until they are fully unbonded. It runs once;
// it is also registered as a cleanup, so it runs after a failed step too.
func (s *suite) TheUserTearsDownTheSupplierAllowlistScenario() {
	s.Helper()

	teardown := s.supplierAllowlistTeardownState()
	if teardown.done {
		return
	}

	if teardown.serviceId != "" {
		addresses := s.getSupplierAllowlist(teardown.serviceId).Addresses
		for chunk := range slices.Chunk(addresses, servicetypes.MaxSupplierAllowlistEntriesPerMsg) {
			s.broadcastTxAndRequireCommitted(
				"tx", "service", "update-supplier-allowlist", teardown.serviceId,
				"--remove", strings.Join(chunk, ","),
				"--from", teardown.ownerAccName,
				keyRingFlag,
				chainIdFlag,
				"--yes",
			)
		}
	}

	unbondingEndHeight := int64(0)
	for _, actor := range teardown.stakedActors {
		actorType, accName := actor[0], actor[1]
		if _, isStaked := s.getStakedAmount(actorType, accName); !isStaked {
			continue
		}

		var endHeight int64
		switch actorType {
		case actorTypeSupplier:
			if !s.getSupplierInfo(accName).IsUnbonding() {
				s.broadcastTxAndRequireCommitted(
					"tx", actorType, "unstake-supplier", accNameToAddrMap[accName],
					"--from", accName, keyRingFlag, chainIdFlag, "--yes",
				)
			}
			endHeight = s.getSupplierUnbondingEndHeight(accName)
		case actorTypeApp:
			if !s.getApplicationInfo(accName).IsUnbonding() {
				s.broadcastTxAndRequireCommitted(
					"tx", actorType, "unstake-application",
					"--from", accName, keyRingFlag, chainIdFlag, "--yes",
				)
			}
			endHeight = s.getApplicationUnbondingHeight(accName)
		default:
			s.Fatalf("unsupported actor type %q", actorType)
		}
		unbondingEndHeight = max(unbondingEndHeight, endHeight)
	}

	// The actors are removed in the EndBlocker of the unbonding end height. Wait in
	// steps: a single waitForBlockHeight call fails after eventTimeout.
	for height := s.getCurrentBlockHeight(); height <= unbondingEndHeight; height = s.getCurrentBlockHeight() {
		s.waitForBlockHeight(min(unbondingEndHeight+1, height+20))
	}

	s.buildAppMap()
	s.buildSupplierMap()

	// Set last: if a step above fails, the registered cleanup runs the teardown again.
	teardown.done = true
}

// supplierAllowlistTeardownState returns the scenario's teardown record, creating it
// and registering the teardown as a cleanup on first use.
func (s *suite) supplierAllowlistTeardownState() *supplierAllowlistTeardown {
	if teardown, ok := s.scenarioState[supplierAllowlistTeardownKey].(*supplierAllowlistTeardown); ok {
		return teardown
	}

	teardown := new(supplierAllowlistTeardown)
	s.scenarioState[supplierAllowlistTeardownKey] = teardown
	s.Cleanup(s.TheUserTearsDownTheSupplierAllowlistScenario)
	return teardown
}

// getSupplierAllowlist queries the service's whole supplier allowlist, page by page.
func (s *suite) getSupplierAllowlist(serviceId string) *servicetypes.QuerySupplierAllowlistResponse {
	s.Helper()

	allowlist := new(servicetypes.QuerySupplierAllowlistResponse)
	var pageKey []byte
	for {
		args := []string{
			"query", "service", "supplier-allowlist", serviceId,
			fmt.Sprintf("--%s=json", cometcli.OutputFlag),
		}
		if len(pageKey) > 0 {
			// Hex, not the JSON's base64: the CLI's binary flag tries hex first.
			args = append(args, "--page-key", hex.EncodeToString(pageKey))
		}
		res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
		require.NoError(s, err)

		var page servicetypes.QuerySupplierAllowlistResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &page)
		allowlist.Permissioned = page.Permissioned
		allowlist.Addresses = append(allowlist.Addresses, page.Addresses...)

		nextKey := page.GetPagination().GetNextKey()
		if len(nextKey) == 0 {
			return allowlist
		}
		require.NotEqual(s, pageKey, nextKey, "supplier allowlist pagination of service %s made no progress", serviceId)
		pageKey = nextKey
	}
}
