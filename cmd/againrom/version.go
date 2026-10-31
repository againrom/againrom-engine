package main

import (
	_ "embed"

	"againrom/internal/buildinfo"
)

// versionFile is the program's tracked VERSION file, one MAJOR.MINOR.PATCH
// line. It is embedded so a plain "go build" carries it.
//
//go:embed VERSION
var versionFile string

var program = buildinfo.Program{Name: "againrom", Source: versionFile}

// versionLine is what -version prints.
func versionLine() string { return program.Line() }

// menuLabel is the small text the main menu shows in its corner.
func menuLabel() string {
	return "Againrom " + program.Version() + " (" + buildinfo.Short(7) + ")"
}
