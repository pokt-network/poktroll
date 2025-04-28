package types

import (
	"encoding/binary"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ binary.ByteOrder

const (
	// SupplierOperatorKeyPrefix is the prefix to retrieve all Supplier
	SupplierOperatorKeyPrefix = "Supplier/operator_address/"

	// SupplierUnstakingHeightKeyPrefix is the prefix for indexing suppliers by their unstaking height
	SupplierUnstakingHeightKeyPrefix = "Supplier/unbonding_height/"

	// ServiceConfigUpdateKeyPrefix is the prefix for indexing service configs by service ID
	ServiceConfigUpdateKeyPrefix = "ServiceConfigUpdate/service_id/"

	// SupplierServiceConfigUpdateKeyPrefix is the prefix for indexing service configs by operator address
	SupplierServiceConfigUpdateKeyPrefix = "ServiceConfigUpdate/operator_address/"

	// ServiceConfigUpdateActivationHeightKeyPrefix is the prefix for indexing service configs by activation height
	ServiceConfigUpdateActivationHeightKeyPrefix = "ServiceConfigUpdate/activation_height/"

	// ServiceConfigUpdateDeactivationHeightKeyPrefix is the prefix for indexing service configs by deactivation height
	ServiceConfigUpdateDeactivationHeightKeyPrefix = "ServiceConfigUpdate/deactivation_height/"
)

// SupplierOperatorKey returns the store key to retrieve a Supplier from the index fields
func SupplierOperatorKey(supplierOperatorAddr string) []byte {
	return StringKey(supplierOperatorAddr)
}

// ServiceConfigUpdateKey returns the store key to retrieve a ServiceConfig from the index fields
// The key is composed of service ID, activation height, and supplier operator address
// This ordering allows efficient range queries for configurations by service ID and activation height
func ServiceConfigUpdateKey(serviceConfigUpdate sharedtypes.ServiceConfigUpdate) []byte {
	var key []byte

	serviceIdKey := StringKey(serviceConfigUpdate.Service.ServiceId)
	key = append(key, serviceIdKey...)

	activationHeightKey := IntKey(serviceConfigUpdate.ActivationHeight)
	key = append(key, activationHeightKey...)

	supplierOperatorAddressKey := StringKey(serviceConfigUpdate.OperatorAddress)
	key = append(key, supplierOperatorAddressKey...)

	return key
}

// SupplierServiceConfigUpdateKey returns the store key to retrieve a ServiceConfig from the index fields
// The key is composed of supplier operator address, activation height, and service ID
// This ordering allows efficient range queries for configurations by supplier operator
// address and activation height
func SupplierServiceConfigUpdateKey(serviceConfigUpdate sharedtypes.ServiceConfigUpdate) []byte {
	var key []byte

	supplierOperatorAddressKey := StringKey(serviceConfigUpdate.OperatorAddress)
	key = append(key, supplierOperatorAddressKey...)

	activationHeightKey := IntKey(serviceConfigUpdate.ActivationHeight)
	key = append(key, activationHeightKey...)

	serviceIdKey := StringKey(serviceConfigUpdate.Service.ServiceId)
	key = append(key, serviceIdKey...)

	return key
}

// ServiceConfigUpdateActivationHeightKey returns the store key to retrieve a ServiceConfig from the index fields
// The key is composed of activation height, service ID, and supplier operator address
func ServiceConfigUpdateActivationHeightKey(serviceConfigUpdate sharedtypes.ServiceConfigUpdate) []byte {
	var key []byte

	activationHeightKey := IntKey(serviceConfigUpdate.ActivationHeight)
	key = append(key, activationHeightKey...)

	serviceIdKey := StringKey(serviceConfigUpdate.Service.ServiceId)
	key = append(key, serviceIdKey...)

	supplierOperatorAddressKey := StringKey(serviceConfigUpdate.OperatorAddress)
	key = append(key, supplierOperatorAddressKey...)

	return key
}

// ServiceConfigUpdateDeactivationHeightKey returns the store key to retrieve a ServiceConfig from the index fields
// The key is composed of deactivation height, service ID, supplier operator address, and activation height
func ServiceConfigUpdateDeactivationHeightKey(serviceConfigUpdate sharedtypes.ServiceConfigUpdate) []byte {
	var key []byte

	deactivationHeightKey := IntKey(serviceConfigUpdate.DeactivationHeight)
	key = append(key, deactivationHeightKey...)

	serviceIdKey := StringKey(serviceConfigUpdate.Service.ServiceId)
	key = append(key, serviceIdKey...)

	supplierOperatorAddressKey := StringKey(serviceConfigUpdate.OperatorAddress)
	key = append(key, supplierOperatorAddressKey...)

	activationHeightKey := IntKey(serviceConfigUpdate.ActivationHeight)
	key = append(key, activationHeightKey...)

	return key
}

// IntKey converts an integer value to a byte slice for use in store keys
// Appends a '/' separator to the end of the key for consistent prefix scanning
func IntKey(intIndex int64) []byte {
	var key []byte

	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(intIndex))
	key = append(key, heightBz...)
	key = append(key, []byte("/")...)

	return key
}

// StringKey converts a string value to a byte slice for use in store keys
// Appends a '/' separator to the end of the key for consistent prefix scanning
func StringKey(strIndex string) []byte {
	var key []byte

	strIndexBz := []byte(strIndex)
	key = append(key, strIndexBz...)
	key = append(key, []byte("/")...)

	return key
}
