package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The pinned census names Goblin_Sling as a class with both thresholds:
// Withdraw 30 and Wimpy 8 (AI-WITHDRAW-027). Units columns 34 and 35.
const withdrawalFixtureWithdraw, withdrawalFixtureWimpy = 30, 8

// withdrawalFixtureFront opens the fixture mission whose creature (map unit
// 91) is placed from a Units row carrying both thresholds; 92 is a person
// with neither.
func withdrawalFixtureFront(t *testing.T) *FrontEnd {
	t.Helper()
	mapBytes := poolFixtureMap(91, 92)
	for off := 20; off+20 <= len(mapBytes); {
		size := int(binary.LittleEndian.Uint32(mapBytes[off+8:]))
		if binary.LittleEndian.Uint32(mapBytes[off+12:]) == 6 {
			binary.LittleEndian.PutUint16(mapBytes[off+20+8:], unitsArmKey)
			break
		}
		off += 20 + size
	}
	f := poolFixtureFrontMap(t, mapBytes)
	f.Table = actorRegistryTable()
	row := unitsRow("Goblin_Sling", map[int]int32{4: 70, 34: withdrawalFixtureWithdraw, 35: withdrawalFixtureWimpy})
	f.Table.Units = dbCollection{row, row}
	f.Humans = fourBaseHumans()
	f.Table.Humans = append(fourBaseHumans(), dbEntry{name: "actor runtime", params: humansParams(50, 0, 5)})
	f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	if err := f.App("withdrawal thresholds").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	return f
}

func withdrawalEntity(t *testing.T, w *sim.World, mapID uint16) sim.Entity {
	t.Helper()
	return poolEntity(t, w, mapID)
}

// withdrawalSaved reads both thresholds from the actor's Order block.
func withdrawalSaved(t *testing.T, doc sav.DocumentData, a *currentActionData, id sim.EntityID) (int32, int32) {
	t.Helper()
	withdraw, wimpy, ok := sav.OrderWithdrawalThresholds(savedRecordRawForTest(t, castOrderRecord(t, doc, a, id), "U158"))
	if !ok {
		t.Fatalf("actor %d has no complete Order block", id)
	}
	return withdraw, wimpy
}

// A SAVE of a mission the engine started writes each actor's class
// thresholds into Order +0x40/+0x44, where the original's withdrawal test
// reads them; a cold LOAD restores them and a second SAVE writes them again.
func TestSaveWritesClassWithdrawalThresholds(t *testing.T) {
	f := withdrawalFixtureFront(t)
	creature, human := withdrawalEntity(t, f.live.world, 91), withdrawalEntity(t, f.live.world, 92)
	if creature.Withdraw != withdrawalFixtureWithdraw || creature.Wimpy != withdrawalFixtureWimpy {
		t.Fatalf("the class creature starts with Withdraw %d Wimpy %d, want %d and %d", creature.Withdraw, creature.Wimpy, withdrawalFixtureWithdraw, withdrawalFixtureWimpy)
	}
	raw, doc, a := saveCurrentEffect(t, f)
	if withdraw, wimpy := withdrawalSaved(t, doc, a, creature.ID); withdraw != withdrawalFixtureWithdraw || wimpy != withdrawalFixtureWimpy {
		t.Fatalf("SAVE wrote Order +0x40 %d +0x44 %d, want %d and %d", withdraw, wimpy, withdrawalFixtureWithdraw, withdrawalFixtureWimpy)
	}
	if withdraw, wimpy := withdrawalSaved(t, doc, a, human.ID); withdraw != human.Withdraw || wimpy != human.Wimpy {
		t.Fatalf("SAVE wrote the Human's thresholds %d/%d, want %d/%d", withdraw, wimpy, human.Withdraw, human.Wimpy)
	}

	cold := openCurrentEffectSave(t, f, raw)
	loaded := withdrawalEntity(t, cold.live.world, 91)
	if loaded.Withdraw != withdrawalFixtureWithdraw || loaded.Wimpy != withdrawalFixtureWimpy {
		t.Fatalf("cold LOAD restored Withdraw %d Wimpy %d", loaded.Withdraw, loaded.Wimpy)
	}
	_, second, b := saveCurrentEffect(t, cold)
	if withdraw, wimpy := withdrawalSaved(t, second, b, loaded.ID); withdraw != withdrawalFixtureWithdraw || wimpy != withdrawalFixtureWimpy {
		t.Fatalf("second SAVE wrote %d/%d", withdraw, wimpy)
	}
}

// The Wimpy command rewrites both thresholds (AI-CLASS-030); the SAVE writes
// the changed values, not the class ones, and they survive LOAD and SAVE.
func TestSaveWritesChangedWithdrawalThresholds(t *testing.T) {
	f := withdrawalFixtureFront(t)
	creature := withdrawalEntity(t, f.live.world, 91)
	sim.Step(f.live.world, []sim.Command{sim.SetPlayerParameter(creature.Owner, sim.PlayerParameterRetreat, sim.RetreatModeHigh)})
	changed := withdrawalEntity(t, f.live.world, 91)
	want := int32(int64(changed.MaxHP) * 30 / 100)
	if changed.Withdraw != want || changed.Wimpy != 0 || want == withdrawalFixtureWithdraw {
		t.Fatalf("the command left Withdraw %d Wimpy %d, want %d and 0", changed.Withdraw, changed.Wimpy, want)
	}
	raw, doc, a := saveCurrentEffect(t, f)
	if withdraw, wimpy := withdrawalSaved(t, doc, a, changed.ID); withdraw != want || wimpy != 0 {
		t.Fatalf("SAVE wrote %d/%d, want the changed %d/0", withdraw, wimpy, want)
	}
	cold := openCurrentEffectSave(t, f, raw)
	if loaded := withdrawalEntity(t, cold.live.world, 91); loaded.Withdraw != want || loaded.Wimpy != 0 {
		t.Fatalf("cold LOAD restored %d/%d, want %d/0", loaded.Withdraw, loaded.Wimpy, want)
	}
	_, second, b := saveCurrentEffect(t, cold)
	if withdraw, wimpy := withdrawalSaved(t, second, b, changed.ID); withdraw != want || wimpy != 0 {
		t.Fatalf("second SAVE wrote %d/%d, want %d/0", withdraw, wimpy, want)
	}
}

// LOAD takes the thresholds from the file's Order block: zeros an older
// writer left there stay zero although the class carries nonzero values,
// as the original keeps them.
func TestLoadKeepsZeroWithdrawalThresholds(t *testing.T) {
	f := withdrawalFixtureFront(t)
	creature := withdrawalEntity(t, f.live.world, 91)
	_, doc, a := saveCurrentEffect(t, f)
	order := savedRecordRawForTest(t, castOrderRecord(t, doc, a, creature.ID), "U158")
	sav.PutOrderWithdrawalThresholds(order, 0, 0)
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := openCurrentEffectSave(t, f, raw)
	if loaded := withdrawalEntity(t, cold.live.world, 91); loaded.Withdraw != 0 || loaded.Wimpy != 0 {
		t.Fatalf("LOAD of zero Order thresholds gave Withdraw %d Wimpy %d", loaded.Withdraw, loaded.Wimpy)
	}
	_, second, b := saveCurrentEffect(t, cold)
	if withdraw, wimpy := withdrawalSaved(t, second, b, creature.ID); withdraw != 0 || wimpy != 0 {
		t.Fatalf("SAVE after the zero LOAD wrote %d/%d", withdraw, wimpy)
	}
}

// An original's SAV carries no action carrier: LOAD takes the thresholds from
// the Order block alone, not from the class row, and the next SAVE writes
// them back where they were.
func TestOriginalLoadReadsOrderWithdrawalThresholds(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("original withdrawal")
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, withdrawal: &[2]int32{7, 5}}
	b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 9, maxHP: 41, human: true}
	payload := savedContainer(poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{nil, a, a}, {}, {b, nil}}}}, nil))
	originals := t.TempDir()
	if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9999.sav")
	creature := poolEntity(t, f.live.world, 91)
	if creature.Withdraw != 7 || creature.Wimpy != 5 {
		t.Fatalf("original LOAD gave Withdraw %d Wimpy %d, want the Order's 7 and 5", creature.Withdraw, creature.Wimpy)
	}
	_, doc, carrier := saveCurrentEffect(t, f)
	if withdraw, wimpy := withdrawalSaved(t, doc, carrier, creature.ID); withdraw != 7 || wimpy != 5 {
		t.Fatalf("SAVE after the original LOAD wrote %d/%d, want 7/5", withdraw, wimpy)
	}
	// The loaded Order block is not the source: a changed current value is
	// what the next SAVE writes.
	sim.Step(f.live.world, []sim.Command{sim.SetPlayerParameter(creature.Owner, sim.PlayerParameterRetreat, sim.RetreatModeLow)})
	changed := poolEntity(t, f.live.world, 91)
	if changed.Withdraw == 7 || changed.Wimpy != 0 {
		t.Fatalf("the command left Withdraw %d Wimpy %d", changed.Withdraw, changed.Wimpy)
	}
	_, doc, carrier = saveCurrentEffect(t, f)
	if withdraw, wimpy := withdrawalSaved(t, doc, carrier, creature.ID); withdraw != changed.Withdraw || wimpy != 0 {
		t.Fatalf("SAVE over a loaded Order wrote %d/%d, want the changed %d/0", withdraw, wimpy, changed.Withdraw)
	}
}
