package game

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/vfs"
)

// recordingSource is a synthetic install that logs the addresses read.
type recordingSource struct {
	files map[string]string
	log   []string
}

func (r *recordingSource) ReadFile(name string) ([]byte, error) {
	r.log = append(r.log, name)
	if b, ok := r.files[name]; ok {
		return []byte(b), nil
	}
	return nil, fs.ErrNotExist
}

func (r *recordingSource) take() string {
	out := strings.Join(r.log, ",")
	r.log = nil
	return out
}

// TestMissionTextTownAddressesAreRecorded holds the town text addresses.
func TestMissionTextTownAddressesAreRecorded(t *testing.T) {
	var got []string
	add := func(name string, path string, ok bool) {
		got = append(got, fmt.Sprintf("%s=%q/%v", name, path, ok))
	}
	for _, c := range [][2]int{{0, 1}, {7, 30}, {31, 5}, {35, 12}, {-1, 3}, {4, 0}, {4, -2}, {100, 7}} {
		p, ok := TownNPCTextPath(c[0], c[1])
		add(fmt.Sprintf("npc(%d,%d)", c[0], c[1]), p, ok)
	}
	for _, c := range [][3]int{{7, 30, 5}, {7, 0, 5}, {7, 0, 0}, {0, 0, 9}} {
		p, ok := tavernTalkTextPath(c[0], c[1], c[2])
		add(fmt.Sprintf("talk(%d,%d,%d)", c[0], c[1], c[2]), p, ok)
	}
	for typ := -1; typ <= 16; typ++ {
		p, ok := TownMercenaryTextPath(typ)
		add(fmt.Sprintf("merc(%d)", typ), p, ok)
	}
	for _, b := range []TownBuilding{TownShop, TownSchool, TownTavern} {
		for _, m := range []int{-1, 0, 1, 31, 34} {
			p, ok := TownBuildingTextPath(b, m)
			add(fmt.Sprintf("building(%v,%d)", b, m), p, ok)
		}
	}
	want := strings.Join(recordedTownAddresses, "\n")
	if have := strings.Join(got, "\n"); have != want {
		t.Fatalf("town addresses changed:\n%s", have)
	}
}

// TestMissionTextAddressesAreRecorded holds the addresses and the text the
// first and the second game's readers take for a mission, an event and a
// failure.
func TestMissionTextAddressesAreRecorded(t *testing.T) {
	second := "#briefing\r\nSecond brief\r\n#event2\r\nSecond event two\r\n#failure2\r\nFailure two\r\n#failure3\r\nFailure three\r\n"
	src := &recordingSource{files: map[string]string{
		"main/text/battle/m10/briefing.txt": "  First brief \r\n",
		"main/text/battle/m10/event02.txt":  "First event two",
		"main/text/mission10.txt":           second,
	}}
	var got []string
	add := func(name, value string) { got = append(got, fmt.Sprintf("%s -> %q [%s]", name, value, src.take())) }
	path, ok := EventTextPath(10, 2)
	got = append(got, fmt.Sprintf("EventTextPath(10,2)=%q/%v", path, ok))
	path, ok = EventTextPath(10, 0)
	got = append(got, fmt.Sprintf("EventTextPath(10,0)=%q/%v", path, ok))
	for _, game := range []base.Game{base.GameROM1, base.GameROM2} {
		add(string(game)+" objectives 10", MissionObjectivesFor(src, game, 10))
		add(string(game)+" objectives 0", MissionObjectivesFor(src, game, 0))
		b, found := ReadEventTextFor(src, game, 10, 2)
		add(fmt.Sprintf("%s event 10,2 %v", game, found), string(b))
		b, found = ReadEventTextFor(src, game, 10, 3)
		add(fmt.Sprintf("%s event 10,3 %v", game, found), string(b))
		b, found = ReadEventTextFor(src, game, 11, 2)
		add(fmt.Sprintf("%s event 11,2 %v", game, found), string(b))
	}
	code := InstallTextCode(src, base.GameROM2.Edition())
	src.take()
	add("failure 10", fmt.Sprintf("%q", secondGameFailureText(src, code, 10)))
	add("failure 11", fmt.Sprintf("%q", secondGameFailureText(src, code, 11)))
	add("failure 0", fmt.Sprintf("%q", secondGameFailureText(src, code, 0)))
	want := strings.Join(recordedMissionText, "\n")
	if have := strings.Join(got, "\n"); have != want {
		t.Fatalf("mission text changed:\n%s", have)
	}
}

func pageFixture(t *testing.T) (dir string, m mod.Mission) {
	t.Helper()
	dir = t.TempDir()
	write := func(name, body string) {
		writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(name)), []byte(body))
	}
	write("map.alm", string(synth.ALM(synth.ALMOptions{Width: 4, Height: 4})))
	write("map.text/en/event01.txt", "Own one 中")
	write("map.text/ru/event01.txt", "Other one 中")
	write("map.text/ru/event03.txt", "Other three 中")
	write("map.text/ru/event07.txt", "Other seven 中")
	write("map.text/event04.txt", "Common four 中")
	write("map.text/en/event02.txt", "Own two 中")
	write("texts/en/event05.txt", "Declared five 中")
	d, err := mod.ParseMissions("m", mod.MissionsFile, []byte("[[mission]]\nkey = \"map\"\nkind = \"replace\"\nnumber = 20\nmap = \"map.alm\"\ntexts = \"texts/{lang}\"\n"), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	m = d.Missions[0]
	if err := m.ReadFiles(dir); err != nil {
		t.Fatal(err)
	}
	return dir, m
}

func pageNames(pages map[string][]byte) string {
	var out []string
	for name, b := range pages {
		out = append(out, name+"="+string(b))
	}
	sort.Strings(out)
	return strings.Join(out, "|")
}

// TestMissionPagesEachReaderIsRecorded holds the pages a mod mission's start,
// the map check and the editor see for one fixture: a map folder, a mod
// folder, the install's pages, an other-language page and a missing page.
func TestMissionPagesEachReaderIsRecorded(t *testing.T) {
	dir, m := pageFixture(t)
	var got []string

	// What the mod loader holds.
	other := []string{}
	for name := range m.OtherLanguageTexts {
		other = append(other, name)
	}
	sort.Strings(other)
	got = append(got, "loaded "+pageNames(m.TextFiles)+" other="+strings.Join(other, ","))

	// The mission start: the pages served into the install's tree.
	install := t.TempDir()
	for name, body := range map[string]string{"scenario/26.alm": "x", "main/text/battle/m25/event02.txt": "Installed two", "main/text/battle/m25/event03.txt": "Installed three", "main/text/battle/m25/event07.txt": "Installed seven"} {
		writeTestPlayInput(t, filepath.Join(install, filepath.FromSlash(name)), []byte(body))
	}
	tree, err := vfs.Open(nil, []string{install})
	if err != nil {
		t.Fatal(err)
	}
	f := guestFixtureFront(t)
	f.Archives = &Archives{Containers: tree}
	guest := guestDecl()
	guest.TextFiles, guest.OtherLanguageTexts, guest.Texts = m.TextFiles, m.OtherLanguageTexts, m.Texts
	if _, err := f.SetModMissions(mod.MissionData{Missions: []mod.Mission{guest}}); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 8; n++ {
		address := fmt.Sprintf("main/text/battle/m25/event%02d.txt", n)
		b, err := f.Archives.Containers.ReadFile(address)
		got = append(got, fmt.Sprintf("start %s=%q err=%v", address, b, err != nil))
	}

	// The map check of the loaded mission, and of the same map opened in the
	// editor over an install holding mission 20's page two.
	x := newTestPlayFixture(t).x
	finding := func(rows []string) string { sort.Strings(rows); return strings.Join(rows, "|") }
	var rows []string
	for _, r := range x.CheckModMap(m, filepath.Join(dir, "map.alm")) {
		rows = append(rows, r.Rule+":"+r.Text)
	}
	got = append(got, "check "+finding(rows))

	source := filepath.Join(dir, "map.alm")
	x.mod = &editorMod{dir: dir, id: "m", missions: mod.MissionData{Missions: []mod.Mission{m}}}
	played, err := x.testPlayMission(source, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, "editor "+pageNames(played.TextFiles))
	model, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 4, Height: 4}))
	if err != nil {
		t.Fatal(err)
	}
	rows = nil
	rowsOf, err := x.CheckMap(model, source)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rowsOf {
		rows = append(rows, r.Rule+":"+r.Text)
	}
	got = append(got, "editor check "+finding(rows))

	// A map beside no mod entry: the map's own folder and the install.
	x.mod = nil
	loose := filepath.Join(t.TempDir(), "loose.alm")
	writeTestPlayInput(t, loose, synth.ALM(synth.ALMOptions{Width: 4, Height: 4}))
	writeTestPlayInput(t, filepath.Join(filepath.Dir(loose), "loose.text", "ru", "event09.txt"), []byte("Other nine"))
	writeTestPlayInput(t, filepath.Join(filepath.Dir(loose), "loose.text", "en", "event02.txt"), []byte("Own loose two"))
	played, err = x.testPlayMission("scenario/20.alm", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, "install "+pageNames(played.TextFiles))
	played, err = x.testPlayMission(loose, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, "loose "+pageNames(played.TextFiles))
	want := strings.Join(recordedMissionPages, "\n")
	if have := strings.Join(got, "\n"); have != want {
		t.Fatalf("mission pages changed:\n%s", have)
	}
}
