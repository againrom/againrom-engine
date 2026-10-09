package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func consumedCorpseScript(t *testing.T, id sim.EntityID) *sim.Script {
	t.Helper()
	s, err := sim.NewScript([]sim.ScriptCheck{
		{Op: sim.ScriptCheckHealth, Register: 0, Unit: id, HasUnit: true, Args: [10]int32{6}},
		{Op: sim.ScriptCheckAlive, Register: 1, Unit: id, HasUnit: true},
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func consumedCorpseSave(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore, name string) []byte {
	t.Helper()
	if app.Screen() == ui.ScreenMap {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(store.Dir, name, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if state, open := app.HeadlessSaveState(); open && state.Confirmation {
		if err := app.HeadlessSaveAction("overwrite"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatal("ordinary SAVE did not finish", app.Screen(), app.HeadlessMessage())
	}
	raw, err := store.Read(name + ".sav")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func consumedCorpseCold(t *testing.T, store SaveStore, name string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("consumed corpse cold LOAD")
	app.Layout(1024, 768)
	app.SetCutscenes(nil)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	_, list, _ := f.SaveSeams(store, OriginalStore{}, nil)
	groundAppLoad(t, app, list, name+".sav")
	return f, app
}

// consumedCorpseTuple checks the consumed body's terminal tuple and, when
// present is set, its record. The SAVE writes a departed actor's record when
// its map placement constructs it (DIV-2501).
func consumedCorpseTuple(t *testing.T, raw []byte, want sim.CurrentTerminalActor, present bool) uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || a.Values[want.ID].CurrentTerminal == nil {
		t.Fatal("raw SAV lacks consumed terminal marker", err)
	}
	got := *a.Values[want.ID].CurrentTerminal
	var object uint16
	bindings := 0
	for _, b := range a.Bindings {
		if !b.Structure && b.ID == want.ID {
			bindings++
			object = b.Object
			if b.Missing == present {
				t.Fatal("terminal binding changed its native-object presence", b)
			}
		}
	}
	if bindings != 1 || present != (object != 0) {
		t.Fatal("terminal binding is not unique", bindings, object)
	}
	if present {
		rooted := false
		for _, root := range doc.DeadActors {
			rooted = rooted || root == object
		}
		if !rooted {
			t.Fatal("raw native terminal is absent from DeadActors")
		}
	}
	var identity uint32
	if present {
		r := &doc.Objects[object-1]
		stage, e1 := savedStructureValue(r, "Stage")
		hp, e2 := savedStructureValue(r, "Health")
		p, e3 := savedMotionRaw(r, "Block12", 12)
		identity, err = savedStructureValue(r, "Identity")
		if e1 != nil || e2 != nil || e3 != nil || err != nil || len(p) != 12 || identity == 0 {
			t.Fatal("raw native consumed record", e1, e2, e3, err)
		}
		if int16(hp) != int16(want.HP) || stage != uint32(want.Stage) || binary.LittleEndian.Uint16(p) != want.Cell || binary.LittleEndian.Uint16(p[2:]) != want.Cell {
			t.Fatalf("production SAV native tuple: Health=%d Stage=%d Cell=%04x, want Health=%d Stage=%d Cell=%04x", int16(hp), stage, binary.LittleEndian.Uint16(p), want.HP, want.Stage, want.Cell)
		}
	}
	if got != want {
		t.Fatalf("raw current-action tuple %+v, want %+v", got, want)
	}
	return identity
}

func consumedCorpseLoadControl(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore, raw []byte, id sim.EntityID, stage, markerStage uint8, wantError bool, bound bool) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("native%d-marker%d", stage, markerStage)
	doc.Label = []byte(name)
	valid, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	a.Values[id].CurrentTerminal.Stage = markerStage
	for _, binding := range a.Bindings {
		if !binding.Structure && binding.ID == id && binding.Object != 0 {
			if err := savedActorSetValue(&doc.Objects[binding.Object-1], "Stage", uint32(stage)); err != nil {
				t.Fatal(err)
			}
		}
	}
	encoded, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
		t.Fatal(err)
	}
	bad, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, name+".sav")
	if err := os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	_, list, _ := f.SaveSeams(store, OriginalStore{}, nil)
	label := ""
	for _, row := range list() {
		if row.Name == localOriginalSaveToken(name+".sav") {
			label = row.Label
		}
	}
	if label == "" {
		t.Fatal("corrupt subject absent from ordinary LOAD list")
	}
	world, hash := f.live.world, f.live.world.Hash()
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	if !wantError {
		want := *a.Values[id].CurrentTerminal
		rows := f.live.world.CurrentTerminalActors()
		if app.Screen() != ui.ScreenMap || f.live.world == world || len(rows) != 1 || rows[0] != want {
			t.Fatal("readable stage/current-marker control", app.Screen(), rows, want)
		}
		if bound {
			sourceDoc, err := sav.DecodeDocumentData(bad)
			if err != nil {
				t.Fatal(err)
			}
			r := generatedActorRecord(t, &sourceDoc, id)
			actualStage, _ := savedStructureValue(r, "Stage")
			hp, _ := savedStructureValue(r, "Health")
			if actualStage != uint32(stage) || int16(hp) != -10001 {
				t.Fatal("same-HP Stage2 source discriminator", actualStage, int16(hp))
			}
		}
		next := consumedCorpseSave(t, f, app, store, "readback-"+name)
		consumedCorpseTuple(t, next, want, true)
		return
	}
	if app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" || f.live.world != world || f.live.world.Hash() != hash {
		t.Fatal("corrupt terminal LOAD was accepted or changed the current World", stage, app.Screen(), app.HeadlessMessage())
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
}

// consumedCorpseHeldBytes makes the World hold base byte 22 and modifier
// byte 40 of one live actor at the given values.
func consumedCorpseHeldBytes(t *testing.T, w *sim.World, id sim.EntityID, base, modifier byte) {
	t.Helper()
	var basis sim.NativeActorBasis
	for _, e := range w.Entities() {
		if e.ID == id {
			basis = e.NativeBasis
		}
	}
	basis.BasePresent, basis.ModifierPresent = true, true
	basis.BaseKnown |= 1 << 22
	basis.ModifierKnown |= 1 << 40
	basis.Base[22], basis.Modifier[40] = base, modifier
	if err := w.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: id, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseConsumedCorpseSAVLoad(t *testing.T) {
	f := releaseFront(t)
	out := os.Getenv("AGAINROM_CONSUMED_CORPSE_WITNESS_DIR")
	if !filepath.IsAbs(out) || !filepath.IsAbs(f.Archives.Root) {
		t.Fatal("AGAINROM_CONSUMED_CORPSE_WITNESS_DIR must name an absolute existing directory outside the install")
	}
	out, err := editorPhysicalDirectory(out)
	if err != nil {
		t.Fatal(err)
	}
	install, err := editorPhysicalDirectory(f.Archives.Root)
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(install, out); err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("consumed corpse output is inside the install", out, err)
	}
	for _, bound := range []bool{true, false} {
		t.Run(fmt.Sprintf("bound-%t", bound), func(t *testing.T) {
			f := releaseFront(t)
			f.Options = OptionsStore{}
			f.SetDeterministicFrames(true)
			dir := filepath.Join(out, filepath.Base(install), fmt.Sprintf("bound-%t", bound))
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			saves, err := os.MkdirTemp(dir, "saves-")
			if err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: saves}
			app := f.App("consumed corpse ordinary SAV")
			app.Layout(1024, 768)
			app.SetCutscenes(nil)
			f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
			party := f.ChargenParty(ui.ChargenResult{Name: "Consumed corpse", Choices: []int{0, 1, 0}, Stats: []int{20, 25, 40, 40}})
			party[0].KnownSpells |= 1 << 25
			if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
				t.Fatal(err)
			}
			w := f.live.world
			var mage, victim sim.Entity
			mageFound, victimFound := false, false
			for _, e := range w.Entities() {
				if !mageFound && e.Owner == sim.SelfSlot && e.Humanoid {
					mage, mageFound = e, true
				}
				if !victimFound && e.Owner != sim.SelfSlot && e.Alive() && !e.Humanoid && e.TypeID != w.Ghost().TypeID {
					victim, victimFound = e, true
				}
			}
			if !mageFound || !victimFound || victim.SourceBinding.Class != 0 {
				t.Fatal("mission lacks native mage/creature", mage, victim)
			}
			placed := false
			for _, d := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if w.HeadlessPlace(victim.ID, mage.X+d[0], mage.Y+d[1]) == nil {
					placed = true
					break
				}
			}
			if !placed {
				t.Fatal("no free cell beside mage")
			}
			if err := w.RestoreScriptProgram(consumedCorpseScript(t, victim.ID)); err != nil {
				t.Fatal(err)
			}
			if err := w.HeadlessKill(victim.ID); err != nil {
				t.Fatal(err)
			}
			for range 256 {
				f.live.tick()
				if e, ok := liveEntity(f, victim.ID); ok && e.Decay == sim.DecayBones {
					victim = e
					break
				}
			}
			if victim.Decay != sim.DecayBones {
				t.Fatal("victim did not reach bones")
			}
			var baselineIdentity uint32
			if bound {
				// The loaded file holds 23/29 in two unknown-meaning bytes the
				// World also holds; the live values change before consumption.
				consumedCorpseHeldBytes(t, w, victim.ID, 23, 29)
				raw := consumedCorpseSave(t, f, app, store, "bones")
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				r := generatedActorRecord(t, &doc, victim.ID)
				baselineIdentity, _ = savedStructureValue(r, "Identity")
				stage, _ := savedStructureValue(r, "Stage")
				hp, _ := savedStructureValue(r, "Health")
				if stage != 2 || int16(hp) != int16(victim.HP) {
					t.Fatal("independent raw baseline is not current bones", stage, int16(hp), victim.HP)
				}
				f, app = consumedCorpseCold(t, store, "bones")
				w = f.live.world
				e, ok := liveEntity(f, victim.ID)
				if !ok || e.SourceBinding.Class != 0 {
					t.Fatal("current LOAD did not retain native policy", e)
				}
			}
			consumedCorpseHeldBytes(t, w, victim.ID, 96, 100)
			f.live.pending = append(f.live.pending, sim.Cast(mage.ID, victim.ID, 25))
			for range 256 {
				f.live.tick()
				if _, ok := liveEntity(f, victim.ID); !ok {
					break
				}
			}
			requireLiveRaisedGhost(t, f, w.Ghost().TypeID)
			want := sim.CurrentTerminalActor{ID: victim.ID, Cell: uint16(victim.X) | uint16(victim.Y)<<8, HP: -10001, Stage: 5, MapUnitID: victim.MapUnitID}
			hash := w.Hash()
			raw := consumedCorpseSave(t, f, app, store, "consumed")
			if err := os.WriteFile(filepath.Join(dir, "consumed.sav"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			identity := consumedCorpseTuple(t, raw, want, true)
			if bound && identity != baselineIdentity || w.Hash() != hash {
				t.Fatal("SAVE changed identity or live World", identity, baselineIdentity)
			}
			cold, coldApp := consumedCorpseCold(t, store, "consumed")
			rows := cold.live.world.CurrentTerminalActors()
			if len(rows) != 1 || rows[0] != want {
				t.Fatal("raw source -> cold terminal/identity", rows, want)
			}
			if !reflect.DeepEqual(w.RemovedNativeActorBases(), cold.live.world.RemovedNativeActorBases()) {
				t.Fatal("raw source -> cold removed native basis")
			}
			if cold.live.world.Tick() != w.Tick() || !reflect.DeepEqual(w.Actions(), cold.live.world.Actions()) || !reflect.DeepEqual(w.CurrentPolicy(), cold.live.world.CurrentPolicy()) {
				t.Fatal("cold LOAD changed tick/actions/policy")
			}
			ghostID := sim.EntityID(0)
			for _, e := range w.Entities() {
				got, ok := liveEntity(cold, e.ID)
				if !ok {
					t.Fatal("cold LOAD lost live actor", e.ID)
				}
				if e.TypeID == w.Ghost().TypeID && e.MapUnitID == 0 && e.Owner == sim.SelfSlot {
					ghostID = e.ID
					if e.Capacity != 300 {
						t.Fatal("raised Ghost lacks its Units row capacity", e.Capacity)
					}
				}
				if !reflect.DeepEqual(e, got) {
					t.Fatal("cold LOAD changed actor fields", e.ID)
				}
			}
			file, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := file.ActorGraph()
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			capacity, err := savedStructureValue(generatedActorRecord(t, &doc, ghostID), "Capacity")
			if err != nil || capacity != 300 || ghostID == 0 || len(graph.Actors) != len(w.Entities()) {
				t.Fatal("raw Ghost capacity/population control", capacity, len(graph.Actors), err)
			}
			for name, world := range map[string]*sim.World{"source-world.bin": w, "cold-world.bin": cold.live.world} {
				form, err := world.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, name), form, 0600); err != nil {
					t.Fatal(err)
				}
			}
			coldHash := cold.live.world.Hash()
			for range 256 {
				cold.live.tick()
			}
			if hp, alive := cold.live.world.ScriptRegister(0), cold.live.world.ScriptRegister(1); hp != -10001 || alive != 0 {
				t.Fatal("cold script Health/Alive", hp, alive)
			}
			requireLiveRaisedGhost(t, cold, w.Ghost().TypeID)
			again := consumedCorpseSave(t, cold, coldApp, store, "resaved")
			if err := os.WriteFile(filepath.Join(dir, "resaved.sav"), again, 0600); err != nil {
				t.Fatal(err)
			}
			if next := consumedCorpseTuple(t, again, want, true); next != identity {
				t.Fatal("resave changed native identity", next, identity)
			}
			controlFront, controlApp := consumedCorpseCold(t, store, "consumed")
			if bound {
				consumedCorpseLoadControl(t, controlFront, controlApp, store, raw, want.ID, 2, 2, false, true)
				consumedCorpseLoadControl(t, controlFront, controlApp, store, raw, want.ID, 2, 5, false, true)
			}
			controlWorld, controlHash := controlFront.live.world, controlFront.live.world.Hash()
			if err := headlessOpenLoad(controlApp); err != nil {
				t.Fatal(err)
			}
			if err := controlApp.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if controlFront.live.world != controlWorld || controlFront.live.world.Hash() != controlHash {
				t.Fatal("cancelled LOAD changed World")
			}
			consumedCorpseLoadControl(t, controlFront, controlApp, store, raw, want.ID, 6, 6, true, bound)
			proof, _ := json.MarshalIndent(struct {
				Tuple                    sim.CurrentTerminalActor
				Identity                 uint32
				SavedHash, ColdHash      uint64
				RawSHA256, ResavedSHA256 string
			}{want, identity, hash, coldHash, fmt.Sprintf("%x", sha256.Sum256(raw)), fmt.Sprintf("%x", sha256.Sum256(again))}, "", "  ")
			if err := os.WriteFile(filepath.Join(dir, "proof.json"), proof, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
}
