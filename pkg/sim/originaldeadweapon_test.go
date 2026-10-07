package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func deadWeaponSentinel() OriginalDeadWeapon {
	return OriginalDeadWeapon{
		Present: true, ArchiveIndex: 0x1234, Cell: 0x0201, PackedCell: 0x0403, FineX: 5, FineY: 6, PositionU06: 0x0807,
		TerrainKey: 0x0c0b0a09, RuntimeID: 0x100f0e0d, T0C: 17, T0E: 0x1312, T08: 0x17161514, T18: 0x1918,
		T1C: 0x1d1c1b1a, Identity: 0x21201f1e, Reference: 0x25242322,
		F40: 0x2726, F42: 0x2928, F44: 42, F45: 43, F46: 44, F48: 0x2e2d, F4A: 0x302f, F47: 49,
		W52: [24]byte{50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73},
		W6A: [22]byte{74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95}, W50: 96,
	}
}

func TestOriginalDead1100TerminalWeaponNativeEveryByteAndNoReplay(t *testing.T) {
	w := deadWorld(t)
	a, b := deadInput(1, 5, -10007), deadInput(3, 5, -10001)
	a.Source.HeldWeapon = deadWeaponSentinel()
	b.Source.HeldWeapon = deadWeaponSentinel()
	b.Source.HeldWeapon.ArchiveIndex++
	b.Source.HeldWeapon.Identity++
	b.Source.HeldWeapon.T0E++
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{a, b}); err != nil {
		t.Fatal(err)
	}
	if len(w.entities) != 1 || w.entities[0].ID != 2 || len(w.sacks) != 0 || w.rng.state != 33 || w.Purse(1) != 12345 {
		t.Fatal("terminal Weapon became a gameplay entity/drop/reward")
	}
	form := w.encode()
	start := len(form) - entityIDFloorLen - spellDeliverySpanLen - 65 - 2500 - 12 - 2*173
	// Independent 173-byte stride: unchanged 74-byte prefix, presence 1,
	// archive index 0x1234 and all 96 Token/Item/Weapon member bytes in order.
	want := []byte{1, 0x34, 0x12}
	for v := byte(1); v <= 96; v++ {
		want = append(want, v)
	}
	if !bytes.Equal(form[start+74:start+173], want) {
		t.Fatalf("weapon wire=%x want=%x", form[start+74:start+173], want)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.OriginalDeadActors(), back.OriginalDeadActors()) || !bytes.Equal(form, back.encode()) {
		t.Fatal("native weapon provenance changed")
	}
	// Every payload byte remains canonical, even though none can affect play.
	for at := start + 75; at < start+173; at++ {
		changed := bytes.Clone(form)
		changed[at] ^= 0x80
		var probe World
		if err := probe.UnmarshalBinary(changed); err != nil {
			t.Fatalf("payload +%d: %v", at-start-75, err)
		}
		if probe.Hash() == w.Hash() || !bytes.Equal(changed, probe.encode()) {
			t.Fatalf("payload +%d was discarded", at-start-75)
		}
	}
	records := w.OriginalDeadActors()
	for range 64 {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("terminal Weapon continuation diverged")
		}
	}
	if len(w.entities) != 1 || w.entities[0].ID != 2 || len(w.sacks) != 0 || w.Purse(1) != 12345 || !reflect.DeepEqual(records, w.OriginalDeadActors()) {
		t.Fatal("inert Weapon changed gameplay or immutable provenance")
	}
}

func TestOriginalDead1100WeaponRefusalIsAtomic(t *testing.T) {
	for name, mutate := range map[string]func(*OriginalDeadActor){
		"late corpse":    func(d *OriginalDeadActor) { d.Source.State = deadInput(1, 4, -237).Source.State },
		"Human":          func(d *OriginalDeadActor) { d.Source.Class = 2 },
		"absent residue": func(d *OriginalDeadActor) { d.Source.HeldWeapon.Present = false },
		"zero identity":  func(d *OriginalDeadActor) { d.Source.HeldWeapon.Identity = 0 },
		"actor identity": func(d *OriginalDeadActor) { d.Source.HeldWeapon.Identity = d.Source.Identity },
		"actor archive":  func(d *OriginalDeadActor) { d.Source.HeldWeapon.ArchiveIndex = d.Source.ArchiveIndex },
	} {
		t.Run(name, func(t *testing.T) {
			w := deadWorld(t)
			before := w.encode()
			d := deadInput(1, 5, -10007)
			d.Source.HeldWeapon = deadWeaponSentinel()
			mutate(&d)
			if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err == nil || !bytes.Equal(before, w.encode()) {
				t.Fatal("unsupported Weapon partially imported")
			}
		})
	}
	w := deadWorld(t)
	d := deadInput(1, 5, -10007)
	d.Source.HeldWeapon = deadWeaponSentinel()
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
		t.Fatal(err)
	}
	valid := w.encode()
	at := len(valid) - entityIDFloorLen - spellDeliverySpanLen - 65 - 2500 - 12 - 173
	for _, flag := range []byte{0, 2, 255} {
		bad := bytes.Clone(valid)
		bad[at+74] = flag
		if err := w.UnmarshalBinary(bad); err == nil || !bytes.Equal(valid, w.encode()) {
			t.Fatal("noncanonical Weapon presence changed receiver")
		}
	}
}
