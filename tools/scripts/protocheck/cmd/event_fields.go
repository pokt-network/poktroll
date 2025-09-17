package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/jhump/protoreflect/desc"            //nolint:staticcheck // deprecated but still functional
	"github.com/jhump/protoreflect/desc/protoparse" //nolint:staticcheck // deprecated but still functional
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
)

var (
	eventFieldsCheckPassed = true
	eventFieldsViolations  []eventFieldViolation
)

type eventFieldViolation struct {
	FilePath    string
	Line        int
	MessageName string
	FieldName   string
	FieldType   string
}

func init() {
	eventFieldsCmd := &cobra.Command{
		Use:   "event-fields",
		Short: "Check that Event messages only contain primitive fields",
		Long: `Checks all protobuf messages whose names begin with "Event" to ensure they
only contain primitive fields. Non-primitive fields in events negatively impact
disk utilization in Cosmos SDK applications.

This command walks through the proto directory tree and analyzes all .proto files,
specifically looking for messages that start with "Event" and flagging any that
contain non-primitive field types.

Primitive types include:
- Scalar types: int32, int64, uint32, uint64, sint32, sint64, fixed32, fixed64, 
  sfixed32, sfixed64, float, double, bool, string, bytes
- Enums

Non-primitive types that will be flagged:
- Message types (like cosmos.base.v1beta1.Coin)
- Repeated fields
- Map fields
- Oneof fields
- google.protobuf.Any

The output groups violations by field type to help prioritize refactoring efforts.`,
		Example: `  # Check event fields in the default proto directory
  protocheck event-fields

  # Check event fields in a specific directory
  protocheck event-fields --root ./my-proto-dir

  # Run with debug logging
  protocheck event-fields --log-level debug`,
		PreRunE: logger.PreRunESetup,
		RunE:    runEventFieldsCheck,
	}

	eventFieldsCmd.Flags().BoolP(
		"help", "h",
		false, "Show help for event-fields check",
	)

	// Add logger flags
	eventFieldsCmd.Flags().StringVar(
		&logger.LogLevel,
		cosmosflags.FlagLogLevel,
		flags.DefaultLogLevel,
		flags.FlagLogLevelUsage,
	)
	eventFieldsCmd.Flags().StringVar(
		&logger.LogOutput,
		flags.FlagLogOutput,
		flags.DefaultLogOutput,
		flags.FlagLogOutputUsage,
	)

	rootCmd.AddCommand(eventFieldsCmd)
}

func runEventFieldsCheck(_ *cobra.Command, _ []string) error {
	logger.Logger.Info().Msg("Checking Event messages for non-primitive fields...")

	protoRootDir := flagRootValue
	err := filepath.Walk(
		protoRootDir,
		forEachMatchingFileWalkFn(
			"*.proto",
			func(path string) {
				// logger.Logger.Debug().Str("file", path).Msg("Processing file")
				if err := checkEventFieldsFn(path); err != nil {
					logger.Logger.Error().Err(err).Str("file", path).Msg("Error checking file")
				}
			},
		),
	)
	if err != nil {
		return err
	}

	// Print results
	if !eventFieldsCheckPassed {
		fmt.Println("\n❌ Event field check FAILED!")
		fmt.Printf("\nFound %d Event message(s) with non-primitive fields:\n\n", len(eventFieldsViolations))

		// Group violations by field type
		violationsByType := make(map[string][]eventFieldViolation)
		for _, v := range eventFieldsViolations {
			violationsByType[v.FieldType] = append(violationsByType[v.FieldType], v)
		}

		for fieldType, violations := range violationsByType {
			fmt.Printf("🔍 Non-primitive type: %s\n", fieldType)
			fmt.Printf("   Found in %d Event message(s):\n", len(violations))
			for _, violation := range violations {
				fmt.Printf("     - %s:%d - %s.%s\n",
					violation.FilePath, violation.Line, violation.MessageName, violation.FieldName)
			}
			fmt.Println()
		}

		os.Exit(CodeEventFieldsCheckFailed)
	}

	fmt.Println("✅ Event field check passed!")
	return nil
}

func checkEventFieldsFn(protoFilePath string) error {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = os.Getenv("HOME") + "/go"
	}

	// Get the current working directory to resolve relative paths
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	parser := protoparse.Parser{
		ImportPaths: []string{
			flagRootValue,
			filepath.Join(cwd, "proto"), // Add the proto directory for local imports
			goPath + "/pkg/mod/github.com/cosmos/gogoproto@v1.7.0",
			goPath + "/pkg/mod/github.com/cosmos/cosmos-proto@v1.0.0-beta.5/proto",
			goPath + "/pkg/mod/github.com/cosmos/cosmos-sdk@v0.53.0/proto",
			goPath + "/pkg/mod/github.com/cosmos/cosmos-sdk@v0.53.0/third_party/proto",
		},
		IncludeSourceCodeInfo: true,
	}

	// Make the path relative to flagRootValue if needed
	relPath, err := filepath.Rel(flagRootValue, protoFilePath)
	if err != nil {
		// If we can't make it relative to flagRootValue, parse directly with just the filename
		// This handles cases where the file is outside the flagRootValue directory (e.g., tests)
		relPath = filepath.Base(protoFilePath)
		parser.ImportPaths = append(parser.ImportPaths, filepath.Dir(protoFilePath))
	}

	fds, err := parser.ParseFiles(relPath)
	if err != nil {
		// Skip files that can't be parsed due to missing dependencies
		// This is expected for files that depend on external proto files
		// log.Printf("Skipping %s due to parse error: %v", protoFilePath, err)
		return nil
	}

	for _, fd := range fds {
		for _, msgDesc := range fd.GetMessageTypes() {
			checkEventMessage(protoFilePath, msgDesc)
		}
	}

	return nil
}

func checkEventMessage(filePath string, msgDesc *desc.MessageDescriptor) {
	// Check if this is an Event message
	if !strings.HasPrefix(msgDesc.GetName(), "Event") {
		return
	}

	// log.Printf("Checking Event message: %s in %s", msgDesc.GetName(), filePath)

	// Check all fields in the message
	for _, field := range msgDesc.GetFields() {
		// log.Printf("  Field: %s, Type: %s, IsPrimitive: %t", field.GetName(), getFieldTypeDescription(field), isPrimitiveField(field))
		if !isPrimitiveField(field) {
			eventFieldsCheckPassed = false

			// Get line number from source code info
			line := 0
			if loc := field.GetSourceInfo(); loc != nil && len(loc.Span) > 0 {
				line = int(loc.Span[0]) + 1 // Convert 0-based to 1-based
			}

			// Convert to absolute path for IDE linking
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				absPath = filePath // fallback to original path if abs fails
			}

			violation := eventFieldViolation{
				FilePath:    absPath,
				Line:        line,
				MessageName: msgDesc.GetName(),
				FieldName:   field.GetName(),
				FieldType:   getFieldTypeDescription(field),
			}
			eventFieldsViolations = append(eventFieldsViolations, violation)
		}
	}

	// No need to check nested messages - only top-level Event messages matter
}

func isPrimitiveField(field *desc.FieldDescriptor) bool {
	// Check if it's a repeated field
	if field.IsRepeated() {
		return false
	}

	// Check if it's a map field
	if field.IsMap() {
		return false
	}

	// Check if it's part of a oneof
	if field.GetOneOf() != nil {
		return false
	}

	// Check the field type
	switch field.GetType() {
	// Primitive scalar types
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		return true

	// Message types are not primitive
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		// Special case: check if it's google.protobuf.Any
		if field.GetMessageType() != nil && field.GetMessageType().GetFullyQualifiedName() == "google.protobuf.Any" {
			return false
		}
		return false

	default:
		return false
	}
}

func getFieldTypeDescription(field *desc.FieldDescriptor) string {
	var typeStr string

	// Handle repeated fields
	if field.IsRepeated() {
		typeStr = "repeated "
	}

	// Handle map fields
	if field.IsMap() {
		keyType := field.GetMessageType().GetFields()[0].GetType().String()
		valueType := field.GetMessageType().GetFields()[1].GetType().String()
		return fmt.Sprintf("map<%s, %s>", keyType, valueType)
	}

	// Handle message types
	if field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		if msgType := field.GetMessageType(); msgType != nil {
			typeStr += msgType.GetFullyQualifiedName()
		} else {
			typeStr += "message"
		}
	} else if field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM {
		if enumType := field.GetEnumType(); enumType != nil {
			typeStr += enumType.GetFullyQualifiedName()
		} else {
			typeStr += "enum"
		}
	} else {
		// Scalar types
		typeStr += strings.ToLower(strings.TrimPrefix(field.GetType().String(), "TYPE_"))
	}

	// Add oneof information
	if oneOf := field.GetOneOf(); oneOf != nil {
		typeStr += fmt.Sprintf(" (in oneof %s)", oneOf.GetName())
	}

	return typeStr
}
