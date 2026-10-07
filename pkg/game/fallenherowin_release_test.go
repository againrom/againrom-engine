package game

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The owner's original SAV of mission 20, the merchant escort: the hero stands
// beside the merchant with the merchant's three guards, a sack lies on the
// next cell, and a camp of monsters waits to the west.
const (
	fallenHeroWinSAV  = "2026-08-02/game0008.sav"
	fallenHeroWinHash = "ebe713078281f3a4fef57fbd5e71be7899b39d98846fa6260f7ba845fab0d466"
)

// fallenHeroWinSavedHero reads the one character a SAV's Player names as its
// hero through the format reader, not through the loader under test.
func fallenHeroWinSavedHero(t *testing.T, raw []byte) sav.Character {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := file.Party()
	if err != nil {
		t.Fatal(err)
	}
	var heroes []sav.Character
	for _, c := range party {
		if c.Hero {
			heroes = append(heroes, c)
		}
	}
	if len(heroes) != 1 {
		t.Fatalf("SAV names %d hero characters, want 1", len(heroes))
	}
	return heroes[0]
}

// fallenHeroWinColdLoad opens main-menu LOAD by name, or requires one file.
func fallenHeroWinColdLoad(t *testing.T, store SaveStore, names ...string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("fallen hero win cold LOAD")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if len(names) == 1 {
		_, list, _ := f.SaveSeams(store, OriginalStore{}, nil)
		groundAppLoad(t, app, list, names[0])
		return f, app
	}
	if len(names) != 0 {
		t.Fatal("cold LOAD requires one SAV name")
	}
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal("cold main-menu LOAD:", err)
	}
	rows := app.HeadlessRows()
	if app.Screen() != ui.ScreenLoad || len(rows) != 1 {
		t.Fatalf("cold LOAD lists %v on %s, want the one file", rows, app.Screen())
	}
	if err := app.HeadlessActivate(rows[0].Text); err != nil || app.Screen() == ui.ScreenLoad {
		t.Fatalf("cold LOAD refused the file: %v %s", err, app.HeadlessMessage())
	}
	return f, app
}

// fallenHeroWinCodes is a multiset of item codes, one entry per item, sorted.
func fallenHeroWinCodes(pieces []sav.Piece) []uint16 {
	var codes []uint16
	for _, p := range pieces {
		for range max(p.Stack, 1) {
			codes = append(codes, p.Code)
		}
	}
	slices.Sort(codes)
	return codes
}

func fallenHeroWinSorted(codes []uint16) []uint16 {
	out := slices.DeleteFunc(slices.Clone(codes), func(c uint16) bool { return c == 0 })
	slices.Sort(out)
	return out
}

// TestReleaseFallenHeroWinKeepsHisMission cold-LOADs the original mission-20
// SAV. The hero picks up the sack beside him, rests, and with the guards kills
// the nearest monster, which earns him experience. Back beside the merchant,
// the K key's command fells him, and the merchant walks to the escort's end
// while the hero lies at health -1 through -9: the script wins. The mission-end
// cull raises him (PARTY-ENDCULL-026), so the town, the city SAVE, its cold
// LOAD and mission 30's entry all hold the experience and the pack he ended
// the mission with, the set he wore, and full health.
func TestReleaseFallenHeroWinKeepsHisMission(t *testing.T) {
	_, raw := groundCorpusFile(t, fallenHeroWinSAV, fallenHeroWinHash)
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, filepath.Base(fallenHeroWinSAV)), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := fallenHeroWinColdLoad(t, store)
	if f.liveMission != 20 || f.live == nil {
		t.Fatalf("cold LOAD opened mission %d, want 20", f.liveMission)
	}
	refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
	hero, merchant, monster := f.live.mission.ids[0], refs[136], refs[53]
	loaded, _ := f.live.entity(hero)
	loadedPack, _ := f.live.world.Carried(hero)
	loadedWorn, _ := f.live.world.Equipped(hero)
	source := fallenHeroWinSavedHero(t, raw)
	for i, xp := range source.SkillXP {
		if int32(xp) != loaded.SkillXP[i] {
			t.Fatalf("loaded hero experience %v, SAV hero %v", loaded.SkillXP, source.SkillXP)
		}
	}
	step := func(phase string) {
		t.Helper()
		if _, kind, up := f.LiveNotice(); up && kind != ui.NoticeSuccess {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if f.live.mission.announced && f.live.mission.outcome != sim.OutcomeWon {
			e, _ := f.live.entity(hero)
			t.Fatalf("%s: mission announced %v with the hero at health %d", phase, f.live.mission.outcome, e.HP)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}

	f.live.orderPickup(hero, 87, 104)
	for i := 0; i < 256 && f.live.pickup.set; i++ {
		step("pickup")
	}
	for i := 0; i < 2048; i++ {
		if e, _ := f.live.entity(hero); e.HP == e.MaxHP {
			break
		}
		step("rest")
	}
	f.live.strike(uint32(hero), uint32(monster))
	for _, unit := range []uint16{137, 138, 139} {
		f.live.strike(uint32(refs[unit]), uint32(monster))
	}
	for i := 0; i < 1024; i++ {
		if e, _ := f.live.entity(monster); !e.Alive() {
			break
		}
		step("fight")
	}
	f.live.enqueue(uint32(hero), 88, 105)
	for i := 0; i < 512; i++ {
		if e, _ := f.live.entity(hero); e.X == 88 && e.Y == 105 {
			break
		}
		step("walk back")
	}
	earned, _ := f.live.entity(hero)
	pack, _ := f.live.world.Carried(hero)
	worn, _ := f.live.world.Equipped(hero)
	if victim, _ := f.live.entity(monster); victim.Alive() || !earned.Alive() || earned.SkillXP == loaded.SkillXP ||
		len(pack) != len(loadedPack)+1 || worn != loadedWorn {
		t.Fatalf("setup: monster alive=%v, hero alive=%v experience %v (loaded %v), pack %d items (loaded %d), worn %v (loaded %v)",
			victim.Alive(), earned.Alive(), earned.SkillXP, loaded.SkillXP, len(pack), len(loadedPack), worn, loadedWorn)
	}

	f.live.affect(uint32(hero), true)
	f.live.enqueue(uint32(merchant), 111, 131)
	for i := 0; i < 1024; i++ {
		if _, kind, up := f.LiveNotice(); up && kind == ui.NoticeSuccess {
			break
		}
		step("escort")
	}
	fallen, _ := f.live.entity(hero)
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeSuccess || f.live.world.Outcome() != sim.OutcomeWon {
		t.Fatalf("no Victory: notice up=%v kind %v, outcome %v, hero at health %d", up, kind, f.live.world.Outcome(), fallen.HP)
	}
	fallenPack, _ := f.live.world.Carried(hero)
	if fallen.Alive() || !fallen.Restorable() || fallen.HP < -9 || fallen.SkillXP != earned.SkillXP || !slices.Equal(fallenPack, pack) {
		t.Fatalf("hero at Victory: health %d restorable=%v experience %v, want a fallen body Heal can raise holding %v",
			fallen.HP, fallen.Restorable(), fallen.SkillXP, earned.SkillXP)
	}
	finished := f.live.world

	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal("Victory acknowledgment:", err)
	}
	for i := 0; i < 4000 && (app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare()); i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal("return frame:", err)
		}
	}
	if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() || !f.Town.Open() || f.Town.Chapter() != 30 {
		t.Fatalf("Victory returned to screen %s, town open=%v chapter %d: want the town square at chapter 30", app.Screen(), f.Town.Open(), f.Town.Chapter())
	}
	if len(f.Carried) == 0 || f.Carried[0].ID != f.live.mission.party[0].ID || f.Carried[0].Carry == nil {
		t.Fatalf("town party of %d does not open with the hero carrying his mission", len(f.Carried))
	}
	carry := f.Carried[0].Carry
	if carry.SkillXP != earned.SkillXP || !slices.Equal(carry.Items, pack) || carry.Equipped != worn {
		t.Fatalf("town hero carries experience %v, %d items, worn %v; want %v, %d items, worn %v",
			carry.SkillXP, len(carry.Items), carry.Equipped, earned.SkillXP, len(pack), worn)
	}
	var raised sim.Entity
	for _, e := range finished.Entities() {
		if e.ID == hero {
			raised = e
		}
	}
	if !raised.Alive() || raised.HP != raised.MaxHP || raised.Decay != sim.DecayNone || raised.Defence != fallen.Defence<<1 {
		t.Fatalf("finished world hero health %d/%d stage %d defence %d, want him raised at full health with defence %d",
			raised.HP, raised.MaxHP, raised.Decay, raised.Defence, fallen.Defence<<1)
	}
	if f.originalCity != nil && f.originalCity.unavailable != nil {
		t.Fatalf("city SAVE state unavailable: %v", f.originalCity.unavailable)
	}

	cityStore := SaveStore{Dir: t.TempDir()}
	prepared, err := f.SaveDialogSeams(cityStore, OriginalStore{}).Prepare(ui.SaveRequest{Directory: cityStore.Dir, Name: "Town after the escort", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal("city SAVE prepare:", err)
	}
	paths, err := prepared.Commit(false)
	if err != nil || len(paths) != 1 {
		t.Fatal("city SAVE commit:", paths, err)
	}
	written, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	saved := fallenHeroWinSavedHero(t, written)
	for i, xp := range saved.SkillXP {
		if int32(xp) != earned.SkillXP[i] {
			t.Fatalf("city SAV hero experience %v, want %v", saved.SkillXP, earned.SkillXP)
		}
	}
	if saved.Stats[sav.StatHealth] != saved.Stats[sav.StatHealthMax] || saved.Stage != 0 ||
		!slices.Equal(fallenHeroWinCodes(saved.Items), fallenHeroWinSorted(pack)) ||
		!slices.Equal(fallenHeroWinCodes(saved.Worn), fallenHeroWinSorted(worn[:])) {
		t.Fatalf("city SAV hero health %d/%d stage %d, items %v, worn %v; want full health at stage 0 holding %v and wearing %v",
			saved.Stats[sav.StatHealth], saved.Stats[sav.StatHealthMax], saved.Stage,
			fallenHeroWinCodes(saved.Items), fallenHeroWinCodes(saved.Worn), fallenHeroWinSorted(pack), fallenHeroWinSorted(worn[:]))
	}

	cold, coldApp := fallenHeroWinColdLoad(t, cityStore)
	if !cold.Town.Open() || cold.Town.Chapter() != 30 || len(cold.Carried) == 0 || cold.Carried[0].Carry == nil {
		t.Fatalf("cold LOAD of the city SAV: town open=%v chapter %d, party of %d", cold.Town.Open(), cold.Town.Chapter(), len(cold.Carried))
	}
	if c := cold.Carried[0].Carry; c.SkillXP != earned.SkillXP || !reflect.DeepEqual(c.Items, pack) || c.Equipped != worn {
		t.Fatalf("cold LOAD hero carries experience %v, %d items, worn %v; want %v, %d items, worn %v",
			c.SkillXP, len(c.Items), c.Equipped, earned.SkillXP, len(pack), worn)
	}
	takeCampaignOffer(t, cold, 30)
	if err := coldApp.OpenMission(cold.MissionOpenerWith(30, cold.NextParty())); err != nil {
		t.Fatal("mission 30 entry:", err)
	}
	next, _ := cold.live.entity(cold.live.mission.ids[0])
	nextPack, _ := cold.live.world.Carried(next.ID)
	nextWorn, _ := cold.live.world.Equipped(next.ID)
	if !next.Alive() || next.HP != next.MaxHP || next.SkillXP != earned.SkillXP || !slices.Equal(nextPack, pack) || nextWorn != worn {
		t.Fatalf("mission 30 hero health %d/%d experience %v, %d items, worn %v; want full health, %v, %d items, worn %v",
			next.HP, next.MaxHP, next.SkillXP, len(nextPack), nextWorn, earned.SkillXP, len(pack), worn)
	}
	t.Logf("hero fallen at health %d at Victory; experience %v -> %v; pack %d -> %d items; defence %d loaded, %d fallen, %d raised, %d at mission 30",
		fallen.HP, loaded.SkillXP, earned.SkillXP, len(loadedPack), len(pack), loaded.Defence, fallen.Defence, raised.Defence, next.Defence)
}
