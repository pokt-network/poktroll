package main

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/tools/scripts/protocheck/goast"
)

const (
	poktrollMoudlePkgsPattern = "github.com/pokt-network/poktroll/x/..."
)

var (
	poktrollModulesRootPkgPath = filepath.Dir(poktrollMoudlePkgsPattern)

	checkStatusErrorsCmd = &cobra.Command{
		Use:    "status-errors [flags]",
		Short:  "Checks that all message handler function errors are wrapped in gRPC status errors.",
		PreRun: setupPrettyLogger,
		RunE:   runStatusErrorsCheck,
	}

	// TODO_IN_THIS_COMMIT: refactor to avoid global logger var.
	//logger                 polylog.Logger
	offendingPkgErrLineSet = make(map[string]struct{})
)

func init() {
	checkStatusErrorsCmd.Flags().StringVarP(&flagModuleValue, flagModule, flagModuleShorthand, flagModuleValue, flagModuleUsage)
	checkStatusErrorsCmd.Flags().StringVarP(&flagLogLevelValue, flagLogLevel, flagLogLevelShorthand, flagLogLevelValue, flagLogLevelUsage)
	rootCmd.AddCommand(checkStatusErrorsCmd)
}

func runStatusErrorsCheck(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	if err := validateModuleFlag(); err != nil {
		return err
	}

	// 0. Get the package info for ALL module packages.
	// 1. Find the message server struct for the given module.
	// 2. Recursively traverse `msg_server_*.go` files to find all of its methods.
	// 3. Recursively traverse the method body to find all of its error returns.
	// 4. Lookup error assignments to ensure that they are wrapped in gRPC status errors.

	// Set up the package configuration
	cfg := &packages.Config{
		Mode:  packages.LoadSyntax,
		Tests: false,
	}

	// TODO_IN_THIS_COMMIT: update comment...
	// Load the package containing the target file or directory
	pkgs, err := packages.Load(cfg, poktrollMoudlePkgsPattern)
	if err != nil {
		return fmt.Errorf("failed to load package: %w", err)
	}

	// Iterate over the keeper packages
	// E.g.:
	// - github.com/pokt-network/poktroll/x/application/keeper
	// - github.com/pokt-network/poktroll/x/application/types
	// - github.com/pokt-network/poktroll/x/gateway/keeper
	// - ...
	for _, pkg := range pkgs {
		if shouldSkipPackage(ctx, pkg) {
			continue
		}

		if err = checkPackage(ctx, pkg, pkgs); err != nil {
			return err
		}
	}

	printResults(ctx)

	return nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func validateModuleFlag() error {
	if flagModuleValue != "*" {
		switch flagModuleValue {
		case "application":
		case "gateway":
		case "proof":
		case "service":
		case "session":
		case "shared":
		case "supplier":
		case "tokenomics":
		default:
			return fmt.Errorf("ERROR: invalid module name: %s", flagModuleValue)
		}
	}
	return nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func shouldSkipPackage(ctx context.Context, pkg *packages.Package) bool {
	logger := polylog.Ctx(ctx)

	if flagModuleValue != "*" {
		moduleRootPath := fmt.Sprintf("%s/%s", poktrollModulesRootPkgPath, flagModuleValue)
		if !strings.HasPrefix(pkg.PkgPath, moduleRootPath) {
			return true
		}
	}

	if pkg.Name != "keeper" {
		return true
	}

	if len(pkg.Errors) > 0 {
		for _, pkgErr := range pkg.Errors {
			logger.Error().Msgf("⚠️ Skipping package %q due to error: %v", pkg.PkgPath, pkgErr)
		}
		return true
	}

	// Access type information
	if pkg.TypesInfo == nil {
		logger.Warn().Msgf("⚠️ No type information available, skipping package %q", pkg.PkgPath)
		return true
	}

	return false
}

// TODO_IN_THIS_COMMIT: move & godoc...
func checkPackage(ctx context.Context, pkg *packages.Package, pkgs []*packages.Package) error {
	for _, astFile := range pkg.Syntax {
		filename := pkg.Fset.Position(astFile.Pos()).Filename

		// Ignore protobuf generated files.
		if strings.HasSuffix(filepath.Base(filename), ".pb.go") {
			continue
		}
		if strings.HasSuffix(filepath.Base(filename), ".pb.gw.go") {
			continue
		}

		ast.Inspect(astFile, newInspectFileFn(ctx, pkg, pkgs))
	}

	return nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func newInspectFileFn(ctx context.Context, pkg *packages.Package, pkgs []*packages.Package) func(ast.Node) bool {
	logger := polylog.Ctx(ctx)

	return func(n ast.Node) bool {
		fnNode, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Skip functions which are not methods.
		if fnNode.Recv == nil {
			return false
		}

		fnNodeTypeObj, ok := pkg.TypesInfo.Defs[fnNode.Name] //.Type.Results.List[0].Type
		if !ok {
			logger.Warn().Msgf("ERROR: unable to find fnNode type def: %s\n", fnNode.Name.Name)
			return true
		}

		// Skip methods which are not exported.
		if !fnNodeTypeObj.Exported() {
			return false
		}

		// Skip methods which have no return arguments.
		if fnNode.Type.Results == nil {
			return false
		}

		// Ensure the last return argument type is error.
		fnResultTypes := fnNode.Type.Results.List
		lastResultType := fnResultTypes[len(fnResultTypes)-1].Type
		if lastResultTypeIdent, ok := lastResultType.(*ast.Ident); ok {
			if lastResultTypeIdent.Name != "error" {
				return false
			}
		}

		fnType := fnNode.Recv.List[0].Type
		typeIdentNode, ok := fnType.(*ast.Ident)
		if !ok {
			return false
		}

		fnPos := pkg.Fset.Position(fnNode.Pos())
		fnFilename := filepath.Base(fnPos.Filename)
		fnSourceHasQueryHandlerPrefix := strings.HasPrefix(fnFilename, "query_")

		if typeIdentNode.Name != "msgServer" && !fnSourceHasQueryHandlerPrefix {
			return false
		}

		// Recursively traverse the function body, looking for non-nil error returns.
		ast.Inspect(fnNode.Body, goast.NewInspectLastReturnArgFn(ctx, pkg, pkgs, appendOffendingLine, exonerateOffendingLine))

		return false
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func appendOffendingLine(_ context.Context, sourceLine string) {
	offendingPkgErrLineSet[sourceLine] = struct{}{}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func exonerateOffendingLine(ctx context.Context, sourceLine string) {
	logger := polylog.Ctx(ctx)

	if _, ok := offendingPkgErrLineSet[sourceLine]; ok {
		logger.Debug().Msgf("exhonerating %s", sourceLine)
		delete(offendingPkgErrLineSet, sourceLine)
	} else {
		logger.Warn().Msgf("can't exonerate %s", sourceLine)
	}
}

// TODO_IN_THIS_COMMIT: move & godoc... exits with code CodeNonStatusGRPCErrorsFound if offending lines found if offending lines found.
func printResults(ctx context.Context) {
	logger := polylog.Ctx(ctx)

	// Print offending lines in package
	pkgsPattern := poktrollMoudlePkgsPattern
	if flagModuleValue != "*" {
		pkgsPattern = fmt.Sprintf("github.com/pokt-network/poktroll/x/%s/...", flagModuleValue)
	}

	numOffendingLines := len(offendingPkgErrLineSet)
	if numOffendingLines == 0 {
		logger.Info().Msgf("🎉 No offending lines in %q 🎉", pkgsPattern)
	} else {
		offendingPkgErrLines := make([]string, 0, len(offendingPkgErrLineSet))
		for offendingPkgErrLine := range offendingPkgErrLineSet {
			offendingPkgErrLines = append(offendingPkgErrLines, offendingPkgErrLine)
		}

		sort.Strings(offendingPkgErrLines)

		msg := fmt.Sprintf(
			"🚨 Found %d offending lines in %q 🚨",
			numOffendingLines, pkgsPattern,
		)
		logger.Info().Msgf(
			"%s:\n%s",
			msg,
			strings.Join(offendingPkgErrLines, "\n"),
		)

		if numOffendingLines > 5 {
			logger.Info().Msg(msg)
		}

		os.Exit(CodeNonStatusGRPCErrorsFound)
	}
}
