package game

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

var actorAttachmentCases = []struct {
	spell uint16
	kind  sim.EffectKind
	wire  uint8
}{
	{5, sim.EffectProtectionFire, 21},
	{10, sim.EffectProtectionWater, 22},
	{15, sim.EffectInvisible, 0},
	{16, sim.EffectProtectionAir, 23},
	{18, sim.EffectAbsorption, 16},
	{22, sim.EffectProtectionEarth, 24},
	{23, sim.EffectBless, 0},
	{24, sim.EffectSpeed, 17},
}

func attachedSpellWorld(t *testing.T) (*FrontEnd, *Mission) {
	t.Helper()
	f, ms := actorEffectFixture(t)
	binding := ms.savedDocument.ActorEffects.Rows[0]
	id := binding.Entity
	for _, actor := range ms.savedDocument.Actors {
		if actor.EntityID == id {
			record := &ms.savedDocument.Document.Objects[actor.ObjectIndex-1]
			savedObjectSetRefs(record, "Effects", nil, true)
			savedObjectSetValue(record, "U144", 0)
		}
	}
	doc, _, err := sav.RetireDocumentData(*ms.savedDocument.Document, []uint16{binding.ObjectIndex})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err = loadOriginalMission(f, raw)
	if err != nil || len(ms.World.ActiveEffects()) != 0 {
		t.Fatal("effect-free fixture LOAD", err)
	}
	var effects []sim.ActiveEffect
	for _, row := range actorAttachmentCases {
		effects = append(effects, sim.ActiveEffect{Target: id, Spell: row.spell, Kind: row.kind,
			Mode: sim.EffectDuration, Magnitude: int32(row.spell), Remaining: 128})
	}
	if err := ms.World.ImportOriginalAttachedEffects(effects); err != nil {
		t.Fatal(err)
	}
	return f, ms
}

func TestAttachedSpellFieldsSaveAndReload(t *testing.T) {
	f, ms := attachedSpellWorld(t)
	want := ms.World.ActiveEffects()
	before := ms.World.Hash()
	state, err := snapshotSavedDocument(ms)
	if err != nil || state.ActorEffects.Unavailable != "" {
		t.Fatal("SAVE projection", err, state)
	}
	if ms.World.Hash() != before {
		t.Fatal("SAVE projection changed the live World")
	}
	seen := 0
	for _, binding := range state.ActorEffects.Rows {
		if binding.ObjectIndex == 0 {
			continue
		}
		seen++
		record := &state.Document.Objects[binding.ObjectIndex-1]
		value, err := savedEffectRecord(record)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range actorAttachmentCases {
			if row.spell != binding.Spell {
				continue
			}
			if value.Token.T0E != 2*row.spell+8 || value.Value.Kind != row.wire || value.Token.RuntimeID != 0 {
				t.Errorf("row %d: type/kind/+04 = %d/%d/%d, want %d/%d/0", row.spell,
					value.Token.T0E, value.Value.Kind, value.Token.RuntimeID, 2*row.spell+8, row.wire)
			}
			if value.Token.T0C != uint8(row.spell) || value.E0C != uint8(row.spell) ||
				value.Value.Mode != 1 || value.Value.Operand != uint32(row.spell)|128<<16 {
				t.Errorf("row %d: current operands changed: %+v", row.spell, value)
			}
		}
	}
	if seen != len(actorAttachmentCases) {
		t.Fatalf("saved %d attachments, want %d", seen, len(actorAttachmentCases))
	}
	raw, err := sav.EncodeDocumentData(*state.Document)
	if err != nil {
		t.Fatal(err)
	}
	cold, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal("binary LOAD", err)
	}
	if got := cold.World.ActiveEffects(); !reflect.DeepEqual(got, want) {
		t.Fatalf("binary LOAD effects = %+v, want %+v; metadata %+v", got, want, cold.savedDocument.ActorEffects)
	}
	second, err := snapshotSavedDocument(cold)
	if err != nil || second.ActorEffects.Unavailable != "" {
		t.Fatal("SAVE after binary LOAD", err)
	}
	for _, binding := range second.ActorEffects.Rows {
		value, err := savedStructureValue(&second.Document.Objects[binding.ObjectIndex-1], "T0E")
		if err != nil || value != uint32(2*binding.Spell+8) {
			t.Errorf("second SAVE row %d: type %d: %v", binding.Spell, value, err)
		}
	}
}

func TestOriginalKindZeroAttachmentsLoadBySpell(t *testing.T) {
	for _, row := range actorAttachmentCases {
		if row.spell != 15 && row.spell != 23 {
			continue
		}
		t.Run(fmt.Sprint(row.spell), func(t *testing.T) {
			f, ms := actorEffectFixture(t)
			state := ms.savedDocument
			binding := state.ActorEffects.Rows[0]
			record := literalSavedEffectRecord(0xabcdef)
			for name, value := range map[string]uint32{"RuntimeID": 0x1234, "T0E": uint32(2*row.spell + 8),
				"T0C": uint32(row.spell), "E0C": uint32(row.spell), "E3C": 0, "E3D": 1, "E40": 7 | 128<<16} {
				savedObjectSetValue(&record, name, value)
			}
			state.Document.Objects[binding.ObjectIndex-1] = record
			for _, actor := range state.Actors {
				if actor.EntityID == binding.Entity {
					savedObjectSetValue(&state.Document.Objects[actor.ObjectIndex-1], "U144", uint32(1)<<row.spell)
				}
			}
			raw, err := sav.EncodeDocumentData(*state.Document)
			if err != nil {
				t.Fatal(err)
			}
			cold, _, err := loadOriginalMission(f, raw)
			if err != nil {
				t.Fatal(err)
			}
			want := []sim.ActiveEffect{{Target: binding.Entity, Spell: row.spell, Kind: row.kind,
				Mode: sim.EffectDuration, Magnitude: 7, Remaining: 128}}
			if got := cold.World.ActiveEffects(); !reflect.DeepEqual(got, want) {
				t.Fatalf("original-shaped kind-0 LOAD = %+v, want %+v; metadata %+v", got, want, cold.savedDocument.ActorEffects)
			}
		})
	}
}

func TestBoundAttachmentTypeHealingPreservesNonzero(t *testing.T) {
	for _, retained := range []uint32{0, 1, 0xbeef} {
		t.Run(fmt.Sprint(retained), func(t *testing.T) {
			_, ms := attachedSpellWorld(t)
			state, err := snapshotSavedDocument(ms)
			if err != nil {
				t.Fatal(err)
			}
			for _, binding := range state.ActorEffects.Rows {
				if binding.ObjectIndex != 0 {
					record := &state.Document.Objects[binding.ObjectIndex-1]
					savedObjectSetValue(record, "T0E", retained)
					savedObjectSetValue(record, "RuntimeID", 0x4567)
				}
			}
			sim.Step(ms.World, nil)
			var previous []byte
			for cycle := range 2 {
				if err := projectSavedActorEffects(state, ms.World); err != nil {
					t.Fatal(err)
				}
				for _, binding := range state.ActorEffects.Rows {
					if binding.ObjectIndex == 0 {
						continue
					}
					value, err := savedEffectRecord(&state.Document.Objects[binding.ObjectIndex-1])
					want := retained
					if want == 0 {
						want = uint32(2*binding.Spell + 8)
					}
					if err != nil || uint32(value.Token.T0E) != want || value.Token.RuntimeID != 0x4567 ||
						value.Value.Operand != uint32(binding.Spell)|127<<16 {
						t.Errorf("cycle %d row %d: retained type %d, want %d; %+v: %v", cycle, binding.Spell, retained, want, value, err)
					}
				}
				raw, err := sav.EncodeDocumentData(*state.Document)
				if err != nil || cycle == 1 && !bytes.Equal(raw, previous) {
					t.Fatal("repeated projection changed saved bytes", err)
				}
				previous = raw
			}
		})
	}
}

func TestOtherAttachedSpellRowsKeepTheirType(t *testing.T) {
	for spell := uint16(0); spell < sim.SpellIDLimit; spell++ {
		if spell == 5 || spell == 10 || spell == 15 || spell == 16 || spell == 18 || spell == 22 || spell == 23 || spell == 24 {
			continue
		}
		e := sim.ActiveEffect{Target: 7, Spell: spell, Kind: sim.EffectHealthRegeneration, Mode: sim.EffectDuration, Magnitude: 3, Remaining: 128}
		record, err := newEffectAttachment(e)
		if err != nil {
			t.Fatal(err)
		}
		value, err := savedEffectRecord(&record)
		if err != nil || value.Token.T0E != 0 || value.Value.Kind != 8 || value.Token.RuntimeID != 0 {
			t.Fatalf("other row %d changed: %+v: %v", spell, value, err)
		}
	}
}

func TestSecondGameEffectTypeHolderStillWritesZero(t *testing.T) {
	doc := sav.DocumentData{Players: []uint16{1}, Objects: []sav.DocumentRecordData{
		{Class: "Player", Values: []sav.DocumentValueData{{Name: "Hero", Value: 7}}},
		{Class: "Human", Values: []sav.DocumentValueData{{Name: "Identity", Value: 7}}},
	}}
	for _, row := range actorAttachmentCases {
		e := sim.ActiveEffect{Target: 7, Spell: row.spell, Kind: row.kind, Mode: sim.EffectDuration, Remaining: 128}
		record, err := newEffectAttachment(e)
		if err != nil {
			t.Fatal(err)
		}
		doc.Objects = append(doc.Objects, record)
	}
	for _, holder := range secondFormHolders {
		if holder.class != "Effect" || holder.field != "T0E" {
			continue
		}
		if err := writeSecondTownHolders(&doc, []savedFormHolder{holder}, Snapshot{}, nil); err != nil {
			t.Fatal(err)
		}
		for i := 2; i < len(doc.Objects); i++ {
			if got, err := savedStructureValue(&doc.Objects[i], "T0E"); err != nil || got != 0 {
				t.Fatal("second-game holder did not replace type", got, err)
			}
		}
		return
	}
	t.Fatal("second-game Effect T0E holder missing")
}
