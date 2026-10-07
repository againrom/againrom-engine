package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Nil is the legacy absence of a connected object producer. Archive indices
// locate the retained graph only; ID belongs to the native monotonic allocator.
type SnapshotSAVObjectBindings struct {
	Version     uint32
	Sacks       []SnapshotSAVObjectBinding
	Unavailable []SnapshotSAVObjectCoverage
	Items       []SnapshotSAVObjectBinding
	Effects     []SnapshotSAVObjectBinding
	Spells      []SnapshotSAVObjectBinding
	Containers  []SnapshotSAVContainerBinding
}

type SnapshotSAVObjectBinding struct {
	ID          sim.SavedObjectID
	ObjectIndex uint16
	Unavailable string
}

type SnapshotSAVObjectCoverage struct {
	ObjectIndex uint16
	Reason      string
}

const (
	savedSackItemsUnavailable       = "source Sack item adoption is unavailable"
	savedSackIncomingUnavailable    = "source Sack has archive references outside World.Sacks"
	savedSackCellAmbiguous          = "distinct source Sacks share one native cell"
	savedSackIdentityUnavailable    = "source Sack Identity is zero or ambiguous"
	savedSackCellUnavailable        = "source Sack lacks its exact native cell key"
	savedSackNativeUnavailable      = "source Sack lacks one matching empty native Sack"
	savedSackContentsUnavailable    = "native Sack contains unbound items; source Contents is not current"
	savedSackConstructorUnavailable = "native Sack constructor has unknown Token operands"
)

func savedSackCoverageReason(reason string) bool {
	switch reason {
	case savedSackItemsUnavailable, savedSackIncomingUnavailable, savedSackCellAmbiguous, savedSackIdentityUnavailable, savedSackCellUnavailable, savedSackNativeUnavailable:
		return true
	}
	return false
}

func cloneSavedObjectBindings(src *SnapshotSAVObjectBindings, doc *sav.DocumentData, allowCurrentSackDrift ...bool) (*SnapshotSAVObjectBindings, error) {
	if src == nil {
		return nil, nil
	}
	if src.Version != 1 && src.Version != 2 || doc == nil || doc.World == nil || len(src.Sacks) > sim.MaxSavedObjects || len(src.Unavailable) > len(doc.Objects) {
		return nil, fmt.Errorf("saved SAV object bindings have invalid version or bounds")
	}
	roots := make(map[uint16]bool)
	for _, index := range doc.World.Sacks {
		if index == 0 || int(index) > len(doc.Objects) || doc.Objects[index-1].Class != "Sack" {
			return nil, fmt.Errorf("saved SAV object binding has an invalid Sack root")
		}
		roots[index] = true
	}
	seen := make(map[uint16]bool)
	allowDrift := len(allowCurrentSackDrift) != 0 && allowCurrentSackDrift[0]
	for i, row := range src.Sacks {
		if row.ID == 0 || i > 0 && src.Sacks[i-1].ID >= row.ID || row.Unavailable != "" && row.Unavailable != savedSackContentsUnavailable && !(src.Version == 2 && row.Unavailable == savedSackConstructorUnavailable) || row.ObjectIndex == 0 && row.Unavailable != "" {
			return nil, fmt.Errorf("saved SAV Sack binding has invalid identity or coverage")
		}
		if row.ObjectIndex != 0 {
			if !roots[row.ObjectIndex] || seen[row.ObjectIndex] {
				return nil, fmt.Errorf("saved SAV Sack binding has no unique top-level object")
			}
			seen[row.ObjectIndex] = true
		}
	}
	for i, row := range src.Unavailable {
		if !roots[row.ObjectIndex] || seen[row.ObjectIndex] || i > 0 && src.Unavailable[i-1].ObjectIndex >= row.ObjectIndex || !savedSackCoverageReason(row.Reason) {
			return nil, fmt.Errorf("saved SAV Sack unavailable coverage is not canonical")
		}
		seen[row.ObjectIndex] = true
	}
	var drift []SnapshotSAVObjectCoverage
	if len(seen) != len(roots) {
		if !allowDrift {
			return nil, fmt.Errorf("saved SAV object bindings omit a source Sack")
		}
		for index := range roots {
			if !seen[index] {
				drift = append(drift, SnapshotSAVObjectCoverage{ObjectIndex: index, Reason: savedSackNativeUnavailable})
			}
		}
	}
	out := &SnapshotSAVObjectBindings{Version: src.Version, Sacks: slices.Clone(src.Sacks), Unavailable: slices.Clone(src.Unavailable)}
	out.Unavailable = append(out.Unavailable, drift...)
	slices.SortFunc(out.Unavailable, func(a, b SnapshotSAVObjectCoverage) int { return int(a.ObjectIndex) - int(b.ObjectIndex) })
	if err := cloneSavedItemBindings(src, doc, out); err != nil {
		return nil, err
	}
	return out, nil
}

func savedSackRecord(record *sav.DocumentRecordData) (sim.SavedObjectToken, uint32, sim.SavedObjectContainer, error) {
	var token sim.SavedObjectToken
	container := sim.SavedObjectContainer{Present: true}
	if record.Class != "Sack" {
		return token, 0, container, fmt.Errorf("saved SAV Sack binding names another class")
	}
	position := false
	for _, raw := range record.Raw {
		if raw.Name == "Block12" && len(raw.Bytes) == len(token.Position) {
			copy(token.Position[:], raw.Bytes)
			position = true
		}
	}
	if !position {
		return token, 0, container, fmt.Errorf("saved SAV Sack lacks its complete Position")
	}
	values := make([]uint32, 11)
	for i, name := range []string{"RuntimeID", "T0C", "T0E", "T08", "T18", "T1C", "Identity", "Reference", "S3C", "Contents1C", "Contents20"} {
		value, err := savedStructureValue(record, name)
		if err != nil {
			return token, 0, container, err
		}
		values[i] = value
	}
	token.RuntimeID, token.T0C, token.T0E, token.T08 = values[0], uint8(values[1]), uint16(values[2]), values[3]
	token.T18, token.T1C, token.Identity, token.Reference = uint16(values[4]), values[5], values[6], values[7]
	container.InsertIndex, container.Accumulator = values[9], int32(values[10])
	return token, values[8], container, nil
}

func savedSackEmpty(record *sav.DocumentRecordData) bool {
	count, refs := false, false
	for _, value := range record.Counts {
		if value.Name == "Contents" {
			count = value.Count == 0
		}
	}
	for _, value := range record.RefSlots {
		if value.Name == "Contents" {
			refs = len(value.Objects) == 0
		}
	}
	return count && refs
}

// Only archive RefSlots and actual document roots are edges. Token keys and
// raw/cell operands do not become references merely by equalling an index.
func savedSackExternalIncoming(doc *sav.DocumentData) map[uint16]bool {
	incoming := make(map[uint16]bool)
	for _, roots := range [][]uint16{doc.Players, doc.DeadActors, doc.World.Buildings, doc.World.Effects} {
		for _, index := range roots {
			incoming[index] = true
		}
	}
	var walk func(*sav.DocumentRecordData)
	walk = func(record *sav.DocumentRecordData) {
		for _, refs := range record.RefSlots {
			for _, index := range refs.Objects {
				incoming[index] = true
			}
		}
		for i := range record.Inline {
			walk(&record.Inline[i].Record)
		}
		for i := range record.Groups {
			walk(&record.Groups[i])
		}
	}
	for i := range doc.Objects {
		walk(&doc.Objects[i])
	}
	return incoming
}

func savedSackCellOperands(world *sim.World) map[uint16]uint32 {
	_, cells, _, _ := world.SavedActorMotions()
	keys := make(map[uint16]uint32, len(cells))
	for _, cell := range cells {
		keys[cell.Cell] = binary.LittleEndian.Uint32(cell.Payload[16:])
	}
	return keys
}

// This is original-import only. A native LOAD never calls the allocator.
func importSavedSackObjects(ms *Mission, state *SnapshotSAVDocument) error {
	return attachSavedObjectAuthority(ms, state, sim.SavedObjectOriginal)
}

// Constructors and imports share the same owner/ordinal consistency checks.
// Origin is supplied by the caller, never inferred from a missing source index.
func attachSavedObjectAuthority(ms *Mission, state *SnapshotSAVDocument, origin sim.SavedObjectOriginKind) error {
	if state == nil || state.Document == nil {
		return nil
	}
	if ms == nil || ms.World == nil || state.Objects != nil || ms.World.SavedObjects() != nil {
		return fmt.Errorf("saved SAV Sack import requires an absent object owner")
	}
	next, err := cloneSavedDocument(state)
	if err != nil {
		return err
	}
	registry := &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: 1}
	metadata := &SnapshotSAVObjectBindings{Version: 2}
	unique, cells, identities := make(map[uint16]bool), make(map[uint16]int), make(map[uint32]int)
	for _, index := range next.Document.World.Sacks {
		if unique[index] {
			continue
		}
		unique[index] = true
		token, _, _, err := savedSackRecord(&next.Document.Objects[index-1])
		if err != nil {
			return err
		}
		cells[binary.LittleEndian.Uint16(token.Position[2:])]++
		identities[token.Identity]++
	}
	byObject := make(map[uint16]sim.SavedObjectID)
	var bindings []sim.SavedSackBinding
	native := ms.World.Sacks()
	cellOperands := savedSackCellOperands(ms.World)
	incoming := savedSackExternalIncoming(next.Document)
	for _, index := range next.Document.World.Sacks {
		if !unique[index] {
			continue
		}
		delete(unique, index)
		record := &next.Document.Objects[index-1]
		token, gold, container, err := savedSackRecord(record)
		if err != nil {
			return err
		}
		cell := binary.LittleEndian.Uint16(token.Position[2:])
		x, y := int32(cell&255), int32(cell>>8)
		reason := ""
		switch {
		case !savedSackAdoptable(next.Document, record):
			reason = savedSackItemsUnavailable
		case incoming[index]:
			reason = savedSackIncomingUnavailable
		case cells[cell] != 1:
			reason = savedSackCellAmbiguous
		case token.Identity == 0 || identities[token.Identity] != 1:
			reason = savedSackIdentityUnavailable
		case cellOperands[cell] != token.Identity:
			reason = savedSackCellUnavailable
		default:
			matches := 0
			for _, sack := range native {
				if sack.X == x && sack.Y == y && sack.Gold == gold && sack.ObjectID == 0 {
					matches++
				}
			}
			if matches != 1 {
				reason = savedSackNativeUnavailable
			}
		}
		if reason != "" {
			metadata.Unavailable = append(metadata.Unavailable, SnapshotSAVObjectCoverage{ObjectIndex: index, Reason: reason})
			continue
		}
		id := registry.NextID
		registry.NextID++
		byObject[index] = id
		registry.Sacks = append(registry.Sacks, sim.SavedSackObject{ID: id, Origin: sim.SavedObjectOrigin{Kind: origin}, Token: token, Gold: gold})
		container.Owner = sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: id}
		registry.Containers = append(registry.Containers, container)
		metadata.Sacks = append(metadata.Sacks, SnapshotSAVObjectBinding{ID: id, ObjectIndex: index})
		bindings = append(bindings, sim.SavedSackBinding{ID: id, X: x, Y: y, Gold: gold})
	}
	for _, index := range next.Document.World.Sacks {
		if id := byObject[index]; id != 0 {
			registry.SackRoots = append(registry.SackRoots, id)
		}
	}
	slices.SortFunc(metadata.Unavailable, func(a, b SnapshotSAVObjectCoverage) int { return int(a.ObjectIndex) - int(b.ObjectIndex) })
	next.Objects = metadata
	itemBindings, err := attachSavedItemObjects(next, ms.World, registry, origin)
	if err != nil {
		return err
	}
	nextWorld := *ms.World
	if err := nextWorld.ImportSavedObjects(registry, bindings, itemBindings...); err != nil {
		return err
	}
	if err := validateSavedObjectBindingWorld(next, &nextWorld); err != nil {
		return err
	}
	*ms.World, ms.savedDocument = nextWorld, next
	return nil
}

func savedSackContainer(registry *sim.SavedObjects, id sim.SavedObjectID) (sim.SavedObjectContainer, bool) {
	owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: id}
	for _, container := range registry.Containers {
		if container.Owner == owner {
			return container, true
		}
	}
	return sim.SavedObjectContainer{}, false
}

func savedSackBindingCoverage(row sim.SavedSackObject, container sim.SavedObjectContainer) (string, error) {
	if row.Coverage.Unsupported != "" || container.Coverage.Unsupported != "" || row.Coverage.Unknown & ^sim.SavedUnknownContainerLoad != 0 || container.Coverage.Unknown & ^(sim.SavedUnknownContainerLoad|sim.SavedUnknownMergePolicy) != 0 || len(container.Items) != 0 {
		return "", fmt.Errorf("saved SAV Sack has unsupported object authority")
	}
	if row.Coverage.Unknown != 0 || container.Coverage.Unknown != 0 {
		if row.Coverage.Unknown != sim.SavedUnknownContainerLoad || container.Coverage.Unknown != sim.SavedUnknownContainerLoad|sim.SavedUnknownMergePolicy {
			return "", fmt.Errorf("saved SAV Sack has inconsistent unbound Contents coverage")
		}
		return savedSackContentsUnavailable, nil
	}
	return "", nil
}

func validateSavedObjectBindingWorld(state *SnapshotSAVDocument, world *sim.World) error {
	return validateSavedObjectBindingWorldDecoded(state, world, false)
}

func validateSavedObjectBindingWorldDecoded(state *SnapshotSAVDocument, world *sim.World, decodedLegacy bool) error {
	var metadata *SnapshotSAVObjectBindings
	if state != nil {
		metadata = state.Objects
	}
	registry := world.SavedObjects()
	if metadata == nil && registry == nil {
		return nil
	}
	if metadata == nil || registry == nil {
		return fmt.Errorf("saved SAV object metadata and native registry presence differ")
	}
	if metadata.Version == 2 {
		return validateSavedItemBindingWorldDecoded(state, world, decodedLegacy)
	}
	if _, err := cloneSavedObjectBindings(metadata, state.Document); err != nil {
		return err
	}
	if err := registry.ValidateNoInFlight(); err != nil {
		return err
	}
	if len(registry.Items)+len(registry.Effects)+len(registry.Spells) != 0 || len(registry.Sacks) != len(metadata.Sacks) {
		return fmt.Errorf("saved SAV object registry exceeds the gold-only producer")
	}
	for _, container := range registry.Containers {
		if container.Owner.Kind != sim.SavedOwnerSack {
			return fmt.Errorf("saved SAV object registry has an unpaired non-Sack container")
		}
	}
	byObject := make(map[uint16]sim.SavedObjectID)
	cellOperands := savedSackCellOperands(world)
	for i, binding := range metadata.Sacks {
		row := registry.Sacks[i]
		if row.ID != binding.ID || row.Origin != (sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal}) || row.Retired != (binding.ObjectIndex == 0) {
			return fmt.Errorf("saved SAV Sack object identity or lifecycle differs")
		}
		if row.Retired {
			continue
		}
		record := &state.Document.Objects[binding.ObjectIndex-1]
		token, gold, expected, err := savedSackRecord(record)
		if err != nil {
			return err
		}
		container, present := savedSackContainer(registry, row.ID)
		if !present || !container.Present || !savedSackEmpty(record) || token != row.Token || gold != row.Gold || expected.InsertIndex != container.InsertIndex || expected.Accumulator != container.Accumulator {
			return fmt.Errorf("saved SAV Sack Token, Gold or Contents differs from native owner")
		}
		if token.Identity == 0 || cellOperands[binary.LittleEndian.Uint16(token.Position[2:])] != token.Identity {
			return fmt.Errorf("saved SAV Sack lost its exact current native cell key")
		}
		reason, err := savedSackBindingCoverage(row, container)
		if err != nil || binding.Unavailable != reason {
			return fmt.Errorf("saved SAV Sack Contents coverage differs from native owner: %v", err)
		}
		byObject[binding.ObjectIndex] = row.ID
	}
	var roots []sim.SavedObjectID
	for _, index := range state.Document.World.Sacks {
		if id := byObject[index]; id != 0 {
			roots = append(roots, id)
		}
	}
	if !slices.Equal(roots, registry.SackRoots) {
		return fmt.Errorf("saved SAV Sack root order or aliases differ from native owner")
	}
	return nil
}

// The caller stages the containing document too. Zero is accepted for an
// already retired binding; an uncovered source object may never disappear.
func remapSavedObjectBindings(bindings *SnapshotSAVObjectBindings, permutation []uint16) error {
	if bindings == nil {
		return nil
	}
	if len(permutation) == 0 || permutation[0] != 0 {
		return fmt.Errorf("saved SAV object remap lacks the null slot")
	}
	sacks, unavailable := slices.Clone(bindings.Sacks), slices.Clone(bindings.Unavailable)
	items, effects, spells := slices.Clone(bindings.Items), slices.Clone(bindings.Effects), slices.Clone(bindings.Spells)
	for _, group := range [][]SnapshotSAVObjectBinding{sacks, items, effects, spells} {
		for i := range group {
			index := group[i].ObjectIndex
			if int(index) >= len(permutation) {
				return fmt.Errorf("saved SAV object remap is out of bounds")
			}
			group[i].ObjectIndex = permutation[index]
			if group[i].ObjectIndex == 0 && !savedDetachedCoverage(group[i].Unavailable) {
				group[i].Unavailable = ""
			}
		}
	}
	for i := range unavailable {
		index := unavailable[i].ObjectIndex
		if index == 0 || int(index) >= len(permutation) || permutation[index] == 0 {
			return fmt.Errorf("saved SAV uncovered Sack was implicitly retired")
		}
		unavailable[i].ObjectIndex = permutation[index]
	}
	slices.SortFunc(unavailable, func(a, b SnapshotSAVObjectCoverage) int { return int(a.ObjectIndex) - int(b.ObjectIndex) })
	bindings.Sacks, bindings.Unavailable = sacks, unavailable
	bindings.Items, bindings.Effects, bindings.Spells = items, effects, spells
	return nil
}

func remapSavedSackDocument(state *SnapshotSAVDocument, permutation []uint16) error {
	remap := func(index *uint16) error {
		if int(*index) >= len(permutation) || *index != 0 && permutation[*index] == 0 {
			return fmt.Errorf("saved SAV Sack retirement lost a non-Sack binding")
		}
		*index = permutation[*index]
		return nil
	}
	for i := range state.Actors {
		if err := remap(&state.Actors[i].ObjectIndex); err != nil {
			return err
		}
	}
	if groups := state.GroupBindings; groups != nil {
		for i := range groups.Groups {
			for _, index := range []*uint16{&groups.Groups[i].PlayerObject, &groups.Groups[i].Reference.ObjectIndex, &groups.Groups[i].Owner.ObjectIndex} {
				if err := remap(index); err != nil {
					return err
				}
			}
		}
		for i := range groups.Members {
			if err := remap(&groups.Members[i].ObjectIndex); err != nil {
				return err
			}
		}
		for i := range groups.Players {
			if err := remap(&groups.Players[i].ObjectIndex); err != nil {
				return err
			}
		}
	}
	if err := remapSavedActorEffects(state, permutation); err != nil {
		return err
	}
	if err := remapSavedWorldEffects(state, permutation); err != nil {
		return err
	}
	return remapSavedObjectBindings(state.Objects, permutation)
}

func projectSavedSackObjects(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.Objects == nil {
		if world.SavedObjects() != nil {
			return fmt.Errorf("saved SAV object registry has no paired document")
		}
		return nil
	}
	next, err := cloneSavedDocument(state)
	if err != nil {
		return err
	}
	if next.Objects.Version == 2 {
		if err := projectCurrentItemGraph(next, world); err != nil {
			return err
		}
		*state = *next
		return nil
	}
	registry := world.SavedObjects()
	if registry == nil || len(registry.Sacks) != len(next.Objects.Sacks) {
		return fmt.Errorf("saved SAV Sack projection lost its native registry")
	}
	var retired []uint16
	for i := range next.Objects.Sacks {
		binding := &next.Objects.Sacks[i]
		row := registry.Sacks[i]
		if row.ID != binding.ID {
			return fmt.Errorf("saved SAV Sack projection identity differs")
		}
		if row.Retired {
			if binding.ObjectIndex != 0 {
				retired = append(retired, binding.ObjectIndex)
			}
			continue
		}
		if binding.ObjectIndex == 0 {
			return fmt.Errorf("saved SAV Sack projection would resurrect a retired object")
		}
		record := &next.Document.Objects[binding.ObjectIndex-1]
		container, present := savedSackContainer(registry, row.ID)
		if !present || !savedSackEmpty(record) {
			return fmt.Errorf("saved SAV Sack projection requires exact empty source Contents")
		}
		reason, err := savedSackBindingCoverage(row, container)
		if err != nil {
			return err
		}
		binding.Unavailable = reason
		for _, field := range []struct {
			name  string
			value uint32
		}{
			{"RuntimeID", row.Token.RuntimeID}, {"T0C", uint32(row.Token.T0C)}, {"T0E", uint32(row.Token.T0E)}, {"T08", row.Token.T08}, {"T18", uint32(row.Token.T18)}, {"T1C", row.Token.T1C}, {"Identity", row.Token.Identity}, {"Reference", row.Token.Reference}, {"S3C", row.Gold}, {"Contents1C", container.InsertIndex}, {"Contents20", uint32(container.Accumulator)},
		} {
			if err := savedActorSetValue(record, field.name, field.value); err != nil {
				return err
			}
		}
		for j := range record.Raw {
			if record.Raw[j].Name == "Block12" {
				copy(record.Raw[j].Bytes, row.Token.Position[:])
			}
		}
	}
	if len(retired) != 0 {
		next.Document.World.Sacks = slices.DeleteFunc(next.Document.World.Sacks, func(index uint16) bool { return slices.Contains(retired, index) })
		doc, permutation, err := sav.RetireDocumentData(*next.Document, retired)
		if err != nil {
			return err
		}
		next.Document = &doc
		if err := remapSavedSackDocument(next, permutation); err != nil {
			return err
		}
	}
	if _, err := cloneSavedDocument(next); err != nil {
		return err
	}
	if err := validateSavedObjectBindingWorld(next, world); err != nil {
		return err
	}
	*state = *next
	return nil
}
