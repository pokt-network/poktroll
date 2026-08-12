// Package cards validates Pocket cards: the small, self-describing JSON documents stored
// onchain in Service.metadata.card and Gateway.metadata.card.
//
// The chain deliberately enforces SIZE ONLY on those payloads -- it never parses them, so
// the card schema can evolve without a consensus change. That makes client-side validation
// the only place a malformed card can be caught before it costs gas and lands onchain, and
// this package is that place.
//
// The schemas embedded here are the canonical ones. docs/pocket_service_card.md documents
// them prose-side and MUST NOT carry a second copy.
package cards

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

//go:embed service_card.schema.json
var serviceCardSchemaJSON []byte

//go:embed gateway_card.schema.json
var gatewayCardSchemaJSON []byte

// Kind identifies which card schema to validate against.
type Kind string

const (
	KindService Kind = "service"
	KindGateway Kind = "gateway"
)

// SchemaKey is the value a card's `schema` key must hold, per Kind.
var SchemaKey = map[Kind]string{
	KindService: "pocket-service-card/v1",
	KindGateway: "pocket-gateway-card/v1",
}

var schemaJSON = map[Kind][]byte{
	KindService: serviceCardSchemaJSON,
	KindGateway: gatewayCardSchemaJSON,
}

// Schema returns the raw JSON Schema document for the given card kind.
func Schema(kind Kind) ([]byte, error) {
	raw, ok := schemaJSON[kind]
	if !ok {
		return nil, fmt.Errorf("unknown card kind %q (want one of: %s)", kind, strings.Join(Kinds(), ", "))
	}
	return raw, nil
}

// Kinds returns the supported card kinds, sorted.
func Kinds() []string {
	kinds := make([]string, 0, len(schemaJSON))
	for kind := range schemaJSON {
		kinds = append(kinds, string(kind))
	}
	sort.Strings(kinds)
	return kinds
}

// Validate checks a card payload against the embedded schema for the given kind.
//
// It reports EVERY problem it finds rather than only the first, so a publisher fixes a
// card in one pass instead of one round-trip per mistake.
//
// The size check mirrors the onchain rule (MaxServiceMetadataSizeBytes) so an oversized
// card fails here rather than after a broadcast.
func Validate(kind Kind, card []byte) error {
	schemaRaw, err := Schema(kind)
	if err != nil {
		return err
	}

	if len(card) == 0 {
		return fmt.Errorf("card is empty")
	}

	if len(card) > sharedtypes.MaxServiceMetadataSizeBytes {
		return fmt.Errorf(
			"card is %d bytes, which exceeds the onchain maximum of %d bytes",
			len(card), sharedtypes.MaxServiceMetadataSizeBytes,
		)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(card))
	if err != nil {
		return fmt.Errorf("card is not valid JSON: %w", err)
	}

	compiled, err := compile(kind, schemaRaw)
	if err != nil {
		return err
	}

	if err := compiled.Validate(instance); err != nil {
		return fmt.Errorf("card does not match the %s card schema:\n%s", kind, formatValidationError(err))
	}

	return nil
}

// formatValidationError renders EVERY schema violation as one indented line pointing at the
// offending JSON path. A publisher fixing a card wants the whole list in one pass.
//
// It walks to the LEAF errors: the tree's interior nodes carry generic wrappers
// ("validation failed"), while the leaves carry the message that actually says what is
// wrong, e.g. which enum values were allowed.
func formatValidationError(err error) string {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return indentLines(err.Error())
	}

	printer := message.NewPrinter(language.English)

	var lines []string
	var walk func(node *jsonschema.ValidationError)
	walk = func(node *jsonschema.ValidationError) {
		if len(node.Causes) > 0 {
			for _, cause := range node.Causes {
				walk(cause)
			}
			return
		}

		location := "/" + strings.Join(node.InstanceLocation, "/")
		if len(node.InstanceLocation) == 0 {
			location = "(root)"
		}

		lines = append(lines, fmt.Sprintf("  %s: %s", location, describe(node, location, printer)))
	}
	walk(validationErr)

	if len(lines) == 0 {
		return indentLines(validationErr.Error())
	}

	// Deduplicate: the same leaf can be reached through more than one branch.
	seen := make(map[string]struct{}, len(lines))
	deduped := lines[:0]
	for _, line := range lines {
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		deduped = append(deduped, line)
	}
	sort.Strings(deduped)

	return strings.Join(deduped, "\n")
}

// describe renders one leaf error, replacing the library's opaque message for the
// deliberately-forbidden `required` key with an explanation of why it is forbidden.
func describe(node *jsonschema.ValidationError, location string, printer *message.Printer) string {
	if strings.HasSuffix(location, "/required") {
		return "`required` is not a card field -- nothing enforces it at stake time or relay time, " +
			"and a consumer may decline a transport. Use `intent` instead (e.g. \"intent\": \"expected\")"
	}
	return node.ErrorKind.LocalizedString(printer)
}

// compiledSchema memoizes one kind's compiled schema. The embedded schemas are immutable,
// so compilation is a pure function of Kind and is worth doing at most once per process.
type compiledSchema struct {
	once   sync.Once
	schema *jsonschema.Schema
	err    error
}

// compiledSchemas holds one memo per Kind. Populated at init so the map itself is never
// written after startup and concurrent Validate calls only touch the per-kind sync.Once.
var compiledSchemas = func() map[Kind]*compiledSchema {
	memos := make(map[Kind]*compiledSchema, len(schemaJSON))
	for kind := range schemaJSON {
		memos[kind] = &compiledSchema{}
	}
	return memos
}()

// compile returns the compiled schema for a card kind, compiling it on first use.
//
// Batch commands (e.g. `tx service edit-service` over a config naming a card per service)
// call Validate once per entry; without this the same schema was parsed and compiled from
// scratch every time.
func compile(kind Kind, schemaRaw []byte) (*jsonschema.Schema, error) {
	memo, ok := compiledSchemas[kind]
	if !ok {
		// Unreachable via Validate, which resolves the kind through Schema() first.
		return compileUncached(kind, schemaRaw)
	}

	memo.once.Do(func() {
		memo.schema, memo.err = compileUncached(kind, schemaRaw)
	})

	return memo.schema, memo.err
}

// compileUncached parses and compiles the embedded schema for a card kind.
func compileUncached(kind Kind, schemaRaw []byte) (*jsonschema.Schema, error) {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaRaw))
	if err != nil {
		return nil, fmt.Errorf("embedded %s card schema is not valid JSON: %w", kind, err)
	}

	// Identify the resource by the schema's own $id. It is only an identifier -- nothing is
	// fetched, since the schemas are self-contained by design -- but using $id keeps a local
	// filesystem path out of user-facing validation errors.
	schemaURL := fmt.Sprintf("https://pocket.network/schemas/pocket-%s-card/v1.json", kind)
	if asMap, ok := schemaDoc.(map[string]any); ok {
		if id, ok := asMap["$id"].(string); ok && id != "" {
			schemaURL = id
		}
	}

	compiler := jsonschema.NewCompiler()
	if addErr := compiler.AddResource(schemaURL, schemaDoc); addErr != nil {
		return nil, fmt.Errorf("embedded %s card schema is not a valid JSON Schema: %w", kind, addErr)
	}

	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("embedded %s card schema failed to compile: %w", kind, err)
	}

	return compiled, nil
}

// Summary returns a one-line human description of a card: its declared schema, size, and
// whether it is compact. Used by the CLI to report what was validated.
func Summary(card []byte) string {
	var declared struct {
		Schema string `json:"schema"`
	}
	_ = json.Unmarshal(card, &declared)

	schemaName := declared.Schema
	if schemaName == "" {
		schemaName = "(no schema key)"
	}

	return fmt.Sprintf("%s, %d bytes (onchain max %d)",
		schemaName, len(card), sharedtypes.MaxServiceMetadataSizeBytes)
}

// indentLines prefixes each line so multi-line output reads as one block.
func indentLines(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}
