package sim

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestDeadMaximumHealthBinaryKeepsExplicitZeroAndCurrentValue(t *testing.T) {
	for _, maxHP := range []int32{0, 421, 65536} {
		for _, stage := range []uint8{4, 5} {
			w := deadWorld(t)
			d := deadInput(1, stage, -237)
			if stage == 5 {
				d.Source.State.HP = -10001
			}
			d.Source.MaxHP, d.Source.MaxHPPresent = maxHP, true
			if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
				t.Fatal(err)
			}
			if i := indexOfEntity(w.entities, d.ID); i >= 0 && w.entities[i].MaxHP != maxHP {
				t.Fatal("late-dead import retained constructor maximum health")
			}
			for cycle := range 2 {
				raw, err := w.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				if raw[0] != originalDeadHealthFormVersion {
					t.Fatal("explicit maximum health did not write its continuation")
				}
				var cold World
				if err := cold.UnmarshalBinary(raw); err != nil {
					t.Fatal(err)
				}
				got := cold.OriginalDeadActors()[0].Source
				if !got.MaxHPPresent || got.MaxHP != maxHP || cold.Hash() != w.Hash() || !bytes.Equal(raw, cold.encode()) {
					t.Fatalf("stage %d cycle %d maximum health changed: %+v", stage, cycle, got)
				}
				w = &cold
			}
		}
	}
}

func TestDeadMaximumHealthLegacyFormKeepsAbsentCarrier(t *testing.T) {
	for _, stage := range []uint8{4, 5} {
		w := deadWorld(t)
		d := deadInput(1, stage, -237)
		if stage == 5 {
			d.Source.State.HP = -10001
		}
		if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
			t.Fatal(err)
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if raw[0] == originalDeadHealthFormVersion {
			t.Fatal("absent source maximum health fabricated a continuation")
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		if cold.originalDead[0].Source.MaxHPPresent || cold.originalDead[0].Source.MaxHP != 0 ||
			!bytes.Equal(raw, cold.encode()) || cold.Hash() != w.Hash() {
			t.Fatal("legacy dead form fabricated retained maximum health or changed canonical bytes")
		}
		got := cold.OriginalDeadActors()[0].Source
		if stage == 4 && (!got.MaxHPPresent || got.MaxHP != 30) || stage == 5 && (got.MaxHPPresent || got.MaxHP != 0) {
			t.Fatalf("stage %d projection confused current and absent maximum health: %+v", stage, got)
		}
	}
}

func TestDeadMaximumHealthRetirementCapturesLiveMutation(t *testing.T) {
	w := deadWorld(t)
	d := deadInput(1, 4, -600)
	d.Source.MaxHP, d.Source.MaxHPPresent = 421, true
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
		t.Fatal(err)
	}
	w.entities[0].MaxHP = 65536
	if got := w.OriginalDeadActors()[0].Source; !got.MaxHPPresent || got.MaxHP != 65536 {
		t.Fatal("retained corpse projection ignored current maximum health", got)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("current maximum health failed binary/hash roundtrip", err)
	}
	w.tick = decayPhase
	Step(w, nil)
	if _, ok := w.Entity(d.ID); ok {
		t.Fatal("fixture did not retire late-dead body")
	}
	got := w.OriginalDeadActors()[0]
	if got.Current.Stage != 5 || !got.Source.MaxHPPresent || got.Source.MaxHP != 65536 {
		t.Fatal("compaction lost current maximum health", got)
	}
	if cold := dpReload(t, w); cold.Hash() != w.Hash() || cold.OriginalDeadActors()[0].Source.MaxHP != 65536 {
		t.Fatal("cold terminal continuation lost changed maximum health")
	}
}

func TestDeadMaximumHealthCurrentImportPreservesOrdinaryWidth(t *testing.T) {
	w := deadWorld(t)
	d := deadInput(2, 4, -237)
	d.Source.MaxHP, d.Source.MaxHPPresent = 65536, true
	e := &w.entities[1]
	e.X, e.Y, e.HP, e.MaxHP = 4, 3, -237, 65536
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}, d.ID); err != nil {
		t.Fatal(err)
	}
	if e.MaxHP != 65536 || w.originalDead[0].Source.MaxHP != 65536 {
		t.Fatal("current dead import narrowed ordinary maximum health")
	}
	if cold := dpReload(t, w); cold.Hash() != w.Hash() || cold.OriginalDeadActors()[0].Source.MaxHP != 65536 {
		t.Fatal("current dead width failed binary/hash roundtrip")
	}
}

func TestDeadMaximumHealthCurrentImportUsesOrdinarySource(t *testing.T) {
	w := deadWorld(t)
	d := deadInput(2, 4, -237)
	d.Source.MaxHP, d.Source.MaxHPPresent = 76, true
	e := &w.entities[1]
	e.X, e.Y, e.HP, e.MaxHP = 4, 3, -237, 307
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}, d.ID); err != nil {
		t.Fatal(err)
	}
	if e.MaxHP != 76 || w.originalDead[0].Source.MaxHP != 76 {
		t.Fatal("current dead import kept constructor maximum health over ordinary source")
	}
	if cold := dpReload(t, w); cold.Hash() != w.Hash() || cold.OriginalDeadActors()[0].Source.MaxHP != 76 {
		t.Fatal("current dead ordinary maximum health failed binary/hash roundtrip")
	}
}

func TestCurrentTerminalMaximumHealthKeepsCaptureAndLegacyMetadata(t *testing.T) {
	for _, maxHP := range []int32{0, 613, 65536} {
		w := spWorld(t, 17, nil, spEnt(3, 10, 11))
		e := &w.entities[0]
		e.HP, e.Decay, e.MaxHP = decayGoneHP-1, decayLast, maxHP
		old := CurrentTerminalActor{ID: e.ID, Cell: uint16(e.Y)<<8 | uint16(e.X), HP: e.HP, Stage: uint8(e.Decay)}
		if err := w.ImportCurrentTerminalActors([]CurrentTerminalActor{old}); err != nil {
			t.Fatal(err)
		}
		got := w.CurrentTerminalActors()[0]
		if !got.MaxHPPresent || got.MaxHP != maxHP || !w.hasCurrentTerminalActor(old) {
			t.Fatal("legacy metadata discarded captured current maximum health", got)
		}
		cold := dpReload(t, w)
		if cold.Hash() != w.Hash() || cold.CurrentTerminalActors()[0] != got {
			t.Fatal("terminal maximum health failed binary/hash roundtrip")
		}
		values := map[EntityID]ActorValues{old.ID: {CurrentTerminal: &old}}
		if err := cold.restoreActorValues(values); err != nil {
			t.Fatal("legacy action metadata refused current terminal actor", err)
		}
	}
	old := CurrentTerminalActor{ID: 3, Cell: 0x0b0a, HP: -10001, Stage: 4}
	raw, err := json.Marshal(old)
	if err != nil || bytes.Contains(raw, []byte("MaxHP")) {
		t.Fatal("absent maximum health changed legacy terminal JSON", string(raw), err)
	}
	old.MaxHPPresent = true
	raw, err = json.Marshal(old)
	if err != nil || !bytes.Contains(raw, []byte(`"MaxHPPresent":true`)) {
		t.Fatal("known zero maximum health lost JSON presence", string(raw), err)
	}
}

func TestDeadMaximumHealthRejectsCorruptCarriers(t *testing.T) {
	w := deadWorld(t)
	d := deadInput(1, 4, -237)
	d.Source.MaxHP = 421
	before := w.Hash()
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err == nil || w.Hash() != before {
		t.Fatal("absent original-dead carrier accepted a value or changed World")
	}
	row := CurrentTerminalActor{ID: 8, Cell: 0x0304, HP: -10001, Stage: 4, MaxHP: 613}
	if err := w.ImportCurrentTerminalActors([]CurrentTerminalActor{row}); err == nil || w.Hash() != before {
		t.Fatal("absent current-terminal carrier accepted a value or changed World")
	}
	d.Source.MaxHPPresent = true
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	span := int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	start := len(raw) - 9 - span
	for name, change := range map[string]func([]byte){
		"absent ID":            func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 99) },
		"count":                func(b []byte) { binary.LittleEndian.PutUint32(b[start:], 2) },
		"conflicting live max": func(b []byte) { binary.LittleEndian.PutUint32(b[start+8:], 613) },
	} {
		t.Run(name, func(t *testing.T) {
			bad := bytes.Clone(raw)
			change(bad)
			var cold World
			if err := cold.UnmarshalBinary(bad); err == nil {
				t.Fatal("corrupt maximum health continuation was accepted")
			}
		})
	}
}
