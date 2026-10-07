package sim

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"testing"
)

func requireOriginalDeadLegacyDigest(t *testing.T, w *World, want uint64) {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(w.OriginalDeadActors()) != 0 {
		t.Fatal("form-69 control contains original dead records")
	}
	if got := fnv1a(strippedWorldOfOriginalDead(form)); got != want {
		t.Fatalf("version69 control digest changed: %016x want %016x", got, want)
	}
}

func deadInput(id EntityID, stage uint8, hp int16) OriginalDeadActor {
	runtime := uint32(30 + id)
	if stage == 5 {
		runtime = 0
	}
	return OriginalDeadActor{ID: id, Source: OriginalDeadSource{Identity: 100 + uint32(id), ArchiveIndex: uint16(id) + 1, MapUnitID: uint16(id) + 20, Class: 1,
		TerrainKey: 0x12345678, OwnerKey: 0x87654321, References: [5]uint32{91, 92, 93, 94, 95}, ContainerPresent: true, ContainerTail: [2]uint32{10000, 0},
		State: DeadActorState{RuntimeID: runtime, Cell: 0x0304, FineX: 128, FineY: 128, Stage: stage, HP: hp}}}
}

func deadWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewStockedWorld(33, Bounds{8, 8}, ModeCanonical, Terrain{}, []Entity{
		{ID: 1, X: 1, Y: 1, MapUnitID: 21, HP: 30, MaxHP: 30, Defence: 20, GoldChance: 100, TreasureMin: 100, TreasureMax: 100},
		{ID: 2, X: 2, Y: 2, MapUnitID: 22, HP: 40, MaxHP: 40, Speed: 256, SkillXP: [6]int32{12, 34, 56, 78, 90, 123}},
		{ID: 3, X: 3, Y: 3, MapUnitID: 23, HP: 40, MaxHP: 40},
	}, nil, Relations{}, nil, []Stock{{ID: 1, Items: []uint16{0x1234}, Equipped: [EquipSlots]uint16{0x2345}}, {ID: 3, Items: []uint16{0x3456}}})
	if err != nil {
		t.Fatal(err)
	}
	w.SetPurse(1, 12345)
	return w
}

func TestOriginalDead1100AtomicIndependentScalarsAndNoReplay(t *testing.T) {
	w := deadWorld(t)
	batch := []OriginalDeadActor{deadInput(1, 3, -40), deadInput(3, 5, -10017)}
	before := w.encode()
	for name, change := range map[string]func(*OriginalDeadActor){
		"signed timer":     func(d *OriginalDeadActor) { d.Source.State.Timer = -128 },
		"early stage":      func(d *OriginalDeadActor) { d.Source.State.Stage = 2 },
		"terminal residue": func(d *OriginalDeadActor) { d.Source.State.HP = -9999 },
		"position":         func(d *OriginalDeadActor) { d.Source.State.Cell = 0x0800 },
		"duplicate key":    func(d *OriginalDeadActor) { d.Source.Identity = 101 },
		"absent ID":        func(d *OriginalDeadActor) { d.ID = 9 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := append([]OriginalDeadActor(nil), batch...)
			change(&bad[1])
			if err := w.ImportOriginalDeadActors(bad); err == nil {
				t.Fatal("accepted invalid batch")
			}
			if !bytes.Equal(before, w.encode()) {
				t.Fatal("partial import changed state")
			}
		})
	}
	if err := w.ImportOriginalDeadActors(batch); err != nil {
		t.Fatal(err)
	}
	if len(w.entities) != 2 || w.entities[0].HP != -40 || w.entities[0].Decay != 3 || w.entities[0].Defence != 20 || w.entities[0].Dwell != 0 || w.entities[0].OffMap || len(w.sacks) != 0 || len(w.carried[0]) != 0 || w.rng.state != 33 {
		t.Fatalf("import replayed death or missed state: %+v", w.entities)
	}
	if w.entities[0].OrdinaryTargetable() || w.entities[0].Dying() {
		t.Fatal("late corpse remains targetable/occupying")
	}
	next, ok := w.NextEntityID()
	if !ok || next != 4 {
		t.Fatalf("terminal ID reused: %d %t", next, ok)
	}
	if got := w.OriginalDeadActors(); got[0].Source.State != batch[0].Source.State || got[0].Current.Stage != 3 || got[1].Current.HP != -10017 {
		t.Fatalf("lost exact tuple: %+v", got)
	}
	// Native round-trip plus continuation must preserve the evolved state,
	// not overwrite it from the immutable source snapshot.
	Step(w, nil)
	if w.entities[0].Decay != 4 {
		t.Fatal("current corpse failed to evolve independently")
	}
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, back.encode()) {
		t.Fatal("dead native roundtrip changed bytes")
	}
	for n := range 64 {
		var commands []Command
		if n == 0 {
			commands = []Command{{Kind: KindMoveTo, Entity: 2, X: 4, Y: 3}}
		}
		Step(w, commands)
		Step(&back, commands)
		if w.Hash() != back.Hash() {
			t.Fatal("dead continuation diverged")
		}
	}
	if w.entities[1].X != 4 || w.entities[1].Y != 3 {
		t.Fatal("late corpse still occupies its cell")
	}
	if w.Purse(1) != 12345 || w.entities[1].SkillXP != [6]int32{12, 34, 56, 78, 90, 123} {
		t.Fatal("import/decay replayed gold or XP")
	}
	if w.OriginalDeadActors()[0].Source.State.HP != w.OriginalDeadActors()[0].Current.HP || w.OriginalDeadActors()[0].Current.HP >= -40 || w.OriginalDeadActors()[1].Current.HP != -10017 || len(w.sacks) != 0 {
		t.Fatal("death mutation left stale health or replayed loot")
	}
}

func TestOriginalDead1100IndependentNativeRecordBytes(t *testing.T) {
	w := deadWorld(t)
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 3, -40)}); err != nil {
		t.Fatal(err)
	}
	// ID; source identity/terrain/owner; five source refs; archive/map/class;
	// container presence/tails; source tuple; current tuple. No encoder helper.
	want, err := hex.DecodeString("01000000" + "650000007856341221436587" +
		"5b0000005c0000005d0000005e0000005f000000" + "020015000101" + "1027000000000000" +
		"1f0000000403808003d8ff00" + "1f0000000403808003d8ff00")
	if err != nil {
		t.Fatal(err)
	}
	want = append(want, make([]byte, 99)...) // absent Weapon flag + 98-byte payload
	want = append(want, 0xad, 0, 0, 0)       // independent 173-byte record span
	form := w.encode()
	end := len(form) - entityIDFloorLen - spellDeliverySpanLen - 65 - 2500 - 8
	if !bytes.Equal(form[end-len(want):end], want) {
		t.Fatalf("native dead record differs: %x", form[end-len(want):end])
	}
	view := w.OriginalDeadActors()
	view[0].Source.References[0] = 0
	if w.OriginalDeadActors()[0].Source.References[0] != 91 {
		t.Fatal("source view aliases canonical record")
	}
}

func TestOriginalDead1100RetirementKeepsTerminalAndReferenceClosure(t *testing.T) {
	w := deadWorld(t)
	w.entities[1].HasAttackTarget = true
	w.entities[1].AttackTarget = 3
	w.entities[1].AttackTargetKind = AttackTargetUnit
	w.entities[1].HasKillCredit = true
	w.entities[1].KillCreditSource = 3
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 4, -600), deadInput(3, 5, -10014)}); err != nil {
		t.Fatal(err)
	}
	for range 64 {
		Step(w, nil)
	}
	if len(w.entities) != 1 || w.entities[0].ID != 2 || w.entities[0].HasAttackTarget || w.entities[0].HasKillCredit {
		t.Fatal("terminal import left dangling actions")
	}
	records := w.OriginalDeadActors()
	if records[0].Current.Stage != 5 || records[0].Current.HP != -10001 || records[0].Source.State != records[0].Current || records[1].Current.HP != -10014 {
		t.Fatalf("wrong retirement/provenance: %+v", records)
	}
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(records, back.OriginalDeadActors()) {
		t.Fatal("terminal tombstones lost")
	}
	for range 64 {
		Step(&back, nil)
	}
	if !reflect.DeepEqual(records, back.OriginalDeadActors()) {
		t.Fatal("terminal tuples decayed")
	}
	if next, _ := back.NextEntityID(); next != 4 {
		t.Fatalf("retired ID reused: %d", next)
	}
}

func TestOriginalDead1100NativeRefusesBrokenBindingAtomically(t *testing.T) {
	w := deadWorld(t)
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 3, -237), deadInput(3, 5, -10011)}); err != nil {
		t.Fatal(err)
	}
	data := w.encode()
	off := len(data) - entityIDFloorLen - spellDeliverySpanLen - 65 - relationLen - 12 - 2*originalDeadRecordLen
	for name, mutate := range map[string]func([]byte){
		"span": func(b []byte) {
			binary.LittleEndian.PutUint32(b[len(b)-entityIDFloorLen-spellDeliverySpanLen-65-relationLen-12:], 0xffffffff)
		},
		"timer":                   func(b []byte) { b[off+61] = 255 },
		"current HP":              func(b []byte) { b[off+71]++ },
		"current stage":           func(b []byte) { b[off+70] = 4 },
		"identity collision":      func(b []byte) { copy(b[off+173+4:off+173+8], b[off+4:off+8]) },
		"terminal becomes entity": func(b []byte) { binary.LittleEndian.PutUint32(b[off+173:], 2) },
	} {
		t.Run(name, func(t *testing.T) {
			bad := append([]byte(nil), data...)
			mutate(bad)
			var back World
			if err := back.UnmarshalBinary(data); err != nil {
				t.Fatal(err)
			}
			before := back.encode()
			if err := back.UnmarshalBinary(bad); err == nil {
				t.Fatal("accepted malformed native dead state")
			}
			if !bytes.Equal(before, back.encode()) {
				t.Fatal("failed decode changed receiver")
			}
		})
	}
}

// An ALM placement may author current health 0 (map 131 places five). ROM1's
// own save then carries that body as a late-dead actor, and the load binds it
// to the fresh map's already-dead body instead of refusing the file.
func TestOriginalDeadBindsAuthoredCorpse(t *testing.T) {
	w, err := NewStockedWorld(33, Bounds{8, 8}, ModeCanonical, Terrain{}, []Entity{
		{ID: 1, X: 1, Y: 1, MapUnitID: 21, HP: 30, MaxHP: 30},
		{ID: 2, X: 2, Y: 2, MapUnitID: 22, HP: 0, MaxHP: 40},
	}, nil, Relations{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.entities[indexOfEntity(w.entities, 2)].Alive() {
		t.Fatal("fixture body is not an authored corpse")
	}
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(2, 5, -10001)}); err != nil {
		t.Fatal("authored corpse refused:", err)
	}
	if got := w.OriginalDeadActors(); len(got) != 1 || got[0].ID != 2 || got[0].Current.Stage != 5 {
		t.Fatalf("authored corpse import = %+v", got)
	}
}
