package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// secondSpellServerID and secondSpellUnitID key the fixture's spell
// creature: its Units row and its placement.
const (
	secondSpellServerID = 9001
	secondSpellUnitID   = 9
)

// secondSlotCreatureMission is the guard mission with one creature of
// Player 2 whose Units row holds Spell 1 at Probability 1, its other cells -1.
func secondSlotCreatureMission() secondMissionContent {
	c := secondGuardMission()
	p := make([]int32, 56)
	for i := range p {
		p[i] = -1
	}
	p[29], p[48], p[49], p[55] = 70, 1, 1, secondSpellServerID
	c.creatures = map[int]dbEntry{2: {name: "Spell_Creature", params: p}}
	c.units = append(c.units, alm.Unit{X: 56<<8 | 0x80, Y: 48<<8 | 0x80, ClassID: 9, ClassSubID: 1,
		ServerID: secondSpellServerID, Owner: 2, UnitID: secondSpellUnitID, GroupID: 6})
	return c
}

// secondSpellCreature is the World's one creature of the fixture placement.
func secondSpellCreature(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	var out []sim.Entity
	for _, e := range w.Entities() {
		if e.MapUnitID == secondSpellUnitID {
			out = append(out, e)
		}
	}
	if len(out) != 1 {
		t.Fatalf("%d entities of placement %d, want 1", len(out), secondSpellUnitID)
	}
	return out[0]
}

// secondCommandSlots is the command block window of the three slot ids
// (+0x7c) and thresholds (+0x88), R2-ENGINE-652 and R2-ENGINE-832.
const (
	secondCommandSlotIDs        = 0x7c
	secondCommandSlotThresholds = 0x88
	secondCommandSlotEnd        = secondCommandSlotThresholds + 4*sim.CreatureSpellSlots
)

func commandSlots(block []byte, ids, thresholds int) [sim.CreatureSpellSlots]sim.CreatureSpell {
	var out [sim.CreatureSpellSlots]sim.CreatureSpell
	for i := range out {
		out[i] = sim.CreatureSpell{ID: binary.LittleEndian.Uint32(block[ids+4*i:]),
			Threshold: binary.LittleEndian.Uint32(block[thresholds+4*i:])}
	}
	return out
}

// checkSecondCreatureSlots: the file's spell creature record holds want in
// its command block at +0x7c and +0x88, every other Unit and Human record
// zero there, and every command byte outside the post, the word and that
// window is zero, as the producer wrote it before the window was carried.
// The engine document the file carries holds want in the creature's order
// block at the first game's +0x78 and +0x84.
func checkSecondCreatureSlots(t *testing.T, name string, raw []byte, want [sim.CreatureSpellSlots]sim.CreatureSpell) {
	t.Helper()
	doc := checkSecondMissionFile(t, raw, 10)
	creatures := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" {
			continue
		}
		block := secondRecordRaw(r, "URaw184")
		if len(block) != 184 {
			t.Fatalf("%s: %s record %d holds a %d-byte command block", name, r.Class, i+1, len(block))
		}
		mapID, _ := savedStructureValue(r, "T08")
		expect := [sim.CreatureSpellSlots]sim.CreatureSpell{}
		if r.Class == "Unit" && mapID == secondSpellUnitID {
			expect = want
			creatures++
		}
		if got := commandSlots(block, secondCommandSlotIDs, secondCommandSlotThresholds); got != expect {
			t.Fatalf("%s: %s record %d (placement %d) command slots %v, want %v", name, r.Class, i+1, mapID, got, expect)
		}
		if w := binary.LittleEndian.Uint16(block[0x16:]); w != secondCommandWord {
			t.Fatalf("%s: %s record %d writes command word %d, want %d", name, r.Class, i+1, w, secondCommandWord)
		}
		for at, v := range block {
			kept := at < 2 || at == 0x16 || at == 0x17 || at >= secondCommandSlotIDs && at < secondCommandSlotEnd
			if !kept && v != 0 {
				t.Fatalf("%s: %s record %d command block byte %#x is %#x", name, r.Class, i+1, at, v)
			}
		}
	}
	if creatures != 1 {
		t.Fatalf("%s: %d records of placement %d, want 1", name, creatures, secondSpellUnitID)
	}
	engine, err := sav.EngineDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	engineDoc := engineSaveDocument(t, engine)
	creatures = 0
	for i := range engineDoc.Objects {
		r := &engineDoc.Objects[i]
		if mapID, _ := savedStructureValue(r, "T08"); r.Class != "Unit" || mapID != secondSpellUnitID {
			continue
		}
		creatures++
		order := secondRecordRaw(r, "U158")
		if len(order) != 148 {
			t.Fatalf("%s: the engine record holds a %d-byte order block", name, len(order))
		}
		if got := commandSlots(order, 0x78, 0x84); got != want {
			t.Fatalf("%s: the engine order slots %v, want %v", name, got, want)
		}
	}
	if creatures != 1 {
		t.Fatalf("%s: the engine document holds %d records of placement %d, want 1", name, creatures, secondSpellUnitID)
	}
}

// A second-game creature whose Units row holds Spell 1 at Probability 1
// writes spell 1 and threshold 0x147 in its command block at +0x7c and
// +0x88 (R2-ENGINE-652, R2-ENGINE-832); the engine document keeps them at
// the first game's +0x78 and +0x84. The engine's LOAD and the LOAD of the
// records alone restore them. Records holding zero slots LOAD the
// template's and SAVE them; records holding other slots LOAD those and
// SAVE them. Fails while the command block holds zero there.
func TestSecondMissionSaveWritesCreatureSpellSlots(t *testing.T) {
	c := secondSlotCreatureMission()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	template := [sim.CreatureSpellSlots]sim.CreatureSpell{{ID: 1, Threshold: sim.CreatureSpellScale}}
	if got := secondSpellCreature(t, f.live.world).CreatureSpells; got != template {
		t.Fatalf("the World's creature holds slots %v, want the template's %v", got, template)
	}
	raw := secondPageSave(t, f, app)
	checkSecondCreatureSlots(t, "SAVE", raw, template)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold, coldApp := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if cold.live == nil {
		t.Fatal("the file does not LOAD")
	}
	if got := secondSpellCreature(t, cold.live.world).CreatureSpells; got != template {
		t.Fatalf("LOAD holds slots %v, want %v", got, template)
	}
	checkSecondCreatureSlots(t, "SAVE after LOAD", secondPageSave(t, cold, coldApp), template)

	other := [sim.CreatureSpellSlots]sim.CreatureSpell{{ID: 2, Threshold: 2 * sim.CreatureSpellScale}, {}, {ID: 3, Threshold: 5}}
	for _, tc := range []struct {
		name          string
		planted, want [sim.CreatureSpellSlots]sim.CreatureSpell
	}{
		{"the file's slots", template, template},
		{"zero slots", [sim.CreatureSpellSlots]sim.CreatureSpell{}, template},
		{"other slots", other, other},
	} {
		bareDoc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		bareDoc.State.ValueRecords = slices.DeleteFunc(slices.Clone(bareDoc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
			return strings.EqualFold(r.Path, sav.NativeFormPath)
		})
		planted := 0
		for i := range bareDoc.Objects {
			r := &bareDoc.Objects[i]
			if mapID, _ := savedStructureValue(r, "T08"); r.Class != "Unit" || mapID != secondSpellUnitID {
				continue
			}
			for k := range r.Raw {
				if r.Raw[k].Name != "URaw184" {
					continue
				}
				block := slices.Clone(r.Raw[k].Bytes)
				for s, slot := range tc.planted {
					binary.LittleEndian.PutUint32(block[secondCommandSlotIDs+4*s:], slot.ID)
					binary.LittleEndian.PutUint32(block[secondCommandSlotThresholds+4*s:], slot.Threshold)
				}
				r.Raw[k].Bytes = block
				planted++
			}
		}
		if planted != 1 {
			t.Fatalf("%s: planted %d command blocks, want 1", tc.name, planted)
		}
		bare, err := sav.EncodeDocumentData(bareDoc)
		if err != nil {
			t.Fatal(err)
		}
		bareDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(bareDir, "game0001.sav"), bare, 0o600); err != nil {
			t.Fatal(err)
		}
		loaded, loadedApp := secondMissionLoadOf(t, c, bareDir, "game0001.sav")
		if loaded.live == nil {
			t.Fatalf("%s: the file's records alone do not LOAD", tc.name)
		}
		if got := secondSpellCreature(t, loaded.live.world).CreatureSpells; got != tc.want {
			t.Fatalf("%s: the records' LOAD holds slots %v, want %v", tc.name, got, tc.want)
		}
		checkSecondCreatureSlots(t, tc.name+": SAVE after the records' LOAD", secondPageSave(t, loaded, loadedApp), tc.want)
	}
}
