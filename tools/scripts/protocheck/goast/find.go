package goast

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"slices"

	"golang.org/x/tools/go/packages"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// TODO_IN_THIS_COMMIT: move & godoc...
func FindDeclOrAssignValueExpr(
	ctx context.Context,
	targetIdent *ast.Ident,
	pkg *packages.Package,
) (valueNode ast.Node, valueNodePos token.Position) {
	valueNode, valueNodePos = FindValueExprByDeclaration(ctx, targetIdent, pkg)

	// TODO_IN_THIS_COMMIT: improve comment...
	// If it does not, search the package for
	// the ident and return the closest assignment.
	if valueNode == nil {
		valueNode, valueNodePos = FindValueExprByAssignment(ctx, targetIdent, pkg)
	}

	return valueNode, valueNodePos
}

// TODO_IN_THIS_COMMIT: move & godoc...
func FindValueExprByDeclaration(
	ctx context.Context,
	targetIdent *ast.Ident,
	pkg *packages.Package,
) (valueNode ast.Node, valueNodePos token.Position) {
	logger := polylog.Ctx(ctx).With("func", "FindValueExprByDeclaration")

	// TODO_IN_THIS_COMMIT: comment...
	for _, fileNode := range pkg.Syntax {
		if valueNode != nil {
			break
		}

		if obj := pkg.TypesInfo.Uses[targetIdent]; obj != nil {
			logger.Debug().Fields(map[string]any{
				"file_path":  fmt.Sprintf(" %s ", pkg.Fset.File(fileNode.Pos()).Name()),
				"target_pos": fmt.Sprintf(" %s ", pkg.Fset.Position(targetIdent.Pos()).String()),
				"decl_pos":   fmt.Sprintf(" %s ", pkg.Fset.Position(obj.Pos()).String()),
			}).Msg("uses")
			valueNodePos = pkg.Fset.Position(obj.Pos())
			valueNode = FindNodeByPosition(pkg.Fset, fileNode, valueNodePos)
			if valueNode != nil {
				logger.Debug().
					Str("decl_node", fmt.Sprintf("%+v", valueNode)).
					Str("decl_pos", fmt.Sprintf(" %s ", valueNodePos)).
					Msg("found decl node")
			}
		}
	}

	// TODO_IN_THIS_COMMIT: improve comment...
	// Look through decl node to see if it contains a valudspec with values.
	// If it does, return the value(s).
	if valueNode != nil {
		ast.Inspect(valueNode, func(n ast.Node) bool {
			switch doa := n.(type) {
			case *ast.ValueSpec:
				logger.Debug().
					Int("len(values)", len(doa.Values)).
					Msgf("value spec: %+v", doa)

				if doa.Values != nil {
					logger.Debug().Msg("doa.Values != nil")
					for _, value := range doa.Values {
						valueNodePos = pkg.Fset.Position(value.Pos())
						valueNode = value
					}
				} else {
					logger.Debug().Msg("dao.Values == nil")
					valueNode = nil
				}
			}

			return true
		})
	} else {
		logger.Debug().Msgf("no declaration or assignment found for ident %q", targetIdent.String())
	}

	return valueNode, valueNodePos
}

// TODO_IN_THIS_COMMIT: move & godoc...
// search for targetIdent by position
func FindNodeByPosition(
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

// TODO_IN_THIS_COMMIT: move & godoc...
func FindValueExprByAssignment(
	ctx context.Context,
	targetIdent *ast.Ident,
	pkg *packages.Package,
) (valueNode ast.Node, valueNodePos token.Position) {
	assignsRhs := collectAssignments(targetIdent, pkg)

	if len(assignsRhs) < 1 {
		return valueNode, valueNodePos
	}

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

	// TODO_IN_THIS_COMMIT: improve comment...
	// ValueNode is the closest assignment whose position is less than or equal to the valueNodePos.
	return FindClosestAssignment(ctx, assignsRhs, targetIdent, pkg)
}

// TODO_IN_THIS_COMMIT: move & godoc...
func FindClosestAssignment(
	ctx context.Context,
	assignsRhs []ast.Expr,
	targetIdent *ast.Ident,
	pkg *packages.Package,
) (valueNode ast.Expr, valueNodePos token.Position) {
	logger := polylog.Ctx(ctx).With("func", "FindClosestAssignment")

	var (
		targetIdentPos = pkg.Fset.Position(targetIdent.Pos())
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
			valueNodePos = rhsPos
			valueNode = rhs
		case rhsPos.Line == targetIdentPos.Line:
			if rhsPos.Column <= targetIdentPos.Column {
				valueNodePos = rhsPos
				valueNode = rhs
			}
		}
	}

	return valueNode, valueNodePos
}

// TODO_IN_THIS_COMMIT: move & godoc...
func collectAssignments(
	targetIdent *ast.Ident,
	pkg *packages.Package,
) (assignsRhs []ast.Expr) {
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

	return assignsRhs
}
