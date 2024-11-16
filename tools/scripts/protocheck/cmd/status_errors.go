package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"

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
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo,
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
	}
	// TODO_IN_THIS_COMMIT: assert only 1 pkg: module's keeper...
	typeInfo := pkgs[0].TypesInfo
	// --- END

	msgServerGlob := filepath.Join(keeperDir, "msg_server_*.go")

	matches, err := filepath.Glob(msgServerGlob)
	if err != nil {
		return err
	}

	// TODO_IN_THIS_COMMIT: extract --- BEGIN
	for _, matchFilePath := range matches {
		fset := token.NewFileSet()

		astFile, err := parser.ParseFile(fset, matchFilePath, nil, parser.AllErrors)
		if err != nil {
			return err
		}

		//fmt.Println("BEFORE...")
		//typeInfo, err := getTypeInfo(fset, matchFilePath, astFile)
		//if err != nil {
		//	return err
		//}
		////typeInfo := types.Info{}
		//fmt.Println("AFTER...")

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

			fmt.Printf("Found msgServer method %q in %s\n", fnNode.Name.Name, matchFilePath)

			// Recursively traverse the function body, looking for non-nil error returns.
			var errorReturns []*ast.IfStmt
			// TODO_IN_THIS_COMMIT: extract --- BEGIN
			ast.Inspect(fnNode.Body, func(n ast.Node) bool {
				// Search for a return statement.
				rtrnStmt, ok := n.(*ast.ReturnStmt)
				if !ok {
					return true
				}

				ifStmt, ok := n.(*ast.IfStmt)
				if !ok {
					// Skip AST branches which are not logically conditional branches.
					//fmt.Println("non if")
					return true
				}
				//fmt.Println("yes if")

				// Match on `if err != nil` statements.
				// TODO_IN_THIS_COMMIT: extract --- BEGIN
				if ifStmt.Cond == nil {
					return false
				}

				errorReturn, ok := ifStmt.Cond.(*ast.BinaryExpr)
				if !ok {
					return false
				}

				if errorReturn.Op != token.NEQ {
					return false
				}

				// Check that the left operand is an error type.
				// TODO_IN_THIS_COMMIT: extract --- BEGIN
				errIdentNode, ok := errorReturn.X.(*ast.Ident)
				if !ok {
					return false
				}

				//errIdentNode.Obj.Kind.String()
				obj := typeInfo.Uses[errIdentNode]
				fmt.Sprintf("obj: %+v", obj)
				// --- END
				// --- END

				errorReturns = append(errorReturns, ifStmt)

				return false
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
		})
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
