package game

import (
	"encoding/json"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type nativeActorBookMode struct {
	LegacyBook     bool
	LegacyBookWire *[32]byte
	ExtraSpells    uint32
}

type nativeActorNativeModes struct {
	present   bool
	books     map[sim.EntityID]nativeActorBookMode
	inventory *struct {
		Present    bool
		Containers []struct{ Owner sim.SavedObjectOwner }
	}
	casters map[[2]uint32]sim.AttachedEffectCaster
}

// Only current mode, presence and subject identities come from the leaf.
// Item, book and Effect values still come from the independent Body reader.
func nativeActorReadModes(doc *sav.DocumentData) (nativeActorNativeModes, error) {
	var out nativeActorNativeModes
	if doc == nil {
		return out, fmt.Errorf("actor modes have no Document")
	}
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		return out, err
	}
	var input struct {
		Version   uint32
		Values    map[sim.EntityID]nativeActorBookMode
		Inventory *struct {
			Present    bool
			Containers []struct{ Owner sim.SavedObjectOwner }
		}
		Actions struct{ EffectCasters []sim.AttachedEffectCaster }
	}
	if err := json.Unmarshal(leaf, &input); err != nil {
		return out, err
	}
	if input.Version != 1 {
		return out, fmt.Errorf("actor modes have invalid version")
	}
	out.present, out.books, out.inventory = true, input.Values, input.Inventory
	if input.Actions.EffectCasters != nil {
		out.casters = make(map[[2]uint32]sim.AttachedEffectCaster)
		for _, row := range input.Actions.EffectCasters {
			key := [2]uint32{uint32(row.Target), uint32(row.Spell)}
			if _, duplicate := out.casters[key]; duplicate || !row.HasCaster && row.Caster != 0 {
				return out, fmt.Errorf("actor modes have ambiguous caster subject")
			}
			out.casters[key] = row
		}
	}
	return out, nil
}

func nativeActorBookDifferences(prefix string, e sim.Entity, raw sim.Spellbook, mask uint32, native bool, modes nativeActorNativeModes, world *sim.World) []string {
	var differences []string
	add := func(s string) { differences = append(differences, prefix+": "+s) }
	if !native {
		if e.Book != raw || e.KnownSpells != mask {
			add("live spellbook differs")
		}
		return differences
	}
	policy, known := modes.books[e.ID]
	if !known {
		add("native book mode unavailable")
		return differences
	}
	if policy.ExtraSpells != 0 {
		add("native extended book membership remains outside ordinary slot comparison")
		return differences
	}
	matchingLegacy := policy.LegacyBook && policy.LegacyBookWire != nil && sim.BookValueAnchor(raw, mask) == *policy.LegacyBookWire
	if matchingLegacy {
		if e.Book.State != sim.BookLegacy || e.Book.Slots != ([28]sim.BookSpell{}) {
			add("native legacy book mode or cached-slot absence differs")
		}
		if raw.State != sim.BookPresent {
			add("native legacy book ordinary presence differs")
		}
	} else if e.Book.State != raw.State {
		add("native book ordinary presence/mode differs")
	}
	if e.KnownSpells != mask {
		add("live spellbook membership differs")
	}
	if err := e.Book.Validate(e.KnownSpells); err != nil {
		add("invalid current book: " + err.Error())
	}
	// These are actual current instance/table/formula operands. The expected
	// tuple below is still the independently read raw slot, including zeros.
	if matchingLegacy {
		for slot := range raw.Slots {
			if mask&(uint32(1)<<uint(slot+1)) == 0 {
				continue
			}
			n := 0
			for _, rule := range world.Spells() {
				if rule.ID == uint16(slot+1) {
					n++
				}
			}
			if n != 1 {
				add(fmt.Sprintf("native book slot%d current rule population%d", slot+1, n))
			}
		}
	}
	values := sim.BookSlotValues(world.Rules(), e, world.Spells())
	for slot, got := range values {
		present := mask&(uint32(1)<<uint(slot+1)) != 0
		if got.Present != present || present && (got.ID != uint8(slot+1) || got.Range != raw.Slots[slot].Range || got.Defensive != raw.Slots[slot].Defensive || got.ManaCost != raw.Slots[slot].ManaCost) {
			add(fmt.Sprintf("live book slot%d presence/value differs", slot+1))
		}
		if !present && got != (sim.SourceItemSpell{}) {
			add(fmt.Sprintf("unlearned book slot%d has current operands", slot+1))
		}
	}
	return differences
}

func nativeActorContainerDifference(a actor1161Record, e sim.Entity, pack []sim.ItemStack, native bool, modes nativeActorNativeModes, registry *sim.SavedObjects, n int) string {
	if !native {
		if n != 1 {
			return fmt.Sprintf("live container population%d", n)
		}
		return ""
	}
	if !modes.present || modes.inventory == nil {
		return "native container presence policy unavailable"
	}
	if modes.inventory.Present != (registry != nil) {
		return "native object registry presence differs"
	}
	expected := 0
	for _, row := range modes.inventory.Containers {
		if row.Owner.Kind == sim.SavedOwnerActorPack && row.Owner.Entity == e.ID {
			expected++
		}
	}
	if expected > 1 || n != expected {
		return fmt.Sprintf("live container population%d current%d", n, expected)
	}
	if expected == 0 && (e.ActorLoad.Present || len(pack) != 0 || len(a.refs["Inventory"]) != 0 || a.values["HasInventory"] != 1 || a.values["Inventory1C"] != 0 || a.values["Inventory20"] != 0) {
		return "native absent container lacks an independently checked ordinary empty constructor"
	}
	return ""
}

func nativeActorCasterMatches(got sim.ActiveEffect, target sim.EntityID, spell uint32, native bool, modes nativeActorNativeModes) bool {
	if !modes.present || modes.casters == nil {
		return !got.HasCaster && got.Caster == 0
	}
	row, present := modes.casters[[2]uint32{uint32(target), spell}]
	return present && got.Target == target && uint32(got.Spell) == spell && got.HasCaster == row.HasCaster && got.Caster == row.Caster
}
