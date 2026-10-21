package game

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mod"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// townModFront is the install's front end with the mods ids of dir set,
// their town layers included.
func townModFront(t *testing.T, dir string, ids ...string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	res := runTownModsFor(t, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), dir, ids...)
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if overrides, err := f.SetModTowns(res.Towns); err != nil || len(overrides) != 0 {
		t.Fatalf("town layer: %v, overrides %v", err, overrides)
	}
	return f
}

// townModReload is a cold LOAD of a town SAV in a fresh front end, with the
// fixture mod when mods is set and with none otherwise.
func townModReload(t *testing.T, raw []byte, mods string) (*FrontEnd, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	if mods != "" {
		g = townModFront(t, mods, "east-town")
	}
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	_, town, err := load("town.sav")
	if err == nil && !town {
		t.Fatal("the town SAV did not open a town")
	}
	return g, err
}

// The fixture mod on the install: East opens after mission 20, stands on the
// world map at its stated point, and travel lands on its square drawn with
// the mod's own pictures, offering mission 31. A town SAVE in East carries
// the mod mark; a cold LOAD with the mod lands in East with its description,
// and a LOAD without the mod refuses naming it.
func TestReleaseModTownOpensTravelsSavesAndLoads(t *testing.T) {
	mods := eastTownMods(t)
	f := townModFront(t, mods, "east-town")
	reachabilityWalkArrive(t, f, "East Hero")
	f.Town.Won(10)
	f.arriveInTown()
	if err := checkEastTown(f, true); err != nil {
		t.Fatal(err)
	}
	if d := f.Campaign.Value().Towns.descriptions["east-town:towns/east.json"]; f.Archives.townArt[d] != nil {
		t.Fatal("East's art is held by the archives every front end of the install shares")
	}
	writeTownModFrames(t, f)
	raw := currentTownSave(t, f)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark, ok, err := readModMark(doc)
	if err != nil || !ok || !slices.ContainsFunc(mark.Mods, func(m mod.EntryRecord) bool { return m.ID == "east-town" }) {
		t.Fatalf("the town SAV's mod mark %+v (present %v): %v", mark.SetRecord, ok, err)
	}
	g, err := townModReload(t, raw, mods)
	if err != nil {
		t.Fatal(err)
	}
	if g.Town.townID() != 2 {
		t.Fatalf("LOAD with the mod lands in town %d, want East", g.Town.townID())
	}
	if d := g.TownScreen().(*townScreen).townDescription(); d == nil || d.Town != "east" {
		t.Fatal("the loaded screen does not compose from East's description")
	}
	if _, err := townModReload(t, raw, ""); err == nil || !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), "east-town") {
		t.Fatalf("LOAD without the mod: %v, want a mod-mark refusal naming east-town", err)
	}
}

// A base-town change and a switch-off on the install: town 1's new name is
// the home scroll's title; switching East off leaves the map without it.
func TestReleaseModTownSwitchedOff(t *testing.T) {
	mods := eastTownMods(t)
	writeTownMod(t, mods, "no-east", "[[town]]\nid = 2\noff = true\n", "east-town")
	f := townModFront(t, mods, "east-town", "no-east")
	reachabilityWalkArrive(t, f, "West Hero")
	f.Town.Won(10)
	f.arriveInTown()
	f.Town.Arrive()
	s := f.TownScreen().(*townScreen)
	f.Town.Won(20)
	// With no other town open and no mission taken the gates stay shut, so
	// the map is entered directly.
	s.enterWorldMap()
	v := s.WorldMapView()
	if v.Words.WorldHomeTitle != "West" {
		t.Fatalf("the home scroll reads %q, want West (%s)", v.Words.WorldHomeTitle, v.Problem)
	}
	for _, m := range v.Missions {
		if m.Kind == placeTown && m.Number == 2 {
			t.Fatalf("the map shows the switched-off East: %+v", m)
		}
	}
	if f.Town.travelTo(2) {
		t.Fatal("the party travels to the switched-off East")
	}
}

// writeTownModFrames writes East's square and the world map seen from East
// as PNG files under AGAINROM_TOWN_MOD_FRAMES, outside the install, when
// that variable is set.
func writeTownModFrames(t *testing.T, f *FrontEnd) {
	t.Helper()
	if townModFramesDir(t, f) == "" {
		return
	}
	a, s := exteriorApp(t, f)
	if a.Screen() != ui.ScreenTown || f.Town.townID() != 2 {
		t.Fatalf("frames: screen %v town %d", a.Screen(), f.Town.townID())
	}
	writeTownModFrame(t, f, a, "east-square")
	s.Choose(3)
	writeTownModFrame(t, f, a, "world-map-from-east")
}

// writeTownModFrame writes a's headless frame as name.png under the root's
// directory of AGAINROM_TOWN_MOD_FRAMES, when that variable is set.
func writeTownModFrame(t *testing.T, f *FrontEnd, a *ui.App, name string) {
	t.Helper()
	dir := townModFramesDir(t, f)
	if dir == "" {
		return
	}
	a.Draw(ebiten.NewImage(640, 480))
	pix, note, err := a.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatalf("frame %s %q: %v", name, note, err)
	}
	writeTownModPNG(t, filepath.Join(dir, name+".png"), pix)
}

// townModFramesDir is the root's directory under AGAINROM_TOWN_MOD_FRAMES,
// created and outside the install; empty when that variable is not set.
func townModFramesDir(t *testing.T, f *FrontEnd) string {
	t.Helper()
	base := os.Getenv("AGAINROM_TOWN_MOD_FRAMES")
	if base == "" {
		return ""
	}
	dir, err := filepath.Abs(filepath.Join(base, strings.ToLower(filepath.Base(f.Archives.Root))))
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(f.Archives.Root, dir); err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("town mod frames must stay outside the install")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeTownModPNG(t *testing.T, path string, pix image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(file, pix)
	if err := file.Close(); err != nil || encodeErr != nil {
		t.Fatal(encodeErr, err)
	}
}

// A Cyrillic town name on the install: the home scroll and the travel card
// draw the name encoded into the install's alphabet, from West and from
// East, and a name holding a character the alphabet lacks refuses the
// launch naming the mod, the file and the line.
func TestReleaseModTownNameDrawsInTheInstallAlphabet(t *testing.T) {
	mods := eastTownMods(t)
	setEastTowns(t, mods, strings.NewReplacer(`"West"`, `"Запад"`, `"East"`, `"Восток"`).Replace)
	f := townModFront(t, mods, "east-town")
	encode := func(name string) string {
		t.Helper()
		b, err := encodeSaveLabel(name, f.textSelector())
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	west, east := encode("Запад"), encode("Восток")
	if f.textSelector() == text.SelectorConverting && (west != "\x87\xa0\xaf\xa0\xa4" || east != "\x82\xae\xe1\xe2\xae\xaa") {
		t.Fatalf("the converting install encodes West as % x and East as % x", west, east)
	}
	reachabilityWalkArrive(t, f, "East Hero")
	f.Town.Won(10)
	f.arriveInTown()
	f.Town.Arrive()
	f.Town.Won(20)
	a, s := exteriorApp(t, f)
	look := func(home string, card int, title string) int {
		t.Helper()
		s.Choose(3)
		v := s.WorldMapView()
		if v.Words.WorldHomeTitle != home {
			t.Fatalf("the home scroll holds % x, want % x (%s)", v.Words.WorldHomeTitle, home, v.Problem)
		}
		i := slices.IndexFunc(v.Missions, func(m ui.WorldMapMission) bool { return m.Kind == placeTown && m.Number == card })
		if i < 0 || v.Missions[i].Title != title {
			t.Fatalf("town %d's card: %+v, want the title % x", card, v.Missions, title)
		}
		return i
	}
	i := look(west, 2, east)
	writeTownModFrame(t, f, a, "world-map-cyrillic-from-west")
	card := ui.WorldMapCardRect(i)
	s.WorldMapClick(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2))
	for n := 0; n < 400 && !s.AtTownSquare(); n++ {
		s.WorldMapTick()
	}
	if f.Town.townID() != 2 || !s.AtTownSquare() {
		t.Fatalf("travel to East: town %d, square %v", f.Town.townID(), s.AtTownSquare())
	}
	look(east, 1, west)
	writeTownModFrame(t, f, a, "world-map-cyrillic-from-east")

	bad := eastTownMods(t)
	setEastTowns(t, bad, strings.NewReplacer(`"East"`, `"East Ω"`).Replace)
	g := releaseFront(t)
	res := runTownModsFor(t, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), bad, "east-town")
	if err := g.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if _, err := g.SetModTowns(res.Towns); err == nil ||
		!strings.Contains(err.Error(), `mod "east-town": data/towns.toml:15: the name "East Ω" cannot be drawn by this install`) {
		t.Fatalf("a name the install cannot draw: %v", err)
	}
}

// On the install a mod town that opens after a mission only it offers
// refuses the launch at its record; the fixture alone is accepted.
func TestReleaseModTownThatCanNeverOpenIsRefused(t *testing.T) {
	townModFront(t, eastTownMods(t), "east-town")
	for _, towns := range []string{"[[town]]\nid = 2\nopens-after = 30\nmissions = [30, 31]\n", "[[town]]\nid = 2\nopens-after = 30\nmissions = [30]\n"} {
		mods := eastTownMods(t)
		writeTownMod(t, mods, "loop", towns, "east-town")
		f := releaseFront(t)
		res := runTownModsFor(t, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), mods, "east-town", "loop")
		if err := f.SetMods(res.Rules, res.Set, false); err != nil {
			t.Fatal(err)
		}
		if _, err := f.SetModTowns(res.Towns); err == nil ||
			!strings.Contains(err.Error(), `mod "loop": data/towns.toml:1: town catalog: town 2 can never open: no town that opens offers mission 30`) {
			t.Errorf("%q: %v", towns, err)
		}
	}
}
