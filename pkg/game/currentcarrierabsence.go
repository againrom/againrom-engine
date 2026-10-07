package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentAbsentCellCarrier struct {
	Cell   uint16
	Anchor [32]byte
}

// A full byte-coordinate keyspace must fit the registry leaf. Each encoded
// row contains only its u16 key and SHA-256 digest, never cell values.
type currentCellAbsence []currentAbsentCellCarrier

// Block anchors cover only two ordinary bytes. A digest prefix keeps three
// full cell-keyspace absence populations within the bounded registry leaf.
type currentAbsentBlockCarrier struct {
	Cell   uint16
	Anchor [16]byte
}

type currentBlockAbsence []currentAbsentBlockCarrier

func (rows currentBlockAbsence) MarshalJSON() ([]byte, error) {
	if len(rows) > 65536 {
		return nil, fmt.Errorf("current block absence exceeds its keyspace")
	}
	var raw []byte
	if rows != nil {
		raw = make([]byte, 18*len(rows))
	}
	for i, row := range rows {
		binary.LittleEndian.PutUint16(raw[18*i:], row.Cell)
		copy(raw[18*i+2:], row.Anchor[:])
	}
	return json.Marshal(raw)
}

func (rows *currentBlockAbsence) UnmarshalJSON(data []byte) error {
	var raw []byte
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw)%18 != 0 || len(raw)/18 > 65536 {
		return fmt.Errorf("current block absence has invalid row framing")
	}
	var next currentBlockAbsence
	if raw != nil {
		next = make(currentBlockAbsence, len(raw)/18)
	}
	for i := range next {
		next[i].Cell = binary.LittleEndian.Uint16(raw[18*i:])
		copy(next[i].Anchor[:], raw[18*i+2:18*i+18])
	}
	*rows = next
	return nil
}

func (rows currentCellAbsence) MarshalJSON() ([]byte, error) {
	if len(rows) > 65536 {
		return nil, fmt.Errorf("current cell absence exceeds its keyspace")
	}
	var raw []byte
	if rows != nil {
		raw = make([]byte, 34*len(rows))
	}
	for i, row := range rows {
		binary.LittleEndian.PutUint16(raw[34*i:], row.Cell)
		copy(raw[34*i+2:], row.Anchor[:])
	}
	return json.Marshal(raw)
}

func (rows *currentCellAbsence) UnmarshalJSON(data []byte) error {
	var raw []byte
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw)%34 != 0 || len(raw)/34 > 65536 {
		return fmt.Errorf("current cell absence has invalid row framing")
	}
	var next currentCellAbsence
	if raw != nil {
		next = make(currentCellAbsence, len(raw)/34)
	}
	for i := range next {
		next[i].Cell = binary.LittleEndian.Uint16(raw[34*i:])
		copy(next[i].Anchor[:], raw[34*i+2:34*i+34])
	}
	*rows = next
	return nil
}

type currentAbsentDiary struct {
	Owner  sim.SavedDiaryOwner
	Object uint16
	Inline uint32
	Anchor [32]byte
}

type currentAbsentCellSack struct {
	Cell     uint16
	Wire     uint32
	Carriers uint8 `json:",omitempty"`
}

type currentDiaryRecord struct {
	Object uint16
	Inline uint32
	Record *sav.DocumentRecordData
}

func currentDiaryRecords(doc *sav.DocumentData, a *currentActionData, world *sim.World) (map[sim.SavedDiaryOwner]currentDiaryRecord, error) {
	state := &SnapshotSAVDocument{Document: doc}
	for _, b := range a.Bindings {
		if !b.Structure && !b.Missing {
			state.Actors = append(state.Actors, SnapshotSAVActor{EntityID: b.ID, ObjectIndex: b.Object})
		}
	}
	records, err := savedDiaryRecords(state, world)
	if err != nil {
		return nil, err
	}
	locations := map[*sav.DocumentRecordData]currentDiaryRecord{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		locations[r] = currentDiaryRecord{Object: uint16(i + 1), Record: r}
		for j := range r.Inline {
			child := &r.Inline[j].Record
			locations[child] = currentDiaryRecord{Object: uint16(i + 1), Inline: uint32(j), Record: child}
		}
	}
	out := make(map[sim.SavedDiaryOwner]currentDiaryRecord, len(records))
	for owner, record := range records {
		location, ok := locations[record]
		if !ok {
			return nil, fmt.Errorf("current Diary has no ordinary record location")
		}
		out[owner] = location
	}
	return out, nil
}

func currentCellCarrierAnchor(cell sav.DocumentCellData, tail bool) [32]byte {
	if tail {
		return sha256.Sum256([]byte{cell.Operation, cell.Power, cell.SourceX, cell.SourceY, cell.TargetX, cell.TargetY})
	}
	// Only the SavedCellRecord projection belongs to this carrier. Baselines,
	// Building and the six trigger bytes have separate current owners.
	raw := []byte{cell.LayerCount, cell.Residue03}
	for _, key := range append([]uint32{cell.GroundActor, cell.AirActor, cell.Sack}, cell.Layers[:]...) {
		raw = binary.LittleEndian.AppendUint32(raw, key)
	}
	raw = binary.LittleEndian.AppendUint16(raw, cell.Residue32)
	return sha256.Sum256(raw)
}

func currentBlockCarrierAnchor(block sav.BlockRecord) [16]byte {
	digest := sha256.Sum256([]byte{block.Dyn, block.Static})
	return [16]byte(digest[:16])
}

func currentMotionCellAnchors(cells []sav.DocumentCellData) map[uint16][32]byte {
	values := make(map[uint16][]byte, len(cells))
	for _, cell := range cells {
		payload := cell.Payload()
		values[cell.Cell] = append(values[cell.Cell], payload[:]...)
	}
	anchors := make(map[uint16][32]byte, len(values))
	for cell, raw := range values {
		anchors[cell] = sha256.Sum256(raw)
	}
	return anchors
}

func currentDiaryAnchor(record *sav.DocumentRecordData) ([32]byte, error) {
	if _, err := sav.ReadDocumentDiary(*record); err != nil {
		return [32]byte{}, err
	}
	raw, err := json.Marshal(record)
	return sha256.Sum256(raw), err
}

// Final keys matter for cell actor/layer operands and Diary Self. Capture the
// absence marker after key completion; no ordinary value is copied into policy.
func finalizeCurrentCarrierAbsence(doc *sav.DocumentData, worldBytes []byte) error {
	if doc == nil || doc.World == nil {
		return nil
	}
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		return err
	}
	var world sim.World
	if err := world.UnmarshalBinary(worldBytes); err != nil {
		return err
	}
	tails, records, sacks := map[uint16]bool{}, map[uint16]bool{}, map[uint16]uint32{}
	for _, row := range world.CellTails() {
		tails[uint16(row.X)|uint16(row.Y)<<8] = true
	}
	for _, row := range world.SavedCellRecords() {
		records[row.Cell] = true
		sacks[row.Cell] = row.Sack
	}
	_, motionCells, motionBlocks, motionPresent := world.SavedActorMotions()
	motionSacks := map[uint16]uint32{}
	for _, row := range motionCells {
		motionSacks[row.Cell] = binary.LittleEndian.Uint32(row.Payload[16:])
	}
	a.AbsentCellTails, a.AbsentCellRecords, a.AbsentDiaries = nil, nil, nil
	a.AbsentCellSacks = nil
	a.AbsentBlocks = nil
	a.AbsentMotionCells = nil
	if motionPresent {
		present := make(map[uint16]bool, len(motionBlocks))
		for _, row := range motionBlocks {
			present[row.Cell] = true
		}
		last := make(map[uint16]int, len(doc.World.Blocks))
		for i, block := range doc.World.Blocks {
			last[block.Cell] = i
		}
		for i, block := range doc.World.Blocks {
			if !present[block.Cell] && last[block.Cell] == i {
				a.AbsentBlocks = append(a.AbsentBlocks, currentAbsentBlockCarrier{Cell: block.Cell, Anchor: currentBlockCarrierAnchor(block)})
			}
		}
	}
	last := map[uint16]int{}
	motionAnchors := currentMotionCellAnchors(doc.World.Cells)
	for i, cell := range doc.World.Cells {
		last[cell.Cell] = i
	}
	for i, cell := range doc.World.Cells {
		if last[cell.Cell] != i {
			continue
		}
		if !tails[cell.Cell] {
			a.AbsentCellTails = append(a.AbsentCellTails, currentAbsentCellCarrier{Cell: cell.Cell, Anchor: currentCellCarrierAnchor(cell, true)})
		}
		if !records[cell.Cell] {
			a.AbsentCellRecords = append(a.AbsentCellRecords, currentAbsentCellCarrier{Cell: cell.Cell, Anchor: currentCellCarrierAnchor(cell, false)})
		}
		if _, present := motionSacks[cell.Cell]; motionPresent && !present {
			a.AbsentMotionCells = append(a.AbsentMotionCells, currentAbsentCellCarrier{Cell: cell.Cell, Anchor: motionAnchors[cell.Cell]})
		}
		if cell.Sack != 0 {
			var carriers uint8
			if records[cell.Cell] && sacks[cell.Cell] == 0 {
				carriers |= sim.CurrentSackKeyRecord
			}
			if value, present := motionSacks[cell.Cell]; present && value == 0 {
				carriers |= sim.CurrentSackKeyMotion
			}
			if carriers != 0 {
				a.AbsentCellSacks = append(a.AbsentCellSacks, currentAbsentCellSack{Cell: cell.Cell, Wire: cell.Sack, Carriers: carriers})
			}
		}
	}
	diaries, err := currentDiaryRecords(doc, a, &world)
	if err != nil {
		return err
	}
	present := map[sim.SavedDiaryOwner]bool{}
	for _, row := range world.SavedDiaries() {
		present[row.Owner] = true
	}
	for owner, record := range diaries {
		if present[owner] {
			continue
		}
		anchor, err := currentDiaryAnchor(record.Record)
		if err != nil {
			return err
		}
		a.AbsentDiaries = append(a.AbsentDiaries, currentAbsentDiary{Owner: owner, Object: record.Object, Inline: record.Inline, Anchor: anchor})
	}
	slices.SortFunc(a.AbsentDiaries, func(x, y currentAbsentDiary) int {
		if x.Owner.Player != y.Owner.Player {
			if x.Owner.Player {
				return -1
			}
			return 1
		}
		if x.Owner.Actor < y.Owner.Actor {
			return -1
		}
		if x.Owner.Actor > y.Owner.Actor {
			return 1
		}
		return 0
	})
	if err := validateCurrentCarrierAbsence(doc, a); err != nil {
		return err
	}
	if err := finalizeCurrentAbsentPlayers(doc, a); err != nil {
		return err
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, raw)
}

func validateCurrentCarrierAbsence(doc *sav.DocumentData, a *currentActionData) error {
	if len(a.AbsentCellTails)+len(a.AbsentCellRecords)+len(a.AbsentMotionCells)+len(a.AbsentDiaries)+len(a.AbsentCellSacks)+len(a.AbsentBlocks) == 0 {
		return nil
	}
	if doc == nil || doc.World == nil || len(a.AbsentCellTails) > 65536 || len(a.AbsentCellRecords) > 65536 || len(a.AbsentMotionCells) > 65536 || len(a.AbsentDiaries) > 65536 || len(a.AbsentCellSacks) > 65536 || len(a.AbsentBlocks) > 65536 {
		return fmt.Errorf("invalid current carrier absence population")
	}
	seenSacks := map[uint16]bool{}
	for _, row := range a.AbsentCellSacks {
		if seenSacks[row.Cell] || row.Wire == 0 || row.Carriers > sim.CurrentSackKeyRecord|sim.CurrentSackKeyMotion {
			return fmt.Errorf("current cell Sack absence has an invalid key")
		}
		seenSacks[row.Cell] = true
	}
	seenBlocks := map[uint16]bool{}
	for _, row := range a.AbsentBlocks {
		if seenBlocks[row.Cell] || row.Anchor == ([16]byte{}) {
			return fmt.Errorf("current block absence has a repeated key or invalid anchor")
		}
		seenBlocks[row.Cell] = true
	}
	for _, rows := range [][]currentAbsentCellCarrier{a.AbsentCellTails, a.AbsentCellRecords, a.AbsentMotionCells} {
		seen := map[uint16]bool{}
		for _, row := range rows {
			if seen[row.Cell] || row.Anchor == ([32]byte{}) {
				return fmt.Errorf("current cell absence has a repeated key or invalid anchor")
			}
			seen[row.Cell] = true
		}
	}
	motionCells := make(map[uint16]bool, len(a.AbsentMotionCells))
	for _, row := range a.AbsentMotionCells {
		motionCells[row.Cell] = true
	}
	for _, row := range a.AbsentCellSacks {
		if motionCells[row.Cell] && row.Carriers&sim.CurrentSackKeyMotion != 0 {
			return fmt.Errorf("current motion cell absence conflicts with a present Sack key carrier")
		}
	}
	for _, row := range a.CellCosts {
		if motionCells[row.Cell] {
			return fmt.Errorf("current motion cell absence conflicts with a present cost carrier")
		}
	}
	owners := map[sim.SavedDiaryOwner]bool{}
	for _, row := range a.AbsentDiaries {
		if row.Owner.Player && row.Owner.Actor != 0 || owners[row.Owner] || row.Object == 0 || int(row.Object) > len(doc.Objects) || row.Anchor == ([32]byte{}) {
			return fmt.Errorf("current Diary absence has an invalid owner, object or anchor")
		}
		owners[row.Owner] = true
		r := &doc.Objects[row.Object-1]
		if row.Owner.Player {
			if r.Class != "Player" || uint64(row.Inline) >= uint64(len(r.Inline)) || r.Inline[row.Inline].Record.Class != "Diary" {
				return fmt.Errorf("current Player Diary absence has no embedded record")
			}
		} else if row.Inline != 0 || r.Class != "Diary" {
			return fmt.Errorf("current actor Diary absence has no ordinary Diary")
		}
	}
	return nil
}

func restoreCurrentCarrierAbsence(world *sim.World, doc *sav.DocumentData, a *currentActionData) error {
	if err := validateCurrentCarrierAbsence(doc, a); err != nil {
		return err
	}
	if len(a.AbsentCellTails)+len(a.AbsentCellRecords)+len(a.AbsentMotionCells)+len(a.AbsentDiaries)+len(a.AbsentCellSacks)+len(a.AbsentBlocks) == 0 {
		return nil
	}
	last := map[uint16]sav.DocumentCellData{}
	for _, cell := range doc.World.Cells {
		last[cell.Cell] = cell
	}
	tailPresent := map[uint16]bool{}
	for _, row := range world.CellTails() {
		tailPresent[uint16(row.X)|uint16(row.Y)<<8] = true
	}
	recordSacks := map[uint16]uint32{}
	recordPresent := map[uint16]bool{}
	for _, row := range world.SavedCellRecords() {
		recordSacks[row.Cell] = row.Sack
		recordPresent[row.Cell] = true
	}
	_, motionRows, motionBlocks, _ := world.SavedActorMotions()
	motionCellPresent := map[uint16]bool{}
	motionSacks := map[uint16]uint32{}
	for _, row := range motionRows {
		motionCellPresent[row.Cell] = true
		motionSacks[row.Cell] = binary.LittleEndian.Uint32(row.Payload[16:])
	}
	motionBlockPresent := map[uint16]bool{}
	for _, row := range motionBlocks {
		motionBlockPresent[row.Cell] = true
	}
	var sacks []sim.CurrentSackKeyAbsence
	for _, row := range a.AbsentCellSacks {
		if cell, ok := last[row.Cell]; ok && cell.Sack == row.Wire {
			carriers := row.Carriers
			if carriers == 0 {
				carriers = sim.CurrentSackKeyRecord // Historical policy named only this carrier.
			}
			if carriers&sim.CurrentSackKeyRecord != 0 && recordSacks[row.Cell] != row.Wire {
				carriers &^= sim.CurrentSackKeyRecord
			}
			if carriers&sim.CurrentSackKeyMotion != 0 && motionSacks[row.Cell] != row.Wire {
				carriers &^= sim.CurrentSackKeyMotion
			}
			if carriers != 0 {
				sacks = append(sacks, sim.CurrentSackKeyAbsence{Cell: row.Cell, Wire: row.Wire, Carriers: carriers})
			}
		}
	}
	if err := world.RestoreCurrentSackKeyAbsence(sacks); err != nil {
		return err
	}
	matching := func(rows []currentAbsentCellCarrier, tail bool, present map[uint16]bool) []uint16 {
		var keys []uint16
		for _, row := range rows {
			if cell, ok := last[row.Cell]; ok && present[row.Cell] && currentCellCarrierAnchor(cell, tail) == row.Anchor {
				keys = append(keys, row.Cell)
			}
		}
		return keys
	}
	diaries, err := currentDiaryRecords(doc, a, world)
	if err != nil {
		return err
	}
	diaryPresent := map[sim.SavedDiaryOwner]bool{}
	for _, row := range world.SavedDiaries() {
		diaryPresent[row.Owner] = true
	}
	var owners []sim.SavedDiaryOwner
	for _, row := range a.AbsentDiaries {
		if !diaryPresent[row.Owner] {
			continue
		}
		current, ok := diaries[row.Owner]
		if !ok || current.Object != row.Object || current.Inline != row.Inline {
			continue // An ordinary owner/edge edit establishes a real Diary.
		}
		anchor, err := currentDiaryAnchor(current.Record)
		if err != nil {
			return err
		}
		if anchor == row.Anchor {
			owners = append(owners, row.Owner)
		}
	}
	lastBlock := make(map[uint16]sav.BlockRecord, len(doc.World.Blocks))
	for _, block := range doc.World.Blocks {
		lastBlock[block.Cell] = block
	}
	var blocks []uint16
	for _, row := range a.AbsentBlocks {
		if block, present := lastBlock[row.Cell]; present && motionBlockPresent[row.Cell] && currentBlockCarrierAnchor(block) == row.Anchor {
			blocks = append(blocks, row.Cell)
		}
	}
	var motionCells []uint16
	motionAnchors := currentMotionCellAnchors(doc.World.Cells)
	for _, row := range a.AbsentMotionCells {
		if anchor, present := motionAnchors[row.Cell]; present && motionCellPresent[row.Cell] && anchor == row.Anchor {
			motionCells = append(motionCells, row.Cell)
		}
	}
	return world.RestoreCurrentCarrierAbsence(matching(a.AbsentCellTails, true, tailPresent), matching(a.AbsentCellRecords, false, recordPresent), owners, blocks, motionCells)
}
