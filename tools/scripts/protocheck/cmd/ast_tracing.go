package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

const grpcStatusImportPath = "google.golang.org/grpc/status"

var TRACE bool

// TODO_IN_THIS_COMMIT: move & godoc...
func walkFuncBody(
	pkg *packages.Package,
	pkgs []*packages.Package,
	shouldAppend,
	shouldExhonerate bool,
) func(ast.Node) bool {
	return func(n ast.Node) bool {
		if n == nil {
			return false
		}

		logger.Debug().
			Str("position", fmt.Sprintf(" %s ", pkg.Fset.Position(n.Pos()).String())).
			Str("node_type", fmt.Sprintf("%T", n)).
			Bool("shouldAppend", shouldAppend).
			Msg("walking function body")

		//position := pkg.Fset.Position(n.Pos())
		//logger.Warn().Msgf("position: %s", position.String())
		//
		//inspectPos := pkg.Fset.Position(n.Pos())
		//fmt.Printf(">>> inspecting %T at %s\n", n, inspectPos)
		//
		//inspectLocation := inspectPos.String()
		//if inspectLocation == "/home/bwhite/Projects/pokt/poktroll/x/proof/keeper/msg_server_create_claim.go:44:3" {
		//	fmt.Printf(">>> found it!\n")
		//}

		switch n := n.(type) {
		//case *ast.BlockStmt:
		//	return true
		//// Search for a return statement.
		//case *ast.IfStmt:
		//	return true
		case *ast.ReturnStmt:
			lastResult := n.Results[len(n.Results)-1]
			inspectPosition := pkg.Fset.Position(lastResult.Pos()).String()

			logger := logger.With(
				"node_type", fmt.Sprintf("%T", lastResult),
				"inspectPosition", fmt.Sprintf(" %s ", inspectPosition),
			)

			logger.Debug().Msgf("lastResult: %+v", lastResult)

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
					logger.Warn().Msg("lastReturnArgNode.Obj is nil")
					return true
				}

				def := pkg.TypesInfo.Uses[lastReturnArgNode]
				if def == nil {
					logger.Warn().Msg("def is nil")
					return true
				}

				if def.Type().String() != "error" {
					logger.Warn().Msg("def is not error")
					//inspectPosition := pkg.Fset.Position(lastReturnArgNode.Pos()).String()
					//break
					return false
				}

				if shouldAppend {
					logger.Debug().Msg("appending potential offending line")
					appendOffendingLine(inspectPosition)
				}
				traceExpressionStack(lastReturnArgNode, pkgs, nil, pkg, lastReturnArgNode, offendingPkgErrLineSet)
				return true

			// `return nil, types.ErrXXX.Wrapf(...)` <-- last arg is a *ast.CallExpr.
			case *ast.CallExpr:
				//logger.Debug().Msg("inspecting ast node")
				if shouldAppend {
					logger.Debug().Msg("appending potential offending line")
					appendOffendingLine(inspectPosition)
				}
				traceExpressionStack(lastReturnArgNode, pkgs, nil, pkg, lastReturnArgNode, offendingPkgErrLineSet)
				//TraverseCallStack(lastReturnArgNode, pkgs, 0, condition(lastReturnArgNode))
				//return false
				return true

			case *ast.SelectorExpr:
				if shouldAppend {
					logger.Debug().Msg("appending potential offending line")
					appendOffendingLine(inspectPosition)
				}
				traceSelectorExpr(lastReturnArgNode, pkg, pkgs, lastReturnArgNode, offendingPkgErrLineSet)
				return true
			}

		default:
			//logger.Warn().Str("node_types", fmt.Sprintf("%T", n)).Msg("NOT traversing ast node")
			return true
		}

		//logger.Debug().Msg("appending potential offending line")
		//appendOffendingLine(inspectPosition)
		//
		//return true

		return true
	}
}

// Helper function to trace selector expressions
func traceSelectorExpr(
	expr *ast.SelectorExpr,
	//scopeNode ast.Node,
	candidatePkg *packages.Package,
	pkgs []*packages.Package,
	candidateNode ast.Node,
	offendingPositions map[string]struct{},
) bool {
	logger.Debug().Msg("tracing selector expression")
	//fmt.Println(">>>>>>>>> TRACE SELECTOR EXPR")
	for _, pkg := range pkgs {
		if selection := pkg.TypesInfo.Selections[expr]; selection != nil {
			//logger.Warn().Msgf("<<<<<<< has selection: %s", selection.String())
			for _, pkg2 := range pkgs {
				position := pkg2.Fset.Position(selection.Obj().Pos())

				var foundNode ast.Node
				for _, fileNode := range pkg2.Syntax {
					foundNode = findNodeByPosition(pkg2.Fset, fileNode, position)
					if foundNode != nil {
						//traceExpressionStack(foundNode, pkgs, expr, pkg, foundNode, offendingPositions)
						logger.Warn().
							Str("node_type", fmt.Sprintf("%T", foundNode)).
							Str("selection_position", fmt.Sprintf(" %s ", position)).
							Str("expr_position", fmt.Sprintf(" %s ", pkg.Fset.Position(expr.Pos()).String())).
							Str("found_node_position", fmt.Sprintf(" %s ", pkg2.Fset.Position(foundNode.Pos()).String())).
							Msg("found node")
						//fmt.Printf(">>>>>>>>>> found node %T %+v\n", foundNode, foundNode)
						//traceExpressionStack(foundNode.(*ast.Ident), pkgs, expr, pkg2, foundNode, offendingPositions)
						var declNode *ast.FuncDecl
						ast.Inspect(fileNode, func(n ast.Node) bool {
							if declNode != nil {
								return false
							}

							if decl, ok := n.(*ast.FuncDecl); ok {
								if decl.Name.Name == foundNode.(*ast.Ident).Name &&
									decl.Pos() < foundNode.Pos() &&
									foundNode.Pos() <= decl.End() {
									declNode = decl
									return false
								}
							}
							return true
						})

						if declNode != nil {
							logger.Warn().Str("decl_position", pkg2.Fset.Position(declNode.Pos()).String()).Msg("tracing decl node")
							logger.Warn().Str("decl_body", pkg2.Fset.Position(declNode.Body.Pos()).String()).Msg("tracing decl node body")
							ast.Inspect(declNode.Body, walkFuncBody(pkg, pkgs, false, false))
							//walkFuncBody(pkg, pkgs)(declNode.Body)
						} else {
							logger.Warn().Msg("could not find decl node")
						}

						//return false
						return true
					}
				}
			}
			return true
		}
	}

	// TODO_IN_THIS_COMMIT: refactor; below happens when the selector is not found within any package.

	// Resolve the base expression
	switch x := expr.X.(type) {
	case *ast.Ident: // e.g., `pkg.Func`
		for _, pkg := range pkgs {
			if obj := pkg.TypesInfo.Uses[x]; obj != nil {
				pkgParts := strings.Split(obj.String(), " ")

				var pkgStr string
				switch {
				// e.g., package (error) google.golang.org/grpc/status
				case strings.HasPrefix(obj.String(), "package ("):
					pkgStr = pkgParts[2]
				// e.g. package fmt
				default:
					pkgStr = pkgParts[1]
				}
				//fmt.Printf(">>> pkgStr: %s\n", pkgStr)

				logger := logger.With(
					"node_type", fmt.Sprintf("%T", x),
					"position", fmt.Sprintf(" %s ", pkg.Fset.Position(x.Pos()).String()),
					"package", strings.Trim(pkgStr, "\"()"),
				)
				logger.Debug().Msg("tracing selector expression")

				//if obj := pkg.TypesInfo.Uses[x]; obj A!= nil {
				//fmt.Printf("Base identifier %s resolved to: %s\n", x.Name, obj)
				//fmt.Printf(">>> obj.String(): %s\n", obj.String())
				//fmt.Printf(">>> strings.Contains(obj.String(), grpcStatusImportPath): %v\n", strings.Contains(obj.String(), grpcStatusImportPath))
				isMatch := strings.Contains(obj.String(), grpcStatusImportPath) &&
					expr.Sel.Name == "Error"
				//if isMatch {
				if isMatch {
					candidateNodePosition := pkg.Fset.Position(candidateNode.Pos()).String()
					//inspectionPosition := pkg.Fset.Position(x.Pos()).String()
					//fmt.Printf("Found target function %s in selector at %s\n", expr.Sel.Name, pkg.Fset.Position(expr.Pos()))
					//fmt.Printf("Found offending candidate %s in selector at %s\n", expr.Sel.Name, pkg.Fset.Position(candidateNode.Pos()).String())
					//fmt.Printf("!!! - offendingPositions: %+v\n", offendingPositions)
					//fmt.Printf("!!! - candPosition: %s", currentPosition)
					//logger.Debug().
					//	Str("candidate_pos", fmt.Sprintf(" %s ", candidateNodePosition)).
					//	//Str("offending_positions", fmt.Sprintf("%+v", offendingPositions)).
					//	Send()
					if _, ok := offendingPositions[candidateNodePosition]; ok {
						logger.Debug().Msgf("exhonerating %s", candidateNodePosition)
						delete(offendingPositions, candidateNodePosition)
					} else {
						logger.Warn().Msgf("can't exhonerating %s", candidateNodePosition)
					}
					return false
				}

				//traceSelectorExpr(expr.Sel, candidatePkg, pkgs, candidateNode, offendingPositions)

				//}
				//case *ast.SelectorExpr: // e.g., `obj.Method.Func`
				//	if traceSelectorExpr(x, info, fset) {
				//		return true
				//	}
			} else if obj = pkg.TypesInfo.Defs[x]; obj != nil {
				logger.Warn().Msgf("no use but def: %+v", obj)
			} else if obj = pkg.TypesInfo.Defs[expr.Sel]; obj != nil {
				logger.Warn().
					Str("pkg_path", pkg.PkgPath).
					Str("name", expr.Sel.Name).
					Msgf("sel def")
				traceExpressionStack(expr.Sel, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
				//} else {
				//	logger.Warn().Msgf("no use or def: %+v, sel: %+v", x, expr.Sel)
			}
		}
	case *ast.SelectorExpr: // e.g., `obj.Method.Func`
		logger.Debug().Msgf("tracing recursive selector expression: %+v", expr)
		return traceSelectorExpr(x, candidatePkg, pkgs, candidateNode, offendingPositions)
	case *ast.CallExpr:
		logger.Debug().Msgf("tracing call expression: %+v", expr)
		//switch callExpr := x.Fun.(type) {
		//case *ast.SelectorExpr: // e.g., `obj.Method.Func`
		//	return traceSelectorExpr(callExpr, pkgs, candidateNode, offendingPositions)
		//default:
		//	logger.Warn().Msgf("skipping sub-selector call expression X type: %T", x)
		//}
		traceExpressionStack(x.Fun, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
	default:
		logger.Warn().Msgf("skipping selector expression X type: %T", x)
	}
	return true
}

// Trace any expression recursively, including selector expressions
func traceExpressionStack(
	exprToTrace ast.Expr,
	pkgs []*packages.Package,
	_ ast.Node,
	candidatePkg *packages.Package,
	candidateNode ast.Node,
	offendingPositions map[string]struct{},
) bool {
	if exprToTrace == nil {
		return false
	}

	logger := logger.With(
		"node_type", fmt.Sprintf("%T", exprToTrace),
		//"position", candidatePkg.Fset.Position(exprToTrace.Pos()).String(),
		//"package", strings.Trim(strings.Split(obj.String(), " ")[2], "\"()"),
	)
	logger.Debug().Msg("tracing expression stack")

	switch expr := exprToTrace.(type) {
	case nil:
		return false
	case *ast.CallExpr:
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			//logger.Debug().Msg("tracing selector expression")
			return traceSelectorExpr(sel, candidatePkg, pkgs, candidateNode, offendingPositions)
		}
		//logger.Debug().Msgf("tracing expression args: %+v", expr)
		//for _, arg := range expr.Args {
		//	// TODO_IN_THIS_COMMIT: return traceExpressionStack... ?
		//	traceExpressionStack(arg, pkgs, candidateNode, offendingPositions)
		//	//return true
		//}
		return true
	case *ast.BinaryExpr:
		logger.Debug().Msg("tracing binary expression")
		// TODO_IN_THIS_COMMIT: return traceExpressionStack... ?
		if traceExpressionStack(expr.X, pkgs, expr, candidatePkg, candidateNode, offendingPositions) {
			traceExpressionStack(expr.Y, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
			//return true
		}
		return true
	case *ast.ParenExpr:
		logger.Debug().Msg("tracing paren expression")
		return traceExpressionStack(expr.X, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
	case *ast.SelectorExpr:
		logger.Debug().Msg("tracing selector expression")
		return traceSelectorExpr(expr, candidatePkg, pkgs, candidateNode, offendingPositions)
	case *ast.Ident:
		logger.Debug().Str("name", expr.Name).Msg("tracing ident")
		//def := candidatePkg.TypesInfo.Defs[expr]
		// TODO_IN_THIS_COMMIT: handle no def...
		//x := def.Parent().Lookup(expr.Name)
		// TODO_IN_THIS_COMMIT: handle no lookup...
		//x.Pos()

		//var candidateFileNode ast.Node
		//for _, fileNode := range candidatePkg.Syntax {
		//	ast.Inspect(fileNode, func(n ast.Node) bool {
		//		if n == candidateNode {
		//			candidateFileNode = fileNode
		//			return false
		//		}
		//		return true
		//	})
		//}
		//for _, pkg := range pkgs {
		//	for _, fileNode := range pkg.Syntax {
		//declOrAssign, _ := findDeclOrAssign(expr, candidateFileNode, candidatePkg)
		//declOrAssign, _ := findDeclOrAssign(expr, expr, candidatePkg)
		//declOrAssign, declOrAssignPos := findDeclOrAssign(expr, scopeNode, candidatePkg)

		// TODO_IN_THIS_COMMIT: return a slice of all decls and assignments
		// and their respective files/pkgs.
		declOrAssign, _ := newFindDeclOrAssign(expr, candidatePkg)
		if declOrAssign == nil {
			logger.Warn().Msgf("no declaration or assignment found for ident %q", expr.String())
			return false
		}
		//logger.Debug().
		//	Str("pkg_path", candidatePkg.PkgPath).
		//	//Str("file_path", fmt.Sprintf(" %s", candidatePkg.Fset.File(candidateFileNode.Pos()).Name())).
		//	Str("decl_or_assign_pos", fmt.Sprintf(" %s ", declOrAssignPos)).
		//	Send()
		//Msgf("found decl or assign: %+v", declOrAssign)
		switch doa := declOrAssign.(type) {
		case ast.Expr:
			traceExpressionStack(doa, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
		case *ast.AssignStmt:
			logger.Warn().Msgf(">>>>>>> assign stmt: %+v", doa)
			// TODO_IN_THIS_COMMIT: what about len(Rhs) > 1?
			traceExpressionStack(doa.Rhs[0], pkgs, expr, candidatePkg, candidateNode, offendingPositions)
		case *ast.ValueSpec:
			// TODO_RESUME_HERE!!!!
			// TODO_RESUME_HERE!!!!
			// TODO_RESUME_HERE!!!!
			// TODO_RESUME_HERE!!!!
			//
			// find "closest" previous assignment...
			//
			logger.Warn().
				//Str("position", fmt.Sprintf(" %s ", pkg.Fset.Position(doa.Pos()).String())).
				Int("len(values)", len(doa.Values)).
				Msgf(">>>>>>> value spec: %+v", doa)

			if doa.Values != nil {
				for _, value := range doa.Values {
					traceExpressionStack(value, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
				}
			}
		default:
			logger.Warn().Msgf("unknown node type 3: %T", doa)
		}
		//}
		//}
		return true
	//case *ast.SliceExpr:
	//	logger.Debug().Msgf("tracing slice expression: %+v", expr)
	//	return true
	default:
		logger.Warn().Msgf("unknown node type 2: %T", exprToTrace)
		return true
	}
}

//func newNewFindDeclOrAssign(
//	targetIdent *ast.Ident,
//	scopeNode ast.Node,
//	pkg *packages.Package,
//) (declNode ast.Node, declPos token.Position) {
//	var nodes []ast.Node
//	ast.Inspect(scopeNode, func(n ast.Node) bool {
//		if n != nil {
//			nodes = append(nodes, n)
//		}
//		return true
//	})
//
//	for i := len(nodes) - 1; i >= 0; i-- {
//
//	}
//}

// TODO_IN_THIS_COMMIT: move & godoc...
func newFindDeclOrAssign(
	targetIdent *ast.Ident,
	//pkgs []*packages.Package,
	pkg *packages.Package,
) (declNode ast.Node, declPos token.Position) {
	//var closestDeclNode ast.Node

	for _, fileNode := range pkg.Syntax {
		if declNode != nil {
			return declNode, declPos
		}

		//fmt.Println(">>>>>>>>> NEW FILE NODE")
		//ast.Inspect(fileNode, func(n ast.Node) bool {
		//	//if declNode != nil {
		//	//	//fmt.Println(">>>>>>>>> EXITING EARLY")
		//	//	return false
		//	//}
		//
		//	if ident, ok := n.(*ast.Ident); ok &&
		//		ident.Name == targetIdent.Name {
		//if obj := pkg.TypesInfo.Defs[targetIdent]; obj != nil {
		//	declPos = pkg.Fset.Position(obj.Pos())
		//	//logger.Debug().Fields(map[string]any{
		//	//	//"pkg_path":   pkg.PkgPath,
		//	//	"file_path":  fmt.Sprintf(" %s ", pkg.Fset.File(fileNode.Pos()).Name()),
		//	//	"target_pos": fmt.Sprintf(" %s ", pkg.Fset.Position(targetIdent.Pos()).String()),
		//	//	"decl_pos":   fmt.Sprintf(" %s ", declPos.String()),
		//	//}).Msg("defs")
		//	declNode = findNodeByPosition(pkg.Fset, fileNode, declPos)
		//	return false
		//} else if obj := pkg.TypesInfo.Uses[targetIdent]; obj != nil {
		if obj := pkg.TypesInfo.Uses[targetIdent]; obj != nil {
			// TODO_IN_THIS_COMMIT: figure out why this is called so frequently.
			logger.Debug().Fields(map[string]any{
				//"pkg_path":   pkg.PkgPath,
				"file_path":  fmt.Sprintf(" %s ", pkg.Fset.File(fileNode.Pos()).Name()),
				"target_pos": fmt.Sprintf(" %s ", pkg.Fset.Position(targetIdent.Pos()).String()),
				"decl_pos":   fmt.Sprintf(" %s ", pkg.Fset.Position(obj.Pos()).String()),
			}).Msg("uses")
			declPos = pkg.Fset.Position(obj.Pos())
			declNode = findNodeByPosition(pkg.Fset, fileNode, declPos)
			logger.Warn().
				Str("decl_node", fmt.Sprintf("%+v", declNode)).
				Str("decl_pos", fmt.Sprintf(" %s ", declPos)).
				Msg("found decl node")
			//return false
		}
		//	}
		//	return true
		//})
	}
	//fmt.Println(">>>>>>>>> DONE")

	return declNode, declPos
}

// TODO_IN_THIS_COMMIT: move & godoc...
// search for targetIdent by position
func findNodeByPosition(
	fset *token.FileSet,
	fileNode *ast.File,
	position token.Position,
) (targetNode ast.Node) {
	//fmt.Println(">>>>>>>>> FIND NODE BY POSITION")

	ast.Inspect(fileNode, func(n ast.Node) bool {
		if targetNode != nil {
			return false
		}

		if n == nil {
			return true
		}

		if n != nil && fset.Position(n.Pos()) == position {
			targetNode = n
			return false
		}

		if targetNode != nil {
			return false
		}

		return true
	})
	return targetNode
}

// Find the declaration or assignment of an identifier
// func findDeclOrAssign(ident *ast.Ident, fileNode ast.Node, pkg *packages.Package) (ast.Expr, *token.Position) {
func findDeclOrAssign(ident *ast.Ident, scopeNode ast.Node, pkg *packages.Package) (ast.Expr, *token.Position) {
	//logger.Debug().Msg("finding decl or assign")

	//fmt.Println("!!!! findDeclOrAssign begin")
	var declOrAssign ast.Expr
	var foundPos token.Position

	//fmt.Println("!!!! findDeclOrAssign inspect")
	//for _, fileNode := range pkg.Syntax {
	//ast.Inspect(fileNode, func(n ast.Node) bool {
	ast.Inspect(scopeNode, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt: // Look for assignments
			//fmt.Println("!!!! findDeclOrAssign case assign")
			for i, lhs := range stmt.Lhs {
				if lhsIdent, ok := lhs.(*ast.Ident); ok && lhsIdent.Name == ident.Name {
					//fmt.Printf("len(rhs): %d len(lhs): %d\n", len(stmt.Rhs), len(stmt.Lhs))
					if len(stmt.Lhs) != len(stmt.Rhs) {
						declOrAssign = stmt.Rhs[0]
					} else {
						declOrAssign = stmt.Rhs[i]
					}
					foundPos = pkg.Fset.Position(stmt.Pos())
				}
			}
		case *ast.ValueSpec: // Look for declarations with initialization
			//fmt.Println("!!!! findDeclOrAssign case value")
			for i, name := range stmt.Names {
				if name.Name == ident.Name && i < len(stmt.Values) {
					declOrAssign = stmt.Values[i]
					foundPos = pkg.Fset.Position(stmt.Pos())
					return false
				}
			}
		}
		return true
	})
	//}

	return declOrAssign, &foundPos
}

// TODO_IN_THIS_COMMIT: move & godoc...
//func getNodeFromPosition(fset *token.FileSet, position token.Position) ast.Node {
//	file := fset.File(position)
//	if file == nil {
//		return nil
//	}
//
//	var node ast.Node
//	ast.Inspect(file, func(n ast.Node) bool {
//		if n == nil {
//			return false
//		}
//
//		if fset.Position(n.Pos()).String() == position.String() {
//			node = n
//			return false
//		}
//
//		return true
//	})
//
//	return node
//}
