package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// projectSavedStructures publishes the represented live roster and its retained
// source fields. SourceKey binds a Token Identity; native handles, authored IDs,
// archive indices and local document indices are separate namespaces.
// SAV-BLDG-037, SAV-TOKEN-034 and SAV-TOKENPOS-074 fix the field layout;
// SAV-CLASSSER-173/176 fix the retained subclass suffixes, not their services.
// This does not invent constructors, destruction transitions or block planes.
func projectSavedStructures(document *sav.DocumentData, world *sim.World) error {
	if document == nil || world == nil {
		return fmt.Errorf("saved SAV structures require a document and world")
	}
	sources, cells, present := world.SavedStructures()
	if !present {
		if len(world.Structures()) == 0 {
			return nil
		}
		next, err := sav.CloneDocumentData(*document)
		if err != nil {
			return err
		}
		projectUnregisteredStructureGeometry(&next, world)
		*document = next
		return nil
	}
	if document.World == nil {
		return fmt.Errorf("saved SAV structures require a world document")
	}
	live := world.Structures()
	if len(live) != len(sources) {
		return fmt.Errorf("saved SAV structures have different live/source rosters")
	}
	// The whole-document clone validates shape, reachability and field widths,
	// and owns every slice before the first mutation. Late failures publish none.
	next, err := sav.CloneDocumentData(*document)
	if err != nil {
		return fmt.Errorf("saved SAV structures: %w", err)
	}
	rootObjects := make(map[uint16]bool, len(next.World.Buildings))
	byKey := make(map[uint32]uint16, len(next.World.Buildings))
	for _, index := range next.World.Buildings {
		if index == 0 || int(index) > len(next.Objects) {
			return fmt.Errorf("saved SAV structures have invalid root %d", index)
		}
		if rootObjects[index] {
			continue // A repeated root is one object, not a second structure.
		}
		rootObjects[index] = true
		record := &next.Objects[index-1]
		key, err := savedStructureValue(record, "Identity")
		if err != nil || key == 0 || byKey[key] != 0 {
			return fmt.Errorf("saved SAV structures have ambiguous root identity %#x", key)
		}
		byKey[key] = index
	}
	if len(rootObjects) != len(sources) {
		return fmt.Errorf("saved SAV structures have different document/source rosters")
	}
	// The original shared key map includes non-structure objects. A valid local
	// object index cannot make a colliding numeric Token key unambiguous.
	for i, record := range next.Objects {
		for _, value := range record.Values {
			if value.Name != "Identity" && value.Name != "This" {
				continue
			}
			if index := byKey[value.Value]; index != 0 && int(index) != i+1 {
				return fmt.Errorf("saved SAV structures have cross-object identity %#x", value.Value)
			}
		}
	}
	seenKeys := make(map[uint32]bool, len(sources))
	byID := make(map[sim.StructureID]uint32, len(sources))
	for i, source := range sources {
		structure := live[i]
		if source.ID != structure.ID || (i > 0 && structure.ID <= live[i-1].ID) || source.SourceKey == 0 || seenKeys[source.SourceKey] {
			return fmt.Errorf("saved SAV structures have invalid current binding at %d", i)
		}
		seenKeys[source.SourceKey] = true
		byID[structure.ID] = source.SourceKey
		index := byKey[source.SourceKey]
		if index == 0 || savedStructureClass(source.Class) != next.Objects[index-1].Class {
			return fmt.Errorf("saved SAV structure %d has no matching document key/class", source.ID)
		}
		if err := projectSavedStructureRecord(&next.Objects[index-1], source, structure); err != nil {
			return fmt.Errorf("saved SAV structure %d: %w", source.ID, err)
		}
	}
	// SAV-CELLLOAD-109/110/111: the complete payload is last-write-wins.
	// Update only the represented fields of the final record. Earlier duplicate
	// records and the final actor/Sack/layer/trigger/residue fields stay intact.
	// Adding/removing a record would require their other current-state owners.
	last := make(map[uint16]int, len(next.World.Cells))
	for i, cell := range next.World.Cells {
		last[cell.Cell] = i
	}
	currentCells := make(map[uint16]bool, len(cells))
	for i, cell := range cells {
		currentCells[cell.Cell] = true
		index, exists := last[cell.Cell]
		if !exists || (i > 0 && cell.Cell <= cells[i-1].Cell) {
			return fmt.Errorf("saved SAV structures have invalid current cell %04x", cell.Cell)
		}
		var key uint32
		if cell.HasStructure {
			key = byID[cell.ID]
			if key == 0 {
				return fmt.Errorf("saved SAV cell %04x has no current structure %d", cell.Cell, cell.ID)
			}
		} else if cell.ID != 0 {
			return fmt.Errorf("saved SAV cell %04x has a non-null cleared binding", cell.Cell)
		}
		next.World.Cells[index].Building = key
		next.World.Cells[index].Cost = cell.BaselineCost
		next.World.Cells[index].Static = cell.BaselineStatic
	}
	for key, index := range last {
		if !currentCells[key] && next.World.Cells[index].Building != 0 {
			return fmt.Errorf("saved SAV cell %04x has an unbound structure", key)
		}
	}
	// A retained variable-size suffix is current state too. Its aggregate count
	// and byte bounds may differ from the imported document's; validate the
	// completed candidate before publishing, not only its input shape.
	checked, err := sav.CloneDocumentData(next)
	if err != nil {
		return fmt.Errorf("saved SAV structures produced an invalid document: %w", err)
	}
	*document = checked
	return nil
}

func savedStructureClass(class sim.SavedStructureClass) string {
	switch class.BaseClass() {
	case sim.SavedBuilding:
		return "Building"
	case sim.SavedOutpost:
		return "Outpost"
	case sim.SavedTavern:
		return "Tavern"
	case sim.SavedShop:
		return "Shop"
	default:
		return ""
	}
}

func projectSavedStructureRecord(record *sav.DocumentRecordData, source sim.SavedStructure, live sim.Structure) error {
	if live.Col < 0 || live.Col > 255 || live.Row < 0 || live.Row > 255 {
		return fmt.Errorf("current position exceeds a source cell byte")
	}
	if (source.Class.BaseClass() != sim.SavedOutpost && (source.OutpostWords != [4]uint32{} || len(source.OutpostRecords) != 0)) ||
		(source.Class.BaseClass() != sim.SavedTavern && source.Tavern9C != 0) || (source.Class.BaseClass() != sim.SavedShop && source.Shop70 != 0) {
		return fmt.Errorf("inactive subclass state")
	}
	for _, field := range []sav.DocumentValueData{
		{Name: "RuntimeID", Value: source.RuntimeID}, {Name: "T08", Value: source.AuthoredID},
		{Name: "T0C", Value: uint32(source.Token0C)}, {Name: "T0E", Value: uint32(source.Token0E)},
		{Name: "T18", Value: uint32(source.Token18)}, {Name: "T1C", Value: source.Token1C},
		{Name: "Reference", Value: source.Reference}, {Name: "B40", Value: uint32(source.Kind)},
		{Name: "B42", Value: uint32(live.Field42)}, {Name: "B44", Value: uint32(live.MaxHealth)},
		{Name: "B46", Value: uint32(source.Field46)}, {Name: "B48", Value: uint32(source.Field48)},
		{Name: "B60", Value: uint32(live.Width)}, {Name: "B61", Value: uint32(live.Height)},
		{Name: "B64", Value: live.Blocking}, {Name: "B68", Value: live.Attach},
	} {
		if err := savedStructureSetValue(record, field.Name, field.Value); err != nil {
			return err
		}
	}
	position := source.Position
	if position[0] != byte(live.Col) || position[1] != byte(live.Row) {
		position[0], position[1] = byte(live.Col), byte(live.Row)
		binary.LittleEndian.PutUint16(position[2:], uint16(live.Row<<8|live.Col))
	}
	base := source.Base52
	// LOAD's later scalar stores overlap the raw +52 image. Do not emit a
	// stale second geometry/blocking value after an ordinary current SAVE.
	base[14], base[15] = live.Width, live.Height
	binary.LittleEndian.PutUint32(base[18:], live.Blocking)
	if err := savedStructureSetRaw(record, "Block12", position[:]); err != nil {
		return err
	}
	if err := savedStructureSetRaw(record, "B52", base[:]); err != nil {
		return err
	}
	switch source.Class.BaseClass() {
	case sim.SavedOutpost:
		if len(source.OutpostRecords) > 1<<20 {
			return fmt.Errorf("Outpost record count exceeds document bound")
		}
		for i, name := range [...]string{"O84", "O88", "O80", "O8C"} {
			if err := savedStructureSetValue(record, name, source.OutpostWords[i]); err != nil {
				return err
			}
		}
		raw := make([]byte, 0, 8*len(source.OutpostRecords))
		for _, item := range source.OutpostRecords {
			raw = append(raw, item[:]...)
		}
		if err := savedStructureSetRaw(record, "O6C", raw); err != nil {
			return err
		}
		for i := range record.Counts {
			if record.Counts[i].Name == "O6C" {
				record.Counts[i].Count = uint32(len(source.OutpostRecords))
				return nil
			}
		}
		return fmt.Errorf("missing Outpost count")
	case sim.SavedTavern:
		return savedStructureSetValue(record, "T9C", source.Tavern9C)
	case sim.SavedShop:
		return savedStructureSetValue(record, "S70", source.Shop70)
	}
	return nil
}

func savedStructureValue(record *sav.DocumentRecordData, name string) (uint32, error) {
	for _, field := range record.Values {
		if field.Name == name {
			return field.Value, nil
		}
	}
	return 0, fmt.Errorf("missing structure value %s", name)
}

func savedStructureSetValue(record *sav.DocumentRecordData, name string, value uint32) error {
	for i := range record.Values {
		if record.Values[i].Name == name {
			record.Values[i].Value = value
			return nil
		}
	}
	return fmt.Errorf("missing structure value %s", name)
}

func savedStructureSetRaw(record *sav.DocumentRecordData, name string, raw []byte) error {
	for i := range record.Raw {
		if record.Raw[i].Name == name {
			record.Raw[i].Bytes = slices.Clone(raw)
			return nil
		}
	}
	return fmt.Errorf("missing structure block %s", name)
}

// Unregistered structures join only a unique root at the same kind and cell.
func projectUnregisteredStructureGeometry(document *sav.DocumentData, world *sim.World) {
	if document.World == nil {
		return
	}
	type position struct{ col, row byte }
	roots := make(map[position][]*sav.DocumentRecordData)
	seen := make(map[uint16]bool, len(document.World.Buildings))
	for _, index := range document.World.Buildings {
		if index == 0 || int(index) > len(document.Objects) || seen[index] {
			continue
		}
		seen[index] = true
		record := &document.Objects[index-1]
		for _, raw := range record.Raw {
			if raw.Name == "Block12" && len(raw.Bytes) == 12 {
				at := position{raw.Bytes[0], raw.Bytes[1]}
				roots[at] = append(roots[at], record)
			}
		}
	}
	for _, structure := range world.Structures() {
		if structure.Col < 0 || structure.Col > 255 || structure.Row < 0 || structure.Row > 255 {
			continue
		}
		candidates := roots[position{byte(structure.Col), byte(structure.Row)}]
		var match *sav.DocumentRecordData
		for _, record := range candidates {
			if kind, err := savedStructureValue(record, "B40"); err != nil || kind != uint32(uint8(structure.Kind)) {
				continue
			}
			if match != nil {
				match = nil
				break
			}
			match = record
		}
		if match == nil {
			continue
		}
		for _, value := range []sav.DocumentValueData{
			{Name: "B42", Value: uint32(structure.Field42)}, {Name: "B44", Value: uint32(structure.MaxHealth)},
			{Name: "B60", Value: uint32(structure.Width)}, {Name: "B61", Value: uint32(structure.Height)},
			{Name: "B64", Value: structure.Blocking}, {Name: "B68", Value: structure.Attach},
		} {
			_ = savedStructureSetValue(match, value.Name, value.Value)
		}
		for _, raw := range match.Raw {
			if raw.Name == "B52" && len(raw.Bytes) == 22 {
				raw.Bytes[14], raw.Bytes[15] = structure.Width, structure.Height
				binary.LittleEndian.PutUint32(raw.Bytes[18:], structure.Blocking)
			}
		}
	}
}
