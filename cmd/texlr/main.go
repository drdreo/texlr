package main

import (
	"context"
	"os"

	"github.com/drdreo/texlr/internal/texlr"
)

var version = "dev"

func main() {
	os.Exit(texlr.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, version))
}
