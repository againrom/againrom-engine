package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

// Independent layout transcription: only the version byte and an empty
// four-byte weight span change. These fixtures contain no explicit weights.
func widenedInstanceWeightPin(old []byte) []byte {
	end := len(old) - relationLen
	out := append([]byte(nil), old[:end]...)
	out = append(out, 0, 0, 0, 0)
	out = append(out, old[end:]...)
	out[0] = 74
	return out
}

func strippedInstanceWeightPin(form []byte) []byte {
	out := strippedActorLoadPin(form)
	if out[0] < 74 {
		return out
	}
	end := len(out) - relationLen - 4
	span := int(binary.LittleEndian.Uint32(out[end:]))
	out = append(out[:end-span], out[end+4:]...)
	out[0] = 73
	return out
}

func TestInstanceWeightTransferLoadAndAllCarriers(t *testing.T) {
	item := ItemInstance{Code: 0x0311, Kind: 1, Price: 71, Weight: 7, WeightPresent: true, Effects: []ItemEffect{{Kind: 15, Operand: 1}}}
	zero := item.Clone()
	zero.Weight = 0
	negative := item.Clone()
	negative.Weight = -9
	legacy := item.Clone()
	legacy.Weight = 0
	legacy.WeightPresent = false
	w := mustStockedWorld(t, 21, []Entity{{ID: 1, X: 4, Y: 4}, {ID: 2, X: 4, Y: 4}}, []Stock{{ID: 1, ItemInstances: []ItemInstance{item, item, zero, negative, legacy}}})
	if e := w.DeclareItemWeights([]ItemWeight{{Code: item.Code, Weight: 30}}); e != nil {
		t.Fatal(e)
	}
	stacks, _ := w.CarriedStacks(1)
	if len(stacks) != 4 || stacks[0].Count != 2 || w.entities[0].Load != 17 {
		t.Fatalf("distinct weights/load %+v %d", stacks, w.entities[0].Load)
	}
	check := func() {
		t.Helper()
		b, e := w.MarshalBinary()
		if e != nil {
			t.Fatal(e)
		}
		var back World
		if e = back.UnmarshalBinary(b); e != nil {
			t.Fatal(e)
		}
		if back.Hash() != w.Hash() {
			t.Fatal("weight hash changed on reload")
		}
		w = &back
	}
	check()
	if e := w.MoveCarried(1, 2, item.Code, 1); e != nil {
		t.Fatal(e)
	}
	if w.entities[0].Load != 14 || w.entities[1].Load != 3 {
		t.Fatal("split weight not applied")
	}
	check()
	Step(w, []Command{{Kind: KindEquip, Entity: 2, X: 0, Y: 3}})
	if w.entities[1].Load != 7 {
		t.Fatal("equipped weight lost")
	}
	check()
	Step(w, []Command{{Kind: KindDropWorn, Entity: 2, X: 4, Y: 4, Spell: 3}})
	if w.entities[1].Load != 0 || !reflect.DeepEqual(w.Sacks()[0].ItemInstances[0], item) {
		t.Fatal("drop weight")
	}
	check()
	if e := w.TakeSack(2, 4, 4); e != nil {
		t.Fatal(e)
	}
	if w.entities[1].Load != 3 {
		t.Fatal("pickup weight")
	}
	Step(w, []Command{{Kind: KindTerminalKill, Entity: 2}})
	if !reflect.DeepEqual(w.Sacks()[0].ItemInstances[0], item) {
		t.Fatal("death weight")
	}
	check()
	// Signed multiplication wraps at int32; halving truncates toward zero.
	w.carried[0] = []ItemStack{StackItem(negative, 3)}
	w.recomputeLoad(0)
	if w.entities[0].Load != -13 {
		t.Fatal("negative halving", w.entities[0].Load)
	}
	w.carried[0] = []ItemStack{StackItem(item, 0x80000000)}
	w.recomputeLoad(0)
	if w.entities[0].Load != -1073741824 {
		t.Fatal("signed product width", w.entities[0].Load)
	}
}

func TestInstanceWeightHashAndMalformedSuffix(t *testing.T) {
	item := ItemInstance{Code: 0x0311, WeightPresent: true, Weight: 0}
	w := mustStockedWorld(t, 31, []Entity{{ID: 1}}, []Stock{{ID: 1, ItemInstances: []ItemInstance{item}}})
	b, e := w.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	plain := mustStockedWorld(t, 31, []Entity{{ID: 1}}, []Stock{{ID: 1, Items: []uint16{item.Code}}})
	if w.Hash() == plain.Hash() {
		t.Fatal("explicit zero not hashed")
	}
	weightEnd := len(b) - entityIDFloorLen - spellDeliverySpanLen - 61 - relationLen - w.actorLoadSectionLen() - 4
	start := weightEnd - 6
	cases := map[string]func([]byte) []byte{
		"misaligned span": func(v []byte) []byte { binary.LittleEndian.PutUint32(v[weightEnd:], 5); return v },
		"huge span":       func(v []byte) []byte { binary.LittleEndian.PutUint32(v[weightEnd:], 0xfffffffc); return v },
		"outside ordinal": func(v []byte) []byte { binary.LittleEndian.PutUint32(v[start:], 0xffffffff); return v },
		"empty slot":      func(v []byte) []byte { binary.LittleEndian.PutUint32(v[start:], 1); return v },
		"duplicate": func(v []byte) []byte {
			r := append([]byte(nil), v[:weightEnd]...)
			r = append(r, v[start:weightEnd]...)
			r = binary.LittleEndian.AppendUint32(r, 12)
			return append(r, v[weightEnd+4:]...)
		},
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			before := plain.Hash()
			if e := plain.UnmarshalBinary(mut(append([]byte(nil), b...))); e == nil {
				t.Fatal("accepted malformed suffix")
			}
			if plain.Hash() != before {
				t.Fatal("partial mutation")
			}
		})
	}
	bad := item
	bad.WeightPresent = false
	bad.Weight = 2
	if plain.ReplaceStock(Stock{ID: 1, ItemInstances: []ItemInstance{bad}}) {
		t.Fatal("accepted absent-weight residue")
	}
}

func TestInstanceWeightReservedScrollReloadAndCancellation(t *testing.T) {
	w := scrollWorld1090(t, 12, 1)
	w.carried[0][0].WeightPresent = true
	w.carried[0][0].Weight = -7
	w.recomputeLoad(0)
	Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
	if len(w.scrollCasts) != 1 || !w.scrollCasts[0].Item.WeightPresent || w.scrollCasts[0].Item.Weight != -7 {
		t.Fatal("reservation lost weight")
	}
	b, e := w.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	var back World
	if e = back.UnmarshalBinary(b); e != nil {
		t.Fatal(e)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("reserved scroll hash")
	}
	Step(&back, []Command{{Kind: KindGroupStance, Entity: 1, X: int32(OrderStandGround)}})
	for n := 0; n < 100 && len(back.scrollCasts) > 0; n++ {
		Step(&back, nil)
	}
	stock, _ := back.CarriedStacks(1)
	if len(stock) != 1 || stock[0].Count != 1 || stock[0].Weight != -7 || !stock[0].WeightPresent || back.entities[0].Load != -3 {
		t.Fatal("cancelled scroll weight", stock)
	}
}
