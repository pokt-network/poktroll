package protoast

import (
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/desc/protoparse/ast"
)

// TODO_IN_THIS_COMMIT: godoc...
type ProtoFileStat struct {
	PkgSource        *ast.SourcePos
	LastOptSource    *ast.SourcePos
	LastImportSource *ast.SourcePos
	HasGogoImport    bool
}

// TODO_IN_THIS_COMMIT: godoc...
type ProtoMsgStat struct {
	Node      *ast.MessageNode
	GoPkgPath string
}

// TODO_IN_THIS_COMMIT: godoc...
var DefaultParser = protoparse.Parser{
	IncludeSourceCodeInfo: true,
}
