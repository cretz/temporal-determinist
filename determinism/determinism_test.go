package determinism_test

import (
	"testing"

	"github.com/cretz/temporal-determinist/determinism"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	identRefs := determinism.DefaultIdentRefs.Clone()
	identRefs["a.BadCall"] = true
	identRefs["a.BadVar"] = true
	identRefs["a.IgnoredCall"] = false
	identRefs["os.Stderr"] = false
	results := analysistest.Run(
		t,
		analysistest.TestData(),
		determinism.NewAnalyzer(determinism.Config{DefaultIdentRefs: identRefs, Debugf: t.Logf}),
		"a",
	)
	if testing.Verbose() {
		// Dump the tree of the "a" package
		for _, result := range results {
			if result.Pass.Pkg.Name() == "a" {
				if res, _ := result.Result.(*determinism.Result); res != nil {
					for _, line := range res.Dump() {
						t.Log(line)
					}
				}
			}
		}
	}
}
