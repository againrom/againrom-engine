package game

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

const (
	// girlType is the tavern's female archer, the mercenary type of the
	// definition rows NPC06_1 to NPC06_4.
	girlType = 6
	// girlChapter is a town whose tavern lists type 6; its mercenary tier is 3.
	girlChapter = 100
)

// girlName is the example mod's display name in the language of the install.
// A panel draws the install's own byte alphabet, so the Russian name is the
// code page bytes of the text.
func girlName(f *FrontEnd) string {
	text := "Archer girl"
	if modrt.LanguageFor(f.ModSet().Base) == "ru" {
		text = "Лучница"
	}
	name, err := encodeSaveLabel(text, f.textSelector())
	if err != nil {
		panic(err)
	}
	return name
}

// characterModFront is a front end on the lawful install, running under the
// archer-girl example mod when withMod is set.
func characterModFront(t *testing.T, withMod bool) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if !withMod {
		return f
	}
	entries, err := mod.Resolve(modItemsDir, []string{"archer-girl"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if err := f.SetModItems(res.Items); err != nil {
		t.Fatal(err)
	}
	if err := f.SetModCharacters(res.Characters); err != nil {
		t.Fatal(err)
	}
	return f
}

// characterTown opens the town of girlChapter with every mercenary type
// enabled, the tavern in view and a purse that buys any squad.
func characterTown(t *testing.T, f *FrontEnd) *townScreen {
	t.Helper()
	c := f.Campaign.Value()
	town := NewTown(c)
	for mission := range c.Chapters {
		if mission < girlChapter {
			town.Won(mission)
		}
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.NextParty()
	f.arriveInTown()
	for typ := 1; typ <= 15; typ++ {
		f.Town.mercEnabled[typ] = true
	}
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	s.composeShopFaces()
	return s
}

// girlMember is the hired archer girl of a party, and false when none is hired.
func girlMember(party []mapload.PartyMember) (mapload.PartyMember, bool) {
	for _, p := range party {
		if p.MercenaryType == girlType {
			return p, true
		}
	}
	return mapload.PartyMember{}, false
}

// girlWitness states what an edited hired girl must be: the peasant archer's
// figure sheet, the bow kept and nothing else worn, her own display name.
func girlWitness(f *FrontEnd, m mapload.PartyMember) error {
	if m.FigureFace != 9 || data.FigureDir(m.FigureDir) != data.FigureDirWomanFighter {
		return fmt.Errorf("figure %s face %d, want the woman fighter's sheet 9", m.FigureDir, m.FigureFace)
	}
	if m.Class != 14 {
		return fmt.Errorf("class %d, want the archer's 14", m.Class)
	}
	worn, _ := memberItemCodes(m)
	for slot, code := range worn {
		if code != 0 && slot != 0 {
			return fmt.Errorf("slot %d holds %#x, want no armour", slot+1, code)
		}
	}
	if worn[0] == 0 {
		return errors.New("the bow is gone")
	}
	if name := partyPanelSubject(m, f.Table, f.Words).Name; name != girlName(f) {
		return fmt.Errorf("the party panel names her %q, want %q", name, girlName(f))
	}
	return nil
}

func TestReleaseModCharactersEditTheTargetRowsOnly(t *testing.T) {
	plain, edited := characterModFront(t, false), characterModFront(t, true)
	before, after := plain.Table.Humans, edited.Table.Humans
	if before.Len() != after.Len() || plain.Humans.Len() != before.Len() || edited.Humans.Len() != after.Len() {
		t.Fatalf("the row count moved: %d to %d", before.Len(), after.Len())
	}
	targets := map[string]string{"NPC06_1": "A_PeasantGuard1", "NPC06_2": "A_PeasantGuard2", "NPC06_3": "A_PeasantGuard3", "NPC06_4": "A_PeasantGuard4"}
	byName := map[string]int{}
	for i := 1; i < before.Len(); i++ {
		byName[before.EntryName(i)] = i
	}
	hit := 0
	for i := 0; i < before.Len(); i++ {
		name := before.EntryName(i)
		if after.EntryName(i) != name || edited.Humans.EntryName(i) != name {
			t.Fatalf("row %d changed its name", i)
		}
		kind, isTarget := targets[name]
		if !isTarget {
			if !slices.Equal(before.EntryParams(i), after.EntryParams(i)) || !slices.Equal(before.EntryStrings(i), after.EntryStrings(i)) {
				t.Fatalf("row %d %q is not a target and changed", i, name)
			}
			if _, named := edited.Table.Mods.CharacterName(name); named && name != "" {
				t.Fatalf("%q is named by the mod", name)
			}
			continue
		}
		hit++
		was, now, like := before.EntryParams(i), after.EntryParams(i), before.EntryParams(byName[kind])
		for slot := range was {
			wantSlot := was[slot]
			if slot >= data.HumanSlotTypeID && slot <= data.HumanSlotGender {
				wantSlot = like[slot]
			}
			if now[slot] != wantSlot {
				t.Fatalf("%s slot %d is %d, want %d", name, slot, now[slot], wantSlot)
			}
		}
		cells, stripped := before.EntryStrings(i), after.EntryStrings(i)
		if len(stripped) != len(cells) || stripped[0] != cells[0] || stripped[0] == "" || stripped[1] != cells[1] {
			t.Fatalf("%s kept cells %q of %q", name, stripped, cells)
		}
		for _, c := range stripped[2:] {
			if c != "" {
				t.Fatalf("%s still holds %q", name, c)
			}
		}
		if !slices.ContainsFunc(cells[2:], func(c string) bool { return c != "" }) {
			t.Fatalf("%s wore nothing to strip: the control is empty", name)
		}
		if n, ok := edited.Table.Mods.CharacterName(name); !ok || n != girlName(edited) {
			t.Fatalf("%s is named %q", name, n)
		}
	}
	if hit != len(targets) {
		t.Fatalf("%d of %d target rows exist", hit, len(targets))
	}
}

func TestReleaseModCharacterReachesTheTavernAndTheHiredParty(t *testing.T) {
	plain, edited := characterModFront(t, false), characterModFront(t, true)
	ps, es := characterTown(t, plain), characterTown(t, edited)

	pv, _, _, ok := ps.tavernCandidateDetail(girlType)
	if !ok {
		t.Fatal("the unmodded tavern has no candidate of type 6")
	}
	ev, _, _, ok := es.tavernCandidateDetail(girlType)
	if !ok {
		t.Fatal("the modded tavern has no candidate of type 6")
	}
	if ev.Subject.Name != girlName(edited) || ev.Subject.Char.Name != girlName(edited) {
		t.Fatalf("the tavern shows %q, want %q", ev.Subject.Name, girlName(edited))
	}
	if pv.Subject.Name == ev.Subject.Name {
		t.Fatalf("the unmodded tavern shows the same name %q", pv.Subject.Name)
	}

	// A mercenary of another type is untouched: same name.
	for _, typ := range []int{4, 7, 10} {
		a, _, _, aok := ps.tavernCandidateDetail(typ)
		b, _, _, bok := es.tavernCandidateDetail(typ)
		if aok != bok || a.Subject.Name != b.Subject.Name {
			t.Fatalf("type %d: %q, %v against %q, %v", typ, a.Subject.Name, aok, b.Subject.Name, bok)
		}
	}

	if _, ok := ps.toggleMercenary(girlType); !ok {
		t.Fatal("the unmodded hire was refused")
	}
	if _, ok := es.toggleMercenary(girlType); !ok {
		t.Fatal("the modded hire was refused")
	}
	pm, pok := girlMember(plain.Carried)
	em, eok := girlMember(edited.Carried)
	if !pok || !eok {
		t.Fatal("the hired squad holds no girl")
	}
	if err := girlWitness(edited, em); err != nil {
		t.Fatalf("modded: %v", err)
	}
	if err := girlWitness(plain, pm); err == nil {
		t.Fatal("the witness passes on the unmodded archer")
	}
	if pm.FigureFace == em.FigureFace || pm.Weapon == nil || em.Weapon == nil || pm.Weapon.Code != em.Weapon.Code {
		t.Fatalf("faces %d and %d, weapons %v and %v", pm.FigureFace, em.FigureFace, pm.Weapon, em.Weapon)
	}
	if pm.Hero != em.Hero {
		t.Fatalf("the statistics changed: %+v against %+v", pm.Hero, em.Hero)
	}
}

// characterModReload is a cold LOAD of a town SAV in a fresh front end that
// runs under the example mod, or under none when plain is set.
func characterModReload(t *testing.T, raw []byte, plain bool) (*FrontEnd, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := characterModFront(t, !plain)
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	_, town, err := load("town.sav")
	if err == nil && !town {
		t.Fatal("the town SAV did not open a town")
	}
	return g, err
}

func TestReleaseModCharacterSavesAndLoadsAsSAV(t *testing.T) {
	f := characterModFront(t, true)
	s := characterTown(t, f)
	if _, ok := s.toggleMercenary(girlType); !ok {
		t.Fatal("the hire was refused")
	}
	want, ok := girlMember(f.Carried)
	if !ok {
		t.Fatal("no girl")
	}
	raw := currentTownSave(t, f)

	g, err := characterModReload(t, raw, false)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := girlMember(g.Carried)
	if !ok {
		t.Fatal("the cold LOAD lost the girl")
	}
	if err := girlWitness(g, got); err != nil {
		t.Fatalf("after the cold LOAD: %v", err)
	}
	wornWant, packWant := memberItemCodes(want)
	wornGot, packGot := memberItemCodes(got)
	if wornWant != wornGot || !slices.Equal(packWant, packGot) || got.FigureFace != want.FigureFace || got.FigureDir != want.FigureDir ||
		got.Class != want.Class || got.DefinitionRow != want.DefinitionRow || got.Name != want.Name {
		t.Fatalf("saved %+v, loaded %+v", want, got)
	}

	// The next action works on the loaded girl: a second SAVE and LOAD carry the same state.
	h, err := characterModReload(t, currentTownSave(t, g), false)
	if err != nil {
		t.Fatal(err)
	}
	if again, ok := girlMember(h.Carried); !ok || girlWitness(h, again) != nil {
		t.Fatalf("after the second LOAD: %v %v", ok, girlWitness(h, again))
	}

	// A LOAD without the mod refuses by the mod mark.
	if _, err := characterModReload(t, raw, true); err == nil || !errors.Is(err, ErrModMark) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}

	// Loss control: the same hire in an unmodded game saves an armoured girl,
	// and the witness refuses her after her own cold LOAD.
	pf := characterModFront(t, false)
	ps := characterTown(t, pf)
	if _, ok := ps.toggleMercenary(girlType); !ok {
		t.Fatal("the unmodded hire was refused")
	}
	pg, err := characterModReload(t, currentTownSave(t, pf), true)
	if err != nil {
		t.Fatal(err)
	}
	plainGirl, ok := girlMember(pg.Carried)
	if !ok || girlWitness(pg, plainGirl) == nil {
		t.Fatalf("the unmodded girl passes the witness: %v", ok)
	}
	if plainGirl.FigureFace == got.FigureFace {
		t.Fatalf("both girls have face %d", got.FigureFace)
	}
}

// girlInMission finds the hired girl of a live mission: her party record and
// the name her map entity is drawn with.
func girlInMission(t *testing.T, f *FrontEnd) (mapload.PartyMember, string) {
	t.Helper()
	live := f.live
	for i, member := range live.mission.party {
		if member.MercenaryType != girlType || i >= len(live.mission.ids) {
			continue
		}
		id := live.mission.ids[i]
		for _, draw := range live.entityDraws() {
			if draw.ID == uint32(id) {
				return member, draw.Name
			}
		}
	}
	t.Fatal("the mission holds no hired girl")
	return mapload.PartyMember{}, ""
}

func TestReleaseModCharacterInAMissionAndItsSave(t *testing.T) {
	f := characterModFront(t, true)
	s := characterTown(t, f)
	if _, ok := s.toggleMercenary(girlType); !ok {
		t.Fatal("the hire was refused")
	}
	app := f.App("mod characters")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(girlChapter, f.Carried)); err != nil {
		t.Fatalf("open mission %d: %v", girlChapter, err)
	}
	member, drawn := girlInMission(t, f)
	if err := girlWitness(f, member); err != nil {
		t.Fatalf("in the mission: %v", err)
	}
	if drawn != girlName(f) {
		t.Fatalf("the map entity is named %q, want %q", drawn, girlName(f))
	}

	snap, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	saved, _, err := f.playerMissionSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	g := characterModFront(t, true)
	ag := g.App("mod characters reload")
	t.Cleanup(ag.StopAudio)
	ag.Layout(1024, 768)
	reopened, _, err := g.RestoreOriginal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := ag.OpenMission(reopened); err != nil {
		t.Fatal(err)
	}
	again, drawn := girlInMission(t, g)
	if err := girlWitness(g, again); err != nil {
		t.Fatalf("after the mission SAVE and cold LOAD: %v", err)
	}
	if drawn != girlName(g) {
		t.Fatalf("after the LOAD the map entity is named %q, want %q", drawn, girlName(g))
	}

	if _, _, err := characterModFront(t, false).RestoreOriginal(saved); err == nil || !errors.Is(err, ErrModMark) {
		t.Fatalf("a mission LOAD without the mod: %v", err)
	}
}

func TestReleaseModCharacterTavernScreenshot(t *testing.T) {
	dir := os.Getenv("AGAINROM_SHOT_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	base := BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))
	var shots [2]image.Image
	for i, withMod := range []bool{false, true} {
		f := characterModFront(t, withMod)
		s := characterTown(t, f)
		s.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: girlType}
		pix, err := ui.ComposeTownScreen(s, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, selected, ok := s.tavernSnapshot(s.tavernCandidates()); !ok || selected.key.id != girlType {
			t.Fatalf("the tavern did not select the archer: %v", selected.key)
		}
		name := "tavern-original-" + base + ".png"
		if withMod {
			name = "tavern-archer-girl-" + base + ".png"
		}
		out, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(out, pix); err != nil {
			t.Fatal(err)
		}
		if err := out.Close(); err != nil {
			t.Fatal(err)
		}
		shots[i] = pix
	}
	if sameImage(shots[0], shots[1]) {
		t.Fatal("the edited tavern screen equals the original one")
	}
}

func sameImage(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}
