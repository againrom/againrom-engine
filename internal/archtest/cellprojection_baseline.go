package archtest

// CellProjectionDebt is the falling-only count of cell-geometry sites per
// production file of pkg/ui and pkg/render outside the projection family:
// sprite, shadow, static, structure, status-bar and effect placements that take
// a lift and an origin as separate terms, a crossing's height interpolation,
// camera bounds and scroll steps in whole cells, sub-cell positions, texture
// and canvas sizes, and products of 32 that are not cells. A file reaching
// zero leaves the map in the same commit.
var CellProjectionDebt = map[string]int{
	"pkg/render/camera/camera.go":       3,
	"pkg/render/debugtext/debugtext.go": 2,
	"pkg/render/terrain/composite.go":   2,
	"pkg/render/terrain/effectmark.go":  1,
	"pkg/render/terrain/lit.go":         2,
	"pkg/render/terrain/shadow.go":      2,
	"pkg/render/terrain/statics.go":     2,
	"pkg/render/terrain/statusbar.go":   6,
	"pkg/render/terrain/structures.go":  5,
	"pkg/render/terrain/tileset.go":     1,
	"pkg/ui/overlay.go":                 7,
	"pkg/ui/qolkeys.go":                 1,
	"pkg/ui/saveapplication.go":         4,
	"pkg/ui/shadow.go":                  1,
	"pkg/ui/spellbolt.go":               3,
	"pkg/ui/statics.go":                 1,
	"pkg/ui/tippanel.go":                1,
	"pkg/ui/viewer.go":                  9,
}
