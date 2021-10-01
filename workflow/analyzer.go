package workflow

import (
	"go/ast"
	"go/types"
	"log"
	"sync"

	"github.com/cretz/temporal-determinist/determinism"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// TODO: Determinism from any workflow function
// TODO: Only serializable parameters from any function

type Config struct {
	// If empty, does not log debug. If present, logs debug regardless of flags.
	Debugf func(string, ...interface{})
}

type analyzer struct {
	debugf    func(string, ...interface{})
	debugFlag bool

	detAnalyzer *analysis.Analyzer
	nonDet      map[*types.Func]determinism.NonDeterminisms
	nonDetLock  sync.RWMutex
}

func NewAnalyzer(config Config) *analysis.Analyzer {
	a := &analyzer{
		debugf:      config.Debugf,
		debugFlag:   config.Debugf != nil,
		detAnalyzer: determinism.NewAnalyzer(determinism.Config{}),
		nonDet:      map[*types.Func]determinism.NonDeterminisms{},
	}
	ret := &analysis.Analyzer{
		Name:     "workflow",
		Doc:      "Analyzes all workflow functions for RegisterWorkflow for non-determinism and parameter accuracy",
		Run:      a.run,
		Requires: []*analysis.Analyzer{a.detAnalyzer},
	}
	ret.Flags.BoolVar(&a.debugFlag, "debug", a.debugFlag, "show debug output")
	return ret
}

func (a *analyzer) run(pass *analysis.Pass) (interface{}, error) {
	// Get the determinism results
	detRes := pass.ResultOf[a.detAnalyzer].(*determinism.Result)
	// Copy all facts to this pass
	a.nonDetLock.Lock()
	for funcType, reasons := range detRes.Funcs {
		a.nonDet[funcType] = reasons
	}
	a.nonDetLock.Unlock()
	// Set debugf based on flag _only_ if not already set
	if a.debugf == nil {
		if a.debugFlag {
			a.debugf = log.Printf
		} else {
			a.debugf = func(string, ...interface{}) {}
		}
	}
	a.debugf("Checking package %v", pass.Pkg.Name())
	// Check every register workflow invocation
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// Only handle calls
			callExpr, _ := n.(*ast.CallExpr)
			if callExpr == nil {
				return true
			}
			// Callee needs to be workflow registry
			callee, _ := typeutil.Callee(pass.TypesInfo, callExpr).(*types.Func)
			const regName = "(go.temporal.io/sdk/worker.WorkflowRegistry).RegisterWorkflow"
			const regOptName = "(go.temporal.io/sdk/worker.WorkflowRegistry).RegisterWorkflowWithOptions"
			if callee == nil || len(callExpr.Args) == 0 || (callee.FullName() != regName && callee.FullName() != regOptName) {
				return true
			}
			// First param should be a function ident or a selector with ident as
			// function
			var fn *types.Func
			switch arg := callExpr.Args[0].(type) {
			case *ast.Ident:
				fn, _ = pass.TypesInfo.ObjectOf(arg).(*types.Func)
			case *ast.SelectorExpr:
				fn, _ = pass.TypesInfo.ObjectOf(arg.Sel).(*types.Func)
			}
			// Report if couldn't get type
			if fn == nil {
				pass.Reportf(callExpr.Args[0].Pos(), "unrecognized function reference format")
				return true
			}
			a.debugf("Checking workflow function %v", fn.FullName())
			// If there are any non-determinisms, we need to mark the diagnostics
			// TODO: The rest
			return true
		})
	}
	return nil, nil
}
