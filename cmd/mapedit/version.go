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

// programVersion is the game version the resource carries, or "unknown" if it
// cannot be read (a test pins that it can).
func programVersion() string {
	v, err := versionres.FileVersion(resource)
	if err != nil {
		return "unknown"
	}
	return v
}

// versionLine is what -version prints.
func versionLine() string { return buildinfo.Line("mapedit", programVersion()) }
