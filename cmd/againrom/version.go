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

// programVersion is the validated version, or "unknown" if the file is
// malformed (a test pins that it is not).
func programVersion() string {
	v, err := buildinfo.ParseVersion(versionFile)
	if err != nil {
		return "unknown"
	}
	return v
}

// versionLine is what -version prints.
func versionLine() string { return buildinfo.Line("againrom", programVersion()) }

// menuLabel is the small text the main menu shows in its corner.
func menuLabel() string {
	return "Againrom " + programVersion() + " (" + buildinfo.Short(7) + ")"
}
