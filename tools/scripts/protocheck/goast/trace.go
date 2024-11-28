package goast

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const grpcStatusImportPath = "google.golang.org/grpc/status"

// TODO_IN_THIS_COMMIT: move & godoc...
// TODO_IN_THIS_COMMIT: detemine whether this actually needs to return anything.
func TraceSelectorExpr(
	ctx context.Context,
	expr *ast.SelectorExpr,
	candidatePkg *packages.Package,
	modulePkgs []*packages.Package,
	candidateNode ast.Node,
	exclude func(context.Context, string),
) bool {
	logger := polylog.Ctx(ctx).With("func", "TraceSelectorExpr")
	logger.Debug().Send()

	// Search for the selector expression in all module packages.
	// TODO_IN_THIS_COMMIT: is it expected and/or guaranteed that it will only be found in one pkg?
	var declNode *ast.FuncDecl
	for _, pkg := range modulePkgs {
		//if declNode != nil {
		//	return true
		//}

		selection := pkg.TypesInfo.Selections[expr]
		if selection == nil {
			continue
		}

		for _, pkg2 := range modulePkgs {
			selectionPos := pkg2.Fset.Position(selection.Obj().Pos())

			for _, fileNode := range pkg2.Syntax {
				selectionNode := FindNodeByPosition(pkg2.Fset, fileNode, selectionPos)
				if selectionNode == nil {
					continue
				}

				logger.Debug().
					Str("node_type", fmt.Sprintf("%T", selectionNode)).
					Str("selection_position", fmt.Sprintf(" %s ", selectionPos)).
					Str("expr_position", fmt.Sprintf(" %s ", pkg.Fset.Position(expr.Pos()).String())).
					Str("found_node_position", fmt.Sprintf(" %s ", pkg2.Fset.Position(selectionNode.Pos()).String())).
					Msg("found node")

				ast.Inspect(fileNode, func(n ast.Node) bool {
					if declNode != nil {
						return false
					}

					if decl, ok := n.(*ast.FuncDecl); ok {
						if decl.Name.Name == selectionNode.(*ast.Ident).Name &&
							decl.Pos() <= selectionNode.Pos() &&
							selectionNode.Pos() <= decl.End() {
							declNode = decl
							return false
						}
					}
					return true
				})

				if declNode != nil {
					logger.Debug().Str("decl_position", pkg2.Fset.Position(declNode.Pos()).String()).Msg("tracing decl node")
					logger.Debug().Str("decl_body", pkg2.Fset.Position(declNode.Body.Pos()).String()).Msg("tracing decl node body")
					ast.Inspect(declNode.Body, NewInspectLastReturnArgFn(ctx, pkg, modulePkgs, nil, nil))
				} else {
					logger.Debug().Msg("could not find decl node")
				}

			}
		}

		// TODO_IN_THIS_COMMIT: note early return...
		return true
	}

	// TODO_IN_THIS_COMMIT: refactor; below happens when the selector is not found within any module package.

	// Resolve the base expression
	switch x := expr.X.(type) {
	case *ast.Ident: // e.g., `pkg.Func`
		for _, pkg := range modulePkgs {
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
					exclude(ctx, pkg.Fset.Position(candidateNode.Pos()).String())
					return false
				}
			} else if obj = pkg.TypesInfo.Defs[x]; obj != nil {
				logger.Debug().Msgf("no use but def: %+v", obj)
			} else if obj = pkg.TypesInfo.Defs[expr.Sel]; obj != nil {
				logger.Debug().
					Str("pkg_path", pkg.PkgPath).
					Str("name", expr.Sel.Name).
					Msgf("sel def")
				TraceExpressionStack(ctx, expr.Sel, modulePkgs, candidatePkg, candidateNode, exclude)
			}
		}
	case *ast.SelectorExpr: // e.g., `obj.Method.Func`
		logger.Debug().Msgf("tracing recursive selector expression: %+v", expr)
		return TraceSelectorExpr(ctx, x, candidatePkg, modulePkgs, candidateNode, exclude)
	case *ast.CallExpr:
		logger.Debug().Msgf("tracing call expression: %+v", expr)
		TraceExpressionStack(ctx, x.Fun, modulePkgs, candidatePkg, candidateNode, exclude)
	default:
		logger.Warn().Msgf("skipping selector expression X type: %T", x)
	}
	return true
}

// Trace any expression recursively, including selector expressions
func TraceExpressionStack(
	ctx context.Context,
	exprToTrace ast.Expr,
	modulePkgs []*packages.Package,
	candidatePkg *packages.Package,
	candidateNode ast.Node,
	exclude func(context.Context, string),
) bool {
	logger := polylog.Ctx(ctx).With(
		"func", "TraceExpressionStack",
		"node_type", fmt.Sprintf("%T", exprToTrace),
	)
	logger.Debug().Send()

	if exprToTrace == nil {
		return false
	}

	switch expr := exprToTrace.(type) {
	case nil:
		return false
	case *ast.CallExpr:
		if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
			return TraceSelectorExpr(ctx, sel, candidatePkg, modulePkgs, candidateNode, exclude)
		}
		return true
	case *ast.BinaryExpr:
		logger.Debug().Msg("tracing binary expression")
		// TODO_IN_THIS_COMMIT: return traceExpressionStack... ?
		if TraceExpressionStack(ctx, expr.X, modulePkgs, candidatePkg, candidateNode, exclude) {
			TraceExpressionStack(ctx, expr.Y, modulePkgs, candidatePkg, candidateNode, exclude)
		}
		return true
	case *ast.ParenExpr:
		logger.Debug().Msg("tracing paren expression")
		return TraceExpressionStack(ctx, expr.X, modulePkgs, candidatePkg, candidateNode, exclude)
	case *ast.SelectorExpr:
		logger.Debug().Msg("tracing selector expression")
		return TraceSelectorExpr(ctx, expr, candidatePkg, modulePkgs, candidateNode, exclude)
	case *ast.Ident:
		logger.Debug().Str("name", expr.Name).Msg("tracing ident")

		valueExpr, _ := FindDeclOrAssignValueExpr(ctx, expr, candidatePkg)
		if valueExpr == nil {
			return false
		}

		switch valueNode := valueExpr.(type) {
		case ast.Expr:
			TraceExpressionStack(ctx, valueNode, modulePkgs, candidatePkg, candidateNode, exclude)
		case *ast.AssignStmt:
			TraceExpressionStack(ctx, valueNode.Rhs[0], modulePkgs, candidatePkg, candidateNode, exclude)
		default:
			logger.Warn().Msgf("unknown value node type: %T", valueNode)
		}

		return true
	default:
		logger.Warn().Msgf("unknown expression node type: %T", exprToTrace)
		return true
	}
}
