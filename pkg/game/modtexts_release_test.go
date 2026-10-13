package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// missionTextCorpus is every installed file under text/battle/m<number>/: the
// event and tip pages, title and briefing, and the world map's briefmap.
func missionTextCorpus(t *testing.T, f *FrontEnd) map[string][]byte {
	t.Helper()
	prefix := mainPrefix + "text/battle/m"
	out := map[string][]byte{}
	for _, e := range f.Archives.Containers.Entries() {
		low := strings.ToLower(e.Address)
		if !strings.HasPrefix(low, prefix) || !strings.HasSuffix(low, ".txt") {
			continue
		}
		b, err := f.Archives.Containers.ReadFile(e.Address)
		if err != nil {
			t.Fatal(err)
		}
		out[e.Address] = b
	}
	return out
}

// TestReleaseMissionTextsRoundTripThroughUTF8 decodes every installed mission
// text to UTF-8 and encodes it back: the install's bytes return exactly.
func TestReleaseMissionTextsRoundTripThroughUTF8(t *testing.T) {
	f := releaseFront(t)
	selector := LanguageSelector(f.Archives.Containers)
	corpus := missionTextCorpus(t, f)
	kinds := map[string]int{}
	files, bytesTotal, bad, withHigh := 0, 0, 0, 0
	distinct := map[byte]bool{}
	for addr, b := range corpus {
		name := addr[strings.LastIndex(addr, "/")+1:]
		kind := strings.TrimRight(name[:len(name)-4], "0123456789")
		kinds[kind]++
		files++
		bytesTotal += len(b)
		if !isASCII(b) {
			withHigh++
			highSet := map[byte]bool{}
			for _, c := range b {
				if c >= 0x80 {
					highSet[c] = true
				}
			}
			for c := range highSet {
				distinct[c] = true
			}
			if utf8.Valid(b) {
				t.Errorf("%s holds bytes above 0x7f and is valid UTF-8, so a mod copy would be read as UTF-8", addr)
			}
		}
		u, ok := DecodeInstallTextFile(b, selector)
		if !ok {
			bad++
			t.Logf("no UTF-8 form: %s (%d bytes)", addr, len(b))
			continue
		}
		if !utf8.Valid(u) || !bytes.Equal(EncodeInstallTextFile(u, selector), b) {
			t.Errorf("%s does not round trip", addr)
		}
	}
	t.Logf("selector %d: %d files, %d bytes, kinds %v, %d without a UTF-8 form, %d with bytes above 0x7f (%d distinct values)", selector, files, bytesTotal, kinds, bad, withHigh, len(distinct))
	if files == 0 || kinds["event"] == 0 || kinds["tips"] == 0 || kinds["title"] == 0 || kinds["briefmap"] == 0 {
		t.Fatalf("the corpus lacks a kind: %v", kinds)
	}
}

func isASCII(b []byte) bool {
	for _, c := range b {
		if c >= 0x80 {
			return false
		}
	}
	return true
}

// modTextUTF8 is a mod page typed in UTF-8; modTextBytes[selector] is what the
// install of that language holds for it (the DOS layout at selector 1, the
// Windows layout at selector 0), written out byte by byte.
const modTextTag = "<part=1>\r\n"

const modTextUTF8 = "<part=1>\r\nAmbush ahead: Засада\r\n"

var modTextBytes = map[int][]byte{
	0: []byte("<part=1>\r\nAmbush ahead: \xc7\xe0\xf1\xe0\xe4\xe0\r\n"),
	1: []byte("<part=1>\r\nAmbush ahead: \x87\xa0\xe1\xa0\xa4\xa0\r\n"),
}

// writeModTexts writes UTF-8 event and briefmap files into the texts folder of
// the replace mod under the mods folder root.
func writeModTexts(t *testing.T, root string, event int) string {
	t.Helper()
	lang := modrt.LanguageFor(BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))))
	dir := filepath.Join(root, replaceModID, "mission", lang, "text")
	for name, body := range map[string]string{
		fmt.Sprintf("event%02d.txt", event): modTextUTF8,
		"briefmap.txt":                      "Ambush ahead: Засада\r\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// worldMapMissionText is what the world map's scroll for mission shows.
func worldMapMissionText(t *testing.T, f *FrontEnd, mission int) (title, brief string) {
	t.Helper()
	old := f.Town.available
	f.Town.available = map[int]bool{mission: true}
	defer func() { f.Town.available = old }()
	s := f.TownScreen().(*townScreen)
	s.enterWorldMap()
	for _, m := range s.WorldMapView().Missions {
		if m.Number == mission {
			return m.Title, m.Briefing
		}
	}
	t.Fatalf("the world map shows no scroll for mission %d", mission)
	return "", ""
}

// TestReleaseModMissionTextsAreEncodedForTheInstall: a replacing mission whose
// texts folder holds UTF-8 files shows, through the live mission's event path
// and the world map, the install's own bytes for them; the same route without
// the mod shows the installed texts unchanged.
func TestReleaseModMissionTextsAreEncodedForTheInstall(t *testing.T) {
	root := replaceMod(t)
	plain := guestFront(t, "")
	replaceRoute(t, plain)
	var event int
	for _, r := range plain.live.mission.state.Raises {
		if r.Event > 0 && r.Event < 100 && int32(r.Event) != ReservedMessageNumber {
			event = int(r.Event)
			break
		}
	}
	if event == 0 {
		t.Fatal("mission 20 raises no event")
	}
	selector := LanguageSelector(plain.Archives.Containers)
	want := modTextBytes[selector]
	if want == nil {
		t.Fatalf("no expected bytes for selector %d", selector)
	}

	// Loss control: no mod, the installed event page and briefmap.
	addr, _ := EventTextPath(20, event)
	installed, installedErr := plain.Archives.Containers.ReadFile(addr)
	shown := func(f *FrontEnd) ([]byte, bool) {
		if !f.live.openDialogue(event) {
			return nil, false
		}
		return f.live.mission.payload, true
	}
	if got, ok := shown(plain); installedErr == nil && (!ok || !bytes.Equal(got, installed)) {
		t.Fatalf("loss control: event %d shows %q, the install holds %q", event, got, installed)
	}
	_, brief := worldMapMissionText(t, plain, 20)
	rawBrief, err := plain.Archives.Containers.ReadFile("main/text/battle/m20/briefmap.txt")
	if err != nil || brief != string(rawBrief) {
		t.Fatalf("loss control: world map briefing %q, installed %q (%v)", brief, rawBrief, err)
	}

	writeModTexts(t, root, event)
	f := guestFront(t, root, replaceModID)
	replaceRoute(t, f)
	got, ok := shown(f)
	if !ok || !bytes.Equal(got, want) {
		t.Fatalf("event %d shows % x (shown %v), want % x", event, got, ok, want)
	}
	body, ok := dialoguePart(got, 1, f.live.mission.dialogueAudience)
	if !ok || !strings.HasPrefix(body, "Ambush ahead: ") {
		t.Fatalf("the dialogue part is %q (%v)", body, ok)
	}
	_, brief = worldMapMissionText(t, f, 20)
	if brief != string(want[len(modTextTag):]) {
		t.Fatalf("world map briefing % x", brief)
	}
}

// TestReleaseAssembleModMissionWritesUTF8Texts: the texts cmd/modmission
// copies out are UTF-8, and the game's encoding of each returns the
// installed file's bytes.
func TestReleaseAssembleModMissionWritesUTF8Texts(t *testing.T) {
	install := os.Getenv("AGAINROM_ASSETS")
	if install == "" {
		t.Skip("no AGAINROM_ASSETS: the copy needs a lawful install")
	}
	f := releaseFront(t)
	selector := LanguageSelector(f.Archives.Containers)
	lang := modrt.LanguageFor(BaseID(InspectInstall(install)))
	files, high := 0, 0
	for _, mission := range []int{10, 20, 110} {
		out := t.TempDir()
		if err := os.WriteFile(filepath.Join(out, "mod.toml"), []byte("id = \"x\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		written, err := AssembleModMission(install, lang, mission, out)
		if err != nil {
			t.Fatal(err)
		}
		for _, rel := range written {
			name := rel[strings.LastIndex(rel, "/")+1:]
			if !strings.HasSuffix(name, ".txt") {
				continue
			}
			got, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatal(err)
			}
			want, err := f.Archives.Containers.ReadFile(fmt.Sprintf("%stext/battle/m%d/%s", mainPrefix, mission, name))
			if err != nil {
				t.Fatal(err)
			}
			if !utf8.Valid(got) || !bytes.Equal(EncodeInstallTextFile(got, selector), want) {
				t.Errorf("mission %d %s: the copy does not encode back to the installed bytes", mission, name)
			}
			files++
			if !isASCII(got) {
				high++
			}
		}
	}
	if files < 6 || (selector == 1 && high == 0) {
		t.Fatalf("copied %d text files, %d with non-ASCII text", files, high)
	}
	t.Logf("selector %d: %d text files copied, %d with non-ASCII text", selector, files, high)
}

// TestReleaseMapReportNamesAMessageTheModGivesNoPage: mission 10's installed
// map, saved into a guest mission of a mod, raises messages whose pages the
// copied texts folder holds, so the report is quiet; with the pages removed it
// names each message once.
func TestReleaseMapReportNamesAMessageTheModGivesNoPage(t *testing.T) {
	root := guestMods(t, guestModID)
	install := os.Getenv("AGAINROM_ASSETS")
	lang := modrt.LanguageFor(BaseID(InspectInstall(install)))
	dir := filepath.Join(root, guestModID)
	host := filepath.Join(dir, "mission", lang, "map.alm")
	raw, err := os.ReadFile(host)
	if err != nil {
		t.Fatal(err)
	}
	x, err := NewMapInspector(install)
	if err != nil {
		t.Fatal(err)
	}
	if err := x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	model, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	rule := func() []ui.InspectionFinding {
		rows, err := x.CheckMap(model, host)
		if err != nil {
			t.Fatal(err)
		}
		return ofRule(rows, "mod-message-text")
	}
	if rows := rule(); len(rows) != 0 {
		t.Fatalf("the copied texts leave %d messages without a page: %+v", len(rows), rows)
	}
	pages, _ := filepath.Glob(filepath.Join(dir, "mission", lang, "text", "event*.txt"))
	if len(pages) == 0 {
		t.Fatal("the copy holds no event page")
	}
	for _, p := range pages {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}
	rows := rule()
	if want := distinctRaises(t, x, raw); len(want) == 0 || len(rows) != len(want) {
		t.Fatalf("without %d pages the report holds %d rows for messages %v", len(pages), len(rows), want)
	}
	for _, r := range rows {
		if r.Error {
			t.Fatalf("an error row: %+v", r)
		}
	}
}

// distinctRaises is the distinct message numbers the map's script sends, as
// the report's rule compiles them.
func distinctRaises(t *testing.T, x *MapInspector, raw []byte) []int32 {
	t.Helper()
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	_, rep, err := tableFiles(x.table).compileScript(m, campaignScriptRefs(m, x.table, nil, 0))
	if err != nil {
		t.Fatal(err)
	}
	var out []int32
	for _, r := range rep.Raises {
		if r.Event > 0 && r.Event != ReservedMessageNumber && !slices.Contains(out, r.Event) {
			out = append(out, r.Event)
		}
	}
	return out
}
