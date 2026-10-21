package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/text"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// eastTownFixture is the fixture mod that names town 1 West and adds East,
// town 2, with a description and art of its own at a stated map point.
const eastTownFixture = "../modrt/testdata/mods/east-town"

// The fixture mod's map point of East.
var (
	modEastPoint = image.Pt(430, 230)
	modEastRect  = image.Rect(410, 205, 450, 235)
)

// eastTownSquare, eastTownMask and eastTownTavern are East's own pictures,
// drawn by code: the square, its hotspot mask and the tavern's centre.
func eastTownSquare() *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := range 480 {
		for x := range 640 {
			pic.SetRGBA(x, y, color.RGBA{R: uint8(x / 3), G: uint8(90 + y/4), B: uint8(200 - y/3), A: 255})
		}
	}
	return pic
}

func eastTownMask() *image.Paletted {
	pal := make(color.Palette, 256)
	for i := range pal {
		pal[i] = color.RGBA{R: uint8(i), G: uint8(i), B: uint8(i), A: 255}
	}
	mask := image.NewPaletted(image.Rect(0, 0, 640, 480), pal)
	for i, b := range []uint8{128, 144, 160, 176, 192} {
		for y := 300; y < 380; y++ {
			for x := 20 + i*120; x < 120+i*120; x++ {
				mask.SetColorIndex(x, y, b)
			}
		}
	}
	return mask
}

func eastTownTavern() *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 320, 480))
	for y := range 480 {
		for x := range 320 {
			pic.SetRGBA(x, y, color.RGBA{R: uint8(150 + x/8), G: uint8(60 + y/8), B: 40, A: 255})
		}
	}
	return pic
}

// writeEastTownFiles writes the files the fixture's East names into the mod
// folder dir: towns/east.json, the first game's description with the
// square's picture and mask and the tavern's centre read from the mod, and
// those three pictures.
func writeEastTownFiles(t *testing.T, dir string) {
	t.Helper()
	var desc map[string]any
	if err := json.Unmarshal(rom1TownJSON, &desc); err != nil {
		t.Fatal(err)
	}
	desc["town"] = "east"
	rekey := func(art any, name, key string) {
		for _, e := range art.([]any) {
			if entry := e.(map[string]any); entry["name"] == name {
				entry["key"] = key
				return
			}
		}
		t.Fatalf("the description holds no art %q", name)
	}
	rekey(desc["art"], "base", "mod:art/east-square.bmp")
	rekey(desc["art"], "mask", "mod:art/east-mask.bmp")
	for _, r := range desc["rooms"].([]any) {
		if room := r.(map[string]any); room["name"] == "tavern" {
			rekey(room["art"], "center", "mod:art/east-tavern.bmp")
		}
	}
	raw, err := json.MarshalIndent(desc, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"towns/east.json":     raw,
		"art/east-square.bmp": synth.BMP24(eastTownSquare()),
		"art/east-mask.bmp":   synth.BMP8(eastTownMask()),
		"art/east-tavern.bmp": synth.BMP24(eastTownTavern()),
	}
	for name, b := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// eastTownMods is a mods folder holding the fixture mod with its files.
func eastTownMods(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, "east-town"), os.DirFS(eastTownFixture)); err != nil {
		t.Fatal(err)
	}
	writeEastTownFiles(t, filepath.Join(dir, "east-town"))
	return dir
}

// writeTownMod writes a mod id under dir whose data/towns.toml is towns,
// loading after the mods after names.
func writeTownMod(t *testing.T, dir, id, towns string, after ...string) {
	t.Helper()
	manifest := fmt.Sprintf("id = %q\ntitle = %q\nversion = \"1.0\"\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n", id, id)
	if len(after) > 0 {
		manifest += fmt.Sprintf("load-after = [%q]\n", strings.Join(after, `", "`))
	}
	files := map[string]string{
		"mod.toml":        manifest,
		"main.star":       "def init(game, settings):\n    game.data.add(\"data/towns.toml\")\n",
		"data/towns.toml": towns,
	}
	for name, s := range files {
		path := filepath.Join(dir, id, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// runTownMods runs the mods ids of dir for the first game's English base.
func runTownMods(t *testing.T, dir string, ids ...string) modrt.Result {
	t.Helper()
	return runTownModsFor(t, "rom1-en", dir, ids...)
}

// runTownModsFor runs the mods ids of dir for the base named base.
func runTownModsFor(t *testing.T, base, dir string, ids ...string) modrt.Result {
	t.Helper()
	entries, err := mod.Resolve(dir, ids)
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, base, nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// layerFront is a front end over the travel fixture's map and campaign,
// with the shipped first-game catalog, before any mod.
func layerFront(t *testing.T) *FrontEnd {
	t.Helper()
	c := townCampaign(t)
	c.MercenaryCount = []int{4}
	c.Towns = townBooks["rom1"]
	containers := travelMapFS(t)
	// The shipped description's art is the install's, which a copy of it
	// names: the fixture install holds each key.
	for _, key := range shippedTownArtKeys(townBooks["rom1"].descriptions["rom1"]) {
		containers.Overlay(key, func() ([]byte, error) { return []byte{0}, nil })
	}
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: containers},
		Font: resolved(alphabetFont(), nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
}

// shippedTownArtKeys are every install key description d reads.
func shippedTownArtKeys(d *town.Description) []string {
	var keys []string
	for _, e := range townArtEntries(d) {
		keys = append(keys, e.Keys()...)
	}
	return keys
}

// alphabetFont is a font of the English install's shape: selector 0, a drawn
// glyph for every record but the lowercase Cyrillic bytes 0xE0 to 0xFF, whose
// records are blank.
func alphabetFont() *text.Font {
	f := missionFont()
	for i := range f.Glyphs {
		if i+text.FirstChar >= 0xE0 {
			continue
		}
		f.Glyphs[i].Pixels[0] = text.Pixel{Level: text.MaxLevel, Painted: true}
	}
	return f
}

// layeredFront is layerFront with the town layers of mods ids of dir set.
func layeredFront(t *testing.T, dir string, ids ...string) (*FrontEnd, []string, error) {
	t.Helper()
	f := layerFront(t)
	overrides, err := f.SetModTowns(runTownMods(t, dir, ids...).Towns)
	return f, overrides, err
}

// checkEastTown is the fixture mod's result on f, a front end whose party
// stands in West: East is closed until mission 20 is won, then stands on
// the map at its stated point titled East, West's home scroll reads West,
// and travel lands on East's square, composed from its own description,
// offering mission 31. With drawn set, the square, its mask and the tavern
// are drawn with the mod's own pictures.
func checkEastTown(f *FrontEnd, drawn bool) error {
	tw := f.Town
	tw.Arrive()
	s := f.TownScreen().(*townScreen)
	if len(tw.travelTowns()) != 0 || tw.travelTo(2) {
		return errors.New("East is open before mission 20")
	}
	tw.Won(20)
	s.Choose(3)
	v := s.WorldMapView()
	i := slices.IndexFunc(v.Missions, func(m ui.WorldMapMission) bool { return m.Kind == placeTown && m.Number == 2 })
	if i < 0 {
		return fmt.Errorf("the map shows no East after mission 20: %+v", v.Missions)
	}
	if m := v.Missions[i]; m.Anchor != modEastPoint || m.Region != modEastRect || m.Title != "East" || !m.Enabled {
		return fmt.Errorf("East on the map: %+v, want %v in %v titled East", m, modEastPoint, modEastRect)
	}
	if v.Words.WorldHomeTitle != "West" {
		return fmt.Errorf("the home scroll reads %q, want West", v.Words.WorldHomeTitle)
	}
	card := ui.WorldMapCardRect(i)
	s.WorldMapClick(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2))
	for n := 0; n < 400 && !s.AtTownSquare(); n++ {
		s.WorldMapTick()
	}
	if tw.townID() != 2 || !s.AtTownSquare() || s.worldPosition != modEastPoint {
		return fmt.Errorf("arrival: town %d, square %v, position %v", tw.townID(), s.AtTownSquare(), s.worldPosition)
	}
	d := s.townDescription()
	if d == nil || d.Town != "east" || d == TownDescription(s.profile()) {
		return errors.New("East's square does not compose from the mod's description")
	}
	if got := offerMissions(tw.Offers(TownShop)); !slices.Equal(got, []int{31}) || len(tw.Offers(TownTavern)) != 0 {
		return fmt.Errorf("East offers shop %v tavern %v, want shop [31] alone", got, offerMissions(tw.Offers(TownTavern)))
	}
	if !drawn {
		return nil
	}
	if tav := s.tavernArt(); tav == nil || !samePixels(tav.Center, eastTownTavern()) {
		return errors.New("East's tavern is not drawn with the mod's centre picture")
	}
	square := s.squareArt()
	if square == nil || len(square.Pictures("base")) != 1 || !samePixels(square.Pictures("base")[0], eastTownSquare()) {
		return errors.New("East's square is not drawn with the mod's picture")
	}
	if square.Mask == nil || !bytes.Equal(square.Mask.Pix, eastTownMask().Pix) {
		return errors.New("East's square does not read the mod's mask")
	}
	return nil
}

// samePixels reports whether two pictures hold the same colours.
func samePixels(a, b image.Image) bool {
	if a == nil || a.Bounds() != b.Bounds() {
		return false
	}
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if color.RGBAModel.Convert(a.At(x, y)) != color.RGBAModel.Convert(b.At(x, y)) {
				return false
			}
		}
	}
	return true
}

// The fixture mod adds East, opened by mission 20, at its stated map point,
// with its own description, art and offers, and names town 1 West. Loss
// control: the same front end without the layer step fails every check.
func TestModTownOpensTravelsAndDrawsItsOwnArt(t *testing.T) {
	f, overrides, err := layeredFront(t, eastTownMods(t), "east-town")
	if err != nil || len(overrides) != 0 {
		t.Fatalf("layer: %v, overrides %v", err, overrides)
	}
	c := f.Campaign.Value().Towns.catalog
	if ids := c.IDs(); !slices.Equal(ids, []int{1, 2}) {
		t.Fatalf("merged catalog towns %v, want [1 2]", ids)
	}
	if west, _ := c.Town(1); west.Name != "West" || west.Description != "rom1" || west.MapPoint.Object != 1 {
		t.Fatalf("town 1 after the change of its name: %+v", west)
	}
	if err := checkEastTown(f, false); err != nil {
		t.Fatal(err)
	}
	book := f.Campaign.Value().Towns
	d := book.descriptions["east-town:towns/east.json"]
	src := modTownArtSource{install: f.Archives.Containers, files: book.modArt(d)}
	for key, want := range map[string][]byte{"mod:art/east-square.bmp": synth.BMP24(eastTownSquare()),
		"mod:art/east-mask.bmp": synth.BMP8(eastTownMask()), "mod:art/east-tavern.bmp": synth.BMP24(eastTownTavern())} {
		if got, err := src.ReadFile(key); err != nil || !bytes.Equal(got, want) {
			t.Fatalf("East's art %s: %v", key, err)
		}
	}
	if _, err := src.ReadFile("mod:art/absent.bmp"); err == nil {
		t.Fatal("a mod key the mod does not carry reads")
	}
	if got, err := src.ReadFile("graphics/global.map/pathmap.bmp"); err != nil || len(got) == 0 {
		t.Fatalf("an install key through East's art source: %v", err)
	}
	if err := checkEastTown(layerFront(t), false); err == nil {
		t.Fatal("loss control: a front end without the layer step passes the fixture's checks")
	}
}

// A later mod switches the fixture's East off: it leaves the catalog and the
// map. Switching off the starting town, or a town no layer holds, refuses
// the launch at the mod, the file and the line.
func TestModSwitchesATownOff(t *testing.T) {
	dir := eastTownMods(t)
	writeTownMod(t, dir, "no-east", "# East is gone.\n[[town]]\nid  = 2\noff = true\n", "east-town")
	f, _, err := layeredFront(t, dir, "east-town", "no-east")
	if err != nil {
		t.Fatal(err)
	}
	if ids := f.Campaign.Value().Towns.catalog.IDs(); !slices.Equal(ids, []int{1}) {
		t.Fatalf("towns after the switch-off: %v", ids)
	}
	f.Town.Arrive()
	f.Town.Won(20)
	if len(f.Town.travelTowns()) != 0 || f.Town.travelTo(2) {
		t.Fatal("a switched-off town is still a travel destination")
	}
	if err := checkEastTown(f, false); err == nil {
		t.Fatal("the switched-off East still passes the fixture's checks")
	}
	for _, c := range []struct{ towns, want string }{
		{"\n[[town]]\nid  = 1\noff = true\n", `mod "bad": data/towns.toml:2: town 1 is the campaign's starting town`},
		{"[[town]]\nid  = 9\noff = true\n", `mod "bad": data/towns.toml:1: town 9 is switched off, but no layer holds it`},
		{"[[town]]\nid  = 1\noffers = \"scenario-bank\"\n", `mod "bad": data/towns.toml:3: town 1 takes offers "scenario-bank"; a first-game town takes offers = "chapter-keys"`},
	} {
		bad := t.TempDir()
		writeTownMod(t, bad, "bad", c.towns)
		if _, _, err := layeredFront(t, bad, "bad"); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.towns, err, c.want)
		}
	}
}

// Two mods that set one field of one town: the later wins when load-after
// orders them, and the override is named; without the order, or when both
// add one new town, the launch refuses naming both mods. Loss control: the
// ordered pair with its order ignored refuses.
func TestTwoModsOnOneTown(t *testing.T) {
	dir := t.TempDir()
	writeTownMod(t, dir, "alpha", "[[town]]\nid   = 1\nname = \"Alpha\"\n")
	writeTownMod(t, dir, "beta", "# Beta renames the town alpha named.\n[[town]]\nid   = 1\nname = \"Beta\"\n", "alpha")
	f, overrides, err := layeredFront(t, dir, "alpha", "beta")
	if err != nil {
		t.Fatal(err)
	}
	if e, _ := f.Campaign.Value().Towns.catalog.Town(1); e.Name != "Beta" {
		t.Fatalf("town 1 is named %q, want the later mod's Beta", e.Name)
	}
	want := `town 1 name: mod "beta" (data/towns.toml:4) overrides mod "alpha"`
	if !slices.Equal(overrides, []string{want}) {
		t.Fatalf("overrides %q, want %q", overrides, want)
	}
	res := runTownMods(t, dir, "alpha", "beta")
	res.Towns.Order = nil
	if _, err := layerFront(t).SetModTowns(res.Towns); err == nil {
		t.Fatal("loss control: the ordered pair with its order ignored is accepted")
	}

	writeTownMod(t, dir, "gamma", "[[town]]\nid   = 1\nname = \"Gamma\"\n")
	if _, _, err := layeredFront(t, dir, "alpha", "gamma"); err == nil ||
		!strings.Contains(err.Error(), `town 1 name is set by mod "alpha" and by mod "gamma"`) {
		t.Fatalf("unordered pair: %v", err)
	}
	east := "[[town]]\nid = 3\ndescription = \"rom1\"\nhooks = \"chapters\"\noffers = \"chapter-keys\"\nmap-point = { object = 3 }\n"
	writeTownMod(t, dir, "east-a", east)
	writeTownMod(t, dir, "east-b", east)
	if _, _, err := layeredFront(t, dir, "east-a", "east-b"); err == nil ||
		!strings.Contains(err.Error(), `town 3 is added by mod "east-a" and named by mod "east-b"`) {
		t.Fatalf("one new town from two mods: %v", err)
	}
	// A mod that names a town another mod adds loads after that mod.
	writeTownMod(t, dir, "east-c", "[[town]]\nid = 3\nname = \"Third\"\n", "east-a")
	if f, _, err := layeredFront(t, dir, "east-a", "east-c"); err != nil {
		t.Fatalf("a mod naming a town an earlier mod adds: %v", err)
	} else if e, _ := f.Campaign.Value().Towns.catalog.Town(3); e.Name != "Third" {
		t.Fatalf("town 3 is named %q, want Third", e.Name)
	}
	writeTownMod(t, dir, "east-d", "[[town]]\nid = 3\nname = \"Third\"\n")
	if _, _, err := layeredFront(t, dir, "east-a", "east-d"); err == nil ||
		!strings.Contains(err.Error(), `town 3 is added by mod "east-a" and named by mod "east-d"`) {
		t.Fatalf("a mod naming a town an earlier mod adds, unordered: %v", err)
	}
}

// Every refusal of a merged catalog names the mod, the file and the line:
// a mission two towns list, an opening mission the campaign does not hold,
// a map point off the map, a description that is not shipped and art the
// mod does not carry. The second game's catalog takes no layer.
func TestTownLayerRefusals(t *testing.T) {
	for _, c := range []struct{ towns, want string }{
		{"[[town]]\nid = 3\ndescription = \"rom1\"\nhooks = \"chapters\"\noffers = \"chapter-keys\"\nmap-point = { object = 3 }\nmissions = [31]\n",
			`mod "bad": data/towns.toml:1: town catalog rom1: mission 31 is listed by towns 2 and 3`},
		{"[[town]]\nid = 2\nopens-after = 25\n", `mod "bad": data/towns.toml:1: town catalog: town 2 opens after mission 25`},
		{"[[town]]\nid = 2\nmap-point = { point = [700, 20], rect = [690, 10, 20, 20] }\n", `mod "bad": data/towns.toml:1: town 2's map point (700,20)`},
		{"[[town]]\nid = 2\nmap-point = { point = [7, 20] }\n", `mod "bad": data/towns.toml:1: town catalog rom1: town 2 map rect [] is not`},
		{"[[town]]\nid = 2\ndescription = \"rom1-east\"\n", `mod "bad": data/towns.toml:1: town catalog rom1: town 2 names description "rom1-east" that is not shipped`},
		{"[[town]]\nid = 2\ndescription = \"rom2-town1\"\n", `description rom2-town1 named under hook sets destinations and chapters`},
		{"[[town]]\nid = 3\ndescription = \"rom2-town1\"\nhooks = \"destinations\"\noffers = \"scenario-bank\"\nmap-point = { object = 3 }\nname = \"Bank\"\n",
			`mod "bad": data/towns.toml:4: town 3 takes hooks "destinations"; a first-game town takes hooks = "chapters"`},
		{"[[town]]\nid = 2\noffers = \"scenario-bank\"\n", `mod "bad": data/towns.toml:3: town 2 takes offers "scenario-bank"; a first-game town takes offers = "chapter-keys"`},
	} {
		dir := eastTownMods(t)
		writeTownMod(t, dir, "bad", c.towns, "east-town")
		if _, _, err := layeredFront(t, dir, "east-town", "bad"); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.towns, err, c.want)
		}
	}
	dir := eastTownMods(t)
	if err := os.Remove(filepath.Join(dir, "east-town", "art", "east-tavern.bmp")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := layeredFront(t, dir, "east-town"); err == nil ||
		!strings.Contains(err.Error(), `mod "east-town": data/towns.toml:11: description towns/east.json: art "mod:art/east-tavern.bmp"`) {
		t.Fatalf("missing art: %v", err)
	}
	res := runTownMods(t, eastTownMods(t), "east-town")
	if _, _, err := layerTownBook(townBooks["rom2"], res.Towns, Campaign{}, nil, nil, modTownInstall{}); err == nil || !strings.Contains(err.Error(), "first game's catalog") {
		t.Fatalf("the second game's catalog took a layer: %v", err)
	}
}

// A town's entry with a stated map point decodes in the catalog format and
// stands at that point on the map.
func TestStatedMapPointPlacesATown(t *testing.T) {
	e := town.Entry{ID: 2, MapPoint: town.MapPoint{Point: []int{5, 6}, Rect: []int{1, 2, 10, 8}}}
	p, r, ok := townMapPoint(e, nil)
	if !ok || p != image.Pt(5, 6) || r != image.Rect(1, 2, 11, 10) {
		t.Fatalf("stated point %v %v %v", p, r, ok)
	}
}

// setEastTowns rewrites the fixture mod's data/towns.toml under dir by edit.
func setEastTowns(t *testing.T, dir string, edit func(string) string) {
	t.Helper()
	path := filepath.Join(dir, "east-town", "data", "towns.toml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(edit(string(b))), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A town's name is written in UTF-8 and drawn in the install's byte
// alphabet: under the converting selector the home scroll and East's card
// show the converted bytes. A character the alphabet lacks refuses the
// launch, naming the mod, the file and the line.
func TestModTownNameDrawsInTheInstallAlphabet(t *testing.T) {
	dir := eastTownMods(t)
	setEastTowns(t, dir, strings.NewReplacer(`"West"`, `"Запад"`, `"East"`, `"Восток"`).Replace)
	f := layerFront(t)
	font := alphabetFont()
	for i := range font.Glyphs {
		font.Glyphs[i].Pixels[0] = text.Pixel{Level: text.MaxLevel, Painted: true}
	}
	font.Selector = text.SelectorConverting
	f.Font = resolved(font, nil)
	if _, err := f.SetModTowns(runTownMods(t, dir, "east-town").Towns); err != nil {
		t.Fatal(err)
	}
	const west, east = "\x87\xa0\xaf\xa0\xa4", "\x82\xae\xe1\xe2\xae\xaa"
	tw := f.Town
	tw.Arrive()
	s := f.TownScreen().(*townScreen)
	tw.Won(20)
	s.Choose(3)
	v := s.WorldMapView()
	if v.Words.WorldHomeTitle != west {
		t.Errorf("the home scroll holds % x, want % x", v.Words.WorldHomeTitle, west)
	}
	i := slices.IndexFunc(v.Missions, func(m ui.WorldMapMission) bool { return m.Kind == placeTown && m.Number == 2 })
	if i < 0 || v.Missions[i].Title != east {
		t.Fatalf("East's card: %+v, want the title % x", v.Missions, east)
	}

	dir = eastTownMods(t)
	setEastTowns(t, dir, strings.NewReplacer(`"East"`, `"East Ω"`).Replace)
	if _, _, err := layeredFront(t, dir, "east-town"); err == nil ||
		!strings.Contains(err.Error(), `mod "east-town": data/towns.toml:15: the name "East Ω" cannot be drawn by this install`) {
		t.Fatalf("a name the install cannot draw: %v", err)
	}
}

// A name made of characters the install encodes but its font draws blank is
// refused at the record, naming the character: the English install stores
// every Windows-1251 byte, and its font draws the Cyrillic capitals but not
// the lowercase letters. A space is no glyph and passes.
func TestModTownNameWithNoGlyphIsRefused(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{"Восток", `the name "Восток" cannot be drawn by this install: the install's font has no glyph for 'о' (U+043E)`},
		{"East Ω", `the name "East Ω" cannot be drawn by this install: SAV label cannot represent U+03A9`},
		{"Big East", ""},
		{"ВОСТОК", ""},
	} {
		dir := eastTownMods(t)
		setEastTowns(t, dir, strings.NewReplacer(`"East"`, `"`+c.name+`"`).Replace)
		_, _, err := layeredFront(t, dir, "east-town")
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%q: %v", c.name, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%q: %v, want %q", c.name, err, c.want)
		}
	}
}

// openingTown is a [[town]] table adding town id, opening after mission
// after and listing missions when missions is not empty.
func openingTown(id, after, missions string) string {
	s := "[[town]]\nid = " + id + "\ndescription = \"rom1\"\nhooks = \"chapters\"\noffers = \"chapter-keys\"\n" +
		"map-point = { point = [100, 100], rect = [90, 90, 20, 20] }\nopens-after = " + after + "\n"
	if missions != "" {
		s += "missions = [" + missions + "]\n"
	}
	return s
}

// A town opens when a town already open offers the mission it opens after;
// a mission no building offers is the campaign's own and open from the
// start. A town that can never open, and towns open from the start that
// offer nothing, refuse the launch at the record that made it so. A chain
// of towns, each opened by a mission of the one before, is accepted.
func TestModTownThatCanNeverOpenIsRefused(t *testing.T) {
	for _, c := range []struct{ towns, want string }{
		{"[[town]]\nid = 2\nopens-after = 30\nmissions = [30, 31]\n", `mod "bad": data/towns.toml:1: town catalog: town 2 can never open: no town that opens offers mission 30`},
		{"[[town]]\nid = 2\nopens-after = 30\nmissions = [30]\n", `mod "bad": data/towns.toml:1: town catalog: town 2 can never open: no town that opens offers mission 30`},
		{openingTown("4", "41", "") + openingTown("3", "40", "40, 41"), `mod "bad": data/towns.toml:1: town catalog: town 4 can never open: no town that opens offers mission 41`},
		{"[[town]]\nid = 1\nmissions = [99]\n", `mod "bad": data/towns.toml:1: town catalog: the towns open from the start (1) offer no mission`},
	} {
		dir := eastTownMods(t)
		writeTownMod(t, dir, "bad", c.towns, "east-town")
		if _, _, err := layeredFront(t, dir, "east-town", "bad"); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.towns, err, c.want)
		}
	}
	dir := eastTownMods(t)
	writeTownMod(t, dir, "chain", openingTown("3", "31", "40")+openingTown("4", "40", "41"), "east-town")
	f, _, err := layeredFront(t, dir, "east-town", "chain")
	if err != nil {
		t.Fatalf("a chain of towns each opened by the one before: %v", err)
	}
	if ids := f.Campaign.Value().Towns.catalog.IDs(); !slices.Equal(ids, []int{1, 2, 3, 4}) {
		t.Fatalf("the chain's catalog holds %v", ids)
	}
}
