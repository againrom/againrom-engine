package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// carrierMatrixDigest is the first eight bytes of the SHA-256 of what a cell
// of the carrier matrix observed, in hex.
func carrierMatrixDigest(v any) string {
	raw, ok := v.([]byte)
	if !ok {
		var err error
		if raw, err = json.Marshal(v); err != nil {
			return "unmarshalable"
		}
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:8])
}

// carrierMatrixMission is a mission front's observed carrier state: its World
// hash, its terminal actors, dead bodies and Diaries, and the application
// state the mission settled with.
func carrierMatrixMission(f *FrontEnd) string {
	w := f.live.world
	return fmt.Sprintf("world %016x terminal %s dead %s diaries %s application %s", w.Hash(),
		carrierMatrixDigest(w.CurrentTerminalActors()), carrierMatrixDigest(w.OriginalDeadActors()),
		carrierMatrixDigest(w.SavedDiaries()), carrierMatrixDigest(f.live.applicationState))
}

// carrierMatrixTown is a town front's carried Diary and purse.
func carrierMatrixTown(f *FrontEnd) string {
	return fmt.Sprintf("knowledge %s gold %d", carrierMatrixDigest(f.Town.knowledge), f.Town.gold)
}

// carrierMatrixKnowledge is a carried Player Diary for the town rows.
var carrierMatrixKnowledge = []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 119,
	Entries: []sim.SavedDiaryEntry{{Index: 64, Count: 2, Remaining: 1022}, {Index: 72, Count: 4, Remaining: 1020}}}}

// carrierMatrix is, for each fixture document of both games, the bytes each
// SAVE wrote, the state each LOAD seated and the LOAD's result, recorded on
// the engine before the left carriers became rows of the set. The rows of a
// planted loaded record were recorded before the builder's and the mission
// writers' reads of the loaded record and the dead body art became steps.
var carrierMatrix = map[string][]string{
	"first fresh mission": {
		"world 14200df18eb10499 terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application 23637c8096e06655",
		"save 11420eb5eecb4236",
		"world 14200df18eb10499 terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application f45e954ac32aea4d",
		"resave 893d7b2f934ed981",
	},
	"first original dead bodies": {
		"world eabd7d6524b01f92 terminal 74234e98afe7498f dead 8f9b00fc133c53f3 diaries 123a99ed4b7c36fa application c033e1dd42529178",
		"save 0193ee9955c25673",
		"world eabd7d6524b01f92 terminal 74234e98afe7498f dead 8f9b00fc133c53f3 diaries 123a99ed4b7c36fa application c033e1dd42529178",
		"resave 0193ee9955c25673",
	},
	"first terminal actor": {
		"world c0aba9a5301f4b76 terminal b7758f72b5181245 dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application f45e954ac32aea4d",
		"save e22781a41dcff548",
		"world c0aba9a5301f4b76 terminal b7758f72b5181245 dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application f45e954ac32aea4d",
		"resave e22781a41dcff548",
	},
	"first original bound corpse": {
		"world 56624fa128a8cad3 terminal 74234e98afe7498f dead 4513390ecf50bdb0 diaries 123a99ed4b7c36fa application c033e1dd42529178",
		"save f43c0f8225d3b94a",
		"world 56624fa128a8cad3 terminal 74234e98afe7498f dead 4513390ecf50bdb0 diaries 123a99ed4b7c36fa application c033e1dd42529178",
		"resave f43c0f8225d3b94a",
	},
	"first town Diary": {
		"knowledge 74234e98afe7498f gold 123",
		"save 4998f369561d606b",
		"knowledge 378644cc182d8ae5 gold 123",
		"resave 4998f369561d606b",
	},
	"second original mission": {
		"world 30a4c9b222372ade terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application d480409aa297d471",
		"save edc1dd2e693679c0",
		"world 30a4c9b222372ade terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application d480409aa297d471",
		"resave edc1dd2e693679c0",
	},
	"second town Diary": {
		"knowledge 378644cc182d8ae5 gold 500",
		"save 64167438e24952d9",
		"knowledge 378644cc182d8ae5 gold 500",
		"resave 64167438e24952d9",
	},
	"first loaded Player name": {
		"planted 6dcab427302e0a73",
		"world 14200df18eb10499 terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application f45e954ac32aea4d",
		`records Player "Matrix Player 0" 9267d3dbed802941; Unit 51000040 U40 0 U5C 0 T0E 0; Unit 51000050 U40 0 U5C 0 T0E 0; Player "Matrix Player 3" 9267d3dbed802941; Human 51000060 U40 0 U5C 0 T0E 21`,
		"resave a2ca01a9e4a7f4ba",
	},
	"first loaded unknown spans": {
		"planted 12301530d8a4eab8",
		"world 14200df18eb10499 terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application f45e954ac32aea4d",
		`records Player "" 97469a6c6a5b9c15; Unit 51000040 U40 0 U5C 0 T0E 0; Unit 51000050 U40 0 U5C 0 T0E 0; Player "Danath" 97469a6c6a5b9c15; Human 51000060 U40 0 U5C 0 T0E 21`,
		"resave 4f5bfedbc3160bb1",
	},
	"first loaded order words": {
		"planted 099fbf0dfb811e14",
		"world eac5f67528b547b2 terminal 74234e98afe7498f dead 8f9b00fc133c53f3 diaries 123a99ed4b7c36fa application c033e1dd42529178",
		`records Player "P" 66687aadf862bd77; Unit 3ec U40 3ee U5C 3ee T0E 0; Human 51000050 U40 3ee U5C 3ee T0E 21; Player "P" 66687aadf862bd77; Player "" 9267d3dbed802941; Unit 3ee U40 0 U5C 0 T0E 0; Unit 3ef U40 0 U5C 0 T0E 0`,
		"resave 099fbf0dfb811e14",
	},
	"first loaded world effects": {
		"world 03436ff699555a94 terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 123a99ed4b7c36fa application c033e1dd42529178",
		"save a46684fcfee2e518",
		"world 03436ff699555a94 terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 123a99ed4b7c36fa application c033e1dd42529178",
		"effects 1",
		"resave a46684fcfee2e518",
	},
	"first virtual dead body art": {
		"save ab9678fc610ff516",
		`records Player "P" 66687aadf862bd77; Unit 3ec U40 0 U5C 0 T0E 0; Human 51000050 U40 0 U5C 0 T0E 21; Player "P" 66687aadf862bd77; Player "" 9267d3dbed802941; Unit 3ee U40 0 U5C 0 T0E 0; Unit 3ef U40 0 U5C 0 T0E 0; Unit 3f0 U40 0 U5C 0 T0E 2a`,
		"world cea50fde4963af2b terminal 74234e98afe7498f dead 6322fae3d5eaf33c diaries 123a99ed4b7c36fa application c033e1dd42529178",
		"art map[4:42]",
		"resave ab9678fc610ff516",
	},
	// The second game's LOAD of its own SAVE with planted Player names and
	// tails writes the hero's name and no tail again.
	"second loaded Player name and unknown spans": {
		"planted f16e236d9bce1ba6",
		"world 30a4c9b222372ade terminal 74234e98afe7498f dead 4f53cda18c2baa0c diaries 4f53cda18c2baa0c application d480409aa297d471",
		`records Player "Gildarius" ; Human ff00001 U40 0 U5C 0 T0E 0`,
		"resave edc1dd2e693679c0",
	},
}

// carrierMatrixRows runs each fixture document: a fresh start or an original
// file, its SAVE, the LOAD of that SAVE and the SAVE again.
func carrierMatrixRows(t *testing.T) map[string][]string {
	t.Helper()
	rows := map[string][]string{}
	run := func(name string, fn func(t *testing.T) []string) {
		t.Run(name, func(t *testing.T) { rows[name] = fn(t) })
	}
	run("first fresh mission", func(t *testing.T) []string {
		f, s := currentCarrierFixture(t)
		first, err := f.ExportCurrentSave(s, "carrier matrix")
		if err != nil {
			t.Fatal(err)
		}
		cold := openCurrentCarrierSave(t, first)
		return []string{carrierMatrixMission(f), "save " + carrierMatrixDigest(first), carrierMatrixMission(cold), "resave " + carrierMatrixDigest(currentRuntimeSave(t, cold))}
	})
	run("first original dead bodies", func(t *testing.T) []string {
		zero := uint32(0)
		living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 197, name: "Living control"}
		late := &poolFixtureActor{mapID: 92, cell: 0x0807, hp: uint16(65536 - 237), maxHP: 421, stage: 4}
		terminal := &poolFixtureActor{mapID: 93, cell: 0x0a09, hp: uint16(65536 - 10017), maxHP: 613, stage: 5, runtime: &zero}
		fresh := func() *FrontEnd { return currentPoolFixtureFront(t, 91, 92, 93) }
		f := fresh()
		raw := completeCurrentDeadDocument(t, f, savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, []*poolFixtureActor{late, terminal})))
		f = openCurrentRetainedRuntime(t, f, raw)
		first := currentRuntimeSave(t, f)
		cold := openCurrentRetainedRuntime(t, fresh(), first)
		return []string{carrierMatrixMission(f), "save " + carrierMatrixDigest(first), carrierMatrixMission(cold), "resave " + carrierMatrixDigest(currentRuntimeSave(t, cold))}
	})
	run("first terminal actor", func(t *testing.T) []string {
		f, _ := bindingOrderFront(t, 2)
		first := currentRuntimeSave(t, f)
		cold := cellStateFront(t)
		cold.Campaign, cold.Table = f.Campaign, f.Table
		open, town, err := cold.RestoreOriginal(first)
		if err == nil && !town {
			err = cold.App("carrier matrix").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("terminal actor LOAD", town, err)
		}
		if len(cold.live.world.CurrentTerminalActors()) == 0 {
			t.Fatal("the fixture's LOAD holds no terminal actor")
		}
		return []string{carrierMatrixMission(f), "save " + carrierMatrixDigest(first), carrierMatrixMission(cold), "resave " + carrierMatrixDigest(currentRuntimeSave(t, cold))}
	})
	run("first original bound corpse", func(t *testing.T) []string {
		f := currentBoundCorpseFixture(t, 0xabcdef77, -14, 0xabcddcba)
		first := currentRuntimeSave(t, f)
		cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), first)
		return []string{carrierMatrixMission(f), "save " + carrierMatrixDigest(first), carrierMatrixMission(cold), "resave " + carrierMatrixDigest(currentRuntimeSave(t, cold))}
	})
	run("first town Diary", func(t *testing.T) []string {
		source, newFront := spellbookCitySource(t)
		f := newFront()
		if _, town, err := f.RestoreOriginal(source); err != nil || !town {
			t.Fatal("original town LOAD", town, err)
		}
		loaded := carrierMatrixTown(f)
		f.Town.knowledge = carrierMatrixKnowledge
		first := exportTown(t, f)
		cold := newFront()
		if _, town, err := cold.RestoreOriginal(first); err != nil || !town {
			t.Fatal("town LOAD", town, err)
		}
		return []string{loaded, "save " + carrierMatrixDigest(first), carrierMatrixTown(cold), "resave " + carrierMatrixDigest(exportTown(t, cold))}
	})
	run("second original mission", func(t *testing.T) []string {
		f, _, dir := secondMissionLoadFile(t, secondMissionFile(secondMissionFixtureHero(), secondMissionStore(4, 0, -1)))
		name, first := worldCarrierSave(t, f, dir)
		cold, _ := secondMissionLoad(t, dir, name)
		_, second := worldCarrierSave(t, cold, t.TempDir())
		return []string{carrierMatrixMission(f), "save " + carrierMatrixDigest(first), carrierMatrixMission(cold), "resave " + carrierMatrixDigest(second)}
	})
	run("second town Diary", func(t *testing.T) []string {
		f, _, _ := secondCampaignFixtureFiles(t, false, identityNames())
		f.Table = identityDollTable()
		f.Carried = []mapload.PartyMember{{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}}
		c := newSecondCampaign()
		c.current, c.available = partyPlace{2, 2}, []partyPlace{{2, 2}, {1, 30}, {1, 31}}
		f.Town.second, f.Town.gold = c, 500
		f.Town.settleCityGroups(f.Carried)
		f.Town.knowledge = carrierMatrixKnowledge
		first := exportTown(t, f)
		cold := coldIdentityTown(t, first)
		return []string{carrierMatrixTown(f), "save " + carrierMatrixDigest(first), carrierMatrixTown(cold), "resave " + carrierMatrixDigest(exportTown(t, cold))}
	})
	run("first loaded Player name", func(t *testing.T) []string {
		return carrierMatrixFreshPlant(t, carrierMatrixPlayerNames)
	})
	run("first loaded unknown spans", func(t *testing.T) []string {
		return carrierMatrixFreshPlant(t, carrierMatrixPlayerTails)
	})
	run("first loaded order words", func(t *testing.T) []string {
		fresh, first := carrierMatrixDeadSave(t, nil)
		doc, err := sav.DecodeDocumentData(first)
		if err != nil {
			t.Fatal(err)
		}
		dead := map[int]bool{}
		for _, root := range doc.DeadActors {
			dead[int(root)-1] = true
		}
		target, _ := savedStructureValue(&doc.Objects[doc.DeadActors[0]-1], "Identity")
		planted := carrierMatrixPlant(t, first, func(doc *sav.DocumentData) {
			for i := range doc.Objects {
				if r := &doc.Objects[i]; !dead[i] && (r.Class == "Unit" || r.Class == "Human") {
					savedObjectSetValue(r, "U40", target)
					savedObjectSetValue(r, "U5C", target)
				}
			}
		})
		cold := openCurrentRetainedRuntime(t, fresh(), planted)
		second := currentRuntimeSave(t, cold)
		return []string{"planted " + carrierMatrixDigest(planted), carrierMatrixMission(cold), carrierMatrixRecords(t, second), "resave " + carrierMatrixDigest(second)}
	})
	run("first loaded world effects", func(t *testing.T) []string {
		front := func() *FrontEnd {
			f := currentPoolFixtureFront(t, 91)
			f.SetDeterministicFrames(true)
			return f
		}
		f := front()
		openWorldEffectsTestSave(t, f, stagedWorldEffectSource(t, 4, 2, 0), "effects.sav")
		first := currentRuntimeSave(t, f)
		cold := front()
		openWorldEffectsTestSave(t, cold, first, "effects-again.sav")
		effects := len(cold.live.mission.state.savedDocument.Document.World.Effects)
		return []string{carrierMatrixMission(f), "save " + carrierMatrixDigest(first), carrierMatrixMission(cold), fmt.Sprintf("effects %d", effects), "resave " + carrierMatrixDigest(currentRuntimeSave(t, cold))}
	})
	run("first virtual dead body art", func(t *testing.T) []string {
		virtual := &poolFixtureActor{mapID: 0, cell: 0x0b0b, hp: uint16(65536 - 40), maxHP: 50, stage: 3, typeID: 0x2a}
		fresh, first := carrierMatrixDeadSave(t, virtual)
		cold := openCurrentRetainedRuntime(t, fresh(), first)
		second := currentRuntimeSave(t, cold)
		return []string{"save " + carrierMatrixDigest(first), carrierMatrixRecords(t, first), carrierMatrixMission(cold),
			fmt.Sprintf("art %v", cold.live.mission.state.DeadArt), "resave " + carrierMatrixDigest(second)}
	})
	run("second loaded Player name and unknown spans", func(t *testing.T) []string {
		f, _, dir := secondMissionLoadFile(t, secondMissionFile(secondMissionFixtureHero(), secondMissionStore(4, 0, -1)))
		_, first := worldCarrierSave(t, f, dir)
		planted := carrierMatrixPlant(t, first, func(doc *sav.DocumentData) {
			carrierMatrixPlayerNames(doc)
			carrierMatrixPlayerTails(doc)
		})
		again := t.TempDir()
		if err := os.WriteFile(filepath.Join(again, "game0002.sav"), planted, 0o600); err != nil {
			t.Fatal(err)
		}
		cold, _ := secondMissionLoad(t, again, "game0002.sav")
		_, second := worldCarrierSave(t, cold, t.TempDir())
		return []string{"planted " + carrierMatrixDigest(planted), carrierMatrixMission(cold), carrierMatrixRecords(t, second), "resave " + carrierMatrixDigest(second)}
	})
	return rows
}

// carrierMatrixPlant is a SAV document with one edit of its records.
func carrierMatrixPlant(t *testing.T, raw []byte, edit func(doc *sav.DocumentData)) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	edit(&doc)
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// carrierMatrixPlayerNames gives each Player record a name of its own.
func carrierMatrixPlayerNames(doc *sav.DocumentData) {
	for i := range doc.Objects {
		if r := &doc.Objects[i]; r.Class == "Player" {
			mustSetText(r, "Name", fmt.Sprintf("Matrix Player %d", i))
		}
	}
}

// carrierMatrixPlayerTails fills each Player tail below the formation byte,
// a span the World does not hold.
func carrierMatrixPlayerTails(doc *sav.DocumentData) {
	for i := range doc.Objects {
		if r := &doc.Objects[i]; r.Class == "Player" {
			for j := range r.Raw {
				if r.Raw[j].Name == "PRaw32" {
					for k := 0; k < sav.PlayerTailFormationByte && k < len(r.Raw[j].Bytes); k++ {
						r.Raw[j].Bytes[k] = byte(0x40 + k)
					}
				}
			}
		}
	}
}

// carrierMatrixFreshPlant SAVEs the fresh first-game mission, plants an edit
// in that SAVE, LOADs it and SAVEs again.
func carrierMatrixFreshPlant(t *testing.T, edit func(doc *sav.DocumentData)) []string {
	t.Helper()
	f, s := currentCarrierFixture(t)
	first, err := f.ExportCurrentSave(s, "carrier matrix")
	if err != nil {
		t.Fatal(err)
	}
	planted := carrierMatrixPlant(t, first, edit)
	cold := openCurrentCarrierSave(t, planted)
	second := currentRuntimeSave(t, cold)
	return []string{"planted " + carrierMatrixDigest(planted), carrierMatrixMission(cold), carrierMatrixRecords(t, second), "resave " + carrierMatrixDigest(second)}
}

// carrierMatrixDeadSave LOADs the first-game original file with a living
// actor and two dead bodies, and one more dead body when extra is not nil,
// and SAVEs it. fresh makes the front a LOAD of that SAVE starts from.
func carrierMatrixDeadSave(t *testing.T, extra *poolFixtureActor) (func() *FrontEnd, []byte) {
	t.Helper()
	zero := uint32(0)
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 197, name: "Living control"}
	late := &poolFixtureActor{mapID: 92, cell: 0x0807, hp: uint16(65536 - 237), maxHP: 421, stage: 4}
	terminal := &poolFixtureActor{mapID: 93, cell: 0x0a09, hp: uint16(65536 - 10017), maxHP: 613, stage: 5, runtime: &zero}
	dead := []*poolFixtureActor{late, terminal}
	if extra != nil {
		dead = append(dead, extra)
	}
	fresh := func() *FrontEnd { return currentPoolFixtureFront(t, 91, 92, 93) }
	f := fresh()
	raw := completeCurrentDeadDocument(t, f, savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, dead)))
	f = openCurrentRetainedRuntime(t, f, raw)
	return fresh, currentRuntimeSave(t, f)
}

// carrierMatrixRecords is what a SAVE wrote for the carriers the loaded record
// keeps: each Player's name and tail, and each actor's identity, order words
// U40 and U5C and art type T0E.
func carrierMatrixRecords(t *testing.T, raw []byte) string {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for i := range doc.Objects {
		r := &doc.Objects[i]
		switch r.Class {
		case "Player":
			name, tail := "", ""
			for _, x := range r.Texts {
				if x.Name == "Name" {
					name = x.Value
				}
			}
			for _, x := range r.Raw {
				if x.Name == "PRaw32" {
					tail = carrierMatrixDigest(x.Bytes)
				}
			}
			out = append(out, fmt.Sprintf("Player %q %s", name, tail))
		case "Unit", "Human", "Humanoid":
			id, _ := savedStructureValue(r, "Identity")
			u40, _ := savedStructureValue(r, "U40")
			u5c, _ := savedStructureValue(r, "U5C")
			t0e, _ := savedStructureValue(r, "T0E")
			out = append(out, fmt.Sprintf("%s %x U40 %x U5C %x T0E %x", r.Class, id, u40, u5c, t0e))
		}
	}
	return "records " + strings.Join(out, "; ")
}

// Each carrier the set took in writes the bytes, seats the World and town
// values and loads with the result the engine had before: over a fresh start
// and an original file of both games, their SAVE, its LOAD and the SAVE again.
func TestTheCarrierMatrixKeepsItsRecordedValues(t *testing.T) {
	got := carrierMatrixRows(t)
	for name, want := range carrierMatrix {
		row := got[name]
		if strings.Join(row, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s:\n got %q\nwant %q", name, row, want)
		}
	}
	for name, row := range got {
		if _, held := carrierMatrix[name]; !held {
			t.Errorf("the table does not hold the row %s: %q", name, row)
		}
	}
}
