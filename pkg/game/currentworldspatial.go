package game

import (
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// fresh is Snapshot.NativeMissionTerrain, carried down from
// currentMissionDocument through currentWorldDocument -- the caller's own
// explicit choice, not whether a prior document exists. A first-ever SAVE of
// an ordinary, continuing mission also has no prior document, and that
// save's own round trip through this engine still needs the full per-cell
// backfill below to reload byte for byte: the reload's own baseline plane
// source (originalstructures.go's grid) is a different, evidence-based
// computation from this document's own currentSpatial/currentBlockPlaneDelta
// planes, and the two do not agree cell for cell outside the sweep window.
// Only a generated mission SAV meant for the original game itself sets this
// true, and only then does currentSpatial/currentBlockPlaneDelta's own
// sweep-window delta (SAV-BLOCK-011/TERR-PASS-053) stay unexpanded by the
// backfill below.
func projectCurrentTerrain(doc *sav.DocumentData, w *sim.World, fresh bool) {
	p := w.CurrentPolicy()
	if p.PlaneCarrier {
		return
	}
	blocks := make(map[uint16]sav.BlockRecord, len(doc.World.Blocks))
	for _, block := range doc.World.Blocks {
		blocks[block.Cell] = block
	}
	_, _, current, _ := w.SavedActorMotions()
	present := make(map[uint16]bool, len(current))
	for _, block := range current {
		blocks[block.Cell] = sav.BlockRecord{Cell: block.Cell, Dyn: block.Dyn, Static: block.Static}
		present[block.Cell] = true
	}
	if !fresh {
		width := int(w.Bounds().Width)
		for i, grid := range p.Terrain.Block {
			cell := uint16(i/width)<<8 | uint16(i%width)
			if present[cell] {
				continue
			}
			b := blocks[cell]
			b.Cell = cell
			stat := grid&3 | (grid&8)>>1
			b.Static = b.Static&^7 | stat
			b.Dyn = b.Dyn&^7 | stat
			blocks[cell] = b
		}
	}
	doc.World.Blocks = make([]sav.BlockRecord, 0, len(blocks))
	for _, block := range blocks {
		doc.World.Blocks = append(doc.World.Blocks, block)
	}
	slices.SortFunc(doc.World.Blocks, func(a, b sav.BlockRecord) int { return int(a.Cell) - int(b.Cell) })
}

func (b *generatedDocumentBuilder) currentActorCells(w *sim.World, m *alm.Map, src entrySource) error {
	planes, err := currentSpatialPlanes(w, m, src)
	if err != nil {
		return err
	}
	cells := map[uint16]sav.DocumentCellData{}
	for _, cell := range b.doc.World.Cells {
		cell.GroundActor, cell.AirActor = 0, 0
		cells[cell.Cell] = cell
	}
	indices := map[sim.EntityID]uint16{}
	for _, a := range b.state.Actors {
		if !a.Retired {
			indices[a.EntityID] = a.ObjectIndex
		}
	}
	for _, e := range w.Entities() {
		if !e.Alive() || e.OffMap {
			continue
		}
		identity, err := savedStructureValue(&b.doc.Objects[indices[e.ID]-1], "Identity")
		if err != nil {
			return err
		}
		for y := int32(0); y < int32(max(uint8(1), e.TokenSize)); y++ {
			for x := int32(0); x < int32(max(uint8(1), e.TokenSize)); x++ {
				key := uint16(e.X+x) | uint16(e.Y+y)<<8
				cell, ok := cells[key]
				if !ok {
					cell = sav.DocumentCellData{Cell: key, Cost: planes.Cost[key], Static: planes.Static[key]}
				}
				if e.Domain == sim.DomainAir {
					cell.AirActor = identity
				} else {
					cell.GroundActor = identity
				}
				cells[key] = cell
			}
		}
	}
	for _, row := range w.SavedCellRecords() {
		cell, ok := cells[row.Cell]
		if !ok {
			cell = sav.DocumentCellData{Cell: row.Cell, Cost: planes.Cost[row.Cell], Static: planes.Static[row.Cell]}
		}
		cell.GroundActor, cell.AirActor = row.Ground.Key, row.Air.Key
		cells[row.Cell] = cell
	}
	areas, err := w.NativeAreaSaveStates()
	if err != nil {
		return err
	}
	clouds := map[uint16]bool{}
	for _, area := range areas {
		if area.Mode != sim.AreaModeCloud {
			continue
		}
		for _, key := range area.Cells {
			if _, ok := cells[key]; !ok {
				cells[key] = sav.DocumentCellData{Cell: key, Cost: planes.Cost[key], Static: planes.Static[key] &^ 0x20}
			}
			clouds[key] = true
		}
	}
	b.doc.World.Cells = make([]sav.DocumentCellData, 0, len(cells))
	for _, cell := range cells {
		b.doc.World.Cells = append(b.doc.World.Cells, cell)
	}
	b.doc.World.Blocks = currentBlockPlaneDelta(planes, cells, clouds)
	slices.SortFunc(b.doc.World.Cells, func(a, b sav.DocumentCellData) int { return int(a.Cell) - int(b.Cell) })
	return nil
}

func (b *generatedDocumentBuilder) currentSpatial(w *sim.World, m *alm.Map, t *mapload.Table, src entrySource) error {
	planes, err := currentSpatialPlanes(w, m, src)
	if err != nil {
		return err
	}
	cells := map[uint16]sav.DocumentCellData{}
	cellAt := func(key uint16) sav.DocumentCellData {
		if c, ok := cells[key]; ok {
			return c
		}
		return sav.DocumentCellData{Cell: key, Cost: planes.Cost[key], Static: planes.Static[key]}
	}
	old, structureCells, hasStructureCells := w.SavedStructures()
	structureKeys := make(map[sim.StructureID]uint32, len(old))
	for _, st := range w.Structures() {
		var s sim.SavedStructure
		for _, v := range old {
			if v.ID == st.ID {
				s = v
				break
			}
		}
		if s.SourceKey == 0 {
			// SAV-1093: every verified original Building token carries
			// publication mask 2 at Token+0x18; zero leaves the structure undrawn.
			// The type word is T0E (Token+17); T0C (Token+16) stays 0, as every
			// original Building record carries it.
			s = sim.SavedStructure{ID: st.ID, Class: sim.GeneratedBuilding, SourceKey: b.identity(), RuntimeID: b.runtime(), Kind: uint8(st.Kind), Token0E: uint16(st.Kind), Token18: 2, Blocking: st.Blocking}
			copy(s.Position[:], constructedPositionBlock(st.Col, st.Row, b.doc.World.TerrainIdentity))
			s.Base52[14], s.Base52[15] = st.Width, st.Height
			binary.LittleEndian.PutUint32(s.Base52[18:], s.Blocking)
			for i, o := range m.Objects {
				if uint32(i) == uint32(st.ID) && o.Field12 != 0 {
					s.AuthoredIndex, s.AuthoredID, s.HasAuthored = uint32(i), uint32(o.Field12), true
				}
			}
		}
		r := mustNewRecord(savedStructureClass(s.Class))
		mustSetValue(&r, "Identity", s.SourceKey)
		if err := projectSavedStructureRecord(&r, s, st); err != nil {
			return err
		}
		index, err := b.append(r)
		if err != nil {
			return err
		}
		b.doc.World.Buildings = append(b.doc.World.Buildings, index)
		structureKeys[st.ID] = s.SourceKey
		if hasStructureCells {
			continue
		}
		for y := int32(0); y < int32(st.Height); y++ {
			for x := int32(0); x < int32(st.Width); x++ {
				if st.Attach>>(uint32(y*int32(st.Width)+x)&31)&1 == 0 {
					continue
				}
				key := uint16(st.Col+x) | uint16(st.Row+y)<<8
				v := cellAt(key)
				if v.Building != 0 {
					continue
				}
				v.Building = s.SourceKey
				// SAV-BLOCK-011/TERR-STRUCT-068: a building attachment cell the
				// terrain arms computation already reads as ground-blocked
				// (bit0) also carries static-object bit 2 in the original
				// encoding (0x05=bit0|bit2); an attachment cell the terrain
				// leaves unblocked stays a plain record cell (0x20 only), the
				// same as a sack or actor cell on open ground. Verified against
				// review/owner-sav-exp0397-r5/candidates/game9237.sav's own
				// block-plane delta: 112 cells at 25/25, 36 at 20/20, matching
				// this rule's split exactly. currentBlockPlaneDelta reads
				// planes.Static/Dynamic directly for a non-record cell and only
				// ORs the record bit onto that pair, so the plane array itself,
				// not just this cell's own DocumentCellData, needs the fold.
				if v.Static&1 != 0 {
					v.Static |= 4
					planes.Static[key] |= 4
					planes.Dynamic[key] = planes.Dynamic[key]&^7 | planes.Static[key]&7
				}
				cells[key] = v
			}
		}
	}
	if hasStructureCells {
		for _, cell := range structureCells {
			v := cellAt(cell.Cell)
			v.Cost, v.Static = cell.BaselineCost, cell.BaselineStatic
			if cell.HasStructure {
				v.Building = structureKeys[cell.ID]
				if v.Building == 0 {
					return fmt.Errorf("current structure cell %04x has no structure %d", cell.Cell, cell.ID)
				}
			}
			cells[cell.Cell] = v
		}
	}
	for _, a := range b.state.Actors {
		for _, e := range w.Entities() {
			if e.ID != a.EntityID || !e.Alive() || e.OffMap {
				continue
			}
			identity, _ := savedStructureValue(&b.doc.Objects[a.ObjectIndex-1], "Identity")
			for y := int32(0); y < int32(max(uint8(1), e.TokenSize)); y++ {
				for x := int32(0); x < int32(max(uint8(1), e.TokenSize)); x++ {
					key := uint16(e.X+x) | uint16(e.Y+y)<<8
					v := cellAt(key)
					if e.Domain == sim.DomainAir {
						v.AirActor = identity
					} else {
						v.GroundActor = identity
					}
					cells[key] = v
				}
			}
		}
	}
	registry := w.SavedObjects()
	heldSacks := map[uint16]uint32{}
	_, heldCells, _, _ := w.SavedActorMotions()
	for _, row := range heldCells {
		heldSacks[row.Cell] = binary.LittleEndian.Uint32(row.Payload[16:])
	}
	for _, row := range w.SavedCellRecords() {
		heldSacks[row.Cell] = row.Sack
	}
	for _, sack := range w.Sacks() {
		// The World's cell record holds the key of the Sack on its cell.
		key := heldSacks[uint16(sack.X)|uint16(sack.Y)<<8]
		if key == 0 || b.claimHeldKey(key) {
			key = b.identity()
		}
		r := mustNewRecord("Sack")
		mustSetToken(&r, nativeCityToken(key, 0, 0, 0))
		mustSetValue(&r, "RuntimeID", b.runtime())
		// Keep the full constructor shape. A row minted in play (a death sack)
		// has SavedUnknownToken and keeps the constructed token and key;
		// projectCurrentSackRoots reads its Identity from this record.
		known := false
		if registry != nil {
			for _, row := range registry.Sacks {
				if row.ID != sack.ObjectID || row.Token.Identity == 0 || row.Coverage.Unknown&sim.SavedUnknownToken != 0 {
					continue
				}
				tok := row.Token
				mustSetRaw(&r, "Block12", tok.Position[:])
				for _, v := range []sav.DocumentValueData{{Name: "RuntimeID", Value: tok.RuntimeID}, {Name: "T0C", Value: uint32(tok.T0C)}, {Name: "T0E", Value: uint32(tok.T0E)}, {Name: "T08", Value: tok.T08}, {Name: "T18", Value: uint32(tok.T18)}, {Name: "T1C", Value: tok.T1C}, {Name: "Identity", Value: tok.Identity}, {Name: "Reference", Value: tok.Reference}} {
					mustSetValue(&r, v.Name, v.Value)
				}
				key = tok.Identity
				known = true
				break
			}
		}
		if !known {
			mustSetRaw(&r, "Block12", constructedPositionBlock(sack.X, sack.Y, b.doc.World.TerrainIdentity))
		}
		// SAV-1093: every verified original Sack token carries publication
		// mask 2 at Token+0x18, bound or freshly generated; zero leaves the
		// sack's cell record absent (TERR-PASS-051/SAV-BLOCK-011) and its
		// pick-up unreachable.
		mustSetValue(&r, "T18", 2)
		mustSetValue(&r, "S3C", sack.Gold)
		var refs []uint16
		var weight int32
		var resolved []sim.ItemInstance
		if sack.ObjectID == 0 {
			items := sack.ItemInstances
			if len(items) == 0 {
				for _, code := range sack.Items {
					items = append(items, sim.PlainItem(code))
				}
			}
			for _, item := range items {
				value := constructedGeneratedItem(item, t)
				index, err := b.item(value, 1, key)
				if err != nil {
					return err
				}
				refs = append(refs, index)
				weight += int32(value.Weight)
				resolved = append(resolved, value)
			}
		}
		if !known {
			mustSetValue(&r, "T1C", sim.SackTokenValue(sack.Gold, resolved))
		}
		mustSetRefs(&r, "Contents", refs)
		mustSetCount(&r, "Contents", uint32(len(refs)))
		mustSetValue(&r, "Contents20", uint32(weight))
		index, err := b.append(r)
		if err != nil {
			return err
		}
		b.doc.World.Sacks = append(b.doc.World.Sacks, index)
		if sack.ObjectID != 0 {
			b.state.Objects.Sacks = append(b.state.Objects.Sacks, SnapshotSAVObjectBinding{ID: sack.ObjectID, ObjectIndex: index})
		} else {
			b.state.Objects.Unavailable = append(b.state.Objects.Unavailable, SnapshotSAVObjectCoverage{ObjectIndex: index, Reason: savedSackNativeUnavailable})
		}
		cell := uint16(sack.X) | uint16(sack.Y)<<8
		v := cellAt(cell)
		v.Sack = key
		cells[cell] = v
	}
	// Sack roots stay in cell order; binding rows are strictly ascending by ID.
	if b.state.Objects != nil {
		slices.SortFunc(b.state.Objects.Sacks, func(a, c SnapshotSAVObjectBinding) int { return cmp.Compare(a.ID, c.ID) })
	}
	for _, tail := range w.CellTails() {
		key := uint16(tail.X) | uint16(tail.Y)<<8
		v := cellAt(key)
		v.Operation, v.Power, v.SourceX, v.SourceY, v.TargetX, v.TargetY = tail.Bytes[0], tail.Bytes[1], tail.Bytes[2], tail.Bytes[3], tail.Bytes[4], tail.Bytes[5]
		cells[key] = v
	}
	// The World's cell records carry each archived cell's layer count, residue
	// spans and slot keys. The record cell stays present with those fields. A
	// motion cell's payload supplies the same span where no record holds it
	// (DIV-2476).
	_, motionCells, _, _ := w.SavedActorMotions()
	for _, row := range motionCells {
		v, p := cellAt(row.Cell), row.Payload
		v.LayerCount, v.Residue03, v.Residue32 = p[2], p[3], binary.LittleEndian.Uint16(p[50:])
		cells[row.Cell] = v
	}
	for _, row := range w.SavedCellRecords() {
		v := cellAt(row.Cell)
		v.LayerCount, v.Residue03 = row.LayerCount, row.Residue0
		v.Residue32 = binary.LittleEndian.Uint16(row.Residue1[:])
		v.GroundActor, v.AirActor, v.Layers = row.Ground.Key, row.Air.Key, row.SpellEffects
		if row.Sack != 0 {
			v.Sack = row.Sack
		}
		// A slot bound to a current actor names the key that actor is
		// written with.
		for _, slot := range []struct {
			s   sim.SavedCellActorSlot
			key *uint32
		}{{row.Ground, &v.GroundActor}, {row.Air, &v.AirActor}} {
			if !slot.s.Bound {
				continue
			}
			for _, a := range b.state.Actors {
				if a.EntityID == slot.s.Entity && !a.Retired {
					*slot.key, _ = savedStructureValue(&b.doc.Objects[a.ObjectIndex-1], "Identity")
				}
			}
		}
		cells[row.Cell] = v
	}
	areas, err := w.NativeAreaSaveStates()
	if err != nil {
		return err
	}
	clouds := map[uint16]bool{}
	for _, area := range areas {
		for _, key := range area.Cells {
			cell := cellAt(key)
			if _, present := cells[key]; !present && area.Mode == sim.AreaModeCloud {
				cell.Static &^= 0x20
			}
			cells[key] = cell
			if area.Mode == sim.AreaModeCloud {
				clouds[key] = true
			}
		}
	}
	recordCells := map[uint16]sav.DocumentCellData{}
	for _, v := range cells {
		b.doc.World.Cells = append(b.doc.World.Cells, v)
		// TERR-PASS-051/SAV-BLOCK-011: bit5 marks a cell with a hash-table
		// record (every sack, actor or Building cell has one); bit6/bit7 mark
		// a ground or air occupant on the dynamic plane only.
		if v.Sack != 0 || v.GroundActor != 0 || v.AirActor != 0 || v.Building != 0 || clouds[v.Cell] {
			recordCells[v.Cell] = v
		}
	}
	b.doc.World.Blocks = currentBlockPlaneDelta(planes, recordCells, clouds)
	slices.SortFunc(b.doc.World.Cells, func(a, b sav.DocumentCellData) int { return int(a.Cell) - int(b.Cell) })
	return nil
}

// currentBlockPlaneDelta is TERR-PASS-053/SAV-BLOCK-011's own writer: a row
// for cell c only inside the original writer's own sweep window,
// [0x807, 0x807+0xe5e7), and only where the cell's dynamic byte exceeds
// 0x0f -- the ingested border (0x1f) and footprint (0x05) bits, plus the
// record bit 0x20 this generator adds at every sack, actor or Building cell
// (TERR-PASS-051, SAV-1093/1097/1098), and the occupant bits 0x40/0x80 on a
// ground/air occupant. It is never the map's own full per-cell census: the
// array is a delta, and a plain terrain cell an ordinary LOAD already
// ingests correctly stays unlisted, exactly as the original writer leaves it.
func currentBlockPlaneDelta(planes *sim.SavedCellPlanes, record map[uint16]sav.DocumentCellData, clouds map[uint16]bool) []sav.BlockRecord {
	const sweepStart, sweepLen = 0x0807, 0xe5e7
	var out []sav.BlockRecord
	for i := 0; i < sweepLen; i++ {
		cell := uint16(sweepStart + i)
		static, dynamic := planes.Static[cell], planes.Dynamic[cell]
		if v, ok := record[cell]; ok {
			static |= 0x20
			if clouds[cell] && v.GroundActor|v.AirActor|v.Building|v.Sack == 0 {
				dynamic |= 0x20
			} else {
				dynamic = static
			}
			if v.GroundActor != 0 {
				dynamic |= 0x40
			}
			if v.AirActor != 0 {
				dynamic |= 0x80
			}
		}
		if dynamic <= 0x0f {
			continue
		}
		out = append(out, sav.BlockRecord{Cell: cell, Dyn: dynamic, Static: static})
	}
	return out
}

// Existing raw planes retain their full authority. Without that carrier the
// current native grid supplies its modeled fields; installed terrain supplies
// only cells and flag bits which that grid does not model.
func currentSpatialPlanes(w *sim.World, m *alm.Map, src entrySource) (*sim.SavedCellPlanes, error) {
	if planes, present := w.SavedCellPlanes(); present {
		return planes, nil
	}
	planes := &sim.SavedCellPlanes{}
	if src != nil {
		raw, err := src.ReadFile(worldPrefix + "data/map.reg")
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if err == nil {
			costs, err := data.OriginalTerrainCostTable(raw)
			if err != nil {
				return nil, err
			}
			planes, err = mapload.OriginalTerrainPlanes(m, costs)
			if err != nil {
				return nil, err
			}
		}
	}
	p := w.CurrentPolicy().Terrain
	width := int(w.Bounds().Width)
	for at, grid := range p.Block {
		key := uint16(at/width)<<8 | uint16(at%width)
		stat := grid&3 | (grid&8)>>1
		// TERR-STRUCT-068/072: a world that has already loaded through
		// originalstructures.go's ImportOriginalStructures carries domain 2's
		// scenery/border bit as grid bit3 already, so (grid&8)>>1 above is
		// enough. A genuinely fresh world -- no prior document, this cell's
		// grid straight from the mission's own opening terrain -- has not
		// folded that bit in yet, and grid still reads the map's raw domain
		// value in bits 0-1 the way originalstructures.go's own grid does
		// before its fold. Reading that raw value here, with the same
		// domain-2-or-Overlay test that fold uses, keeps this plane's Static
		// byte-identical to what a later LOAD of the same cell would
		// recompute, whether or not this particular export ever ran the fold.
		if grid&2 != 0 || (at < len(m.Overlay) && m.Overlay[at] != 0) {
			stat |= 4
		}
		planes.Static[key] = planes.Static[key]&^7 | stat
		planes.Dynamic[key] = planes.Dynamic[key]&^7 | planes.Static[key]&7
		planes.Cost[key], planes.Height[key], planes.CostKnown[key] = p.Cost[at], p.Height[at], 1
	}
	return planes, nil
}
