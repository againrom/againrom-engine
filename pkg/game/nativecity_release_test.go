package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func nativeCityBareMember(m mapload.PartyMember) mapload.PartyMember {
	m.Worn, m.WornItems = [sim.EquipSlots]uint16{}, [sim.EquipSlots]sim.ItemInstance{}
	m.Carried, m.CarriedItems = nil, nil
	m.Weapon, m.WeaponMaterialized = nil, false
	m.KnownSpells, m.Book = 0, sim.Spellbook{}
	m.SpellbookPresent, m.SpellbookRestored = false, false
	return m
}

func TestReleaseNativeTownSaveBarePartySessionStateRoundTripsAsSAV(t *testing.T) {
	f := releaseFront(t)
	result := ui.ChargenResult{Name: "Native Hero", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	if len(generated) != 1 {
		t.Fatalf("ChargenParty length = %d, want 1", len(generated))
	}
	generated[0] = nativeCityBareMember(generated[0])
	f.Carried = generated
	f.Town.Won(10)
	f.Town.Won(20)
	f.arriveInTown()
	if f.originalCity != nil {
		t.Fatal("a campaign started in Againrom must never bind an original city baseline")
	}
	const purse = 543210
	f.Town.gold = purse
	wantChapter := f.Town.Chapter()

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot.Mission != 0 {
		t.Fatalf("town snapshot mission = %d, want 0", snapshot.Mission)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ExportNativeCitySave refused current session state: %v", err)
	}

	// Fresh-process witness: the sole producer writes SAV and a brand-new
	// FrontEnd loads it back to the same town state.
	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q, want the sole SAV format", name)
	}
	restored := releaseFront(t)
	_, _, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{Dir: dir}, nil)
	open, town, err := load(name)
	if err != nil || !town || open != nil {
		t.Fatalf("load SAV town save = opener %v town %v err %v", open != nil, town, err)
	}
	foundHero := false
	for _, member := range restored.Carried {
		if member.Name == generated[0].Name && member.StartingHero {
			foundHero = true
		}
	}
	if !foundHero {
		t.Fatalf("restored roster = %+v, want the starting hero named %q", restored.Carried, generated[0].Name)
	}
	if restored.Town.Gold() != purse {
		t.Fatalf("restored gold = %d, want %d", restored.Town.Gold(), purse)
	}
	if restored.Town.Chapter() != wantChapter {
		t.Fatalf("restored chapter = %d, want %d", restored.Town.Chapter(), wantChapter)
	}
}

func TestReleaseNativeTownSaveRosterChangeAcrossSAVSaves(t *testing.T) {
	f := releaseFront(t)
	result := ui.ChargenResult{Name: "Roster Hero", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	generated[0] = nativeCityBareMember(generated[0])
	second := mapload.PartyMember{ID: "hired-1", Name: "Hired Companion", MercenaryType: 0}
	f.Carried = append(append([]mapload.PartyMember(nil), generated...), second)
	f.Town.Won(10)
	f.Town.Won(20)
	f.arriveInTown()
	f.Town.gold = 1000
	f.addChapterCompanions(f.Town.Chapter())
	wantRoster := mapload.CloneParty(f.Carried)

	exports := func(t *testing.T) {
		t.Helper()
		snapshot, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if raw, err := f.ExportNativeCitySave(snapshot, label); err != nil || len(raw) == 0 {
			t.Fatal("bare current party must write SAV", len(raw), err)
		}
	}
	saveAndReload := func(t *testing.T) *FrontEnd {
		t.Helper()
		dir := t.TempDir()
		save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
		name, err := save(false)
		if err != nil {
			t.Fatalf("save(false): %v", err)
		}
		if !IsOriginal(name) {
			t.Fatalf("bare current party wrote %q, want SAV", name)
		}
		restored := releaseFront(t)
		_, _, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{Dir: dir}, nil)
		if open, town, err := load(name); err != nil || !town || open != nil {
			t.Fatalf("load SAV town = opener %v town %v err %v", open != nil, town, err)
		}
		return restored
	}

	exports(t)
	beforeDismissal := saveAndReload(t)
	if got := len(beforeDismissal.Carried); got != len(wantRoster) {
		t.Fatalf("restored roster before dismissal = %d, want %d members", got, len(wantRoster))
	}
	for i, member := range beforeDismissal.Carried {
		if member.ID != wantRoster[i].ID || member.Name != wantRoster[i].Name || member.Hero != wantRoster[i].Hero || member.StartingHero != wantRoster[i].StartingHero {
			t.Fatal("restored roster changed current member", i)
		}
	}

	f.Carried = generated // the player dismissed the second member

	exports(t)
	afterDismissal := saveAndReload(t)
	if got := len(afterDismissal.Carried); got != 1 || afterDismissal.Carried[0].Name != generated[0].Name || !afterDismissal.Carried[0].StartingHero {
		t.Fatalf("restored roster after dismissal = %+v, want one hero named %q", afterDismissal.Carried, generated[0].Name)
	}
}

// TestReleaseNativeTownSaveFighterWornSlotTwelveRoundTrips closes review
// pass1's R-2: nativeCityAttachItems used to write armour slot n at
// unit.Equipment[n] instead of unit.Equipment[n-1], which for a slot-12
// piece hung it off unit.Equipment[12] — the character's own Diary
// reference, not a thirteenth armour slot (program.go's Humanoid programme:
// twelve "Worn" refs then one "Diary" ref, read back as one 13-long run and
// split only by position). A male fighter is armed with a slot-12 piece by
// chargen itself (Choices {0,0,3}: male, fighter) — the shape the story's
// own mage-only fixture (Choices {1,1,3}) never exercised, since her three
// pieces land in slots 1, 7 and 8. This witness proves that piece is still
// worn, at the same slot, after a real native-SAV round trip, and that the
// document itself never grows a Diary from it.
func TestReleaseNativeTownSaveFighterWornSlotTwelveRoundTrips(t *testing.T) {
	f := releaseFront(t)
	result := ui.ChargenResult{Name: "Slot Twelve Fighter", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	if len(generated) != 1 {
		t.Fatalf("ChargenParty length = %d, want 1", len(generated))
	}
	hero := generated[0]
	if hero.Mage {
		t.Fatal("fixture assumption broke: class choice 0 must chargen a fighter")
	}
	if hero.Worn[11] == 0 {
		t.Fatal("fixture assumption broke: chargen must arm a male fighter's own slot-12 armour piece")
	}
	f.Carried = generated
	f.arriveInTown()
	if f.originalCity != nil {
		t.Fatal("a campaign started in Againrom must never bind an original city baseline")
	}

	// The same pre-existing, out-of-scope session-state alignment
	// TestReleaseNativeTownSaveRealPartyEmitsNativeSAVAndRoundTrips uses:
	// chapter 30 declares Mercenaries [14] and AddHero [22], so a freshly
	// arrived Town mismatches (R-2's own predecessor) and would refuse for a
	// session-state reason this witness is not about.
	f.Town.mercEnabled[14] = true
	wantChapter := f.Town.Chapter()
	f.addChapterCompanions(wantChapter)
	f.Town.gold = 1000

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	bytes, err := f.ExportNativeCitySave(snapshot, label)
	if err != nil {
		t.Fatalf("ExportNativeCitySave must write this equipped fighter: %v", err)
	}

	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q as .ags; this fighter must emit a native .sav", name)
	}

	restored := releaseFront(t)
	_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := restoredList()
	if len(entries) != 1 {
		t.Fatalf("restored save list = %+v, want exactly one entry", entries)
	}
	open, town, err := load(entries[0].Name)
	if err != nil || !town || open != nil {
		t.Fatalf("load native town .sav = opener %v town %v err %v", open != nil, town, err)
	}
	var found *mapload.PartyMember
	for i := range restored.Carried {
		if restored.Carried[i].StartingHero {
			found = &restored.Carried[i]
		}
	}
	if found == nil {
		t.Fatalf("restored roster = %+v, want the starting hero", restored.Carried)
	}
	if found.Worn != hero.Worn {
		t.Fatalf("restored Worn = %#v, want %#v (slot 12 must round-trip as armour, not vanish into a diary)", found.Worn, hero.Worn)
	}

	// Document-level check: HasDiary is not exposed through the gameplay
	// restore above at all (sav.Character, not mapload.PartyMember), so this
	// is the only place a Diary wrongly minted from the slot-12 piece would
	// be visible.
	openFile, err := sav.Open(bytes)
	if err != nil {
		t.Fatalf("sav.Open on the generator's own output: %v", err)
	}
	prov, err := openFile.CityProvenance()
	if err != nil {
		t.Fatalf("CityProvenance: %v", err)
	}
	source, err := prov.SourceParty()
	if err != nil {
		t.Fatalf("SourceParty: %v", err)
	}
	var heroChar *sav.Character
	for i := range source {
		if source[i].Name == hero.Name {
			heroChar = &source[i]
		}
	}
	if heroChar == nil {
		t.Fatalf("SourceParty() = %+v, want a character named %q", source, hero.Name)
	}
	if heroChar.HasDiary {
		t.Fatal("SourceParty() hero has a Diary; the slot-12 armour piece must not hang off the Diary reference")
	}
	if got := len(heroChar.Worn); got != 3 {
		t.Fatalf("SourceParty() hero has %d Worn pieces, want 3 (weapon + two armour, slot 12 included): %+v", got, heroChar.Worn)
	}
}

func TestReleaseNativeTownSaveRealPartyEmitsNativeSAVAndRoundTrips(t *testing.T) {
	f := releaseFront(t)
	result := ui.ChargenResult{Name: "Equipped Hero", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	if len(generated) != 1 {
		t.Fatalf("ChargenParty length = %d, want 1", len(generated))
	}
	hero := generated[0]
	if hero.Worn[0] == 0 {
		t.Fatal("fixture assumption broke: chargen must arm a starting weapon")
	}
	if len(hero.Carried) == 0 {
		t.Fatal("fixture assumption broke: chargen must pack the campaign-documents item")
	}
	if hero.KnownSpells == 0 {
		t.Fatal("fixture assumption broke: a chargen'd mage must start with a known spell")
	}
	f.Carried = generated
	f.arriveInTown()
	if f.originalCity != nil {
		t.Fatal("a campaign started in Againrom must never bind an original city baseline")
	}

	f.Town.mercEnabled[14] = true

	wantChapter := f.Town.Chapter()
	f.addChapterCompanions(wantChapter)
	var companion mapload.PartyMember
	for _, member := range f.Carried {
		if member.CompanionNPC == 22 {
			companion = member
		}
	}
	if companion.CompanionNPC != 22 {
		t.Fatal("fixture assumption broke: addChapterCompanions(wantChapter) granted no npc-22 companion")
	}
	if companion.Hero.Body == 0 || companion.Hero.Mind == 0 {
		t.Fatalf("fixture assumption broke: the granted companion carries a zero-stat Hero: %+v", companion.Hero)
	}

	const purse = 24680
	f.Town.gold = purse
	wantRosterLen := len(f.Carried)

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	bytes, err := f.ExportNativeCitySave(snapshot, label)
	if err != nil {
		t.Fatalf("ExportNativeCitySave must now write a real chargen'd party's equipment, inventory and spellbook: %v", err)
	}
	if len(bytes) == 0 {
		t.Fatal("ExportNativeCitySave returned no bytes")
	}

	// Production path: SaveSeams must choose this SAME native writer, not the
	// .ags fallback.
	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q as .ags; a real chargen'd party must now emit a native .sav", name)
	}

	// WriteOriginal always publishes into store.Dir (savestore.go), and the
	// ordinary player picks a freshly-made native save back up through the
	// load window's own list, which is the one place a local original save
	// receives its "local-sav:" load-window token (resume.go,
	// localOriginalSaveToken) so the load seam's own dispatch reads it back
	// through SaveStore rather than treating it as a row from the read-only
	// install directory. Reloading through list() here is that same
	// production dispatch, not a test-only shortcut.
	restored := releaseFront(t)
	_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := restoredList()
	if len(entries) != 1 {
		t.Fatalf("restored save list = %+v, want exactly one entry", entries)
	}
	open, town, err := load(entries[0].Name)
	if err != nil || !town || open != nil {
		t.Fatalf("load native town .sav = opener %v town %v err %v", open != nil, town, err)
	}
	if got := len(restored.Carried); got != wantRosterLen {
		t.Fatalf("restored roster length = %d, want %d: %+v", got, wantRosterLen, restored.Carried)
	}
	var found, foundCompanion *mapload.PartyMember
	for i := range restored.Carried {
		switch {
		case restored.Carried[i].Name == hero.Name && restored.Carried[i].StartingHero:
			found = &restored.Carried[i]
		case restored.Carried[i].CompanionNPC == 22:
			foundCompanion = &restored.Carried[i]
		}
	}
	if found == nil {
		t.Fatalf("restored roster = %+v, want the starting hero named %q", restored.Carried, hero.Name)
	}
	// Hero identity: name, sex/class (FigureDir/Mage), portrait (FigureFace)
	// and the class-derived Profile all round-trip through nativeCityDefRow's
	// data.ChargenBase binding (DIV-887, amended), not a template substituted
	// for the player's own choice.
	if found.FigureDir != hero.FigureDir || found.Mage != hero.Mage || found.FigureFace != hero.FigureFace || found.Profile != hero.Profile {
		t.Fatalf("restored hero identity = %+v, want %+v", found, hero)
	}
	if found.Worn != hero.Worn {
		t.Fatalf("restored Worn = %#v, want %#v", found.Worn, hero.Worn)
	}
	if len(found.Carried) != len(hero.Carried) {
		t.Fatalf("restored Carried = %#v, want %#v", found.Carried, hero.Carried)
	}
	for i, code := range hero.Carried {
		if found.Carried[i] != code {
			t.Fatalf("restored Carried = %v, want %v", found.Carried, hero.Carried)
		}
	}
	if found.KnownSpells != hero.KnownSpells {
		t.Fatalf("restored KnownSpells = %#x, want %#x", found.KnownSpells, hero.KnownSpells)
	}
	if found.Hero != hero.Hero {
		t.Fatalf("restored hero stats (Body/Mind/Skill) = %+v, want %+v", found.Hero, hero.Hero)
	}
	if foundCompanion == nil {
		t.Fatalf("restored roster = %+v, want the chapter companion (npc 22)", restored.Carried)
	}
	if foundCompanion.Name != companion.Name {
		t.Fatalf("restored companion name = %q, want %q", foundCompanion.Name, companion.Name)
	}
	if foundCompanion.FigureDir != companion.FigureDir || foundCompanion.Mage != companion.Mage || foundCompanion.FigureFace != companion.FigureFace || foundCompanion.Profile != companion.Profile {
		t.Fatalf("restored companion identity = %+v, want %+v", foundCompanion, companion)
	}
	if foundCompanion.Hero != companion.Hero {
		t.Fatalf("restored companion stats (Body/Mind/Skill) = %+v, want %+v", foundCompanion.Hero, companion.Hero)
	}
	if restored.Town.Gold() != purse {
		t.Fatalf("restored gold = %d, want %d", restored.Town.Gold(), purse)
	}
	if restored.Town.Chapter() != wantChapter {
		t.Fatalf("restored chapter = %d, want %d", restored.Town.Chapter(), wantChapter)
	}

	// Reverse conversion: the same document, read back through the
	// import-continuity reader ordinary LOAD uses for a real original city
	// save, resolves the identical roster shape rather than a shape that
	// only this generator's own writer can interpret.
	openFile, err := sav.Open(bytes)
	if err != nil {
		t.Fatalf("sav.Open on the generator's own output: %v", err)
	}
	prov, err := openFile.CityProvenance()
	if err != nil {
		t.Fatalf("CityProvenance on the generator's own output: %v", err)
	}
	roster := prov.Roster()
	if len(roster) != wantRosterLen {
		t.Fatalf("SourceParty roster length = %d, want %d", len(roster), wantRosterLen)
	}
	source, err := prov.SourceParty()
	if err != nil {
		t.Fatalf("SourceParty: %v", err)
	}
	if len(source) != wantRosterLen {
		t.Fatalf("SourceParty() = %d characters, want %d", len(source), wantRosterLen)
	}
	var heroChar *sav.Character
	for i := range source {
		if source[i].Name == hero.Name {
			heroChar = &source[i]
		}
	}
	if heroChar == nil {
		t.Fatalf("SourceParty() = %+v, want a character named %q", source, hero.Name)
	}
	if got := len(heroChar.Worn); got == 0 {
		t.Fatal("SourceParty() hero carries no Worn pieces; the equipped party's own weapon/armour did not round-trip")
	}
	if heroChar.KnownSpells() != hero.KnownSpells {
		t.Fatalf("SourceParty() hero KnownSpells = %#x, want %#x", heroChar.KnownSpells(), hero.KnownSpells)
	}
}

// DIV-890
func TestReleaseNativeTownSaveHiredMercenaryEmitsNativeSAVAndRoundTrips(t *testing.T) {
	f := releaseFront(t)
	result := ui.ChargenResult{Name: "Tavern Hero", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	if len(generated) != 1 {
		t.Fatalf("ChargenParty length = %d, want 1", len(generated))
	}
	hero := generated[0]
	f.Carried = generated
	f.arriveInTown()
	if f.originalCity != nil {
		t.Fatal("a campaign started in Againrom must never bind an original city baseline")
	}

	// Chapter 30 is this campaign's own first town chapter (Campaign.TownBegins:
	// missions 10/20 are prologue, offered by no building) and declares
	// exactly one tavern type, 14.
	f.Town.gold = 5_000_000
	wantChapter := f.Town.Chapter()
	if declared := f.Campaign.Value().Chapters[wantChapter].Mercenaries; len(declared) != 1 || declared[0] != 14 {
		t.Fatalf("fixture assumption broke: chapter %d declares Mercenaries %v, want [14]", wantChapter, declared)
	}

	f.addChapterCompanions(wantChapter)

	// Hire a real squad through the production tavern action
	// (townScreen.toggleMercenary), the same call the live UI makes, rather
	// than a hand-built PartyMember.
	screen, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("fixture assumption broke: TownScreen is not *townScreen")
	}
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 14 refused")
	}
	var wantSquad []mapload.PartyMember
	for _, member := range f.Carried {
		if member.MercenaryType == 14 {
			wantSquad = append(wantSquad, member)
		}
	}
	if len(wantSquad) == 0 {
		t.Fatal("fixture assumption broke: toggleMercenary(14) hired no one")
	}
	if !f.Town.mercHired[14] {
		t.Fatal("fixture assumption broke: toggleMercenary(14) left mercHired[14] false")
	}
	const purse = 13579
	f.Town.gold = purse
	wantRosterLen := len(f.Carried)

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ExportNativeCitySave must now write a hired mercenary's campaign-record state: %v", err)
	}

	// Production path: SaveSeams must choose this SAME native writer, not
	// the .ags fallback, exactly as it does with no one hired.
	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q as .ags; a hired mercenary must now emit a native .sav", name)
	}
	restored := releaseFront(t)
	_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := restoredList()
	if len(entries) != 1 {
		t.Fatalf("restored save list = %+v, want exactly one entry", entries)
	}
	open, town, err := load(entries[0].Name)
	if err != nil || !town || open != nil {
		t.Fatalf("load native town .sav = opener %v town %v err %v", open != nil, town, err)
	}
	if got := len(restored.Carried); got != wantRosterLen {
		t.Fatalf("restored roster length = %d, want %d: %+v", got, wantRosterLen, restored.Carried)
	}
	var gotSquad []mapload.PartyMember
	foundHero := false
	for _, member := range restored.Carried {
		if member.MercenaryType == 14 {
			gotSquad = append(gotSquad, member)
		}
		if member.Name == hero.Name && member.StartingHero {
			foundHero = true
		}
	}
	if !foundHero {
		t.Fatalf("restored roster = %+v, want the starting hero named %q", restored.Carried, hero.Name)
	}
	if len(gotSquad) != len(wantSquad) {
		t.Fatalf("restored roster carries %d type-14 mercenaries, want %d: %+v", len(gotSquad), len(wantSquad), restored.Carried)
	}
	for i, want := range wantSquad {
		got := gotSquad[i]
		if got.Name != want.Name || got.Class != want.Class || got.Profile != want.Profile ||
			got.FigureDir != want.FigureDir || got.FigureFace != want.FigureFace || got.Hero != want.Hero {
			t.Fatalf("restored hired member[%d] = %+v, want %+v", i, got, want)
		}
		if got.Worn != want.Worn {
			t.Fatalf("restored hired member[%d] Worn = %v, want %v", i, got.Worn, want.Worn)
		}
		if len(got.Carried) != len(want.Carried) {
			t.Fatalf("restored hired member[%d] Carried = %v, want %v", i, got.Carried, want.Carried)
		}
		for k, code := range want.Carried {
			if got.Carried[k] != code {
				t.Fatalf("restored hired member[%d] Carried = %v, want %v", i, got.Carried, want.Carried)
			}
		}
		if got.KnownSpells != want.KnownSpells {
			t.Fatalf("restored hired member[%d] KnownSpells = %#x, want %#x", i, got.KnownSpells, want.KnownSpells)
		}
	}
	if !restored.Town.mercHired[14] {
		t.Fatal("restored Town.mercHired[14] = false, want true")
	}
	if got := restored.Town.MercenaryPool(14); got != len(wantSquad) {
		t.Fatalf("restored Town.MercenaryPool(14) = %d, want the saved working count %d", got, len(wantSquad))
	}
	if restored.Town.Gold() != purse {
		t.Fatalf("restored gold = %d, want %d", restored.Town.Gold(), purse)
	}
	if restored.Town.Chapter() != wantChapter {
		t.Fatalf("restored chapter = %d, want %d", restored.Town.Chapter(), wantChapter)
	}

	bytes, err := f.ExportNativeCitySave(snapshot, label)
	if err != nil {
		t.Fatalf("re-running ExportNativeCitySave for the reverse-conversion check: %v", err)
	}
	openFile, err := sav.Open(bytes)
	if err != nil {
		t.Fatalf("sav.Open on the generator's own output: %v", err)
	}
	prov, err := openFile.CityProvenance()
	if err != nil {
		t.Fatalf("CityProvenance on the generator's own output: %v", err)
	}
	source, err := prov.SourceParty()
	if err != nil {
		t.Fatalf("SourceParty: %v", err)
	}
	if len(source) != wantRosterLen {
		t.Fatalf("SourceParty() = %d characters, want %d (the whole roster, hired squad included)", len(source), wantRosterLen)
	}

	// HERO-SKILLGATE-074
	identitiesByType := make(map[uint8][]sav.Character)
	for _, character := range source {
		typ := uint8(character.DisplayBacking)
		if typ != 0 {
			if character.Name != "" {
				t.Fatalf("hired actor has explicit technical name %q", character.Name)
			}
			identitiesByType[typ] = append(identitiesByType[typ], character)
		}
	}
	for _, want := range wantSquad {
		queue := identitiesByType[want.MercenaryType]
		if len(queue) == 0 {
			t.Fatalf("SourceParty has no unclaimed hire type %d", want.MercenaryType)
		}
		identity := queue[0].Key
		identitiesByType[want.MercenaryType] = queue[1:]
		human, err := prov.Human(identity)
		if err != nil {
			t.Fatalf("CityProvenance.Human(%#x) for %q: %v", identity, want.Name, err)
		}
		if wantWord := uint16(want.Class); human.TypeID != wantWord {
			t.Fatalf("written typeWord for hired %q = %#04x, want %#04x (member.Class, SAV-616's own confirmed hire value for type 14)", want.Name, human.TypeID, wantWord)
		}
	}
}

// Dismissal must not derive pool arrays from party membership (MERC-POOL-011/012, SAV-1084/1085).
func TestReleaseNativeCitySAVDismissalPreservesLoadedWorkingPool(t *testing.T) {
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Pool Persistence Hero")
	f.Town.gold = 5_000_000

	const hireType = 14
	// Arrays say 3; persisted party carries only 2 of the hired type.
	f.Town.mercEnabled[hireType] = true
	f.Town.mercPool[hireType] = 3
	f.Town.mercCapacity[hireType] = 3
	if _, ok := screen.toggleMercenary(hireType); !ok {
		t.Fatal("fixture assumption broke: could not hire type 14")
	}
	var hired int
	party := make([]mapload.PartyMember, 0, len(f.Carried)-1)
	for _, member := range f.Carried {
		if member.MercenaryType == hireType {
			hired++
			if hired > 2 {
				continue
			}
		}
		party = append(party, member)
	}
	if hired != 3 {
		t.Fatalf("fixture assumption broke: initial type-%d party count = %d, want 3", hireType, hired)
	}
	f.Carried = mapload.OwnParty(party)
	if got := countTownMercenaries(f.Carried, hireType); got != 2 {
		t.Fatalf("fixture party count = %d, want 2", got)
	}

	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("first city SAVE = %q, err %v", name, err)
	}
	wire, err := store.Read(name)
	if err != nil {
		t.Fatalf("read first city SAV: %v", err)
	}
	file, err := sav.Open(wire)
	if err != nil {
		t.Fatalf("open first city SAV: %v", err)
	}
	campaign, present, err := file.Campaign()
	if err != nil || !present {
		t.Fatalf("first SAV campaign = present %v, err %v", present, err)
	}
	if len(campaign.MercenaryWorking) != 15 || len(campaign.MercenaryPristine) != 15 ||
		len(campaign.MercenaryHired) != 15 || campaign.MercenaryWorking[hireType-1] != 3 ||
		campaign.MercenaryPristine[hireType-1] != 3 || !campaign.MercenaryHired[hireType-1] {
		t.Fatalf("first SAV pool = working %v pristine %v hired %v, want 3/3/true", campaign.MercenaryWorking, campaign.MercenaryPristine, campaign.MercenaryHired)
	}

	loaded := releaseFront(t)
	_, rows, load := agsSaveSeams(loaded, store, OriginalStore{}, nil)
	entries := rows()
	if len(entries) != 1 {
		t.Fatalf("first reload save rows = %+v, want one local SAV", entries)
	}
	open, inTown, err := load(entries[0].Name)
	if err != nil || !inTown || open != nil {
		t.Fatalf("load hired city SAV = opener %v town %v err %v", open != nil, inTown, err)
	}
	if loaded.Town.MercenaryPool(hireType) != 3 || loaded.Town.MercenaryCapacity(hireType) != 3 ||
		!loaded.Town.MercenaryHired(hireType) || countTownMercenaries(loaded.Carried, hireType) != 2 {
		t.Fatalf("loaded state = working %d pristine %d hired %v party %d, want 3/3/true/2",
			loaded.Town.MercenaryPool(hireType), loaded.Town.MercenaryCapacity(hireType),
			loaded.Town.MercenaryHired(hireType), countTownMercenaries(loaded.Carried, hireType))
	}
	loadedScreen, ok := loaded.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("loaded town screen is not *townScreen")
	}
	if _, ok := loadedScreen.toggleMercenary(hireType); !ok {
		t.Fatal("dismissal of loaded hired type was refused")
	}
	if loaded.Town.MercenaryPool(hireType) != 3 || loaded.Town.MercenaryCapacity(hireType) != 3 ||
		loaded.Town.MercenaryHired(hireType) || countTownMercenaries(loaded.Carried, hireType) != 0 {
		t.Fatalf("dismissed state = working %d pristine %d hired %v party %d, want 3/3/false/0",
			loaded.Town.MercenaryPool(hireType), loaded.Town.MercenaryCapacity(hireType),
			loaded.Town.MercenaryHired(hireType), countTownMercenaries(loaded.Carried, hireType))
	}

	saveAgain, _, _ := agsSaveSeams(loaded, store, OriginalStore{}, nil)
	name, err = saveAgain(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("city SAVE after dismissal = %q, err %v", name, err)
	}
	wire, err = store.Read(name)
	if err != nil {
		t.Fatalf("read dismissal city SAV: %v", err)
	}
	file, err = sav.Open(wire)
	if err != nil {
		t.Fatalf("open dismissal city SAV: %v", err)
	}
	campaign, present, err = file.Campaign()
	if err != nil || !present {
		t.Fatalf("dismissal SAV campaign = present %v, err %v", present, err)
	}
	if campaign.MercenaryWorking[hireType-1] != 3 || campaign.MercenaryPristine[hireType-1] != 3 ||
		campaign.MercenaryHired[hireType-1] {
		t.Fatalf("dismissal SAV pool = working %d pristine %d hired %v, want 3/3/false",
			campaign.MercenaryWorking[hireType-1], campaign.MercenaryPristine[hireType-1],
			campaign.MercenaryHired[hireType-1])
	}
}

func countTownMercenaries(party []mapload.PartyMember, typ int) int {
	count := 0
	for _, member := range party {
		if int(member.MercenaryType) == typ {
			count++
		}
	}
	return count
}

// TestReleaseNativeTownSaveHiredMercenaryFlagOnlyFileStillLoads is the
// brief's own third required witness: a file written before this story — a
// set campaign-record hire flag and pool cell, no persisted actor record for
// that type — must still load. restoreHiredMercenaries (nativecityrestore.go)
// is now a per-type fallback, live only when no member of that type already
// reached f.Carried some other way. No writer this story ships can still
// produce that exact pre-story document shape (nativeCityData now writes a
// real record whenever live Party carries a Human-type hire), so this
// witness drives the fallback directly the way a decoded legacy document
// would: Town.mercHired/mercPool set by hand, with f.Carried carrying no
// member of that type, is precisely what campaignProgressFromSAV would leave
// behind decoding one.
func TestReleaseNativeTownSaveHiredMercenaryFlagOnlyFileStillLoads(t *testing.T) {
	f := releaseFront(t)
	hero, _ := reachabilityWalkArrive(t, f, "Legacy Flag Hero")
	f.addChapterCompanions(f.Town.Chapter())

	const hireType = 14
	const wantCount = 2
	for _, member := range f.Carried {
		if member.MercenaryType == hireType {
			t.Fatalf("fixture assumption broke: %+v already carries a type-%d member before the fallback runs", f.Carried, hireType)
		}
	}
	f.Town.mercHired[hireType] = true
	f.Town.mercPool[hireType] = wantCount
	wantRosterLen := len(f.Carried) + wantCount

	if err := f.CampaignSession.restoreHiredMercenaries(f.Table); err != nil {
		t.Fatalf("restoreHiredMercenaries: %v", err)
	}
	if got := len(f.Carried); got != wantRosterLen {
		t.Fatalf("roster length after the fallback = %d, want %d", got, wantRosterLen)
	}
	gotCount := 0
	foundHero := false
	for _, member := range f.Carried {
		if member.MercenaryType == hireType {
			gotCount++
		}
		if member.Name == hero.Name && member.StartingHero {
			foundHero = true
		}
	}
	if gotCount != wantCount {
		t.Fatalf("fallback rebuilt %d type-%d mercenaries, want %d", gotCount, hireType, wantCount)
	}
	if !foundHero {
		t.Fatalf("roster = %+v, want the starting hero named %q untouched", f.Carried, hero.Name)
	}
	if f.Town.MercenaryPool(hireType) != wantCount {
		t.Fatalf("Town.MercenaryPool(%d) = %d, want the saved working count %d after rebuilding the party", hireType, f.Town.MercenaryPool(hireType), wantCount)
	}
	if !f.Town.mercHired[hireType] {
		t.Fatal("Town.mercHired lost the hire flag")
	}
}

func TestReleaseNativeTownSaveShopBaredSessionRoundTripsAsSAV(t *testing.T) {
	f := releaseFront(t)
	result := ui.ChargenResult{Name: "Session Hero", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	if len(generated) != 1 {
		t.Fatalf("ChargenParty length = %d, want 1", len(generated))
	}
	if generated[0].Mage {
		t.Fatal("fixture assumption broke: class choice 0 must chargen a fighter; a mage cannot reach the shop-bare shape (review pass1, reachability)")
	}
	f.Carried = generated

	// R-2/R-3: win enough main missions that the tavern's accumulated
	// mercenary-unlock history and Town.won both grow past whatever a
	// reload would derive from the chapter reached.
	for _, mission := range []int{10, 20, 30, 40} {
		f.Town.Won(mission)
	}
	f.arriveInTown()
	if f.originalCity != nil {
		t.Fatal("a campaign started in Againrom must never bind an original city baseline")
	}

	// R-4: chapter 30's own AddHero grant, exactly as the real win-flow
	// grants it on first arrival (frontend.go:2220), then dismissed by the
	// player — the shape RestoreOriginal's town arm would otherwise regrant.
	if grants := f.Campaign.Value().Chapters[30].AddHero; len(grants) == 0 || grants[0] != 22 {
		t.Fatalf("fixture assumption broke: chapter 30 AddHero = %v, want [22]", grants)
	}
	f.addChapterCompanions(30)
	kept, dismissed := make([]mapload.PartyMember, 0, len(f.Carried)), false
	for _, member := range f.Carried {
		if member.CompanionNPC == 22 {
			dismissed = true
			continue
		}
		kept = append(kept, member)
	}
	if !dismissed {
		t.Fatal("fixture assumption broke: addChapterCompanions(30) did not grant npc 22")
	}
	f.Carried = kept

	// R-3: a marker history entry, a bound quick spell the dismissed caster
	// left behind, and an outstanding offered mission — all session state
	// with no document field at all.
	screen := f.TownScreen().(*townScreen)
	marked := 0
	for _, mission := range []int{50, 60, 70, 100, 110, 30, 40} {
		screen.markWorldSelected(mission)
		if screen.worldSelectedOnce[mission] {
			marked = mission
			break
		}
	}
	if marked == 0 {
		t.Fatal("fixture assumption broke: no candidate mission recorded a world-map marker on this root")
	}
	f.quickSpells = [4]uint32{1, 0, 5, 0}
	f.Offered = 999
	f.Town.gold = 500

	// Drive the review's own reachable route: unequip every occupied doll
	// slot to the table, move the pack to the table, sell.
	screen.room = roomShop
	for slot := 1; slot <= sim.EquipSlots; slot++ {
		if screen.shopPartyMember(screen.shopMemberIndex()).Worn[slot-1] != 0 {
			if act := screen.shopUnequipToTable(slot); act.Msg == "" {
				t.Fatalf("shopUnequipToTable(%d) returned no message", slot)
			}
		}
	}
	if len(screen.shopPackStacks()) > 0 {
		screen.shopFromPack(0, true)
	}
	if act := screen.shopSell(); act.Msg == "" {
		t.Fatal("shopSell returned no message")
	}
	// The state at the moment of SAVE is what the SAV producer must
	// reproduce exactly.
	wantHero := f.Carried[0]
	wantMercEnabled := f.Town.mercEnabled
	wantWon := make(map[int]bool, len(f.Town.won))
	for k, v := range f.Town.won {
		wantWon[k] = v
	}
	wantSelected := make(map[int]bool, len(screen.worldSelectedOnce))
	for k, v := range screen.worldSelectedOnce {
		wantSelected[k] = v
	}
	wantQuickSpells := f.quickSpells
	wantOffered := f.Offered
	wantGold := f.Town.Gold()
	wantChapter := f.Town.Chapter()

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ExportNativeCitySave refused current session state: %v", err)
	}

	dir := t.TempDir()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q, want the sole SAV format", name)
	}

	restored := releaseFront(t)
	_, _, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{Dir: dir}, nil)
	if open, town, err := load(name); err != nil || !town || open != nil {
		t.Fatalf("load SAV town save = opener %v town %v err %v", open != nil, town, err)
	}

	var restoredHero *mapload.PartyMember
	for i := range restored.Carried {
		if restored.Carried[i].StartingHero {
			restoredHero = &restored.Carried[i]
		}
		if restored.Carried[i].CompanionNPC == 22 {
			t.Fatalf("restored roster = %+v, want the dismissed chapter companion to stay dismissed", restored.Carried)
		}
	}
	if restoredHero == nil {
		t.Fatalf("restored roster = %+v, want the starting hero", restored.Carried)
	}
	if restoredHero.Name != wantHero.Name || restoredHero.FigureDir != wantHero.FigureDir ||
		restoredHero.Mage != wantHero.Mage || restoredHero.Body != wantHero.Body ||
		restoredHero.BodyDir != wantHero.BodyDir || restoredHero.Class != wantHero.Class ||
		restoredHero.FigureFace != wantHero.FigureFace || restoredHero.Profile != wantHero.Profile {
		t.Fatalf("restored hero identity = %+v, want %+v", restoredHero, wantHero)
	}
	if restoredHero.Hero != wantHero.Hero {
		t.Fatalf("restored hero stats = %+v, want %+v", restoredHero.Hero, wantHero.Hero)
	}
	if restored.Town.mercEnabled != wantMercEnabled {
		t.Fatalf("restored mercenary-enabled set = %v, want %v", restored.Town.mercEnabled, wantMercEnabled)
	}
	if len(restored.Town.won) != len(wantWon) {
		t.Fatalf("restored Town.won = %v, want %v", restored.Town.won, wantWon)
	}
	for k, v := range wantWon {
		if restored.Town.won[k] != v {
			t.Fatalf("restored Town.won = %v, want %v", restored.Town.won, wantWon)
		}
	}
	restoredScreen := restored.TownScreen().(*townScreen)
	if len(restoredScreen.worldSelectedOnce) != len(wantSelected) {
		t.Fatalf("restored worldSelectedOnce = %v, want %v", restoredScreen.worldSelectedOnce, wantSelected)
	}
	for k, v := range wantSelected {
		if restoredScreen.worldSelectedOnce[k] != v {
			t.Fatalf("restored worldSelectedOnce = %v, want %v", restoredScreen.worldSelectedOnce, wantSelected)
		}
	}
	if restored.quickSpells != wantQuickSpells {
		t.Fatalf("restored quickSpells = %v, want %v", restored.quickSpells, wantQuickSpells)
	}
	if restored.Offered != wantOffered {
		t.Fatalf("restored Offered = %d, want %d", restored.Offered, wantOffered)
	}
	if restored.Town.Gold() != wantGold {
		t.Fatalf("restored gold = %d, want %d", restored.Town.Gold(), wantGold)
	}
	if restored.Town.Chapter() != wantChapter {
		t.Fatalf("restored chapter = %d, want %d", restored.Town.Chapter(), wantChapter)
	}
}

func reachabilityWalkStates() []int {
	return []int{10, 20, 30, 40, 41, 50, 60, 70, 71, 80, 90, 100, 110, 111, 120, 121, 130, 140}
}

// reachabilityWalkArrive builds one chargen'd hero, arrives in town and
// returns the party plus the live *townScreen the walk drives markWorldSelected
// through — the same construction every other witness in this file uses,
// factored out so the two walk witnesses below do not duplicate it.
func reachabilityWalkArrive(t *testing.T, f *FrontEnd, name string) (mapload.PartyMember, *townScreen) {
	t.Helper()
	result := ui.ChargenResult{Name: name, Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	if len(generated) != 1 {
		t.Fatalf("ChargenParty length = %d, want 1", len(generated))
	}
	f.Carried = generated
	f.arriveInTown()
	if f.originalCity != nil {
		t.Fatal("a campaign started in Againrom must never bind an original city baseline")
	}
	screen, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("fixture assumption broke: TownScreen is not *townScreen")
	}
	return generated[0], screen
}

// DIV-884
func TestReleaseNativeTownSaveReachabilityWalkAllReachableStatesWriteNative(t *testing.T) {
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Walk Hero")
	f.Town.gold = 5_000_000
	states := reachabilityWalkStates()
	var refusals []string
	wrote := 0
	for _, m := range states {
		screen.markWorldSelected(m)
		if _, accepted := f.Town.Won(m); !accepted {
			t.Fatalf("Town.Won(%d) was not accepted; the walk itself no longer matches the campaign it was measured against", m)
		}
		f.addChapterCompanions(f.Town.Chapter())
		snapshot, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatalf("Snapshot after Won(%d): %v", m, err)
		}
		if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
			refusals = append(refusals, fmt.Sprintf("Won(%d) at chapter %d: write refused: %v", m, f.Town.Chapter(), err))
			continue
		}

		dir := t.TempDir()
		store := SaveStore{Dir: dir}
		save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
		name, err := save(false)
		if err != nil {
			t.Fatalf("Won(%d): save(false): %v", m, err)
		}
		if !IsOriginal(name) {
			t.Fatalf("Won(%d): save(false) published %q as .ags, not a native .sav, right after ExportNativeCitySave itself succeeded", m, name)
		}
		loader := releaseFront(t)
		_, loaderList, load := agsSaveSeams(loader, SaveStore{Dir: dir}, OriginalStore{}, nil)
		entries := loaderList()
		if len(entries) != 1 {
			t.Fatalf("Won(%d): save list = %+v, want exactly one entry", m, entries)
		}
		if open, town, err := load(entries[0].Name); err != nil || !town || open != nil {
			refusals = append(refusals, fmt.Sprintf("Won(%d) at chapter %d: wrote a native SAV that failed to load: opener %v town %v err %v", m, f.Town.Chapter(), open != nil, town, err))
			continue
		}
		wrote++
	}
	screen.markWorldSelected(150)
	if _, accepted := f.Town.Won(150); !accepted {
		t.Fatal("Town.Won(150) was not accepted")
	}
	completed, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.ExportNativeCitySave(completed, "Completed"); err == nil {
		t.Fatal("the completed campaign was written as a native SAV")
	}
	t.Logf("native SAV reachability walk: %d/%d states write and load a native SAV", wrote, len(states))
	if len(refusals) != 0 || wrote != len(states) {
		t.Fatalf("%d/%d states refuse or fail to load a native SAV:\n%s",
			len(refusals), len(states), strings.Join(refusals, "\n"))
	}
}

// TestReleaseNativeTownSaveReachabilityWalkThreeCheckpointsRoundTrip is the
// brief's required second witness: a full round trip (write, fresh-process
// load, reverse conversion, field-by-field comparison), at three of the
// walk's 18 states, one of which carries a hired mercenary. Chapter 30 is
// the walk's own first reachable town chapter and declares tavern type 14
// (TestReleaseNativeTownSaveHiredMercenaryEmitsNativeSAVAndRoundTrips below
// proves the hire mechanics in isolation); hiring it once, before the walk's
// first Won call, and never returning it, proves the squad survives not
// only one save/load cycle but every further completion in the same
// session — Town.Won alone never runs mercenaryBoundary's mission-end merge
// (frontend.go:2146), so nothing culls or re-tallies it along the way,
// exactly like an ordinary player who hires early and keeps the squad for
// the rest of the campaign.
func TestReleaseNativeTownSaveReachabilityWalkThreeCheckpointsRoundTrip(t *testing.T) {
	f := releaseFront(t)
	hero, screen := reachabilityWalkArrive(t, f, "Checkpoint Hero")
	f.Town.gold = 5_000_000

	if declared := f.Campaign.Value().Chapters[30].Mercenaries; len(declared) != 1 || declared[0] != 14 {
		t.Fatalf("fixture assumption broke: chapter 30 declares Mercenaries %v, want [14]", declared)
	}
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 14 refused")
	}
	if !f.Town.mercHired[14] {
		t.Fatal("fixture assumption broke: toggleMercenary(14) left mercHired[14] false")
	}
	var wantMercCount int
	for _, member := range f.Carried {
		if member.MercenaryType == 14 {
			wantMercCount++
		}
	}
	if wantMercCount == 0 {
		t.Fatal("fixture assumption broke: toggleMercenary(14) hired no one")
	}

	states := reachabilityWalkStates()
	// These checkpoints span early, middle and late campaign towns; the full
	// reachability witness above covers every intermediate state.
	checkpoints := map[int]bool{30: true, 90: true, 140: true}
	checked := 0
	for _, m := range states {
		screen.markWorldSelected(m)
		if _, accepted := f.Town.Won(m); !accepted {
			t.Fatalf("Town.Won(%d) was not accepted", m)
		}
		f.addChapterCompanions(f.Town.Chapter())
		if !checkpoints[m] {
			continue
		}
		checked++
		f.Town.gold = 10000 + m
		wantGold, wantChapter := f.Town.Gold(), f.Town.Chapter()
		wantRosterLen := len(f.Carried)
		wantSelected := append([]int(nil), screen.worldSelectedOnceMissions()...)

		snapshot, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatalf("checkpoint Won(%d): Snapshot: %v", m, err)
		}
		bytes, nativeErr := f.ExportNativeCitySave(snapshot, label)
		if nativeErr != nil {
			t.Fatalf("checkpoint Won(%d): ExportNativeCitySave: %v", m, nativeErr)
		}

		dir := t.TempDir()
		store := SaveStore{Dir: dir}
		save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
		name, err := save(false)
		if err != nil {
			t.Fatalf("checkpoint Won(%d): save(false): %v", m, err)
		}
		if !IsOriginal(name) {
			t.Fatalf("checkpoint Won(%d): save(false) published %q as .ags, want a native .sav", m, name)
		}
		restored := releaseFront(t)
		_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
		entries := restoredList()
		if len(entries) != 1 {
			t.Fatalf("checkpoint Won(%d): restored save list = %+v, want exactly one entry", m, entries)
		}
		open, town, err := load(entries[0].Name)
		if err != nil || !town || open != nil {
			t.Fatalf("checkpoint Won(%d): load town save = opener %v town %v err %v", m, open != nil, town, err)
		}
		if got := len(restored.Carried); got != wantRosterLen {
			t.Fatalf("checkpoint Won(%d): restored roster length = %d, want %d", m, got, wantRosterLen)
		}
		foundHero, gotMercCount := false, 0
		for _, member := range restored.Carried {
			if member.Name == hero.Name && member.StartingHero {
				foundHero = true
			}
			if member.MercenaryType == 14 {
				gotMercCount++
			}
		}
		if !foundHero {
			t.Fatalf("checkpoint Won(%d): restored roster = %+v, want the starting hero named %q", m, restored.Carried, hero.Name)
		}
		if gotMercCount != wantMercCount {
			t.Fatalf("checkpoint Won(%d): restored roster carries %d type-14 mercenaries, want %d", m, gotMercCount, wantMercCount)
		}
		if !restored.Town.mercHired[14] {
			t.Fatalf("checkpoint Won(%d): restored Town.mercHired[14] = false, want true", m)
		}
		if restored.Town.Gold() != wantGold {
			t.Fatalf("checkpoint Won(%d): restored gold = %d, want %d", m, restored.Town.Gold(), wantGold)
		}
		if restored.Town.Chapter() != wantChapter {
			t.Fatalf("checkpoint Won(%d): restored chapter = %d, want %d", m, restored.Town.Chapter(), wantChapter)
		}
		// SAV-609: the picture pointer rehydrates at world-map entry, not at
		// LOAD, so the UI-visited cache (townScreen.worldSelectedOnce) is
		// still empty right after a native load — Town.selectedMarkerMissions()
		// is the persisted latch enterWorldMap's own pre-existing
		// reconstruction call seeds that cache from, and what this story's
		// write/read changes actually round-trip.
		if got := restored.Town.selectedMarkerMissions(); !equalIntSets(got, wantSelected) {
			t.Fatalf("checkpoint Won(%d): restored marker history = %v, want %v", m, got, wantSelected)
		}

		// Reverse conversion: the import-continuity reader a real original
		// city save uses resolves the same roster shape from this writer's
		// own output, not a shape only this generator's writer understands.
		openFile, err := sav.Open(bytes)
		if err != nil {
			t.Fatalf("checkpoint Won(%d): sav.Open on the generator's own output: %v", m, err)
		}
		prov, err := openFile.CityProvenance()
		if err != nil {
			t.Fatalf("checkpoint Won(%d): CityProvenance: %v", m, err)
		}
		source, err := prov.SourceParty()
		if err != nil {
			t.Fatalf("checkpoint Won(%d): SourceParty: %v", m, err)
		}
		if len(source) != wantRosterLen {
			t.Fatalf("checkpoint Won(%d): SourceParty() = %d characters, want %d", m, len(source), wantRosterLen)
		}
	}
	if checked != len(checkpoints) {
		t.Fatalf("checked %d checkpoints, want %d: the walk's own states no longer include every configured checkpoint", checked, len(checkpoints))
	}
}

// worldSelectedOnceMissions reports a stable, sorted view of a townScreen's
// once-selected mission set for a witness's own comparison; the live field
// is a map, whose iteration order carries no meaning (AGENTS.md: no map
// order in hashed or compared state).
func (t *townScreen) worldSelectedOnceMissions() []int {
	out := make([]int, 0, len(t.worldSelectedOnce))
	for m := range t.worldSelectedOnce {
		out = append(out, m)
	}
	sortInts(out)
	return out
}

func equalIntSets(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalTownOffers(a, b []TownOffer) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func partyTypeAndIDSequence(party []mapload.PartyMember) ([]int, []string) {
	types := make([]int, len(party))
	ids := make([]string, len(party))
	for i, member := range party {
		types[i] = int(member.MercenaryType)
		ids[i] = member.ID
	}
	return types, ids
}

func TestReleaseNativeTownSaveTavernOffersSurviveReload(t *testing.T) {
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Tavern Hero")
	f.Town.gold = 5_000_000
	states := reachabilityWalkStates()
	checked := 0
	acceptedChecked := false
	for _, m := range states {
		screen.markWorldSelected(m)
		if _, accepted := f.Town.Won(m); !accepted {
			t.Fatalf("Town.Won(%d) was not accepted", m)
		}
		f.addChapterCompanions(f.Town.Chapter())

		// R-1's own accepted-mission reproduction: chapter 100's shipped Inn
		// offers [100 101] and 101 has no own [Mission101] section; taking it
		// live must still show it as accepted after a native SAVE and reload.
		acceptedCase := f.Town.Chapter() == 100
		if acceptedCase {
			if mission, ok := f.Town.Take(TownTavern, 1); !ok || mission != 101 {
				t.Fatalf("fixture assumption broke: Take(TownTavern, 1) at chapter 100 = (%d, %v), want (101, true)", mission, ok)
			}
			if got := f.Town.Available(); !equalIntSets(got, []int{101}) {
				t.Fatalf("fixture assumption broke: live Town.Available() after Take = %v, want [101]", got)
			}
		}

		wantTavern := f.Town.Offers(TownTavern)
		wantSchool := f.Town.Offers(TownSchool)
		wantShop := f.Town.Offers(TownShop)

		snapshot, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatalf("Won(%d): Snapshot: %v", m, err)
		}
		if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
			t.Fatalf("Won(%d): ExportNativeCitySave refused: %v", m, err)
		}

		dir := t.TempDir()
		save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
		name, err := save(false)
		if err != nil {
			t.Fatalf("Won(%d): save(false): %v", m, err)
		}
		if !IsOriginal(name) {
			t.Fatalf("Won(%d): save(false) published %q as .ags, want a native .sav", m, name)
		}
		restored := releaseFront(t)
		_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
		entries := restoredList()
		if len(entries) != 1 {
			t.Fatalf("Won(%d): save list = %+v, want exactly one entry", m, entries)
		}
		if open, town, err := load(entries[0].Name); err != nil || !town || open != nil {
			t.Fatalf("Won(%d): load native town .sav = opener %v town %v err %v", m, open != nil, town, err)
		}
		checked++
		if got := restored.Town.Offers(TownTavern); !equalTownOffers(got, wantTavern) {
			t.Fatalf("Won(%d): restored tavern offers = %+v, want %+v", m, got, wantTavern)
		}
		if got := restored.Town.Offers(TownSchool); !equalTownOffers(got, wantSchool) {
			t.Fatalf("Won(%d): restored school offers = %+v, want %+v", m, got, wantSchool)
		}
		if got := restored.Town.Offers(TownShop); !equalTownOffers(got, wantShop) {
			t.Fatalf("Won(%d): restored shop offers = %+v, want %+v", m, got, wantShop)
		}
		if acceptedCase {
			acceptedChecked = true
			available := restored.Town.Available()
			if !equalIntSets(available, []int{101}) {
				t.Fatalf("Won(%d): restored Town.Available() = %v, want exactly accepted mission101", m, available)
			}
		}
	}
	if checked != len(states) {
		t.Fatalf("checked %d states, want %d", checked, len(states))
	}
	if !acceptedChecked {
		t.Fatal("fixture assumption broke: chapter 100's own accepted-mission case never ran")
	}
}

// TestReleaseNativeTownSaveStrippedHeroBesideHiredMercenaryRoundTrips sells a
// hero's equipment with a hired squad in the party and saves the town. The
// shop's picker stands on player characters alone (TOWN-138), so the sale is
// the hero's own: the SAV keeps the sale and the purse, and the hired member
// comes back as it was hired.
func TestReleaseNativeTownSaveStrippedHeroBesideHiredMercenaryRoundTrips(t *testing.T) {
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Stripped Hero")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 5_000_000
	wantChapter := f.Town.Chapter()
	if declared := f.Campaign.Value().Chapters[wantChapter].Mercenaries; len(declared) == 0 || declared[0] != 14 {
		t.Fatalf("fixture assumption broke: chapter %d declares Mercenaries %v, want a list starting 14", wantChapter, declared)
	}
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 14 refused")
	}
	hiredIndex := -1
	for i, member := range f.Carried {
		if member.MercenaryType == 14 {
			hiredIndex = i
			break
		}
	}
	if hiredIndex < 0 {
		t.Fatal("fixture assumption broke: toggleMercenary(14) hired no one")
	}
	wantHired := f.Carried[hiredIndex]

	// Strip every occupied doll slot of the member the shop shows to the
	// table, then sell.
	screen.room = roomShop
	heroIndex := screen.shopMemberIndex()
	if f.Carried[heroIndex].Hired() {
		t.Fatalf("the shop shows the hired member %d", heroIndex)
	}
	stripped := 0
	for slot := 1; slot <= sim.EquipSlots; slot++ {
		if !(*screen.shopWornItemSlots(heroIndex))[slot-1].Empty() {
			if act := screen.shopUnequipToTable(slot); act.Msg == "" {
				t.Fatalf("shopUnequipToTable(%d) returned no message", slot)
			}
			stripped++
		}
	}
	if stripped == 0 {
		t.Fatal("fixture assumption broke: the hero wore nothing to strip")
	}
	if act := screen.shopSell(); act.Msg == "" {
		t.Fatal("shopSell returned no message")
	}
	wantPurse := f.Town.Gold()
	wantHero := f.Carried[heroIndex]
	wantRosterLen := len(f.Carried)
	if worn, pack := memberItemCodes(f.Carried[hiredIndex]); fmt.Sprint(worn, pack) != fmt.Sprint(memberItemCodes(wantHired)) {
		t.Fatalf("the sale changed the hired member: worn %v pack %v", worn, pack)
	}

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ExportNativeCitySave refused a sold hero beside a hired member: %v", err)
	}

	dir := t.TempDir()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q as .ags; the sold hero beside a hired member must keep the native .sav", name)
	}
	restored := releaseFront(t)
	_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := restoredList()
	if len(entries) != 1 {
		t.Fatalf("save list = %+v, want exactly one entry", entries)
	}
	if open, town, err := load(entries[0].Name); err != nil || !town || open != nil {
		t.Fatalf("load native town .sav = opener %v town %v err %v", open != nil, town, err)
	}
	if got := len(restored.Carried); got != wantRosterLen {
		t.Fatalf("restored roster length = %d, want %d", got, wantRosterLen)
	}
	// Health: no member of an Againrom-started campaign carries a
	// between-visit wound state (Saved is populated only on restore), so both
	// sides derive it fresh from Hero, Profile and the worn items, which are
	// the fields compared beside it.
	same := func(who string, got, want mapload.PartyMember) {
		t.Helper()
		gotWorn, gotPack := memberItemCodes(got)
		wantWorn, wantPack := memberItemCodes(want)
		if gotWorn != wantWorn || !slices.Equal(gotPack, wantPack) {
			t.Fatalf("restored %s worn %v pack %v, want %v and %v", who, gotWorn, gotPack, wantWorn, wantPack)
		}
		if got.KnownSpells != want.KnownSpells {
			t.Fatalf("restored %s KnownSpells = %#x, want %#x", who, got.KnownSpells, want.KnownSpells)
		}
		_, wantHealth, _ := mapload.PartyDisplayWithTable(want, f.Table)
		_, gotHealth, _ := mapload.PartyDisplayWithTable(got, f.Table)
		if gotHealth != wantHealth {
			t.Fatalf("restored %s derived health = %d, want %d", who, gotHealth, wantHealth)
		}
	}
	heroFound, hiredFound := false, false
	for _, got := range restored.Carried {
		switch {
		case got.MercenaryType == 14 && !hiredFound:
			hiredFound = true
			same("hired member", got, wantHired)
		case !got.Hired() && got.Name == wantHero.Name && !heroFound:
			heroFound = true
			same("sold hero", got, wantHero)
		}
	}
	if !heroFound || !hiredFound {
		t.Fatalf("restored roster = %+v, want the sold hero and the hired type-14 member", restored.Carried)
	}
	if restored.Town.Gold() != wantPurse {
		t.Fatalf("restored gold = %d, want %d: the sale price must stay in the purse", restored.Town.Gold(), wantPurse)
	}
	if restored.Town.Chapter() != wantChapter {
		t.Fatalf("restored chapter = %d, want %d", restored.Town.Chapter(), wantChapter)
	}
}

// hireForOrderTest advances a fresh chargen'd hero to chapter 50 — the
// campaign's own first chapter to declare two tavern types, [14 6 10 13] —
// granting each intervening chapter's own companion exactly as the real
// win-flow does, and returns the live *townScreen. Shared by the R-3
// ascending and descending witnesses below so the two differ only in hire
// order.
func hireForOrderTest(t *testing.T, name string) (*FrontEnd, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, name)
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 5_000_000
	if _, accepted := f.Town.Won(30); !accepted {
		t.Fatal("fixture assumption broke: Town.Won(30) was not accepted")
	}
	f.addChapterCompanions(f.Town.Chapter())
	if _, accepted := f.Town.Won(40); !accepted {
		t.Fatal("fixture assumption broke: Town.Won(40) was not accepted")
	}
	f.addChapterCompanions(f.Town.Chapter())
	if declared := f.Campaign.Value().Chapters[f.Town.Chapter()].Mercenaries; len(declared) < 2 || declared[0] != 14 || declared[1] != 6 {
		t.Fatalf("fixture assumption broke: chapter %d declares Mercenaries %v, want it to start [14 6]", f.Town.Chapter(), declared)
	}
	return f, screen
}

func TestReleaseNativeTownSaveHiredMercenaryAscendingOrderRoundTrips(t *testing.T) {
	f, screen := hireForOrderTest(t, "Ascending Hero")
	wantChapter := f.Town.Chapter()

	// Ascending type order: 6 before 14.
	if _, ok := screen.toggleMercenary(6); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 6 refused")
	}
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 14 refused")
	}
	wantTypes, wantIDs := partyTypeAndIDSequence(f.Carried)
	wantRosterLen := len(f.Carried)

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ExportNativeCitySave must accept an already-ascending hire order: %v", err)
	}

	dir := t.TempDir()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q as .ags; an ascending hire order must emit a native .sav", name)
	}
	restored := releaseFront(t)
	_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := restoredList()
	if len(entries) != 1 {
		t.Fatalf("save list = %+v, want exactly one entry", entries)
	}
	if open, town, err := load(entries[0].Name); err != nil || !town || open != nil {
		t.Fatalf("load native town .sav = opener %v town %v err %v", open != nil, town, err)
	}
	if got := len(restored.Carried); got != wantRosterLen {
		t.Fatalf("restored roster length = %d, want %d", got, wantRosterLen)
	}
	gotTypes, gotIDs := partyTypeAndIDSequence(restored.Carried)
	if !equalIntSets(gotTypes, wantTypes) {
		t.Fatalf("restored party MercenaryType sequence = %v, want %v", gotTypes, wantTypes)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("restored party ID sequence = %v, want %v", gotIDs, wantIDs)
		}
	}
	if restored.Town.Chapter() != wantChapter {
		t.Fatalf("restored chapter = %d, want %d", restored.Town.Chapter(), wantChapter)
	}
}

// TestReleaseNativeTownSaveHiredMercenaryDescendingOrderRoundTrips is
// DIV-907's own inversion (SAV-616/617): hiring two Human-type squads in
// descending type order (14 then 6) used to leave
// nativeCityHireOrderMismatch with no ascending-type rebuild that could
// reproduce it
// (TestReleaseNativeTownSaveHiredMercenaryDescendingOrderRefusesAndAGSFallbackRoundTrips's
// own name, before this story). A Human-type hire's own actor record now
// keeps its own live party position regardless of hire order
// (nativeCityHumanHiredMembers), so the native writer accepts this shape too
// and the reload keeps the true [14 14 14 6 6 6 6]-shaped order, not the
// ascending rebuild's own [6 6 6 6 14 14 14].
func TestReleaseNativeTownSaveHiredMercenaryDescendingOrderRoundTrips(t *testing.T) {
	f, screen := hireForOrderTest(t, "Descending Hero")
	wantChapter := f.Town.Chapter()

	// Descending type order: 14 before 6 (the review's own reproduction).
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 14 refused")
	}
	if _, ok := screen.toggleMercenary(6); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 6 refused")
	}
	wantTypes, wantIDs := partyTypeAndIDSequence(f.Carried)
	wantRosterLen := len(f.Carried)

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ExportNativeCitySave must now accept a descending Human-type hire order: %v", err)
	}

	dir := t.TempDir()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q as .ags; a descending Human-type hire order must now keep the native .sav", name)
	}
	restored := releaseFront(t)
	_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := restoredList()
	if len(entries) != 1 {
		t.Fatalf("save list = %+v, want exactly one entry", entries)
	}
	if open, town, err := load(entries[0].Name); err != nil || !town || open != nil {
		t.Fatalf("load native town .sav = opener %v town %v err %v", open != nil, town, err)
	}
	if got := len(restored.Carried); got != wantRosterLen {
		t.Fatalf("restored roster length = %d, want %d", got, wantRosterLen)
	}
	gotTypes, gotIDs := partyTypeAndIDSequence(restored.Carried)
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("restored party MercenaryType sequence = %v, want %v", gotTypes, wantTypes)
	}
	for i := range wantTypes {
		if gotTypes[i] != wantTypes[i] {
			t.Fatalf("restored party MercenaryType sequence = %v, want %v: the true descending hire order must survive its own persisted records", gotTypes, wantTypes)
		}
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("restored party ID sequence = %v, want %v", gotIDs, wantIDs)
		}
	}
	if restored.Town.Chapter() != wantChapter {
		t.Fatalf("restored chapter = %d, want %d", restored.Town.Chapter(), wantChapter)
	}
}

// hireForSiegeOrderTest arrives a fresh chargen'd hero at the campaign's own
// first town visit (chapter 30), the earliest chapter the tavern offers both
// a siege type (Catapult, type 1, offered from chapter 10) and every
// Human type (offered from chapter 30 on) in the same visit — reachable in
// ordinary play, not a fixture-only shape. Gold is loaded so both hires
// succeed regardless of price.
func hireForSiegeOrderTest(t *testing.T, name string) (*FrontEnd, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, name)
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 5_000_000
	return f, screen
}

func TestReleaseNativeTownSaveHiredMercenarySiegeThenHumanOrderRoundTrips(t *testing.T) {
	f, screen := hireForSiegeOrderTest(t, "Siege Then Human")

	// Siege first, then Human — the shape master wrote natively and the
	// first landed candidate refused.
	if _, ok := screen.toggleMercenary(1); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for siege type 1 refused")
	}
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 14 refused")
	}
	wantTypes, wantIDs := partyTypeAndIDSequence(f.Carried)
	wantRosterLen := len(f.Carried)

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ExportNativeCitySave must accept siege-then-Human order (master wrote it natively): %v", err)
	}

	dir := t.TempDir()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q as .ags; siege-then-Human order must emit a native .sav, as master did", name)
	}
	restored := releaseFront(t)
	_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := restoredList()
	if len(entries) != 1 {
		t.Fatalf("save list = %+v, want exactly one entry", entries)
	}
	if open, town, err := load(entries[0].Name); err != nil || !town || open != nil {
		t.Fatalf("load native town .sav = opener %v town %v err %v", open != nil, town, err)
	}
	if got := len(restored.Carried); got != wantRosterLen {
		t.Fatalf("restored roster length = %d, want %d", got, wantRosterLen)
	}
	gotTypes, gotIDs := partyTypeAndIDSequence(restored.Carried)
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("restored party MercenaryType sequence = %v, want %v", gotTypes, wantTypes)
	}
	for i := range wantTypes {
		if gotTypes[i] != wantTypes[i] {
			t.Fatalf("restored party MercenaryType sequence = %v, want %v: siege-then-Human order must survive the reload", gotTypes, wantTypes)
		}
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("restored party ID sequence = %v, want %v", gotIDs, wantIDs)
		}
	}
}

// A current Human-then-siege roster is written by the same SAV producer as
// every other town state. The cold reload must retain its exact member order.
func TestReleaseNativeTownSaveHiredMercenaryHumanThenSiegeOrderPreserved(t *testing.T) {
	f, screen := hireForSiegeOrderTest(t, "Human Then Siege")

	// Human first, then siege — the direction this correction trades away,
	// matching master's own pre-1129 behavior for it.
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 14 refused")
	}
	if _, ok := screen.toggleMercenary(1); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for siege type 1 refused")
	}
	wantTypes, wantIDs := partyTypeAndIDSequence(f.Carried)
	wantRosterLen := len(f.Carried)

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := f.ExportNativeCitySave(snapshot, label); err != nil {
		t.Fatalf("ExportNativeCitySave refused Human-then-siege order: %v", err)
	}

	dir := t.TempDir()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q, want the sole SAV format", name)
	}
	restored := releaseFront(t)
	_, _, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{Dir: dir}, nil)
	if _, town, err := load(name); err != nil || !town {
		t.Fatalf("load SAV = town %v err %v", town, err)
	}
	if got := len(restored.Carried); got != wantRosterLen {
		t.Fatalf("restored roster length = %d, want %d", got, wantRosterLen)
	}
	gotTypes, gotIDs := partyTypeAndIDSequence(restored.Carried)
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("restored party MercenaryType sequence = %v, want %v", gotTypes, wantTypes)
	}
	for i := range wantTypes {
		if gotTypes[i] != wantTypes[i] {
			t.Fatalf("restored party MercenaryType sequence = %v, want %v: SAV must preserve true hire order", gotTypes, wantTypes)
		}
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("restored party ID sequence = %v, want %v", gotIDs, wantIDs)
		}
	}
}

func TestReleaseNativeTownSaveHiredMercenaryCombatBlockRoundTrips(t *testing.T) {
	f, screen := hireForSiegeOrderTest(t, "Combat Block Hero")
	if _, ok := screen.toggleMercenary(10); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 10 refused")
	}
	var wantSquad []mapload.PartyMember
	for _, member := range f.Carried {
		if member.MercenaryType == 10 {
			wantSquad = append(wantSquad, member)
		}
	}
	if len(wantSquad) == 0 {
		t.Fatal("fixture assumption broke: toggleMercenary(10) hired no one")
	}

	dir := t.TempDir()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatalf("save(false): %v", err)
	}
	if !IsOriginal(name) {
		t.Fatalf("save(false) published %q as .ags; a solo type-10 hire must still emit a native .sav", name)
	}
	restored := releaseFront(t)
	_, restoredList, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := restoredList()
	if len(entries) != 1 {
		t.Fatalf("save list = %+v, want exactly one entry", entries)
	}
	if _, town, err := load(entries[0].Name); err != nil || !town {
		t.Fatalf("load native .sav = town %v err %v", town, err)
	}
	var gotSquad []mapload.PartyMember
	for _, member := range restored.Carried {
		if member.MercenaryType == 10 {
			gotSquad = append(gotSquad, member)
		}
	}
	if len(gotSquad) != len(wantSquad) {
		t.Fatalf("restored roster carries %d type-10 mercenaries, want %d", len(gotSquad), len(wantSquad))
	}
	for i := range wantSquad {
		wantDerived, wantHealth, wantMana := mapload.PartyDisplayWithTable(wantSquad[i], f.Table)
		gotDerived, gotHealth, gotMana := mapload.PartyDisplayWithTable(gotSquad[i], restored.Table)
		if wantDerived != gotDerived || wantHealth != gotHealth || wantMana != gotMana {
			t.Errorf("mercenary %d combat block differs across reload\n live %+v hp=%d mana=%d\n got  %+v hp=%d mana=%d",
				i, wantDerived, wantHealth, wantMana, gotDerived, gotHealth, gotMana)
		}
	}
}

// sourcePartyOf decodes bytes — a native SAV writer's own output —
// through the same reverse-conversion reader a real original city save uses
// and returns its resolved roster.
func sourcePartyOf(t *testing.T, saveBytes []byte) []sav.Character {
	t.Helper()
	openFile, err := sav.Open(saveBytes)
	if err != nil {
		t.Fatalf("sav.Open: %v", err)
	}
	prov, err := openFile.CityProvenance()
	if err != nil {
		t.Fatalf("CityProvenance: %v", err)
	}
	source, err := prov.SourceParty()
	if err != nil {
		t.Fatalf("SourceParty: %v", err)
	}
	return source
}

// heroCharacterOf finds the one sav.Character whose own Hero field names it
// the player's starting character (party.go's own doc comment: exactly one
// on every corpus save this project has read) so a witness compares that one
// character across two decoded documents by identity, not by document
// position — SAV-GRPORD-058: the actor list carries no meaningful order.
func heroCharacterOf(t *testing.T, source []sav.Character) sav.Character {
	t.Helper()
	var found []sav.Character
	for _, c := range source {
		if c.Hero {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		t.Fatalf("SourceParty() carries %d Hero-flagged characters, want exactly 1: %+v", len(found), source)
	}
	return found[0]
}

// TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeWithNothingChanged
// is the chain's own baseline: no mutation between the two SAVEs.
func TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeWithNothingChanged(t *testing.T) {
	f := releaseFront(t)
	hero, _ := reachabilityWalkArrive(t, f, "Reload Hero Bare")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 24680
	wantChapter := f.Town.Chapter()
	wantRosterLen := len(f.Carried)

	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save1, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name1, err := save1(false)
	if err != nil {
		t.Fatalf("checkpoint one save(false): %v", err)
	}
	if !IsOriginal(name1) {
		t.Fatalf("checkpoint one save(false) published %q as .ags, want native", name1)
	}
	bytes1, err := store.Read(name1)
	if err != nil {
		t.Fatalf("reading checkpoint one's own published bytes: %v", err)
	}

	reloaded := releaseFront(t)
	save2, list2, load2 := agsSaveSeams(reloaded, store, OriginalStore{}, nil)
	entries := list2()
	if len(entries) != 1 {
		t.Fatalf("checkpoint one's save list = %+v, want exactly one entry", entries)
	}
	open, town, err := load2(entries[0].Name)
	if err != nil || !town || open != nil {
		t.Fatalf("loading checkpoint one = opener %v town %v err %v", open != nil, town, err)
	}
	if reloaded.originalCity == nil || reloaded.originalCity.unavailable != nil {
		t.Fatalf("RestoreOriginal must bind a usable imported-city baseline: %+v", reloaded.originalCity)
	}

	// Nothing changes between the two SAVEs: this shape's own point.
	name2, err := save2(false)
	if err != nil {
		t.Fatalf("checkpoint two save(false): %v", err)
	}
	if !IsOriginal(name2) {
		t.Fatalf("checkpoint two save(false) published %q as .ags, want a second native .sav", name2)
	}
	bytes2, err := store.Read(name2)
	if err != nil {
		t.Fatalf("reading checkpoint two's own published bytes: %v", err)
	}

	prov1 := sourcePartyOf(t, bytes1)
	prov2 := sourcePartyOf(t, bytes2)
	if len(prov1) != wantRosterLen || len(prov2) != wantRosterLen {
		t.Fatalf("SourceParty() = %d/%d characters at checkpoints one/two, want %d/%d (the hero plus the chapter companion, no hired squad)", len(prov1), len(prov2), wantRosterLen, wantRosterLen)
	}
	hero1, hero2 := heroCharacterOf(t, prov1), heroCharacterOf(t, prov2)
	if hero1.Name != hero2.Name {
		t.Fatalf("checkpoint two's own hero = %+v, want checkpoint one's %+v", hero2, hero1)
	}

	twiceReloaded := releaseFront(t)
	_, list3, load3 := agsSaveSeams(twiceReloaded, store, OriginalStore{}, nil)
	// list3 reports the UI-facing opaque token (localOriginalSavePrefix +
	// bare name, resume.go), not the bare store name save()/store.Read()
	// use — the same distinction load's own doc comment names.
	wantToken := localOriginalSavePrefix + name2
	loadName := ""
	for _, e := range list3() {
		if e.Name == wantToken {
			loadName = e.Name
		}
	}
	if loadName == "" {
		t.Fatalf("checkpoint two token %q not found in save list %+v", wantToken, list3())
	}
	open3, town3, err := load3(loadName)
	if err != nil || !town3 || open3 != nil {
		t.Fatalf("loading checkpoint two = opener %v town %v err %v", open3 != nil, town3, err)
	}
	if got := len(twiceReloaded.Carried); got != wantRosterLen {
		t.Fatalf("checkpoint two roster length = %d, want %d", got, wantRosterLen)
	}
	foundHero := false
	for _, member := range twiceReloaded.Carried {
		if member.Name == hero.Name && member.StartingHero {
			foundHero = true
		}
	}
	if !foundHero {
		t.Fatalf("checkpoint two roster = %+v, want the starting hero named %q", twiceReloaded.Carried, hero.Name)
	}
	if twiceReloaded.Town.Gold() != reloaded.Town.Gold() {
		t.Fatalf("checkpoint two gold = %d, want checkpoint one's %d", twiceReloaded.Town.Gold(), reloaded.Town.Gold())
	}
	if twiceReloaded.Town.Chapter() != wantChapter {
		t.Fatalf("checkpoint two chapter = %d, want %d", twiceReloaded.Town.Chapter(), wantChapter)
	}
}

func TestReleaseRestoreOriginalRejectsMissingOrRepeatedCurrentPartyIdentity(t *testing.T) {
	f := currentTown(t, nil, nil)
	raw := currentTownSave(t, f)
	if len(f.Carried) < 2 {
		t.Fatalf("fixture has %d carried members, want at least two", len(f.Carried))
	}
	beforeIDs := make([]string, len(f.Carried))
	for i, member := range f.Carried {
		beforeIDs[i] = member.ID
	}
	beforeChapter := f.Town.Chapter()

	for _, tc := range []struct {
		name string
		edit func(*currentActionData)
	}{
		{"absent current identity", func(a *currentActionData) { a.Party[0].ID = nil }},
		{"repeated current identity", func(a *currentActionData) { a.Party[1].ID = append([]byte(nil), a.Party[0].ID...) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil || len(a.Party) < 2 {
				t.Fatalf("fixture lacks a populated current party: action=%v err=%v", a != nil, err)
			}
			tc.edit(a)
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			bad, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := f.RestoreOriginal(bad); err == nil {
				t.Fatal("RestoreOriginal accepted an absent or repeated current-party identity")
			}
			if len(f.Carried) != len(beforeIDs) || f.Town.Chapter() != beforeChapter {
				t.Fatal("rejected current-party identity changed the live city")
			}
			for i, id := range beforeIDs {
				if f.Carried[i].ID != id {
					t.Fatalf("rejected current-party identity changed member %d ID from %q to %q", i, id, f.Carried[i].ID)
				}
			}
		})
	}
}

func TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeWithHiredSquad(t *testing.T) {
	f := releaseFront(t)
	hero, screen := reachabilityWalkArrive(t, f, "Reload Hero Hired")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 24680
	wantChapter := f.Town.Chapter()

	declared := f.Campaign.Value().Chapters[wantChapter].Mercenaries
	if len(declared) == 0 {
		t.Fatalf("fixture assumption broke: chapter %d declares no Mercenaries", wantChapter)
	}
	hireType := declared[0]
	if _, ok := screen.toggleMercenary(hireType); !ok {
		t.Fatalf("fixture assumption broke: production tavern hire for type %d refused", hireType)
	}
	var wantSquad []mapload.PartyMember
	for _, member := range f.Carried {
		if member.MercenaryType == uint8(hireType) {
			wantSquad = append(wantSquad, member)
		}
	}
	if len(wantSquad) == 0 {
		t.Fatal("fixture assumption broke: toggleMercenary hired no one")
	}
	wantRosterLen := len(f.Carried)

	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save1, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name1, err := save1(false)
	if err != nil {
		t.Fatalf("checkpoint one save(false): %v", err)
	}
	if !IsOriginal(name1) {
		t.Fatalf("checkpoint one save(false) published %q as .ags, want native", name1)
	}
	bytes1, err := store.Read(name1)
	if err != nil {
		t.Fatalf("reading checkpoint one's own published bytes: %v", err)
	}

	reloaded := releaseFront(t)
	save2, list2, load2 := agsSaveSeams(reloaded, store, OriginalStore{}, nil)
	entries := list2()
	if len(entries) != 1 {
		t.Fatalf("checkpoint one's save list = %+v, want exactly one entry", entries)
	}
	open, town, err := load2(entries[0].Name)
	if err != nil || !town || open != nil {
		t.Fatalf("loading checkpoint one = opener %v town %v err %v", open != nil, town, err)
	}
	if reloaded.originalCity == nil || reloaded.originalCity.unavailable != nil {
		t.Fatalf("RestoreOriginal must bind a usable imported-city baseline: %+v", reloaded.originalCity)
	}
	if got := len(reloaded.Carried); got != wantRosterLen {
		t.Fatalf("checkpoint one reload roster length = %d, want %d: the hired squad must be restored, from its own persisted record or (siege only) restoreHiredMercenaries's rebuild", got, wantRosterLen)
	}
	if !reloaded.Town.mercHired[hireType] {
		t.Fatal("checkpoint one reload Town.mercHired lost the hire")
	}

	// Nothing further changes: the hired squad was already on board before
	// checkpoint one's own SAVE.
	name2, err := save2(false)
	if err != nil {
		t.Fatalf("checkpoint two save(false): %v", err)
	}
	if !IsOriginal(name2) {
		t.Fatalf("checkpoint two save(false) published %q as .ags, want a second native .sav with the hired squad still on board", name2)
	}
	bytes2, err := store.Read(name2)
	if err != nil {
		t.Fatalf("reading checkpoint two's own published bytes: %v", err)
	}

	prov1 := sourcePartyOf(t, bytes1)
	prov2 := sourcePartyOf(t, bytes2)
	want := wantRosterLen
	if hireType <= 2 {
		want -= len(wantSquad)
	}
	if len(prov1) != want || len(prov2) != want {
		t.Fatalf("SourceParty() = %d/%d characters at checkpoints one/two, want %d/%d", len(prov1), len(prov2), want, want)
	}
	hero1, hero2 := heroCharacterOf(t, prov1), heroCharacterOf(t, prov2)
	if hero1.Name != hero2.Name {
		t.Fatalf("checkpoint two's own hero = %+v, want checkpoint one's %+v", hero2, hero1)
	}

	twiceReloaded := releaseFront(t)
	_, list3, load3 := agsSaveSeams(twiceReloaded, store, OriginalStore{}, nil)
	// list3 reports the UI-facing opaque token (localOriginalSavePrefix +
	// bare name, resume.go), not the bare store name save()/store.Read()
	// use — the same distinction load's own doc comment names.
	wantToken := localOriginalSavePrefix + name2
	loadName := ""
	for _, e := range list3() {
		if e.Name == wantToken {
			loadName = e.Name
		}
	}
	if loadName == "" {
		t.Fatalf("checkpoint two token %q not found in save list %+v", wantToken, list3())
	}
	open3, town3, err := load3(loadName)
	if err != nil || !town3 || open3 != nil {
		t.Fatalf("loading checkpoint two = opener %v town %v err %v", open3 != nil, town3, err)
	}
	if got := len(twiceReloaded.Carried); got != wantRosterLen {
		t.Fatalf("checkpoint two roster length = %d, want %d", got, wantRosterLen)
	}
	foundHero, gotSquad := false, 0
	for _, member := range twiceReloaded.Carried {
		if member.Name == hero.Name && member.StartingHero {
			foundHero = true
		}
		if member.MercenaryType == uint8(hireType) {
			gotSquad++
		}
	}
	if !foundHero {
		t.Fatalf("checkpoint two roster = %+v, want the starting hero named %q", twiceReloaded.Carried, hero.Name)
	}
	if gotSquad != len(wantSquad) {
		t.Fatalf("checkpoint two roster carries %d type-%d mercenaries, want %d", gotSquad, hireType, len(wantSquad))
	}
	if !twiceReloaded.Town.mercHired[hireType] {
		t.Fatal("checkpoint two Town.mercHired lost the hire")
	}
}

func TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeAfterWorldMapOpened(t *testing.T) {
	f := releaseFront(t)
	hero, screen := reachabilityWalkArrive(t, f, "Reload Hero WorldMap")
	for _, m := range []int{10, 20, 30, 40} {
		screen.markWorldSelected(m)
		if _, accepted := f.Town.Won(m); !accepted {
			t.Fatalf("Town.Won(%d) was not accepted", m)
		}
		f.addChapterCompanions(f.Town.Chapter())
	}
	screen.markWorldSelected(50)
	wantSelected := screen.worldSelectedOnceMissions()
	if len(wantSelected) == 0 {
		t.Fatal("fixture assumption broke: markWorldSelected(50) left worldSelectedOnce empty")
	}
	f.Town.gold = 24680
	wantChapter := f.Town.Chapter()
	wantRosterLen := len(f.Carried)

	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save1, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name1, err := save1(false)
	if err != nil {
		t.Fatalf("checkpoint one save(false): %v", err)
	}
	if !IsOriginal(name1) {
		t.Fatalf("checkpoint one save(false) published %q as .ags, want native", name1)
	}
	bytes1, err := store.Read(name1)
	if err != nil {
		t.Fatalf("reading checkpoint one's own published bytes: %v", err)
	}

	reloaded := releaseFront(t)
	save2, list2, _ := agsSaveSeams(reloaded, store, OriginalStore{}, nil)
	if entries := list2(); len(entries) != 1 {
		t.Fatalf("checkpoint one's save list = %+v, want exactly one entry", entries)
	}
	openLocalTownSAV(t, reloaded, dir, name1)
	if reloaded.originalCity == nil || reloaded.originalCity.unavailable != nil {
		t.Fatalf("RestoreOriginal must bind a usable imported-city baseline: %+v", reloaded.originalCity)
	}
	reloadedScreen, ok := reloaded.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("fixture assumption broke: reloaded TownScreen is not *townScreen")
	}
	if got := reloadedScreen.worldSelectedOnceMissions(); !equalIntSets(got, wantSelected) {
		t.Fatalf("LOAD restored worldSelectedOnce = %v, want the saved markers %v", got, wantSelected)
	}
	lossy := releaseFront(t)
	lossDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(lossDir, "altered.sav"), alterSAV(t, bytes1, func(doc *sav.DocumentData) bool {
		n := len(doc.Campaign.Markers)
		doc.Campaign.Markers = slices.DeleteFunc(doc.Campaign.Markers, func(m sav.CityCampaignMarkerData) bool { return m.Value == 50 })
		return len(doc.Campaign.Markers) != n
	}), 0o600); err != nil {
		t.Fatal(err)
	}
	openLocalTownSAV(t, lossy, lossDir, "altered.sav")
	if got := lossy.TownScreen().(*townScreen).worldSelectedOnceMissions(); slices.Contains(got, 50) {
		t.Fatalf("loss control: a SAV without marker 50 loaded worldSelectedOnce %v", got)
	}
	reloadedScreen.enterWorldMap()
	if got := reloadedScreen.worldSelectedOnceMissions(); !equalIntSets(got, wantSelected) {
		t.Fatalf("after enterWorldMap: worldSelectedOnce = %v, want %v", got, wantSelected)
	}

	name2, err := save2(false)
	if err != nil {
		t.Fatalf("checkpoint two save(false): %v", err)
	}
	if !IsOriginal(name2) {
		t.Fatalf("checkpoint two save(false) published %q as .ags, want a second native .sav after opening the world map", name2)
	}
	bytes2, err := store.Read(name2)
	if err != nil {
		t.Fatalf("reading checkpoint two's own published bytes: %v", err)
	}

	prov1 := sourcePartyOf(t, bytes1)
	prov2 := sourcePartyOf(t, bytes2)
	if len(prov1) != wantRosterLen || len(prov2) != wantRosterLen {
		t.Fatalf("SourceParty() = %d/%d characters at checkpoints one/two, want %d/%d (the hero plus every chapter companion granted so far)", len(prov1), len(prov2), wantRosterLen, wantRosterLen)
	}
	hero1, hero2 := heroCharacterOf(t, prov1), heroCharacterOf(t, prov2)
	if hero1.Name != hero2.Name {
		t.Fatalf("checkpoint two's own hero = %+v, want checkpoint one's %+v", hero2, hero1)
	}

	twiceReloaded := releaseFront(t)
	_, list3, load3 := agsSaveSeams(twiceReloaded, store, OriginalStore{}, nil)
	// list3 reports the UI-facing opaque token (localOriginalSavePrefix +
	// bare name, resume.go), not the bare store name save()/store.Read()
	// use — the same distinction load's own doc comment names.
	wantToken := localOriginalSavePrefix + name2
	loadName := ""
	for _, e := range list3() {
		if e.Name == wantToken {
			loadName = e.Name
		}
	}
	if loadName == "" {
		t.Fatalf("checkpoint two token %q not found in save list %+v", wantToken, list3())
	}
	open3, town3, err := load3(loadName)
	if err != nil || !town3 || open3 != nil {
		t.Fatalf("loading checkpoint two = opener %v town %v err %v", open3 != nil, town3, err)
	}
	if got := len(twiceReloaded.Carried); got != wantRosterLen {
		t.Fatalf("checkpoint two roster length = %d, want %d", got, wantRosterLen)
	}
	foundHero := false
	for _, member := range twiceReloaded.Carried {
		if member.Name == hero.Name && member.StartingHero {
			foundHero = true
		}
	}
	if !foundHero {
		t.Fatalf("checkpoint two roster = %+v, want the starting hero named %q", twiceReloaded.Carried, hero.Name)
	}
	if twiceReloaded.Town.Chapter() != wantChapter {
		t.Fatalf("checkpoint two chapter = %d, want %d", twiceReloaded.Town.Chapter(), wantChapter)
	}
	if got := twiceReloaded.Town.selectedMarkerMissions(); !equalIntSets(got, wantSelected) {
		t.Fatalf("checkpoint two marker history = %v, want %v", got, wantSelected)
	}
}

// TestReleaseNativeTownSaveReachabilityWalkSecondSaveAllStayNative is the
// required correction's own item 1, first clause: the reviewer's 18-state
// walk, extended by one LOAD and one SAVE with nothing further changed.
// Measured on the returned candidate: 13 of the walk's 18 writing states —
// every one whose campaign record carried a world-map marker, Won(50)
// through Won(140) — hard-failed checkpoint two outright. Mechanism:
// captureSession's own persisted-history baseline (correct since this
// story's first pass) was compared, inside marshal, against townScreen's
// lazy load-emptied UI cache (SAV-609) instead of the same projection, and
// the .ags fallback's own anti-forgery gate then rejected that
// now-legitimately-nonzero baseline as forged (originalCityFromSnapshot).
// On master (baacc771, before this story) the second SAVE simply runs
// ExportNativeCitySave fresh, exactly as the first one did, so it is native
// in every state. This witness requires every checkpoint two to remain a
// native .sav, not merely a file of some kind.
func TestReleaseNativeTownSaveReachabilityWalkSecondSaveAllStayNative(t *testing.T) {
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Walk Second Save All Native")
	f.Town.gold = 5_000_000
	states := reachabilityWalkStates()
	var refusals []string
	native := 0
	for _, m := range states {
		screen.markWorldSelected(m)
		if _, accepted := f.Town.Won(m); !accepted {
			t.Fatalf("Town.Won(%d) was not accepted; the walk no longer matches the campaign it was measured against", m)
		}
		f.addChapterCompanions(f.Town.Chapter())

		dir := t.TempDir()
		store := SaveStore{Dir: dir}
		save1, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
		name1, err := save1(false)
		if err != nil || !IsOriginal(name1) {
			refusals = append(refusals, fmt.Sprintf("Won(%d): checkpoint one = %q err %v, not a native .sav", m, name1, err))
			continue
		}

		reloaded := releaseFront(t)
		save2, list2, load2 := agsSaveSeams(reloaded, store, OriginalStore{}, nil)
		entries := list2()
		if len(entries) != 1 {
			t.Fatalf("Won(%d): checkpoint one's save list = %+v, want exactly one entry", m, entries)
		}
		if _, town, err := load2(entries[0].Name); err != nil || !town {
			t.Fatalf("Won(%d): loading checkpoint one: town %v err %v", m, town, err)
		}

		name2, err := save2(false)
		switch {
		case err != nil:
			refusals = append(refusals, fmt.Sprintf("Won(%d): checkpoint two SAVE hard-failed: %v", m, err))
		case !IsOriginal(name2):
			refusals = append(refusals, fmt.Sprintf("Won(%d): checkpoint two fell back to %q", m, name2))
		default:
			native++
		}
	}
	t.Logf("second SAVE after LOAD, nothing changed: %d/%d states stay native", native, len(states))
	if len(refusals) != 0 || native != len(states) {
		t.Fatalf("%d/%d states did not stay native across LOAD -> SAVE with nothing changed:\n%s",
			len(refusals), len(states), strings.Join(refusals, "\n"))
	}
}

// The required correction's own item 1, second clause (LOAD -> world map ->
// SAVE) is
// TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeAfterWorldMapOpened
// above: it reopens the world map onto markers the reloaded baseline already
// carries and checkpoint two stays native. A companion shape — open a
// marker the baseline does NOT carry, post-reload, then SAVE — was drafted
// and dropped here: campaignProgress.projection() (campaignprogress.go),
// which builds Campaign.Markers for BOTH marshal and EncodeSave, only ever
// reflects what was already decode-frozen at LOAD (populated once, in
// campaignProgressFromSAV, from the file's own Markers list); only
// nativeCampaignProjection (nativecity.go) reads the live once-selected UI
// cache (Snapshot.WorldSelectedOnce) into Campaign.Markers, and DIV-910
// means a reloaded session's SAVE never reaches that function again.

// A quick-spell change made after LOAD remains a native SAV checkpoint.
func TestReleaseNativeTownSaveSecondSaveAfterReloadCustomQuickSpellStaysNative(t *testing.T) {
	f := releaseFront(t)
	_, _ = reachabilityWalkArrive(t, f, "Quick Spell After Reload")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 24680
	wantChapter := f.Town.Chapter()

	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save1, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name1, err := save1(false)
	if err != nil || !IsOriginal(name1) {
		t.Fatalf("checkpoint one = %q err %v, want a native .sav", name1, err)
	}

	reloaded := releaseFront(t)
	save2, list2, load2 := agsSaveSeams(reloaded, store, OriginalStore{}, nil)
	entries := list2()
	if len(entries) != 1 {
		t.Fatalf("checkpoint one's save list = %+v, want exactly one entry", entries)
	}
	if _, town, err := load2(entries[0].Name); err != nil || !town {
		t.Fatalf("loading checkpoint one: town %v err %v", town, err)
	}
	if reloaded.originalCity == nil {
		t.Fatal("fixture assumption broke: reload bound no imported-city baseline")
	}
	want := reloaded.originalCity.baselineQuickSpells
	want[0] = 65535 // No original book cell; native AGS must retain the ID.
	if want == reloaded.originalCity.baselineQuickSpells {
		t.Fatal("fixture assumption broke: the mutated slots still equal the baseline")
	}
	reloaded.quickSpells = want

	name2, err := save2(false)
	if err != nil {
		t.Fatalf("checkpoint two save(false): %v", err)
	}
	if !IsOriginal(name2) {
		t.Fatalf("checkpoint two published %q, want the sole SAV format", name2)
	}

	third := releaseFront(t)
	_, _, load3 := agsSaveSeams(third, store, OriginalStore{Dir: dir}, nil)
	if _, town, err := load3(name2); err != nil || !town {
		t.Fatalf("loading checkpoint two: town %v err %v", town, err)
	}
	if third.quickSpells != want {
		t.Fatalf("checkpoint two quick spells = %v, want %v", third.quickSpells, want)
	}
	if third.Town.Chapter() != wantChapter {
		t.Fatalf("checkpoint two chapter = %d, want %d", third.Town.Chapter(), wantChapter)
	}
}

// TestReleaseNativeTownSaveSecondSaveAfterReloadStrippedHeroBesideHiredMercenarySavesAsSAV
// hires a squad, saves a native SAV, reloads it, then strips the hero's worn
// equipment in the shop and sells it. marshal is the writer a reloaded native
// session's SAVE goes through (DIV-910), and it writes each member's current
// worn set. The second reload must keep the hero stripped, the hired member as
// it was, and the sale price in the purse.
func TestReleaseNativeTownSaveSecondSaveAfterReloadStrippedHeroBesideHiredMercenarySavesAsSAV(t *testing.T) {
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Strip After Reload")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 5_000_000
	wantChapter := f.Town.Chapter()
	declared := f.Campaign.Value().Chapters[wantChapter].Mercenaries
	if len(declared) == 0 {
		t.Fatalf("fixture assumption broke: chapter %d declares no mercenaries", wantChapter)
	}
	hireType := declared[0]
	if _, ok := screen.toggleMercenary(hireType); !ok {
		t.Fatalf("fixture assumption broke: production tavern hire for type %d refused", hireType)
	}

	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save1, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name1, err := save1(false)
	if err != nil || !IsOriginal(name1) {
		t.Fatalf("checkpoint one = %q err %v, want a native .sav", name1, err)
	}

	reloaded := releaseFront(t)
	save2, list2, load2 := agsSaveSeams(reloaded, store, OriginalStore{}, nil)
	entries := list2()
	if len(entries) != 1 {
		t.Fatalf("checkpoint one's save list = %+v, want exactly one entry", entries)
	}
	if _, town, err := load2(entries[0].Name); err != nil || !town {
		t.Fatalf("loading checkpoint one: town %v err %v", town, err)
	}
	rs, ok := reloaded.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("fixture assumption broke: reloaded TownScreen is not *townScreen")
	}
	hiredIndex := -1
	for i, m := range reloaded.Carried {
		if int(m.MercenaryType) == hireType {
			hiredIndex = i
			break
		}
	}
	if hiredIndex < 0 {
		t.Fatal("fixture assumption broke: reload rebuilt no hired member")
	}

	rs.room = roomShop
	heroIndex := rs.shopMemberIndex()
	if reloaded.Carried[heroIndex].Hired() {
		t.Fatalf("the shop shows the hired member %d", heroIndex)
	}
	stripped := 0
	for slot := 1; slot <= sim.EquipSlots; slot++ {
		if !(*rs.shopWornItemSlots(heroIndex))[slot-1].Empty() {
			if act := rs.shopUnequipToTable(slot); act.Msg == "" {
				t.Fatalf("shopUnequipToTable(%d) returned no message", slot)
			}
			stripped++
		}
	}
	if stripped == 0 {
		t.Fatal("fixture assumption broke: the reloaded hero wore nothing to strip")
	}
	if act := rs.shopSell(); act.Msg == "" {
		t.Fatal("shopSell returned no message")
	}
	wantPurse := reloaded.Town.Gold()
	// The member's own items, not its Worn code list, are what the town
	// shows and the SAV writes.
	heroName := reloaded.Carried[heroIndex].Name
	wantHeroWorn, wantHeroPack := memberItemCodes(reloaded.Carried[heroIndex])
	wantHiredWorn, wantHiredPack := memberItemCodes(reloaded.Carried[hiredIndex])
	wantRosterLen := len(reloaded.Carried)

	name2, err := save2(false)
	if err != nil {
		t.Fatalf("checkpoint two save(false): %v", err)
	}
	if !IsOriginal(name2) {
		t.Fatalf("checkpoint two published %q; a stripped hero beside a hired member must write SAV", name2)
	}

	third := releaseFront(t)
	_, _, load3 := agsSaveSeams(third, store, OriginalStore{}, nil)
	if _, town, err := load3(localOriginalSaveToken(name2)); err != nil || !town {
		t.Fatalf("loading checkpoint two: town %v err %v", town, err)
	}
	if got := len(third.Carried); got != wantRosterLen {
		t.Fatalf("checkpoint two roster length = %d, want %d", got, wantRosterLen)
	}
	heroFound, hiredFound := false, false
	for _, m := range third.Carried {
		worn, pack := memberItemCodes(m)
		switch {
		case int(m.MercenaryType) == hireType && !hiredFound:
			hiredFound = true
			if worn != wantHiredWorn || !slices.Equal(pack, wantHiredPack) {
				t.Fatalf("checkpoint two hired member worn %v pack %v, want %v and %v", worn, pack, wantHiredWorn, wantHiredPack)
			}
		case !m.Hired() && m.Name == heroName && !heroFound:
			heroFound = true
			if worn != wantHeroWorn || !slices.Equal(pack, wantHeroPack) {
				t.Fatalf("checkpoint two hero worn %v pack %v, want %v and %v: the stripped state must survive the SAV reload", worn, pack, wantHeroWorn, wantHeroPack)
			}
		}
	}
	if !heroFound || !hiredFound {
		t.Fatal("checkpoint two rebuilt no hero or no hired member")
	}
	if third.Town.Gold() != wantPurse {
		t.Fatalf("checkpoint two gold = %d, want %d: the sale price must not return", third.Town.Gold(), wantPurse)
	}
	if third.Town.Chapter() != wantChapter {
		t.Fatalf("checkpoint two chapter = %d, want %d", third.Town.Chapter(), wantChapter)
	}
}

// A descending hire order made after LOAD remains ordered across the next SAV.
func TestReleaseNativeTownSaveSecondSaveAfterReloadDescendingHireOrderStaysNative(t *testing.T) {
	f, screen := hireForOrderTest(t, "Descending After Reload")
	wantChapter := f.Town.Chapter()

	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save1, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name1, err := save1(false)
	if err != nil || !IsOriginal(name1) {
		t.Fatalf("checkpoint one = %q err %v, want a native .sav", name1, err)
	}
	_ = screen

	reloaded := releaseFront(t)
	save2, list2, load2 := agsSaveSeams(reloaded, store, OriginalStore{}, nil)
	entries := list2()
	if len(entries) != 1 {
		t.Fatalf("checkpoint one's save list = %+v, want exactly one entry", entries)
	}
	if _, town, err := load2(entries[0].Name); err != nil || !town {
		t.Fatalf("loading checkpoint one: town %v err %v", town, err)
	}
	rs, ok := reloaded.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("fixture assumption broke: reloaded TownScreen is not *townScreen")
	}

	// Descending type order: 14 before 6 (the review's own reproduction).
	if _, ok := rs.toggleMercenary(14); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 14 refused after reload")
	}
	if _, ok := rs.toggleMercenary(6); !ok {
		t.Fatal("fixture assumption broke: production tavern hire for type 6 refused after reload")
	}
	wantTypes, wantIDs := partyTypeAndIDSequence(reloaded.Carried)
	wantRosterLen := len(reloaded.Carried)

	name2, err := save2(false)
	if err != nil {
		t.Fatalf("checkpoint two save(false): %v", err)
	}
	if !IsOriginal(name2) {
		t.Fatalf("checkpoint two published %q, want the sole SAV format", name2)
	}

	third := releaseFront(t)
	_, _, load3 := agsSaveSeams(third, store, OriginalStore{Dir: dir}, nil)
	if _, town, err := load3(name2); err != nil || !town {
		t.Fatalf("loading checkpoint two: town %v err %v", town, err)
	}
	if got := len(third.Carried); got != wantRosterLen {
		t.Fatalf("checkpoint two roster length = %d, want %d", got, wantRosterLen)
	}
	gotTypes, gotIDs := partyTypeAndIDSequence(third.Carried)
	if !equalIntSets(gotTypes, wantTypes) {
		t.Fatalf("checkpoint two party MercenaryType sequence = %v, want %v: SAV must keep the true hire order", gotTypes, wantTypes)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("checkpoint two party ID sequence = %v, want %v", gotIDs, wantIDs)
		}
	}
	if third.Town.Chapter() != wantChapter {
		t.Fatalf("checkpoint two chapter = %d, want %d", third.Town.Chapter(), wantChapter)
	}
}
