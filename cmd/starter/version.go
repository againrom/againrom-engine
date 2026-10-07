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

func programVersion() string {
	v, err := buildinfo.ParseVersion(versionFile)
	if err != nil {
		return "unknown"
	}
	return v
}

func versionLine() string { return buildinfo.Line("starter", programVersion()) }

func shortRevision() string { return buildinfo.Short(7) }
