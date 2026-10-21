package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/internal/synth"
	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/random"
	"againrom/pkg/vfs"
)

// The fixture mod's generated art: the main menu's and the town square's
// background, each one colour no installed picture holds, at the install's
// size.
var (
	modMenuColour   = color.RGBA{R: 0xfe, G: 0x01, B: 0xfd, A: 0xff}
	modSquareColour = color.RGBA{R: 0x02, G: 0xfd, B: 0xfe, A: 0xff}
)

const (
	modMenuPicture   = mainPrefix + "graphics/mainmenu/menu_.bmp"
	modSquarePicture = graphicsPrefix + "interface/town/townmain.bmp"
)

// modToneWAV is a generated 16-bit mono sound: a square wave no installed
// sound holds.
func modToneWAV() []byte {
	const frames, rate = 2205, 22050
	data := make([]byte, 2*frames)
	for i := 0; i < frames; i++ {
		v := int16(7000)
		if i/25%2 == 1 {
			v = -7000
		}
		binary.LittleEndian.PutUint16(data[2*i:], uint16(v))
	}
	fm := make([]byte, 16)
	binary.LittleEndian.PutUint16(fm[0:], 1)
	binary.LittleEndian.PutUint16(fm[2:], 1)
	binary.LittleEndian.PutUint32(fm[4:], rate)
	binary.LittleEndian.PutUint32(fm[8:], rate*2)
	binary.LittleEndian.PutUint16(fm[12:], 2)
	binary.LittleEndian.PutUint16(fm[14:], 16)
	out := binary.LittleEndian.AppendUint32([]byte("RIFF"), uint32(4+8+len(fm)+8+len(data)))
	out = append(out, "WAVE"...)
	out = append(out, "fmt "...)
	out = append(out, binary.LittleEndian.AppendUint32(nil, uint32(len(fm)))...)
	out = append(out, fm...)
	out = append(out, "data"...)
	out = append(out, binary.LittleEndian.AppendUint32(nil, uint32(len(data)))...)
	return append(out, data...)
}

// modFilesInstall is the install's archives, the front end the fixture mod is
// written for, and the first sound slot the install's bank plays.
func modFilesInstall(t *testing.T) (*FrontEnd, int) {
	t.Helper()
	f := releaseFront(t)
	slot := -1
	for s := 1; s < 1024 && slot < 0; s++ {
		if _, ok := f.SoundBank.Sample(s); ok {
			slot = s
		}
	}
	if slot < 0 {
		t.Fatal("the install's sound bank plays no slot")
	}
	return f, slot
}

// installText is the install's text file at address as UTF-8 lines, read as
// the text readers read it.
func installText(t *testing.T, f *FrontEnd, address string) []string {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile(address)
	if err != nil {
		t.Fatal(err)
	}
	utf, ok := DecodeInstallTextFile(f.textCode().Bytes(raw), f.textSelector())
	if !ok {
		t.Fatalf("%s does not read back as UTF-8", address)
	}
	return strings.Split(string(utf), "\n")
}

// withFirstLine is lines with the first line's text replaced by s, its line
// end kept.
func withFirstLine(lines []string, s string) []byte {
	out := slices.Clone(lines)
	out[0] = s
	if strings.HasSuffix(lines[0], "\r") {
		out[0] += "\r"
	}
	return []byte(strings.Join(out, "\n"))
}

// readerBriefing is s as the briefing reader prints it once the install's
// font bytes stand for it.
func readerBriefing(f *FrontEnd, s string) string {
	return strings.TrimSpace(vfs.DecodeText(EncodeInstallTextFile([]byte(s), f.textSelector())))
}

// writeInstallFilesMod writes the fixture mod id into mods: a generated
// picture over the main menu's background, and on the first game over the
// town square's; a generated sound over the bank's slot; and per language
// the install's sites.txt with its first line, the home title, made "East"
// on English and "Восток" on Russian, and on the first game mission 10's
// briefing in that language. extra is added to mod.toml and script, when
// given, is the mod's main.star.
func writeInstallFilesMod(t *testing.T, f *FrontEnd, slot int, mods, id, extra, script string) {
	t.Helper()
	dir := filepath.Join(mods, id)
	write := func(rel string, data []byte) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("mod.toml", []byte(fmt.Sprintf("id = %q\ntitle = %q\nversion = \"1.0\"\n%s", id, id, extra)))
	if script != "" {
		write("main.star", []byte(script))
	}
	write("files/common/"+modMenuPicture, synth.BMP24(solidPicture(640, 480, modMenuColour)))
	if f.Archives.Game() == base.GameROM1 {
		write("files/common/"+modSquarePicture, synth.BMP24(solidPicture(640, 480, modSquareColour)))
		write("files/en/main/text/battle/m10/briefing.txt", []byte("Go east."))
		write("files/ru/main/text/battle/m10/briefing.txt", []byte("Идите на восток."))
	}
	write("files/common/sfx/"+f.SoundBank.entries[slot], modToneWAV())
	sites := installText(t, f, SitesTextPath)
	write("files/en/main/text/sites.txt", withFirstLine(sites, "East"))
	write("files/ru/main/text/sites.txt", withFirstLine(sites, "Восток"))
}

// dropInstallFilesTexts removes the texts of the fixture mod id, leaving its
// pictures and its sound.
func dropInstallFilesTexts(t *testing.T, mods, id string) {
	t.Helper()
	for _, lang := range []string{"en", "ru"} {
		if err := os.RemoveAll(filepath.Join(mods, id, "files", lang)); err != nil {
			t.Fatal(err)
		}
	}
}

// modFilesRun is what one session shows and writes.
type modFilesRun struct {
	menu, square, worldMap []byte
	title, briefing        string
	sample                 audio.Sample
	townSAV                []byte
	hash                   uint64
	world, missionSAV      []byte
}

// modFilesFront is a front end over the install built through the
// launcher's steps with the mods res ran, none when res is nil.
func modFilesFront(t *testing.T, res *modrt.Result) *FrontEnd {
	t.Helper()
	var files mod.FileData
	if res != nil {
		files = res.Files
	}
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the mod's install files need a lawful install")
	}
	f, err := NewModFrontEnd(root, files)
	if err != nil {
		t.Fatal(err)
	}
	cleanupFrontAudio(t, f)
	if f.Archives.Game() == base.GameROM2 {
		f.MusicPlayer = &secondTestMusic{}
	}
	f.randomService().SetStreamSeed(random.TownWildlife, 1)
	f.randomService().SetStreamSeed(random.AmbientBirds, 1)
	f.SetDeterministicFrames(true)
	if res != nil {
		if err := f.SetMods(res.Rules, res.Set, false); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

// modFilesSession plays one first-game session: the main menu's background,
// the town square and world map frames and the home title, a town SAV, and
// mission 10's World and SAV after 40 ticks.
func modFilesSession(t *testing.T, res *modrt.Result, slot int) modFilesRun {
	t.Helper()
	var run modFilesRun
	f := modFilesFront(t, res)
	run.menu = slices.Clone(f.Assets.Base.Pix)
	run.sample, _ = f.SoundBank.Sample(slot)
	run.briefing = MissionObjectivesFor(f.Archives.Containers, f.Archives.Game(), 10)
	reachabilityWalkArrive(t, f, "Files Hero")
	f.Town.Won(10)
	f.arriveInTown()
	a, s := exteriorApp(t, f)
	frame := func() []byte {
		t.Helper()
		a.Draw(ebiten.NewImage(640, 480))
		pix, note, err := a.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatalf("frame %q: %v", note, err)
		}
		return pix.Pix
	}
	run.square = frame()
	run.townSAV = baseDumpSAV(t, f, false)
	f.Town.announceMission(f.Town.currentMain())
	s.Choose(3)
	v := s.WorldMapView()
	if !s.AtWorldMap() || v.Problem != "" {
		t.Fatalf("world map: at it %v, %s", s.AtWorldMap(), v.Problem)
	}
	run.title, _ = decodeSaveLabel(v.Words.WorldHomeTitle, f.textSelector())
	run.worldMap = frame()

	g := modFilesFront(t, res)
	mw, _ := heroSwingMission(t, g)
	for range 40 {
		mw.tick()
	}
	run.hash, run.world = mw.world.Hash(), marshalWorld(t, mw.world)
	run.missionSAV = baseDumpSAV(t, g, true)
	return run
}

// diffModFilesRuns names each part of got that differs from want.
func diffModFilesRuns(want, got modFilesRun) []string {
	var out []string
	for _, c := range []struct {
		name string
		same bool
	}{
		{"menu picture", bytes.Equal(want.menu, got.menu)},
		{"sound", reflect.DeepEqual(want.sample, got.sample)},
		{"briefing", want.briefing == got.briefing},
		{"town square frame", bytes.Equal(want.square, got.square)},
		{"world map frame", bytes.Equal(want.worldMap, got.worldMap)},
		{"home title", want.title == got.title},
		{"town SAV", bytes.Equal(want.townSAV, got.townSAV)},
		{"World hash", want.hash == got.hash},
		{"world bytes", bytes.Equal(want.world, got.world)},
		{"mission SAV", bytes.Equal(want.missionSAV, got.missionSAV)},
	} {
		if !c.same {
			out = append(out, c.name)
		}
	}
	return out
}

// colourCount is how many RGBA pixels of pix are c.
func colourCount(pix []byte, c color.RGBA) int {
	n := 0
	for i := 0; i+3 < len(pix); i += 4 {
		if pix[i] == c.R && pix[i+1] == c.G && pix[i+2] == c.B {
			n++
		}
	}
	return n
}

// installLanguageNames are the home title and briefing the fixture mod gives
// the install's language.
func installLanguageNames(t *testing.T) (title, briefing string) {
	t.Helper()
	switch modrt.LanguageFor(BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))) {
	case "ru":
		return "Восток", "Идите на восток."
	case "en":
		return "East", "Go east."
	}
	t.Fatal("the install's language has no fixture text")
	return "", ""
}

// The fixture mod on the first game's install: the main menu and the town
// square draw its generated pictures, the world map's home title and
// mission 10's briefing read its text in the install's language, and the
// sound bank plays its generated sound. Its texts make it a mod that can
// change state, so its SAVs carry the mod mark; these texts change none, so
// without the mark the town and mission SAV bytes and mission 10's World equal
// the session without it. Two sessions without a mod agree first, so the
// comparison can fail.
func TestReleaseModInstallFilesDrawPlayAndPrintTheModsFiles(t *testing.T) {
	f, slot := modFilesInstall(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the root holds the second game")
	}
	title, briefing := installLanguageNames(t)
	mods := t.TempDir()
	writeInstallFilesMod(t, f, slot, mods, "install-art", "", "")
	res := runTownModsFor(t, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), mods, "install-art")
	if len(res.Set.Mods) != 1 || res.Set.Mods[0].StateFree || len(res.Files.Files) != 5 {
		t.Fatalf("the mod set %+v carries %d files", res.Set.Mods, len(res.Files.Files))
	}

	plain := modFilesSession(t, nil, slot)
	if diff := diffModFilesRuns(plain, modFilesSession(t, nil, slot)); len(diff) != 0 {
		t.Fatalf("two sessions without a mod differ in %v; the comparison would prove nothing", diff)
	}
	got := modFilesSession(t, &res, slot)
	for _, raw := range []*[]byte{&got.townSAV, &got.missionSAV} {
		stripped, ids := withoutModMark(t, *raw)
		if len(ids) != 1 || ids[0] != "install-art" {
			t.Errorf("a SAV made with the text mod carries the mark %v", ids)
		}
		*raw = stripped
	}
	diff := diffModFilesRuns(plain, got)
	want := []string{"menu picture", "sound", "briefing", "town square frame", "world map frame", "home title"}
	if !slices.Equal(diff, want) {
		t.Fatalf("the mod changes %v, want %v", diff, want)
	}
	if got.title != title || got.briefing != readerBriefing(f, briefing) || plain.title == title {
		t.Errorf("home title %q (without the mod %q), briefing %q; want %q and %q", got.title, plain.title, got.briefing, title, briefing)
	}
	if n := colourCount(got.menu, modMenuColour); n != 640*480 || colourCount(plain.menu, modMenuColour) != 0 {
		t.Errorf("the menu holds %d pixels of the mod's colour", n)
	}
	square, without := colourCount(got.square, modSquareColour), colourCount(plain.square, modSquareColour)
	if square < 640*480/4 || without != 0 {
		t.Errorf("the town square frame holds %d pixels of the mod's colour, %d without the mod", square, without)
	}
	tone, err := audio.DecodeWAV(modToneWAV(), audio.DeviceRate)
	if err != nil || !reflect.DeepEqual(got.sample, tone) {
		t.Errorf("slot %d plays another sound than the mod's: %v", slot, err)
	}
	t.Logf("square: %d pixels of the mod's colour; home title %q; slot %d", square, got.title, slot)
}

// A town SAV made with a mod of pictures and a sound loads without it, and one
// made with the same files and a script that sets a rule carries a mark naming
// that mod and is refused without it.
func TestReleaseModInstallFilesSaveLoadsWithoutAStateFreeMod(t *testing.T) {
	f, slot := modFilesInstall(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the root holds the second game")
	}
	mods := t.TempDir()
	writeInstallFilesMod(t, f, slot, mods, "install-art", "", "")
	writeInstallFilesMod(t, f, slot, mods, "install-rule", "", "def init(game, settings):\n    game.rules.skill_cap = 120\n")
	dropInstallFilesTexts(t, mods, "install-art")
	dropInstallFilesTexts(t, mods, "install-rule")
	baseID := BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))
	for _, c := range []struct {
		id     string
		marked bool
	}{{"install-art", false}, {"install-rule", true}} {
		res := runTownModsFor(t, baseID, mods, c.id)
		g := modFilesFront(t, &res)
		reachabilityWalkArrive(t, g, "Files Hero")
		raw := baseDumpSAV(t, g, false)
		stripped, ids := withoutModMark(t, raw)
		if c.marked != (len(ids) == 1 && ids[0] == c.id) || !c.marked && !bytes.Equal(stripped, raw) {
			t.Errorf("%s: the mark names %v", c.id, ids)
		}
		_, err := townModReload(t, raw, "")
		if c.marked != (err != nil) || c.marked && !errors.Is(err, ErrModMark) {
			t.Errorf("%s: a LOAD without the mod: %v", c.id, err)
		}
	}
}

// Two mods serving one path: without an order the front end is refused
// naming both; with the second loading after the first, the second's file
// is read and the override is named.
func TestReleaseModInstallFilesOfTwoModsFollowTheLoadOrder(t *testing.T) {
	f, slot := modFilesInstall(t)
	mods := t.TempDir()
	writeInstallFilesMod(t, f, slot, mods, "first-art", "", "")
	writeInstallFilesMod(t, f, slot, mods, "second-art", "", "")
	baseID := BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))
	res := runTownModsFor(t, baseID, mods, "first-art", "second-art")
	if _, err := NewModFrontEnd(os.Getenv("AGAINROM_ASSETS"), res.Files); err == nil || !strings.Contains(err.Error(), "neither loads after the other") {
		t.Fatalf("unordered: %v", err)
	}
	writeInstallFilesMod(t, f, slot, mods, "second-art", "load-after = [\"first-art\"]\n", "")
	sites := installText(t, f, SitesTextPath)
	common := filepath.Join(mods, "second-art", "files", "common", "main", "text")
	if err := os.MkdirAll(common, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(common, "sites.txt"), withFirstLine(sites, "Second"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"en", "ru"} {
		if err := os.Remove(filepath.Join(mods, "second-art", "files", lang, "main", "text", "sites.txt")); err != nil {
			t.Fatal(err)
		}
	}
	res = runTownModsFor(t, baseID, mods, "first-art", "second-art")
	g := modFilesFront(t, &res)
	if title, _ := decodeSaveLabel(g.Words.WorldHomeTitle, g.textSelector()); title != "Second" {
		t.Errorf("the home title reads %q", title)
	}
	overrides := g.ModFileOverrides()
	want := `install file main/text/sites.txt: mod "second-art" (files/common/main/text/sites.txt) overrides mod "first-art"`
	if !slices.ContainsFunc(overrides, func(o string) bool { return strings.HasPrefix(o, want) }) {
		t.Errorf("overrides %q", overrides)
	}
}

// The fixture mod on the second game's install: the main menu draws its
// picture, the home title and mission 10's briefing, read out of the second
// game's mission text file, read its text in the install's language, and
// the bank plays its sound. Without the mod each equals the install's.
func TestReleaseSecondModInstallFilesDrawPlayAndPrintTheModsFiles(t *testing.T) {
	secondGameRoot(t)
	f, slot := modFilesInstall(t)
	title, _ := installLanguageNames(t)
	mods := t.TempDir()
	writeInstallFilesMod(t, f, slot, mods, "install-art", "", "")
	address := mainPrefix + "text/mission10.txt"
	lines := installText(t, f, address)
	text := strings.Join(lines, "\n")
	at := strings.Index(strings.ToLower(text), "#briefing")
	if at < 0 {
		t.Fatalf("%s holds no briefing section", address)
	}
	body := at + len("#briefing") + 2
	end := strings.IndexByte(text[body:], '#')
	if end < 0 {
		t.Fatalf("%s: the briefing section has no end", address)
	}
	for lang, briefing := range map[string]string{"en": "Go east.", "ru": "Идите на восток."} {
		edited := text[:body] + briefing + text[body+end:]
		if err := os.WriteFile(filepath.Join(mods, "install-art", "files", lang, filepath.FromSlash(address)), []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, briefing := installLanguageNames(t)
	res := runTownModsFor(t, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), mods, "install-art")
	for _, c := range []struct {
		res                    *modrt.Result
		menu                   int
		title, briefing, sound bool
	}{{nil, 0, false, false, false}, {&res, 640 * 480, true, true, true}} {
		g := modFilesFront(t, c.res)
		gotTitle, _ := decodeSaveLabel(g.Words.WorldHomeTitle, g.textSelector())
		gotBriefing := MissionObjectivesFor(g.Archives.Containers, g.Archives.Game(), 10)
		sample, _ := g.SoundBank.Sample(slot)
		tone, _ := audio.DecodeWAV(modToneWAV(), audio.DeviceRate)
		if colourCount(g.Assets.Base.Pix, modMenuColour) != c.menu || (gotTitle == title) != c.title ||
			(gotBriefing == readerBriefing(g, briefing)) != c.briefing || reflect.DeepEqual(sample, tone) != c.sound {
			t.Errorf("mod %v: menu %d pixels, title %q, briefing %q, the mod's sound %v",
				c.res != nil, colourCount(g.Assets.Base.Pix, modMenuColour), gotTitle, gotBriefing, reflect.DeepEqual(sample, tone))
		}
	}
}

// A mod that serves only the body list changes the hero's class, so it is
// not state-free: its SAV carries the mod mark, is refused without the mod
// and loads with it. A text is state, a picture is not (see
// TestReleaseModInstallFilesSaveLoadsWithoutAStateFreeMod).
func TestReleaseModInstallFilesTextModIsMarkedAndItsSaveNeedsTheMod(t *testing.T) {
	f, _ := modFilesInstall(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the root holds the second game")
	}
	lines := installText(t, f, HeroPictureAddress)
	body := slices.Clone(lines)
	for i, l := range body {
		if strings.TrimSpace(l) == "" {
			continue
		}
		body[i] = "unarmed"
		if strings.HasSuffix(l, "\r") {
			body[i] += "\r"
		}
	}
	mods := t.TempDir()
	id := "body-text"
	if err := os.MkdirAll(filepath.Join(mods, id), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mods, id, "mod.toml"), []byte(fmt.Sprintf("id = %q\ntitle = %q\nversion = \"1.0\"\n", id, id)), 0o644); err != nil {
		t.Fatal(err)
	}
	text := filepath.Join(mods, id, "files", "common", filepath.FromSlash(HeroPictureAddress))
	if err := os.MkdirAll(filepath.Dir(text), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(text, []byte(strings.Join(body, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	res := runTownModsFor(t, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), mods, id)
	if len(res.Files.Files) != 1 || res.Set.Mods[0].StateFree {
		t.Fatalf("the text mod serves %d files and is state-free %v", len(res.Files.Files), res.Set.Mods[0].StateFree)
	}
	g := modFilesFront(t, &res)
	if before, after := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)[0].Class, MissionParty(g.StartWeapon.Value(), g.Bodies, g.Table)[0].Class; before == after {
		t.Fatalf("the hero's class stays %d under the body list", before)
	}
	reachabilityWalkArrive(t, g, "Files Hero")
	raw := baseDumpSAV(t, g, false)
	if _, ids := withoutModMark(t, raw); len(ids) != 1 || ids[0] != id {
		t.Fatalf("the SAV's mark names %v, want [%s]", ids, id)
	}
	if _, err := townModReload(t, raw, ""); err == nil || !errors.Is(err, ErrModMark) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}
	with := modFilesFront(t, &res)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, load := with.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	if _, town, err := load("town.sav"); err != nil || !town {
		t.Fatalf("a LOAD with the mod: town %v: %v", town, err)
	}
}
