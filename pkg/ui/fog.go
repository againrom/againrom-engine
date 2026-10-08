package ui

import "againrom/pkg/render/terrain"

// The three fog states a cell holds, per participant (spec Terms "Fog
// state"). The decoded law's fourth combination, visible-but-unexplored, has
// no representative here on purpose (spec Terms: "A cell cannot be visible
// without having been explored").
const (
	FogUnseen   uint8 = 0
	FogExplored uint8 = 1
	FogVisible  uint8 = 2
)

// SetFog projects an ordinary fog sample without changing its backing slice.
// Cheat exploration and transient visibility live only in the client plane.
func (v *Viewer) SetFog(plane []byte, cols, rows int) {
	v.cheatFogSource = plane
	if len(v.cheatExplored) != 0 {
		plane = append([]byte(nil), plane...)
		for i := 0; i < min(len(plane), len(v.cheatExplored)); i++ {
			if v.cheatExplored[i] != 0 && v.cheatVisible {
				plane[i] = FogVisible
			} else if v.cheatExplored[i] != 0 && plane[i] == FogUnseen {
				plane[i] = FogExplored
			}
		}
	}
	v.fogPlane, v.fogCols, v.fogRows = plane, cols, rows
}

func (v *Viewer) SetCheatMapReveal(on bool) {
	v.fogReveal = on
	if !on {
		return
	}
	v.cheatVisible = true
	v.cheatExplored = make([]byte, len(v.fogPlane))
	for i := range v.cheatExplored {
		v.cheatExplored[i] = 1
	}
	v.SetFog(v.cheatFogSource, v.fogCols, v.fogRows)
}

func (v *Viewer) RefreshCheatFog() {
	if !v.fogReveal && v.cheatVisible {
		v.cheatVisible = false
		v.SetFog(v.cheatFogSource, v.fogCols, v.fogRows)
	}
}

// fogAt applies reveal, then reads a bounded cell in the client plane.
// An empty plane is fog-free; cells outside a nonempty plane are unseen.
func (v *Viewer) fogAt(col, row int) uint8 {
	if v.fogReveal {
		return FogVisible
	}
	if len(v.fogPlane) == 0 {
		return FogVisible
	}
	if col < 0 || row < 0 || col >= v.fogCols || row >= v.fogRows {
		return FogUnseen
	}
	i := row*v.fogCols + col
	if i < 0 || i >= len(v.fogPlane) {
		return FogUnseen
	}
	return v.fogPlane[i]
}

var fogScaleTable = [3]float32{
	FogUnseen:   0,
	FogExplored: 0.5,
	FogVisible:  1,
}

// fogScale is the shroud factor for one fog state (AC-8): 1 for visible, one
// half for explored, 0 for unseen.
//
// THE BOUNDS GUARD IS DEFENSIVE, not reachable through fogAt: the plane's
// only writer (pkg/game, 0118 T3) produces no byte outside 0..2, and fogAt
// returns only FogUnseen, FogExplored, FogVisible, one of the plane's own
// stored bytes, or nothing but those three constants.
func fogScale(state uint8) float32 {
	if int(state) >= len(fogScaleTable) {
		return 0
	}
	return fogScaleTable[state]
}

// SetFogReveal changes the temporary display override without editing fog.
func (v *Viewer) SetFogReveal(on bool) { v.fogReveal = on }

// ToggleFogReveal flips the reveal and reports the state it left it in —
// ToggleGrid's and ToggleReadout's own shape, applied to the reveal.
func (v *Viewer) ToggleFogReveal() bool {
	v.fogReveal = !v.fogReveal
	return v.fogReveal
}

// FogRevealed reports whether the reveal is on now.
func (v *Viewer) FogRevealed() bool { return v.fogReveal }

// fogGateEntity reports whether a unit at (col,row) owned by owner may be
// drawn now: a unit the local participant owns is drawn whatever its cell
// answers — command already reads the same comparison for the arming gate
// (command.go's canArmAttack) — and every other unit only while its cell
// is FogVisible right now, not merely explored.
func (v *Viewer) fogGateEntity(owner uint32, col, row int) bool {
	return owner == v.localOwner || v.fogAt(col, row) == FogVisible
}

// fogGateSack reports whether a ground sack at (col,row) may be drawn now:
// only while its cell is FogVisible, with no owner to except it — a sack
// belongs to nobody the way a unit belongs to its owner.
func (v *Viewer) fogGateSack(col, row int) bool {
	return v.fogAt(col, row) == FogVisible
}

// fogGateGround reports whether a structure or a static object at (col,row)
// may be drawn now (spec D-5): anything but FogUnseen. A building or a tree
// stays drawn once its cell has been explored, even when nothing stands
// there to see it right now — the disclosed divergence from a unit or a
// sack, both of which need the cell visible THIS INSTANT.
func (v *Viewer) fogGateGround(col, row int) bool {
	return v.fogAt(col, row) != FogUnseen
}

// fogGateStaticPlacements drops a placement whose own cell fails
// fogGateGround (spec D-5), leaving every survivor's order and identity
// untouched.
//
// IT RETURNS places UNCHANGED — same slice, same backing array — WHEN
// NOTHING IS DROPPED, and that is not an optimisation on top of a filter, it
// is the reason this exists as two passes rather than one: staticPlacements'
// own pointer-identity contract (viewer.go,
// TestSetFlatSelectsAListAndRebuildsNeither) promises a caller the very
// slice the builder produced whenever nothing changed it, and a fog-free
// viewer — every viewer built before this story, and every one this
// story's own tests do not call SetFog on — must keep that promise
// exactly. Since fogGateGround answers true for every cell while no plane is
// pushed (fogAt is FogVisible everywhere), the first pass below finds
// nothing to drop and the second pass never runs.
func (v *Viewer) fogGateStaticPlacements(places []terrain.StaticPlacement) []terrain.StaticPlacement {
	drop := false
	for _, p := range places {
		if !v.fogGateGround(p.Cell.X, p.Cell.Y) {
			drop = true
			break
		}
	}
	if !drop {
		return places
	}
	out := make([]terrain.StaticPlacement, 0, len(places))
	for _, p := range places {
		if v.fogGateGround(p.Cell.X, p.Cell.Y) {
			out = append(out, p)
		}
	}
	return out
}

// Structure fragments may reach an explored pixel while their ground cell is
// unseen. Keep the sheet pieces for camera culling and depth ordering; the
// final shroud clips their pixels against the projected terrain frontier.
func (v *Viewer) fogGateStructurePlacements(places []terrain.StructurePlacement) []terrain.StructurePlacement {
	return places
}
