// decode_relay.go - Debug utility for RelayRequest unmarshaling failures
//
// This tool helps diagnose base64-encoded RelayRequest data that fails to unmarshal.
// It provides detailed analysis of protobuf structure and field validation.
//
// Usage:
//   go run tools/scripts/decode_relay/main.go <base64_encoded_relay_request>
//
// Common use cases:
//   - Debug unmarshaling errors from relayer proxy logs
//   - Analyze malformed RelayRequest data
//   - Validate RelayRequest field structure
//   - Inspect protobuf wire format for debugging

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("❌ RelayRequest Debug Tool")
		fmt.Println("")
		fmt.Println("This tool helps debug base64-encoded RelayRequest data that fails to unmarshal.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  go run decode_relay.go <base64_encoded_relay_request>")
		fmt.Println("")
		fmt.Println("Example:")
		fmt.Println("  go run decode_relay.go 'ChYKFGFwcGxpY2F0aW9uX2FkZHJlc3M='")
		fmt.Println("")
		fmt.Println("To get base64 data:")
		fmt.Println("  - Check relayer proxy logs for 'body_bytes' field")
		fmt.Println("  - Look for unmarshaling error messages with base64 data")
		os.Exit(1)
	}

	base64Data := os.Args[1]
	fmt.Printf("🔍 Debugging RelayRequest unmarshaling failure\n")
	fmt.Println("")

	// Step 1: Decode base64
	fmt.Println("📦 Step 1: Decoding base64 data...")
	binaryData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		fmt.Printf("❌ Base64 decode failed: %v\n", err)
		fmt.Println("")
		fmt.Println("💡 Common causes:")
		fmt.Println("   - Invalid base64 characters")
		fmt.Println("   - Missing padding")
		fmt.Println("   - Corrupted data")
		os.Exit(1)
	}

	fmt.Printf("✅ Base64 decoded successfully: %d bytes\n", len(binaryData))
	if len(binaryData) == 0 {
		fmt.Println("❌ Decoded data is empty")
		os.Exit(1)
	}

	// Show binary data info
	fmt.Printf("📊 Binary data (hex): %x\n", binaryData)
	fmt.Println("")

	// Step 2: Attempt RelayRequest unmarshaling
	fmt.Println("🔬 Step 2: Attempting RelayRequest unmarshaling...")
	var relayRequest types.RelayRequest
	err = relayRequest.Unmarshal(binaryData)
	if err != nil {
		fmt.Printf("❌ RelayRequest unmarshal failed: %v\n", err)
		fmt.Println("")
		fmt.Println("🔍 Step 3: Analyzing protobuf structure for debugging...")
		analyzeProtobufStructure(binaryData)
		return
	}

	fmt.Println("✅ RelayRequest unmarshaled successfully!")
	fmt.Println("")

	// Step 3: Validate fields
	fmt.Println("✅ Step 3: Validating RelayRequest fields...")
	validateRelayRequest(&relayRequest)

	// Step 4: Display readable output
	fmt.Println("📄 Step 4: Displaying readable RelayRequest structure...")
	jsonData, err := json.MarshalIndent(relayRequest, "", "  ")
	if err != nil {
		fmt.Printf("❌ JSON marshal failed: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
	fmt.Println("")
	fmt.Println("🎉 RelayRequest debugging completed successfully!")
}

func validateRelayRequest(req *types.RelayRequest) {
	fmt.Println("🔍 Validating RelayRequest structure:")
	fmt.Println("")

	// Validate meta field
	if req.Meta.SessionHeader == nil {
		fmt.Println("❌ Meta field: Missing or invalid")
	} else {
		fmt.Println("✅ Meta field: Present and valid")
	}
	validateRelayRequestMetadata(&req.Meta)

	// Validate payload
	if len(req.Payload) == 0 {
		fmt.Println("❌ Payload field: Empty (no API request data)")
		fmt.Println("   💡 This means the RelayRequest contains no actual API call data")
	} else {
		fmt.Printf("✅ Payload field: Present (%d bytes)\n", len(req.Payload))
		// Show payload preview if it looks like text
		if isPrintableText(req.Payload) {
			fmt.Printf("   📋 Payload preview: %s\n", truncateString(string(req.Payload), 100))
		} else {
			fmt.Printf("   📋 Payload (hex): %x\n", truncateBytes(req.Payload, 50))
		}
	}

	fmt.Println("")
}

func validateRelayRequestMetadata(meta *types.RelayRequestMetadata) {
	fmt.Println("🔍 Validating RelayRequestMetadata:")

	// Validate session header
	if meta.SessionHeader == nil {
		fmt.Println("❌ SessionHeader: Missing (required for relay routing)")
		fmt.Println("   💡 SessionHeader contains app address, service ID, and session info")
	} else {
		fmt.Println("✅ SessionHeader: Present")
		validateSessionHeader(meta.SessionHeader)
	}

	// Validate signature
	if len(meta.Signature) == 0 {
		fmt.Println("❌ Signature: Empty (relay authentication will fail)")
		fmt.Println("   💡 Signature proves the application authorized this relay")
	} else {
		fmt.Printf("✅ Signature: Present (%d bytes)\n", len(meta.Signature))
		fmt.Printf("   📋 Signature (hex): %x\n", truncateBytes(meta.Signature, 32))
	}

	// Validate supplier operator address
	if meta.SupplierOperatorAddress == "" {
		fmt.Println("❌ SupplierOperatorAddress: Empty (relay routing will fail)")
		fmt.Println("   💡 This should be the address of the supplier handling the relay")
	} else {
		fmt.Printf("✅ SupplierOperatorAddress: %s\n", meta.SupplierOperatorAddress)
	}

	fmt.Println("")
}

func validateSessionHeader(header *sessiontypes.SessionHeader) {
	fmt.Println("🔍 Validating SessionHeader:")

	// Validate application address
	if header.ApplicationAddress == "" {
		fmt.Println("❌ ApplicationAddress: Empty (required for billing)")
		fmt.Println("   💡 Should be a valid Pokt network address (pokt1...)")
	} else {
		fmt.Printf("✅ ApplicationAddress: %s\n", header.ApplicationAddress)
		if !isValidPoktAddress(header.ApplicationAddress) {
			fmt.Println("   ⚠️  Warning: Address format may be invalid")
		}
	}

	// Validate service ID
	if header.ServiceId == "" {
		fmt.Println("❌ ServiceId: Empty (required for relay routing)")
		fmt.Println("   💡 Should match a registered service (e.g., 'ethereum', 'polygon')")
	} else {
		fmt.Printf("✅ ServiceId: %s\n", header.ServiceId)
	}

	// Validate session ID
	if header.SessionId == "" {
		fmt.Println("❌ SessionId: Empty (required for session management)")
		fmt.Println("   💡 Should be a unique identifier for this session")
	} else {
		fmt.Printf("✅ SessionId: %s\n", header.SessionId)
	}

	// Validate session start block height
	if header.SessionStartBlockHeight == 0 {
		fmt.Println("❌ SessionStartBlockHeight: 0 (invalid block height)")
		fmt.Println("   💡 Should be a positive block number when session started")
	} else {
		fmt.Printf("✅ SessionStartBlockHeight: %d\n", header.SessionStartBlockHeight)
	}

	// Validate session end block height
	if header.SessionEndBlockHeight == 0 {
		fmt.Println("❌ SessionEndBlockHeight: 0 (invalid block height)")
		fmt.Println("   💡 Should be a positive block number when session ends")
	} else {
		fmt.Printf("✅ SessionEndBlockHeight: %d\n", header.SessionEndBlockHeight)
		// Validate session duration
		if header.SessionStartBlockHeight > 0 {
			duration := header.SessionEndBlockHeight - header.SessionStartBlockHeight
			fmt.Printf("   📊 Session duration: %d blocks\n", duration)
			if duration <= 0 {
				fmt.Println("   ⚠️  Warning: Session end height should be greater than start height")
			}
		}
	}

	fmt.Println("")
}

func analyzeProtobufStructure(data []byte) {
	fmt.Println("🔍 Manual protobuf structure analysis:")
	fmt.Println("")
	fmt.Println("This low-level analysis helps identify structural issues in the protobuf data.")
	fmt.Println("")

	if len(data) == 0 {
		fmt.Println("❌ Data is empty - cannot analyze structure")
		return
	}

	pos := 0
	fieldCount := 0

	fmt.Println("📋 Expected RelayRequest structure:")
	fmt.Println("   Field 1 (meta): RelayRequestMetadata (wire type 2 - length-delimited)")
	fmt.Println("   Field 2 (payload): bytes (wire type 2 - length-delimited)")
	fmt.Println("")
	fmt.Println("🔍 Parsing protobuf fields:")

	for pos < len(data) {
		if pos >= len(data) {
			fmt.Printf("❌ Unexpected end of data at position %d\n", pos)
			break
		}

		// Read field header (varint encoding)
		fieldHeader, headerLen := readVarint(data[pos:])
		if headerLen == 0 {
			fmt.Printf("❌ Failed to read field header at position %d\n", pos)
			fmt.Println("   💡 This might indicate corrupted protobuf data")
			break
		}

		pos += headerLen
		fieldNumber := fieldHeader >> 3
		wireType := fieldHeader & 0x7

		fmt.Printf("Field %d: number=%d, wireType=%d", fieldCount+1, fieldNumber, wireType)

		// Determine expected field and wire type
		switch fieldNumber {
		case 1:
			fmt.Print(" (meta - RelayRequestMetadata)")
			if wireType != 2 {
				fmt.Printf(" ⚠️  Expected wire type 2, got %d", wireType)
			}
		case 2:
			fmt.Print(" (payload - bytes)")
			if wireType != 2 {
				fmt.Printf(" ⚠️  Expected wire type 2, got %d", wireType)
			}
		default:
			fmt.Printf(" ❌ Unknown field %d (not part of RelayRequest)", fieldNumber)
		}

		// Try to read the field value based on wire type
		valueLen, err := getFieldLength(data[pos:], int(wireType))
		if err != nil {
			fmt.Printf(" - ERROR: %v\n", err)
			fmt.Println("   💡 This indicates malformed protobuf data")
			break
		}

		if pos+valueLen > len(data) {
			fmt.Printf(" - ERROR: field extends beyond data (need %d bytes, have %d)\n", valueLen, len(data)-pos)
			fmt.Println("   💡 This indicates truncated or corrupted data")
			break
		}

		fmt.Printf(" - %d bytes\n", valueLen)
		pos += valueLen
		fieldCount++
	}

	fmt.Println("")
	fmt.Printf("📊 Parse summary: %d fields parsed, %d/%d bytes consumed\n", fieldCount, pos, len(data))

	if pos < len(data) {
		fmt.Printf("❌ %d bytes remaining unparsed\n", len(data)-pos)
		fmt.Println("   💡 This may indicate:")
		fmt.Println("      - Extra data appended to valid protobuf")
		fmt.Println("      - Truncated data from valid protobuf")
		fmt.Println("      - Parsing stopped due to errors")
		fmt.Println("      - Data corruption")
	} else if fieldCount == 0 {
		fmt.Println("❌ No valid fields found")
		fmt.Println("   💡 This suggests the data is not valid protobuf")
	} else {
		fmt.Println("✅ All data consumed successfully")
	}

	fmt.Println("")
	fmt.Println("🔍 Troubleshooting tips:")
	fmt.Println("   - Check if the base64 data was truncated or corrupted")
	fmt.Println("   - Verify the data comes from a RelayRequest (not RelayResponse)")
	fmt.Println("   - Look for encoding issues in the source system")
	fmt.Println("   - Check if protobuf schema versions match")
}

// Helper functions for protobuf parsing and validation

func readVarint(data []byte) (uint64, int) {
	var result uint64
	var shift uint
	for i, b := range data {
		if i >= 10 { // varint can't be longer than 10 bytes
			return 0, 0
		}
		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, i + 1
		}
		shift += 7
	}
	return 0, 0 // incomplete varint
}

func getFieldLength(data []byte, wireType int) (int, error) {
	switch wireType {
	case 0: // Varint
		_, len := readVarint(data)
		if len == 0 {
			return 0, fmt.Errorf("incomplete varint")
		}
		return len, nil
	case 1: // 64-bit
		if len(data) < 8 {
			return 0, fmt.Errorf("need 8 bytes for 64-bit field, have %d", len(data))
		}
		return 8, nil
	case 2: // Length-delimited
		length, lengthBytes := readVarint(data)
		if lengthBytes == 0 {
			return 0, fmt.Errorf("incomplete length prefix")
		}
		totalLength := lengthBytes + int(length)
		if len(data) < totalLength {
			return 0, fmt.Errorf("need %d bytes for length-delimited field, have %d", totalLength, len(data))
		}
		return totalLength, nil
	case 5: // 32-bit
		if len(data) < 4 {
			return 0, fmt.Errorf("need 4 bytes for 32-bit field, have %d", len(data))
		}
		return 4, nil
	default:
		return 0, fmt.Errorf("unsupported wire type %d", wireType)
	}
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// truncateBytes truncates a byte slice to the specified length
func truncateBytes(data []byte, maxLen int) []byte {
	if len(data) <= maxLen {
		return data
	}
	return data[:maxLen]
}

// isPrintableText checks if byte slice contains printable text
func isPrintableText(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

// isValidPoktAddress checks if string looks like a valid Pokt address
func isValidPoktAddress(addr string) bool {
	if len(addr) < 5 {
		return false
	}
	return addr[:4] == "pokt" || addr[:5] == "pokt1"
}
