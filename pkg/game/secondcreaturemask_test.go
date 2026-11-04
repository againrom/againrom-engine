package game

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondMaskSpells is the fixture mask's book: spells 2 and 3. The mask
// also names spell 29, which no book holds (DIV-2621).
const secondMaskSpells = uint32(1)<<2 | uint32(1)<<3

// secondMaskSkills is the fixture row's five school skills after its mask.
var secondMaskSkills = [5]int32{40, 50, 60, 70, 80}

// secondMaskCreatureMission is the guard mission with one dragon of Player
// 2 whose Units row holds no Spell cell and Spell Power 30. With mask, the
// row's knownSpells cell names spells 2, 3 and 29 and the five cells after
// it are secondMaskSkills (R2-ENGINE-652); without, they are -1.
func secondMaskCreatureMission(mask bool) secondMissionContent {
	c := secondGuardMission()
	p := make([]int32, 62)
	for i := range p {
		p[i] = -1
	}
	p[29], p[54], p[55] = 0x47, 30, secondSpellServerID
	if mask {
		p[56] = int32(secondMaskSpells | 1<<29)
		copy(p[57:], secondMaskSkills[:])
	}
	c.creatures = map[int]dbEntry{2: {name: "Mask_Dragon", params: p}}
	c.units = append(c.units, alm.Unit{X: 56<<8 | 0x80, Y: 48<<8 | 0x80, ClassID: 9, ClassSubID: 1,
		ServerID: secondSpellServerID, Owner: 2, UnitID: secondSpellUnitID, GroupID: 6})
	return c
}

// checkSecondMaskCreature: the World's dragon holds the mask's book, school
// skills skills and no creature spell slot.
func checkSecondMaskCreature(t *testing.T, name string, w *sim.World, skills [5]int32) {
	t.Helper()
	e := secondSpellCreature(t, w)
	if e.KnownSpells != secondMaskSpells || e.Book.State != sim.BookPresent {
		t.Fatalf("%s: the dragon knows %#x in book state %d, want %#x present", name, e.KnownSpells, e.Book.State, secondMaskSpells)
	}
	if err := e.Book.Validate(e.KnownSpells); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if got := [5]int32(e.Skill[1:6]); got != skills {
		t.Fatalf("%s: the dragon's school skills %v, want %v", name, got, skills)
	}
	if e.CreatureSpells != ([sim.CreatureSpellSlots]sim.CreatureSpell{}) {
		t.Fatalf("%s: the dragon holds spell slots %v", name, e.CreatureSpells)
	}
}

// checkSecondMaskPanel: the selected dragon's card holds the World's spells,
// and its spellcaster hover lists them.
func checkSecondMaskPanel(t *testing.T, name string, f *FrontEnd, app *ui.App) {
	t.Helper()
	e := secondSpellCreature(t, f.live.world)
	live := f.live
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	inspectionCentre(live, int(e.X), int(e.Y))
	live.push()
	if err := app.HeadlessSelectEntity(uint32(e.ID)); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	subject, ok := live.view.InspectionPanel()
	if !ok || subject.ID != uint32(e.ID) {
		t.Fatalf("%s: the card shows %d (%v), want the dragon %d", name, subject.ID, ok, e.ID)
	}
	if subject.Char.Band != ui.CharacterBandCreature || subject.KnownSpells != secondMaskSpells {
		t.Fatalf("%s: the card holds band %d, spells %#x, want a creature with %#x", name, subject.Char.Band, subject.KnownSpells, secondMaskSpells)
	}
	words := f.Words
	words.Hover[192], words.ItemSpellNames[2], words.ItemSpellNames[3] = "Spells: ", "Two", "Three"
	if lines, ok := ui.MonsterSpellHint(words, subject.KnownSpells); !ok || !slices.Equal(lines, []string{"Spells: Two, Three"}) {
		t.Fatalf("%s: the card's spell list %q/%v", name, lines, ok)
	}
}

// secondRecordsAlone is raw without the engine document, so that LOAD reads
// the second game's records alone.
func secondRecordsAlone(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc.State.ValueRecords = slices.DeleteFunc(slices.Clone(doc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
		return strings.EqualFold(r.Path, sav.NativeFormPath)
	})
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// secondMaskLoad LOADs raw under c.
func secondMaskLoad(t *testing.T, c secondMissionContent, raw []byte) (*FrontEnd, *ui.App) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if f.live == nil {
		t.Fatal("the file does not LOAD")
	}
	return f, app
}

// A second-game creature whose Units row holds a knownSpells mask gets the
// mask's book and the five school skills after it (R2-ENGINE-652): on a
// direct mission start, on a campaign mission start from the map, after the
// engine's LOAD and after the LOAD of the records alone, and its card lists
// the spells. A file this engine wrote before the mask was read holds the
// dragon's book empty; its LOAD keeps the class's book and the file's
// school skills. Fails while the second game reads no mask.
func TestSecondCreatureMaskGivesItsBook(t *testing.T) {
	c := secondMaskCreatureMission(true)
	direct := secondMissionFrontOf(t, c)
	directApp := direct.App("second-mission")
	directApp.Layout(1024, 768)
	if err := directApp.OpenMission(direct.MissionOpener(10)); err != nil || direct.live == nil {
		t.Fatalf("direct mission 10: %v %s", err, directApp.HeadlessMessage())
	}
	checkSecondMaskCreature(t, "direct start", direct.live.world, secondMaskSkills)
	checkSecondMaskPanel(t, "direct start", direct, directApp)

	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnterFromMap(t, f, app, screen, nil)
	checkSecondMaskCreature(t, "campaign start", f.live.world, secondMaskSkills)
	raw := secondPageSave(t, f, app)

	loaded, loadedApp := secondMaskLoad(t, c, raw)
	checkSecondMaskCreature(t, "LOAD", loaded.live.world, secondMaskSkills)
	checkSecondMaskPanel(t, "LOAD", loaded, loadedApp)
	resaved := secondPageSave(t, loaded, loadedApp)
	bare, _ := secondMaskLoad(t, c, secondRecordsAlone(t, resaved))
	checkSecondMaskCreature(t, "LOAD of the records", bare.live.world, secondMaskSkills)

	old := secondMaskCreatureMission(false)
	of, oapp, oscreen, _ := secondBagFrontOf(t, secondBagSample(), old)
	secondBagEnterFromMap(t, of, oapp, oscreen, nil)
	e := secondSpellCreature(t, of.live.world)
	if e.KnownSpells != 0 || e.Book.State != sim.BookPresent {
		t.Fatalf("without the mask the dragon knows %#x in book state %d, want an empty book", e.KnownSpells, e.Book.State)
	}
	oldSkills := [5]int32(e.Skill[1:6])
	oldRaw := secondPageSave(t, of, oapp)
	oldLoaded, _ := secondMaskLoad(t, c, oldRaw)
	checkSecondMaskCreature(t, "LOAD of an empty book", oldLoaded.live.world, oldSkills)
	oldBare, _ := secondMaskLoad(t, c, secondRecordsAlone(t, oldRaw))
	checkSecondMaskCreature(t, "LOAD of the records' empty book", oldBare.live.world, oldSkills)
}
