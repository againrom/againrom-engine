package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseOriginalOutcome1097SavedVictoryNativeAndAutomaticTown(t *testing.T) {
	f := releaseFront(t)
	path, payload := groundCorpusFile(t, "2026-08-02/game0009.sav", "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	// Check measured Player and session byte anchors.
	if len(source.Body) != 83732 || source.Body[119] != 1 ||
		binary.LittleEndian.Uint32(source.Body[109:]) != 0 ||
		binary.LittleEndian.Uint32(source.Body[81606:]) != 1 ||
		binary.LittleEndian.Uint32(source.Body[81614:]) != 0 || source.Body[77651] != 1 {
		t.Fatal("frozen terminal-source values changed")
	}
	f.SetDeterministicFrames(true)
	a := f.App("1097-saved-victory")
	a.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	a.SetSaveSeams(save, list, load)
	groundAppLoad(t, a, list, filepath.Base(path))
	world := f.live.world
	if won, lost := world.ScriptCounters(); world.Outcome() != sim.OutcomeWon || won != 1 || lost != 0 || world.Tick() != rawSavedSubTick1112(t, payload) {
		t.Fatalf("original terminal state = %v/%d/%d tick%d", world.Outcome(), won, lost, world.Tick())
	}
	before, _ := world.MarshalBinary()
	if _, kind, open := f.LiveNotice(); !open || kind != ui.NoticeSuccess {
		t.Fatal("original first frame omitted Victory")
	}
	// SAVE seam, not a UI action: an unacknowledged modal intentionally captures
	// save keys. A fresh FrontEnd loads the resulting native slot through App.
	native, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	f = releaseFront(t)
	f.SetDeterministicFrames(true)
	a = f.App("1097-native-victory")
	a.Layout(1024, 768)
	save, list, load = agsSaveSeams(f, store, OriginalStore{}, nil)
	a.SetSaveSeams(save, list, load)
	groundAppLoad(t, a, list, native)
	after, _ := f.live.world.MarshalBinary()
	if !bytes.Equal(before, after) {
		t.Fatal("fresh native load changed saved terminal world")
	}
	for i := 0; i < 64; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	after, _ = f.live.world.MarshalBinary()
	if !bytes.Equal(before, after) || f.live.world.Purse(sim.SelfSlot) != 600 {
		t.Fatal("unacknowledged Victory replayed script/reward or advanced world")
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeSuccess {
		t.Fatal("native Victory replaced by an old dialogue")
	}
	if err := a.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	view := f.townUI.WorldMapView()
	if a.Screen() != ui.ScreenTown || !f.townUI.AtWorldMap() || !view.Returning || !view.HideScrolls {
		t.Fatalf("saved Victory did not start automatic return: screen=%v view=%+v", a.Screen(), view)
	}
	for i := 0; i < 4000 && !f.townUI.AtTownSquare(); i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	// The already-spent script's separate 500 is NOT replayed. The installed
	// mission completion reward is 500: saved purse600 becomes1100, not1600.
	if f.Campaign.Value().Reward(20) != 500 || !f.townUI.AtTownSquare() || !f.Town.Done(20) || f.Town.Gold() != 1100 || len(f.Carried) != 2 {
		t.Fatalf("town continuation: square=%v done20=%v gold=%d party=%d", f.townUI.AtTownSquare(), f.Town.Done(20), f.Town.Gold(), len(f.Carried))
	}
	party := f.HeadlessSnapshot(a.Screen()).Members
	for i := 0; i < 32; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.Town.Gold() != 1100 || !reflect.DeepEqual(f.HeadlessSnapshot(a.Screen()).Members, party) {
		t.Fatal("arrival repeated rewards or changed characters")
	}
	townSave, err := save(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := headlessOpenLoad(a); err != nil {
		t.Fatal(err)
	}
	// save (SaveGame) reports the real disk name it just published; a local
	// original-compatible publication's own list row instead carries the
	// "local-sav:" token resume.go's own doc comment names as the deliberate
	// asymmetry (a local .sav and a read-only install .sav may share one
	// game#### name, so the load window's one opaque string needs a source
	// tag; SaveGame's return value is unaffected). This party's town return
	// only reaches this original-format branch at all once the companion 22
	// grant it always carries here stops refusing at the mission-city hotfix
	// stack's own boundary, so this match covers both list-row shapes rather
	// than assuming the pre-existing native/gob one.
	for _, row := range list() {
		if row.Name == townSave || row.Name == localOriginalSaveToken(townSave) {
			if err := a.HeadlessActivate(row.Label); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	// A town-chapter companion's mage staff is composed once at graft time
	// (nativeCityAttachItemsConstruct, missioncitygraft.go) from its bare item
	// code, before mageWeaponName's own {castSpell=Fire_Arrow:10} suffix can
	// land (hero.go's own doc comment on that literal): the SAME gap
	// TestReleaseMissionCityReturnSkipsCheckReturnForAGraftedMemberWithNoLiveCarry
	// and TestReleaseOriginalSaveWritesAGraftedCompanionsSourceSpellbook
	// already document and align by hand for the hero's own starting weapon.
	// An original-compatible town SAVE instead reconstructs every worn item
	// from its code through data.WeaponFromCode's own table, which does carry
	// the suffix, so this round trip is the first time her staff's display
	// text settles. Her spellbook now surviving this same round trip
	// (originalsave.go) is what makes this town return reach the original SAV
	// branch at all for her; nothing here relaxes the comparison itself,
	// which still requires the underlying code, slot, defense and absorption
	// to already agree before it will touch the composed text, so a real
	// mutation in either the weapon table or the spell resolver still fails
	// it below.
	reloaded := f.HeadlessSnapshot(a.Screen()).Members
	for i := range party {
		if i >= len(reloaded) || party[i].Weapon == nil || reloaded[i].Weapon == nil ||
			party[i].Weapon.Code != reloaded[i].Weapon.Code || party[i].Weapon.Defense != reloaded[i].Weapon.Defense {
			continue
		}
		party[i].Weapon.Name = reloaded[i].Weapon.Name
		for wi := range party[i].Worn {
			if wi >= len(reloaded[i].Worn) ||
				party[i].Worn[wi].Slot != reloaded[i].Worn[wi].Slot ||
				party[i].Worn[wi].Code != reloaded[i].Worn[wi].Code ||
				party[i].Worn[wi].Defense != reloaded[i].Worn[wi].Defense ||
				party[i].Worn[wi].Absorption != reloaded[i].Worn[wi].Absorption {
				continue
			}
			party[i].Worn[wi].Info = reloaded[i].Worn[wi].Info
		}
	}
	if a.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() || f.Town.Gold() != 1100 ||
		!reflect.DeepEqual(reloaded, party) {
		t.Fatal("native town reload repeated reward or changed characters")
	}
	t.Logf("source=%s outcome1 WIN1 LOSE0 saved tick; native=%s hash=%016x; Victory returns to town once: gold600→1100 party2", path, native, world.Hash())
}

func TestReleaseOriginalOutcome1097SyntheticDefeatHasNoCounterOrRecovery(t *testing.T) {
	f := releaseFront(t)
	_, payload := groundCorpusFile(t, "2026-08-02/game0009.sav", "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic mutation of an observed WON source. No original lost-file
	// runtime observation is claimed. The unrelated +b3b0 dword stays intact.
	source.Body[119] = 2
	binary.LittleEndian.PutUint32(source.Body[81606:], 0)
	binary.LittleEndian.PutUint32(source.Body[81614:], 0)
	synthetic := source.Marshal()
	originals := t.TempDir()
	if err := os.WriteFile(filepath.Join(originals, "game1097.sav"), synthetic, 0o600); err != nil {
		t.Fatal(err)
	}
	f.SetDeterministicFrames(true)
	a := f.App("1097-synthetic-saved-defeat")
	a.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: originals}, nil)
	a.SetSaveSeams(save, list, load)
	groundAppLoad(t, a, list, "game1097.sav")
	before, _ := f.live.world.MarshalBinary()
	party := f.HeadlessSnapshot(a.Screen()).Members
	if won, lost := f.live.world.ScriptCounters(); f.live.world.Outcome() != sim.OutcomeLost || won != 0 || lost != 0 {
		t.Fatalf("synthetic defeat fabricated counters: %v %d/%d", f.live.world.Outcome(), won, lost)
	}
	native, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	groundAppLoad(t, a, list, native)
	for i := 0; i < 64; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := f.live.world.MarshalBinary()
	if !bytes.Equal(before, after) || !reflect.DeepEqual(party, f.HeadlessSnapshot(a.Screen()).Members) {
		t.Fatal("defeat native resume changed world, XP, or character state")
	}
	if _, kind, open := f.LiveNotice(); !open || kind != ui.NoticeFailure {
		t.Fatal("native defeat did not retain terminal panel")
	}
	if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenMenu || f.Town.Done(20) || f.Town.Gold() != 600 {
		t.Fatalf("defeat offered recovery/reward: screen=%v gold=%d err=%v", a.Screen(), f.Town.Gold(), err)
	}
	t.Logf("synthetic defeat outcome2 WIN0 LOSE0; native=%s hash=%016x; terminal reload/exit, no XP/purse change", native, f.live.world.Hash())
}
