package game

import (
	"encoding/json"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// The undying witness's map: campaign map 10 with two triggers added in the
// editor. "Undying on" (constant 1 == constant 1) sets the hero undying on
// the first trigger pass; "Below" (health below -25 on the hero == 1) clears
// the state and sets variable 99 to 1.
const (
	undyingHero     = 10001
	undyingVariable = 99
)

// undyingNodes are the record indexes of the witness's added script.
type undyingNodes struct {
	conditions, actions, triggers int
}

func authorUndyingScript(t *testing.T, e *ui.MapEditor) undyingNodes {
	t.Helper()
	s := e.Document().Edits.(*mapInspectionEdits)
	n := undyingNodes{conditions: s.model.Count(mapedit.KindCondition), actions: s.model.Count(mapedit.KindAction),
		triggers: s.model.Count(mapedit.KindTrigger)}
	for _, kind := range []string{"Condition", "Condition", "Action", "Action", "Action", "Trigger", "Trigger"} {
		if _, err := s.AddScript(kind); err != nil {
			t.Fatal(err)
		}
	}
	set := func(kind string, index int, values ...ui.InspectionFieldValue) {
		t.Helper()
		if _, err := s.SetFields(kind, index, values); err != nil {
			t.Fatalf("%s %d: %v", kind, index, err)
		}
	}
	field := func(key string, v int64) ui.InspectionFieldValue { return ui.InspectionFieldValue{Key: key, Value: v} }
	name := func(text string) ui.InspectionFieldValue {
		return ui.InspectionFieldValue{Key: "name", Text: true, TextValue: text}
	}
	below, undying := int64(mapload.AgainromScriptOperation(true)), int64(mapload.AgainromScriptOperation(false))
	set("Condition", n.conditions, name("One"), field("value0", 1))
	set("Condition", n.conditions+1, name("Hero below -25"), field("type0", mapload.ScriptUnit), field("type1", mapload.ScriptInt),
		field("value0", undyingHero), field("value1", -25), field("opcode", below))
	set("Action", n.actions, name("Hero undying"), field("type0", mapload.ScriptUnit), field("type1", mapload.ScriptInt),
		field("value0", undyingHero), field("value1", 1), field("opcode", undying))
	set("Action", n.actions+1, name("Hero mortal"), field("type0", mapload.ScriptUnit), field("type1", mapload.ScriptInt),
		field("value0", undyingHero), field("value1", 0), field("opcode", undying))
	set("Action", n.actions+2, name("Mark 99"), field("type1", mapload.ScriptInt), field("value0", undyingVariable), field("value1", 1),
		field("opcode", int64(sim.ScriptInstantSetVariable)))
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := m.Script()
	id := func(cond bool, i int) int64 {
		if cond {
			return int64(sc.Conditions[i].ID)
		}
		return int64(sc.Actions[i].ID)
	}
	one := id(true, n.conditions)
	set("Trigger", n.triggers, name("Undying on"),
		field("left0", one), field("right0", one), field("action0", id(false, n.actions)))
	set("Trigger", n.triggers+1, name("Below"),
		field("left0", id(true, n.conditions+1)), field("right0", one),
		field("action0", id(false, n.actions+1)), field("action1", id(false, n.actions+2)))
	// The edits went to the document's model; a Save As and Open shows them
	// in the editor as an author reopening the map sees them.
	path := filepath.Join(t.TempDir(), "undying.alm")
	if _, err := s.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	if !e.Document().AgainromOnly {
		t.Fatal("the reopened map is not marked Againrom-only")
	}
	return n
}

// undyingFrames draws the trigger popup of "Below" and the node popup of each
// Againrom node headless, in the install's language, into dir when it is set.
func undyingFrames(t *testing.T, e *ui.MapEditor, n undyingNodes, lang, dir string, encode func(string) string) {
	t.Helper()
	e.Select(recordIndex(t, e, "Trigger", n.triggers+1), true)
	trigDetails(t, e)
	lines := trigLines(e)
	editorShot(t, e, dir, lang+"-undying-trigger", image.Rectangle{})
	r, ok := e.RowButtonRect(2, 0, false)
	trigClick(t, e, r, ok, "Edit pair 1 left")
	editorShot(t, e, dir, lang+"-undying-health-below", image.Rectangle{})
	r, ok = e.PopupButtonRect(2)
	trigClick(t, e, r, ok, "Back")
	e.Select(recordIndex(t, e, "Trigger", n.triggers), true)
	if e.WizardShown() {
		trigDetails(t, e)
	}
	r, ok = e.RowButtonRect(11, 0, false)
	trigClick(t, e, r, ok, "Edit action 1")
	editorShot(t, e, dir, lang+"-undying-set", image.Rectangle{})
	for range 4 {
		_ = e.Key("escape")
	}
	if !strings.Contains(lines, "-25") {
		t.Fatalf("the trigger popup:\n%s", lines)
	}
	if !strings.Contains(e.StatusLine(), encode(words.For(lang).Text("editor.againrom.map"))) {
		t.Fatalf("the status strip lacks the mark: %q", e.StatusLine())
	}
}

// undyingWitness is what the SAVE process hands the cold LOAD process.
type undyingWitness struct {
	Dir, Save     string
	Hero          sim.EntityID
	Saved, Struck uint64
	StruckHP      int32
	Fired         uint64
}

// undyingGuest opens the guest mission by its campaign route, as a SAV that
// a cold LOAD restores holds it.
func undyingGuest(t *testing.T, dir string) *FrontEnd {
	t.Helper()
	f := guestFront(t, dir, guestModID)
	app, next := guestRoute(t, f, []int{0, 0, 0})
	if next != guestMissionNo {
		t.Fatalf("the guest route answered %d", next)
	}
	if err := app.OpenMission(f.MissionOpener(next)); err != nil {
		t.Fatal(err)
	}
	return f
}

// undyingHeroOf is the hero's entity in f's live world.
func undyingHeroOf(t *testing.T, f *FrontEnd, id sim.EntityID) sim.Entity {
	t.Helper()
	e, ok := f.live.world.Entity(id)
	if !ok {
		t.Fatal("the hero is not in the world")
	}
	return e
}

// undyingApproach steps f's live world from tick start: the hero is undying
// after two trigger passes and walks; it takes a blow to 20 below zero, is
// ordered on, and walks at negative health, not falling. It returns
// the tick of the second blow, the one after the last tick it steps to.
func undyingApproach(t *testing.T, f *FrontEnd, hero sim.EntityID, start uint64) uint64 {
	t.Helper()
	w := f.live.world
	for w.Tick() < start+32 {
		sim.Step(w, nil)
	}
	from := undyingHeroOf(t, f, hero)
	if !from.Undying {
		t.Fatalf("the hero is not undying on tick %d", w.Tick())
	}
	sim.Step(w, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: from.X + 4, Y: from.Y + 1})})
	for w.Tick() < start+40 {
		sim.Step(w, nil)
	}
	sim.Step(w, []sim.Command{sim.Damage(hero, undyingHeroOf(t, f, hero).HP+20)})
	hit := undyingHeroOf(t, f, hero)
	sim.Step(w, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: hit.X + 4, Y: hit.Y + 1})})
	for w.Tick() < start+80 {
		sim.Step(w, nil)
	}
	e := undyingHeroOf(t, f, hero)
	if hit.HP != -20 || !e.Alive() || e.HP >= 0 || e.Decay != sim.DecayNone || e.X == hit.X && e.Y == hit.Y {
		t.Fatalf("the undying hero after the blow to %d: hp %d alive %v decay %d cell %d,%d -> %d,%d",
			hit.HP, e.HP, e.Alive(), e.Decay, hit.X, hit.Y, e.X, e.Y)
	}
	return w.Tick() + 1
}

// undyingRun steps f's live world from where it stands to the tick struck,
// deals the blow that takes the hero to 35 below zero, and steps until
// variable 99 is set. It returns the hero's health before the blow and the
// tick the variable was set on.
func undyingRun(t *testing.T, f *FrontEnd, id sim.EntityID, struck uint64) (int32, uint64) {
	t.Helper()
	w := f.live.world
	for w.Tick() < struck-1 {
		sim.Step(w, nil)
	}
	hp := undyingHeroOf(t, f, id).HP
	sim.Step(w, []sim.Command{sim.Damage(id, hp+35)})
	for range 64 {
		if w.ScriptRegister(undyingVariable) == 1 {
			return hp, w.Tick()
		}
		sim.Step(w, nil)
	}
	t.Fatalf("health-below -25 did not fire by tick %d", w.Tick())
	return 0, 0
}

// TestReleaseScriptUndyingSAVContinuation proves the undying state and the
// health-below condition on the lawful install through the editor and the
// guest route: the hero, made undying, stays alive and acting at negative
// health; a SAVE taken then and cold LOADed in a separate process keeps the
// state, and health-below fires on the same tick as in the run without the
// SAVE, and its clearing action lets the hero die.
func TestReleaseScriptUndyingSAVContinuation(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	if input := os.Getenv("AGAINROM_UNDYING_SAV_INPUT"); input != "" {
		undyingLoadSide(t, input)
		return
	}
	x := installedInspector(t)
	lang := x.pageLanguage()
	dir := guestMods(t, guestModID)
	saveEditedGuestMap(t, x, dir, func(e *ui.MapEditor) {
		n := authorUndyingScript(t, e)
		undyingFrames(t, e, n, lang, os.Getenv("AGAINROM_UNDYING_FRAMES"), x.inspectionUTF8)
	})

	// The run without a SAVE.
	plain := undyingGuest(t, dir)
	hero, _ := leaderOf(t, plain)
	w := plain.live.world
	start := w.Tick()
	struckAt := undyingApproach(t, plain, hero, start)
	hp, fired := undyingRun(t, plain, hero, struckAt)
	after := undyingHeroOf(t, plain, hero)
	if after.Undying || after.Alive() || after.Decay != sim.DecayFallen {
		t.Fatalf("after the clear: undying %v alive %v decay %d hp %d", after.Undying, after.Alive(), after.Decay, after.HP)
	}

	// The same route with a SAVE before the blow.
	g := undyingGuest(t, dir)
	gw := g.live.world
	if gw.Tick() != start || undyingApproach(t, g, hero, start) != struckAt {
		t.Fatal("the two runs differ before the SAVE")
	}
	saved := guestMissionSave(t, g)
	savedAt := gw.Tick()
	if ghp, gfired := undyingRun(t, g, hero, struckAt); ghp != hp || gfired != fired {
		t.Fatalf("after a SAVE the blow met %d and fired on %d; without it %d and %d", ghp, gfired, hp, fired)
	}
	root := t.TempDir()
	in := filepath.Join(root, "undying")
	if err := os.WriteFile(in+".sav", saved, 0o600); err != nil {
		t.Fatal(err)
	}
	proof, _ := json.Marshal(undyingWitness{Dir: dir, Save: in + ".sav", Hero: hero, Saved: savedAt, Struck: struckAt, StruckHP: hp, Fired: fired})
	if err := os.WriteFile(in+".json", proof, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_UNDYING_SAV_INPUT="+in)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "cold LOAD:") {
		t.Fatalf("the cold LOAD process: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "cold LOAD:") {
			t.Log(strings.TrimSpace(line))
		}
	}
	t.Logf("lang=%s hero=%d undying by tick %d; SAVE on tick %d (%d bytes); blow on tick %d at hp %d; health-below fired on tick %d",
		lang, hero, start+32, savedAt, len(saved), struckAt, hp, fired)
}

// No installed map holds an Againrom node, so none is marked Againrom-only and
// the editor writes each as before.
func TestReleaseNoInstalledMapIsAgainromOnly(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	scripts := 0
	for i, b := range maps {
		m, err := alm.Open(b)
		if err != nil {
			continue
		}
		sc, err := m.Script()
		if err != nil {
			continue
		}
		scripts++
		if mapload.ScriptAgainromOnly(sc) {
			t.Errorf("%s holds an Againrom node", keys[i])
		}
	}
	if scripts == 0 {
		t.Fatal("no installed map has a script")
	}
	t.Logf("%d installed maps, %d scripts, none Againrom-only", len(maps), scripts)
}

func undyingLoadSide(t *testing.T, input string) {
	raw, err := os.ReadFile(input + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var want undyingWitness
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(want.Save)
	if err != nil {
		t.Fatal(err)
	}
	h := guestFront(t, want.Dir, guestModID)
	open, town, err := h.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("cold LOAD: town %v %v", town, err)
	}
	if err := h.App("undying load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if got := h.live.world.Tick(); got != want.Saved {
		t.Fatalf("the cold LOAD stands on tick %d, the SAVE on %d", got, want.Saved)
	}
	hero := undyingHeroOf(t, h, want.Hero)
	if !hero.Undying || !hero.Alive() || hero.HP >= 0 {
		t.Fatalf("the loaded hero: undying %v alive %v hp %d", hero.Undying, hero.Alive(), hero.HP)
	}
	hp, fired := undyingRun(t, h, want.Hero, want.Struck)
	if hp != want.StruckHP || fired != want.Fired {
		t.Fatalf("after the cold LOAD the blow met %d and fired on %d; without a SAVE %d and %d", hp, fired, want.StruckHP, want.Fired)
	}
	after := undyingHeroOf(t, h, want.Hero)
	if after.Undying || after.Alive() || after.Decay != sim.DecayFallen {
		t.Fatalf("after the clear: undying %v alive %v decay %d hp %d", after.Undying, after.Alive(), after.Decay, after.HP)
	}
	t.Logf("cold LOAD: tick %d, hero undying at hp %d; blow on tick %d at hp %d; health-below fired on tick %d; the hero fell",
		want.Saved, hero.HP, want.Struck, hp, fired)
}
