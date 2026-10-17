package game

import (
	"image"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// fogPeriod is how many world ticks pass between one refresh of the visible
// layer and the next. A successful local Teleport also refreshes on arrival,
// as requested from owner observation.
//
// It is 32 because the decoded law's own map-wide clear runs on the
// presentation tick at that period, and our presentation clock is already
// driven from the world tick through SetLightClock — so refreshing on the
// world tick reproduces the shipped cadence with one clock fewer rather than
// inventing a period of our own. The consequence a player sees is D-3's own
// point in reverse: ground a unit has left stays lit until the period turns
// over rather than going dark the instant the unit steps off, exactly as the
// original's fog does.
//
// It is also load-bearing rather than merely faithful: one march is a 41x41
// window walk per living owned entity, so refreshing it on every tick rather
// than every 32nd would multiply that cost by 32 for no answer a player
// could tell apart from the periodic one.
const fogPeriod = 32

// localTeleportArrived reports a local Teleport that moved its living caster
// in this step. A blocked placement moves nothing and reports no relocation.
func (mw *mapWorld) localTeleportArrived(moved []sim.Relocation) bool {
	for _, r := range moved {
		if r.Owner != sim.SelfSlot || (r.FromX == r.ToX && r.FromY == r.ToY) {
			continue
		}
		if caster, ok := mw.entity(r.Caster); ok && caster.Owner == sim.SelfSlot && caster.HP > 0 && caster.X == r.ToX && caster.Y == r.ToY {
			return true
		}
	}
	return false
}

// fogPlane is one participant's whole fog state: which cells its own march
// lit on the last refresh, which cells have EVER been lit by any refresh, and
// the map's own dimensions those two layers are sized to.
//
// cols and rows are the WORLD's bounds (sim.World.Bounds()), not the render
// grid's: the plane indexes the same cells Sight's stamp does, and the two
// must agree without either side converting.
type fogPlane struct {
	cols, rows int
	visible    []byte
	explored   []byte
}

// exploredAt reports whether the local participant has ever seen one map
// cell. A nil or empty plane keeps the established no-fog behaviour: every
// cell is available, and the simulation remains responsible for map bounds.
// A populated plane fails closed outside its own measured extent.
func (p *fogPlane) exploredAt(x, y int) bool {
	if p == nil || len(p.explored) == 0 {
		return true
	}
	if x < 0 || y < 0 || x >= p.cols || y >= p.rows {
		return false
	}
	i := y*p.cols + x
	return i >= 0 && i < len(p.explored) && p.explored[i] != 0
}

// contains is the presentation plane's bounds check without its explored-state
// policy. A nil or empty plane leaves bounds to the simulation, matching the
// established no-fog path.
func (p *fogPlane) contains(x, y int) bool {
	if p == nil || len(p.explored) == 0 {
		return true
	}
	return x >= 0 && y >= 0 && x < p.cols && y < p.rows
}

func newFogPlane(cols, rows int) *fogPlane {
	n := cols * rows
	if n < 0 {
		n = 0
	}
	return &fogPlane{cols: cols, rows: rows, visible: make([]byte, n), explored: make([]byte, n)}
}

// refresh recomputes p's visible layer from w.Sight(owner) — pkg/sim's
// exported fog reader — and folds the result into explored.
//
// Original Fog import ORs its exploration through mapWorld.applyExplored.
// Native checkpoint restoration replaces both sampled planes before the first
// visible frame; it is a return to a saved state, not a gameplay refresh.
//
// The two slices are walked to the shorter of p's own length and the stamp
// Sight returns, rather than assumed equal: both are sized from the SAME
// world's bounds in the one caller that builds a plane (newMapWorldWith), so
// they agree in practice, and the bound here is the same safe disagreement
// R-4 already chose for pkg/ui's fogAt — answer from what was measured
// rather than index past either slice.
func (p *fogPlane) refresh(w *sim.World, owner uint32) {
	stamp := w.Sight(owner)
	n := len(p.visible)
	if len(stamp) < n {
		n = len(stamp)
	}
	for i := 0; i < n; i++ {
		lit := stamp[i] != 0
		if lit {
			p.visible[i] = 1
			p.explored[i] = 1
		} else {
			p.visible[i] = 0
		}
	}
}

// project returns the three-state plane pkg/ui's SetFog wants: ui.FogVisible
// where p's visible layer is lit, else ui.FogExplored where p's explored
// layer is lit, else ui.FogUnseen — one fresh slice per call, so the
// caller may hand it across the render seam without either side aliasing the
// other's memory.
//
// USING ui.Fog* HERE RATHER THAN 0/1/2 is not this package borrowing
// pkg/ui's meaning for its own state — p.visible and p.explored above stay
// plain bytes, and this is the one function that translates them at the
// seam. internal/archtest's allow-map permits pkg/game to import pkg/ui
// already (this package's own view field is a *ui.Viewer), so naming the
// three constants a caller a few lines away already reads is the plainer
// spelling, not a new dependency.
func (p *fogPlane) project() []byte {
	out := make([]byte, len(p.visible))
	for i, v := range p.visible {
		switch {
		case v != 0:
			out[i] = ui.FogVisible
		case p.explored[i] != 0:
			out[i] = ui.FogExplored
		default:
			out[i] = ui.FogUnseen
		}
	}
	return out
}

// stampMoved is the original's per-tick stamp (TERR-TILE-079): a drawable of
// the local player whose cell differs from the cell it last stamped at ORs its
// sight into the visible and explored layers. Nothing is cleared; the clear
// stays with the period. The engine moves a body onto its next cell at the
// first tick of the crossing, so that tick is the point of the move.
//
// A body met for the first time only records its cell: the opening refresh,
// the period and the teleport arrival already light it. The constructor
// records every body once, so the first step already compares. The second game's
// map screen keeps the periodic refresh alone.
func (mw *mapWorld) stampMoved() {
	if mw.fog == nil {
		return
	}
	if mw.world.SecondGame() {
		return
	}
	if mw.stamped == nil {
		mw.stamped = make(map[sim.EntityID]image.Point)
	}
	var moved []sim.EntityID
	for _, e := range mw.world.EntityView() {
		if e.Owner != sim.SelfSlot {
			continue
		}
		at := image.Point{X: int(e.X), Y: int(e.Y)}
		if last, seen := mw.stamped[e.ID]; seen && last != at {
			moved = append(moved, e.ID)
		}
		mw.stamped[e.ID] = at
	}
	if len(moved) == 0 {
		return
	}
	mw.fog.fold(mw.world.SightOf(sim.SelfSlot, moved))
}

// stampAfterLoad is the party's first stamp after a LOAD (TERR-FOG-268): a
// freshly created drawable stamps on its first presentation call, so before
// the first visible frame the party's sight is ORed into the visible and the
// explored layers on top of the restored Fog plane. Every first-game LOAD
// takes it; the second game's map screen keeps the periodic refresh alone.
func (mw *mapWorld) stampAfterLoad() {
	if mw.fog == nil || mw.world == nil || mw.world.SecondGame() {
		return
	}
	mw.fog.fold(mw.world.Sight(sim.SelfSlot))
	if mw.view != nil {
		mw.push()
	}
}

// fold ORs a stamp into the visible and explored layers without clearing.
func (p *fogPlane) fold(stamp []byte) {
	n := len(p.visible)
	if len(stamp) < n {
		n = len(stamp)
	}
	for i := 0; i < n; i++ {
		if stamp[i] != 0 {
			p.visible[i] = 1
			p.explored[i] = 1
		}
	}
}
