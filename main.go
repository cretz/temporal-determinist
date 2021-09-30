package main

import (
	"github.com/cretz/temporal-determinist/determinist"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(&determinist.New(determinist.Config{}).Analyzer)
}
