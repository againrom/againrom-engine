package game

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// globalsMillFolder is the committed mod whose map replaces mission 10 and
// whose triggers and reply read, set and add its three globals.
const globalsMillFolder = "testdata/globals-mill"

// The global nodes' opcodes, as the op table states them.
const (
	opReadGlobal = uint32(scriptprofile.ReadGlobalOp)
	opSetGlobal  = uint32(scriptprofile.SetGlobalOp)
	opAddGlobal  = uint32(scriptprofile.AddGlobalOp)
)

// globalsMillMap is the fixture's map: a 40x40 .almx with one unit of side 1.
// Trigger "Setup" sets global 1 to 1, adds 2 to globals 2 and 3 and opens
// the mill dialogue, whose first reply adds 3 to global 2 (action 21).
// Trigger "Paid" sets variable 50 to 1 once global 2 is 10, global 1 is 1
// and global 3 is below 0. global names the id nodes 11 and 1 use.
func globalsMillMap(t *testing.T, global uint32) []byte {
	t.Helper()
	conditions := []scriptNode{
		{label: "Quest", opcode: opReadGlobal, id: 1, values: []uint32{global}, tags: []uint32{1}},
		{label: "One", opcode: 0x10002, id: 2, values: []uint32{1}, tags: []uint32{1}},
		{label: "Coins", opcode: opReadGlobal, id: 3, values: []uint32{2}, tags: []uint32{1}},
		{label: "Ten", opcode: 0x10002, id: 4, values: []uint32{10}, tags: []uint32{1}},
		{label: "Wrap", opcode: opReadGlobal, id: 5, values: []uint32{3}, tags: []uint32{1}},
		{label: "Zero", opcode: 0x10002, id: 6, values: []uint32{0}, tags: []uint32{1}},
	}
	actions := []scriptNode{
		{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}},
		{label: "Begin", opcode: opSetGlobal, id: 11, values: []uint32{global, 1}, tags: []uint32{1, 1}},
		{label: "Coins", opcode: opAddGlobal, id: 12, values: []uint32{2, 2}, tags: []uint32{1, 1}},
		{label: "Wrap", opcode: opAddGlobal, id: 13, values: []uint32{3, 2}, tags: []uint32{1, 1}},
		{label: "Pay", opcode: opAddGlobal, id: 21, values: []uint32{2, 3}, tags: []uint32{1, 1}},
		{label: "Paid", opcode: 3, id: 31, values: []uint32{50, 1}, tags: []uint32{1, 1}},
		{label: "mill", opcode: 2, id: 41, values: []uint32{1, 1}, tags: []uint32{1, 1}},
	}
	o := synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: scriptPayload(actions, conditions)}
	b := built(t, o, func(ed *mapedit.Editor) {
		if err := ed.SetUnitField(0, "owner", 1); err != nil {
			t.Fatal(err)
		}
	}, nil).Bytes()
	out, err := alm.AppendScriptRecord(b, alm.ScriptRecord{Variables: 100, Triggers: []alm.ScriptTrigger{
		{Name: "Setup", Once: 1, Acts: []uint32{11, 12, 13, 41}},
		{Name: "Paid", Once: 1, Left: []uint32{3, 1, 5}, Right: []uint32{4, 2, 6}, Cmp: []uint32{0, 0, 3}, Acts: []uint32{31}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func readGlobalsMill(t *testing.T, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(globalsMillFolder, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// globalsMillCopy is a copy of the fixture with its map and globals file
// replaced where given.
func globalsMillCopy(t *testing.T, mapBytes, globals []byte) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "millglobals")
	for _, rel := range []string{"mod.toml", "main.star", "data/missions.toml", "data/globals.toml", "maps/mill.almx", "maps/mill.dialogs/mill.toml",
		"maps/yard.almx", "maps/yard.dialogs/yard.toml"} {
		b := readGlobalsMill(t, rel)
		switch {
		case rel == "maps/mill.almx" && mapBytes != nil:
			b = mapBytes
		case rel == mod.GlobalsFile && globals != nil:
			b = globals
		}
		writeFileAll(t, filepath.Join(dir, filepath.FromSlash(rel)), b)
	}
	return dir
}

// globalsMillFront is the two-map fixture front with the mod at dir
// launched over it.
func globalsMillFront(t *testing.T, dir string) *FrontEnd {
	t.Helper()
	f := globalsMillBase(t, readGlobalsMill(t, "maps/mill.almx"))
	if _, err := launchMods(t, f, dir); err != nil {
		t.Fatal(err)
	}
	return f
}

// The committed map is the one globalsMillMap builds.
func TestGlobalsMillMapIsTheBuiltMap(t *testing.T) {
	if !bytes.Equal(readGlobalsMill(t, "maps/mill.almx"), globalsMillMap(t, 1)) {
		t.Fatal("testdata/globals-mill/maps/mill.almx is not the map globalsMillMap builds")
	}
}

// The mod's globals file reads and writes back unchanged.
func TestGlobalsFileReadsAndWritesBack(t *testing.T) {
	src := readGlobalsMill(t, mod.GlobalsFile)
	d, err := mod.ParseGlobals("millglobals", mod.GlobalsFile, src)
	if err != nil {
		t.Fatal(err)
	}
	want := []mod.Global{
		{Mod: "millglobals", File: mod.GlobalsFile, Line: 1, ID: 1, Name: "goblins_driven",
			Notes: map[string]string{"en": "1 once the mill quest began.", "ru": "1, когда задание мельницы началось."}},
		{Mod: "millglobals", File: mod.GlobalsFile, Line: 7, ID: 2, Name: "mill_coins", Initial: 5},
		{Mod: "millglobals", File: mod.GlobalsFile, Line: 12, ID: 3, Name: "near_wrap", Initial: math.MaxInt32 - 1},
	}
	if !reflect.DeepEqual(d.Globals, want) {
		t.Fatalf("read %+v", d.Globals)
	}
	k, ok := mod.ScriptDataKind(mod.GlobalsFile)
	if !ok || k.Name != "globals" || !k.State {
		t.Fatalf("the globals kind is %+v", k)
	}
}

// A mission seats the initials of the mod that plays it; the same map
// outside a mod sees none, so its reads are 0 and its writes go nowhere.
func TestGlobalsMillSeatsTheInitials(t *testing.T) {
	f := globalsMillFront(t, globalsMillFolder)
	w := openExtendedMission(t, f, 0)
	want := []sim.ScriptGlobal{{ID: 1}, {ID: 2, Value: 5}, {ID: 3, Value: math.MaxInt32 - 1}}
	if got := w.ScriptGlobals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("seated %v, want %v", got, want)
	}
	plain := extendedScriptFront(t, readGlobalsMill(t, "maps/mill.almx"))
	pw := openExtendedMission(t, plain, 40)
	if g := pw.ScriptGlobals(); g != nil {
		t.Fatalf("a map outside a mod holds globals %v", g)
	}
}

// The setup trigger sets and adds, the add wraps, the reply adds, and the
// later trigger compares and acts only after the reply.
func TestGlobalsMillTriggersAndReplyReadSetAndAdd(t *testing.T) {
	f := globalsMillFront(t, globalsMillFolder)
	w := openExtendedMission(t, f, 40)
	mw := f.live
	want := []sim.ScriptGlobal{{ID: 1, Value: 1}, {ID: 2, Value: 7}, {ID: 3, Value: math.MinInt32}}
	if got := w.ScriptGlobals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("after the setup %v, want %v", got, want)
	}
	if mw.mission.dialogue == nil || len(mw.mission.replies) != 2 {
		t.Fatalf("the mill dialogue did not open with two replies: %+v", mw.mission.replies)
	}
	if w.ScriptRegister(50) != 0 {
		t.Fatal("the comparison held before the reply")
	}
	mw.advanceNotice(ui.NoticeReply1)
	missionSteps(mw, 2)
	w = f.live.world
	if got := w.ScriptGlobals()[1]; got.Value != 10 {
		t.Fatalf("the reply left global 2 at %d", got.Value)
	}
	if w.ScriptRegister(50) != 1 {
		t.Fatal("the later trigger did not compare the globals and act")
	}
}

// The same start and steps give the same globals and World hash.
func TestGlobalsMillReplayIsEqual(t *testing.T) {
	run := func() (uint64, []sim.ScriptGlobal) {
		f := globalsMillFront(t, globalsMillFolder)
		w := openExtendedMission(t, f, 40)
		f.live.advanceNotice(ui.NoticeReply1)
		missionSteps(f.live, 2)
		w = f.live.world
		return w.Hash(), w.ScriptGlobals()
	}
	h1, g1 := run()
	h2, g2 := run()
	if h1 != h2 || !reflect.DeepEqual(g1, g2) {
		t.Fatalf("the replay gave %x %v, then %x %v", h1, g1, h2, g2)
	}
}

// A node naming a global its mod does not declare is refused by the launch,
// at the mission row's file and line, naming the map, the trigger and the
// reply and the id; the same map run outside the check reads it as 0.
func TestGlobalsMillUndeclaredIDIsRefused(t *testing.T) {
	dir := globalsMillCopy(t, globalsMillMap(t, 7), nil)
	_, err := launchMods(t, globalsMillBase(t, globalsMillMap(t, 7)), dir)
	if err == nil {
		t.Fatal("the launch admitted a map naming global 7")
	}
	for _, want := range []string{"data/missions.toml:1", "maps/mill.almx", "Trigger #0", "Trigger #1", "global 7", `"millglobals"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	// A reply naming the node is refused at the reply as well.
	dialog := strings.Replace(string(readGlobalsMill(t, "maps/mill.dialogs/mill.toml")), "do = [21]", "do = [21, 11]", 1)
	m := globalsMillMap(t, 7)
	dir = globalsMillCopy(t, m, nil)
	writeFileAll(t, filepath.Join(dir, "maps", "mill.dialogs", "mill.toml"), []byte(dialog))
	_, err = launchMods(t, globalsMillBase(t, m), dir)
	if err == nil || !strings.Contains(err.Error(), "dialog mill, node 1, reply 1 names global 7") {
		t.Fatalf("the reply's refusal: %v", err)
	}
	// Declared, the same map passes.
	globals := append(readGlobalsMill(t, mod.GlobalsFile), "\n[[global]]\nid = 7\nname = \"late\"\n"...)
	if _, err := launchMods(t, globalsMillBase(t, m), globalsMillCopy(t, m, globals)); err != nil {
		t.Fatalf("a declared global 7 is refused: %v", err)
	}
	// Run without the check, the undeclared global reads 0 and takes no
	// write: the setup's set of global 7 leaves nothing, and "Paid" never
	// holds.
	f := globalsMillFront(t, globalsMillFolder)
	w := openExtendedMission(t, f, 0)
	if err := w.SeatScriptGlobals([]sim.ScriptGlobal{{ID: 2, Value: 5}, {ID: 3, Value: math.MaxInt32 - 1}}); err != nil {
		t.Fatal(err)
	}
	missionSteps(f.live, 40)
	f.live.advanceNotice(ui.NoticeReply1)
	missionSteps(f.live, 2)
	w = f.live.world
	if g := w.ScriptGlobals(); len(g) != 2 || g[0].ID != 2 || g[0].Value != 10 {
		t.Fatalf("an undeclared global took a write: %v", g)
	}
	if w.ScriptRegister(50) != 0 {
		t.Fatal("an undeclared global did not read 0")
	}
}

// A mission SAVE keeps the live values of the globals: LOAD gives the same
// values, trigger state and World hash, and after the reply "Paid" fires on
// the loaded side as on the saved one. A SAV without the values seats the
// initials, and a World without globals writes no globals into the SAV.
func TestGlobalsMillSaveAndLoadKeepTheValues(t *testing.T) {
	f := globalsMillFront(t, globalsMillFolder)
	w := openExtendedMission(t, f, 40)
	want := []sim.ScriptGlobal{{ID: 1, Value: 1}, {ID: 2, Value: 7}, {ID: 3, Value: math.MinInt32}}
	if got := w.ScriptGlobals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("before SAVE %v, want %v", got, want)
	}
	save := func(f *FrontEnd) []byte {
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	load := func(raw []byte) *FrontEnd {
		cold := globalsMillFront(t, globalsMillFolder)
		open, town, err := cold.RestoreOriginal(raw)
		if err == nil && !town {
			err = cold.App("globals reload").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("LOAD", town, err)
		}
		return cold
	}
	// saveAndLoad saves f's mission and loads it into a new front, whose
	// World must equal the saved one.
	saveAndLoad := func(f *FrontEnd) (*FrontEnd, []byte) {
		w, raw := f.live.world, save(f)
		cold := load(raw)
		got := cold.live.world
		if g := got.ScriptGlobals(); !reflect.DeepEqual(g, w.ScriptGlobals()) || got.Hash() != w.Hash() ||
			!reflect.DeepEqual(got.ScriptExtendedState(), w.ScriptExtendedState()) || !sim.EqualTriggers(got.Script().Triggers(), w.Script().Triggers()) {
			t.Fatalf("after LOAD globals %v hash %x, saved %v %x", g, got.Hash(), w.ScriptGlobals(), w.Hash())
		}
		return cold, raw
	}
	_, raw := saveAndLoad(f)
	// The same SAV without the values, as an older save, seats the initials.
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions.Program == nil || !reflect.DeepEqual(actions.Program.Globals, want) {
		t.Fatal("the supplement lacks the globals", err)
	}
	actions.Program.Globals = nil
	encoded, err := json.Marshal(actions)
	if err == nil {
		err = sav.SetNativeActions(&doc.State, encoded)
	}
	if err == nil {
		raw, err = sav.EncodeDocumentData(doc)
	}
	if err != nil {
		t.Fatal(err)
	}
	initials := []sim.ScriptGlobal{{ID: 1}, {ID: 2, Value: 5}, {ID: 3, Value: math.MaxInt32 - 1}}
	if g := load(raw).live.world.ScriptGlobals(); !reflect.DeepEqual(g, initials) {
		t.Fatalf("an older SAV seated %v, want %v", g, initials)
	}
	// The reply adds 3 to global 2 on a tick before "Paid" compares; a save
	// taken on that tick leaves the comparison to the loaded side.
	f.live.advanceNotice(ui.NoticeReply1)
	for i := 0; i < 32 && f.live.world.ScriptGlobals()[1].Value != 10; i++ {
		f.live.tick()
	}
	if g := f.live.world.ScriptGlobals()[1]; g.Value != 10 || f.live.world.ScriptRegister(50) != 0 {
		t.Fatalf("after the reply global 2 is %d, variable 50 %d", g.Value, f.live.world.ScriptRegister(50))
	}
	cold, _ := saveAndLoad(f)
	missionSteps(f.live, 2)
	missionSteps(cold.live, 2)
	if a, b := f.live.world, cold.live.world; a.ScriptRegister(50) != 1 || b.ScriptRegister(50) != 1 || a.Hash() != b.Hash() {
		t.Fatalf("Paid: saved side %d, loaded side %d, globals %v", a.ScriptRegister(50), b.ScriptRegister(50), b.ScriptGlobals())
	}
	plain := extendedScriptFront(t, readGlobalsMill(t, "maps/mill.almx"))
	openExtendedMission(t, plain, 40)
	if doc, err = sav.DecodeDocumentData(save(plain)); err != nil {
		t.Fatal(err)
	}
	if actions, err = readCurrentActions(&doc); err != nil || actions.Program == nil {
		t.Fatal("the supplement lacks the program", err)
	}
	if b, _ := json.Marshal(actions.Program); actions.Program.Globals != nil || bytes.Contains(b, []byte(`"Globals"`)) {
		t.Fatalf("a World without globals saved %s", b)
	}
}

// The editor's live check names a global the open mod does not declare, as
// the launch refuses it; the same map naming a declared global shows none.
func TestGlobalsMillEditorCheckNamesAnUndeclaredGlobal(t *testing.T) {
	for _, c := range []struct {
		global uint32
		want   bool
	}{{7, true}, {1, false}} {
		x, s := contextEditor(t)
		n := openInW(t, s, globalsMillCopy(t, globalsMillMap(t, c.global), nil))
		if x.mod == nil || filepath.Base(n.source) != "mill.almx" {
			t.Fatalf("global %d: the editor opened %s", c.global, n.source)
		}
		var rows []string
		for _, r := range n.Report() {
			if r.Rule == "global-undeclared" {
				rows = append(rows, r.Line())
			}
		}
		if got := len(rows) != 0; got != c.want {
			t.Fatalf("global %d: the live check's global rows %q", c.global, rows)
		}
		if c.want && !strings.Contains(strings.Join(rows, "\n"), "7") {
			t.Fatalf("the rows do not name global 7: %q", rows)
		}
	}
}

// A globals file is refused by its file and line.
func TestGlobalsFileRefusalsNameFileAndLine(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"[[global]]\nid = 0\nname = \"a\"\n", "data/globals.toml:2"},
		{"[[global]]\nid = 1025\nname = \"a\"\n", "data/globals.toml:2"},
		{"[[global]]\nid = 1\nname = \"A\"\n", "data/globals.toml:3"},
		{"[[global]]\nid = 1\nname = \"a\"\ninitial = 2147483648\n", "data/globals.toml:4"},
		{"[[global]]\nid = 1\nname = \"a\"\n[[global]]\nid = 1\nname = \"b\"\n", "data/globals.toml:4"},
		{"[[global]]\nid = 1\nname = \"a\"\n[[global]]\nid = 2\nname = \"a\"\n", "data/globals.toml:4"},
		{"[[global]]\nname = \"a\"\n", "data/globals.toml:1"},
		{"[[global]]\nid = 1\nname = \"a\"\nfr = \"x\"\n", "data/globals.toml:4"},
	} {
		_, err := mod.ParseGlobals("m", mod.GlobalsFile, []byte(c.src))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %s", c.src, err, c.want)
		}
	}
}
