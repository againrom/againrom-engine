package game

import (
	"bytes"
	"cmp"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/locale"
	"againrom/pkg/mod"

	"github.com/hajimehoshi/ebiten/v2"
)

// townLayerKeys are the keys the towns decoder reads, as its own refusal of
// an unknown key lists them.
func townLayerKeys(t *testing.T) []string {
	t.Helper()
	_, _, err := mod.ParseTowns("probe", mod.TownsFile, []byte("[[town]]\nid = 1\nno-such-key = 1\n"))
	m := regexp.MustCompile(`\(keys: ([^)]*)\)`).FindStringSubmatch(err.Error())
	if m == nil {
		t.Fatalf("the decoder's unknown-key refusal lists no keys: %v", err)
	}
	return strings.Split(m[1], ", ")
}

// releaseBaseDump writes the base dump of the install into mods/base-dump
// and returns the towns file it wrote.
func releaseBaseDump(t *testing.T, f *FrontEnd, mods string) []byte {
	t.Helper()
	report, err := f.WriteBaseDump(filepath.Join(mods, "base-dump"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"mod.toml", "main.star", mod.TownsFile}; !slices.Equal(report.Files, want) || len(report.Notes) != 0 {
		t.Fatalf("report %+v, want files %v and no note", report, want)
	}
	towns, err := os.ReadFile(filepath.Join(mods, "base-dump", filepath.FromSlash(mod.TownsFile)))
	if err != nil {
		t.Fatal(err)
	}
	return towns
}

// The base dump of the install lists its town with every key the towns
// decoder reads. The name is the home title the world map draws, as UTF-8
// that encodes back to the install's bytes: Cyrillic on the Russian
// install.
func TestReleaseBaseDumpListsTheTownWithEveryKey(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	towns := releaseBaseDump(t, f, t.TempDir())
	records, _, err := mod.ParseTowns("base-dump", mod.TownsFile, towns)
	if err != nil || len(records) != 1 {
		t.Fatalf("%d records: %v\n%s", len(records), err, towns)
	}
	r := records[0]
	have := []string{"id"}
	for _, field := range r.Fields {
		have = append(have, field.Name)
	}
	if keys := townLayerKeys(t); !slices.Equal(have, keys) {
		t.Fatalf("town %d holds the keys %v, want the decoder's %v", r.Entry.ID, have, keys)
	}
	back, err := encodeSaveLabel(r.Entry.Name, f.textSelector())
	if err != nil || back != f.Words.WorldHomeTitle {
		t.Fatalf("name %q encodes to % x (%v), want the home title % x", r.Entry.Name, back, err, f.Words.WorldHomeTitle)
	}
	l, _ := locale.ByBaseSuffix(BaseID(InspectInstall(f.Archives.Root)))
	if cyrillic := strings.ContainsFunc(r.Entry.Name, func(c rune) bool { return unicode.Is(unicode.Cyrillic, c) }); cyrillic != (l.Code == "ru") {
		t.Fatalf("name %q on the %s install", r.Entry.Name, l.Code)
	}
	t.Logf("%s", towns)
}

// baseDumpRun is what a player sees and what is saved in one session: the
// town square and the world map, the home scroll's title, the town SAV, and
// mission 10's World hash, world bytes and mission SAV after 40 ticks.
type baseDumpRun struct {
	square, worldMap []byte
	title            string
	townSAV          []byte
	hash             uint64
	world            []byte
	missionSAV       []byte
}

// baseDumpFront is the install's front end with the town layer of mod id of
// mods applied. With launcher set it takes the mod set first, as the
// launcher does, so its SAVs carry the mod mark.
func baseDumpFront(t *testing.T, mods, id string, launcher bool) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if id == "" {
		return f
	}
	res := runTownModsFor(t, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), mods, id)
	if launcher {
		if err := f.SetMods(res.Rules, res.Set, false); err != nil {
			t.Fatal(err)
		}
	}
	if overrides, err := f.SetModTowns(res.Towns); err != nil || len(overrides) != 0 {
		t.Fatalf("town layer of %s: %v, overrides %v", id, err, overrides)
	}
	return f
}

func baseDumpSAV(t *testing.T, f *FrontEnd, mission bool) []byte {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	name, err := save(mission)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("SAVE wrote %q: %v", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// baseDumpSession plays one session with the mod id of mods, none when id
// is empty.
func baseDumpSession(t *testing.T, mods, id string, launcher bool) baseDumpRun {
	t.Helper()
	var run baseDumpRun
	f := baseDumpFront(t, mods, id, launcher)
	reachabilityWalkArrive(t, f, "Dump Hero")
	f.Town.Won(10)
	f.arriveInTown()
	a, s := exteriorApp(t, f)
	frame := func(name string) []byte {
		t.Helper()
		a.Draw(ebiten.NewImage(640, 480))
		pix, note, err := a.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatalf("frame %q: %v", note, err)
		}
		if dir := townModFramesDir(t, f); dir != "" {
			writeTownModPNG(t, filepath.Join(dir, "base-dump-"+cmp.Or(id, "none")+map[bool]string{true: "-launcher"}[launcher]+"-"+name+".png"), pix)
		}
		return pix.Pix
	}
	run.square = frame("square")
	run.townSAV = baseDumpSAV(t, f, false)
	// The tavern's announcement of the next mission opens the gates.
	f.Town.announceMission(f.Town.currentMain())
	s.Choose(3)
	v := s.WorldMapView()
	if !s.AtWorldMap() || v.Problem != "" {
		t.Fatalf("world map: at it %v, %s", s.AtWorldMap(), v.Problem)
	}
	run.title, run.worldMap = v.Words.WorldHomeTitle, frame("world-map")

	g := baseDumpFront(t, mods, id, launcher)
	mw, _ := heroSwingMission(t, g)
	for range 40 {
		mw.tick()
	}
	run.hash, run.world = mw.world.Hash(), marshalWorld(t, mw.world)
	run.missionSAV = baseDumpSAV(t, g, true)
	return run
}

// withoutModMark is raw with the mod mark taken out, and the mods the mark
// names; it fails on a mark that carries more than the mod set.
func withoutModMark(t *testing.T, raw []byte) ([]byte, []string) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark, ok, err := readModMark(doc)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	if ok {
		if len(mark.Domain) != 0 || len(mark.Items) != 0 || len(mark.Layers) != 0 {
			t.Fatalf("the mod mark carries values: %+v", mark)
		}
		for _, m := range mark.Mods {
			ids = append(ids, m.ID)
		}
	}
	sav.ClearNativeMods(&doc.State)
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out, ids
}

// diffBaseDumpRuns names each part of got that differs from want.
func diffBaseDumpRuns(want, got baseDumpRun) []string {
	var out []string
	for _, c := range []struct {
		name string
		same bool
	}{
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

// A mod made of the unchanged base dump, or of its one record, changes
// nothing: the town square and world map frames, the home title, the town
// and mission SAV bytes and mission 10's World hash and bytes equal the
// session without it. Through the launcher's steps the SAVs differ only by
// the mod mark, which names the mod and carries no value. The loss control:
// one edited key, the name, shows on the home scroll and fails the
// comparison.
func TestReleaseBaseDumpAsAModChangesNothing(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	mods := t.TempDir()
	towns := releaseBaseDump(t, releaseFront(t), mods)
	plain := baseDumpSession(t, "", "", false)
	if diff := diffBaseDumpRuns(plain, baseDumpSession(t, "", "", false)); len(diff) != 0 {
		t.Fatalf("two sessions without a mod differ in %v; the comparison would prove nothing", diff)
	}
	if len(plain.townSAV) == 0 || len(plain.missionSAV) == 0 || len(plain.square) == 0 || len(plain.worldMap) == 0 {
		t.Fatal("the session wrote no SAV or frame")
	}
	t.Logf("town SAV %d bytes, mission SAV %d bytes, world %d bytes, hash %#x, title % x", len(plain.townSAV), len(plain.missionSAV), len(plain.world), plain.hash, plain.title)

	first := strings.Index(string(towns), "\n[[town]]\n")
	record := string(towns[first+1:])
	if next := strings.Index(record, "\n[[town]]\n"); next >= 0 {
		record = record[:next+1]
	}
	writeTownMod(t, mods, "one-town", record)
	for _, id := range []string{"base-dump", "one-town"} {
		if diff := diffBaseDumpRuns(plain, baseDumpSession(t, mods, id, false)); len(diff) != 0 {
			t.Errorf("%s: the session differs in %v", id, diff)
		}
		got := baseDumpSession(t, mods, id, true)
		for _, raw := range []*[]byte{&got.townSAV, &got.missionSAV} {
			stripped, ids := withoutModMark(t, *raw)
			if !slices.Equal(ids, []string{id}) {
				t.Errorf("%s: the mod mark names %v", id, ids)
			}
			*raw = stripped
		}
		if diff := diffBaseDumpRuns(plain, got); len(diff) != 0 {
			t.Errorf("%s through the launcher: the session differs in %v beside the mod mark", id, diff)
		}
	}

	edited := strings.Replace(record, "\nname ", "\nname        = \"Dumped\"\n#", 1)
	if edited == record {
		t.Fatalf("the record holds no name:\n%s", record)
	}
	writeTownMod(t, mods, "edited", edited)
	got := baseDumpSession(t, mods, "edited", false)
	diff := diffBaseDumpRuns(plain, got)
	if got.title != "Dumped" || !slices.Equal(diff, []string{"world map frame", "home title"}) {
		t.Fatalf("an edited name: title %q, differences %v, want the world map frame and the home title", got.title, diff)
	}
}

// The second game's base dump lists its three towns, says in its report and
// its towns file that the second game reads no towns layer, and loads as a
// mod that layers nothing.
func TestReleaseSecondTownBaseDumpListsItsThreeTowns(t *testing.T) {
	f := secondGameFront(t)
	mods := t.TempDir()
	report, err := f.WriteBaseDump(filepath.Join(mods, "base-dump"))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Notes) != 1 || !strings.Contains(report.Notes[0], "the second game reads no towns layer") {
		t.Fatalf("notes %q", report.Notes)
	}
	towns, err := os.ReadFile(filepath.Join(mods, "base-dump", filepath.FromSlash(mod.TownsFile)))
	if err != nil {
		t.Fatal(err)
	}
	records, _, err := mod.ParseTowns("base-dump", mod.TownsFile, towns)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int
	for _, r := range records {
		ids = append(ids, r.Entry.ID)
	}
	if !slices.Equal(ids, []int{1, 2, 3}) || !strings.Contains(string(towns), "# "+report.Notes[0]) {
		t.Fatalf("towns %v:\n%s", ids, towns)
	}
	res := runTownModsFor(t, report.Base, mods, "base-dump")
	if !res.Towns.Empty() {
		t.Fatalf("the second game's dump layers %d towns", len(res.Towns.Records))
	}
	t.Logf("%s", towns)
}
