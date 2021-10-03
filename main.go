package main

import (
	"github.com/cretz/temporal-determinist/workflow"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(workflow.NewChecker(workflow.Config{}).NewAnalyzer())
}
