package game

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

type areaContinuationWitness struct {
	Area        sim.CellEffect
	Purse       uint32
	Attachments []areaAttachmentWitness
}

type areaAttachmentWitness struct {
	Runtime       uint32
	Spell         uint16
	Kind          sim.EffectKind
	Mode          sim.EffectMode
	Magnitude     int32
	Remaining     uint16
	HasCaster     bool
	CasterRuntime uint32
}

func areaAttachments(w *sim.World) []areaAttachmentWitness {
	var result []areaAttachmentWitness
	for _, effect := range w.ActiveEffects() {
		for _, entity := range w.Entities() {
			if entity.ID == effect.Target {
				caster := uint32(0)
				if effect.HasCaster {
					for _, actor := range w.Entities() {
						if actor.ID == effect.Caster {
							caster = actor.SourceBinding.RuntimeID
							break
						}
					}
				}
				result = append(result, areaAttachmentWitness{entity.SourceBinding.RuntimeID, effect.Spell, effect.Kind, effect.Mode, effect.Magnitude, effect.Remaining, effect.HasCaster, caster})
				break
			}
		}
	}
	return result
}

func loadAreaContinuation(t *testing.T, path string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("area SAV LOAD town=%t: %v", town, err)
	}
	if err := f.App("area SAV continuation").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func requireAreaClock(t *testing.T, f *FrontEnd, want sim.CellEffect, tick int) {
	t.Helper()
	drivers := f.live.world.SavedWorldEffectDrivers()
	alive := tick <= int(want.Remaining)
	timer, stage := int(want.Remaining)-tick, 0
	if want.Mode == sim.AreaModeRing {
		limits := map[uint16]int{4: 2, 9: 6, 21: 32}
		phase := int(want.Phase) + tick
		alive = phase < 3*(limits[want.Spell]-1)
		timer, stage = 2-phase%3, phase/3+1
	}
	if drivers == nil || len(drivers.Areas) == 0 {
		// A current SAV may retire the ordinary AreaEffect graph on LOAD and
		// carry the same clock as a native area row instead.
		areas, err := f.live.world.NativeAreaSaveStates()
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, area := range areas {
			if area.Spell != want.Spell || !alive || int(area.Remaining) != timer || int(area.Stage) != stage {
				continue
			}
			count++
			if want.Mode == sim.AreaModeCloud {
				var cells [][2]int32
				for _, key := range area.Cells {
					cells = append(cells, [2]int32{int32(key & 255), int32(key >> 8)})
				}
				if !reflect.DeepEqual(cells, want.Cells) {
					t.Fatalf("current native cloud coverage changed at tick%d: %+v", tick, cells)
				}
			}
		}
		if (count == 1) != alive {
			t.Fatalf("tick%d: %d native current areas, alive=%t", tick, count, alive)
		}
		return
	}
	count := 0
	for _, driver := range drivers.Areas {
		if driver.Root < 0 {
			continue
		}
		count++
		effect := f.live.world.SavedSpellEffects()[driver.Root]
		if !alive || driver.Spell != want.Spell || driver.Mode != want.Mode || int(effect.AE4C) != timer || int(effect.AE48[3]) != stage {
			t.Fatalf("tick%d: area driver=%+v fields=%+v want alive%t timer%d stage%d", tick, driver, effect, alive, timer, stage)
		}
		if want.Mode == sim.AreaModeCloud {
			var cells [][2]int32
			for _, key := range driver.Cells {
				cells = append(cells, [2]int32{int32(key & 255), int32(key >> 8)})
			}
			if !reflect.DeepEqual(cells, want.Cells) {
				t.Fatal("current cloud coverage changed", tick)
			}
		}
	}
	if (count == 1) != alive {
		t.Fatalf("tick%d: %d current areas, alive=%t", tick, count, alive)
	}
}

func TestReleaseNativeAreaSAVContinuation(t *testing.T) {
	for _, spell := range []uint16{3, 4, 7, 8, 9, 12, 17, 19, 21} {
		t.Run(fmt.Sprint(spell), func(t *testing.T) { nativeAreaSAVWitness(t, spell, false) })
	}
	t.Run("caster-shield-and-light", func(t *testing.T) { nativeAreaSAVWitness(t, 12, true) })
	t.Run("next-pulse", nativeAreaPulseContinuation)
}

func nativeAreaSAVWitness(t *testing.T, spell uint16, actorCast bool) {
	if path := os.Getenv("AGAINROM_AREA_SAV_INPUT"); path != "" {
		proof, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var want areaContinuationWitness
		if err := json.Unmarshal(proof, &want); err != nil {
			t.Fatal(err)
		}
		f := loadAreaContinuation(t, path)
		requireAreaClock(t, f, want.Area, 0)
		if f.live.world.Purse(sim.SelfSlot) != want.Purse || !reflect.DeepEqual(areaAttachments(f.live.world), want.Attachments) {
			t.Fatalf("initial attachment/purse changed: %+v want %+v", areaAttachments(f.live.world), want.Attachments)
		}
		end := int(want.Area.Remaining) + 1
		if want.Area.Mode == sim.AreaModeRing {
			end = 96
		}
		if end > 1200 {
			t.Fatal("unbounded witness duration", end)
		}
		for tick := 1; tick <= end; tick++ {
			f.live.tick()
			requireAreaClock(t, f, want.Area, tick)
			if tick == 1 {
				attachments := areaAttachments(f.live.world)
				second := saveCorpseMission(t, f, t.TempDir())
				f = loadAreaContinuation(t, second)
				requireAreaClock(t, f, want.Area, tick)
				if !reflect.DeepEqual(areaAttachments(f.live.world), attachments) {
					t.Fatal("second SAV changed attachments")
				}
			}
		}
		t.Logf("spell%d: two SAV LOADs retain current clock, payload and cells; expired after %d checked ticks", spell, end)
		return
	}
	_, raw := groundCorpusFile(t, "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd")
	f := releaseFront(t)
	f.Options = OptionsStore{}
	app, _, _ := openWorldEffectsTestSave(t, f, raw, "area-source.sav")
	_ = app
	var caster sim.EntityID
	for _, e := range f.live.world.ActiveEffects() {
		if e.Spell == 12 && e.Remaining == 12 {
			caster = e.Target
		}
	}
	quiet, err := sim.NewScript(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	installTestScript(t, f, quiet)
	for range 40 {
		f.live.tick()
	}
	if len(f.live.world.CellEffects()) != 0 {
		t.Fatal("starting area has not expired")
	}
	if actorCast {
		if err := f.live.world.HeadlessPlace(caster, 19, 40); err != nil {
			t.Fatal(err)
		}
		f.live.attackOrCast(uint32(caster), uint32(caster), 18, 19, 40, false)
		for tick := 0; ; tick++ {
			found := false
			for _, e := range f.live.world.ActiveEffects() {
				found = found || e.Spell == 18 && e.HasCaster && e.Caster == caster
			}
			if found && releaseEntity(t, f.live, caster).CastWait == 0 {
				break
			}
			if tick > 256 {
				t.Fatal("real Shield cast did not create its attachment")
			}
			f.live.tick()
		}
		f.live.attackOrCast(uint32(caster), 0, 12, 19, 40, true)
	} else {
		script, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantCastAtCell, Args: [10]int32{19, 41, 19, 40, int32(spell), 30}}}, []sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 999}})
		if err != nil {
			t.Fatal(err)
		}
		installTestScript(t, f, script)
	}
	for tick := 0; ; tick++ {
		books, scrolls, scripts := f.live.world.NativeCastContinuations()
		if f.live.world.HasNativeAreaEffects() && books+scrolls+scripts == 0 && f.live.world.PendingSpellDeliveries() == 0 {
			break
		}
		if tick > 256 {
			t.Fatal("cast never reached a completed area", spell)
		}
		f.live.tick()
	}
	if actorCast {
		for range 16 {
			f.live.tick()
		}
		found := false
		for _, e := range f.live.world.ActiveEffects() {
			if e.HasCaster && e.Caster == caster {
				found = true
			}
		}
		if !found {
			t.Fatal("caster-bearing Shield expired before the area checkpoint")
		}
	}
	areas := f.live.world.CellEffects()
	if len(areas) != 1 {
		t.Fatal("expected one newly created area", areas)
	}
	want := areaContinuationWitness{Area: areas[0], Purse: f.live.world.Purse(sim.SelfSlot), Attachments: areaAttachments(f.live.world)}
	path := saveCorpseMission(t, f, t.TempDir())
	proof, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(t.Name(), "/")
	for i := range parts {
		parts[i] = "^" + regexp.QuoteMeta(parts[i]) + "$"
	}
	cmd := exec.Command(os.Args[0], "-test.run="+strings.Join(parts, "/"), "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_AREA_SAV_INPUT="+filepath.Clean(path))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh-process area continuation: %v\n%s", err, output)
	}
	t.Logf("%s", output)
}
