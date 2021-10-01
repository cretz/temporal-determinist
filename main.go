package main

import (
	"github.com/cretz/temporal-determinist/determinism"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(determinism.NewAnalyzer(determinism.Config{}))
}
