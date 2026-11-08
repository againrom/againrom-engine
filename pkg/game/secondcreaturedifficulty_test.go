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
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondThresholdRow is the fixture creature's Units row columns that the
// difficulty and the thresholds read.
type secondThresholdRow struct{ health, toHit, defence, withdraw, wimpy int32 }

// secondBatRow is a row as the second game's bats hold: its Withdraw above
// half its health (R2-ASSET-229).
var secondBatRow = secondThresholdRow{health: 24, toHit: 10, defence: 20, withdraw: 15, wimpy: 4}

// secondThresholdMission is the guard mission with one creature of Player 2
// whose Units row holds row's columns, every other cell -1.
func secondThresholdMission(row secondThresholdRow) secondMissionContent {
	c := secondGuardMission()
	p := make([]int32, 56)
	for i := range p {
		p[i] = -1
	}
	p[4], p[14], p[15], p[34], p[35] = row.health, row.toHit, row.defence, row.withdraw, row.wimpy
	p[29], p[55] = 70, secondSpellServerID
	c.creatures = map[int]dbEntry{2: {name: "Bat_Sonic", params: p}}
	c.units = append(c.units, alm.Unit{X: 56<<8 | 0x80, Y: 48<<8 | 0x80, ClassID: 9, ClassSubID: 1,
		ServerID: secondSpellServerID, Owner: 2, UnitID: secondSpellUnitID, GroupID: 6})
	return c
}

// secondThresholdEnter opens the fixture mission at difficulty d.
func secondThresholdEnter(t *testing.T, c secondMissionContent, d mapload.Difficulty) (*FrontEnd, *ui.App) {
	t.Helper()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	f.Difficulty = d
	secondBagEnter(t, f, app, screen)
	return f, app
}

// A creature a second-game map places takes the second game's difficulty
// (R2-ENGINE-1002): word 1 halves its maximum and its health and shifts its
// Wimpy right by one; word 2 changes nothing; word 3 adds 50 to its to-hit
// and defence and half again to its maximum. Withdraw is untouched at every
// value. Fails while the second game runs the first game's two thirds.
func TestSecondCreatureDifficultyAtCreation(t *testing.T) {
	for _, tc := range []struct {
		name        string
		diff        mapload.Difficulty
		row         secondThresholdRow
		hp, wi, hit int32
		def         int32
	}{
		{"word 1", mapload.DifficultyEasy, secondBatRow, 12, 2, 10, 20},
		{"word 1 odd health and wimpy", mapload.DifficultyEasy, secondThresholdRow{health: 25, toHit: 10, defence: 20, withdraw: 15, wimpy: 5}, 12, 2, 10, 20},
		{"word 2", mapload.DifficultyNormal, secondBatRow, 24, 4, 10, 20},
		{"word 3", mapload.DifficultyHard, secondBatRow, 36, 4, 60, 70},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := secondThresholdEnter(t, secondThresholdMission(tc.row), tc.diff)
			e := secondSpellCreature(t, f.live.world)
			if e.HP != tc.hp || e.MaxHP != tc.hp {
				t.Fatalf("health %d/%d, want %d/%d", e.HP, e.MaxHP, tc.hp, tc.hp)
			}
			if e.Withdraw != tc.row.withdraw || e.Wimpy != tc.wi {
				t.Fatalf("Withdraw %d, Wimpy %d, want %d and %d", e.Withdraw, e.Wimpy, tc.row.withdraw, tc.wi)
			}
			if e.ToHit != tc.hit || e.Defence != tc.def {
				t.Fatalf("to-hit %d, defence %d, want %d and %d", e.ToHit, e.Defence, tc.hit, tc.def)
			}
		})
	}
}

// A map whose type-0 dword 0x84 is above 1 sets the session flag that keeps
// every creature at its row (R2-ENGINE-509, R2-ENGINE-1002), at every
// difficulty. Fails while the map's word is not asked.
func TestSecondCreatureDifficultyMapFlagKeepsRow(t *testing.T) {
	c := secondThresholdMission(secondBatRow)
	f, _ := secondThresholdEnter(t, c, mapload.DifficultyNormal)
	m, err := alm.OpenROM2(secondMissionMapOf(128, 128, c))
	if err != nil {
		t.Fatal(err)
	}
	for _, diff := range []mapload.Difficulty{mapload.DifficultyEasy, mapload.DifficultyHard} {
		copied := *m
		copied.Meta.Word70 = 4
		w, err := mapload.FromALMWith(&copied, f.Table, diff)
		if err != nil {
			t.Fatal(err)
		}
		e := secondSpellCreature(t, w)
		if e.MaxHP != secondBatRow.health || e.Wimpy != secondBatRow.wimpy {
			t.Fatalf("difficulty %d: maximum %d, Wimpy %d, want the row's %d and %d", diff, e.MaxHP, e.Wimpy, secondBatRow.health, secondBatRow.wimpy)
		}
	}
}

// secondCommandThresholds reads the second game's command block +0x40 and
// +0x44 of the fixture creature's record in doc.
func secondCommandThresholds(t *testing.T, doc sav.DocumentData, block string) (int32, int32) {
	t.Helper()
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if mapID, _ := savedStructureValue(r, "T08"); r.Class != "Unit" || mapID != secondSpellUnitID {
			continue
		}
		raw := secondRecordRaw(r, block)
		if len(raw) < 0x48 {
			t.Fatalf("the creature's %s holds %d bytes", block, len(raw))
		}
		return int32(binary.LittleEndian.Uint32(raw[0x40:])), int32(binary.LittleEndian.Uint32(raw[0x44:]))
	}
	t.Fatal("the file holds no record of the creature")
	return 0, 0
}

// secondWithoutEngineLeaf is doc without the engine document it carries:
// the file the original would load.
func secondWithoutEngineLeaf(doc sav.DocumentData) sav.DocumentData {
	doc.State.ValueRecords = slices.DeleteFunc(slices.Clone(doc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
		return strings.EqualFold(r.Path, sav.NativeFormPath)
	})
	return doc
}

// secondPlantCommandThresholds writes the two thresholds into the creature's
// block in doc.
func secondPlantCommandThresholds(t *testing.T, doc *sav.DocumentData, block string, withdraw, wimpy int32) {
	t.Helper()
	planted := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if mapID, _ := savedStructureValue(r, "T08"); r.Class != "Unit" || mapID != secondSpellUnitID {
			continue
		}
		for k := range r.Raw {
			if r.Raw[k].Name == block {
				b := slices.Clone(r.Raw[k].Bytes)
				binary.LittleEndian.PutUint32(b[0x40:], uint32(withdraw))
				binary.LittleEndian.PutUint32(b[0x44:], uint32(wimpy))
				r.Raw[k].Bytes = b
				planted++
			}
		}
	}
	if planted != 1 {
		t.Fatalf("planted %d %s blocks, want 1", planted, block)
	}
}

func secondThresholdLoad(t *testing.T, c secondMissionContent, b []byte) sim.Entity {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	f, _ := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if f.live == nil {
		t.Fatal("the file does not LOAD")
	}
	return secondSpellCreature(t, f.live.world)
}

// A second-game SAVE writes the creature's current Withdraw and Wimpy at
// command +0x40 and +0x44 and at the engine block's same offsets, and LOAD
// of every file route takes them from the file (R2-ENGINE-1005): an
// original's file with values holds them, and a file holding 0, as every
// file this engine wrote before, keeps 0: such a creature never withdraws
// (R2-ENGINE-1003, DIV-4699). Fails while SAVE writes 0 and LOAD ignores the
// file.
func TestSecondCreatureThresholdsFollowTheFile(t *testing.T) {
	c := secondThresholdMission(secondBatRow)
	f, app := secondThresholdEnter(t, c, mapload.DifficultyNormal)
	if e := secondSpellCreature(t, f.live.world); e.Withdraw != 15 || e.Wimpy != 4 {
		t.Fatalf("the World holds %d and %d, want 15 and 4", e.Withdraw, e.Wimpy)
	}
	raw := secondPageSave(t, f, app)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if w, wi := secondCommandThresholds(t, doc, "URaw184"); w != 15 || wi != 4 {
		t.Fatalf("the file's command block holds %d and %d, want 15 and 4", w, wi)
	}
	engine, err := sav.EngineDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	engineDoc := engineSaveDocument(t, engine)
	if w, wi := secondCommandThresholds(t, engineDoc, "U158"); w != 15 || wi != 4 {
		t.Fatalf("the engine order block holds %d and %d, want 15 and 4", w, wi)
	}
	if e := secondThresholdLoad(t, c, raw); e.Withdraw != 15 || e.Wimpy != 4 {
		t.Fatalf("LOAD of the file holds %d and %d, want 15 and 4", e.Withdraw, e.Wimpy)
	}

	for _, tc := range []struct {
		name         string
		withdraw, wi int32
	}{{"the original's values", 15, 4}, {"values the row does not hold", 3, 1}, {"zero", 0, 0}} {
		t.Run("bare file "+tc.name, func(t *testing.T) {
			bare := secondWithoutEngineLeaf(doc)
			secondPlantCommandThresholds(t, &bare, "URaw184", tc.withdraw, tc.wi)
			b, err := sav.EncodeDocumentData(bare)
			if err != nil {
				t.Fatal(err)
			}
			if e := secondThresholdLoad(t, c, b); e.Withdraw != tc.withdraw || e.Wimpy != tc.wi {
				t.Fatalf("LOAD holds %d and %d, want %d and %d", e.Withdraw, e.Wimpy, tc.withdraw, tc.wi)
			}
		})
	}

	// A file this engine wrote before the thresholds were written holds 0 in
	// the engine block too: LOAD keeps that 0 (DIV-4699).
	engineDoc = engineSaveDocument(t, engine)
	secondPlantCommandThresholds(t, &engineDoc, "U158", 0, 0)
	encodedEngine, err := sav.EncodeDocumentData(engineDoc)
	if err != nil {
		t.Fatal(err)
	}
	old, err := sav.StoreEngineDocument(doc, encodedEngine)
	if err != nil {
		t.Fatal(err)
	}
	secondPlantCommandThresholds(t, &old, "URaw184", 0, 0)
	b, err := sav.EncodeDocumentData(old)
	if err != nil {
		t.Fatal(err)
	}
	if e := secondThresholdLoad(t, c, b); e.Withdraw != 0 || e.Wimpy != 0 {
		t.Fatalf("LOAD of an old engine file holds %d and %d, want 0 and 0", e.Withdraw, e.Wimpy)
	}
}
