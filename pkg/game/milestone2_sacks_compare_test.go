package game

import (
	"bytes"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func sack1151Origins(origins []sav.DocumentObjectOrigin) map[uint16]uint16 {
	out := make(map[uint16]uint16, len(origins))
	for _, origin := range origins {
		out[origin.ArchiveIndex] = origin.ObjectIndex
	}
	return out
}

// References may repeat an archive index. Distinct reachable archive objects
// must still have distinct DTO identities, including equal-valued children.
// Check the join itself before using it to translate expected edges or fields.
func sacks1151CheckOrigins(source sackByteSource, origins map[uint16]uint16, bad func(string, ...any)) {
	byLocal := make(map[uint16]uint16, len(source.rows))
	for _, index := range source.indices() {
		local := origins[index]
		if local == 0 {
			bad("archive object %d lacks a DTO origin", index)
			continue
		}
		if previous := byLocal[local]; previous != 0 {
			bad("archive-to-DTO identity collapse: distinct archive objects %d and %d map to DTO %d", previous, index, local)
			continue
		}
		byLocal[local] = index
	}
}

func sacks1151DocumentDifferences(source sackByteSource, origins map[uint16]uint16, doc *sav.DocumentData) []string {
	var differences []string
	bad := func(format string, args ...any) {
		differences = append(differences, fmt.Sprintf("Document: "+format, args...))
	}
	if doc == nil || doc.World == nil {
		return []string{"Document: complete world document absent"}
	}
	sacks1151CheckOrigins(source, origins, bad)
	translate := func(refs []uint16) []uint16 {
		out := make([]uint16, len(refs))
		for i, ref := range refs {
			out[i] = origins[ref]
			if ref != 0 && out[i] == 0 {
				bad("archive object %d lacks a DTO origin", ref)
			}
		}
		return out
	}
	if uint32(len(doc.World.Sacks)) != source.count {
		bad("Sack count %d, raw count %d", len(doc.World.Sacks), source.count)
	}
	if want := translate(source.roots); !slices.Equal(doc.World.Sacks, want) {
		bad("Sack root order/aliases %v, raw %v", doc.World.Sacks, want)
	}
	for _, index := range source.indices() {
		r := source.rows[index]
		local := origins[index]
		if local == 0 || int(local) > len(doc.Objects) {
			bad("%s archive %d omitted", r.class, index)
			continue
		}
		got := &doc.Objects[local-1]
		label := fmt.Sprintf("%s archive %d", r.class, index)
		if got.Class != r.class {
			bad("%s class %q", label, got.Class)
		}
		if len(got.Values) != len(r.values) || len(got.Raw) != len(r.raw) || len(got.Counts) != len(r.counts) || len(got.RefSlots) != len(r.refs) || len(got.Texts)+len(got.Inline)+len(got.Groups) != 0 {
			bad("%s field population differs", label)
		}
		seen := map[string]bool{}
		for _, v := range got.Values {
			want, ok := r.values[v.Name]
			if !ok || seen[v.Name] || v.Value != want {
				bad("%s %s=%#x, raw %#x (present=%v)", label, v.Name, v.Value, want, ok)
			}
			seen[v.Name] = true
		}
		for name := range r.values {
			if !seen[name] {
				bad("%s omitted scalar %s", label, name)
			}
		}
		seen = map[string]bool{}
		for _, v := range got.Raw {
			want, ok := r.raw[v.Name]
			if !ok || seen[v.Name] || !bytes.Equal(v.Bytes, want) {
				bad("%s raw %s differs", label, v.Name)
			}
			seen[v.Name] = true
		}
		for name := range r.raw {
			if !seen[name] {
				bad("%s omitted raw %s", label, name)
			}
		}
		seen = map[string]bool{}
		for _, v := range got.Counts {
			want, ok := r.counts[v.Name]
			if !ok || seen[v.Name] || v.Count != want {
				bad("%s %s count %d, raw %d", label, v.Name, v.Count, want)
			}
			seen[v.Name] = true
		}
		for name := range r.counts {
			if !seen[name] {
				bad("%s omitted count %s", label, name)
			}
		}
		seen = map[string]bool{}
		for _, v := range got.RefSlots {
			refs, ok := r.refs[v.Name]
			want := translate(refs)
			if !ok || seen[v.Name] || !slices.Equal(v.Objects, want) {
				bad("%s %s order/aliases %v, raw %v", label, v.Name, v.Objects, want)
			}
			seen[v.Name] = true
		}
		for name := range r.refs {
			if !seen[name] {
				bad("%s omitted references %s", label, name)
			}
		}
	}
	return differences
}

// Only Definition is outside the SAV: the installed table binds it after
// LOAD. Compare every serialized SourceEquipment member without treating
// this engine's table decode as an independent original-file expectation.
func sack1151SameItem(got, want sim.ItemStack) bool {
	got.SourceEquipment.Definition = want.SourceEquipment.Definition
	return sim.StackStateEqual(got, want)
}

func sacks1151LiveDifferences(source sackByteSource, origins map[uint16]uint16, bindings *SnapshotSAVObjectBindings, registry *sim.SavedObjects, sacks []sim.Sack) (differences, gaps []string) {
	bad := func(format string, args ...any) {
		differences = append(differences, fmt.Sprintf("World: "+format, args...))
	}
	if bindings == nil || registry == nil {
		return []string{"World: Sack/item object registry or bindings absent"}, nil
	}
	sacks1151CheckOrigins(source, origins, bad)
	byObject := map[uint16]sim.SavedObjectID{}
	byID := map[sim.SavedObjectID]uint16{}
	for _, group := range [][]SnapshotSAVObjectBinding{bindings.Sacks, bindings.Items, bindings.Effects, bindings.Spells} {
		for _, b := range group {
			if b.ObjectIndex == 0 {
				continue
			}
			if b.ID == 0 {
				bad("DTO object %d lacks a native identity", b.ObjectIndex)
				continue
			}
			if byObject[b.ObjectIndex] != 0 {
				bad("duplicate object binding %d", b.ObjectIndex)
			}
			if previous := byID[b.ID]; previous != 0 && previous != b.ObjectIndex {
				bad("DTO-to-native identity collapse: distinct DTO objects %d and %d map to native %d", previous, b.ObjectIndex, b.ID)
			}
			byID[b.ID] = b.ObjectIndex
			byObject[b.ObjectIndex] = b.ID
			if b.Unavailable != "" {
				gaps = append(gaps, fmt.Sprintf("DTO object %d: %s", b.ObjectIndex, b.Unavailable))
			}
		}
	}
	ids := func(refs []uint16) []sim.SavedObjectID {
		out := make([]sim.SavedObjectID, len(refs))
		for i, ref := range refs {
			out[i] = byObject[origins[ref]]
			if ref != 0 && out[i] == 0 {
				bad("archive object %d has no native identity", ref)
			}
		}
		return out
	}
	unavailable := map[uint16]string{}
	for _, b := range bindings.Unavailable {
		unavailable[b.ObjectIndex] = b.Reason
	}
	if uint32(len(sacks)) != source.count {
		bad("ground Sack count %d, raw count %d", len(sacks), source.count)
	}
	var roots []sim.SavedObjectID
	adopted, ownedItems := 0, 0
	checkedChildren := map[uint16]bool{}
	for slot, index := range source.roots {
		r := source.rows[index]
		token := sack1151Token(r)
		local, id := origins[index], byObject[origins[index]]
		label := fmt.Sprintf("Sack slot %d archive %d identity %#x", slot, index, token.Identity)
		var native *sim.Sack
		matches := 0
		for i := range sacks {
			if sacks[i].X == int32(token.Position[2]) && sacks[i].Y == int32(token.Position[3]) {
				native = &sacks[i]
				matches++
			}
		}
		if matches != 1 {
			bad("%s has %d native cell matches", label, matches)
			continue
		}
		if native.Gold != r.values["S3C"] || native.ObjectID != id {
			bad("%s ground Gold/ObjectID differs", label)
		}
		if id == 0 {
			if reason := unavailable[local]; reason != "" {
				gaps = append(gaps, fmt.Sprintf("%s: %s; Token/container/item graph unavailable in World, checked in Document", label, reason))
			} else {
				bad("%s unadopted without named coverage", label)
			}
		} else {
			adopted++
			roots = append(roots, id)
			found := 0
			for _, s := range registry.Sacks {
				if s.ID == id {
					found++
					if s.Token != token || s.Gold != r.values["S3C"] || s.Retired || s.Origin.Kind != sim.SavedObjectOriginal {
						bad("%s retained Token/Gold/lifecycle differs", label)
					}
				}
			}
			if found != 1 {
				bad("%s has %d live object records", label, found)
			}
			containers := 0
			for _, c := range registry.Containers {
				if c.Owner != (sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: id}) {
					continue
				}
				containers++
				if !c.Present || c.InsertIndex != r.values["Contents1C"] || c.Accumulator != int32(r.values["Contents20"]) {
					bad("%s container tail/presence differs: %+v", label, c)
				}
				if want := ids(r.refs["Contents"]); !slices.Equal(c.Items, want) {
					bad("%s container item order/aliases %v, raw %v", label, c.Items, want)
				}
			}
			if containers != 1 {
				bad("%s has %d container tails", label, containers)
			}
		}
		var expanded uint64
		for ordinal, ref := range r.refs["Contents"] {
			item := source.rows[ref]
			if item == nil {
				bad("%s item slot %d is null", label, ordinal)
				continue
			}
			itemID := byObject[origins[ref]]
			want := source.item(item, itemID)
			if id != 0 {
				ownedItems++
				got, ok := registry.Item(itemID)
				if !ok {
					bad("%s Item archive %d omitted", label, ref)
				} else {
					if got.Token != sack1151Token(item) || got.F45 != uint8(item.values["F45"]) || got.F46 != uint8(item.values["F46"]) || got.F47 != uint8(item.values["F47"]) || got.F48 != uint16(item.values["F48"]) {
						bad("Item archive %d retained Token/scalars differ", ref)
					}
					if got.Owner != (sim.SavedObjectOwner{}) || !registry.HasLocation(got.ID, sim.SavedItemLocation{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: id}, Index: uint32(ordinal)}) || !sack1151SameItem(got.Value, want) {
						bad("Item archive %d owner/value/count differs", ref)
					}
					if !slices.Equal(got.Effects, ids(item.refs["Effects"])) {
						bad("Item archive %d Effect order/aliases differ", ref)
					}
					spell := ids(item.refs["WeaponSpell"])
					if len(spell) == 1 && got.Spell != spell[0] || len(spell) == 0 && got.Spell != 0 {
						bad("Item archive %d Spell identity differs", ref)
					}
				}
				for _, children := range [][]uint16{item.refs["Effects"], item.refs["WeaponSpell"]} {
					for _, child := range children {
						if child == 0 || checkedChildren[child] {
							continue
						}
						checkedChildren[child] = true
						cr, cid := source.rows[child], byObject[origins[child]]
						found := 0
						if cr.class == "Effect" {
							for _, e := range registry.Effects {
								if e.ID == cid {
									found++
									if e.Token != sack1151Token(cr) || e.Value != sack1151Effect(cr) || e.E0C != uint8(cr.values["E0C"]) || e.Retired {
										bad("Effect archive %d Token/value/state differs", child)
									}
								}
							}
						} else if cr.class == "Spell" {
							for _, s := range registry.Spells {
								if s.ID == cid {
									found++
									if s.Value != sack1151Spell(cr) || s.This != cr.values["This"] || s.Retired {
										bad("Spell archive %d scalar/identity differs", child)
									}
								}
							}
						}
						if found != 1 {
							bad("child archive %d has %d live objects", child, found)
						}
					}
				}
			}
			for range want.Count {
				if expanded >= uint64(len(native.ItemInstances)) {
					bad("%s Item archive %d expanded unit %d omitted", label, ref, expanded)
					break
				}
				got := sim.StackItem(native.ItemInstances[expanded], want.Count)
				if !sack1151SameItem(got, want) {
					bad("%s expanded unit %d Item archive %d identity/value differs", label, expanded, ref)
				}
				expanded++
			}
		}
		if expanded != uint64(len(native.ItemInstances)) || expanded != uint64(len(native.Items)) {
			bad("%s expanded item population %d/%d, raw %d", label, len(native.ItemInstances), len(native.Items), expanded)
		}
		for i, item := range native.ItemInstances {
			if i >= len(native.Items) || native.Items[i] != item.Code {
				bad("%s public item code %d differs", label, i)
			}
		}
	}
	if !slices.Equal(registry.SackRoots, roots) {
		bad("Sack root order/aliases %v, raw adopted %v", registry.SackRoots, roots)
	}
	if len(registry.Sacks) != adopted {
		bad("live Sack object population %d, raw adopted %d", len(registry.Sacks), adopted)
	}
	actualItems := 0
	for _, item := range registry.Items {
		for _, at := range registry.Locations(item.ID) {
			if at.Owner.Kind == sim.SavedOwnerSack {
				actualItems++
			}
		}
	}
	if actualItems != ownedItems {
		bad("Sack-owned Item population %d, raw adopted %d", actualItems, ownedItems)
	}
	return differences, gaps
}
