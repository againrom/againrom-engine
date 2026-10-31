package main

import (
	_ "embed"

	"againrom/internal/buildinfo"
	"againrom/internal/versionres"
)

// resource is the program's Windows version resource object, which
// internal/versionres builds from the game's VERSION file. It is embedded on
// every platform so the editor reports the game's version without a second
// VERSION file.
//
//go:embed rsrc_windows_amd64.syso
var resource []byte

var program = buildinfo.Program{Name: "mapedit", Source: resourceVersion()}

// resourceVersion is the version the resource carries, or "" if unreadable.
func resourceVersion() string {
	v, err := versionres.FileVersion(resource)
	if err != nil {
		return ""
	}
	return v
}

// versionLine is what -version prints.
func versionLine() string { return program.Line() }
