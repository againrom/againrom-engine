package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// importSavedActorMotions runs only on the private original-LOAD candidate.
// Native LOAD restores its own canonical state and never calls this producer.
// SAV-TOKENPOS-074 and SAV-UNITPROG-156 supply literal independent operands;
// SAV-CELLLOAD-110 makes cell-slot identity independent of actor Position.
func importSavedActorMotions(ms *Mission, state *SnapshotSAVDocument) error {
	if state == nil || state.Document == nil {
		return nil
	}
	doc := state.Document
	if ms == nil || ms.World == nil || doc.World == nil {
		return fmt.Errorf("saved actor motion requires a complete world")
	}
	// Resolve only exact, unique actor keys. Zero and unresolved keys remain
	// explicit payload state; a map/unit/archive ordinal is not a replacement.
	byKey := make(map[uint32]sim.EntityID)
	ambiguous := make(map[uint32]bool)
	bind := func(key uint32, id sim.EntityID) {
		if key == 0 {
			return
		}
		if prior, exists := byKey[key]; exists && prior != id {
			ambiguous[key] = true
		}
		byKey[key] = id
	}
	var motions []sim.SavedActorMotion
	deadlines := make(map[sim.EntityID]uint32, len(state.Actors))
	for _, binding := range state.Actors {
		if binding.Retired || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(doc.Objects) {
			return fmt.Errorf("saved actor motion has an invalid original binding")
		}
		record := &doc.Objects[binding.ObjectIndex-1]
		deadline, err := savedStructureValue(record, "U138")
		if err != nil {
			return err
		}
		deadlines[binding.EntityID] = deadline
		motion, err := readSavedActorMotion(record, binding.EntityID)
		if err != nil {
			return err
		}
		key, err := savedStructureValue(record, "Identity")
		if err != nil {
			return err
		}
		bind(key, binding.EntityID)
		motions = append(motions, motion)
	}
	// A key colliding with another archive object cannot resolve as an actor.
	// Unrepresented corpse slots remain literal unresolved keys, not a guessed
	// living-motion binding.
	keyObjects := make(map[uint32]int)
	for i, record := range doc.Objects {
		for _, value := range record.Values {
			if value.Name != "This" && value.Name != "Identity" {
				continue
			}
			if prior, exists := keyObjects[value.Value]; value.Value != 0 && exists && prior != i {
				ambiguous[value.Value] = true
			}
			keyObjects[value.Value] = i
		}
	}
	if _, exists := byKey[doc.World.TerrainIdentity]; exists {
		ambiguous[doc.World.TerrainIdentity] = true
	}
	slot := func(key uint32) sim.SavedActorSlot {
		id, found := byKey[key]
		if found && !ambiguous[key] {
			return sim.SavedActorSlot{Key: key, Entity: id, Bound: true}
		}
		return sim.SavedActorSlot{Key: key}
	}
	lastCell := make(map[uint16]int, len(doc.World.Cells))
	for i, cell := range doc.World.Cells {
		lastCell[cell.Cell] = i
	}
	cells := make([]sim.SavedActorCell, 0, len(lastCell))
	for i, cell := range doc.World.Cells {
		if lastCell[cell.Cell] == i {
			cells = append(cells, sim.SavedActorCell{Cell: cell.Cell, Ground: slot(cell.GroundActor), Air: slot(cell.AirActor), Payload: cell.Payload()})
		}
	}
	lastBlock := make(map[uint16]int, len(doc.World.Blocks))
	for i, block := range doc.World.Blocks {
		lastBlock[block.Cell] = i
	}
	blocks := make([]sim.SavedActorBlock, 0, len(lastBlock))
	for i, block := range doc.World.Blocks {
		if lastBlock[block.Cell] == i {
			blocks = append(blocks, sim.SavedActorBlock{Cell: block.Cell, Dyn: block.Dyn, Static: block.Static})
		}
	}
	if err := ms.World.ImportOriginalActorMotions(motions, cells, blocks); err != nil {
		return err
	}
	return ms.World.ImportOriginalActionClocks(deadlines)
}

func readSavedActorMotion(record *sav.DocumentRecordData, entity sim.EntityID) (sim.SavedActorMotion, error) {
	motion := sim.SavedActorMotion{Entity: entity}
	position, err := savedMotionRaw(record, "Block12", 12)
	if err != nil {
		return motion, err
	}
	mover, err := savedMotionRaw(record, "U154", 180)
	if err != nil {
		return motion, err
	}
	motion.Position = sim.SavedActorPosition{Cell: binary.LittleEndian.Uint16(position), PackedCell: binary.LittleEndian.Uint16(position[2:]),
		FineX: position[4], FineY: position[5], Residue: binary.LittleEndian.Uint16(position[6:]), TerrainKey: binary.LittleEndian.Uint32(position[8:])}
	copy(motion.Mover[:], mover)
	action, err := savedMotionRaw(record, "U54", 4)
	if err != nil {
		return motion, err
	}
	motion.ActorAction = binary.LittleEndian.Uint32(action)
	if motion.StaticRoute, err = savedMotionWords(record, "U15C"); err != nil {
		return motion, err
	}
	motion.DynamicRoute, err = savedMotionWords(record, "U178")
	return motion, err
}

func savedMotionRaw(record *sav.DocumentRecordData, name string, width int) ([]byte, error) {
	var raw []byte
	found := false
	for _, field := range record.Raw {
		if field.Name == name {
			if found || len(field.Bytes) != width {
				return nil, fmt.Errorf("saved actor motion has invalid %s width", name)
			}
			raw, found = field.Bytes, true
		}
	}
	if !found {
		return nil, fmt.Errorf("saved actor motion lacks %s", name)
	}
	return raw, nil
}

func savedMotionWords(record *sav.DocumentRecordData, name string) ([]uint16, error) {
	var count uint32
	found := false
	for _, field := range record.Counts {
		if field.Name == name {
			if found || field.Count > savedGroupFieldListLimit {
				return nil, fmt.Errorf("saved actor motion has invalid %s count", name)
			}
			count, found = field.Count, true
		}
	}
	if !found {
		return nil, fmt.Errorf("saved actor motion lacks %s count", name)
	}
	raw, err := savedMotionRaw(record, name, int(count)*2)
	if err != nil {
		return nil, err
	}
	words := make([]uint16, count)
	for i := range words {
		words[i] = binary.LittleEndian.Uint16(raw[2*i:])
	}
	return words, nil
}

// projectSavedActorMotions projects current canonical motion, never the source
// blob. Its native Issue/Current state keeps unsupported continuation explicit;
// this producer alone does not admit original-format world export.
func projectSavedActorMotions(state *SnapshotSAVDocument, world *sim.World) error {
	motions, cells, blocks, present := world.SavedActorMotions()
	if !present {
		return nil // Historical AGS has no reconstructed spatial authority.
	}
	if state == nil || state.Document == nil || state.Document.World == nil {
		return fmt.Errorf("saved actor motion has no world document")
	}
	next, err := sav.CloneDocumentData(*state.Document)
	if err != nil {
		return err
	}
	bindings := make(map[sim.EntityID]SnapshotSAVActor, len(state.Actors))
	for _, binding := range state.Actors {
		bindings[binding.EntityID] = binding
	}
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	for _, motion := range motions {
		if !motion.Current {
			continue
		}
		binding, found := bindings[motion.Entity]
		if !found || binding.Retired || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(next.Objects) {
			return fmt.Errorf("saved actor motion %d has no current document binding", motion.Entity)
		}
		e, exists := entities[motion.Entity]
		key, keyErr := savedStructureValue(&next.Objects[binding.ObjectIndex-1], "Identity")
		if !exists || keyErr != nil || key != e.SourceBinding.Identity || savedActorClass(e.SourceBinding.Class) != next.Objects[binding.ObjectIndex-1].Class {
			return fmt.Errorf("saved actor motion %d document identity differs from its native binding", motion.Entity)
		}
		record, sites, err := cloneSavedGroupFieldRecord(&next.Objects[binding.ObjectIndex-1])
		if err != nil {
			return err
		}
		position, err := sites.fixedRaw(&record, "Block12", 12)
		if err != nil {
			return err
		}
		p := motion.Position
		binary.LittleEndian.PutUint16(position, p.Cell)
		binary.LittleEndian.PutUint16(position[2:], p.PackedCell)
		position[4], position[5] = p.FineX, p.FineY
		binary.LittleEndian.PutUint16(position[6:], p.Residue)
		binary.LittleEndian.PutUint32(position[8:], p.TerrainKey)
		mover, err := sites.fixedRaw(&record, "U154", 180)
		if err != nil {
			return err
		}
		// These two independently current fields were projected by the actor
		// value owner. Retaining the original crossing must not undo them.
		facing, rotationSpeed := mover[0], mover[10]
		saved := motion.SavedMover()
		copy(mover, saved[:])
		mover[0], mover[10] = facing, rotationSpeed
		action, err := sites.fixedRaw(&record, "U54", 4)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(action, motion.ActorAction)
		if err := sites.wordList(&record, "U15C", motion.StaticRoute); err != nil {
			return err
		}
		if err := sites.wordList(&record, "U178", motion.DynamicRoute); err != nil {
			return err
		}
		next.Objects[binding.ObjectIndex-1] = record
	}
	last := make(map[uint16]int, len(next.World.Cells))
	_, currentPlanes := world.SavedCellPlanes()
	if currentPlanes {
		// Current node membership supersedes archive history. Remove every
		// earlier overlay of a deleted key, or LOAD would resurrect it.
		next.World.Cells = make([]sav.DocumentCellData, 0, len(cells))
		for _, cell := range cells {
			next.World.Cells = append(next.World.Cells, sav.DocumentCellFromPayload(cell.Cell, cell.Payload))
		}
		next.World.Blocks = make([]sav.BlockRecord, 0, len(blocks))
		for _, block := range blocks {
			next.World.Blocks = append(next.World.Blocks, sav.BlockRecord{Cell: block.Cell, Dyn: block.Dyn, Static: block.Static})
		}
	}
	for i, cell := range next.World.Cells {
		last[cell.Cell] = i
	}
	for _, cell := range cells {
		if index, exists := last[cell.Cell]; exists {
			// Structure/effect/Sack owners may have newer fields. Motion owns
			// only actor edges of an existing complete payload.
			next.World.Cells[index].GroundActor = cell.Ground.Key
			next.World.Cells[index].AirActor = cell.Air.Key
		} else {
			last[cell.Cell] = len(next.World.Cells)
			next.World.Cells = append(next.World.Cells, sav.DocumentCellFromPayload(cell.Cell, cell.Payload))
		}
	}
	// Preserve earlier duplicate overlay order; update only each last record.
	lastBlock := make(map[uint16]int, len(next.World.Blocks))
	for i, block := range next.World.Blocks {
		lastBlock[block.Cell] = i
	}
	for _, block := range blocks {
		value := sav.BlockRecord{Cell: block.Cell, Dyn: block.Dyn, Static: block.Static}
		if index, exists := lastBlock[block.Cell]; exists {
			next.World.Blocks[index] = value
		} else {
			lastBlock[block.Cell] = len(next.World.Blocks)
			next.World.Blocks = append(next.World.Blocks, value)
		}
	}
	// Block rows are keyed by cell in ascending order; an appended row joins
	// the World-built plane delta at its cell.
	slices.SortStableFunc(next.World.Blocks, func(a, b sav.BlockRecord) int { return int(a.Cell) - int(b.Cell) })
	checked, err := sav.CloneDocumentData(next)
	if err != nil {
		return err
	}
	*state.Document = checked
	return nil
}

// Native LOAD must not adopt two conflicting current positions or cell edges.
// Explicitly superseded motion remains a coverage gap, not authority to repair
// a native World from its retained original document.
func validateSavedActorMotionsWorld(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.Document == nil {
		return nil
	}
	motions, cells, blocks, present := world.SavedActorMotions()
	if !present {
		return nil
	}
	bindings := make(map[sim.EntityID]SnapshotSAVActor, len(state.Actors))
	for _, binding := range state.Actors {
		bindings[binding.EntityID] = binding
	}
	for _, motion := range motions {
		if !motion.Current {
			continue
		}
		binding, found := bindings[motion.Entity]
		if !found || binding.Retired || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(state.Document.Objects) {
			return fmt.Errorf("saved actor motion %d lacks a document binding", motion.Entity)
		}
		record := &state.Document.Objects[binding.ObjectIndex-1]
		position, err := savedMotionRaw(record, "Block12", 12)
		if err != nil {
			return err
		}
		p := motion.Position
		if binary.LittleEndian.Uint16(position) != p.Cell || binary.LittleEndian.Uint16(position[2:]) != p.PackedCell ||
			position[4] != p.FineX || position[5] != p.FineY || binary.LittleEndian.Uint16(position[6:]) != p.Residue || binary.LittleEndian.Uint32(position[8:]) != p.TerrainKey {
			return fmt.Errorf("saved actor motion %d document Position differs from World", motion.Entity)
		}
		mover, err := savedMotionRaw(record, "U154", 180)
		if err != nil {
			return err
		}
		if !savedMoverMatches(mover, motion) {
			return fmt.Errorf("saved actor motion %d document mover differs from World", motion.Entity)
		}
		action, err := savedMotionRaw(record, "U54", 4)
		if err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(action) != motion.ActorAction {
			return fmt.Errorf("saved actor motion %d document action differs from World", motion.Entity)
		}
		for _, pair := range []struct {
			name  string
			words []uint16
		}{{"U15C", motion.StaticRoute}, {"U178", motion.DynamicRoute}} {
			words, err := savedMotionWords(record, pair.name)
			if err != nil {
				return err
			}
			if !slices.Equal(words, pair.words) {
				return fmt.Errorf("saved actor motion %d document %s differs from World", motion.Entity, pair.name)
			}
		}
	}
	if state.Document.World == nil {
		return fmt.Errorf("saved actor motion has no document world")
	}
	lastCell := make(map[uint16]sav.DocumentCellData)
	for _, cell := range state.Document.World.Cells {
		lastCell[cell.Cell] = cell
	}
	_, currentPlanes := world.SavedCellPlanes()
	if currentPlanes && (len(lastCell) != len(cells) || len(state.Document.World.Cells) != len(cells)) {
		return fmt.Errorf("saved actor document has a different current cell population")
	}
	for _, cell := range cells {
		value, exists := lastCell[cell.Cell]
		if !exists || value.GroundActor != cell.Ground.Key || value.AirActor != cell.Air.Key {
			return fmt.Errorf("saved actor cell %04x document links differ from World", cell.Cell)
		}
		if currentPlanes && value.Payload() != cell.Payload {
			return fmt.Errorf("saved actor cell %04x document payload differs from World", cell.Cell)
		}
	}
	lastBlock := make(map[uint16]sav.BlockRecord)
	for _, block := range state.Document.World.Blocks {
		lastBlock[block.Cell] = block
	}
	if currentPlanes && (len(lastBlock) != len(blocks) || len(state.Document.World.Blocks) != len(blocks)) {
		return fmt.Errorf("saved actor document has a different current block population")
	}
	for _, block := range blocks {
		value, exists := lastBlock[block.Cell]
		if !exists || value.Dyn != block.Dyn || value.Static != block.Static {
			return fmt.Errorf("saved actor block %04x document planes differ from World", block.Cell)
		}
	}
	return nil
}

// savedMoverMatches compares a document mover with the World's, apart from
// facing and rotation speed. Both sides name their claim at +06 first, so a
// document written before that rule still matches.
func savedMoverMatches(mover []byte, motion sim.SavedActorMotion) bool {
	doc := sim.SavedActorMotion{}
	copy(doc.Mover[:], mover)
	got, want := doc.SavedMover(), motion.SavedMover()
	return bytes.Equal(got[1:10], want[1:10]) && bytes.Equal(got[11:], want[11:])
}
