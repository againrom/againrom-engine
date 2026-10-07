package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const ogreSourceHash = "281de84e2c89d1e4933a46cd73361f2d5480dc02d56bc6a87fb7071ece32b0c6"

type ogreActorCut struct {
	ID                                   sim.EntityID
	MapUnitID                            uint16
	Class, TypeID, HP, MaxHP             int32
	Owner                                uint32
	TokenSize, Reach, Facing, ActorState uint8
	Position                             [2]int32
	HasTarget                            bool
	TargetX, TargetY                     int32
	Transit, TransitTotal                uint16
	AttackTarget                         sim.EntityID
	AttackTargetKind                     sim.AttackTargetKind
	HasAttackTarget, AcquirePursuit      bool
	AttackPhase                          sim.AttackPhase
	AttackCountdown                      int32
}

type ogreSAVCut struct {
	Tick       uint64
	Ogre, Hero ogreActorCut
	SAVHash    string
}

func ogreMapActor(t *testing.T, w *sim.World, mapID uint16) sim.Entity {
	t.Helper()
	var found sim.Entity
	count := 0
	for _, e := range w.Entities() {
		if e.MapUnitID == mapID {
			found, count = e, count+1
		}
	}
	if count != 1 {
		t.Fatalf("map actor %d has %d live World bindings", mapID, count)
	}
	return found
}

func ogreCaptureActor(t *testing.T, w *sim.World, e sim.Entity) ogreActorCut {
	t.Helper()
	x, y, present := w.ActorFinePosition(e.ID)
	if !present {
		if e.Transit != 0 {
			t.Fatal("stationary witness found an actor crossing without fine position", e.ID)
		}
		x, y = 128, 128
	}
	return ogreActorCut{
		ID: e.ID, MapUnitID: e.MapUnitID, Class: e.Class, TypeID: e.TypeID,
		HP: e.HP, MaxHP: e.MaxHP, Owner: e.Owner,
		TokenSize: e.TokenSize, Reach: e.Reach, Facing: e.Facing, ActorState: e.ActorState,
		Position: [2]int32{e.X*256 + int32(x), e.Y*256 + int32(y)}, HasTarget: e.HasTarget,
		TargetX: e.TargetX, TargetY: e.TargetY, Transit: e.Transit, TransitTotal: e.TransitTotal,
		AttackTarget: e.AttackTarget, AttackTargetKind: e.AttackTargetKind,
		HasAttackTarget: e.HasAttackTarget, AcquirePursuit: e.AcquirePursuit,
		AttackPhase: e.AttackPhase, AttackCountdown: e.AttackCountdown,
	}
}

func ogreCapture(t *testing.T, f *FrontEnd) ogreSAVCut {
	t.Helper()
	w, present := f.LiveWorld()
	if !present || f.liveMission != 40 {
		t.Fatal("mission 40 World is not live")
	}
	return ogreSAVCut{Tick: w.Tick(), Ogre: ogreCaptureActor(t, w, ogreMapActor(t, w, 99)), Hero: ogreCaptureActor(t, w, ogreMapActor(t, w, 6))}
}

func ogreStep(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	before := f.live.world.Tick()
	if _, kind, up := f.LiveNotice(); up {
		if kind == ui.NoticeSuccess {
			t.Fatal("attack witness unexpectedly reached mission success")
		}
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal("close LOAD notice", err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatal("attack witness left the map", app.Screen(), app.HeadlessMessage())
	}
	if f.live.world.Tick() == before {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Tick() <= before {
		t.Fatal("App frame did not advance the real World clock")
	}
}

func ogrePhysicalHurt(before, after ogreSAVCut) bool {
	return before.Ogre.HasAttackTarget && before.Ogre.AttackTarget == before.Hero.ID &&
		before.Ogre.AttackPhase == sim.AttackCharging && after.Ogre.AttackPhase != sim.AttackCharging &&
		after.Hero.HP < before.Hero.HP
}

func ogreF2Save(t *testing.T, f *FrontEnd, app *ui.App) (string, []byte) {
	t.Helper()
	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	before := f.live.world.Hash()
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenGameMenu {
		t.Fatal("F2 SAVE", err, app.Screen())
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || !IsOriginal(entries[0].Name) {
		t.Fatal("F2 did not emit one current SAV", entries, err)
	}
	raw, err := store.Read(entries[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("return from F2 SAVE", err, app.Screen())
	}
	if f.live.world.Hash() != before {
		t.Fatal("F2 SAVE changed the charged World")
	}
	if err := os.WriteFile(filepath.Join(dir, "ogre-current.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	return dir, raw
}

func ogreCheckWire(t *testing.T, raw []byte, want ogreSAVCut) {
	t.Helper()
	doc, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || a.Policy == nil {
		t.Fatal("current SAV has no current action/clock authority", err)
	}
	if doc.Head.Mission != 40 || uint64(doc.Head.CounterA)|uint64(a.Policy.TickHigh)<<32 != want.Tick {
		t.Fatalf("saved mission/clock %d/%d:%d, want 40/%d", doc.Head.Mission, a.Policy.TickHigh, doc.Head.CounterA, want.Tick)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []ogreActorCut{want.Ogre, want.Hero} {
		object, count := uint16(0), 0
		for _, b := range a.Bindings {
			if !b.Structure && !b.Missing && b.ID == expected.ID {
				object, count = b.Object, count+1
			}
		}
		if count != 1 || object == 0 || int(object) > len(doc.Objects) {
			t.Fatalf("actor %d has no unique current SAV object binding", expected.ID)
		}
		archive := uint16(0)
		for _, origin := range origins {
			if origin.ObjectIndex == object {
				archive = origin.ArchiveIndex
			}
		}
		var actor sav.ActorRecord
		count = 0
		for _, candidate := range graph.Actors {
			if candidate.ArchiveIndex == archive {
				actor, count = candidate, count+1
			}
		}
		if count != 1 || actor.Identity == 0 || actor.RuntimeID == 0 || actor.MapUnitID != expected.MapUnitID ||
			actor.TokenSize != expected.TokenSize || actor.OwnerSlot != uint16(expected.Owner) ||
			int32(actor.HP) != expected.HP || actor.Facing != expected.Facing ||
			[2]int32{int32(actor.Col()*256) + int32(actor.FineX), int32(actor.Row()*256) + int32(actor.FineY)} != expected.Position {
			t.Fatalf("saved actor at object %d/archive %d disagrees with live cut: %+v want %+v", object, archive, actor.Actor, expected)
		}
		r := doc.Objects[object-1]
		if int32(savedRecordValueForTest(t, r, "HealthMax")) != expected.MaxHP ||
			uint8(savedRecordValueForTest(t, r, "U12C")) != expected.Reach {
			t.Fatal("saved maximum health/reach does not match live actor", expected.ID)
		}
		count = 0
		for _, action := range a.Actions.Actors {
			if action.Entity != expected.ID {
				continue
			}
			count++
			if action.HasAttackTarget != expected.HasAttackTarget || action.AttackTarget != expected.AttackTarget ||
				action.AttackTargetKind != expected.AttackTargetKind || action.AcquirePursuit != expected.AcquirePursuit ||
				action.AttackPhase != expected.AttackPhase || action.AttackCountdown != expected.AttackCountdown ||
				action.HasTarget != expected.HasTarget || action.TargetX != expected.TargetX || action.TargetY != expected.TargetY ||
				action.Transit != expected.Transit || action.TransitTotal != expected.TransitTotal {
				t.Fatalf("saved action differs for actor %d: %+v want %+v", expected.ID, action, expected)
			}
		}
		if count != 1 {
			t.Fatal("current SAV has no unique actor action", expected.ID)
		}
		t.Logf("wire actor=%d map=%d object=%d archive=%d key=%d runtime=%d size=%d position=%v hp=%d phase=%d/%d", expected.ID, expected.MapUnitID, object, archive, actor.Identity, actor.RuntimeID, actor.TokenSize, expected.Position, actor.HP, expected.AttackPhase, expected.AttackCountdown)
	}
}

func ogreColdApp(t *testing.T, path string) (*FrontEnd, *ui.App) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := shopOrderFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, filepath.Base(path))
	t.Cleanup(app.StopAudio)
	return f, app
}

func ogreRequireSourceHash(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != ogreSourceHash {
		t.Fatalf("Ogre source save changed: %s", got)
	}
	return raw
}

func TestReleaseOgreTouchingHeroLoadsChargesAndResumesSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_OGRESTUCK_CURRENT"); path != "" {
		proof, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var want ogreSAVCut
		if err := json.Unmarshal(proof, &want); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != want.SAVHash {
			t.Fatal("cold current SAV bytes differ from the emitted cut", err)
		}
		ogreCheckWire(t, raw, want)
		f, app := ogreColdApp(t, path)
		got := ogreCapture(t, f)
		want.SAVHash = ""
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cold LOAD changed the emitted current cut: got %+v want %+v", got, want)
		}
		for range 256 {
			before := ogreCapture(t, f)
			ogreStep(t, f, app)
			after := ogreCapture(t, f)
			if ogrePhysicalHurt(before, after) {
				t.Logf("cold next physical strike: tick %d->%d hero HP %d->%d", before.Tick, after.Tick, before.Hero.HP, after.Hero.HP)
				return
			}
		}
		t.Fatal("cold retained Ogre attack produced no physical HP loss within 256 ticks")
	}
	path := os.Getenv("AGAINROM_OGRESTUCK_INPUT")
	if path == "" {
		t.Skip("AGAINROM_OGRESTUCK_INPUT is not set")
	}
	raw := ogreRequireSourceHash(t, path)
	t.Cleanup(func() { ogreRequireSourceHash(t, path) })
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	bound := false
	for _, actor := range graph.Actors {
		if actor.ArchiveIndex == 82 && actor.Identity == 1358955280 && actor.RuntimeID == 43 && actor.MapUnitID == 99 &&
			actor.Class == "Unit" && actor.TypeID == 66 && actor.DefRow == 80 && actor.TokenSize == 2 &&
			actor.Col() == 28 && actor.Row() == 25 && actor.FineX == 128 && actor.FineY == 128 {
			bound = true
		}
	}
	if !bound {
		t.Fatal("hash-bound source has no exact Ogre actor")
	}
	f := shopOrderFront(t)
	if f.Table.Units.EntryName(80) != "Ogre" {
		t.Fatal("installed Units row does not bind the source Ogre")
	}
	app, _ := openOriginalSAVApp(t, f, raw, "ogrestuck.sav")
	t.Cleanup(app.StopAudio)
	initial := ogreCapture(t, f)
	if initial.Ogre.TypeID != 66 || initial.Ogre.TokenSize != 2 || initial.Ogre.Reach != 1 ||
		initial.Ogre.Position != [2]int32{28*256 + 128, 25*256 + 128} || initial.Ogre.Owner != 5 ||
		initial.Hero.Position != [2]int32{28*256 + 128, 27*256 + 128} || initial.Hero.TokenSize != 1 ||
		initial.Hero.HP != 161 || !initial.Ogre.HasAttackTarget || initial.Ogre.AttackTarget != initial.Hero.ID {
		t.Fatalf("source App LOAD did not retain the adjacent Ogre/hero state: %+v", initial)
	}
	sawCharge, sawHurt := false, false
	for tick := range 256 {
		before := ogreCapture(t, f)
		ogreStep(t, f, app)
		after := ogreCapture(t, f)
		if after.Ogre.Position != initial.Ogre.Position || after.Hero.Position != initial.Hero.Position {
			t.Fatal("touching bodies changed position before the attack proof", after)
		}
		sawCharge = sawCharge || after.Ogre.AttackPhase == sim.AttackCharging
		if tick == 15 && !sawCharge {
			t.Fatal("already-targeting touching Ogre never charged within 16 App ticks", after)
		}
		if ogrePhysicalHurt(before, after) {
			sawHurt = true
			t.Logf("source next physical strike: tick %d->%d hero HP %d->%d", before.Tick, after.Tick, before.Hero.HP, after.Hero.HP)
		}
		if !sawHurt || after.Ogre.AttackPhase != sim.AttackCharging || after.Ogre.AttackCountdown < 2 {
			continue
		}
		dir, saved := ogreF2Save(t, f, app)
		if cut := ogreCapture(t, f); !reflect.DeepEqual(cut, after) {
			t.Fatal("actual F2 SAVE changed the charged World cut", cut, after)
		}
		after.SAVHash = fmt.Sprintf("%x", sha256.Sum256(saved))
		ogreCheckWire(t, saved, after)
		proof, err := json.MarshalIndent(after, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		current := filepath.Join(dir, "ogre-current.sav")
		if output := os.Getenv("AGAINROM_OGRESTUCK_REVIEW_DIR"); output != "" {
			scope, err := filepath.Abs(filepath.Join("..", "..", "..", "review", "ogrestuck"))
			if err != nil || !filepath.IsAbs(output) {
				t.Fatal("Ogre review output must be an absolute path under review/ogrestuck", err)
			}
			relative, err := filepath.Rel(scope, filepath.Clean(output))
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				t.Fatal("Ogre review output is outside its owned scope", output, err)
			}
			output = filepath.Join(output, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
			if err := os.MkdirAll(output, 0700); err != nil {
				t.Fatal(err)
			}
			current = filepath.Join(output, "ogre-current.sav")
			if err := os.WriteFile(current, saved, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(current+".json", append(proof, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
		t.Logf("F2 SAV %s sha256=%s tick=%d source=%s", current, after.SAVHash, after.Tick, ogreSourceHash)
		runSpellWitnessChild(t, current, "AGAINROM_OGRESTUCK_CURRENT")
		return
	}
	t.Fatal("Ogre did not hurt the hero and reach a later physical charge within 256 ticks")
}
