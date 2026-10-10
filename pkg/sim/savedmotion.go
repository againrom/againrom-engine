package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
	"sort"
)

// SavedActorPosition is the literal Token Position, including its independently
// stored packed-cell copy and otherwise uninterpreted residue/terrain key.
type SavedActorPosition struct {
	Cell, PackedCell uint16
	FineX, FineY     uint8
	Residue          uint16
	TerrainKey       uint32
}

// SavedActorMotion owns the current imported Position and mover, not a second
// renderer position. The two route lists are distinct from World.Route.
// Active executes only an admitted crossing to its first center. A centered
// pending turn uses the existing Mover bytes while Active stays false. Issue marks
// unsupported continuation/projection while preserving native saveability.
// Current becomes false when a later native producer supersedes this state.
type SavedActorMotion struct {
	Entity                    EntityID
	Position                  SavedActorPosition
	Mover                     [180]byte
	StaticRoute, DynamicRoute []uint16
	Current, Active           bool
	Issue                     string
	ActorAction               uint32
}

// SavedActorSlot separates an original numeric key from its exact native
// binding. Zero and unresolved keys are retained, never cast to EntityID.
type SavedActorSlot struct {
	Key    uint32
	Entity EntityID
	Bound  bool
}

// SavedActorCell retains an existing complete cell payload. Ground/Air are
// validated typed projections of its +04/+08 keys. No new payload is invented.
type SavedActorCell struct {
	Cell        uint16
	Ground, Air SavedActorSlot
	Payload     [52]byte
}

type SavedActorBlock struct {
	Cell        uint16
	Dyn, Static uint8
}

type savedActorMotionState struct {
	Motions []SavedActorMotion
	Cells   []SavedActorCell
	Blocks  []SavedActorBlock
}

func cloneActorMotions(s *savedActorMotionState) *savedActorMotionState {
	if s == nil {
		return nil
	}
	out := *s
	out.Motions, out.Cells, out.Blocks = slices.Clone(s.Motions), slices.Clone(s.Cells), slices.Clone(s.Blocks)
	for i := range out.Motions {
		out.Motions[i].StaticRoute = slices.Clone(out.Motions[i].StaticRoute)
		out.Motions[i].DynamicRoute = slices.Clone(out.Motions[i].DynamicRoute)
	}
	return &out
}

func (w *World) SavedActorMotions() ([]SavedActorMotion, []SavedActorCell, []SavedActorBlock, bool) {
	s := cloneActorMotions(w.savedMotion)
	if s == nil {
		return nil, nil, nil, false
	}
	return s.Motions, s.Cells, s.Blocks, true
}

func (w *World) motionFor(id EntityID) *SavedActorMotion {
	if w.savedMotion == nil {
		return nil
	}
	i := sort.Search(len(w.savedMotion.Motions), func(i int) bool { return w.savedMotion.Motions[i].Entity >= id })
	if i == len(w.savedMotion.Motions) || w.savedMotion.Motions[i].Entity != id {
		return nil
	}
	return &w.savedMotion.Motions[i]
}

func (w *World) motionActive(id EntityID) bool {
	m := w.motionFor(id)
	return m != nil && m.Current && (m.Active || savedTurnQueued(m))
}

func (w *World) ActorMotionActive(id EntityID) bool { return w.motionActive(id) }

func (w *World) ActorFinePosition(id EntityID) (uint8, uint8, bool) {
	m := w.motionFor(id)
	if m == nil || !m.Current {
		return 0, 0, false
	}
	return m.Position.FineX, m.Position.FineY, true
}

func (w *World) SavedActorMotionIssues() []string {
	var out []string
	if w.savedMotion != nil {
		for _, m := range w.savedMotion.Motions {
			if m.Issue != "" {
				out = append(out, fmt.Sprintf("actor %d: %s", m.Entity, m.Issue))
			}
		}
	}
	return out
}

// ImportOriginalActorMotions adopts a detached batch on the caller's private
// load candidate. Its restored session timestamp need not be zero. Existing
// motion authority cannot be replaced through this one-time import operation.
// Semantic unsupported cases stay readable with an issue; malformed bindings
// fail atomically. Nil in all three arguments explicitly means absent authority.
func (w *World) ImportOriginalActorMotions(motions []SavedActorMotion, cells []SavedActorCell, blocks []SavedActorBlock) error {
	if w == nil || w.savedMotion != nil {
		return fmt.Errorf("saved motion requires a candidate without motion authority")
	}
	if motions == nil && cells == nil && blocks == nil {
		w.savedMotion = nil
		return nil
	}
	s := cloneActorMotions(&savedActorMotionState{Motions: motions, Cells: cells, Blocks: blocks})
	sort.Slice(s.Motions, func(i, j int) bool { return s.Motions[i].Entity < s.Motions[j].Entity })
	// The game bridge supplies the final overlay per key, not archive history.
	sort.Slice(s.Cells, func(i, j int) bool { return s.Cells[i].Cell < s.Cells[j].Cell })
	sort.Slice(s.Blocks, func(i, j int) bool { return s.Blocks[i].Cell < s.Blocks[j].Cell })
	staged := *w
	staged.entities = slices.Clone(w.entities)
	staged.routes = slices.Clone(w.routes)
	staged.savedMotion = s
	for i := range s.Motions {
		m := &s.Motions[i]
		at := indexOfEntity(staged.entities, m.Entity)
		if at < 0 {
			return fmt.Errorf("saved motion names missing actor %d", m.Entity)
		}
		e := &staged.entities[at]
		nameClaimedRouteCell(&m.Mover)
		m.Current, m.Active, m.Issue = true, false, ""
		e.X, e.Y = int32(m.Position.Cell&255), int32(m.Position.Cell>>8)
		e.clearTransit()
		e.Facing = m.Mover[0]
		e.clearTurn()
		m.Issue = staged.motionAdmissionIssue(*m, *e)
		m.Active = m.Issue == "" && (m.Position.FineX != 128 || m.Position.FineY != 128)
		if !m.Active && m.Issue == "" && !pendingSavedMotionTurn(*m) {
			// A motion imported already centered has no crossing for
			// advanceSavedMotion to finish; this is its only entry point.
			if staged.beginSavedRouteContinuation(at, m) {
				if err := mintedContinuationFault(*m, *e, staged.routes[at]); err != nil {
					return fmt.Errorf("saved motion actor %d: %w", m.Entity, err)
				}
			}
		}
	}
	if err := staged.savedMotionFault(); err != nil {
		return err
	}
	w.entities, w.savedMotion, w.routes = staged.entities, s, staged.routes
	w.refreshSavedPlaneBlocks()
	return nil
}

// mintedContinuationFault mirrors, at the one moment they are still
// guaranteed true, the position and liveness invariants savedMotionFault
// applies to a Current motion. beginSavedRouteContinuation immediately
// clears Current, so savedMotionFault's own !m.Current branch never
// re-derives them for the motions this story mints; this stands in their
// place, checked once, here, before the staged import commits.
//
// It is deliberately NOT folded into savedMotionFault itself. Once native
// movement has taken even one step along the continued route, m.Position
// (frozen until death or splitSavedMotions restores it) and the entity's
// own X/Y (updated every step, advanceSavedMotion/step.go) disagree. Re-running this
// comparison at an arbitrary later Save or Load would refuse every ordinary
// save taken after the unit's first native step past where continuation
// began, not merely a hostile one.
//
// route is the exact []cell beginSavedRouteContinuation just wrote to
// w.routes[i]. World.UnmarshalBinary's decodeRoutes (binary.go) has two
// further refusals for a stored route besides the bounds/adjacency/domain
// ones motionAdmissionIssue already mirrors before continuation ever runs: a
// route on an entity with no target, and a route whose last cell is not that
// target (F-1). beginSavedRouteContinuation sets e.HasTarget and
// e.TargetX/TargetY from route's own last element in the same two statements
// that write w.routes[i], so neither shape is reachable through this call
// site today; both are checked here anyway; a future change that separated
// those two writes would otherwise silently mint a continuation the byte
// form refuses, exactly as the domain check's own absence once did.
func mintedContinuationFault(m SavedActorMotion, e Entity, route []cell) error {
	if e.X != int32(m.Position.Cell&255) || e.Y != int32(m.Position.Cell>>8) {
		return fmt.Errorf("minted continuation Position disagrees with the actor it began from")
	}
	if m.Position.Cell != m.Position.PackedCell {
		return fmt.Errorf("minted continuation Position cell representations disagree")
	}
	if !e.Alive() || e.OffMap {
		return fmt.Errorf("minted continuation names a dead or off-map actor")
	}
	if !e.HasTarget || len(route) == 0 {
		return fmt.Errorf("minted continuation route names an actor with no target")
	}
	if last := route[len(route)-1]; last.x != e.TargetX || last.y != e.TargetY {
		return fmt.Errorf("minted continuation route ends elsewhere than the actor's target")
	}
	return nil
}

func (w *World) motionAdmissionIssue(m SavedActorMotion, e Entity) string {
	if !e.Alive() || e.OffMap {
		return "original motion is not an on-map living crossing"
	}
	if m.Position.Cell != m.Position.PackedCell {
		return "Position cell representations disagree"
	}
	if _, ok := w.cellIndex(e.X, e.Y); !ok {
		return "original Position is outside the native map"
	}
	// StaticRoute/DynamicRoute elements are the packed cell `(y<<8)|x`
	// MOVE-ROUTE-004 defines and SAV-630 confirms by both the save and the
	// load arm; a hostile or truncated file can still claim a cell the
	// native map never had, which must refuse admission rather than index
	// out of bounds later.
	//
	// SAV-630's own corpus corroboration reads every one of 1,556 consecutive
	// element pairs WITHIN either list (1,146 static, 410 dynamic) as
	// Chebyshev-distance exactly 1 — never 0, never >1. beginSavedRouteContinuation
	// (pkg/sim/savedmotionroute.go) hands a list straight to w.routes[i], the
	// same wire form UnmarshalBinary refuses to decode a non-neighbour pair
	// into (binary.go's own "which are not neighbours" check): admission is
	// the place to refuse a file that violates this ahead of that later,
	// harder-to-diagnose failure, on the out-of-bounds check just above's own
	// precedent. This checks each list's OWN internal pairs only, never a
	// pair spanning the two lists: DIV-951 does not concatenate them.
	// A route cell this unit's own domain cannot cross is expected input:
	// MOVE-ROUTE-004 states the original's own route extractor "does not
	// re-test passability", so a route may name a closed cell. The route is
	// kept and the byte form holds it.
	for _, route := range [2][]uint16{m.StaticRoute, m.DynamicRoute} {
		prevX, prevY, has := int32(0), int32(0), false
		for _, packed := range route {
			x, y := int32(packed&0xff), int32(packed>>8)
			_, ok := w.cellIndex(x, y)
			if !ok {
				return "saved route names a cell outside the native map"
			}
			if has {
				dx, dy := x-prevX, y-prevY
				if dx < 0 {
					dx = -dx
				}
				if dy < 0 {
					dy = -dy
				}
				if dx > 1 || dy > 1 || (dx == 0 && dy == 0) {
					return "saved route has two consecutive cells that are not neighbours"
				}
			}
			prevX, prevY, has = x, y, true
		}
	}
	if m.Position.FineX == 128 && m.Position.FineY == 128 {
		if binary.LittleEndian.Uint16(m.Mover[0xaa:]) != 0 || binary.LittleEndian.Uint16(m.Mover[0xac:]) != 0 {
			return "centered mover has unsupported pending operands"
		}
		if pendingSavedMotionTurn(m) && m.Mover[10] == 0 {
			return "centered original turn has zero RotationSpeed"
		}
		return ""
	}
	o := w.savedOrder(e.ID)
	if o == nil || o.RepairStage != 0 || o.Raw[9] != 3 {
		return "noncentered mover lacks supported progress3"
	}
	if binary.LittleEndian.Uint16(m.Mover[0xaa:]) == 0 || binary.LittleEndian.Uint16(m.Mover[0xac:]) >= binary.LittleEndian.Uint16(m.Mover[0xaa:]) {
		return "active mover has unsupported elapsed/total"
	}
	if m.Mover[0xb0] == 0 && m.Mover[0xb1] == 0 || binary.LittleEndian.Uint16(m.Mover[0xae:]) > 7 {
		return "active mover has unsupported direction/steps"
	}
	if e.Turning() || w.bookCastInFlight(e.ID) || w.scrollInFlight(e.ID) {
		return "active mover conflicts with a native action"
	}
	return ""
}

func (w *World) savedMotionFault() error {
	s := w.savedMotion
	if s == nil {
		return nil
	}
	if len(s.Motions) > 1<<20 || len(s.Cells) > 65536 || len(s.Blocks) > 65536 {
		return fmt.Errorf("saved motion population exceeds bounds")
	}
	span := uint64(12) + uint64(len(s.Cells))*64 + uint64(len(s.Blocks))*4
	for i, m := range s.Motions {
		if i > 0 && s.Motions[i-1].Entity >= m.Entity || len(m.StaticRoute) > 65536 || len(m.DynamicRoute) > 65536 || len(m.Issue) > 256 {
			return fmt.Errorf("invalid saved motion identity/count/issue")
		}
		span += 211 + uint64(len(m.Issue)) + 2*uint64(len(m.StaticRoute)+len(m.DynamicRoute))
		if span > maxSavedMotionBytes {
			return fmt.Errorf("saved motion byte population exceeds bounds")
		}
		at := indexOfEntity(w.entities, m.Entity)
		if !m.Current {
			if m.Active || m.Issue == "" {
				return fmt.Errorf("superseded motion lacks its explicit issue")
			}
			continue
		}
		if at < 0 || w.entities[at].X != int32(m.Position.Cell&255) || w.entities[at].Y != int32(m.Position.Cell>>8) {
			return fmt.Errorf("saved motion Position disagrees with current actor")
		}
		if _, inside := w.cellIndex(w.entities[at].X, w.entities[at].Y); !inside && (m.Active || m.Issue == "") {
			return fmt.Errorf("out-of-map saved motion lacks its inactive issue")
		}
		if m.Issue == "" && (!w.entities[at].Alive() || w.entities[at].OffMap || m.Position.Cell != m.Position.PackedCell || !m.Active && (m.Position.FineX != 128 || m.Position.FineY != 128 || binary.LittleEndian.Uint16(m.Mover[0xaa:]) != 0 || binary.LittleEndian.Uint16(m.Mover[0xac:]) != 0 || pendingSavedMotionTurn(m) && m.Mover[10] == 0)) {
			return fmt.Errorf("unsupported saved motion lacks its explicit issue")
		}
		if m.Active && w.entities[at].OffMap {
			return fmt.Errorf("off-map actor has an active saved crossing")
		}
		if m.Active && (!w.entities[at].Alive() || m.Position.Cell != m.Position.PackedCell || m.Position.FineX == 128 && m.Position.FineY == 128 || binary.LittleEndian.Uint16(m.Mover[0xaa:]) == 0 || binary.LittleEndian.Uint16(m.Mover[0xac:]) >= binary.LittleEndian.Uint16(m.Mover[0xaa:]) || binary.LittleEndian.Uint16(m.Mover[0xae:]) > 7 || m.Mover[0xb0] == 0 && m.Mover[0xb1] == 0 || w.entities[at].Transit != 0 || w.entities[at].Stride.Present) {
			return fmt.Errorf("invalid active saved crossing")
		}
	}
	boundKeys := make(map[uint32]EntityID)
	actorKeys := make(map[EntityID]uint32)
	for i, c := range s.Cells {
		if i > 0 && s.Cells[i-1].Cell >= c.Cell {
			return fmt.Errorf("saved actor cells are not strictly ordered")
		}
		for layer, slot := range []SavedActorSlot{c.Ground, c.Air} {
			if slot.Key != binary.LittleEndian.Uint32(c.Payload[4+4*layer:]) || !slot.Bound && slot.Entity != 0 || slot.Bound && (slot.Key == 0 || w.motionFor(slot.Entity) == nil) {
				return fmt.Errorf("saved actor cell %04x has an invalid typed slot", c.Cell)
			}
			if !slot.Bound {
				continue
			}
			if prior, present := boundKeys[slot.Key]; present && prior != slot.Entity {
				return fmt.Errorf("saved actor key binds multiple actors")
			}
			if prior, present := actorKeys[slot.Entity]; present && prior != slot.Key {
				return fmt.Errorf("saved actor binds multiple original keys")
			}
			if at := indexOfEntity(w.entities, slot.Entity); at >= 0 {
				if key := w.entities[at].SourceBinding.Identity; key != 0 && key != slot.Key {
					return fmt.Errorf("saved actor cell key disagrees with source identity")
				}
			}
			boundKeys[slot.Key], actorKeys[slot.Entity] = slot.Entity, slot.Key
		}
	}
	for i, c := range s.Blocks {
		if i > 0 && s.Blocks[i-1].Cell >= c.Cell {
			return fmt.Errorf("saved actor block cells are not strictly ordered")
		}
	}
	return nil
}

func (w *World) invalidateActorMotion(id EntityID, reason string) {
	if m := w.motionFor(id); m != nil && m.Current {
		m.Current, m.Active, m.Issue = false, false, reason
	}
}

func (w *World) noteActorMotionOrder(id EntityID) {
	if m := w.motionFor(id); m != nil && m.Current {
		m.Issue = "native order supersedes original route continuation"
		if !m.Active {
			m.Current = false
		}
	}
}

func (w *World) motionCell(key uint16) *SavedActorCell {
	if w.savedMotion == nil {
		return nil
	}
	i := sort.Search(len(w.savedMotion.Cells), func(i int) bool { return w.savedMotion.Cells[i].Cell >= key })
	if i == len(w.savedMotion.Cells) || w.savedMotion.Cells[i].Cell != key {
		return nil
	}
	return &w.savedMotion.Cells[i]
}

func motionSlot(c *SavedActorCell, layer int) *SavedActorSlot {
	if layer == 0 {
		return &c.Ground
	}
	return &c.Air
}

func (w *World) currentMotionSlot(slot SavedActorSlot) bool {
	if slot.Key == 0 {
		return false
	}
	if !slot.Bound {
		return true
	}
	m := w.motionFor(slot.Entity)
	return m != nil && m.Current
}

// A reservation is a typed actor-owned footprint at the literal mover+80
// packed cell. Zero means absent under MOVE-CLAIM-007. No source pointer is
// interpreted here, and the actual saved slots remain independent.
func (w *World) motionReserves(m *SavedActorMotion, x, y int32) bool {
	if m == nil || !m.Current || !m.Active {
		return false
	}
	at := indexOfEntity(w.entities, m.Entity)
	if at < 0 {
		return false
	}
	for _, offset := range []int{0x80, 0xa6} {
		key := binary.LittleEndian.Uint16(m.Mover[offset:])
		if key != 0 && footprintsOverlap(x, y, 1, int32(key&255), int32(key>>8), w.entities[at].TokenSize) {
			return true
		}
	}
	return false
}

func (w *World) motionOwnsCell(id EntityID, layer int, x, y int32) bool {
	m := w.motionFor(id)
	if m == nil || !m.Current {
		return false
	}
	if x >= 0 && x < 256 && y >= 0 && y < 256 {
		if c := w.motionCell(uint16(y)<<8 | uint16(x)); c != nil {
			slot := motionSlot(c, layer)
			if slot.Bound && slot.Entity == id {
				return true
			}
		}
	}
	return w.motionReserves(m, x, y)
}

func (w *World) occupySavedMotions(s *routeScratch) {
	if w.savedMotion == nil {
		return
	}
	for _, c := range w.savedMotion.Cells {
		if at, ok := w.cellIndex(int32(c.Cell&255), int32(c.Cell>>8)); ok {
			for layer, slot := range []SavedActorSlot{c.Ground, c.Air} {
				if w.currentMotionSlot(slot) {
					s.occ[layer*int(gridCells(w.bounds))+at]++
				}
			}
		}
	}
	for i := range w.savedMotion.Motions {
		m := &w.savedMotion.Motions[i]
		at := indexOfEntity(w.entities, m.Entity)
		if !m.Current || !m.Active || at < 0 {
			continue
		}
		e := &w.entities[at]
		seen := map[uint16]bool{}
		for _, offset := range []int{0x80, 0xa6} {
			key := binary.LittleEndian.Uint16(m.Mover[offset:])
			if key == 0 {
				continue
			}
			for dy := int32(0); dy < footprintSide(e.TokenSize); dy++ {
				for dx := int32(0); dx < footprintSide(e.TokenSize); dx++ {
					x, y := int32(key&255)+dx, int32(key>>8)+dy
					if n, ok := w.cellIndex(x, y); ok {
						k := uint16(y)<<8 | uint16(x)
						if seen[k] {
							continue
						}
						seen[k] = true
						c := w.motionCell(k)
						if c == nil || !motionSlot(c, e.Domain.layer()).Bound || motionSlot(c, e.Domain.layer()).Entity != e.ID {
							s.occ[int(e.Domain.layer())*int(gridCells(w.bounds))+n]++
						}
					}
				}
			}
		}
	}
}

// advanceSavedMotion is the only active imported displacement. Its known step
// and cleanup follow MOVE-STEP-010, MOVE-CLOCK-032 and MOVE-CLAIM-007/REFRESH-012; complete
// boundary callbacks are not claimed by the current-state projection.
// New-cell construction, failed attachment and full cell recomputation are
// unimplemented branches, not unknown constructor laws (SAV-CELLENTRY-582).
func (w *World) advanceSavedMotion(i int, scratch *routeScratch) bool {
	e := &w.entities[i]
	m := w.motionFor(e.ID)
	if m == nil || !m.Current {
		return false
	}
	if !m.Active {
		advanced := w.advanceSavedTurn(i, m)
		if advanced && !m.Current {
			// Route handoff releases imported occupancy just as crossing
			// completion does. The native actor now owns its current cell.
			touched := []uint16{m.Position.Cell}
			for _, offset := range []int{0x80, 0xa6} {
				if key := binary.LittleEndian.Uint16(m.Mover[offset:]); key != 0 {
					touched = append(touched, key)
				}
			}
			w.refreshMotionBlocks(i, touched)
			scratch.occupy(w)
		}
		return advanced
	}
	// AI-ORDER-039/MOVE-STEP-040: progress3 publishes actor+54=1 on
	// this executed actor update, independently of reaching the center.
	nativeStride := m.Issue == "" && m.NativeStrideCompatible()
	m.ActorAction = 1
	p := m.Position
	touched := []uint16{p.Cell}
	for _, offset := range []int{0x70, 0x80, 0xa6} {
		if key := binary.LittleEndian.Uint16(m.Mover[offset:]); key != 0 {
			touched = append(touched, key)
		}
	}
	x := uint16(uint16(p.Cell&255)<<8|uint16(p.FineX)) + uint16(int16(int8(m.Mover[0xb0])))
	y := uint16(uint16(p.Cell>>8)<<8|uint16(p.FineY)) + uint16(int16(int8(m.Mover[0xb1])))
	dir := binary.LittleEndian.Uint16(m.Mover[0xae:])
	if !nativeStride && (dir == 1 || dir == 5) && uint8(x) == 0 && uint8(y) == 0 {
		x--
	}
	next := p
	next.Cell, next.PackedCell = uint16(y>>8)<<8|uint16(x>>8), uint16(y>>8)<<8|uint16(x>>8)
	next.FineX, next.FineY = uint8(x), uint8(y)
	entered := next // entry caches precede the final fractional snap
	elapsed := binary.LittleEndian.Uint16(m.Mover[0xac:]) + 1
	arrives := elapsed >= binary.LittleEndian.Uint16(m.Mover[0xaa:])
	if arrives {
		next.FineX, next.FineY = 128, 128
	}
	if next.Cell != p.Cell {
		if issue := w.motionBoundaryIssue(i, next.Cell); issue != "" {
			m.Active, m.Issue = false, issue
			return true
		}
		// MOVE-STEP-010 hands the old packed cell to the occupancy marker.
		// The qualified native stride instead retains its accepted origin:
		// a negative diagonal can cross its two near-cell boundaries on
		// adjacent ticks, without starting a second stride (DIV-1189).
		if !nativeStride {
			binary.LittleEndian.PutUint16(m.Mover[0xa6:], p.PackedCell)
		}
		if w.moveMotionSlots(i, next.Cell) {
			// SAV-CELLLEAVE-584: successful detach stores the actor's old
			// Position bytes0/1/4/5, not the queried footprint cell.
			copy(m.Mover[0x86:0x8a], []byte{byte(p.Cell), byte(p.Cell >> 8), p.FineX, p.FineY})
		}
		// SAV-CELLFAIL-583: the entry's cell and fraction cache writes
		// occur before its per-cell loop and before the caller's snap.
		copy(m.Mover[0x82:0x86], []byte{byte(entered.Cell), byte(entered.Cell >> 8), entered.FineX, entered.FineY})
		if m.Issue == "" && !nativeStride {
			m.Issue = "original boundary speed callback is not executed"
		}
		if w.savedCellPlanes == nil {
			w.requestFootprintCasts(i, int32(next.Cell&255), int32(next.Cell>>8))
		}
	}
	m.Position = next
	e.X, e.Y = int32(next.Cell&255), int32(next.Cell>>8)
	binary.LittleEndian.PutUint16(m.Mover[0xac:], elapsed)
	if arrives {
		m.Active = false
		m.ActorAction = 0 // SAV-CROSSNEXT-585: progress3 center completion
		for _, at := range []int{0xa8, 0xaa, 0xac, 0x80, 0xa6} {
			binary.LittleEndian.PutUint16(m.Mover[at:], 0)
		}
		if o := w.savedOrder(e.ID); o != nil {
			o.Raw[9] = 0
		}
		if pendingSavedMotionTurn(*m) && m.Mover[10] == 0 && m.Issue == "" {
			m.Issue = "centered original turn has zero RotationSpeed"
		}
		// A pending turn owns the following body updates before route
		// handoff. Other named gaps still defer the continuation (DIV-952).
		// Arrival consumes the dynamic list independently of the turn leaf.
		if m.Issue != "" || pendingSavedMotionTurn(*m) || !w.beginSavedRouteContinuation(i, m) {
			m.DynamicRoute = nil
		}
	}
	touched = append(touched, next.Cell)
	w.refreshMotionBlocks(i, touched)
	scratch.occupy(w)
	return true
}

// Update only existing block deltas touched by this actor's physical transition.
// The lower six dynamic bits and complete static byte remain source-owned.
// Missing delta records are not invented from translated native terrain.
func (w *World) refreshMotionBlocks(i int, touched []uint16) {
	if w.savedCellPlanes != nil {
		w.refreshPlaneMotionReservations(i, touched)
		return
	}
	e := w.entities[i]
	m := w.motionFor(e.ID)
	for j := range w.savedMotion.Blocks {
		b := &w.savedMotion.Blocks[j]
		x, y := int32(b.Cell&255), int32(b.Cell>>8)
		covered := false
		for _, key := range touched {
			if footprintsOverlap(x, y, 1, int32(key&255), int32(key>>8), e.TokenSize) {
				covered = true
				break
			}
		}
		if !covered {
			continue
		}
		// An existing cell record is the explicit owner of its actor bits.
		c := w.motionCell(b.Cell)
		if c == nil {
			continue
		}
		for layer, mask := range []byte{0x40, 0x80} {
			if layer != e.Domain.layer() {
				continue
			}
			occupied := w.currentMotionSlot(*motionSlot(c, layer))
			for k := range w.savedMotion.Motions {
				o := &w.savedMotion.Motions[k]
				at := indexOfEntity(w.entities, o.Entity)
				if at >= 0 && w.entities[at].Domain.layer() == layer && w.motionReserves(o, x, y) {
					occupied = true
				}
			}
			if occupied {
				b.Dyn |= mask
			} else {
				b.Dyn &^= mask
			}
		}
	}
	if m != nil {
		for _, c := range w.savedMotion.Cells {
			if !entityCoversCell(&e, int32(c.Cell&255), int32(c.Cell>>8)) {
				continue
			}
			at := sort.Search(len(w.savedMotion.Blocks), func(k int) bool { return w.savedMotion.Blocks[k].Cell >= c.Cell })
			if at == len(w.savedMotion.Blocks) || w.savedMotion.Blocks[at].Cell != c.Cell {
				m.Issue = "current actor occupancy requires an unrepresented block delta"
			}
		}
	}
}

func (w *World) motionBoundaryIssue(i int, cell uint16) string {
	e := w.entities[i]
	if w.savedCellPlanes != nil {
		if issue := w.planeMotionBoundaryIssue(i, cell); issue != "" {
			return issue
		}
	}
	if !w.terrainOpenFootprint(e, int32(cell&255), int32(cell>>8)) {
		return "crossing requires unsupported blocked terrain attachment"
	}
	n := footprintSide(e.TokenSize)
	for dy := int32(0); dy < n; dy++ {
		for dx := int32(0); dx < n; dx++ {
			x, y := int32(cell&255)+dx, int32(cell>>8)+dy
			if _, ok := w.cellIndex(x, y); !ok || x > 255 || y > 255 {
				return "crossing reaches unsupported map boundary"
			}
			c := w.motionCell(uint16(y)<<8 | uint16(x))
			if c == nil {
				if w.savedCellPlanes == nil {
					return "crossing needs unimplemented new cell construction"
				}
				continue
			}
			slot := motionSlot(c, e.Domain.layer())
			if w.savedCellPlanes == nil && slot.Key != 0 && (!slot.Bound || slot.Entity != e.ID) {
				return "crossing requires unsupported failed cell attachment"
			}
		}
	}
	if !w.occupancyOpenFootprint(newRouteScratch(w), i, int32(cell&255), int32(cell>>8)) {
		return "crossing requires unsupported contested reservation attachment"
	}
	if w.motionActorKey(e.ID) == 0 {
		return "crossing has no source actor key for new cell slots"
	}
	return ""
}

func (w *World) moveMotionSlots(i int, to uint16) bool {
	if w.savedCellPlanes != nil {
		return w.movePlaneMotionSlots(i, to)
	}
	e := w.entities[i]
	actorKey := w.motionActorKey(e.ID)
	layer := e.Domain.layer()
	detached := false
	for k := range w.savedMotion.Cells {
		c := &w.savedMotion.Cells[k]
		slot := motionSlot(c, layer)
		if slot.Bound && slot.Entity == e.ID && entityCoversCell(&e, int32(c.Cell&255), int32(c.Cell>>8)) {
			*slot = SavedActorSlot{}
			binary.LittleEndian.PutUint32(c.Payload[4+4*int(layer):], 0)
			w.syncCurrentCellActor(c.Cell, layer, *slot)
			detached = true
			// SAV-CELLLEAVE-584: residue and unused tail bytes do not
			// retain an otherwise empty node. Keeping it needs a visible
			// coverage gap until deletion and baseline restoration execute.
			if motionCellDeletionEligible(c.Payload) {
				w.motionFor(e.ID).Issue = "original empty cell deletion and baseline restoration are not executed"
			}
		}
	}
	for dy := int32(0); dy < footprintSide(e.TokenSize); dy++ {
		for dx := int32(0); dx < footprintSide(e.TokenSize); dx++ {
			key := uint16((int32(to>>8)+dy)*256 + int32(to&255) + dx)
			c := w.motionCell(key)
			*motionSlot(c, layer) = SavedActorSlot{Key: actorKey, Entity: e.ID, Bound: true}
			binary.LittleEndian.PutUint32(c.Payload[4+4*int(layer):], actorKey)
			w.syncCurrentCellActor(c.Cell, layer, *motionSlot(c, layer))
		}
	}
	return detached
}

func motionCellDeletionEligible(payload [52]byte) bool {
	if payload[2] != 0 || payload[0x2c] != 0 {
		return false
	}
	for _, offset := range []int{4, 8, 12, 16} {
		if binary.LittleEndian.Uint32(payload[offset:]) != 0 {
			return false
		}
	}
	return true
}

// Source identity or an independently supplied unique typed slot is the only
// admissible new slot key. EntityID is never serialized as an original key.
func (w *World) motionActorKey(id EntityID) uint32 {
	at := indexOfEntity(w.entities, id)
	if at < 0 {
		return 0
	}
	key := w.entities[at].SourceBinding.Identity
	for _, c := range w.savedMotion.Cells {
		for _, slot := range []SavedActorSlot{c.Ground, c.Air} {
			if slot.Bound && slot.Entity == id {
				if key != 0 && key != slot.Key {
					return 0
				}
				key = slot.Key
			}
		}
	}
	return key
}
