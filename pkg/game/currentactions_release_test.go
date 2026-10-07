package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func worldEntityByID(t *testing.T, f *FrontEnd, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("actor %d disappeared", id)
	return sim.Entity{}
}

// Expectations are recorded from the uninterrupted current World before any
// cold LOAD. Local actor/item handles are joined by source runtime identity
// and ownership, never by equality with a produced Document.
func currentActionSample(t *testing.T, f *FrontEnd, runtime uint32) json.RawMessage {
	t.Helper()
	w := f.live.world
	e := worldEntityByRuntimeID(t, f, runtime)
	runtimeID := func(id sim.EntityID, structure bool) (sim.EntityID, error) {
		if structure {
			return id, nil
		}
		for _, v := range w.Entities() {
			if v.ID == id {
				return sim.EntityID(v.SourceBinding.RuntimeID), nil
			}
		}
		return id, nil
	}
	a := w.Actions()
	if err := a.RemapActors(runtimeID); err != nil {
		t.Fatal(err)
	}
	var actor sim.ActorContinuation
	for _, v := range a.Actors {
		if uint32(v.Entity) == runtime {
			actor = v
		}
	}
	// Imported-motion bookkeeping is a source adapter fact. The exact native
	// route, stride, phase, deadline and all next-tick samples remain compared.
	actor.ImportedMotion = false
	actor.MotionIssue = ""
	items, _ := w.CarriedItems(e.ID)
	for i := range items {
		items[i].ObjectID = 0
	}
	for i := range a.Scrolls {
		a.Scrolls[i].Item.ObjectID = 0
	}
	var reservations []sim.ItemStack
	for _, v := range a.Reservations.Items {
		item := v.Value
		item.ObjectID = 0
		reservations = append(reservations, item)
	}
	value := struct {
		Tick         uint64
		Actor        sim.ActorContinuation
		HP, Mana     int32
		Book         sim.Spellbook
		Items        []sim.ItemInstance
		Scrolls      []sim.ScrollCast
		Scripts      []sim.ScriptCast
		Books        []sim.BookContinuation
		Reservations []sim.ItemStack
		WorldEffects int
		Bolts        []SnapshotSpellBolt
		Heals        []SnapshotHealBurst
		Runs         []SnapshotCastRun
	}{Tick: w.Tick(), Actor: actor, HP: e.HP, Mana: e.Mana, Book: e.Book, Items: items, Scrolls: a.Scrolls, Scripts: a.Scripts, Books: a.Books, Reservations: reservations, WorldEffects: len(w.CurrentWorldEffectOrder())}
	var visuals SnapshotResidue
	f.live.actionVisuals(&visuals)
	value.Bolts, value.Heals, value.Runs = visuals.SpellBolts, visuals.HealBursts, visuals.CastRuns
	for i := range value.Runs {
		value.Runs[i].Entity, _ = runtimeID(value.Runs[i].Entity, false)
	}
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type currentActionProof struct {
	Runtime uint32
	Actor   sim.EntityID
	Mode    string
	Offset  int
	Samples []json.RawMessage
}

func currentActionInput(t *testing.T, f *FrontEnd, p currentActionProof, step int) {
	t.Helper()
	if p.Mode == "refund" && p.Offset+step == 2 {
		e := worldEntityByID(t, f, p.Actor)
		// The same MapOrder seam used by the mission pointer cancels/refunds
		// the pending scroll; no direct mutation of its reservation is used.
		f.live.enqueue(uint32(e.ID), int(e.X+1), int(e.Y))
	}
}

func currentActionCold(t *testing.T, path string) {
	t.Helper()
	var p currentActionProof
	b, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	f := loadAreaContinuation(t, path)
	for step, want := range p.Samples {
		got := currentActionSample(t, f, p.Runtime)
		if !bytes.Equal(got, want) {
			t.Fatalf("%s cold action sample %d:\nwant%s\ngot %s", p.Mode, p.Offset+step, want, got)
		}
		if p.Offset == 0 && step == 4 {
			second := saveCorpseMission(t, f, t.TempDir())
			next := p
			next.Offset = 4
			next.Samples = p.Samples[4:]
			encoded, _ := json.Marshal(next)
			if err = os.WriteFile(second+".json", encoded, 0600); err != nil {
				t.Fatal(err)
			}
			emitSpellWitness(t, second, "action-"+p.Mode+"-second", encoded)
			runSpellWitnessChild(t, second, "AGAINROM_CURRENT_ACTION_INPUT")
		}
		if step+1 < len(p.Samples) {
			currentActionInput(t, f, p, step)
			f.live.tick()
		}
	}
}

func currentActionSaveProof(t *testing.T, f *FrontEnd, runtime uint32, actor sim.EntityID, mode string) {
	t.Helper()
	before := f.live.world.Hash()
	path := saveCorpseMission(t, f, t.TempDir())
	if f.live.world.Hash() != before {
		t.Fatal("ordinary SAVE changed the current World")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatalf("action supplement %v", err)
	}
	if len(f.live.world.ScrollCasts()) > 0 && len(a.Actions.Scrolls) == 0 {
		t.Fatal("reserved scroll omitted")
	}
	requireCurrentActionWire(t, doc, f.live.world, a)
	p := currentActionProof{Runtime: runtime, Actor: actor, Mode: mode}
	for step := 0; step <= 64; step++ {
		p.Samples = append(p.Samples, currentActionSample(t, f, runtime))
		if step < 64 {
			currentActionInput(t, f, p, step)
			f.live.tick()
		}
	}
	if len(f.live.world.ScrollCasts()) != 0 {
		t.Fatal("scroll did not complete/cancel within its real cadence")
	}
	proof, _ := json.Marshal(p)
	if err = os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	emitSpellWitness(t, path, "action-"+mode+"-first", proof)
	runSpellWitnessChild(t, path, "AGAINROM_CURRENT_ACTION_INPUT")
}

func requireCurrentActionWire(t *testing.T, doc sav.DocumentData, w *sim.World, actions *currentActionData) {
	t.Helper()
	actor := func(id sim.EntityID) *sav.DocumentRecordData {
		// The current-action supplement is the authoritative join between the
		// logical actor used by the action and its SAV object. Generated actors
		// are allowed to have source runtime zero; projectCurrentRuntimeIDs
		// assigns their non-zero wire runtime only after this binding is made.
		// Joining those actors through the live SourceBinding.RuntimeID would
		// therefore either miss the wire object or ambiguously select another
		// generated actor.
		for _, binding := range actions.Bindings {
			if binding.Structure || binding.Missing || binding.ID != id || binding.Object == 0 || int(binding.Object) > len(doc.Objects) {
				continue
			}
			return &doc.Objects[binding.Object-1]
		}
		var runtime uint32
		for _, e := range w.Entities() {
			if e.ID == id {
				runtime = e.SourceBinding.RuntimeID
			}
		}
		var found *sav.DocumentRecordData
		for i := range doc.Objects {
			v, err := savedStructureValue(&doc.Objects[i], "RuntimeID")
			if err == nil && v == runtime {
				if found != nil {
					t.Fatal("ambiguous independent runtime join")
				}
				found = &doc.Objects[i]
			}
		}
		if found == nil {
			t.Fatal("current action actor missing from wire")
		}
		return found
	}
	check := func(caster, target sim.EntityID, cell bool, remaining, phase, progress uint8, complete bool, spell uint16) {
		r := actor(caster)
		key := uint32(0)
		if !cell {
			key = savedRecordValueForTest(t, *actor(target), "Identity")
		}
		state := uint32(13)
		if cell {
			state = 14
		}
		wirePhase := uint32(0)
		if phase == 1 {
			wirePhase = 5
		}
		if phase == 2 {
			wirePhase = 7
		}
		if binary.LittleEndian.Uint32(savedRecordRawForTest(t, *r, "U54")) != state || binary.LittleEndian.Uint32(savedRecordRawForTest(t, *r, "U58")) != wirePhase || savedRecordValueForTest(t, *r, "U5C") != key || savedRecordValueForTest(t, *r, "U6C") != uint32(remaining) {
			t.Fatal("native cast wire retained donor target/phase/timer")
		}
		flag := uint32(0)
		if complete {
			flag = 1
		}
		if savedRecordValueForTest(t, *r, "U136") != flag {
			t.Fatal("native completion latch lost")
		}
		unitCast := byte(1)
		if cell {
			unitCast = 0
		}
		if got := savedRecordRawForTest(t, *r, "U158")[0x5c]; got != unitCast {
			t.Fatalf("native cast order +0x5c = %d, want %d", got, unitCast)
		}
		if spell != 0 {
			refs, _ := savedObjectRefs(r, "Spells")
			if int(spell) > len(refs) || refs[spell-1] == 0 {
				t.Fatal("current native spell reference absent")
			}
			sr := doc.Objects[refs[spell-1]-1]
			spellKey := savedRecordValueForTest(t, sr, "This")
			order := savedRecordRawForTest(t, *r, "U158")
			if savedRecordValueForTest(t, *r, "U64") != spellKey || binary.LittleEndian.Uint32(order[0x30:]) != spellKey || binary.LittleEndian.Uint32(order[0x28:]) != key || order[0x15] != progress {
				t.Fatal("native order uses obsolete cast operands")
			}
			for _, e := range w.Entities() {
				if e.ID == caster && savedRecordValueForTest(t, sr, "S09") != uint32(e.Book.Slots[spell-1].Range) {
					t.Fatal("current book range lost in native Spell")
				}
			}
		}
	}
	for _, c := range w.Actions().Books {
		check(c.Caster, c.Target, c.AtCell, c.Remaining, c.Phase, c.Progress, c.Complete, c.Spell)
	}
	for _, c := range w.ScrollCasts() {
		phase := uint8(0)
		if c.Started {
			phase = 1
		}
		check(c.Caster, c.Target, c.AtCell, c.Remaining, phase, 0, false, 0)
	}
}

func TestReleaseCurrentActionScrollSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_CURRENT_ACTION_INPUT"); path != "" {
		currentActionCold(t, path)
		return
	}
	for _, mode := range []string{"release", "refund"} {
		t.Run(mode, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := f.ChargenParty(ui.ChargenResult{Name: "Current scroll", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
			var scroll sim.ItemInstance
			for _, v := range shopScrollPool(f.Table, 100000, rand.New(rand.NewSource(1218))) {
				id, _, ok := sim.ScrollSpell(v.Instance())
				if ok && id == 1 {
					scroll = v.Instance()
					break
				}
			}
			if scroll.Code == 0 {
				t.Fatal("installed shop did not construct Fire Arrow scroll")
			}
			party[0].Carried = []uint16{scroll.Code, scroll.Code}
			party[0].CarriedItems = []sim.ItemInstance{scroll, scroll}
			app := f.App("current scroll SAV")
			app.Layout(1024, 768)
			if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
				t.Fatal(err)
			}
			for n := 0; n < 32; n++ {
				if err := stepConsumableApp(app); err != nil {
					t.Fatal(err)
				}
			}
			id := f.live.mission.ids[0]
			e, _ := f.live.entity(id)
			f.live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
			if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
				t.Fatal(err)
			}
			usePackCell(t, app, 0)
			x, y, err := app.HeadlessEntityPoint(uint32(id))
			if err != nil {
				t.Fatal(err)
			}
			for _, edge := range []string{"press", "release"} {
				if err = app.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			for n := 0; n < 8 && len(f.live.world.ScrollCasts()) == 0; n++ {
				if err = stepConsumableApp(app); err != nil {
					t.Fatal(err)
				}
			}
			if len(f.live.world.ScrollCasts()) != 1 {
				items, _ := f.live.world.CarriedItems(id)
				t.Fatalf("real inventory/map input did not reserve: pending%v items%+v actor%+v", f.live.pending, items, e)
			}
			c := f.live.world.ScrollCasts()[0]
			if c.Item.ObjectID == 0 {
				t.Fatal("generated current scroll lacks registry ownership")
			}
			t.Logf("%s: runtime%d reserved object%d, remaining%d, started%v; installed scroll%+v", mode, e.SourceBinding.RuntimeID, c.Item.ObjectID, c.Remaining, c.Started, scroll)
			currentActionSaveProof(t, f, e.SourceBinding.RuntimeID, e.ID, mode)
			left, _ := f.live.world.CarriedItems(id)
			count := uint32(0)
			for _, v := range left {
				if v.Code == scroll.Code {
					count++
				}
			}
			want := uint32(1)
			if mode == "refund" {
				want = 2
			}
			if count != want {
				t.Fatalf("%s has %d scrolls, want%d", mode, count, want)
			}
		})
	}
}

func TestReleaseCurrentActionScriptSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_CURRENT_ACTION_INPUT"); path != "" {
		currentActionCold(t, path)
		return
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Current cell entry", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("current script SAV")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	id := f.live.mission.ids[0]
	e, _ := f.live.entity(id)
	// Explicit current map fixture, installed through the normal construction
	// API. The pending cast is created only by real ground-footprint entry.
	// This is a native persistence witness, not an original-runtime receipt.
	tail := sim.CellTail{X: e.X + 1, Y: e.Y, Bytes: [6]byte{13, 1, byte(e.X), byte(e.Y), byte(e.X + 1), byte(e.Y)}}
	if err := f.live.world.DeclareCellTails([]sim.CellTail{tail}); err != nil {
		t.Fatal(err)
	}
	f.live.enqueue(uint32(id), int(e.X+1), int(e.Y))
	for n := 0; n < 120 && len(f.live.world.ScriptCasts()) == 0; n++ {
		f.live.tick()
	}
	casts := f.live.world.ScriptCasts()
	if len(casts) != 1 || !casts[0].AtUnit || casts[0].Target != id || casts[0].Spell != 13 {
		t.Fatalf("ordinary cell entry did not queue its cast: %v", casts)
	}
	t.Logf("runtime%d actual cell-entry cast %+v", e.SourceBinding.RuntimeID, casts[0])
	currentActionSaveProof(t, f, e.SourceBinding.RuntimeID, e.ID, "script-entry")
}

func TestReleaseCurrentActionBookSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_CURRENT_ACTION_INPUT"); path != "" {
		currentActionCold(t, path)
		return
	}
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	dir := os.Getenv("AGAINROM_OBJECT_AGS_DIR")
	if dir == "" {
		dir = filepath.Join(filepath.Dir(os.Getenv("AGAINROM_ASSETS")), "..", "engine", "saves")
	}
	path := filepath.Join(dir, "122323.ags")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != "903efb70bf0e042dc90a3b6b3bbd9ce36aca83534dbc662d4b12c678d0477702" {
		t.Fatal("original input changed")
	}
	defer func() {
		after, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(after) != digest {
			t.Fatal("witness changed original input")
		}
	}()
	s, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := f.Restore(s)
	if err != nil || town {
		t.Fatal(err)
	}
	if err = f.App("current book charge").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	caster := worldEntityByRuntimeID(t, f, 6)
	f.live.attackOrCast(uint32(caster.ID), 0, 7, 64, 118, true)
	for range 3 {
		f.live.tick()
	}
	books := f.live.world.Actions().Books
	found := false
	for _, c := range books {
		if c.Caster == caster.ID && c.Spell == 7 && c.Remaining > 0 && c.Paid {
			found = true
		}
	}
	if !found {
		t.Fatalf("normal MapAttack did not reach paid book charge: %v", books)
	}
	currentActionSaveProof(t, f, caster.SourceBinding.RuntimeID, caster.ID, "book-charge")
}
