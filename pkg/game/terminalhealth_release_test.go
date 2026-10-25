package game

import (
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
)

func TestReleaseTerminalHealthMaximumOrdinarySAV(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	_, source := groundCorpusFile(t, "2026-08-02/game0002.sav", "b1ce079cc2b3f1bc101862c1dcf2afd8237421458b3e2474c4447efe5df2e761")
	doc, err := sav.DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	maxima := map[uint16]uint16{20: 421, 36: 613}
	var living uint16
	for i := range doc.Objects {
		actor := &doc.Objects[i]
		if actor.Class != "Unit" && actor.Class != "Human" && actor.Class != "Humanoid" {
			continue
		}
		id, _ := savedStructureValue(actor, "T08")
		stage, _ := savedStructureValue(actor, "Stage")
		health, _ := savedStructureValue(actor, "Health")
		if living == 0 && id != 0 && id != 20 && id != 36 && stage == 0 && int16(health) > 0 && health <= 197 {
			living = uint16(id)
			maxima[living] = 197
		}
		if maxHP, selected := maxima[uint16(id)]; selected {
			mustSetValue(actor, "HealthMax", uint32(maxHP))
		}
	}
	if living == 0 {
		t.Fatal("source has no living authored control")
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	wire := terminalHealthWire(t, raw)
	want := map[uint16]terminalHealthWords{}
	for id, maxHP := range maxima {
		words, present := wire[id]
		if !present || words.maxHP != maxHP {
			t.Fatal("independent source maximum differs", id, words, maxHP)
		}
		want[id] = words
	}
	if want[20].health != -237 || want[20].stage != 4 || want[36].health != -10017 || want[36].stage != 5 || want[living].stage != 0 {
		t.Fatal("source late-dead, terminal or living population changed", want)
	}
	app, _ := openOriginalSAVApp(t, f, raw, "terminal-health-source.sav")
	for cycle := range 2 {
		_, _, written := menuSAVE(t, f, app, OriginalStore{})
		requireTerminalHealthWire(t, written, want)
		cold := releaseFront(t)
		coldApp, _ := openOriginalSAVApp(t, cold, written, "terminal-health-current.sav")
		for _, id := range []uint16{living, 20} {
			actor := poolEntity(t, cold.live.world, id)
			if actor.HP != int32(want[id].health) || actor.MaxHP != int32(want[id].maxHP) {
				t.Errorf("cycle %d cold actor %d HP/MaxHP = %d/%d, want %d/%d", cycle, id, actor.HP, actor.MaxHP, want[id].health, want[id].maxHP)
			}
		}
		for _, actor := range cold.live.world.Entities() {
			if actor.MapUnitID == 36 {
				t.Fatal("terminal source actor returned to the live population")
			}
		}
		f, app = cold, coldApp
	}
}
