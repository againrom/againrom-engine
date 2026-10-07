package game

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestSecondMissionProofRetainsRawActorNameBytes(t *testing.T) {
	before := secondSaveSample{Poses: []secondPoseSample{{Name: []byte{'A', 0xff, 0x83}, Art: []byte{0xc0, 0xff}}}}
	raw, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	var after secondSaveSample
	if err := json.Unmarshal(raw, &after); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("fresh-process proof changed installed actor name bytes")
	}
	after.Poses[0].Name[1] ^= 1
	if reflect.DeepEqual(before, after) {
		t.Fatal("full sample comparison admitted a one-byte actor name mutation")
	}
	t.Log("raw actor Name/Art bytes survive JSON proof; independent one-byte Name mutation fails the full sample comparison")
}

func secondChangeAnimationJSON(t *testing.T, raw []byte, field, value string) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("no current actions", err)
	}
	var root, session, view, animation map[string]json.RawMessage
	if err := json.Unmarshal(leaf, &root); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(root["Session"], &session); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(session["View"], &view); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(view["Animation"], &animation); err != nil {
		t.Fatal(err)
	}
	if value == "" {
		delete(animation, field)
	} else {
		animation[field] = json.RawMessage(value)
	}
	view["Animation"], err = json.Marshal(animation)
	if err != nil {
		t.Fatal(err)
	}
	session["View"], err = json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	root["Session"], err = json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err = json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	result, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func secondMissionPresentationControls(t *testing.T, f *FrontEnd, app *ui.App, out string, raw []byte, mission int) {
	t.Helper()
	unchanged := secondSaveSampleNow(t, f, app)
	reject := func(t *testing.T, changed []byte) {
		town, live := f.Town, f.live
		if _, _, err := f.RestoreOriginal(changed); err == nil {
			t.Fatal("malformed presentation admitted")
		}
		if f.Town != town || f.live != live {
			t.Fatal("malformed presentation adopted candidate")
		}
		secondAssertSample(t, unchanged, secondSaveSampleNow(t, f, app))
	}
	for _, invalid := range []struct{ name, field, value string }{
		{"negative count", "Count", "-1"}, {"overflow count", "Count", "4294967296"},
		{"string count", "Count", `"5"`}, {"fractional count", "Count", "1.5"},
		{"null count", "Count", "null"}, {"missing count", "Count", ""},
		{"negative remainder", "RemainderUS", "-1"}, {"range remainder", "RemainderUS", "1000000"},
		{"string remainder", "RemainderUS", `"17"`}, {"fractional remainder", "RemainderUS", "1.5"},
		{"null remainder", "RemainderUS", "null"}, {"missing remainder", "RemainderUS", ""},
		{"negative crossing phase", "PhaseUS", "-1"}, {"range crossing phase", "PhaseUS", "1000000"},
		{"string crossing phase", "PhaseUS", `"17"`}, {"fractional crossing phase", "PhaseUS", "1.5"},
		{"null crossing phase", "PhaseUS", "null"}, {"missing crossing phase", "PhaseUS", ""},
		{"negative crossing period", "PhasePeriodUS", "-1"}, {"range crossing period", "PhasePeriodUS", "12345"},
		{"string crossing period", "PhasePeriodUS", `"62000"`}, {"fractional crossing period", "PhasePeriodUS", "1.5"},
		{"null crossing period", "PhasePeriodUS", "null"}, {"missing crossing period", "PhasePeriodUS", ""},
	} {
		t.Run(invalid.name, func(t *testing.T) { reject(t, secondChangeAnimationJSON(t, raw, invalid.field, invalid.value)) })
	}
	for _, invalid := range []string{"motion distance", "motion tick", "motion actor", "motion duplicate"} {
		t.Run(invalid, func(t *testing.T) {
			changed := secondChangeSave(t, raw, func(a *currentActionData) {
				if len(a.Animation) == 0 || a.Animation[0].Motion == nil {
					t.Fatal("no captured movement discriminator")
				}
				row := &a.Animation[0]
				switch invalid {
				case "motion distance":
					row.Motion.Distance = -1
				case "motion tick":
					row.Motion.Tick = 2147483648
				case "motion actor":
					row.Entity = 4000000
				case "motion duplicate":
					a.Animation = append(a.Animation, *row)
				}
			})
			reject(t, changed)
		})
	}
	captured := secondSaveSampleNow(t, f, app)
	changed := secondChangeSave(t, raw, func(a *currentActionData) {
		a.Session.View.Animation.Count += 4
		a.Session.View.Animation.RemainderUS++
	})
	if err := os.WriteFile(filepath.Join(out, "clock discriminator.sav"), changed, 0600); err != nil {
		t.Fatal(err)
	}
	clock, clockApp := secondMissionColdAt(t, out, "clock discriminator.sav", mission)
	observed := secondSaveSampleNow(t, clock, clockApp)
	if !bytes.Equal(captured.World, observed.World) || captured.Animation.Count+4 != observed.Animation.Count || captured.Animation.RemainderUS+1 != observed.Animation.RemainderUS || captured.Viewport == observed.Viewport {
		t.Fatal("animation count/remainder discriminator was lost or changed World")
	}
	historical := secondChangeSave(t, raw, func(a *currentActionData) {
		a.Session.View.Animation = nil
		for i := range a.Animation {
			a.Animation[i].Motion = nil
		}
		a.Animation = slices.DeleteFunc(a.Animation, func(row currentAnimationClock) bool { return row.Swing == nil && row.Phase == nil })
	})
	if err := os.WriteFile(filepath.Join(out, "historical presentation.sav"), historical, 0600); err != nil {
		t.Fatal(err)
	}
	old, _ := secondMissionColdAt(t, out, "historical presentation.sav", mission)
	oldWorld, err := old.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(captured.World, oldWorld) || old.live.view.SaveAnimation() != (ui.AnimationState{}) || len(old.live.walk) != 0 || len(old.live.prev) != 0 {
		t.Fatal("absent presentation supplements changed historical constructor defaults or World")
	}
	if err := os.WriteFile(filepath.Join(out, "phase discriminator.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	phase, phaseApp := secondMissionColdAt(t, out, "phase discriminator.sav", mission)
	secondAssertSample(t, captured, secondSaveSampleNow(t, phase, phaseApp))
	phase.live.view.SetPhase(0, 0)
	stale := secondSaveSampleNow(t, phase, phaseApp)
	if !bytes.Equal(captured.World, stale.World) || captured.Viewport == stale.Viewport {
		t.Fatal("zero-period coarse crossing discriminator was invisible or changed World")
	}
	phase.live.view.SetPhase(captured.Animation.PhaseUS, captured.Animation.PhasePeriodUS)
	secondAssertSample(t, captured, secondSaveSampleNow(t, phase, phaseApp))
	f.live.view.SetPhase(17000, 62000)
	fractional := secondSaveSampleNow(t, f, app)
	if fractional.Viewport == captured.Viewport || !bytes.Equal(fractional.World, captured.World) {
		t.Fatal("nonzero crossing phase discriminator was invisible or changed World")
	}
	fractionalOut, err := os.MkdirTemp(out, "fractional-")
	if err != nil {
		t.Fatal(err)
	}
	f.ConfigureSaveSeams(app, SaveStore{Dir: fractionalOut}, OriginalStore{}, nil)
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	loaded, loadedApp := secondMissionColdAt(t, fractionalOut, string(quickSaveBase(0))+".sav", mission)
	secondAssertSample(t, fractional, secondSaveSampleNow(t, loaded, loadedApp))
	if err := f.live.view.RestoreAnimation(captured.Animation); err != nil {
		t.Fatal(err)
	}
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	secondAssertSample(t, captured, secondSaveSampleNow(t, f, app))
	t.Logf("M%d presentation: count/remainder, period0 and nonzero-phase first-frame discriminators, historical absence, malformed type/range/binding atomic refusals", mission)
}

func secondMissionTerminalRefusals(t *testing.T, f *FrontEnd, app *ui.App, out string) {
	t.Helper()
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []struct {
		op   int32
		want sim.Outcome
	}{{sim.ScriptInstantWin, sim.OutcomeWon}, {sim.ScriptInstantLose, sim.OutcomeLost}} {
		trigger := sim.ScriptTrigger{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true}
		program, err := sim.NewROM2Script(nil, []sim.ScriptInstant{{Op: outcome.op, Args: [10]int32{1}}}, []sim.ScriptTrigger{trigger})
		if err != nil {
			t.Fatal(err)
		}
		world, err := sim.NewControlledScriptWorld(f.live.world, program)
		if err != nil {
			t.Fatal(err)
		}
		for range 16 {
			sim.Step(world, nil)
		}
		if world.Outcome() != outcome.want {
			t.Fatal("controlled negative stimulus did not decide World")
		}
		terminal := snapshot
		terminal.World, err = world.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		for _, export := range []func(Snapshot, string) ([]byte, error){f.ExportCurrentSave, f.ExportCurrentWorldSave, f.ExportNativeCitySave, f.ExportOriginalSave} {
			if raw, err := export(terminal, "terminal refusal"); err == nil || len(raw) != 0 {
				t.Fatal("terminal current SAV was produced")
			}
		}
		original := f.live.world
		f.live.world = world
		f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
		if err := app.HeadlessKey("f4"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(out, string(quickSaveBase(0))+".sav")); !os.IsNotExist(err) {
			t.Fatal("terminal F4 published output", err)
		}
		if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
			t.Fatal("terminal negative stimulus did not reach ordinary F2 SAVE", err)
		}
		if err := app.HeadlessSaveEdit(out, "Refused terminal", ui.SaveSAV); err != nil {
			t.Fatal(err)
		}
		before, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("save"); err == nil {
			t.Fatal("terminal ordinary F2 SAVE succeeded")
		}
		if _, err := os.Stat(filepath.Join(out, "Refused terminal.sav")); !os.IsNotExist(err) {
			t.Fatal("terminal F2 published output", err)
		}
		after, _, err := f.Snapshot(true)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("terminal ordinary refusal changed active capture", err)
		}
		f.live.world = original
		if err := app.HeadlessSaveAction("cancel"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("return"); err != nil {
			t.Fatal(err)
		}
	}
	after, _, err := f.Snapshot(true)
	if err != nil || !reflect.DeepEqual(snapshot, after) {
		t.Fatal("terminal producer refusal changed source capture", err)
	}
	t.Log("controlled won/lost World refuses ordinary F2/F4, sole producer and compatibility exports without state/output mutation; no M20 completion is claimed")
}
