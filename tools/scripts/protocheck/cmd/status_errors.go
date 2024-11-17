package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

var (
	flagModule          = "module"
	flagModuleShorthand = "m"
	flagModuleValue     = "*"
	flagModuleUsage     = "If present, only check message handlers of the given module."

	statusErrorsCheckCmd = &cobra.Command{
		Use:   "status-errors [flags]",
		Short: "Checks that all message handler function errors are wrapped in gRPC status errors.",
		RunE:  runStatusErrorsCheck,
	}

	poktrollModules = map[string]struct{}{
		"application": {},
		//"gateway":     {},
		//"service":     {},
		//"session":     {},
		//"shared":      {},
		//"supplier":    {},
		//"proof":       {},
		//"tokenomics":  {},
	}
)

func init() {
	statusErrorsCheckCmd.Flags().StringVarP(&flagModule, flagModuleShorthand, "m", flagModuleValue, flagModuleUsage)
	rootCmd.AddCommand(statusErrorsCheckCmd)
}

// TODO_IN_THIS_COMMIT: pre-run: drop patch version in go.mod; post-run: restore.
func runStatusErrorsCheck(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// TODO_IN_THIS_COMMIT: to support this, need to load all modules but only inspect target module.
	//if flagModule != "*" {
	//	if _, ok := poktrollModules[flagModule]; !ok {
	//		return fmt.Errorf("unknown module %q", flagModule)
	//	}
	//
	//	if err := checkModule(ctx, flagModule); err != nil {
	//		return err
	//	}
	//}

	//for module := range poktrollModules {
	//	if err := checkModule(ctx, module); err != nil {
	if err := checkModule(ctx); err != nil {
		return err
	}
	//}

	return nil
}

func checkModule(_ context.Context) error {

	// 0. Get the package info for the given module's keeper package.
	// 1. Find the message server struct for the given module.
	// 2. Recursively traverse `msg_server_*.go` files to find all of its methods.
	// 3. Recursively traverse the method body to find all of its error returns.
	// 4. Lookup error assignments to ensure that they are wrapped in gRPC status errors.

	// TODO: import polyzero for side effects.
	//logger := polylog.Ctx(ctx)

	//xDir := filepath.Join(".", "x")
	//xDir := filepath.Join(".", "x", "application")
	//moduleDir := filepath.Join(".", "x", "application")
	//moduleDir := filepath.Join(".", "x", moduleName)
	//keeperDir := filepath.Join(moduleDir, "keeper")

	// TODO_IN_THIS_COMMIT: extract --- BEGIN
	// Set up the package configuration
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.LoadSyntax,
		//Mode:  packages.LoadAllSyntax,
		Tests: false, // Set to true if you also want to load test files
	}

	// Load the package containing the target file or directory
	poktrollPkgPathPattern := "github.com/pokt-network/poktroll/x/..."
	//moduleKeeperPkgPath := filepath.Join(poktrollPkgPathPattern, keeperDir)
	//xPkgPath := filepath.Join(poktrollPkgPathPattern, xDir)
	//fmt.Printf(">>> pkg path: %s\n", moduleKeeperPkgPath)
	//pkgs, err := packages.Load(cfg, moduleKeeperPkgPath)
	pkgs, err := packages.Load(cfg, poktrollPkgPathPattern)
	//pkgs, err := packages.Load(cfg, "github.com/pokt-network/poktroll/x/application")
	if err != nil {
		log.Fatalf("Failed to load package: %v", err)
	}

	offendingPkgErrLines := make([]string, 0)

	// Iterate over the keeper packages
	// E.g.:
	// - github.com/pokt-network/poktroll/x/application/keeper
	// - github.com/pokt-network/poktroll/x/gateway/keeper
	// - ...
	for _, pkg := range pkgs {
		fmt.Printf("pkg: %+v\n", pkg)
		if pkg.Name != "keeper" {
			continue
		}

		if len(pkg.Errors) > 0 {
			for _, pkgErr := range pkg.Errors {
				log.Printf("Package error: %v", pkgErr)
			}
			continue
		}

		// Print the package name and path
		//fmt.Printf("Package: %s (Path: %s)\n", pkg.Name, pkg.PkgPath)

		// Access type information
		info := pkg.TypesInfo
		if info == nil {
			log.Println("No type information available")
			continue
		}

		typeInfo := pkg.TypesInfo
		// --- END

		// TODO_IN_THIS_COMMIT: extract --- BEGIN
		for _, astFile := range pkg.Syntax {
			ast.Inspect(astFile, func(n ast.Node) bool {
				fnNode, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}

				// Skip functions which are not methods.
				if fnNode.Recv == nil {
					return false
				}

				fnType := fnNode.Recv.List[0].Type
				typeIdentNode, ok := fnType.(*ast.Ident)
				if !ok {
					return false
				}

				if typeIdentNode.Name != "msgServer" {
					return false
				}

				//fmt.Printf("Found msgServer method %q in %s\n", fnNode.Name.Name, matchFilePath)
				//fmt.Printf("in %q in %s\n", fnNode.Name.Name, astFile.Name.Name)

				condition := func(returnErrNode ast.Node) func(*ast.Ident, types.Object) bool {
					return func(sel *ast.Ident, typeObj types.Object) bool {
						isStatusError := sel.Name == "Error" && typeObj.Pkg().Path() == "google.golang.org/grpc/status"
						pos := pkg.Fset.Position(returnErrNode.Pos())
						if !isStatusError {
							//fmt.Printf("fnNode: %+v", fnNode)
							//fmt.Printf("typeIdentNode: %+v", typeIdentNode)
							offendingPkgErrLines = append(offendingPkgErrLines, fmt.Sprintf("%s:%d:%d", pos.Filename, pos.Line, pos.Column))
						}

						return isStatusError
						//return true
						//return false
					}
				}

				// Recursively traverse the function body, looking for non-nil error returns.
				//var errorReturns []*ast.IfStmt
				// TODO_IN_THIS_COMMIT: extract --- BEGIN
				ast.Inspect(fnNode.Body, func(n ast.Node) bool {
					switch n := n.(type) {
					case *ast.BlockStmt:
						return true
					// Search for a return statement.
					case *ast.ReturnStmt:
						lastReturnArg := n.Results[len(n.Results)-1]

						switch lastReturnArgNode := lastReturnArg.(type) {
						// `return nil, err` <-- last arg is an *ast.Ident.
						case *ast.Ident:
							//fmt.Printf("ast.Ident: %T: %+v\n", lastReturnArg, lastReturnArgNode)
							//return true

							//defs := typeInfo.Defs[lastReturnArgNode]
							//fmt.Printf("type defs: %+v\n", defs)

							//use := typeInfo.Uses[lastReturnArgNode]
							//fmt.Printf("type use: %+v\n", use)

							// TODO_IN_THIS_COMMIT: No need to check that the last return
							// arg is an error type if we checked that the function returns
							// an error as the last arg.
							if lastReturnArgNode.Name == "err" {
								if lastReturnArgNode.Obj == nil {
									return true
								}

								// TODO_IN_THIS_COMMIT: factor out and call in a case in the switch above where we handle *ast.AssignStmt
								switch node := lastReturnArgNode.Obj.Decl.(type) {
								case *ast.AssignStmt:
									//fmt.Printf("errAssignStmt found: %+v\n", node)

									selection := typeInfo.Selections[node.Rhs[0].(*ast.CallExpr).Fun.(*ast.SelectorExpr)]
									//fmt.Printf("type selection: %+v\n", selection)

									// TODO_IN_THIS_COMMIT: account for other cases...

									if selection == nil {
										fmt.Printf("ERROR: selection is nil\n")
										//return true
										return false
									}

									traverseFunctionBody(selection.Obj().(*types.Func), pkgs, 0, condition(lastReturnArgNode))

									return false
									//default:
									//return true
								}

								return false
								//return true
							}
						// `return nil, types.ErrXXX.Wrapf(...)` <-- last arg is a *ast.CallExpr.
						case *ast.CallExpr:
							//fmt.Printf("ast.CallExpr: %T: %+v\n", lastReturnArg, lastReturnArgNode)

							TraverseCallStack(lastReturnArgNode, pkgs, 0, condition(lastReturnArgNode))

							return false
							//return true
						default:
							//panic(fmt.Sprintf("unknown AST node type: %T: %+v", lastReturnArg, lastReturnArg))
							fmt.Printf("unknown AST node type: %T: %+v\n", lastReturnArg, lastReturnArg)
						}

						return false
						//return true
					}

					return true
				})
				// --- END

				//// TODO_IN_THIS_COMMIT: extract --- BEGIN
				//for _, errorReturn := range errorReturns {
				//	// Check if the error return is wrapped in a gRPC status error.
				//	//ifStmt, ok := errorReturn.If.(*ast.IfStmt)
				//	//if !ok {
				//	//	return false
				//	//}
				//	ifStmt := errorReturn //.If.(*ast.IfStmt)
				//
				//	switch node := ifStmt.Cond.(type) {
				//	case *ast.BinaryExpr:
				//		if node.Op != token.NEQ {
				//			return false
				//		}
				//	}
				//}
				//// --- END

				return false
				//return true
			})
		}

	}
	// --- END

	// Print offending lines in package
	// TODO_IN_THIS_COMMIT: refactor to const.
	pkgsPattern := "github.com/pokt-network/poktroll/x/..."
	numOffendingLines := len(offendingPkgErrLines)
	if numOffendingLines == 0 {
		fmt.Printf("No offending lines in %s\n", pkgsPattern)
	} else {
		msg := fmt.Sprintf(
			"\nFound %d offending lines in %s:",
			numOffendingLines, pkgsPattern,
		)
		fmt.Printf(
			"\n%s\n%s\n%s\n",
			msg,
			strings.Join(offendingPkgErrLines, "\n"),
			msg,
		)
	}

	return nil
}

// TraverseCallStack recursively traverses the call stack starting from a *ast.CallExpr.
func TraverseCallStack(call *ast.CallExpr, pkgs []*packages.Package, indent int, condition func(*ast.Ident, types.Object) bool) {
	fun := call.Fun
	switch fn := fun.(type) {
	case *ast.Ident:
		// Local or top-level function

		var useObj types.Object
		for _, pkg := range pkgs {
			useObj = pkg.TypesInfo.Uses[fn]
			if useObj != nil {
				break
			}
		}
		if useObj != nil {
			//fmt.Printf("%sFunction: %s\n", indentSpaces(indent), useObj.Name())
			if fnDecl, ok := useObj.(*types.Func); ok {
				traverseFunctionBody(fnDecl, pkgs, indent+2, condition)
			}
		}
	case *ast.SelectorExpr:
		// Method call like obj.Method()
		sel := fn.Sel
		var selection *types.Selection
		for _, pkg := range pkgs {
			selection = pkg.TypesInfo.Selections[fn]
			if selection != nil {
				break
			}
		}
		if selection != nil {
			// Instance method
			//fmt.Printf("%sMethod: %s on %s\n", indentSpaces(indent), sel.Name, selection.Recv())
			if method, ok := selection.Obj().(*types.Func); ok {
				traverseFunctionBody(method, pkgs, indent+2, condition)
			}
		} else {
			// Static or package-level call
			var useObj types.Object
			for _, pkg := range pkgs {
				useObj = pkg.TypesInfo.Uses[sel]
				if useObj != nil {
					break
				}
			}
			if useObj != nil {
				//fmt.Printf("%sFunction: %s (package-level: %s)\n", indentSpaces(indent), sel.Name, useObj.Pkg().Path())
				if condition(sel, useObj) {
					//fmt.Println(">>> STATUS ERROR FOUND!")
					return
				}

				if fnDecl, ok := useObj.(*types.Func); ok {
					traverseFunctionBody(fnDecl, pkgs, indent+2, condition)
				}
			}
		}
	default:
		fmt.Printf("%sUnknown function type: %T\n", indentSpaces(indent), fun)
	}

	// Recursively inspect arguments for nested calls
	for _, arg := range call.Args {
		if nestedCall, ok := arg.(*ast.CallExpr); ok {
			TraverseCallStack(nestedCall, pkgs, indent+2, condition)
		}
	}
}

// traverseFunctionBody analyzes the body of a function or method to find further calls.
func traverseFunctionBody(fn *types.Func, pkgs []*packages.Package, indent int, condition func(*ast.Ident, types.Object) bool) {
	//fmt.Printf("fn package path: %s\n", fn.Pkg().Path())
	//fmt.Printf("path has prefix: %v\n", strings.HasPrefix(fn.Pkg().Path(), "github.com/pokt-network/poktroll"))
	// Don't traverse beyond poktroll module root (i.e. assume deps won't return status errors).
	if !strings.HasPrefix(fn.Pkg().Path(), "github.com/pokt-network/poktroll") {
		return
	}

	// TODO_IN_THIS_COMMIT: Implement & log when this happens.
	// DEV_NOTE: If targetFileName is not present in any package,
	// we assume that a status error will not be returned by the
	// function; so we MUST mark it as offending.

	for _, pkg := range pkgs {
		// Find the declaration of the function in the AST
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				funcDecl, ok := node.(*ast.FuncDecl)
				if !ok {
					return true // Not the target function, continue
				}
				targetFileName := pkg.Fset.Position(fn.Pos()).Filename
				nodeFileName := pkg.Fset.Position(funcDecl.Pos()).Filename
				//fmt.Printf("nodeFileName: %s\n", nodeFileName)
				if nodeFileName != targetFileName {
					return true // Not the target function, continue
				}

				if funcDecl.Name.Name == fn.Name() {
					// Found the function, inspect its body for calls
					ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
						if call, ok := n.(*ast.CallExpr); ok {
							TraverseCallStack(call, pkgs, indent, condition)
						}
						return true
					})
					return false // Stop after finding the target function
				}
				return true
			})
		}
	}
}

// Helper function to generate indentation
func indentSpaces(indent int) string {
	return strings.Repeat(" ", indent)
}
