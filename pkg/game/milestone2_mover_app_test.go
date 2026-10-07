package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func mover1160Initial(t *testing.T, want mover1160Source, ms *Mission) {
	t.Helper()
	doc := want.documentDifferences(ms.savedDocument)
	live, _, p := want.worldDifferences(ms.World, ms.savedDocument)
	if len(doc)+len(live) != 0 || p.compared == 0 {
		t.Fatal("original BEFORE Snapshot", doc, live, p)
	}
}

// These are observed current/native Document values, never an original-byte
// oracle. Read every Unit-family record, including ones without a live carrier.
func mover1160Retained(state *SnapshotSAVDocument) (map[uint16]mover1160Record, error) {
	if state == nil || state.Document == nil || state.Unavailable != "" {
		return nil, fmt.Errorf("retained mover Document unavailable")
	}
	out := map[uint16]mover1160Record{}
	for i, o := range state.Document.Objects {
		if !unit1156Class(o.Class) {
			continue
		}
		r := mover1160Record{class: o.Class}
		fields := map[string][]byte{}
		for _, f := range o.Raw {
			if f.Name != "Block12" && f.Name != "U154" && f.Name != "U158" && !slices.Contains(mover1160Lists[:], f.Name) {
				continue
			}
			if _, duplicate := fields[f.Name]; duplicate {
				return nil, fmt.Errorf("DTO%d duplicate %s", i+1, f.Name)
			}
			fields[f.Name] = f.Bytes
		}
		if len(fields) != 6 || len(fields["Block12"]) != 12 || len(fields["U154"]) != 180 || len(fields["U158"]) != 148 {
			return nil, fmt.Errorf("DTO%d incomplete/wrong-width movement blocks", i+1)
		}
		copy(r.position[:], fields["Block12"])
		copy(r.mover[:], fields["U154"])
		copy(r.order[:], fields["U158"])
		for j, name := range mover1160Lists {
			count, value := 0, uint32(0)
			for _, f := range o.Counts {
				if f.Name == name {
					count++
					value = f.Count
				}
			}
			b := fields[name]
			if count != 1 || uint64(value)*2 != uint64(len(b)) {
				return nil, fmt.Errorf("DTO%d invalid %s count", i+1, name)
			}
			r.routes[j] = make([]uint16, len(b)/2)
			for k := range r.routes[j] {
				r.routes[j][k] = binary.LittleEndian.Uint16(b[2*k:])
			}
		}
		out[uint16(i+1)] = r
	}
	return out, nil
}

// DIV-1092: equal Worlds can retain different historic Position/Mover/routes
// after Current becomes false. Name such differences; never make them current
// to force equality. All orders, classes, populations, current motion fields,
// and independently current facing/rotation bytes still compare strictly.
func mover1160ContinuationDifferences(a, b *SnapshotSAVDocument, w *sim.World) (differences, historic []string) {
	left, err := mover1160Retained(a)
	if err != nil {
		return []string{err.Error()}, nil
	}
	right, err := mover1160Retained(b)
	if err != nil {
		return []string{err.Error()}, nil
	}
	if len(left) != len(right) || !reflect.DeepEqual(a.Actors, b.Actors) {
		differences = append(differences, "retained actor population/bindings differ")
	}
	superseded := map[uint16]bool{}
	motions, _, _, _ := w.SavedActorMotions()
	for _, m := range motions {
		if m.Current {
			continue
		}
		for _, binding := range a.Actors {
			if binding.EntityID == m.Entity {
				superseded[binding.ObjectIndex] = true
			}
		}
	}
	for _, index := range slices.Sorted(maps.Keys(left)) {
		l := left[index]
		r, found := right[index]
		if !found || l.class != r.class {
			differences = append(differences, fmt.Sprintf("DTO%d class/presence differs", index))
			continue
		}
		if l.order != r.order || !slices.Equal(l.routes[2], r.routes[2]) {
			differences = append(differences, fmt.Sprintf("DTO%d Order148/path differs", index))
		}
		if l.position != r.position || l.mover != r.mover || !slices.Equal(l.routes[0], r.routes[0]) || !slices.Equal(l.routes[1], r.routes[1]) {
			message := fmt.Sprintf("DTO%d Position12/Mover180/static/dynamic differs", index)
			if superseded[index] && l.mover[0] == r.mover[0] && l.mover[10] == r.mover[10] {
				historic = append(historic, message)
			} else {
				differences = append(differences, message)
			}
		}
	}
	return
}

func mover1160Snapshot(t *testing.T, f *FrontEnd) Snapshot {
	t.Helper()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if diff := mover1160CurrentMotionDifferences(s.SavedDocument, f.live.world); len(diff) != 0 {
		t.Fatal("current World vs projected movement Document", diff)
	}
	if diff := mover1160CurrentOrderDifferences(s.SavedDocument, f.live.world); len(diff) != 0 {
		t.Fatal("current World vs projected order Document", diff)
	}
	return s
}

// Current World values are the native persistence oracle only. Original LOAD
// expectations remain the independent Body reader and initial carrier check.
func mover1160CurrentMotionDifferences(state *SnapshotSAVDocument, w *sim.World) []string {
	retained, err := mover1160Retained(state)
	if err != nil {
		return []string{err.Error()}
	}
	byEntity := map[sim.EntityID]uint16{}
	for _, b := range state.Actors {
		byEntity[b.EntityID] = b.ObjectIndex
	}
	entities := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		entities[e.ID] = e
	}
	var differences []string
	motions, _, _, _ := w.SavedActorMotions()
	for _, m := range motions {
		if !m.Current {
			continue
		}
		r, found := retained[byEntity[m.Entity]]
		if !found {
			differences = append(differences, fmt.Sprintf("current motion%d lacks DTO", m.Entity))
			continue
		}
		p := m.Position
		var position [12]byte
		binary.LittleEndian.PutUint16(position[:], p.Cell)
		binary.LittleEndian.PutUint16(position[2:], p.PackedCell)
		position[4], position[5] = p.FineX, p.FineY
		binary.LittleEndian.PutUint16(position[6:], p.Residue)
		binary.LittleEndian.PutUint32(position[8:], p.TerrainKey)
		mover := m.Mover
		e := entities[m.Entity]
		mover[0], mover[10] = e.Facing, e.ActorLoad.Source.MoverSpeed
		if r.position != position || r.mover != mover {
			differences = append(differences, fmt.Sprintf("current motion%d Position/Mover differ from Document", m.Entity))
		}
		for j, route := range [][]uint16{m.StaticRoute, m.DynamicRoute} {
			if !slices.Equal(r.routes[j], route) {
				differences = append(differences, fmt.Sprintf("current motion%d %s differs: Document=%v World=%v", m.Entity, mover1160Lists[j], r.routes[j], route))
			}
		}
	}
	return differences
}

func mover1160Menu(t *testing.T, app *ui.App) {
	t.Helper()
	// Original LOAD can restore an open character/inventory pane. Escape
	// closes that first; each event still uses the ordinary App dispatch.
	for i := 0; i < 6 && app.Screen() != ui.ScreenGameMenu; i++ {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatal("ordinary map menu did not open", app.Screen())
	}
}

func mover1160App(t *testing.T, raw []byte, front func(*testing.T) *FrontEnd) {
	t.Helper()
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := mover1160Expected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	f := front(t)
	f.SetDeterministicFrames(true)
	app := f.App("Mover and route acceptance")
	app.Layout(1024, 768)
	path := filepath.Join(t.TempDir(), "mover.sav")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	if app.Screen() != ui.ScreenMenu {
		t.Fatal("original title LOAD door absent")
	}
	groundAppLoad(t, app, list, "mover.sav")
	mover1160Initial(t, want, f.live.mission.state)
	previous := f.live
	mover1160Menu(t, app)
	groundAppLoad(t, app, list, "mover.sav")
	if previous == f.live {
		t.Fatal("map-menu original LOAD did not replace driver")
	}
	mover1160Initial(t, want, f.live.mission.state)
	initial, _, _, _ := f.live.world.SavedActorMotions()
	beforeTick := f.live.world.Tick()
	f.live.tick()
	if f.live.world.Tick() != beforeTick+1 {
		t.Fatal("supported change tick did not advance")
	}
	changed := 0
	now, _, _, _ := f.live.world.SavedActorMotions()
	for _, a := range initial {
		for _, b := range now {
			if a.Entity == b.Entity && a.Current && b.Current && a.Issue == "" && b.Issue == "" && (a.Position != b.Position || a.Mover != b.Mover) {
				changed++
			}
		}
	}
	if changed == 0 {
		t.Fatal("no supported Position/Mover state changed before menu SAVE")
	}
	before := mover1160Snapshot(t, f)
	if reflect.DeepEqual(initial, now) {
		t.Fatal("changed-state witness is vacuous")
	}
	mover1160Menu(t, app)
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatal("menu SAVE using the explicit AGS codec", entries, err)
	}
	name := entries[0].Name
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	// Only this private source is removed. No original executable is run.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("private source remains", err)
	}
	fresh := front(t)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("Source free mover LOAD")
	app2.Layout(1024, 768)
	save, list, load = nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(save, list, load)
	groundAppLoad(t, app2, list, name)
	left, err := mover1160Retained(before.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	right, err := mover1160Retained(fresh.live.mission.state.savedDocument)
	if err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != fresh.live.world.Hash() || !reflect.DeepEqual(left, right) {
		t.Fatal("fresh native LOAD lost World or complete retained movement Document")
	}
	historic := map[string]bool{}
	for step := 1; step <= 20; step++ {
		lt, rt := f.live.world.Tick(), fresh.live.world.Tick()
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Tick() != lt+1 || fresh.live.world.Tick() != rt+1 || f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("step%d did not advance equally", step)
		}
		a, b := mover1160Snapshot(t, f), mover1160Snapshot(t, fresh)
		diff, debt := mover1160ContinuationDifferences(a.SavedDocument, b.SavedDocument, f.live.world)
		if len(diff) != 0 {
			t.Fatalf("step%d current retained movement: %v", step, diff)
		}
		for _, s := range debt {
			historic[s] = true
		}
	}
	for _, s := range spell1152Keys(historic) {
		t.Log("DIV-1092 historical boundary: " + s)
	}
	// A following actual client move goes through the ordinary App pointer
	// route on both sessions, then must advance and install a real target.
	var actor sim.Entity
	foundActor := false
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.Alive() && !e.OffMap && e.SourceBinding.Class != 0 {
			actor = e
			foundActor = true
			break
		}
	}
	if !foundActor {
		for _, e := range f.live.world.Entities() {
			t.Logf("subject id%d owner%d class%d hp%d decay%d cell%d,%d", e.ID, e.Owner, e.SourceBinding.Class, e.HP, e.Decay, e.X, e.Y)
		}
		t.Fatal("following action has no live owner actor")
	}
	x, y := -1, -1
	for pairIndex, pair := range []struct {
		f   *FrontEnd
		app *ui.App
	}{{f, app}, {fresh, app2}} {
		mover1160Menu(t, pair.app)
		if err := pair.app.HeadlessGameMenuAction("return"); err != nil {
			t.Fatal(err)
		}
		pair.f.live.push()
		pair.f.live.view.Camera().CenterOn(float64(actor.X)*32, float64(actor.Y)*32)
		if err := pair.app.HeadlessSelectEntity(uint32(actor.ID)); err != nil {
			t.Fatalf("following App pair%d selection: %v", pairIndex, err)
		}
		px, py := -1, -1
		for sy := 0; sy < 768 && px < 0; sy += 4 {
			for sx := 0; sx < 1024; sx += 4 {
				cx, cy, err := pair.app.HeadlessDropCell(sx, sy)
				if err != nil {
					continue
				}
				if x < 0 {
					dx, dy := cx-int(actor.X), cy-int(actor.Y)
					if dx < 0 {
						dx = -dx
					}
					if dy < 0 {
						dy = -dy
					}
					if max(dx, dy) < 2 || max(dx, dy) > 4 {
						continue
					}
					occupied := false
					for _, e := range pair.f.live.world.Entities() {
						if e.Alive() && int(e.X) == cx && int(e.Y) == cy {
							occupied = true
						}
					}
					if occupied {
						continue
					}
					x, y = cx, cy
				}
				if cx == x && cy == y {
					px, py = sx, sy
					break
				}
			}
		}
		if px < 0 {
			t.Fatalf("following App move has no visible ground near actor%d cell%d,%d target%d,%d", actor.ID, actor.X, actor.Y, x, y)
		}
		if err := pair.app.HeadlessPointer("press", px, py); err != nil {
			t.Fatal(err)
		}
		if err := pair.app.HeadlessPointer("release", px, py); err != nil {
			t.Fatal(err)
		}
		if len(pair.f.live.pending) == 0 {
			t.Fatal("App move queued no real command")
		}
		prior := pair.f.live.world.Tick()
		pair.f.live.tick()
		e, ok := pair.f.live.entity(actor.ID)
		if !ok || pair.f.live.world.Tick() != prior+1 || !e.HasTarget || e.TargetX != int32(x) || e.TargetY != int32(y) {
			t.Fatal("following App move did not install target", e)
		}
	}
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("following App action diverged")
	}
	a, b := mover1160Snapshot(t, f), mover1160Snapshot(t, fresh)
	if diff, _ := mover1160ContinuationDifferences(a.SavedDocument, b.SavedDocument, f.live.world); len(diff) != 0 {
		t.Fatal("following App Move lost current movement/order projection", diff)
	}
	t.Logf("mover routes: %d raw actors; title+map-menu original LOAD checked BEFORE Snapshot; %d supported carriers changed; menu SAVE, private source removed, fresh native LOAD with complete movement Document equality; EACH of20 advancing World/current Document pairs; next App Move to%d,%d; %d named historic Document differences", len(want.records), changed, x, y, len(historic))
}

func mover1160AppFixture(t *testing.T) []byte {
	t.Helper()
	f := unit1158FixtureFront(t)
	a := &poolFixtureActor{mapID: 91, cell: 0x1414, hp: 31, maxHP: 101, mana: 23, maxMana: 103, name: "turn", profile: literalProfile1107()}
	b := &poolFixtureActor{mapID: 92, cell: 0x1416, hp: 37, maxHP: 107, mana: 29, maxMana: 109, name: "survivor", profile: literalProfile1107()}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a, b}}}}, nil)
	mover := body[a.off+179 : a.off+359]
	mover[0], mover[1], mover[10], mover[0x9d], mover[0xa0] = 28, 96, 20, 3, 1
	for i := range 32 {
		mover[32+i] = byte(0x41 + i)
	}
	// Three separate literal nonempty lists, with unequal lengths and a
	// repeated order-path cell. No decoder/writer supplies their values.
	splice := func(at, n int, value []byte) {
		b := append([]byte(nil), body[:at]...)
		b = append(b, value...)
		body = append(b, body[at+n:]...)
	}
	splice(a.off+507, 2, []byte{3, 0, 20, 20, 26, 26, 20, 20})
	splice(a.off+41, 4, []byte{2, 0, 21, 20, 22, 20, 1, 0, 20, 21})
	return completeDocumentTail1115(t, f, savedContainer(body))
}

func TestMover1160AppContinuation(t *testing.T) {
	mover1160App(t, mover1160AppFixture(t), unit1158FixtureFront)
}

func TestMover1160NativeAndDocumentLossControls(t *testing.T) {
	f := unit1158FixtureFront(t)
	raw := mover1160AppFixture(t)
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.App("native movement control").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	want := mover1160Snapshot(t, f)
	form, err := EncodeSave(want, "baseline")
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := DecodeSave(form)
	if err != nil {
		t.Fatal(err)
	}
	if diff, debt := mover1160ContinuationDifferences(want.SavedDocument, back.SavedDocument, f.live.world); len(diff)+len(debt) != 0 {
		t.Fatal("native baseline", diff, debt)
	}
	for _, name := range []string{"U158", "U154", "Block12", "U15C", "U178", "U158_90"} {
		t.Run("Document-"+name, func(t *testing.T) {
			bad := back
			bad.SavedDocument = unit1156Clone(t, back.SavedDocument)
			for i := range bad.SavedDocument.Document.Objects {
				r := &bad.SavedDocument.Document.Objects[i]
				if !unit1156Class(r.Class) {
					continue
				}
				for j := range r.Raw {
					if r.Raw[j].Name == name {
						if len(r.Raw[j].Bytes) == 0 {
							r.Raw[j].Bytes = []byte{1, 0}
							for k := range r.Counts {
								if r.Counts[k].Name == name {
									r.Counts[k].Count = 1
								}
							}
						} else {
							r.Raw[j].Bytes[len(r.Raw[j].Bytes)-1] ^= 0x80
						}
					}
				}
				break
			}
			if !bytes.Equal(bad.World, want.World) {
				t.Fatal("Document control changed hashed World")
			}
			diff, debt := mover1160ContinuationDifferences(want.SavedDocument, bad.SavedDocument, f.live.world)
			if len(diff) == 0 || len(debt) != 0 {
				t.Fatal("World hash cannot detect this Document loss", diff, debt)
			}
			if name == "U158" {
				// The pointer residue is not hashed. Even a valid native roundtrip
				// must not hide its loss from the retained-Document assertion.
				form, err := EncodeSave(bad, "lost order tail")
				if err != nil {
					t.Fatal(err)
				}
				decoded, _, err := DecodeSave(form)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(decoded.World, want.World) {
					t.Fatal("native Document loss control changed World")
				}
				if diff, _ := mover1160ContinuationDifferences(want.SavedDocument, decoded.SavedDocument, f.live.world); len(diff) == 0 {
					t.Fatal("native roundtrip hid order pointer loss")
				}
			}
		})
	}
	// Corrupt the actual native World form at a unique synthetic mover span.
	needle := []byte("ABCDEFGHIJKLMNOPQRSTUVWX")
	at := bytes.Index(want.World, needle)
	if at < 0 || bytes.Count(want.World, needle) != 1 {
		t.Fatal("native mover control has no unique literal subject")
	}
	bad := slices.Clone(want.World)
	bad[at+3] ^= 0x80
	var world sim.World
	if err := world.UnmarshalBinary(bad); err == nil && world.Hash() == f.live.world.Hash() {
		t.Fatal("native World mover loss escaped byte-form/hash check")
	}
}
