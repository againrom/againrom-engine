package game

import (
	"bytes"
	"encoding/json"
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
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// The extended fixture's condition nodes: reads of variables 120, 5, 130,
// 146 and 148, and the constants 1, 7 and 0.
var extendedConditions = []scriptNode{
	{label: "Var120", opcode: 19, id: 1, values: []uint32{120}, tags: []uint32{1}},
	{label: "One", opcode: 0x10002, id: 2, values: []uint32{1}, tags: []uint32{1}},
	{label: "Var5", opcode: 19, id: 3, values: []uint32{5}, tags: []uint32{1}},
	{label: "Seven", opcode: 0x10002, id: 4, values: []uint32{7}, tags: []uint32{1}},
	{label: "Var130", opcode: 19, id: 5, values: []uint32{130}, tags: []uint32{1}},
	{label: "Zero", opcode: 0x10002, id: 6, values: []uint32{0}, tags: []uint32{1}},
	{label: "Var146", opcode: 19, id: 7, values: []uint32{146}, tags: []uint32{1}},
	{label: "Var148", opcode: 19, id: 8, values: []uint32{148}, tags: []uint32{1}},
}

// The extended fixture's action nodes: the drop cell, the two setups (ids 1
// and 2), the OR trigger's six actions (11..16), and the actions of the
// trigger armed before a save (17) and of the trigger past position 1000 (18).
var extendedActions = []scriptNode{
	{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}},
	{label: "Set120", opcode: 3, id: 1, values: []uint32{120, 1}, tags: []uint32{1, 1}},
	{label: "Set5", opcode: 3, id: 2, values: []uint32{5, 7}, tags: []uint32{1, 1}},
	{label: "Set140", opcode: 3, id: 11, values: []uint32{140, 11}, tags: []uint32{1, 1}},
	{label: "Inc141", opcode: 8, id: 12, values: []uint32{141}, tags: []uint32{1}},
	{label: "Inc141 again", opcode: 8, id: 13, values: []uint32{141}, tags: []uint32{1}},
	{label: "Set142", opcode: 3, id: 14, values: []uint32{142, 5}, tags: []uint32{1, 1}},
	{label: "Inc101", opcode: 8, id: 15, values: []uint32{101}, tags: []uint32{1}},
	{label: "Set143", opcode: 3, id: 16, values: []uint32{143, 1}, tags: []uint32{1, 1}},
	{label: "Set147", opcode: 3, id: 17, values: []uint32{147, 1}, tags: []uint32{1, 1}},
	{label: "Inc149", opcode: 8, id: 18, values: []uint32{149}, tags: []uint32{1}},
}

// extendedTail is the map position of the fixture's last trigger, past the
// thousand latches the session block holds.
const extendedTail = 1001

// extendedScriptMap is a 40x40 map carrying the extended script record of
// 150 variables. Trigger 0 runs setup once. Trigger 1, "Either", fires once
// when group 0 holds (variable 120 is 1, variable 130 is 0, 7 > 0) or group
// 1 holds (variable 5 is 7, 1 > 0, 0 <= 1), and runs six actions. Trigger
// 2 fires once when variable 146 is 1; fillers that never hold run to
// position 1000, and the trigger at extendedTail fires once when variable
// 148 is 1.
func extendedScriptMap(t *testing.T, units []synth.ALMUnit, setup ...uint32) []byte {
	t.Helper()
	never := alm.ScriptTrigger{Name: "Filler", Group: []uint32{0}, Left: []uint32{6}, Right: []uint32{2}, Cmp: []uint32{0}}
	triggers := []alm.ScriptTrigger{
		{Name: "Setup", Once: 1, Acts: setup},
		{Name: "Either", Once: 1, Group: []uint32{0, 0, 0, 1, 1, 1},
			Left: []uint32{1, 5, 4, 3, 2, 6}, Right: []uint32{2, 6, 6, 4, 6, 2}, Cmp: []uint32{0, 0, 2, 0, 2, 5},
			Acts: []uint32{11, 12, 13, 14, 15, 16}},
		{Name: "After", Once: 1, Group: []uint32{0}, Left: []uint32{7}, Right: []uint32{2}, Cmp: []uint32{0}, Acts: []uint32{17}},
	}
	for len(triggers) < extendedTail {
		triggers = append(triggers, never)
	}
	triggers = append(triggers, alm.ScriptTrigger{Name: "Tail", Once: 1, Group: []uint32{0}, Left: []uint32{8}, Right: []uint32{2},
		Cmp: []uint32{0}, Acts: []uint32{18}})
	b := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: units, Type7Payload: scriptPayload(extendedActions, extendedConditions)})
	out, err := alm.AppendScriptRecord(b, alm.ScriptRecord{Variables: 150, Triggers: triggers})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// extendedUnits is the one unit the mission fixture places.
var extendedUnits = []synth.ALMUnit{{X: 15<<8 | 128, Y: 16<<8 | 128, UnitID: 91}}

// extendedScriptFront is the mission-10 fixture front over map m.
func extendedScriptFront(t *testing.T, m []byte) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t)
	f.Archives.Containers = poolFixtureFrontMap(t, m).Archives.Containers
	return f
}

func openExtendedMission(t *testing.T, f *FrontEnd, ticks int) *sim.World {
	t.Helper()
	if err := f.App("extended script").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	for range ticks {
		f.live.tick()
	}
	return f.live.world
}

// TestMapListHoldsExtendedMapFiles lists an .almx beside an .alm, either case.
func TestMapListHoldsExtendedMapFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Either.almx", "Plain.ALM", "Near.almxx", "Other.ALMX"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	maps, err := DirMaps(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(maps.Names(), ","); got != "Either.almx,Other.ALMX,Plain.ALM" {
		t.Fatalf("listed %s", got)
	}
}

func TestExtendedMapListsLoadsAndRunsEachGroup(t *testing.T) {
	dir := t.TempDir()
	m := extendedScriptMap(t, extendedUnits, 1)
	if err := os.WriteFile(filepath.Join(dir, "Either.almx"), m, 0o600); err != nil {
		t.Fatal(err)
	}
	loose, err := vfs.Open(nil, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	maps, err := DirMaps(dir, loose)
	if err != nil || len(maps.Names()) != 1 {
		t.Fatal("the .almx is not listed", err)
	}
	if b, err := maps.Read(maps.Names()[0]); err != nil || !bytes.Equal(b, m) {
		t.Fatal("the listed .almx reads otherwise", err)
	}
	for _, c := range []struct {
		name  string
		setup []uint32
		fires bool
	}{{"group 0", []uint32{1}, true}, {"group 1", []uint32{2}, true}, {"neither", nil, false}} {
		t.Run(c.name, func(t *testing.T) {
			w := openExtendedMission(t, extendedScriptFront(t, extendedScriptMap(t, extendedUnits, c.setup...)), 160)
			s := w.Script()
			if s == nil || !s.Extended() || s.Variables() != 150 || len(s.Triggers()) != extendedTail+1 {
				t.Fatal("the mission does not run the extended program")
			}
			either := s.Triggers()[1]
			if len(either.Conditions) != 6 || len(either.Actions) != 6 || either.Conditions[2].Group != 0 || either.Conditions[3].Group != 1 {
				t.Fatalf("the OR trigger compiled to %+v", either)
			}
			want := map[int32]int32{140: 11, 141: 2, 142: 5, 101: 1, 143: 1}
			for v, n := range want {
				if !c.fires {
					n = 0
				}
				if got := w.ScriptRegister(v); got != n {
					t.Errorf("variable %d is %d, want %d", v, got, n)
				}
			}
			if w.ScriptLatched(1) != c.fires || w.ScriptLatched(2) || w.ScriptLatched(extendedTail) {
				t.Errorf("latches: Either %v, After %v, Tail %v", w.ScriptLatched(1), w.ScriptLatched(2), w.ScriptLatched(extendedTail))
			}
			if c.name == "group 0" && w.ScriptRegister(120) != 1 || w.ScriptRegister(93) < 5 {
				t.Errorf("variable 120 is %d after %d passes", w.ScriptRegister(120), w.ScriptRegister(93))
			}
		})
	}
}

func TestExtendedScriptSaveRoundTripResumesTheSameWorld(t *testing.T) {
	m := extendedScriptMap(t, extendedUnits, 1)
	f := extendedScriptFront(t, m)
	w := openExtendedMission(t, f, 160)
	if !w.ScriptLatched(1) || w.ScriptRegister(140) != 11 {
		t.Fatal("the OR trigger did not fire before the save")
	}
	// Loss control: variables 146 and 148 arm two triggers and the latch of
	// the one past position 1000 is set; a SAV that lost any of the three
	// would fire After late or never, or fire Tail.
	st := w.ScriptExtendedState()
	st.Variables[146-100], st.Variables[148-100], st.Latches[extendedTail-1000] = 1, 1, 1
	if err := w.RestoreScriptExtendedState(st); err != nil {
		t.Fatal(err)
	}
	hash := w.Hash()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil || w.Hash() != hash {
		t.Fatal("SAVE changed the World", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions.Program == nil || actions.Program.Extended == nil || actions.Program.Extended.Variables != 150 ||
		actions.Program.Extended.State.Variables[146-100] != 1 {
		t.Fatal("the supplement lacks the extended program", err)
	}
	// The original session block holds the first hundred variables, the pass
	// counter among them.
	if doc.World.Session.Results[93] != w.ScriptRegister(93) || doc.World.Session.Results[0] != 0 {
		t.Fatal("the session block does not hold the first hundred variables", doc.World.Session.Results[93])
	}
	load := func(raw []byte) *FrontEnd {
		cold := extendedScriptFront(t, m)
		open, town, err := cold.RestoreOriginal(raw)
		if err == nil && !town {
			err = cold.App("extended script reload").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("LOAD", town, err)
		}
		return cold
	}
	cold := load(raw)
	got := cold.live.world
	if got.Hash() != hash || !reflect.DeepEqual(got.ScriptExtendedState(), w.ScriptExtendedState()) ||
		!sim.EqualTriggers(got.Script().Triggers(), w.Script().Triggers()) || !got.ScriptLatched(extendedTail) || got.ScriptLatched(2) {
		t.Fatal("the loaded World is not the saved one")
	}
	// The next action: After fires on the first pass after the reload, on
	// both sides; Tail, armed but latched, does not.
	for tick := 0; tick < 40; tick++ {
		f.live.tick()
		cold.live.tick()
		if cold.live.world.Hash() != f.live.world.Hash() {
			t.Fatal("the loaded World diverged at tick", tick)
		}
	}
	got = cold.live.world
	if !got.ScriptLatched(2) || got.ScriptRegister(147) != 1 || got.ScriptRegister(149) != 0 || got.ScriptRegister(141) != 2 {
		t.Fatal("the next pass after the reload", got.ScriptLatched(2), got.ScriptRegister(147), got.ScriptRegister(149))
	}
	// The variable comes from the supplement: a SAV whose supplement says 0
	// loads a World in which After never fires.
	actions.Program.Extended.State.Variables[146-100] = 0
	leaf, err := json.Marshal(actions)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	changed, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	lost := load(changed)
	for range 40 {
		lost.live.tick()
	}
	if lost.live.world.ScriptRegister(146) != 0 || lost.live.world.ScriptLatched(2) || lost.live.world.ScriptRegister(147) != 0 {
		t.Fatal("the variable did not come from the supplement")
	}
}

// guestAlmxManifest is a guest mission after mission 20 whose map is MAP.
const guestAlmxManifest = `[[mission]]
key      = "prologue"
number   = 25
map      = "MAP"
title    = "t"
briefing = "b"
after    = 20
guest    = 22
chapter  = 30
carry    = "join"
`

func TestModMissionManifestNamesAnAlmxMap(t *testing.T) {
	dir := t.TempDir()
	m := extendedScriptMap(t, nil, 1)
	if err := os.MkdirAll(filepath.Join(dir, "maps"), 0o755); err != nil {
		t.Fatal(err)
	}
	look := func(key string) (string, bool) {
		v, ok := map[string]string{"t": "Title", "b": "Briefing"}[key]
		return v, ok
	}
	for _, name := range []string{"maps/either.almx", "maps/either.alm"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), m, 0o600); err != nil {
			t.Fatal(err)
		}
		d, err := mod.ParseMissions("m", mod.MissionsFile, []byte(strings.Replace(guestAlmxManifest, "MAP", name, 1)), look, "en")
		if err != nil {
			t.Fatal(err)
		}
		for i := range d.Missions {
			if err := d.Missions[i].ReadFiles(dir); err != nil {
				t.Fatal(err)
			}
		}
		f := guestFixtureFront(t)
		_, err = f.SetModMissions(d)
		if strings.HasSuffix(name, ".alm") {
			if err == nil || !strings.Contains(err.Error(), "name it .almx") {
				t.Fatalf("%s: %v", name, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got, err := f.Archives.Containers.ReadFile("scenario/25.alm"); err != nil || !bytes.Equal(got, m) {
			t.Fatalf("%s is not served as mission 25: %v", name, err)
		}
	}
}

func TestMapEditorRefusesAnExtendedScriptMap(t *testing.T) {
	dir := t.TempDir()
	x := checkInspector()
	for name, b := range map[string][]byte{"either.almx": extendedScriptMap(t, nil, 1), "either.alm": extendedScriptMap(t, nil, 1),
		"plain.almx": guestMapBytes()} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := x.Open(p); err == nil || !strings.Contains(err.Error(), "step 3") {
			t.Fatalf("%s: %v", name, err)
		}
	}
	model, err := mapedit.New(guestMapBytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := refuseExtendedMap(model, filepath.Join(dir, "plain.alm")); err != nil {
		t.Fatal(err)
	}
}
