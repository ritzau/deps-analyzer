//go:build tools
// +build tools

// This file ensures that build tools are tracked as dependencies
package tools

import (
	_ "github.com/evanw/esbuild/cmd/esbuild"
)
