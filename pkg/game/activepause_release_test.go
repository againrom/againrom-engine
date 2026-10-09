package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type activePauseSample struct {
	World     []byte
	Hash      uint64
	Queue     []sim.Command
	Commanded []uint32
	View      ui.SaveApplicationState
}

type activePauseProof struct {
	SHA256, Label string
	Samples       []activePauseSample
	QueueLost     bool
}

func activePauseSampleNow(t *testing.T, f *FrontEnd) activePauseSample {
	t.Helper()
	r := f.live.residue()
	slices.Sort(r.Commanded)
	return activePauseSample{World: marshalWorld(t, f.live.world), Hash: f.live.world.Hash(), Queue: slices.Clone(f.live.pending), Commanded: r.Commanded, View: f.live.view.SaveApplication()}
}

func activePauseCompare(t *testing.T, got, want activePauseSample) {
	t.Helper()
	if !bytes.Equal(got.World, want.World) || got.Hash != want.Hash || !slices.Equal(got.Queue, want.Queue) || !slices.Equal(got.Commanded, want.Commanded) || !reflect.DeepEqual(got.View, want.View) {
		t.Fatalf("independent source mismatch: World%v hash%x/%x queue%v/%v commanded%v/%v view%+v/%+v", bytes.Equal(got.World, want.World), got.Hash, want.Hash, got.Queue, want.Queue, got.Commanded, want.Commanded, got.View, want.View)
	}
}

func releasePauseMission(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	if f.live == nil || app.Screen() != ui.ScreenMap {
		t.Fatal("pause input requires a live map")
	}
	before := activePauseSampleNow(t, f)
	beforeStopped, beforeNotice := f.live.stopped, app.HeadlessNoticeOpen()
	if !before.View.PlayerPaused {
		if err := app.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
	}
	after := activePauseSampleNow(t, f)
	if !after.View.PlayerPaused || !f.live.stopped || !bytes.Equal(after.World, before.World) || after.Hash != before.Hash || !slices.Equal(after.Queue, before.Queue) || !slices.Equal(after.Commanded, before.Commanded) || after.View.PeriodUS != before.View.PeriodUS || after.View.Unpaced != before.View.Unpaced {
		t.Fatalf("pause input: PlayerPaused=%v (%v->%v) Stopped=%v (%v->%v) notice=%v->%v WorldBytesEqual=%v HashEqual=%v (%x/%x) QueueEqual=%v CommandedEqual=%v PeriodEqual=%v (%d/%d) UnpacedEqual=%v (%v/%v); queue=%+v/%+v commanded=%v/%v",
			after.View.PlayerPaused, before.View.PlayerPaused, after.View.PlayerPaused, f.live.stopped, beforeStopped, f.live.stopped, beforeNotice, app.HeadlessNoticeOpen(), bytes.Equal(after.World, before.World), after.Hash == before.Hash, after.Hash, before.Hash, slices.Equal(after.Queue, before.Queue), slices.Equal(after.Commanded, before.Commanded), after.View.PeriodUS == before.View.PeriodUS, after.View.PeriodUS, before.View.PeriodUS, after.View.Unpaced == before.View.Unpaced, after.View.Unpaced, before.View.Unpaced, after.Queue, before.Queue, after.Commanded, before.Commanded)
	}
}

func activePauseResume(t *testing.T, app *ui.App) {
	t.Helper()
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
}

func activePauseCold(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path + ".proof.json")
	if err != nil {
		t.Fatal(err)
	}
	var proof activePauseProof
	if err := json.Unmarshal(payload, &proof); err != nil || len(proof.Samples) != 3 || fmt.Sprintf("%x", sha256.Sum256(raw)) != proof.SHA256 {
		t.Fatal("invalid independent source proof", err)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "paused.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	app := f.App("ordinary paused cold LOAD")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	rows := app.HeadlessRows()
	if len(rows) != 1 || rows[0].Text != proof.Label+" - mission 20" {
		t.Fatal("ordinary paused chooser", rows)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenMap || !f.live.mission.resumed {
		t.Fatal("ordinary paused cold LOAD", err, app.Screen())
	}
	if proof.QueueLost {
		got := activePauseSampleNow(t, f)
		if !bytes.Equal(got.World, proof.Samples[0].World) || got.Hash != proof.Samples[0].Hash || !reflect.DeepEqual(got.View, proof.Samples[0].View) || len(got.Queue) != 0 || len(proof.Samples[0].Queue) == 0 {
			t.Fatal("queue-only loss changed pre-resume World/View or retained commands")
		}
	} else {
		activePauseCompare(t, activePauseSampleNow(t, f), proof.Samples[0])
	}
	paused := activePauseSampleNow(t, f)
	releasePauseMission(t, f, app)
	releasePauseMission(t, f, app)
	activePauseCompare(t, activePauseSampleNow(t, f), paused)
	before := marshalWorld(t, f.live.world)
	for range 24 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(before, marshalWorld(t, f.live.world)) || !f.live.stopped {
		t.Fatal("cold paused idle advanced autonomous time")
	}
	activePauseResume(t, app)
	if proof.QueueLost {
		if f.live.world.Hash() == proof.Samples[1].Hash || bytes.Equal(marshalWorld(t, f.live.world), proof.Samples[1].World) {
			t.Fatal("queue-only loss did not alter the first resumed action")
		}
		t.Log("fresh ordinary LOAD queue-only loss: identical pre-resume World/View, different first action")
		return
	}
	activePauseCompare(t, activePauseSampleNow(t, f), proof.Samples[1])
	for range 24 {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
	activePauseCompare(t, activePauseSampleNow(t, f), proof.Samples[2])
	if len(f.live.pending) != 0 {
		t.Fatal("cold resumed queue drained twice")
	}
	t.Log("fresh-process ordinary App LOAD: paused source, first Space resume and 24 later frames match")
}

func TestReleaseActivePauseOrderedSAVAndResume(t *testing.T) {
	if path := os.Getenv("AGAINROM_ACTIVE_PAUSE_INPUT"); path != "" {
		activePauseCold(t, path)
		return
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	dir := t.TempDir()
	if output := os.Getenv("AGAINROM_ACTIVE_PAUSE_OUTPUT"); output != "" {
		if !filepath.IsAbs(output) {
			t.Fatal("active pause output must be absolute")
		}
		dir = filepath.Join(output, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: filepath.Join(dir, "saves")}
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	party := f.ChargenParty(ui.ChargenResult{Name: "Pause mage", Choices: []int{0, 1, 3}, Stats: []int{31, 27, 24, 29}})
	ally := f.ChargenParty(ui.ChargenResult{Name: "Pause ally", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	party[0].ID, ally[0].ID, ally[0].StartingHero = "pause-main", "pause-ally", false
	party[0].KnownSpells, party[0].SpellbookRestored = 1<<6, true
	var scroll sim.ItemInstance
	for _, candidate := range shopScrollPool(f.Table, 100000, rand.New(rand.NewSource(27))) {
		if spell, _, ok := sim.ScrollSpell(candidate.Instance()); ok && spell == 1 {
			scroll = candidate.Instance()
			break
		}
	}
	if scroll.Code == 0 {
		t.Fatal("installed scroll unavailable")
	}
	party[0].CarriedItems = []sim.ItemInstance{mapload.ItemInstanceFromCode(0xe02, f.Table), scroll, scroll}
	party[0].Carried = []uint16{0xe02, scroll.Code, scroll.Code}
	party = append(party, ally...)
	app := f.App("active pause current source")
	app.Layout(1024, 768)
	now := time.Unix(100, 0)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	key := func(name string) {
		t.Helper()
		if err := app.HeadlessKey(name); err != nil {
			t.Fatal(name, err)
		}
	}
	key("0")
	key("numpad-plus")
	key("numpad-plus")
	key("ctrl-numpad-plus")
	if f.live.world.Tick() != 0 || !f.live.stopped || !f.live.unpaced || !f.live.view.SaveApplication().PlayerPaused || f.live.clock.Period() != 41000 {
		t.Fatal("cadence plus pause before first tick was not retained")
	}
	for _, width := range []int{640, 1024} {
		app.Layout(width, width*3/4)
		lines := f.live.view.MessageLines()
		if len(lines) == 0 || strings.Contains(lines[0].Text, "?") || f.Font.Value().Advance(lines[0].Text) > 80 {
			t.Fatal("installed paused HUD caption absent or oversized", width, lines)
		}
	}
	app.Layout(1024, 768)
	main, other := f.live.mission.ids[0], f.live.mission.ids[1]
	e, _ := f.live.world.Entity(main)
	f.live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	if err := app.HeadlessSelectEntity(uint32(main)); err != nil {
		t.Fatal(err)
	}
	before := marshalWorld(t, f.live.world)
	groundX, groundY, err := app.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	click := func(x, y int) {
		t.Helper()
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
	}
	click(groundX, groundY)
	key("a")
	x, y, err := app.HeadlessEntityPoint(uint32(other))
	if err != nil {
		t.Fatal(err)
	}
	click(x, y)
	if !f.live.view.SaveApplication().SpellBookOpen {
		key("b")
	}
	x, y, err = app.HeadlessSpellPoint(6)
	if err != nil {
		t.Fatal(err)
	}
	click(x, y)
	x, y, err = app.HeadlessEntityPoint(uint32(other))
	if err != nil {
		t.Fatal(err)
	}
	click(x, y)
	if !f.live.view.SaveApplication().InventoryOpen {
		key("i")
	}
	usePackCell(t, app, 0)
	usePackCell(t, app, 1)
	x, y, err = app.HeadlessEntityPoint(uint32(other))
	if err != nil {
		t.Fatal(err)
	}
	click(x, y)
	key("e")
	key("ctrl-f")
	click(groundX, groundY)
	if !bytes.Equal(before, marshalWorld(t, f.live.world)) {
		t.Fatal("paused command construction spent resources or stepped World")
	}
	kinds := map[uint8]bool{}
	for _, c := range f.live.pending {
		kinds[c.Kind] = true
	}
	for _, kind := range []uint8{sim.KindGroupMoveTo, sim.KindAttack, sim.KindCast, sim.KindUsePotion, sim.KindUseScroll, sim.KindPlayerParameter} {
		if !kinds[kind] {
			t.Fatal("actual paused App input did not queue kind", kind, f.live.pending)
		}
	}
	source := activePauseSampleNow(t, f)
	paths := []string{filepath.Join(store.Dir, "Paused expedition.sav"), filepath.Join(store.Dir, "quick-save-1.sav"), filepath.Join(store.Dir, "timed-autosave-1.sav")}
	labels := []string{"Paused expedition", "quicksave 1", "timed autosave 1"}
	key("f2")
	if err := app.HeadlessSaveEdit(store.Dir, "Paused expedition", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() == ui.ScreenGameMenu {
		if err := app.HeadlessGameMenuAction("return"); err != nil {
			t.Fatal(err)
		}
	}
	key("f4")
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	activePauseCompare(t, activePauseSampleNow(t, f), source)
	var saves [][]byte
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal("missing-write loss control", path, err)
		}
		saves = append(saves, raw)
	}
	activePauseResume(t, app)
	first := activePauseSampleNow(t, f)
	if f.live.world.Tick() != 1 || len(first.Queue) != 0 || first.View.PlayerPaused || bytes.Equal(first.World, source.World) {
		t.Fatal("Space did not apply one resumed tick", f.live.world.Tick(), first.Queue)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.ExportCurrentSave(s, "Drained paused queue")
	if err != nil {
		t.Fatal(err)
	}
	drainedDoc, err := sav.DecodeDocumentData(second)
	if err != nil {
		t.Fatal(err)
	}
	drained, err := readCurrentActions(&drainedDoc)
	if err != nil || drained == nil || drained.Pending == nil || len(drained.Pending.Commands) != 0 {
		t.Fatal("second SAV retained the drained queue", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "drained.sav"), second, 0600); err != nil {
		t.Fatal(err)
	}
	for range 24 {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
	last := activePauseSampleNow(t, f)
	for i, path := range paths {
		proof := activePauseProof{SHA256: fmt.Sprintf("%x", sha256.Sum256(saves[i])), Label: labels[i], Samples: []activePauseSample{source, first, last}}
		payload, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".proof.json", payload, 0600); err != nil {
			t.Fatal(err)
		}
		runSpellWitnessChild(t, path, "AGAINROM_ACTIVE_PAUSE_INPUT")
	}
	lossDoc, err := sav.DecodeDocumentData(saves[0])
	if err != nil {
		t.Fatal(err)
	}
	lossActions, err := readCurrentActions(&lossDoc)
	if err != nil || lossActions == nil || lossActions.Pending == nil {
		t.Fatal(err)
	}
	lossActions.Pending.Commands = nil
	payload, err := json.Marshal(lossActions)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&lossDoc.State, payload); err != nil {
		t.Fatal(err)
	}
	lossRaw, err := sav.EncodeDocumentData(lossDoc)
	if err != nil {
		t.Fatal(err)
	}
	lossPath := filepath.Join(dir, "queue-only-loss.sav")
	proof := activePauseProof{SHA256: fmt.Sprintf("%x", sha256.Sum256(lossRaw)), Label: labels[0], Samples: []activePauseSample{source, first, last}, QueueLost: true}
	payload, err = json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lossPath, lossRaw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lossPath+".proof.json", payload, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, lossPath, "AGAINROM_ACTIVE_PAUSE_INPUT")
	app.Layout(640, 480)
	optionsBefore, optionsTick, resumePeriod := marshalWorld(t, f.live.world), f.live.world.Tick(), f.live.clock.Period()
	openOptions := func() {
		t.Helper()
		openMissionGameMenu(t, app)
		if err := app.HeadlessGameMenuAction("game-options"); err != nil {
			t.Fatal(err)
		}
	}
	openOptions()
	click(126, 116)
	if f.live.view.SaveApplication().PlayerPaused || !bytes.Equal(optionsBefore, marshalWorld(t, f.live.world)) {
		t.Fatal("installed zero draft changed player intent or simulation")
	}
	if err := app.HeadlessGameMenuAction("options-cancel"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("return"); err != nil || f.live.view.SaveApplication().PlayerPaused || !f.live.unpaced || f.live.clock.Period() != resumePeriod {
		t.Fatal("installed zero Cancel changed exact cadence", err)
	}
	openOptions()
	click(126, 116)
	for _, action := range []string{"page-return", "return"} {
		if err := app.HeadlessGameMenuAction(action); err != nil {
			t.Fatal(err)
		}
	}
	if !f.live.stopped || !f.live.view.SaveApplication().PlayerPaused || !f.live.unpaced || f.live.clock.Period() != resumePeriod {
		t.Fatal("installed zero OK did not retain the exact paused resume mode")
	}
	openOptions()
	click(227, 116)
	for _, action := range []string{"page-return", "return"} {
		if err := app.HeadlessGameMenuAction(action); err != nil {
			t.Fatal(err)
		}
	}
	storedRung, err := (OptionsStore{Path: f.Options.Path}).GameSpeed()
	if err != nil || storedRung != terrain.DefaultCadenceRung+1 || f.live.view.SaveApplication().PlayerPaused || f.live.unpaced || f.live.clock.Period() != terrain.CadencePeriod(storedRung) || f.live.world.Tick() != optionsTick || !bytes.Equal(optionsBefore, marshalWorld(t, f.live.world)) {
		t.Fatal("installed positive OK failed persistence, resume or no-tick control", err, storedRung)
	}
	if err := app.HeadlessStep(); err != nil || f.live.stopped || f.live.world.Tick()-optionsTick > 1 {
		t.Fatal("installed positive return retained modal stop or repaid paused time", err)
	}
	t.Logf("source tick0 hash%x queue%d: actual paused input; manual/timed/quick ordinary SAV; three independent cold processes plus queue-only loss; installed zero Cancel/OK and positive persisted resume", source.Hash, len(source.Queue))
}
