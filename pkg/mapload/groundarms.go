package mapload

import "againrom/pkg/formats/alm"

// GroundArm is one reason a cell is closed to a ground mover.
type GroundArm uint8

const (
	GroundBorder    GroundArm = 1 << iota // the eight-cell ring round the map
	GroundMountain                        // the classifier's Mountain
	GroundWater                           // the raw water index range
	GroundFlag                            // the tile word's impassable bit 13
	GroundObject                          // a static object on the overlay plane
	GroundStructure                       // a placed structure's footprint
)

// GroundArmRow is one arm with the word ID the editor names it by.
type GroundArmRow struct {
	Arm  GroundArm
	Word string
}

// GroundArmRows is every arm, in the order the editor names a cell's arms.
// It is the one table of the arms: the host's refusals and the editor's
// passability view both read it.
var GroundArmRows = [...]GroundArmRow{
	{Arm: GroundBorder, Word: "editor.pass.border"},
	{Arm: GroundMountain, Word: "editor.pass.mountain"},
	{Arm: GroundWater, Word: "editor.pass.water"},
	{Arm: GroundFlag, Word: "editor.pass.flag"},
	{Arm: GroundObject, Word: "editor.pass.object"},
	{Arm: GroundStructure, Word: "editor.pass.structure"},
}

// GroundArms is, per in-bounds cell and row-major, why PassabilityWith(m, t)
// closes the cell to a ground mover: zero exactly where that plane's ground
// bit is clear. It reads the plane and the occupancy of the one derivation
// the world is built from. A cell a structure attached carries
// GroundStructure alone, since the footprint overrides every terrain arm
// there; any other closed cell carries each terrain arm that holds for it.
func GroundArms(m *alm.Map, t *Table) []GroundArm {
	plane, _, taken := deriveTaken(m, t)
	w, h, ok := extent(m)
	if !ok {
		return nil
	}
	out := make([]GroundArm, w*h)
	for i := range out {
		if plane[i]&blockGround == 0 {
			continue
		}
		if taken != nil && taken[i] {
			out[i] = GroundStructure
			continue
		}
		a := cellArms(planeAt(m.Tiles, i), planeAt(m.Overlay, i), i%w, i/w, w, h)
		for _, arm := range [...]struct {
			on  bool
			arm GroundArm
		}{{a.border, GroundBorder}, {a.mountain, GroundMountain}, {a.water, GroundWater}, {a.impassable, GroundFlag}, {a.scenery, GroundObject}} {
			if arm.on {
				out[i] |= arm.arm
			}
		}
	}
	return out
}
