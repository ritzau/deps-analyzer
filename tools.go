//go:build tools
// +build tools

// This file ensures that build tools are tracked as dependencies
package tools

import (
	_ "github.com/bazelbuild/buildtools/buildifier"
	_ "github.com/evilmartians/lefthook"
	_ "github.com/evanw/esbuild/cmd/esbuild"
	_ "golang.org/x/tools/cmd/goimports"
)
