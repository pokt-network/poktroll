//go:build e2e

package e2e

import (
	"encoding/base64"
	"fmt"
	"os"

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
		"--experimental-metadata-base64", metadataBase64,
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
		"--experimental-metadata-base64", metadataBase64,
		"--from", accName,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}

	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "failed to update service with metadata: %v", err)
	s.pocketd.result = res
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
	require.NotEmpty(s, service.Metadata.ExperimentalApiSpecs, "service %s metadata is empty", serviceId)

	// Verify the metadata size is reasonable
	require.LessOrEqual(s, len(service.Metadata.ExperimentalApiSpecs), 262144,
		"service %s metadata exceeds 256 KiB limit", serviceId)

	s.Logf("Service %s exists with %d bytes of metadata", serviceId, len(service.Metadata.ExperimentalApiSpecs))
}
