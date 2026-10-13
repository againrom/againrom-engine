package game

import (
	"os"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Mission 20 flyers: Sonic Bats, Ghosts, Bees.
var deadFlyerUnits = map[uint16]bool{59: true, 60: true, 68: true, 69: true, 70: true, 71: true, 145: true, 146: true, 147: true, 148: true}

func deadFlyerFront(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	dir := t.TempDir()
	store := SaveStore{Dir: dir + "/saves"}
	f.Options = OptionsStore{Path: dir + "/options.txt"}
	app := f.App("dead flyers")
	app.Layout(1024, 768)
	app.SetCutscenes(nil)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	party := f.ChargenParty(ui.ChargenResult{Name: "Dead flyers", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	return f, app
}

func deadFlyerExport(t *testing.T, f *FrontEnd) (sav.DocumentData, []byte) {
	t.Helper()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return doc, raw
}

// deadFlyerRecord reads an actor record's stage and health, if doc has one.
func deadFlyerRecord(t *testing.T, doc *sav.DocumentData, id sim.EntityID) (stage uint32, hp int16, present bool) {
	t.Helper()
	actions, err := readCurrentActions(doc)
	if err != nil || actions == nil {
		t.Fatal("current binding absent", err)
	}
	for _, b := range actions.Bindings {
		if !b.Structure && !b.Missing && b.ID == id && b.Object != 0 && int(b.Object) <= len(doc.Objects) {
			r := &doc.Objects[b.Object-1]
			s, _ := savedStructureValue(r, "Stage")
			h, _ := savedStructureValue(r, "Health")
			return s, int16(h), true
		}
	}
	return 0, 0, false
}

// deadFlyerCount counts actor records and those pinned at the teardown health.
func deadFlyerCount(doc *sav.DocumentData) (pinned, actors int) {
	for i := range doc.Objects {
		r := &doc.Objects[i]
		switch r.Class {
		case "Unit", "Human", "Humanoid":
		default:
			continue
		}
		actors++
		s, e1 := savedStructureValue(r, "Stage")
		h, e2 := savedStructureValue(r, "Health")
		if e1 == nil && e2 == nil && s == 1 && int16(h) <= -1000 {
			pinned++
		}
	}
	return
}

func TestReleaseDeadFlyersLeaveNoActorInSave(t *testing.T) {
	f, _ := deadFlyerFront(t)
	var flyers []sim.EntityID
	var control sim.EntityID
	party := map[sim.EntityID]bool{}
	for _, id := range f.live.mission.ids {
		party[id] = true
	}
	for _, e := range f.live.world.Entities() {
		switch {
		case deadFlyerUnits[e.MapUnitID] && e.Domain != sim.DomainGround:
			flyers = append(flyers, e.ID)
		case control == 0 && e.Domain == sim.DomainGround && e.MapUnitID != 0 && !party[e.ID] && e.Alive():
			control = e.ID
		}
	}
	if len(flyers) != len(deadFlyerUnits) || control == 0 {
		t.Fatal("mission 20 fixture lost its flyers or a ground creature", len(flyers), control)
	}
	for _, id := range append(append([]sim.EntityID{}, flyers...), control) {
		body, _ := f.live.entity(id)
		if err := f.live.world.HeadlessDamage(id, body.HP); err != nil {
			t.Fatal(err)
		}
	}
	f.live.push()
	for range 3 {
		f.live.tick()
	}

	// During the countdown every body is in the world and the file.
	during, _ := deadFlyerExport(t, f)
	for _, id := range append(append([]sim.EntityID{}, flyers...), control) {
		e, in := f.live.entity(id)
		stage, hp, present := deadFlyerRecord(t, &during, id)
		if !in || e.Decay != sim.DecayFallen || !present || stage != 1 || int32(hp) != e.HP {
			t.Fatal("dying body not kept with its stage and health", id, in, e, present, stage, hp)
		}
	}
	_, actorsDuring := deadFlyerCount(&during)

	for n := 0; n < 600; n++ {
		f.live.tick()
		gone := true
		for _, id := range flyers {
			if _, in := f.live.entity(id); in {
				gone = false
			}
		}
		if gone {
			break
		}
	}
	for _, id := range flyers {
		if _, in := f.live.entity(id); in {
			t.Fatal("flyer outlived its dying countdown", id)
		}
	}
	after, afterRaw := deadFlyerExport(t, f)
	for _, id := range flyers {
		if stage, hp, present := deadFlyerRecord(t, &after, id); present {
			t.Fatal("torn-down flyer is still an actor in the SAV", id, stage, hp)
		}
	}
	pinned, actorsAfter := deadFlyerCount(&after)
	if pinned != 0 || actorsAfter != actorsDuring-len(flyers) {
		t.Fatal("torn-down flyers left records behind", pinned, actorsDuring, actorsAfter)
	}
	// Loss control: the ground creature's body is still written.
	if _, _, present := deadFlyerRecord(t, &after, control); !present {
		t.Fatal("ground corpse lost from the SAV", control)
	}
	// The file loads without the flyers; departed rows remain for scripts.
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	open, town, err := cold.RestoreOriginal(afterRaw)
	if err != nil || town {
		t.Fatal("LOAD of the saved file", town, err)
	}
	coldApp := cold.App("dead flyers cold")
	coldApp.Layout(1024, 768)
	coldApp.SetCutscenes(nil)
	if err := coldApp.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	for _, id := range flyers {
		if _, in := cold.live.entity(id); in {
			t.Fatal("LOAD put a torn-down flyer back", id)
		}
	}
	if _, in := cold.live.entity(control); !in {
		t.Fatal("LOAD lost the ground body", control)
	}
	if cold.live.world.Hash() != f.live.world.Hash() {
		a, _ := f.live.world.MarshalBinary()
		b, _ := cold.live.world.MarshalBinary()
		n := 0
		for n < len(a) && n < len(b) && a[n] == b[n] {
			n++
		}
		t.Fatal("cold LOAD World differs from the saved one", len(a), len(b), n)
	}
	rows := 0
	for _, row := range cold.live.world.CurrentTerminalActors() {
		if row.LeavesNoBody() {
			rows++
		}
	}
	if rows != len(flyers) {
		t.Fatal("departed rows of the flyers after LOAD", rows, len(flyers))
	}
}

// An old save with torn-down flyers loads and is re-saved without them.
func TestReleaseDeadFlyersOldSaveLoadsAndIsCleaned(t *testing.T) {
	path := os.Getenv("AGAINROM_DEAD_FLYERS_SAV")
	if path == "" {
		t.Skip("no AGAINROM_DEAD_FLYERS_SAV: the owner save with torn-down flyers")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if pinned, _ := deadFlyerCount(&source); pinned != 10 {
		t.Skip("local save is not the owner save", pinned)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	app := f.App("old dead flyers")
	app.Layout(1024, 768)
	app.SetCutscenes(nil)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	for range 10 {
		f.live.tick()
	}
	again, _ := deadFlyerExport(t, f)
	pinned, actors := deadFlyerCount(&again)
	_, actorsSource := deadFlyerCount(&source)
	if pinned != 0 || actors != actorsSource-10 {
		t.Fatal("re-saved game9274 keeps torn-down flyers", pinned, actorsSource, actors)
	}
}
