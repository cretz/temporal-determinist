package determinist

import (
	"go/token"
	"go/types"
)

type NonDetermisms []Reason

func (NonDetermisms) AFact() {}

type Reason interface {
	Pos() *token.Position
}

type reasonBase struct {
	pos *token.Position
}

func (r *reasonBase) Pos() *token.Position { return r.pos }

type ReasonFuncCall struct {
	reasonBase
	Func  *types.Func
	Child Reason
}

type ReasonGlobalVarAccess struct {
	reasonBase
	Var *types.Var
}

type ReasonConcurrency struct {
	reasonBase
	Kind ConcurrencyKind
}

type ConcurrencyKind int

const (
	Go ConcurrencyKind = iota
	Select
	Recv
	Send
)

type ReasonMapRange struct {
	reasonBase
}
