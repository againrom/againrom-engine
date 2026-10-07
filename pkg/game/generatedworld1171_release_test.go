package game

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseGeneratedWorldSAV1171(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "First world", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("generated world SAV")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	initial, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	assertNativeWorldWire(t, f, initial)
	var hero, target sim.Entity
	heroFound, targetFound := false, false
	distance := int32(1 << 30)
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot {
			hero, heroFound = e, true
			break
		}
	}
	unhurt := map[sim.EntityID]int32{}
	for _, e := range f.live.world.Entities() {
		unhurt[e.ID] = e.HP
		dx, dy := e.X-hero.X, e.Y-hero.Y
		if e.Domain == hero.Domain && f.live.world.Relations().Hostile(hero.Owner, e.Owner) && dx*dx+dy*dy < distance {
			distance = dx*dx + dy*dy
			target, targetFound = e, true
		}
	}
	if !heroFound || !targetFound {
		t.Fatal("native melee fixture lacks a hero or hostile target")
	}
	t.Logf("hero %d at %d,%d targets %d at %d,%d HP%d", hero.ID, hero.X, hero.Y, target.ID, target.X, target.Y, target.HP)
	f.live.strike(uint32(hero.ID), uint32(target.ID))

	// Save at the actual damage boundary and at both following checkpoints.
	// A valid current state cannot postpone SAVE until an easier walking state.
	const margin, window = 20, 40
	entity := generatedCampaignEntity
	store := SaveStore{Dir: t.TempDir()}
	var native, cold, secondCold, thirdCold *FrontEnd
	var coldApp *ui.App
	var natural, first, second, cli, blow, fall uint64
	combat, lossChecked, stage := false, false, 0
	successor := func(want, got *FrontEnd, cut string) {
		assertCurrentWorldEqual(t, want.live.world, got.live.world, cut)
	}
	firstSAV := func() {
		f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
		openMissionGameMenu(t, app)
		s, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		rawAGS, err := EncodeSave(s, "same native checkpoint")
		if err != nil {
			t.Fatal(err)
		}
		decoded, _, err := DecodeSave(rawAGS)
		if err != nil {
			t.Fatal(err)
		}
		native = releaseFront(t)
		native.SetDeterministicFrames(true)
		nativeApp := native.App("same native checkpoint")
		open, town, err := native.Restore(decoded)
		if err != nil || town {
			t.Fatal(err)
		}
		if err = nativeApp.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		nativeApp.Layout(1024, 768)
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveEdit(store.Dir, "first current combat", ui.SaveSAV); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("save"); err != nil {
			t.Fatal(err)
		}
		entries, err := store.List()
		if err != nil || len(entries) != 1 || !strings.HasSuffix(entries[0].Name, ".sav") {
			t.Fatalf("ordinary generated SAVE: %v %v", entries, err)
		}
		out, err := store.Read(entries[0].Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sav.Open(out); err != nil {
			t.Fatal(err)
		}
		t.Logf("ordinary generated actors=%d structures=%d SAV=%d", len(f.live.world.Entities()), len(f.live.world.Structures()), len(out))
		cold, coldApp = loadSAVWindow(t, store, entries[0].Name)
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		t.Logf("cold actors=%d structures=%d", len(cold.live.world.Entities()), len(cold.live.world.Structures()))
		t.Logf("checkpoint RNG native=%#x cold=%#x", f.live.world.RandomState(), cold.live.world.RandomState())
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "first checkpoint")
	}
	secondSAV := func() {
		cold.ConfigureSaveSeams(coldApp, store, OriginalStore{}, nil)
		openMissionGameMenu(t, coldApp)
		if _, _, err := cold.Snapshot(true); err != nil {
			t.Fatal(err)
		}
		if err := coldApp.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		if err := coldApp.HeadlessSaveEdit(store.Dir, "second current combat", ui.SaveSAV); err != nil {
			t.Fatal(err)
		}
		if err := coldApp.HeadlessSaveAction("save"); err != nil {
			t.Fatal(err)
		}
		entries, err := store.List()
		if err != nil || len(entries) != 2 {
			t.Fatalf("second ordinary SAVE %v %v", entries, err)
		}
		// Identify the entry whose native counter is the second cut directly; list
		// order is UI policy and the other entry is the earlier ordinary SAVE. A
		// literal byte-for-byte copy under a third name would share its label with
		// its source entry and make the picker's own activation ambiguous.
		var secondName string
		for _, e := range entries {
			b, readErr := store.Read(e.Name)
			if readErr != nil {
				t.Fatal(readErr)
			}
			file, readErr := sav.Open(b)
			if readErr == nil && file.Head.CounterA == uint32(cold.live.world.Tick()) {
				secondName = e.Name
			}
		}
		if secondName == "" {
			t.Fatal("second ordinary SAV absent")
		}
		if err := coldApp.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		secondCold, _ = loadSAVWindow(t, store, secondName)
		assertCurrentWorldEqual(t, cold.live.world, secondCold.live.world, "second checkpoint")
	}
	cliSAV := func() {
		// The legacy import converter reaches the same current-state producer
		// through a fresh receiver, independently of the App menu entry.
		secondSnapshot, _, err := secondCold.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		secondAGS, err := EncodeSave(secondSnapshot, "generated world CLI route")
		if err != nil {
			t.Fatal(err)
		}
		receiver := releaseFront(t)
		receiver.SetDeterministicFrames(true)
		cliConverted, _, err := receiver.ConvertSave(secondAGS, "sav", nil)
		if err != nil {
			t.Fatal("CLI ags->sav converter must accept the generated-world checkpoint", err)
		}
		if _, err := sav.Open(cliConverted); err != nil {
			t.Fatal(err)
		}
		cliStore := SaveStore{Dir: t.TempDir()}
		cliName, err := cliStore.WriteOriginal("", cliConverted)
		if err != nil {
			t.Fatal(err)
		}
		thirdCold, _ = loadSAVWindow(t, cliStore, cliName)
		assertCurrentWorldEqual(t, secondCold.live.world, thirdCold.live.world, "CLI route checkpoint")
		t.Logf("CLI ags->sav converter: %d bytes, cold actors=%d structures=%d", len(cliConverted), len(thirdCold.live.world.Entities()), len(thirdCold.live.world.Structures()))
	}
	var due uint64
	for n := 0; ; n++ {
		now := f.live.world.Tick()
		if stage < 3 && blow != 0 && now >= due {
			switch stage {
			case 0:
				firstSAV()
				first = now
			case 1:
				secondSAV()
				second = now
			default:
				cliSAV()
				cli = now
			}
			t.Logf("SAV %d at required checkpoint %d", stage+1, now)
			stage, due = stage+1, now+window
		}
		if !lossChecked && combat {
			s, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			assertCurrentCombatWire(t, f, s, initial)
			lossChecked = true
			t.Logf("loss controls at first changed health tick %d", natural)
		}
		if stage == 3 && lossChecked && blow != 0 && fall != 0 && now >= cli+window && now >= fall+margin {
			break
		}
		if n == 2400 {
			liveHero, victim := entity(t, f, hero.ID), entity(t, f, target.ID)
			if blow == 0 {
				t.Fatalf("no melee damage tick%d hero=(%d,%d) target=(%d,%d) attack%v path%v", now, liveHero.X, liveHero.Y, victim.X, victim.Y, liveHero.HasAttackTarget, liveHero.HasTarget)
			}
			t.Fatalf("compared windows incomplete at tick %d: SAV stage %d, blow %d, fall %d", now, stage, blow, fall)
		}
		f.live.tick()
		if native != nil {
			native.live.tick()
			originalForm, _ := f.live.world.MarshalBinary()
			nativeForm, _ := native.live.world.MarshalBinary()
			if !bytes.Equal(originalForm, nativeForm) {
				t.Fatalf("native checkpoint differs at successor%d", f.live.world.Tick()-first)
			}
			cold.live.tick()
			successor(f, cold, fmt.Sprintf("first successor%d", f.live.world.Tick()-first))
		}
		if secondCold != nil {
			secondCold.live.tick()
			successor(cold, secondCold, fmt.Sprintf("second successor%d", f.live.world.Tick()-second))
		}
		if thirdCold != nil {
			thirdCold.live.tick()
			successor(secondCold, thirdCold, fmt.Sprintf("CLI successor%d", f.live.world.Tick()-cli))
		}
		now = f.live.world.Tick()
		victim := entity(t, f, target.ID)
		liveHero := entity(t, f, hero.ID)
		dx, dy := victim.X-liveHero.X, victim.Y-liveHero.Y
		if blow == 0 && dx*dx+dy*dy < 8 {
			for _, fe := range []*FrontEnd{f, native, cold, secondCold, thirdCold} {
				if fe != nil {
					fe.live.strike(uint32(entity(t, fe, hero.ID).ID), uint32(entity(t, fe, target.ID).ID))
				}
			}
		}
		if blow == 0 && victim.HP < target.HP && victim.HasKillCredit && victim.KillCreditSource == hero.ID {
			blow, due = now, now
			t.Logf("melee damage at tick %d: %d -> %d", now, target.HP, victim.HP)
		}
		if fall == 0 && !victim.Alive() {
			fall = now
			t.Logf("target falls at tick %d", now)
		}
		for _, e := range f.live.world.Entities() {
			if !combat && e.HP < unhurt[e.ID] {
				combat, natural = true, now
			}
		}
	}
	if first != blow || second != first+window || cli != second+window || fall <= first {
		t.Fatalf("exact checkpoints lost: blow%d first%d second%d CLI%d fall%d", blow, first, second, cli, fall)
	}
	t.Logf("compared windows: first %d-%d, second %d-%d, CLI %d-%d", first, f.live.world.Tick(), second, f.live.world.Tick(), cli, f.live.world.Tick())
}
