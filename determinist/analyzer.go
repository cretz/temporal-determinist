package determinist

import (
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const Doc = `
TODO(cretz): Document
`

type Config struct {
	// If empty, uses DefaultIdentRefs
	IdentRefs IdentRefs
}

type Determinist struct {
	analysis.Analyzer
	Config
}

func New(config Config) *Determinist {
	if len(config.IdentRefs) == 0 {
		config.IdentRefs = DefaultIdentRefs
	}
	// Clone the map
	config.IdentRefs = config.IdentRefs.Clone()
	d := &Determinist{
		Analyzer: analysis.Analyzer{
			Name:       "determinist",
			Doc:        Doc,
			Requires:   []*analysis.Analyzer{inspect.Analyzer},
			ResultType: reflect.TypeOf((*Result)(nil)),
			FactTypes:  []analysis.Fact{NonDetermisms{}},
		},
		Config: config,
	}
	d.Analyzer.Run = d.run
	// Set flags
	d.Flags.Var(d.IdentRefs, "ident-refs", "comma-separated list of functions or vars to include or exclude "+
		"(append '=false' to exclude)")
	return d
}

type Result struct {
	Funcs               map[*types.Func]NonDetermisms
	RegisteredWorkflows []*types.Func
}

func (d *Determinist) Run(pass *analysis.Pass) (*Result, error) {
	v, err := d.run(pass)
	if err != nil {
		return nil, err
	}
	return v.(*Result), nil
}

func (d *Determinist) run(pass *analysis.Pass) (interface{}, error) {
	res := &Result{Funcs: map[*types.Func]NonDetermisms{}}
	// Collect all non-deterministic functions in the package
	d.findNonDetermisticFuncs(pass, res)
	// Collect all registered workflows in the package
	d.findRegisteredWorkflows(pass, res)
	return res, nil
}

func (d *Determinist) findNonDetermisticFuncs(pass *analysis.Pass, res *Result) {
	// TODO(cretz): This
}

func (d *Determinist) findRegisteredWorkflows(pass *analysis.Pass, res *Result) {
	// TODO(cretz): This
}
