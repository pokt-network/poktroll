package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

const grpcStatusImportPath = "google.golang.org/grpc/status"

// TODO_IN_THIS_COMMIT: move & godoc...
func walkFuncBody(
	pkg *packages.Package,
	pkgs []*packages.Package,
	shouldAppend bool,
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

		switch n := n.(type) {
		case *ast.ReturnStmt:
			lastResult := n.Results[len(n.Results)-1]
			inspectPosition := pkg.Fset.Position(lastResult.Pos()).String()

			logger := logger.With(
				"node_type", fmt.Sprintf("%T", lastResult),
				"inspectPosition", fmt.Sprintf(" %s ", inspectPosition),
			)

			logger.Debug().Msgf("lastResult: %+v", lastResult)

			switch lastReturnArgNode := lastResult.(type) {
			// E.g. `return nil, err` <-- last arg is an *ast.Ident.
			case *ast.Ident:
				// DEV_NOTE: No need to check that the last return arg is an error type
				// if we checked that the function returns an error as the last arg.
				if lastReturnArgNode.Obj == nil {
					logger.Debug().Msg("lastReturnArgNode.Obj is nil")
					return true
				}

				def := pkg.TypesInfo.Uses[lastReturnArgNode]
				if def == nil {
					logger.Debug().Msg("def is nil")
					return true
				}

				if def.Type().String() != "error" {
					logger.Debug().Msg("def is not error")
					return false
				}

				if shouldAppend {
					logger.Debug().Msg("appending potential offending line")
					appendOffendingLine(inspectPosition)
				}
				traceExpressionStack(lastReturnArgNode, pkgs, nil, pkg, lastReturnArgNode, offendingPkgErrLineSet)
				return true

			// E.g. `return nil, types.ErrXXX.Wrapf(...)` <-- last arg is a *ast.CallExpr.
			case *ast.CallExpr:
				if shouldAppend {
					logger.Debug().Msg("appending potential offending line")
					appendOffendingLine(inspectPosition)
				}
				traceExpressionStack(lastReturnArgNode, pkgs, nil, pkg, lastReturnArgNode, offendingPkgErrLineSet)
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
			return true
		}

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
	for _, pkg := range pkgs {
		if selection := pkg.TypesInfo.Selections[expr]; selection != nil {
			for _, pkg2 := range pkgs {
				position := pkg2.Fset.Position(selection.Obj().Pos())

				var foundNode ast.Node
				for _, fileNode := range pkg2.Syntax {
					foundNode = findNodeByPosition(pkg2.Fset, fileNode, position)
					if foundNode != nil {
						logger.Debug().
							Str("node_type", fmt.Sprintf("%T", foundNode)).
							Str("selection_position", fmt.Sprintf(" %s ", position)).
							Str("expr_position", fmt.Sprintf(" %s ", pkg.Fset.Position(expr.Pos()).String())).
							Str("found_node_position", fmt.Sprintf(" %s ", pkg2.Fset.Position(foundNode.Pos()).String())).
							Msg("found node")

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
							logger.Debug().Str("decl_position", pkg2.Fset.Position(declNode.Pos()).String()).Msg("tracing decl node")
							logger.Debug().Str("decl_body", pkg2.Fset.Position(declNode.Body.Pos()).String()).Msg("tracing decl node body")
							ast.Inspect(declNode.Body, walkFuncBody(pkg, pkgs, false))
						} else {
							logger.Debug().Msg("could not find decl node")
						}

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

				logger := logger.With(
					"node_type", fmt.Sprintf("%T", x),
					"position", fmt.Sprintf(" %s ", pkg.Fset.Position(x.Pos()).String()),
					"package", strings.Trim(pkgStr, "\"()"),
				)
				logger.Debug().Msg("tracing selector expression")

				isMatch := strings.Contains(obj.String(), grpcStatusImportPath) &&
					expr.Sel.Name == "Error"
				if isMatch {
					candidateNodePosition := pkg.Fset.Position(candidateNode.Pos()).String()
					if _, ok := offendingPositions[candidateNodePosition]; ok {
						logger.Debug().Msgf("exhonerating %s", candidateNodePosition)
						delete(offendingPositions, candidateNodePosition)
					} else {
						logger.Warn().Msgf("can't exhonerate %s", candidateNodePosition)
					}
					return false
				}
			} else if obj = pkg.TypesInfo.Defs[x]; obj != nil {
				logger.Debug().Msgf("no use but def: %+v", obj)
			} else if obj = pkg.TypesInfo.Defs[expr.Sel]; obj != nil {
				logger.Debug().
					Str("pkg_path", pkg.PkgPath).
					Str("name", expr.Sel.Name).
					Msgf("sel def")
				traceExpressionStack(expr.Sel, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
			}
		}
	case *ast.SelectorExpr: // e.g., `obj.Method.Func`
		logger.Debug().Msgf("tracing recursive selector expression: %+v", expr)
		return traceSelectorExpr(x, candidatePkg, pkgs, candidateNode, offendingPositions)
	case *ast.CallExpr:
		logger.Debug().Msgf("tracing call expression: %+v", expr)
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

	logger := logger.With("node_type", fmt.Sprintf("%T", exprToTrace))
	logger.Debug().Msg("tracing expression stack")

	switch expr := exprToTrace.(type) {
	case nil:
		return false
	case *ast.CallExpr:
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			return traceSelectorExpr(sel, candidatePkg, pkgs, candidateNode, offendingPositions)
		}
		return true
	case *ast.BinaryExpr:
		logger.Debug().Msg("tracing binary expression")
		// TODO_IN_THIS_COMMIT: return traceExpressionStack... ?
		if traceExpressionStack(expr.X, pkgs, expr, candidatePkg, candidateNode, offendingPositions) {
			traceExpressionStack(expr.Y, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
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

		// TODO_IN_THIS_COMMIT: return a slice of all decls and assignments
		// and their respective files/pkgs.
		declOrAssign, _ := newFindDeclOrAssign(expr, candidatePkg)
		if declOrAssign == nil {
			return false
		}

		switch doa := declOrAssign.(type) {
		case ast.Expr:
			traceExpressionStack(doa, pkgs, expr, candidatePkg, candidateNode, offendingPositions)
		case *ast.AssignStmt:
			traceExpressionStack(doa.Rhs[0], pkgs, expr, candidatePkg, candidateNode, offendingPositions)
		default:
			logger.Warn().Msgf("unknown node type 3: %T", doa)
		}
		return true
	default:
		logger.Warn().Msgf("unknown node type 2: %T", exprToTrace)
		return true
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func newFindDeclOrAssign(
	targetIdent *ast.Ident,
	pkg *packages.Package,
) (declNode ast.Node, declPos token.Position) {
	for _, fileNode := range pkg.Syntax {
		if declNode != nil {
			break
		}

		if obj := pkg.TypesInfo.Uses[targetIdent]; obj != nil {
			logger.Debug().Fields(map[string]any{
				"file_path":  fmt.Sprintf(" %s ", pkg.Fset.File(fileNode.Pos()).Name()),
				"target_pos": fmt.Sprintf(" %s ", pkg.Fset.Position(targetIdent.Pos()).String()),
				"decl_pos":   fmt.Sprintf(" %s ", pkg.Fset.Position(obj.Pos()).String()),
			}).Msg("uses")
			declPos = pkg.Fset.Position(obj.Pos())
			declNode = findNodeByPosition(pkg.Fset, fileNode, declPos)
			if declNode != nil {
				logger.Debug().
					Str("decl_node", fmt.Sprintf("%+v", declNode)).
					Str("decl_pos", fmt.Sprintf(" %s ", declPos)).
					Msg("found decl node")
			}
		}
	}

	// TODO_IN_THIS_COMMIT: improve comment...
	// Look through decl node to see if it contains a valudspec with values.
	// If it does, return the value(s).
	if declNode != nil {
		ast.Inspect(declNode, func(n ast.Node) bool {
			switch doa := n.(type) {
			case *ast.ValueSpec:
				logger.Debug().
					Int("len(values)", len(doa.Values)).
					Msgf(">>>>>>> value spec: %+v", doa)

				if doa.Values != nil {
					logger.Debug().Msg("dao.Values != nil")
					for _, value := range doa.Values {
						declPos = pkg.Fset.Position(value.Pos())
						declNode = value
					}
				} else {
					logger.Debug().Msg("dao.Values == nil")
					declNode = nil
				}
			}

			return true
		})
	} else {
		logger.Debug().Msgf("no declaration or assignment found for ident %q", targetIdent.String())
	}

	// TODO_IN_THIS_COMMIT: improve comment...
	// If it does not, search the package for
	// the ident and return the closest assignment.
	if declNode == nil {
		var assignsRhs []ast.Expr
		for _, fileNode := range pkg.Syntax {
			ast.Inspect(fileNode, func(n ast.Node) bool {
				if assign, ok := n.(*ast.AssignStmt); ok {
					for lhsIdx, lhs := range assign.Lhs {
						// TODO_TECHDEBT: Ignoring assignments via selectors for now.
						// E.g., `a.b = c` will not be considered.
						lhsIdent, lhsIsIdent := lhs.(*ast.Ident)
						if !lhsIsIdent {
							continue
						}

						if lhsIdent.Name != targetIdent.Name {
							continue
						}

						rhsIdx := 0
						if len(assign.Lhs) == len(assign.Rhs) {
							rhsIdx = lhsIdx
						}

						rhs := assign.Rhs[rhsIdx]
						assignsRhs = append(assignsRhs, rhs)
					}
				}
				return true
			})
		}

		if len(assignsRhs) > 0 {
			// TODO_IN_THIS_COMMIT: comment explaining what's going on here...
			slices.SortFunc[[]ast.Expr, ast.Expr](assignsRhs, func(a, b ast.Expr) int {
				aPos := pkg.Fset.Position(a.Pos())
				bPos := pkg.Fset.Position(b.Pos())

				if aPos.Filename == bPos.Filename {
					switch {
					case aPos.Line < bPos.Line:
						return -1
					case aPos.Line > bPos.Line:
						return 1
					default:
						return 0
					}
				} else {
					return 1
				}

			})

			// DeclNode is the closest assignment whose position is less than or equal to the declPos.
			var (
				closestAssignPos  token.Position
				closestAssignNode ast.Expr
				targetIdentPos    = pkg.Fset.Position(targetIdent.Pos())
			)
			for _, rhs := range assignsRhs {
				if rhs == nil {
					continue
				}

				// DEV_NOTE: using pkg here assumes that rhs is in the same file as targetIdent.
				// This SHOULD ALWAYS be the case for error type non-initialization declarations
				// (e.g. var err error). I.e. we SHOULD NEVER be assigning an error value directly
				// from aa pkg-level error variable.
				rhsPos := pkg.Fset.Position(rhs.Pos())
				switch {
				case rhsPos.Filename != targetIdentPos.Filename:
					// TODO_TECHDEBT: handle case where rhs ident is defined in a different file.
					logger.Debug().
						Str("assignment_position", rhsPos.String()).
						Msg("ignoring assignment from different file")
					continue
				case rhsPos.Line < targetIdentPos.Line:
					closestAssignPos = rhsPos
					closestAssignNode = rhs
				case rhsPos.Line == targetIdentPos.Line:
					if rhsPos.Column <= targetIdentPos.Column {
						closestAssignPos = rhsPos
						closestAssignNode = rhs
					}
				}
			}
			declPos = closestAssignPos
			declNode = closestAssignNode
		}
	}

	return declNode, declPos
}

// TODO_IN_THIS_COMMIT: move & godoc...
// search for targetIdent by position
func findNodeByPosition(
	fset *token.FileSet,
	fileNode *ast.File,
	position token.Position,
) (targetNode ast.Node) {
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
	var declOrAssign ast.Expr
	var foundPos token.Position

	ast.Inspect(scopeNode, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt: // Look for assignments
			for i, lhs := range stmt.Lhs {
				if lhsIdent, ok := lhs.(*ast.Ident); ok && lhsIdent.Name == ident.Name {
					if len(stmt.Lhs) != len(stmt.Rhs) {
						declOrAssign = stmt.Rhs[0]
					} else {
						declOrAssign = stmt.Rhs[i]
					}
					foundPos = pkg.Fset.Position(stmt.Pos())
				}
			}
		case *ast.ValueSpec: // Look for declarations with initialization
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

	return declOrAssign, &foundPos
}
