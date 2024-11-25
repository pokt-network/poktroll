package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

const grpcStatusImportPath = "google.golang.org/grpc/status"

// Helper function to trace selector expressions
func traceSelectorExpr(
	expr *ast.SelectorExpr,
	pkgs []*packages.Package,
	candidateNode ast.Node,
	offendingPositions map[string]struct{},
) bool {
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
					}
					return false
				}
				//}
				//case *ast.SelectorExpr: // e.g., `obj.Method.Func`
				//	if traceSelectorExpr(x, info, fset) {
				//		return true
				//	}
			} else if obj = pkg.TypesInfo.Defs[x]; obj != nil {
				logger.Warn().Msgf("no use but def: %+v", obj)
			}
		}
	case *ast.SelectorExpr: // e.g., `obj.Method.Func`
		return traceSelectorExpr(x, pkgs, candidateNode, offendingPositions)
	case *ast.CallExpr:
		logger.Debug().Msgf("tracing call expression: %+v", expr)
		switch callExpr := x.Fun.(type) {
		case *ast.SelectorExpr: // e.g., `obj.Method.Func`
			return traceSelectorExpr(callExpr, pkgs, candidateNode, offendingPositions)
		default:
			logger.Warn().Msgf("skipping sub-selector call expression X type: %T", x)
		}
	default:
		logger.Warn().Msgf("skipping selector expression X type: %T", x)
	}
	return true
}

// Trace any expression recursively, including selector expressions
func traceExpressionStack(
	exprToTrace ast.Expr,
	pkgs []*packages.Package,
	candidateNode ast.Node,
	offendingPositions map[string]struct{},
) bool {
	if exprToTrace == nil {
		return false
	}

	logger := logger.With(
		"node_type", fmt.Sprintf("%T", exprToTrace),
		//"position", pkg.Fset.Position(x.Pos()).String(),
		//"package", strings.Trim(strings.Split(obj.String(), " ")[2], "\"()"),
	)
	logger.Debug().Msg("tracing expression stack")

	switch expr := exprToTrace.(type) {
	case nil:
		return false
	case *ast.CallExpr:
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			//logger.Debug().Msg("tracing selector expression")
			return traceSelectorExpr(sel, pkgs, candidateNode, offendingPositions)
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
		if traceExpressionStack(expr.X, pkgs, candidateNode, offendingPositions) {
			traceExpressionStack(expr.Y, pkgs, candidateNode, offendingPositions)
			//return true
		}
		return true
	case *ast.ParenExpr:
		logger.Debug().Msg("tracing paren expression")
		return traceExpressionStack(expr.X, pkgs, candidateNode, offendingPositions)
	case *ast.SelectorExpr:
		logger.Debug().Msg("tracing selector expression")
		return traceSelectorExpr(expr, pkgs, candidateNode, offendingPositions)
	case *ast.Ident:
		logger.Debug().Msg("tracing ident")
		//fmt.Printf(">>> exprToTrace: %+v\n", expr)
		//var srcPkg *packages.Package
		for _, pkg := range pkgs {
			//if obj := pkg.TypesInfo.Defs[expr]; obj != nil {
			//	srcPkg = pkg
			for _, fileNode := range pkg.Syntax {
				declOrAssign, _ := findDeclOrAssign(expr, fileNode, pkg.Fset)
				if declOrAssign == nil {
					continue
				}
				logger.Debug().
					Str("pkg_path", pkg.PkgPath).
					Str("file_path", pkg.Fset.File(fileNode.Pos()).Name()).
					Str("decl_or_assign_pos", pkg.Fset.Position(declOrAssign.Pos()).String()).
					Send()
				//Msgf("found decl or assign: %+v", declOrAssign)
				traceExpressionStack(declOrAssign, pkgs, candidateNode, offendingPositions)
			}
			//}
		}
		//if srcPkg == nil {
		//	logger.Warn().Msgf("no pkg found for expr: %+v", expr)
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

// Find the declaration or assignment of an identifier
func findDeclOrAssign(ident *ast.Ident, node ast.Node, fset *token.FileSet) (ast.Expr, *token.Position) {
	//logger.Debug().Msg("finding decl or assign")

	//fmt.Println("!!!! findDeclOrAssign begin")
	var declOrAssign ast.Expr
	var foundPos token.Position

	//fmt.Println("!!!! findDeclOrAssign inspect")
	ast.Inspect(node, func(n ast.Node) bool {
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
					foundPos = fset.Position(stmt.Pos())
				}
			}
		case *ast.ValueSpec: // Look for declarations with initialization
			//fmt.Println("!!!! findDeclOrAssign case value")
			for i, name := range stmt.Names {
				if name.Name == ident.Name && i < len(stmt.Values) {
					declOrAssign = stmt.Values[i]
					foundPos = fset.Position(stmt.Pos())
					return false
				}
			}
		}
		return true
	})

	return declOrAssign, &foundPos
}
