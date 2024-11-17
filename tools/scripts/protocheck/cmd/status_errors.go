package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
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

	if flagModule != "*" {
		if _, ok := poktrollModules[flagModule]; !ok {
			return fmt.Errorf("unknown module %q", flagModule)
		}

		if err := checkModule(ctx, flagModule); err != nil {
			return err
		}
	}

	for module := range poktrollModules {
		if err := checkModule(ctx, module); err != nil {
			return err
		}
	}

	return nil
}

func checkModule(_ context.Context, moduleName string) error {

	// 0. Get the package info for the given module's keeper package.
	// 1. Find the message server struct for the given module.
	// 2. Recursively traverse `msg_server_*.go` files to find all of its methods.
	// 3. Recursively traverse the method body to find all of its error returns.
	// 4. Lookup error assignments to ensure that they are wrapped in gRPC status errors.

	// TODO: import polyzero for side effects.
	//logger := polylog.Ctx(ctx)

	moduleDir := filepath.Join(".", "x", moduleName)
	keeperDir := filepath.Join(moduleDir, "keeper")

	// TODO_IN_THIS_COMMIT: extract --- BEGIN
	// Set up the package configuration
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.LoadSyntax,
		//Mode:  packages.LoadAllSyntax,
		Tests: false, // Set to true if you also want to load test files
	}

	// Load the package containing the target file or directory
	poktrollPkgPathRoot := "github.com/pokt-network/poktroll"
	moduleKeeperPkgPath := filepath.Join(poktrollPkgPathRoot, keeperDir)
	pkgs, err := packages.Load(cfg, moduleKeeperPkgPath)
	if err != nil {
		log.Fatalf("Failed to load package: %v", err)
	}

	// Iterate over the packages
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, pkgErr := range pkg.Errors {
				log.Printf("Package error: %v", pkgErr)
			}
			continue
		}

		// Print the package name and path
		fmt.Printf("Package: %s (Path: %s)\n", pkg.Name, pkg.PkgPath)

		// Access type information
		info := pkg.TypesInfo
		if info == nil {
			log.Println("No type information available")
			continue
		}

		// Inspect the type information
		//for ident, obj := range info.Defs {
		//	if obj != nil {
		//		fmt.Printf("Identifier: %s, Type: %s\n", ident.Name, obj.Type())
		//	}
		//}
		// TODO_IN_THIS_COMMIT: assert only 1 pkg: module's keeper...
		typeInfo := pkgs[0].TypesInfo
		// --- END

		//msgServerGlob := filepath.Join(keeperDir, "msg_server_*.go")
		//
		//matches, err := filepath.Glob(msgServerGlob)
		//if err != nil {
		//	return err
		//}

		offendingPkgErrLines := make([]string, 0)

		// TODO_IN_THIS_COMMIT: extract --- BEGIN
		//for _, matchFilePath := range matches[:1] {
		//for _, astFile := range pkgs[0].Syntax {
		for _, astFile := range pkg.Syntax {
			//fset := token.NewFileSet()
			//
			//astFile, err := parser.ParseFile(fset, matchFilePath, nil, parser.AllErrors)
			//if err != nil {
			//	return err
			//}

			//fmt.Println("BEFORE...")
			//typeInfo, err := getTypeInfo(fset, matchFilePath, astFile)
			//if err != nil {
			//	return err
			//}
			////typeInfo := types.Info{}
			//fmt.Println("AFTER...")

			//// Skip files which don't match the msg_server_*.go pattern.
			//if !strings.HasPrefix(astFile.Name.Name, "msg_server_") {
			//	continue
			//}

			//ast.Walk
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
				fmt.Printf("in %q in %s\n", fnNode.Name.Name, astFile.Name.Name)

				condition := func(sel *ast.Ident, typeObj types.Object) bool {
					isStatusError := sel.Name == "Error" && typeObj.Pkg().Path() == "google.golang.org/grpc/status"
					pos := pkg.Fset.Position(sel.Pos())
					if !isStatusError {
						fmt.Printf("fnNode: %+v", fnNode)
						fmt.Printf("typeIdentNode: %+v", typeIdentNode)
						offendingPkgErrLines = append(offendingPkgErrLines, fmt.Sprintf("%s:%d:%d", pos.Filename, pos.Line, pos.Column))
					}

					return isStatusError
					//return true
					//return false
				}

				// Recursively traverse the function body, looking for non-nil error returns.
				var errorReturns []*ast.IfStmt
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
							fmt.Printf("ast.Ident: %T: %+v\n", lastReturnArg, lastReturnArgNode)
							//return true

							defs := typeInfo.Defs[lastReturnArgNode]
							fmt.Printf("type defs: %+v\n", defs)

							use := typeInfo.Uses[lastReturnArgNode]
							fmt.Printf("type use: %+v\n", use)

							// TODO_IN_THIS_COMMIT: No need to check that the last return
							// arg is an error type if we checked that the function returns
							// an error as the last arg.
							if lastReturnArgNode.Name == "err" {
								def := typeInfo.Defs[lastReturnArgNode]
								fmt.Printf("def: %+v\n", def)

								if lastReturnArgNode.Obj == nil {
									return true
								}

								// TODO_IN_THIS_COMMIT: factor out and call in a case in the switch above where we handle *ast.AssignStmt
								switch node := lastReturnArgNode.Obj.Decl.(type) {
								case *ast.AssignStmt:
									// TODO_IN_THIS_COMMIT: extract --- BEGIN
									//errAssignStmt, ok := node.(*ast.AssignStmt)
									//if !ok {
									//	panic(fmt.Sprintf("not an ast.AssignStmt: %T: %+v", node, node))
									//}
									//errAssignStmt := node

									//use := typeInfo.Uses[errAssignStmt.Rhs[0]]
									//def := typeInfo.Defs[errAssignStmt.Rhs[0]]
									//_type := typeInfo.Types[errAssignStmt.Rhs[0]]
									//impl := typeInfo.Implicits[errAssignStmt.Rhs[0]]
									//inst := typeInfo.Instances[errAssignStmt.Rhs[0]]

									fmt.Printf("errAssignStmt found: %+v\n", node)
									//fmt.Printf("use: %+v\n", use)
									//fmt.Printf("def: %+v\n", def)
									//fmt.Printf("_type: %+v\n", _type)
									//fmt.Printf("impl: %+v\n", impl)
									//fmt.Printf("inst: %+v\n", inst)
									// --- END

									selection := typeInfo.Selections[node.Rhs[0].(*ast.CallExpr).Fun.(*ast.SelectorExpr)]
									fmt.Printf("type selection: %+v\n", selection)

									// TODO_IN_THIS_COMMIT: account for other cases...
									//posNode := GetNodeAtPos(astFile, pkg.Fset, node.Rhs[0].(*ast.CallExpr).Fun.Pos())
									//fmt.Printf("posNode: %+v\n", posNode)

									traverseFunctionBody(selection.Obj().(*types.Func), pkg, 0, condition)

									return false
									//default:
									//return true
								}
								//errAssignStmt, ok := lastReturnIdent.Obj.Decl.(*ast.AssignStmt)
								//if !ok {
								//	panic(fmt.Sprintf("not an ast.AssignStmt: %T: %+v", lastReturnIdent.Obj.Decl, lastReturnIdent.Obj.Decl))
								//}
								//
								////use := typeInfo.Uses[errAssignStmt.Rhs[0]]
								//def := typeInfo.Defs[lastReturnArgNode]
								//_type := typeInfo.Types[errAssignStmt.Rhs[0]]
								//impl := typeInfo.Implicits[errAssignStmt.Rhs[0]]
								////inst := typeInfo.Instances[errAssignStmt.Rhs[0]]
								//
								//fmt.Printf("return found: %+v\n", n)
								////fmt.Printf("use: %+v\n", use)
								//fmt.Printf("def: %+v\n", def)
								//fmt.Printf("_type: %+v\n", _type)
								//fmt.Printf("impl: %+v\n", impl)
								////fmt.Printf("inst: %+v\n", inst)
								//
								////errAssignStmt.Rhs

								return false
								//return true
							}
						// `return nil, types.ErrXXX.Wrapf(...)` <-- last arg is a *ast.CallExpr.
						case *ast.CallExpr:
							fmt.Printf("ast.CallExpr: %T: %+v\n", lastReturnArg, lastReturnArgNode)

							TraverseCallStack(lastReturnArgNode, pkg, 0, condition)

							//// TODO_IN_THIS_COMMIT: handle other types of CallExprs
							//switch sel := lastReturnArgNode.Fun.(type) {
							//case *ast.SelectorExpr:
							//	_type := typeInfo.Types[sel]
							//	fmt.Printf("sel types: %T: %+v\n", _type, _type)
							//
							//	selections := typeInfo.Selections[sel]
							//	fmt.Printf("sel selections: %+v\n", selections)
							//default:
							//	panic(fmt.Sprintf("unknown AST node type: %T: %+v", lastReturnArg, lastReturnArg))
							//}
							//
							//return true
							return false
							//return true
						default:
							//panic(fmt.Sprintf("unknown AST node type: %T: %+v", lastReturnArg, lastReturnArg))
							fmt.Printf("unknown AST node type: %T: %+v\n", lastReturnArg, lastReturnArg)
						}

						//use := typeInfo.Uses[lastReturnIdent]
						//def := typeInfo.Defs[lastReturnIdent]
						//_type := typeInfo.Types[lastReturnIdent]
						//impl := typeInfo.Implicits[lastReturnIdent]
						//inst := typeInfo.Instances[lastReturnIdent]
						//
						////fmt.Printf("return found: %+v\n", n)
						//fmt.Printf("use: %+v\n", use)
						//fmt.Printf("def: %+v\n", def)
						//fmt.Printf("_type: %+v\n", _type)
						//fmt.Printf("impl: %+v\n", impl)
						//fmt.Printf("inst: %+v\n", inst)

						return false
						//return true
					}

					return true

					//ifStmt, ok := n.(*ast.IfStmt)
					//if !ok {
					//	// Skip AST branches which are not logically conditional branches.
					//	//fmt.Println("non if")
					//	return true
					//}
					////fmt.Println("yes if")
					//
					//// Match on `if err != nil` statements.
					//// TODO_IN_THIS_COMMIT: extract --- BEGIN
					//if ifStmt.Cond == nil {
					//	return false
					//}
					//
					//errorReturn, ok := ifStmt.Cond.(*ast.BinaryExpr)
					//if !ok {
					//	return false
					//}
					//
					//if errorReturn.Op != token.NEQ {
					//	return false
					//}
					//
					//// Check that the left operand is an error type.
					//// TODO_IN_THIS_COMMIT: extract --- BEGIN
					//errIdentNode, ok := errorReturn.X.(*ast.Ident)
					//if !ok {
					//	return false
					//}
					//
					////errIdentNode.Obj.Kind.String()
					//obj := typeInfo.Uses[errIdentNode]
					//fmt.Sprintf("obj: %+v", obj)
					//// --- END
					//// --- END
					//
					//errorReturns = append(errorReturns, ifStmt)
					//
					//return false
				})
				// --- END

				// TODO_IN_THIS_COMMIT: extract --- BEGIN
				for _, errorReturn := range errorReturns {
					// Check if the error return is wrapped in a gRPC status error.
					//ifStmt, ok := errorReturn.If.(*ast.IfStmt)
					//if !ok {
					//	return false
					//}
					ifStmt := errorReturn //.If.(*ast.IfStmt)

					switch node := ifStmt.Cond.(type) {
					case *ast.BinaryExpr:
						if node.Op != token.NEQ {
							return false
						}

						//statusErrorIdentNode, ok := ifStmtCond.X.(*ast.Ident)
						//if !ok {
						//	continue
						//}

						//fmt.Printf("Found error return %q in %s\n", statusErrorIdentNode.Name, matchFilePath)
					}
				}
				// --- END

				return false
				//return true
			})
		}

		// Print offending lines in package
		fmt.Printf("offending lines in %s:\n%s\n", pkg.PkgPath, strings.Join(offendingPkgErrLines, "\n"))
	}
	// --- END

	return nil
}

//// TODO_IN_THIS_COMMIT: move & refactor...
//var _ ast.Visitor = (*Visitor)(nil)
//
//type Visitor struct{}
//
//// TODO_IN_THIS_COMMIT: move & refactor...
//func (v *Visitor) Visit(node ast.Node) ast.Visitor {
//
//}

// TODO_IN_THIS_COMMIT: move & godoc...
//func getTypeInfo(fset *token.FileSet, filePath string, fileNode *ast.File) (*types.Info, error) {
//	//conf := types.Config{
//	//	Importer: importer.For("source", nil),
//	//}
//	//info := &types.Info{
//	//	Types: make(map[ast.Expr]types.TypeAndValue),
//	//	Defs:  make(map[*ast.Ident]types.Object),
//	//	Uses:  make(map[*ast.Ident]types.Object),
//	//}
//	//if _, err := conf.Check(fileNode.Name.Name, fset, []*ast.File{fileNode}, info); err != nil {
//	//	return nil, err
//	//}
//	//
//	//return info, nil
//	return &types.Info{}, nil
//}

// TraverseCallStack recursively traverses the call stack starting from a *ast.CallExpr.
func TraverseCallStack(call *ast.CallExpr, pkg *packages.Package, indent int, condition func(*ast.Ident, types.Object) bool) {
	fun := call.Fun
	switch fn := fun.(type) {
	case *ast.Ident:
		// Local or top-level function
		obj := pkg.TypesInfo.Uses[fn]
		if obj != nil {
			fmt.Printf("%sFunction: %s\n", indentSpaces(indent), obj.Name())
			if fnDecl, ok := obj.(*types.Func); ok {
				traverseFunctionBody(fnDecl, pkg, indent+2, condition)
			}
		}
	case *ast.SelectorExpr:
		// Method call like obj.Method()
		sel := fn.Sel
		obj := pkg.TypesInfo.Selections[fn]
		if obj != nil {
			// Instance method
			fmt.Printf("%sMethod: %s on %s\n", indentSpaces(indent), sel.Name, obj.Recv())
			if method, ok := obj.Obj().(*types.Func); ok {
				traverseFunctionBody(method, pkg, indent+2, condition)
			}
		} else {
			// Static or package-level call
			typeObj := pkg.TypesInfo.Uses[sel]
			if typeObj != nil {
				fmt.Printf("%sFunction: %s (package-level: %s)\n", indentSpaces(indent), sel.Name, typeObj.Pkg().Path())
				if condition(sel, typeObj) {
					fmt.Println(">>> STATUS ERROR FOUND!")
					return
				}

				if fnDecl, ok := typeObj.(*types.Func); ok {
					traverseFunctionBody(fnDecl, pkg, indent+2, condition)
				}
			}
		}
	default:
		fmt.Printf("%sUnknown function type: %T\n", indentSpaces(indent), fun)
	}

	// Recursively inspect arguments for nested calls
	for _, arg := range call.Args {
		if nestedCall, ok := arg.(*ast.CallExpr); ok {
			TraverseCallStack(nestedCall, pkg, indent+2, condition)
		}
	}
}

// traverseFunctionBody analyzes the body of a function or method to find further calls.
func traverseFunctionBody(fn *types.Func, pkg *packages.Package, indent int, condition func(*ast.Ident, types.Object) bool) {
	// Find the declaration of the function in the AST
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			funcDecl, ok := node.(*ast.FuncDecl)
			if !ok || pkg.Fset.Position(funcDecl.Pos()).Filename != pkg.Fset.Position(fn.Pos()).Filename {
				return true // Not the target function, continue
			}
			if funcDecl.Name.Name == fn.Name() {
				// Found the function, inspect its body for calls
				ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok {
						TraverseCallStack(call, pkg, indent, condition)
					}
					return true
				})
				return false // Stop after finding the target function
			}
			return true
		})
	}
}

// Helper function to generate indentation
func indentSpaces(indent int) string {
	return strings.Repeat(" ", indent)
}
