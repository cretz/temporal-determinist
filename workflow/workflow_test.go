package workflow_test

import (
	"testing"

	"github.com/cretz/temporal-determinist/workflow"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	analysistest.Run(
		t,
		analysistest.TestData(),
		workflow.NewAnalyzer(workflow.Config{Debugf: t.Logf}),
		"a",
	)
}
