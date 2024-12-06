//go:build tools
// +build tools

package main

import (
	_ "github.com/google/wire/cmd/wire"
	_ "github.com/moznion/gonstructor/cmd/gonstructor"
	_ "github.com/vektra/mockery/v2"
)
