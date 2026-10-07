package sim

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// SavedCellPlanes holds original plane bytes independently of the translated
// native grid. CostKnown=0 is explicit missing authority, not a zero cost.
// Costs is the original Cost[0..10] table; recomputation uses Costs[5].
type SavedCellPlanes struct {
	Cost, Static, Dynamic, Height [65536]byte
	CostKnown                     [65536]byte
	Costs                         [11]byte
}

const savedCellPlanePayloadLen = 5*65536 + 11

func (w *World) SavedCellPlanes() (*SavedCellPlanes, bool) {
	if w.savedCellPlanes == nil {
		return nil, false
	}
	out := *w.savedCellPlanes
	return &out, true
}

// ImportOriginalCellPlanes installs caller-provided original LOAD state, not a
// constructor replay. Timestamp is not a construction guard. Adoption is atomic
// and one-time; native LOAD restores the already-persisted section instead.
func (w *World) ImportOriginalCellPlanes(planes *SavedCellPlanes) error {
	if w == nil || w.savedCellPlanes != nil {
		return fmt.Errorf("saved cell planes require a candidate without plane authority")
	}
	if planes == nil {
		return nil
	}
	if err := savedCellPlaneFault(planes); err != nil {
		return err
	}
	for _, tail := range w.cellTails {
		if tail.Bytes != [cellTailLen]byte{} && w.motionCell(tail.Key) == nil && planes.CostKnown[tail.Key] == 0 {
			return fmt.Errorf("native cell-tail construction lacks known current cost at %04x", tail.Key)
		}
	}
	out := *planes
	w.savedCellPlanes = &out
	// These are explicit current native tail owners, not actor geometry or an
	// inferred replay of original LOAD. Existing archived nodes are not rebuilt.
	for _, tail := range w.cellTails {
		w.syncSavedCellTail(tail.Key, tail.Bytes)
	}
	if w.savedMotion != nil {
		for _, c := range w.savedMotion.Cells {
			w.syncSavedStructureCell(c)
		}
	}
	w.syncNativeSavedPlaneGrid()
	w.refreshSavedPlaneBlocks()
	return nil
}

// ConstructSavedCellPlanes executes the shared cell recompute for the new
// world's already registered native owners. Original LOAD never calls it.
func (w *World) ConstructSavedCellPlanes(planes *SavedCellPlanes) error {
	if w == nil || w.tick != 0 || w.savedCellPlanes != nil || w.savedMotion == nil {
		return fmt.Errorf("saved cell construction requires fresh spatial authority")
	}
	staged := *w
	staged.savedMotion = cloneActorMotions(w.savedMotion)
	staged.grid = append([]byte(nil), w.grid...)
	staged.savedStructureCells = append([]SavedStructureCell(nil), w.savedStructureCells...)
	if err := staged.ImportOriginalCellPlanes(planes); err != nil {
		return err
	}
	for _, cell := range staged.savedMotion.Cells {
		if issue := staged.recomputeSavedCell(cell.Cell); issue != "" {
			return fmt.Errorf("saved cell %04x: %s", cell.Cell, issue)
		}
	}
	staged.refreshSavedPlaneBlocks()
	*w = staged
	return nil
}

func savedCellPlaneFault(p *SavedCellPlanes) error {
	if p == nil {
		return nil
	}
	if p.Costs[0] != 255 {
		return fmt.Errorf("saved cell cost table class zero must be 255")
	}
	for _, known := range p.CostKnown {
		if known > 1 {
			return fmt.Errorf("saved cell cost knowledge is not boolean")
		}
	}
	return nil
}

// The native grid and saved Block records are projections of current raw
// authority. Refuse conflicting native envelopes instead of choosing a reader.
func (w *World) savedCellPlaneStateFault() error {
	p := w.savedCellPlanes
	if err := savedCellPlaneFault(p); err != nil || p == nil {
		return err
	}
	for _, tail := range w.cellTails {
		if c := w.motionCell(tail.Key); c != nil {
			for j, value := range tail.Bytes {
				if c.Payload[0x2c+j] != value {
					return fmt.Errorf("saved cell payload disagrees with current tail at %04x", tail.Key)
				}
			}
		} else if tail.Bytes != [cellTailLen]byte{} {
			// An unresolved later native tail creation remains saveable only
			// with the explicit unknown-cost motion issue that disclosed it.
			knownGap := false
			if p.CostKnown[tail.Key] == 0 && w.savedMotion != nil {
				for _, m := range w.savedMotion.Motions {
					if m.Issue == "new cell construction lacks known current cost" {
						knownGap = true
					}
				}
			}
			if !knownGap {
				return fmt.Errorf("current nonzero cell tail has no Cell record at %04x", tail.Key)
			}
		}
	}
	for y := int32(0); y < w.bounds.Height && y < 256; y++ {
		for x := int32(0); x < w.bounds.Width && x < 256; x++ {
			key := uint16(y)<<8 | uint16(x)
			at, _ := w.cellIndex(x, y)
			if w.grid[at]&^blockMagicWall != p.Static[key]&3|(p.Static[key]&4)<<1 {
				return fmt.Errorf("saved cell static plane disagrees with native grid at %04x", key)
			}
		}
	}
	if w.savedMotion != nil {
		at := 0
		for key := 0x807; key <= 0xeded; key++ {
			if p.Dynamic[key] <= 15 {
				continue
			}
			want := SavedActorBlock{Cell: uint16(key), Dyn: p.Dynamic[key], Static: p.Static[key]}
			if at >= len(w.savedMotion.Blocks) || w.savedMotion.Blocks[at] != want {
				return fmt.Errorf("saved cell planes disagree with current Block records at %04x", key)
			}
			at++
		}
		if at != len(w.savedMotion.Blocks) {
			return fmt.Errorf("saved cell planes have extra current Block records")
		}
	}
	return nil
}

func (w *World) syncSavedCellTail(key uint16, tail [cellTailLen]byte) {
	if w.savedCellPlanes == nil {
		return
	}
	created := false
	if w.motionCell(key) == nil && tail != [cellTailLen]byte{} {
		if issue := w.createSavedCell(key); issue != "" {
			if w.savedMotion != nil {
				for i := range w.savedMotion.Motions {
					w.savedMotion.Motions[i].Issue = issue
				}
			}
			return
		}
		created = true
	}
	if c := w.motionCell(key); c != nil {
		copy(c.Payload[0x2c:0x32], tail[:])
		if created {
			w.recomputeSavedCell(key)
			w.refreshSavedPlaneBlocks()
		}
	}
}

func (w *World) appendSavedCellPlanes(b []byte) []byte {
	if p := w.savedCellPlanes; p != nil {
		b = append(b, p.Cost[:]...)
		b = append(b, p.Static[:]...)
		b = append(b, p.Dynamic[:]...)
		b = append(b, p.Height[:]...)
		b = append(b, p.CostKnown[:]...)
		b = append(b, p.Costs[:]...)
		return binary.LittleEndian.AppendUint32(b, savedCellPlanePayloadLen)
	}
	return binary.LittleEndian.AppendUint32(b, 0)
}

func splitSavedCellPlanes(data []byte) ([]byte, *SavedCellPlanes, error) {
	if len(data) < headerLen+4 {
		return nil, nil, fmt.Errorf("truncated saved cell plane footer")
	}
	end := len(data) - 4
	n := binary.LittleEndian.Uint32(data[end:])
	if n == 0 {
		return data[:end], nil, nil
	}
	if n != savedCellPlanePayloadLen || end-headerLen < savedCellPlanePayloadLen {
		return nil, nil, fmt.Errorf("invalid fixed saved cell plane span")
	}
	start := end - savedCellPlanePayloadLen
	src := data[start:end]
	p := new(SavedCellPlanes)
	for _, dst := range [][]byte{p.Cost[:], p.Static[:], p.Dynamic[:], p.Height[:], p.CostKnown[:], p.Costs[:]} {
		copy(dst, src)
		src = src[len(dst):]
	}
	if err := savedCellPlaneFault(p); err != nil {
		return nil, nil, err
	}
	return data[:start], p, nil
}

func savedPlaneKey(x, y int32) (uint16, bool) {
	if x < 0 || x > 255 || y < 0 || y > 255 {
		return 0, false
	}
	return uint16(y)<<8 | uint16(x), true
}

func savedStaticMask(d Domain) byte {
	if d == DomainGround {
		return 1
	}
	if d == DomainGhost {
		return 4
	}
	return 2
}

// The native grid stays a translated routing mirror; original bits do not
// enter its reserved bits. Canonical raw planes remain the source authority.
func (w *World) syncNativeSavedPlaneCell(key uint16) {
	if p := w.savedCellPlanes; p != nil {
		if at, ok := w.cellIndex(int32(key&255), int32(key>>8)); ok {
			w.grid[at] = p.Static[key]&3 | (p.Static[key]&4)<<1 | w.grid[at]&blockMagicWall
		}
	}
}

func (w *World) syncNativeSavedPlaneGrid() {
	if w.savedCellPlanes == nil {
		return
	}
	for y := int32(0); y < w.bounds.Height && y < 256; y++ {
		for x := int32(0); x < w.bounds.Width && x < 256; x++ {
			w.syncNativeSavedPlaneCell(uint16(y)<<8 | uint16(x))
		}
	}
}

// TERR-PASS-053 stores a flat, inclusive range, not a rectangular interior.
func (w *World) refreshSavedPlaneBlocks() {
	if w.savedCellPlanes == nil || w.savedMotion == nil {
		return
	}
	out := make([]SavedActorBlock, 0, len(w.savedMotion.Blocks))
	for k := 0x807; k <= 0xeded; k++ {
		if d := w.savedCellPlanes.Dynamic[k]; d > 15 {
			out = append(out, SavedActorBlock{Cell: uint16(k), Dyn: d, Static: w.savedCellPlanes.Static[k]})
		}
	}
	w.savedMotion.Blocks = out
}

func (w *World) savedCellPayload(c SavedActorCell) [52]byte {
	p := c.Payload
	at := sort.Search(len(w.cellTails), func(i int) bool { return w.cellTails[i].Key >= c.Cell })
	if at < len(w.cellTails) && w.cellTails[at].Key == c.Cell {
		copy(p[0x2c:0x32], w.cellTails[at].Bytes[:])
	}
	return p
}

// This lookup uses the exact current source key, never the geometric cache.
func (w *World) savedCellBuilding(key uint32) (Structure, SavedStructure, bool) {
	for _, source := range w.savedStructures {
		if source.SourceKey == key {
			at := indexOfStructure(w.structures, source.ID)
			if at >= 0 {
				return w.structures[at], source, true
			}
		}
	}
	return Structure{}, SavedStructure{}, false
}

func (w *World) savedCellRecomputeIssue(c SavedActorCell) string {
	if key := binary.LittleEndian.Uint32(c.Payload[12:]); key != 0 {
		if _, _, found := w.savedCellBuilding(key); !found {
			return "cell recomputation has an unresolved Building key"
		}
	}
	return ""
}

// savedCellCost is the cost byte a recompute of the cell leaves: its baseline,
// the cracked-ground cost where a Building footprint cell opens, then one
// two-bit left shift per occupied layer slot (TERR-STRUCT-071, MAGIC-AREACOST-046).
func (w *World) savedCellCost(c SavedActorCell) uint8 {
	p := w.savedCellPlanes
	payload := w.savedCellPayload(c)
	cost := payload[0]
	if buildingKey := binary.LittleEndian.Uint32(payload[12:]); buildingKey != 0 {
		b, s, _ := w.savedCellBuilding(buildingKey)
		bit := uint32((int32(c.Cell>>8)-b.Row)*int32(b.Width)+int32(c.Cell&255)-b.Col) & 31
		if s.Blocking&(uint32(1)<<bit) == 0 {
			cost = p.Costs[5]
		}
	}
	for layer, spell := range areaLayerSpells {
		if binary.LittleEndian.Uint32(payload[20+4*layer:]) != 0 || w.nativeAreaLayerPresent(c.Cell, spell) {
			cost <<= 2
		}
	}
	return cost
}

// TERR-STRUCT-078: byte-width shifts deliberately wrap. All six slots count
// independently of the retained layer-count byte; the fourth also blocks.
func (w *World) recomputeSavedCell(key uint16) string {
	c := w.motionCell(key)
	p := w.savedCellPlanes
	if c == nil || p == nil {
		return ""
	}
	if issue := w.savedCellRecomputeIssue(*c); issue != "" {
		return issue
	}
	c.Payload = w.savedCellPayload(*c)
	cost, stat := w.savedCellCost(*c), c.Payload[1]|0x20
	dyn := stat
	carry := p.Static[key] & 0x10
	if c.Ground.Key != 0 {
		dyn |= 0x40
	}
	if c.Air.Key != 0 {
		dyn |= 0x80
	}
	if buildingKey := binary.LittleEndian.Uint32(c.Payload[12:]); buildingKey != 0 {
		b, s, _ := w.savedCellBuilding(buildingKey)
		bit := uint32((int32(key>>8)-b.Row)*int32(b.Width)+int32(key&255)-b.Col) & 31
		if s.Blocking&(uint32(1)<<bit) != 0 {
			stat |= 5
			dyn |= 5
		} else {
			stat &= 0xfa
			dyn &= 0xfa
		}
	}
	if binary.LittleEndian.Uint32(c.Payload[32:]) != 0 || w.nativeAreaLayerPresent(key, 19) {
		stat |= 5
		dyn |= 5
	}
	w.writeSavedCost(key, cost)
	p.Static[key], p.Dynamic[key] = stat|carry, dyn|carry
	w.syncSavedStructureCell(*c)
	w.syncNativeSavedPlaneCell(key)
	return ""
}

func (w *World) syncSavedStructureCell(c SavedActorCell) {
	if !w.hasSavedStructures {
		return
	}
	out := SavedStructureCell{Cell: c.Cell, BaselineCost: c.Payload[0], BaselineStatic: c.Payload[1]}
	if key := binary.LittleEndian.Uint32(c.Payload[12:]); key != 0 {
		if b, _, ok := w.savedCellBuilding(key); ok {
			out.ID, out.HasStructure = b.ID, true
		}
	}
	at := sort.Search(len(w.savedStructureCells), func(i int) bool { return w.savedStructureCells[i].Cell >= c.Cell })
	if at < len(w.savedStructureCells) && w.savedStructureCells[at].Cell == c.Cell {
		w.savedStructureCells[at] = out
	} else {
		w.savedStructureCells = append(w.savedStructureCells, SavedStructureCell{})
		copy(w.savedStructureCells[at+1:], w.savedStructureCells[at:])
		w.savedStructureCells[at] = out
	}
	w.rebuildStructureSlots()
}

func (w *World) createSavedCell(key uint16) string {
	if w.motionCell(key) != nil {
		return ""
	}
	p := w.savedCellPlanes
	if p == nil || p.CostKnown[key] == 0 {
		return "new cell construction lacks known current cost"
	}
	// SAV-CELLENTRY-582: zero52, then capture both baselines before setting
	// this-record bit on the current static plane. Existing records are reused.
	c := SavedActorCell{Cell: key}
	c.Payload[0], c.Payload[1] = p.Cost[key], p.Static[key]
	if w.savedMotion == nil {
		w.savedMotion = &savedActorMotionState{}
	}
	at := sort.Search(len(w.savedMotion.Cells), func(i int) bool { return w.savedMotion.Cells[i].Cell >= key })
	w.savedMotion.Cells = append(w.savedMotion.Cells, SavedActorCell{})
	copy(w.savedMotion.Cells[at+1:], w.savedMotion.Cells[at:])
	w.savedMotion.Cells[at] = c
	p.Static[key] |= 0x20
	w.syncSavedStructureCell(c)
	w.syncNativeSavedPlaneCell(key)
	return ""
}

func (w *World) deleteEmptySavedCell(key uint16) bool {
	c := w.motionCell(key)
	p := w.savedCellPlanes
	if c == nil || p == nil {
		return false
	}
	c.Payload = w.savedCellPayload(*c)
	if !motionCellDeletionEligible(c.Payload) || w.nativeAreaCellPresent(key) {
		return false
	}
	carry := p.Static[key] & 0x10
	w.writeSavedCost(key, c.Payload[0])
	p.Static[key], p.Dynamic[key] = c.Payload[1]|carry, c.Payload[1]|carry
	at := sort.Search(len(w.savedMotion.Cells), func(i int) bool { return w.savedMotion.Cells[i].Cell >= key })
	w.savedMotion.Cells = append(w.savedMotion.Cells[:at], w.savedMotion.Cells[at+1:]...)
	w.removeCurrentCellRecord(key)
	if j := sort.Search(len(w.cellTails), func(i int) bool { return w.cellTails[i].Key >= key }); j < len(w.cellTails) && w.cellTails[j].Key == key {
		w.cellTails = append(w.cellTails[:j], w.cellTails[j+1:]...)
	}
	if w.hasSavedStructures {
		j := sort.Search(len(w.savedStructureCells), func(i int) bool { return w.savedStructureCells[i].Cell >= key })
		if j < len(w.savedStructureCells) && w.savedStructureCells[j].Cell == key {
			w.savedStructureCells = append(w.savedStructureCells[:j], w.savedStructureCells[j+1:]...)
		}
		w.rebuildStructureSlots()
	}
	w.syncNativeSavedPlaneCell(key)
	return true
}

func (w *World) planeMotionBoundaryIssue(i int, to uint16) string {
	e := w.entities[i]
	for _, anchor := range []uint16{uint16(e.Y)<<8 | uint16(e.X), to} {
		for dy := int32(0); dy < footprintSide(e.TokenSize); dy++ {
			for dx := int32(0); dx < footprintSide(e.TokenSize); dx++ {
				key, ok := savedPlaneKey(int32(anchor&255)+dx, int32(anchor>>8)+dy)
				if !ok {
					return "crossing reaches unsupported plane boundary"
				}
				if c := w.motionCell(key); c != nil {
					if issue := w.savedCellRecomputeIssue(*c); issue != "" {
						return issue
					}
				} else if anchor == to && w.savedCellPlanes.CostKnown[key] == 0 {
					return "new cell construction lacks known current cost"
				}
			}
		}
	}
	return ""
}

// Known lifecycle execution for an admitted boundary. The exceptional callback
// arms remain disclosed by the motion Issue; no callback return is invented.
func (w *World) movePlaneMotionSlots(i int, to uint16) bool {
	e := w.entities[i]
	actorKey := w.motionActorKey(e.ID)
	layer := e.Domain.layer()
	detached := false
detach:
	for dy := int32(0); dy < footprintSide(e.TokenSize); dy++ {
		for dx := int32(0); dx < footprintSide(e.TokenSize); dx++ {
			key := uint16(e.Y+dy)<<8 | uint16(e.X+dx)
			c := w.motionCell(key)
			if c == nil || motionSlot(c, layer).Key == 0 {
				break detach
			}
			// SAV-CELLLEAVE-584 clears the selected nonzero slot, even when
			// its saved pointer differs from the actor doing the detach.
			*motionSlot(c, layer) = SavedActorSlot{}
			binary.LittleEndian.PutUint32(c.Payload[4+4*layer:], 0)
			w.syncCurrentCellActor(key, layer, *motionSlot(c, layer))
			w.recomputeSavedCell(key)
			w.deleteEmptySavedCell(key)
			detached = true
		}
	}
	for dy := int32(0); dy < footprintSide(e.TokenSize); dy++ {
		for dx := int32(0); dx < footprintSide(e.TokenSize); dx++ {
			key := uint16(int32(to>>8)+dy)<<8 | uint16(int32(to&255)+dx)
			if issue := w.createSavedCell(key); issue != "" {
				w.motionFor(e.ID).Issue = issue
				return detached
			}
			c := w.motionCell(key)
			w.requestCellCast(i, int32(key&255), int32(key>>8))
			if motionSlot(c, layer).Key != 0 {
				// A taken slot refuses the entry through a body that does
				// nothing: no store, no recompute, and the loop over the later
				// cells ends. The step goes on (MOVE-088).
				return detached
			}
			*motionSlot(c, layer) = SavedActorSlot{Key: actorKey, Entity: e.ID, Bound: true}
			binary.LittleEndian.PutUint32(c.Payload[4+4*layer:], actorKey)
			w.syncCurrentCellActor(key, layer, *motionSlot(c, layer))
			w.recomputeSavedCell(key)
		}
	}
	return detached
}

func (w *World) refreshPlaneMotionReservations(i int, touched []uint16) {
	e := w.entities[i]
	layer := e.Domain.layer()
	mask := byte(0x40)
	if layer == 1 {
		mask = 0x80
	}
	for _, anchor := range touched {
		for dy := int32(0); dy < footprintSide(e.TokenSize); dy++ {
			for dx := int32(0); dx < footprintSide(e.TokenSize); dx++ {
				x, y := int32(anchor&255)+dx, int32(anchor>>8)+dy
				key, ok := savedPlaneKey(x, y)
				if !ok {
					continue
				}
				occupied := false
				if c := w.motionCell(key); c != nil {
					occupied = w.currentMotionSlot(*motionSlot(c, layer))
				}
				for j := range w.savedMotion.Motions {
					m := &w.savedMotion.Motions[j]
					at := indexOfEntity(w.entities, m.Entity)
					if at >= 0 && w.entities[at].Domain.layer() == layer && w.motionReserves(m, x, y) {
						occupied = true
					}
				}
				if occupied {
					w.savedCellPlanes.Dynamic[key] |= mask
				} else {
					w.savedCellPlanes.Dynamic[key] &^= mask
				}
			}
		}
	}
	w.refreshSavedPlaneBlocks()
}
