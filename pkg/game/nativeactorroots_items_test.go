package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type nativeActorItemPolicy struct {
	ID                            sim.SavedObjectID
	Object                        uint16
	Kind                          uint8
	Retired                       bool
	WeightKnown, EquipmentKnown   bool
	WeightAnchor, EquipmentAnchor *[32]byte
	IdentityPresent               *bool
	IdentityAnchor                *uint32
	CountLift                     *json.RawMessage
	EffectsUnsupported            bool
	NativeRecordKnown             bool
	NativeRecordAnchor            *[32]byte
	NativeRecordAnchorVersion     uint8
}

type nativeActorItemModes struct {
	present     bool
	byObject    map[uint16]nativeActorItemPolicy
	playerRoots []uint16
	objectKeys  map[uint16]uint32
}

// Only input identity and presence modes are read here. All compared values
// below come from sackByteSource, the actual holder or its retained carrier.
func nativeActorReadItemModes(doc *sav.DocumentData, f *sav.File) (nativeActorItemModes, error) {
	var out nativeActorItemModes
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		return out, err
	}
	var input struct {
		Version   uint32
		Inventory *struct {
			Version    uint32
			ItemRoots  []json.RawMessage
			BookActors []json.RawMessage
		}
		Ownership []nativeActorItemPolicy
	}
	if err := json.Unmarshal(leaf, &input); err != nil {
		return out, err
	}
	if input.Version != 1 || input.Inventory == nil {
		return out, fmt.Errorf("exact native item identity mode unavailable")
	}
	legacy := input.Inventory.Version == 0 || input.Inventory.Version == 1
	if !legacy && input.Inventory.Version != sim.SavedObjectsVersion {
		return out, fmt.Errorf("exact native item identity mode unavailable")
	}
	if legacy && (len(input.Inventory.ItemRoots) != 0 || len(input.Inventory.BookActors) != 0) {
		return out, fmt.Errorf("legacy native item mode has explicit roots")
	}
	out.present, out.byObject = true, make(map[uint16]nativeActorItemPolicy)
	ids := make(map[sim.SavedObjectID]bool)
	for _, p := range input.Ownership {
		if p.Kind != 1 || p.Object == 0 {
			continue
		}
		if p.Retired || int(p.Object) > len(doc.Objects) || !savedItemClass(doc.Objects[p.Object-1].Class) {
			return out, fmt.Errorf("native item mode has an invalid ordinary subject")
		}
		if _, duplicate := out.byObject[p.Object]; duplicate || p.ID != 0 && ids[p.ID] {
			return out, fmt.Errorf("native item mode has an aliased subject/identity")
		}
		for _, a := range []struct {
			known  bool
			anchor *[32]byte
		}{{p.WeightKnown, p.WeightAnchor}, {p.EquipmentKnown, p.EquipmentAnchor}} {
			if a.anchor != nil && (a.known || *a.anchor == ([32]byte{})) {
				return out, fmt.Errorf("native item mode has a conflicting presence anchor")
			}
		}
		if p.NativeRecordAnchorVersion > 1 || p.NativeRecordAnchorVersion != 0 && p.NativeRecordAnchor == nil {
			return out, fmt.Errorf("native item mode has an invalid record anchor version")
		}
		out.byObject[p.Object] = p
		if p.ID != 0 {
			ids[p.ID] = true
		}
	}
	players, err := nativeExpectedPlayers(f)
	if err != nil {
		return out, err
	}
	out.playerRoots, out.objectKeys = players.roots, map[uint16]uint32{}
	locations, err := f.DocumentObjectLocations()
	if err != nil {
		return out, err
	}
	for _, loc := range locations {
		offset := loc.Off + 29
		switch loc.Class {
		case "Player":
			row := players.players[loc.ArchiveIndex]
			if row == nil {
				return out, fmt.Errorf("native item mode lacks a raw Player key")
			}
			out.objectKeys[loc.ArchiveIndex] = row.values["This"]
			continue
		case "Diary":
			continue
		case "Spell":
			offset = loc.Off + 5
		}
		if offset < 0 || offset > len(f.Body)-4 {
			return out, fmt.Errorf("native item mode raw object key exceeds body")
		}
		out.objectKeys[loc.ArchiveIndex] = binary.LittleEndian.Uint32(f.Body[offset:])
	}
	return out, nil
}

func nativeActorItemRecordAnchor(history sim.NativeItemRecord, version uint8, modes nativeActorItemModes) [32]byte {
	var wire []byte
	if version != 0 {
		wire = []byte{1, 0}
		var subject uint16
		matches := 0
		for object, key := range modes.objectKeys {
			if history.Token.Reference != 0 && key == history.Token.Reference {
				subject, matches = object, matches+1
			}
		}
		var ordinal uint32
		if matches == 1 {
			for i, root := range modes.playerRoots {
				if root == subject {
					if ordinal != 0 {
						ordinal = 0
						break
					}
					ordinal = uint32(i + 1)
				}
			}
		}
		if ordinal != 0 {
			wire[1] = 1
			wire = binary.LittleEndian.AppendUint32(wire, ordinal)
			history.Token.Reference = 0
		}
	}
	wire, _ = binary.Append(wire, binary.LittleEndian, history)
	return sha256.Sum256(wire)
}

// The independent raw item projection supplies only the hashed ordinary
// representation. Matching an anchor admits absence, never a scalar value.
func nativeActorItemAnchor(raw sim.ItemStack, equipment bool) [32]byte {
	b := binary.LittleEndian.AppendUint16(nil, raw.Code)
	if equipment {
		x, _ := json.Marshal(raw.SourceEquipment)
		b = append(b, x...)
	} else {
		b = binary.LittleEndian.AppendUint16(b, uint16(raw.Weight))
	}
	return sha256.Sum256(b)
}

func nativeActorItemDifferences(prefix string, got sim.ItemStack, ref uint16, worn bool, location sim.SavedItemLocation,
	r *sackByteReader, origins map[uint16]uint16, ids map[uint16]sim.SavedObjectID,
	state *SnapshotSAVDocument, world *sim.World, modes nativeActorItemModes) []string {
	var differences []string
	add := func(s string) { differences = append(differences, prefix+": "+s) }
	row := r.source.rows[ref]
	if row == nil {
		return []string{prefix + ": raw item unavailable"}
	}
	if err := got.Instance().ValidateWeight(); err != nil {
		add("invalid actual item presence/residue: " + err.Error())
	}
	object, id := origins[ref], ids[origins[ref]]
	raw := r.source.item(row, id)
	want := raw.Clone()
	if worn {
		// Worn holders represent one instance, not an inventory stack.
		// A different serialized count is not silently discarded.
		if raw.Count != 1 {
			add("raw worn count is not one")
		}
		want.Count = 1
	}
	policy, policyFound := modes.byObject[object]
	weightAbsent, equipmentAbsent := false, false
	if modes.present {
		if !policyFound {
			add("exact native item mode unavailable")
		} else {
			if policy.CountLift != nil {
				add("native wide count requires a separate source comparison")
			}
			weightAbsent = !policy.WeightKnown && policy.WeightAnchor != nil && *policy.WeightAnchor == nativeActorItemAnchor(raw, false)
			equipmentAbsent = !policy.EquipmentKnown && policy.EquipmentAnchor != nil && *policy.EquipmentAnchor == nativeActorItemAnchor(raw, true)
			if weightAbsent {
				want.WeightPresent, want.Weight = false, 0
			}
			if equipmentAbsent {
				want.SourceEquipment = sim.SourceEquipment{}
			} else if policy.EquipmentKnown {
				want.SourceEquipment.EffectsUnsupported = policy.EffectsUnsupported
			}
			if policy.ID != id || got.ObjectID != policy.ID {
				add("native current identity/binding differs")
			}
		}
	}
	if id == 0 && (!modes.present || !policyFound || policy.ID != 0) {
		add("item lost identity")
	}
	if id == 0 {
		var history sim.NativeItemRecord
		history.Class = raw.SourceEquipment.Class
		t := &history.Token
		copy(t.Position[:], row.raw["Block12"])
		t.RuntimeID, t.Identity, t.Reference = row.values["RuntimeID"], row.values["Identity"], row.values["Reference"]
		t.T0C, t.T0E, t.T08, t.T18 = uint8(row.values["T0C"]), uint16(row.values["T0E"]), row.values["T08"], uint16(row.values["T18"])
		history.F45, history.F46, history.F47, history.F48 = uint8(row.values["F45"]), uint8(row.values["F46"]), uint8(row.values["F47"]), uint16(row.values["F48"])
		if policy.NativeRecordKnown || policy.NativeRecordAnchor == nil || *policy.NativeRecordAnchor != nativeActorItemRecordAnchor(history, policy.NativeRecordAnchorVersion, modes) || history.Class != 0 && !policy.WeightKnown && want.WeightPresent {
			want.NativeRecord = &history
		}
	}
	// The installed definition is outside these ordinary bytes. Every stored
	// class, row, own kind, raw block and Spell scalar remains compared.
	actual := got.Clone()
	actual.SourceEquipment.Definition = want.SourceEquipment.Definition
	if !sim.StackStateEqual(actual, want) {
		add("actual holder item values/presence differ")
	}
	bindings := 0
	if state.Objects != nil {
		for _, b := range state.Objects.Items {
			if b.ObjectIndex == object || id != 0 && b.ID == id {
				bindings++
				if b.ObjectIndex != object || b.ID != id {
					add("current item has a conflicting typed binding")
				}
			}
		}
	}
	if id == 0 {
		if bindings != 0 {
			add("native absent identity has a typed binding")
		}
		// The caller also checks the independent raw graph against retained
		// Document token bytes and exact holder ordinals. This is Document
		// fallback, not evidence of a current SavedObjectToken or distinct ID.
		return differences
	}
	registry := world.SavedObjects()
	if bindings != 1 || registry == nil {
		add("exact current item binding/registry unavailable")
		return differences
	}
	item, found := registry.Item(id)
	if !found || item.Retired || item.ID != got.ObjectID || !registry.HasLocation(id, location) {
		add("exact typed current item/holder relation unavailable")
		return differences
	}
	token := item.Token
	identity := row.values["Identity"]
	if modes.present && policy.IdentityPresent != nil && !*policy.IdentityPresent && policy.IdentityAnchor != nil && *policy.IdentityAnchor == identity {
		identity = 0
	}
	if token.Identity != identity || token.RuntimeID != row.values["RuntimeID"] || token.Reference != row.values["Reference"] ||
		!bytes.Equal(token.Position[:], row.raw["Block12"]) || token.T0E != uint16(row.values["T0E"]) ||
		token.T08 != row.values["T08"] || token.T18 != uint16(row.values["T18"]) || token.T1C != row.values["T1C"] ||
		item.F45 != uint8(row.values["F45"]) || item.F46 != uint8(row.values["F46"]) || item.F47 != uint8(row.values["F47"]) || item.F48 != uint16(row.values["F48"]) {
		add("typed current token/item raw fields differ")
	}
	if !equipmentAbsent && token.T0C != uint8(row.values["T0C"]) {
		add("typed current definition row differs")
	}
	childID := func(ref uint16, bindings []SnapshotSAVObjectBinding) sim.SavedObjectID {
		if ref == 0 {
			return 0
		}
		var id sim.SavedObjectID
		n := 0
		for _, b := range bindings {
			if b.ObjectIndex == origins[ref] {
				n++
				id = b.ID
			}
		}
		if n != 1 || id == 0 {
			add("item child exact typed identity binding unavailable")
		}
		return id
	}
	var effects []sim.SavedObjectID
	for _, ref := range row.refs["Effects"] {
		child := childID(ref, state.Objects.Effects)
		effects = append(effects, child)
		matches := 0
		for _, current := range registry.Effects {
			if current.ID == child {
				matches++
				rawChild := r.source.rows[ref]
				if rawChild == nil || current.Retired || current.Value != sack1151Effect(rawChild) || current.E0C != uint8(rawChild.values["E0C"]) {
					add("item typed Effect values/state differ")
				}
			}
		}
		if matches != 1 {
			add("item typed Effect population differs")
		}
	}
	if !slices.Equal(item.Effects, effects) {
		add("item ordered Effect identities/alias count differ")
	}
	var spell sim.SavedObjectID
	if refs := row.refs["WeaponSpell"]; len(refs) == 1 && refs[0] != 0 {
		spell = childID(refs[0], state.Objects.Spells)
		matches := 0
		for _, current := range registry.Spells {
			if current.ID == spell {
				matches++
				if current.Retired || current.Value != sack1151Spell(r.source.rows[refs[0]]) {
					add("item typed Spell scalar values differ")
				}
			}
		}
		if matches != 1 {
			add("item typed Spell population differs")
		}
	}
	if item.Spell != spell {
		add("item owned Spell exact identity differs")
	}
	return differences
}

func nativeActorItemLocationDifferences(expected map[sim.SavedObjectID][]sim.SavedItemLocation, live map[sim.EntityID]bool, world *sim.World) []string {
	var out []string
	registry := world.SavedObjects()
	if registry == nil {
		return out
	}
	for _, item := range registry.Items {
		var got []sim.SavedItemLocation
		for _, loc := range registry.Locations(item.ID) {
			if loc.Owner.Kind == sim.SavedOwnerActorPack || loc.Owner.Kind == sim.SavedOwnerActorWorn {
				if live[loc.Owner.Entity] {
					got = append(got, loc)
				} else if _, exists := world.Entity(loc.Owner.Entity); exists {
					out = append(out, fmt.Sprintf("item%d unexpected live actor ownership edge", item.ID))
				}
			}
		}
		want := slices.Clone(expected[item.ID])
		if len(got) != len(want) {
			out = append(out, fmt.Sprintf("item%d actual actor ownership/alias count %d raw%d", item.ID, len(got), len(want)))
			continue
		}
		for _, loc := range got {
			at := slices.Index(want, loc)
			if at < 0 {
				out = append(out, fmt.Sprintf("item%d actual actor ownership ordinal/slot differs", item.ID))
				break
			}
			want = append(want[:at], want[at+1:]...)
		}
	}
	return out
}
