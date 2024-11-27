package goast

import (
	"context"
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/packages"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// TODO_IN_THIS_COMMIT: move & godoc...
func NewInspectLastReturnArgFn(
	ctx context.Context,
	pkg *packages.Package,
	modulePkgs []*packages.Package,
	flag func(context.Context, string),
	exclude func(context.Context, string),
) func(ast.Node) bool {
	logger := polylog.Ctx(ctx)

	return func(n ast.Node) bool {
		if n == nil {
			return false
		}

		logger.Debug().
			Str("position", fmt.Sprintf(" %s ", pkg.Fset.Position(n.Pos()).String())).
			Str("node_type", fmt.Sprintf("%T", n)).
			Bool("flagging", flag != nil).
			Msg("walking function body")

		switch n := n.(type) {
		case *ast.ReturnStmt:
			lastResult := n.Results[len(n.Results)-1]
			inspectPosition := pkg.Fset.Position(lastResult.Pos()).String()

			logger = logger.With(
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

				if flag != nil {
					logger.Debug().Msg("appending potential offending line")
					flag(ctx, inspectPosition)
				}
				TraceExpressionStack(ctx, lastReturnArgNode, modulePkgs, pkg, lastReturnArgNode, exclude)
				return true

			// E.g. `return nil, types.ErrXXX.Wrapf(...)` <-- last arg is a *ast.CallExpr.
			case *ast.CallExpr:
				if flag != nil {
					logger.Debug().Msg("appending potential offending line")
					flag(ctx, inspectPosition)
				}
				TraceExpressionStack(ctx, lastReturnArgNode, modulePkgs, pkg, lastReturnArgNode, exclude)
				return true

			case *ast.SelectorExpr:
				if flag != nil {
					logger.Debug().Msg("appending potential offending line")
					flag(ctx, inspectPosition)
				}
				TraceSelectorExpr(ctx, lastReturnArgNode, pkg, modulePkgs, lastReturnArgNode, exclude)
				return true
			}

		default:
			return true
		}

		return true
	}
}
