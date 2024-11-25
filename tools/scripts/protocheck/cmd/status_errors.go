package main

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	flagModule          = "module"
	flagModuleShorthand = "m"
	// TODO_IN_THIS_COMMIT: support this flag.
	flagModuleValue = "*"
	flagModuleUsage = "If present, only check message handlers of the given module."

	flagLogLevel          = "log-level"
	flagLogLevelShorthand = "l"
	flagLogLevelValue     = "info"
	flagLogLevelUsage     = "The logging level (debug|info|warn|error)"

	statusErrorsCheckCmd = &cobra.Command{
		Use:    "status-errors [flags]",
		Short:  "Checks that all message handler function errors are wrapped in gRPC status errors.",
		PreRun: setupLogger,
		RunE:   runStatusErrorsCheck,
	}

	logger                 polylog.Logger
	offendingPkgErrLineSet = make(map[string]struct{})
)

func init() {
	statusErrorsCheckCmd.Flags().StringVarP(&flagModuleValue, flagModule, flagModuleShorthand, flagModuleValue, flagModuleUsage)
	statusErrorsCheckCmd.Flags().StringVarP(&flagLogLevelValue, flagLogLevel, flagLogLevelShorthand, flagLogLevelValue, flagLogLevelUsage)
	rootCmd.AddCommand(statusErrorsCheckCmd)
}

func setupLogger(_ *cobra.Command, _ []string) {
	logger = polyzero.NewLogger(
		polyzero.WithWriter(zerolog.ConsoleWriter{Out: os.Stdout}),
		polyzero.WithLevel(polyzero.ParseLevel(flagLogLevelValue)),
	)
}

// TODO_IN_THIS_COMMIT: pre-run: drop patch version in go.mod; post-run: restore.
func runStatusErrorsCheck(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// TODO_IN_THIS_COMMIT: extract to validation function.
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

	// TODO_IN_THIS_COMMIT: add hack/work-around to temporarily strip patch version from go.mod.
	// TODO_IN_THIS_COMMIT: add hack/work-around to temporarily strip patch version from go.mod.
	// TODO_IN_THIS_COMMIT: add hack/work-around to temporarily strip patch version from go.mod.

	// TODO_IN_THIS_COMMIT: to support this, need to load all modules but only inspect target module.
	//if flagModule != "*" {
	// ...
	//}

	//for module := range poktrollModules {
	//	if err := checkModule(ctx, module); err != nil {
	if err := checkModule(ctx); err != nil {
		return err
	}
	//}

	return nil
}

// TODO_IN_THIS_COMMIT: 2-step check
//   1. Collect all return statements from `msgServer` methods and `Keeper` methods in `query_*.go` files.
//   2. For each return statement, check the type:
//     *ast.Ident: search this package ...
//     *ast.SelectorExpr: search the package of its declaration...
//     ...for an *ast.AssignStmt with the given *ast.Ident as the left-hand side.

func checkModule(_ context.Context) error {

	// 0. Get the package info for the given module's keeper package.
	// 1. Find the message server struct for the given module.
	// 2. Recursively traverse `msg_server_*.go` files to find all of its methods.
	// 3. Recursively traverse the method body to find all of its error returns.
	// 4. Lookup error assignments to ensure that they are wrapped in gRPC status errors.

	// TODO: import polyzero for side effects.
	//logger := polylog.Ctx(ctx)

	// TODO_IN_THIS_COMMIT: extract --- BEGIN
	// Set up the package configuration
	cfg := &packages.Config{
		//Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.LoadSyntax,
		Mode:  packages.LoadSyntax,
		Tests: false,
	}

	// Load the package containing the target file or directory
	poktrollPkgPathPattern := "github.com/pokt-network/poktroll/x/..."
	//logger.Info().Msgf("Loading package(s) in %s", poktrollPkgPathPattern)

	pkgs, err := packages.Load(cfg, poktrollPkgPathPattern)
	if err != nil {
		return fmt.Errorf("failed to load package: %w", err)
	}

	// Iterate over the keeper packages
	// E.g.:
	// - github.com/pokt-network/poktroll/x/application/keeper
	// - github.com/pokt-network/poktroll/x/gateway/keeper
	// - ...
	for _, pkg := range pkgs {
		if flagModuleValue != "*" {
			moduleRootPath := fmt.Sprintf("github.com/pokt-network/poktroll/x/%s", flagModuleValue)
			if !strings.HasPrefix(pkg.PkgPath, moduleRootPath) {
				continue
			}
		}

		if pkg.Name != "keeper" {
			continue
		}

		if len(pkg.Errors) > 0 {
			for _, pkgErr := range pkg.Errors {
				logger.Error().Msgf("Package error: %v", pkgErr)
			}
			continue
		}

		// Access type information
		info := pkg.TypesInfo
		if info == nil {
			logger.Warn().Msgf("No type information available, skipping package %q", pkg.PkgPath)
			continue
		}

		// --- END

		//filenames := make([]string, 0)
		//for _, astFile := range pkg.Syntax {
		//	filenames = append(filenames, filepath.Base(pkg.Fset.Position(astFile.Pos()).Filename))
		//}
		//fmt.Printf(">>> filenames:\n%s\n", strings.Join(filenames, "\n"))

		// TODO_IN_THIS_COMMIT: extract --- BEGIN
		// TODO_IN_THIS_COMMIT: check the filename and only inspect each once!
		for _, astFile := range pkg.Syntax {
			filename := pkg.Fset.Position(astFile.Pos()).Filename

			// Ignore protobuf generated files.
			if strings.HasSuffix(filepath.Base(filename), ".pb.go") {
				continue
			}
			if strings.HasSuffix(filepath.Base(filename), ".pb.gw.go") {
				continue
			}

			// TODO_IN_THIS_COMMIT: remove!
			//fmt.Printf(">>> filename: %s\n", filename)
			//if filename != "/home/bwhite/Projects/pokt/poktroll/x/application/keeper/msg_server_delegate_to_gateway.go" {
			//	continue
			//}

			ast.Inspect(astFile, func(n ast.Node) bool {
				fnNode, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}

				// Skip functions which are not methods.
				if fnNode.Recv == nil {
					return false
				}

				fnNodeTypeObj, ok := info.Defs[fnNode.Name] //.Type.Results.List[0].Type
				if !ok {
					fmt.Printf("ERROR: unable to find fnNode type def: %s\n", fnNode.Name.Name)
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

				//fmt.Printf(">>> fNode.Name.Name: %s\n", fnNode.Name.Name)
				//if fnNode.Name.Name != "AllApplications" {
				//	return false
				//}

				// TODO_IN_THIS_COMMIT: check the signature of the method to ensure it returns an error type.
				fnResultsList := fnNode.Type.Results.List
				fnLastResultType := fnResultsList[len(fnResultsList)-1].Type
				if fnLastResultIdent, ok := fnLastResultType.(*ast.Ident); ok {
					if fnLastResultIdent.Name != "error" {
						return false
					}
				}

				fnType := fnNode.Recv.List[0].Type
				typeIdentNode, ok := fnType.(*ast.Ident)
				if !ok {
					return false
				}

				fnPos := pkg.Fset.Position(fnNode.Pos())
				//fmt.Printf(">>> fnNode.Pos(): %s\n", fnPos.String())
				fnFilename := filepath.Base(fnPos.Filename)
				fnSourceHasQueryHandlerPrefix := strings.HasPrefix(fnFilename, "query_")
				//fnSourceHasQueryHandlerPrefix := false

				if typeIdentNode.Name != "msgServer" && !fnSourceHasQueryHandlerPrefix {
					return false
				}

				// TODO_IN_THIS_COMMIT: figure out why this file hangs the command.
				isExcludedFile := false
				for _, excludedFile := range []string{"query_get_session.go"} {
					if fnFilename == excludedFile {
						isExcludedFile = true
					}
				}
				if isExcludedFile {
					return false
				}

				// Recursively traverse the function body, looking for non-nil error returns.
				// TODO_IN_THIS_COMMIT: extract --- BEGIN
				//fmt.Printf(">>> walking func from file: %s\n", pkg.Fset.Position(astFile.Pos()).Filename)
				ast.Inspect(fnNode.Body, func(n ast.Node) bool {
					if n == nil {
						return false
					}
					//inspectPos := pkg.Fset.Position(n.Pos())
					//fmt.Printf(">>> inspecting %T at %s\n", n, inspectPos)

					//inspectLocation := inspectPos.String()
					//if inspectLocation == "/home/bwhite/Projects/pokt/poktroll/x/proof/keeper/msg_server_create_claim.go:44:3" {
					//	fmt.Printf(">>> found it!\n")
					//}

					switch n := n.(type) {
					case *ast.BlockStmt:
						return true
					// Search for a return statement.
					case *ast.ReturnStmt:
						lastResult := n.Results[len(n.Results)-1]
						inspectPosition := pkg.Fset.Position(lastResult.Pos()).String()

						logger := logger.With(
							"node_type", fmt.Sprintf("%T", lastResult),
							"inspectPosition", fmt.Sprintf(" %s ", inspectPosition),
						)

						switch lastReturnArgNode := lastResult.(type) {
						// `return nil, err` <-- last arg is an *ast.Ident.
						case *ast.Ident:
							//logger.Debug().Fields(map[string]any{
							//	"node_type": fmt.Sprintf("%T", lastReturnArgNode),
							//	"inspectPosition":  pkg.Fset.Position(lastReturnArgNode.Pos()).String(),
							//}).Msg("traversing ast node")

							// TODO_IN_THIS_COMMIT: No need to check that the last return
							// arg is an error type if we checked that the function returns
							// an error as the last arg.
							//if lastReturnArgNode.Name == "err" {
							if lastReturnArgNode.Obj == nil {
								return true
							}

							def := pkg.TypesInfo.Uses[lastReturnArgNode]
							if def == nil {
								logger.Warn().Msg("def is nil")
								return true
							}

							if def.Type().String() != "error" {
								//logger.Warn().Msg("def is not error")
								//inspectPosition := pkg.Fset.Position(lastReturnArgNode.Pos()).String()
								//break
								return false
							}

							logger.Debug().Msg("appending potential offending line")
							appendOffendingLine(inspectPosition)
							traceExpressionStack(lastReturnArgNode, pkgs, lastReturnArgNode, offendingPkgErrLineSet)
							return true

						// `return nil, types.ErrXXX.Wrapf(...)` <-- last arg is a *ast.CallExpr.
						case *ast.CallExpr:
							//logger.Debug().Msg("inspecting ast node")
							logger.Debug().Msg("appending potential offending line")
							appendOffendingLine(inspectPosition)
							traceExpressionStack(lastReturnArgNode, pkgs, lastReturnArgNode, offendingPkgErrLineSet)
							//TraverseCallStack(lastReturnArgNode, pkgs, 0, condition(lastReturnArgNode))
							//return false
							return true

						case *ast.SelectorExpr:
							logger.Debug().Msg("appending potential offending line")
							appendOffendingLine(inspectPosition)
							traceSelectorExpr(lastReturnArgNode, pkgs, lastReturnArgNode, offendingPkgErrLineSet)
							return true

						default:
							logger.Warn().Msg("NOT traversing ast node")
							return true
						}

						//logger.Debug().Msg("appending potential offending line")
						//appendOffendingLine(inspectPosition)
						//
						//return true
					}

					return true
				})
				// --- END

				return false
			})
		}

	}

	// --- END

	// TODO_IN_THIS_COMMIT: extract --- BEGIN
	// TODO_IN_THIS_COMMIT: figure out why there are duplicate offending lines.
	// Print offending lines in package
	// TODO_IN_THIS_COMMIT: refactor to const.
	pkgsPattern := "github.com/pokt-network/poktroll/x/..."
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
	}
	// --- END

	return nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func appendOffendingLine(sourceLine string) {
	offendingPkgErrLineSet[sourceLine] = struct{}{}
}
