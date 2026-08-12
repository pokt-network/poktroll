//go:build e2e

package e2e

import (
	"encoding/base64"
	"fmt"
	"os"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/stretchr/testify/require"
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
