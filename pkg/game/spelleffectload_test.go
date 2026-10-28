package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

// spellEffectRootsBoundNative names, by position in the file's SpellEffect
// list, the roots its current actions bind as native areas or deliveries.
// LOAD moves each such root out of the carried graph into the native driver
// (restoreCurrentAreas, restoreCurrentSpellDeliveries; DIV-1271, DIV-2885),
// and SAVE writes it back from that driver. It reads the rows independently
// of the production binding.
func spellEffectRootsBoundNative(doc *sav.DocumentData) (map[int]bool, error) {
	if doc == nil || doc.World == nil {
		return nil, fmt.Errorf("spell effects: the document has no world")
	}
	actions, err := readCurrentActions(doc)
	if err != nil {
		return nil, err
	}
	native := map[int]bool{}
	if actions == nil {
		return native, nil
	}
	named := map[uint16]bool{}
	for _, row := range actions.NativeAreas {
		named[row.Object] = true
	}
	for _, row := range actions.NativeDeliveries {
		named[row.Object] = true
	}
	for i, index := range doc.World.Effects {
		if named[index] {
			native[i] = true
		}
	}
	return native, nil
}

// spellEffectsWithoutOffsets copies a decoded list with every record's source
// offset cleared, so the lists of two files compare by their fields alone.
func spellEffectsWithoutOffsets(list []sav.SpellEffect) []sav.SpellEffect {
	effect := func(e *sav.Effect) *sav.Effect {
		if e == nil {
			return nil
		}
		out := *e
		out.Off = 0
		return &out
	}
	var one func(sav.SpellEffect) sav.SpellEffect
	nested := func(e *sav.SpellEffect) *sav.SpellEffect {
		if e == nil {
			return nil
		}
		out := one(*e)
		return &out
	}
	one = func(e sav.SpellEffect) sav.SpellEffect {
		e.Off = 0
		e.PE48, e.AE44 = effect(e.PE48), effect(e.AE44)
		e.ST44, e.ST48 = nested(e.ST44), nested(e.ST48)
		return e
	}
	out := make([]sav.SpellEffect, len(list))
	for i, e := range list {
		out[i] = one(e)
	}
	return out
}

func TestSpellEffectRootsBoundNativeFollowTheActionsLeaf(t *testing.T) {
	doc := sav.DocumentData{World: &sav.DocumentWorldData{Effects: []uint16{3, 5, 7}}}
	native, err := spellEffectRootsBoundNative(&doc)
	if err != nil || len(native) != 0 {
		t.Fatalf("a file without current actions: %v, %v; want every root carried", native, err)
	}
	setRows := func(areas []currentNativeArea, deliveries []currentNativeDelivery) {
		raw, err := json.Marshal(currentActionData{Version: 1, NativeAreas: areas, NativeDeliveries: deliveries})
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, raw); err != nil {
			t.Fatal(err)
		}
	}
	setRows([]currentNativeArea{{Object: 5}}, []currentNativeDelivery{{Object: 7}})
	native, err = spellEffectRootsBoundNative(&doc)
	if err != nil || !reflect.DeepEqual(native, map[int]bool{1: true, 2: true}) {
		t.Fatalf("an area row and a delivery row: %v, %v; want roots 1 and 2 native and root 0 carried", native, err)
	}
	setRows([]currentNativeArea{{Object: 9}}, nil)
	native, err = spellEffectRootsBoundNative(&doc)
	if err != nil || len(native) != 0 {
		t.Fatalf("a row naming no root: %v, %v; want every root carried", native, err)
	}
	if _, err := spellEffectRootsBoundNative(&sav.DocumentData{}); err == nil {
		t.Fatal("a document without a world was accepted")
	}
}

func TestSpellEffectsWithoutOffsetsComparesFieldsOnly(t *testing.T) {
	list := func(off int, ae4c uint16) []sav.SpellEffect {
		return []sav.SpellEffect{{Off: off, Class: "SpellTransport",
			ST44: &sav.SpellEffect{Off: off + 0x50, Class: "AreaEffect", AE4C: ae4c, AE44: &sav.Effect{Off: off + 0xa0, Class: "Effect", E40: 9}}}}
	}
	a, b := list(0x100, 4), list(0x900, 4)
	if !reflect.DeepEqual(spellEffectsWithoutOffsets(a), spellEffectsWithoutOffsets(b)) {
		t.Fatal("two lists differing only in offsets compare unequal")
	}
	if a[0].Off != 0x100 || a[0].ST44.Off != 0x150 || a[0].ST44.AE44.Off != 0x1a0 {
		t.Fatal("the helper edited the list it copies")
	}
	if reflect.DeepEqual(spellEffectsWithoutOffsets(a), spellEffectsWithoutOffsets(list(0x100, 5))) {
		t.Fatal("two lists differing in a nested field compare equal")
	}
}
