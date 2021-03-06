package workflow

import (
	"go/ast"
	"go/types"
	"log"
	"strings"

	"github.com/cretz/temporal-determinist/determinism"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// DefaultIdentRefs are additional overrides of determinism.DefaultIdentRefs for
// safe Temporal library functions.
var DefaultIdentRefs = determinism.DefaultIdentRefs.Clone().SetAll(determinism.IdentRefs{
	// Reported as non-deterministic because it internally starts a goroutine, so
	// mark deterministic explicitly
	"go.temporal.io/sdk/internal.propagateCancel": false,
	// Reported as non-deterministic because it iterates over a map, so mark
	// deterministic explicitly
	"(*go.temporal.io/sdk/internal.cancelCtx).cancel": false,
})

// Config is config for NewChecker.
type Config struct {
	// If empty, uses DefaultIdentRefs.
	DefaultIdentRefs determinism.IdentRefs
	// If nil, uses log.Printf.
	DebugfFunc func(string, ...interface{})
	// Must be set to true to see advanced debug logs.
	Debug bool
	// Must be set to true to see advanced determinism debug logs.
	DeterminismDebug bool
	// If set, the file and line/col position is present on nested errors.
	IncludePosOnMessage bool
}

// Checker checks if functions passed RegisterWorkflow are non-deterministic
// based on the results from the checker of the adjacent determinism package.
type Checker struct {
	DebugfFunc          func(string, ...interface{})
	Debug               bool
	IncludePosOnMessage bool
	Determinism         *determinism.Checker
}

// NewChecker creates a Checker for the given config.
func NewChecker(config Config) *Checker {
	// Set default refs but we don't have to clone since the determinism
	// constructor will do that
	if config.DefaultIdentRefs == nil {
		config.DefaultIdentRefs = DefaultIdentRefs
	}
	// Default debug
	if config.DebugfFunc == nil {
		config.DebugfFunc = log.Printf
	}
	// Build checker
	return &Checker{
		DebugfFunc:          config.DebugfFunc,
		Debug:               config.Debug,
		IncludePosOnMessage: config.IncludePosOnMessage,
		Determinism: determinism.NewChecker(determinism.Config{
			DefaultIdentRefs: config.DefaultIdentRefs,
			DebugfFunc:       config.DebugfFunc,
			Debug:            config.DeterminismDebug,
		}),
	}
}

func (c *Checker) debugf(f string, v ...interface{}) {
	if c.Debug {
		c.DebugfFunc(f, v...)
	}
}

// NewAnalyzer creates a Go analysis analyzer that can be used in existing
// tools. There is a -set-decl flag for adding ident refs overrides, a
// -workflow-debug flag for enabling debug logs, a -determinism-debug flag for
// enabling determinism debug logs, and a -show-pos flag for showing position on
// nested errors. This analyzer does not have any results but does set the same
// facts as the determinism analyzer (*determinism.NonDeterminisms).
func (c *Checker) NewAnalyzer() *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name:      "workflow",
		Doc:       "Analyzes all RegisterWorkflow functions for non-determinism",
		Run:       func(p *analysis.Pass) (interface{}, error) { return nil, c.Run(p) },
		FactTypes: []analysis.Fact{&determinism.NonDeterminisms{}},
	}
	// Set flags
	a.Flags.Var(determinism.NewIdentRefsFlag(c.Determinism.IdentRefs), "set-decl",
		"qualified function/var to include/exclude, overriding the default (append '=false' to exclude)")
	a.Flags.BoolVar(&c.Debug, "workflow-debug", c.Debug, "show workflow debug output")
	a.Flags.BoolVar(&c.Determinism.Debug, "determism-debug", c.Determinism.Debug, "show determinism debug output")
	a.Flags.BoolVar(&c.IncludePosOnMessage, "show-pos", c.IncludePosOnMessage,
		"show file positions on determinism messages")
	return a
}

// Run executes this checker for the given pass.
func (c *Checker) Run(pass *analysis.Pass) error {
	// Run determinism pass
	if _, err := c.Determinism.Run(pass); err != nil {
		return err
	}
	c.debugf("Checking package %v", pass.Pkg.Path())
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
			c.debugf("Checking workflow function %v", fn.FullName())
			// If there are any non-determinisms, we need to mark the diagnostics
			var reasons determinism.NonDeterminisms
			if pass.ImportObjectFact(fn, &reasons) && len(reasons) > 0 {
				// One report per reason
				for _, reason := range reasons {
					lines := determinism.NonDeterminisms{reason}.AppendChildReasonLines(
						fn.FullName(), nil, 0, c.IncludePosOnMessage)
					pass.Report(analysis.Diagnostic{Pos: callExpr.Pos(), Message: strings.Join(lines, "\n")})
				}
			}
			return true
		})
	}
	return nil
}
