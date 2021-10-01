package determinism

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
)

type NonDeterminisms []Reason

func (*NonDeterminisms) AFact() {}

func (n *NonDeterminisms) String() string {
	if n == nil {
		return "<none>"
	}
	var str string
	for _, reason := range *n {
		if str != "" {
			str += ", "
		}
		str += reason.String()
	}
	return str
}

type Reason interface {
	Pos() *token.Position
	// Should not include position
	String() string
}

type reasonBase struct {
	pos *token.Position
}

func (r *reasonBase) Pos() *token.Position { return r.pos }

type ReasonDecl struct {
	reasonBase
}

func (r *ReasonDecl) String() string {
	return "declared non-deterministic"
}

type ReasonFuncCall struct {
	reasonBase
	Func  *types.Func
	Child NonDeterminisms
}

func (r *ReasonFuncCall) String() string {
	return "calls non-determistic function " + r.Func.FullName()
}

func (r *ReasonFuncCall) appendChildReasonLines(s []string, depth int) []string {
	for _, childReason := range r.Child {
		s = append(s, fmt.Sprintf("%v is non-deterministic, reason: %v",
			strings.Repeat("  ", depth)+r.Func.FullName(), childReason))
		// Recurse if func call
		if funcCall, _ := childReason.(*ReasonFuncCall); funcCall != nil {
			s = funcCall.appendChildReasonLines(s, depth+1)
		}
	}
	return s
}

type ReasonVarAccess struct {
	reasonBase
	Var *types.Var
}

func (r *ReasonVarAccess) String() string {
	return "accesses non-determistic var " + r.Var.Pkg().Path() + "." + r.Var.Name()
}

type ReasonConcurrency struct {
	reasonBase
	Kind ConcurrencyKind
}

func (r *ReasonConcurrency) String() string {
	switch r.Kind {
	case ConcurrencyKindGo:
		return "starts goroutine"
	case ConcurrencyKindRecv:
		return "receives from channel"
	case ConcurrencyKindSend:
		return "sends to channel"
	case ConcurrencyKindRange:
		return "iterates over channel"
	default:
		return "<unknown-kind>"
	}
}

type ConcurrencyKind int

const (
	ConcurrencyKindGo ConcurrencyKind = iota
	ConcurrencyKindRecv
	ConcurrencyKindSend
	ConcurrencyKindRange
)

type ReasonMapRange struct {
	reasonBase
}

func (r *ReasonMapRange) String() string {
	return "iterates over map"
}
