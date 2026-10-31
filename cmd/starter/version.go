package main

import (
	_ "embed"

	"againrom/internal/buildinfo"
)

// versionFile is the program's tracked VERSION file, one MAJOR.MINOR.PATCH
// line, embedded so a plain "go build" carries it.
//
//go:embed VERSION
var versionFile string

var program = buildinfo.Program{Name: "starter", Source: versionFile}

func versionLine() string { return program.Line() }

func shortRevision() string { return buildinfo.Short(7) }
