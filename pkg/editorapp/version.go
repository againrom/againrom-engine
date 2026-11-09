package editorapp

import "againrom/internal/buildinfo"

// program is the editor's name in its version line; Source is the game
// program's version, which Run receives, so the editor has no version source
// of its own.
var program = buildinfo.Program{Name: "mapedit"}

// versionLine is what -version prints.
func versionLine() string { return program.Line() }
