package determinism

import (
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"reflect"
	"sort"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

type Config struct {
	// If empty, uses DefaultIdentRefs.
	DefaultIdentRefs IdentRefs
	// If empty, does not log debug. If present, logs debug regardless of flags.
	Debugf func(string, ...interface{})
}

type analyzer struct {
	identRefs            IdentRefs
	identRefOverrideFlag IdentRefs

	debugf    func(string, ...interface{})
	debugFlag bool
}

func NewAnalyzer(config Config) *analysis.Analyzer {
	a := &analyzer{
		identRefs:            config.DefaultIdentRefs,
		identRefOverrideFlag: IdentRefs{},
		debugf:               config.Debugf,
		debugFlag:            config.Debugf != nil,
	}
	// Use default ident refs if none set
	if len(a.identRefs) == 0 {
		a.identRefs = DefaultIdentRefs
	}
	// Clone the ident refs
	a.identRefs = a.identRefs.Clone()
	// Build analyzer
	ret := &analysis.Analyzer{
		Name:       "determinism",
		Doc:        "Analyzes all functions and marks whether they are deterministic",
		Run:        a.run,
		ResultType: reflect.TypeOf((*Result)(nil)),
		FactTypes:  []analysis.Fact{&NonDeterminisms{}},
	}
	// Set flag for ident ref overrides
	ret.Flags.Var(a.identRefOverrideFlag, "ident-refs",
		"comma-separated list of functions or vars to include or exclude, overriding the default "+
			"(append '=false' to exclude)")
	ret.Flags.BoolVar(&a.debugFlag, "debug", a.debugFlag, "show debug output")
	return ret
}

type Result struct {
	// Only includes top-level functions
	Funcs map[*types.Func]NonDeterminisms
}

func (r *Result) Dump() (lines []string) {
	// Get func types and sort first for determinism
	funcTypes := make([]*types.Func, 0, len(r.Funcs))
	for funcType := range r.Funcs {
		funcTypes = append(funcTypes, funcType)
	}
	sort.Slice(funcTypes, func(i, j int) bool { return funcTypes[i].FullName() < funcTypes[j].FullName() })
	// Build lines
	for _, funcType := range funcTypes {
		for _, reason := range r.Funcs[funcType] {
			lines = append(lines, funcType.FullName()+" is non-deterministic, reason: "+reason.String())
			// Dump the child tree too
			if funcCall, _ := reason.(*ReasonFuncCall); funcCall != nil {
				lines = funcCall.appendChildReasonLines(lines, 1)
			}
		}
	}
	return
}

func (a *analyzer) run(pass *analysis.Pass) (interface{}, error) {
	// Override ident refs with overrides (safe to do repeatedly)
	for k, v := range a.identRefOverrideFlag {
		a.identRefs[k] = v
	}
	// Set debugf based on flag _only_ if not already set
	if a.debugf == nil {
		if a.debugFlag {
			a.debugf = log.Printf
		} else {
			a.debugf = func(string, ...interface{}) {}
		}
	}
	a.debugf("Checking package %v", pass.Pkg.Name())
	// Collect all non-determinisms in the package
	res := &Result{Funcs: map[*types.Func]NonDeterminisms{}}
	a.findNonDeterminisms(pass, res)
	return res, nil
}

func (a *analyzer) findNonDeterminisms(pass *analysis.Pass, res *Result) {
	// Collect all top-level func decls and their types. Also mark var decls as
	// non-deterministic if pattern matches.
	funcDecls := map[*types.Func]*ast.FuncDecl{}
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				// Collect top-level func
				if funcType, _ := pass.TypesInfo.ObjectOf(decl.Name).(*types.Func); funcType != nil {
					funcDecls[funcType] = decl
				}
			case *ast.GenDecl:
				// Set top-level vars that match pattern as non-deterministic
				for _, spec := range decl.Specs {
					if valueSpec, _ := spec.(*ast.ValueSpec); valueSpec != nil {
						for _, varName := range valueSpec.Names {
							if varType, _ := pass.TypesInfo.ObjectOf(varName).(*types.Var); varType != nil && varType.Pkg() != nil {
								fullName := varType.Pkg().Path() + "." + varType.Name()
								if a.identRefs[fullName] {
									a.debugf("Marking %v as non-determistic because it matched a pattern", fullName)
									pos := pass.Fset.Position(varType.Pos())
									reasons := NonDeterminisms{&ReasonDecl{reasonBase: reasonBase{&pos}}}
									pass.ExportObjectFact(varType, &reasons)
								}
							}
						}
					}
				}
			}
		}
	}
	// Walk the decls capturing non-deterministic ones
	parents := map[*types.Func]bool{}
	for funcType := range funcDecls {
		a.applyNonDeterminisms(pass, funcType, funcDecls, parents, res.Funcs)
	}
	// Set non-empty non-determisms as facts
	for funcType, nonDet := range res.Funcs {
		nonDet := nonDet
		if len(nonDet) > 0 {
			pass.ExportObjectFact(funcType, &nonDet)
		}
	}
}

// Returns true if non-deterministic, false otherwise. Parents must be non-nil
// and all values must be true.
func (a *analyzer) applyNonDeterminisms(
	pass *analysis.Pass,
	fn *types.Func,
	packageDecls map[*types.Func]*ast.FuncDecl,
	parents map[*types.Func]bool,
	results map[*types.Func]NonDeterminisms,
) NonDeterminisms {
	// Check to make sure not recursive
	if parents[fn] {
		// Recursive call is not marked non-deterministic
		return nil
	}
	// Check if determinisms already set or it's in a different package (which
	// means we can't re-set later)
	reasons, alreadySet := results[fn]
	if alreadySet || pass.ImportObjectFact(fn, &reasons) || fn.Pkg() != pass.Pkg {
		return reasons
	}
	// Check if matches pattern
	var skip bool
	if match, ok := a.identRefs[fn.FullName()]; match {
		a.debugf("Marking %v as non-determistic because it matched a pattern", fn.FullName())
		pos := pass.Fset.Position(fn.Pos())
		reasons = append(reasons, &ReasonDecl{reasonBase: reasonBase{&pos}})
	} else if ok && !match {
		a.debugf("Skipping %v because it matched a pattern", fn.FullName())
		skip = true
	}
	// If not skipped and has top-level decl, walk the declaration body checking
	// for non-determinism
	if !skip && packageDecls[fn] != nil {
		ast.Inspect(packageDecls[fn], func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				// Check if the call is on a non-deterministic
				if callee, _ := typeutil.Callee(pass.TypesInfo, n).(*types.Func); callee != nil {
					// Put self on parents, then remove
					parents[fn] = true
					calleeNonDet := a.applyNonDeterminisms(pass, callee, packageDecls, parents, results)
					delete(parents, fn)
					// If the callee is non-deterministic, mark this as such
					if len(calleeNonDet) > 0 {
						a.debugf("Marking %v as non-determistic because it calls %v", fn.FullName(), callee.FullName())
						pos := pass.Fset.Position(n.Pos())
						reasons = append(reasons, &ReasonFuncCall{reasonBase: reasonBase{&pos}, Func: callee, Child: calleeNonDet})
					}
				}
			case *ast.GoStmt:
				// Any go statement is non-deterministic
				a.debugf("Marking %v as non-determistic because it starts a goroutine", fn.FullName())
				pos := pass.Fset.Position(n.Pos())
				reasons = append(reasons, &ReasonConcurrency{reasonBase: reasonBase{&pos}, Kind: ConcurrencyKindGo})
			case *ast.Ident:
				// Check if ident is for a non-deterministic var
				if varType, _ := pass.TypesInfo.ObjectOf(n).(*types.Var); varType != nil {
					var ignore NonDeterminisms
					if pass.ImportObjectFact(varType, &ignore) {
						a.debugf("Marking %v as non-determistic because it accesses %v.%v",
							fn.FullName(), varType.Pkg().Path(), varType.Name())
						pos := pass.Fset.Position(n.Pos())
						reasons = append(reasons, &ReasonVarAccess{reasonBase: reasonBase{&pos}, Var: varType})
					}
				}
			case *ast.RangeStmt:
				// Map and chan ranges are non-deterministic
				rangeType := pass.TypesInfo.TypeOf(n.X)
				// Unwrap named type
				for {
					if namedType, _ := rangeType.(*types.Named); namedType != nil {
						rangeType = namedType.Underlying()
					} else {
						break
					}
				}
				switch rangeType.(type) {
				case *types.Map:
					a.debugf("Marking %v as non-determistic because it iterates over a map", fn.FullName())
					pos := pass.Fset.Position(n.Pos())
					reasons = append(reasons, &ReasonMapRange{reasonBase: reasonBase{&pos}})
				case *types.Chan:
					a.debugf("Marking %v as non-determistic because it iterates over a channel", fn.FullName())
					pos := pass.Fset.Position(n.Pos())
					reasons = append(reasons, &ReasonConcurrency{reasonBase: reasonBase{&pos}, Kind: ConcurrencyKindRange})
				}
			case *ast.SendStmt:
				// Any send statement is non-deterministic
				a.debugf("Marking %v as non-determistic because it sends to a channel", fn.FullName())
				pos := pass.Fset.Position(n.Pos())
				reasons = append(reasons, &ReasonConcurrency{reasonBase: reasonBase{&pos}, Kind: ConcurrencyKindSend})
			case *ast.UnaryExpr:
				// If the operator is a receive, it is non-deterministic
				if n.Op == token.ARROW {
					a.debugf("Marking %v as non-determistic because it receives from a channel", fn.FullName())
					pos := pass.Fset.Position(n.Pos())
					reasons = append(reasons, &ReasonConcurrency{reasonBase: reasonBase{&pos}, Kind: ConcurrencyKindRecv})
				}
			}
			return true
		})
	}
	// Put the reasons fact on the func, even if it is empty
	results[fn] = reasons
	return reasons
}
