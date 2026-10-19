package game

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func numeralMission(t *testing.T, f *FrontEnd, mission int) *ui.App {
	t.Helper()
	f.SetDeterministicFrames(true)
	a := f.App("damage numerals")
	a.SetCutscenes(nil)
	held := false
	opener := f.MissionOpenerWith(mission, MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table))
	if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := opener()
		drive := func() {
			if !held {
				tick()
			}
		}
		return v, drive, order, cadence, affect, advance, attack, grab, stance, march, err
	}); err != nil {
		t.Fatal(err)
	}
	a.Layout(1024, 768)
	for n := 0; a.HeadlessNoticeOpen() && n < 32; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if a.HeadlessNoticeOpen() {
		t.Fatal("mission notice remained open")
	}
	held = true
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	return a
}

func numeralWitnessOutput(t *testing.T, f *FrontEnd, name string, proof any, cell image.Point) {
	t.Helper()
	out := os.Getenv("AGAINROM_DAMAGE_NUMERAL_WITNESS_DIR")
	if out == "" {
		return
	}
	dir, err := effectRimOutputPath(f.Archives.Root, out)
	if err != nil {
		t.Fatal(err)
	}
	dir = filepath.Join(dir, filepath.Base(f.Archives.Root))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	frame, _, err := f.live.view.HeadlessMapFrame(cell, 320, 224)
	if err != nil {
		t.Fatal(err)
	}
	launchWritePNG(t, dir, name+".png", frame)
	raw, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseDamageNumeralsKeepRepeatedGarrisonTicks(t *testing.T) {
	f := releaseFront(t)
	a := numeralMission(t, f, 41)
	mw := f.live
	hero := mw.mission.ids[0]
	anchor := sim.EntityID(9)
	first, ok := mw.world.Entity(anchor)
	if !ok {
		t.Fatal("no garrison anchor")
	}
	stray, ok := mw.world.Entity(28)
	if !ok {
		t.Fatal("no roaming pack")
	}
	for _, e := range mw.world.Entities() {
		if e.Owner == stray.Owner && e.Group == stray.Group {
			mw.pending = append(mw.pending, sim.Kill(e.ID))
		}
	}
	if err := mw.world.HeadlessPlace(hero, 52, 61); err != nil {
		t.Fatal(err)
	}
	mw.pending = append(mw.pending, sim.Attack(hero, anchor))
	type hit struct {
		Tick          uint64
		Victim        sim.EntityID
		Before, After int32
	}
	var hits []hit
	expected := map[[2]uint64]int{}
	var repeated uint32
	var figures []ui.DamageNumeralState
	for batch := 0; batch < 500 && repeated == 0; batch++ {
		for range 8 {
			h, _ := mw.world.Entity(hero)
			if h.HP > 0 && h.HP < h.MaxHP/2 {
				if err := mw.world.HeadlessHeal(hero); err != nil {
					t.Fatal(err)
				}
				mw.push()
			}
			mw.tickStep(nil, true, func(report sim.Report) {
				for _, e := range report.Damages {
					if e.BeforeHP > e.AfterHP {
						hits = append(hits, hit{report.Tick, e.Target, e.BeforeHP, e.AfterHP})
						expected[[2]uint64{uint64(e.Target), report.Tick}] += int(e.BeforeHP - e.AfterHP)
					}
				}
			})
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		figures = mw.view.DamageNumeralStates()
		counts := map[uint32]int{}
		for _, n := range figures {
			if n.Amount != expected[[2]uint64{uint64(n.Victim), n.Tick}] {
				t.Fatalf("causal amount mismatch %+v expected%v", n, expected)
			}
			counts[n.Victim]++
		}
		// Two live records for the same victim reject the per-frame and life-window merge rivals.
		for _, n := range figures {
			if counts[n.Victim] >= 2 {
				repeated = n.Victim
				break
			}
		}
	}
	if repeated == 0 {
		t.Fatalf("mission41 group%d owner%d: no repeated live victim; hits=%+v", first.Group, first.Owner, hits)
	}
	actor, _ := mw.world.Entity(sim.EntityID(repeated))
	before, err := mw.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	hash := mw.world.Hash()
	mw.push()
	mw.push()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	after, err := mw.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || hash != mw.world.Hash() {
		t.Fatal("presentation changed canonical bytes/hash")
	}
	t.Logf("mission41 garrison group%d victim%d hits%+v figures%+v hash%x", first.Group, repeated, hits, figures, hash)
	proof := struct {
		Mission       int
		Victim        uint32
		Hits          []hit
		Figures       []ui.DamageNumeralState
		Hash          uint64
		Qualification string
	}{41, repeated, hits, figures, hash, "Installed mission41 garrison; hero placed at gate, roaming pack removed, hero healed below half. Eight real simulation ticks per headless presentation frame. No injected damage on the repeated victim. Snapshot replay preserves World bytes/hash. CPU 320x224 crop omits shroud/HUD and GPU blend modes."}
	numeralWitnessOutput(t, f, "garrison", proof, image.Pt(int(actor.X), int(actor.Y)))
}

func numeralCommandControls(t *testing.T, f *FrontEnd, mission int) {
	t.Helper()
	a := numeralMission(t, f, mission)
	mw := f.live
	id := mw.mission.ids[0]
	if err := mw.world.HeadlessHeal(id); err != nil {
		t.Fatal(err)
	}
	mw.push()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	mw.pending = append(mw.pending, sim.Damage(id, 7), sim.Damage(id, 13))
	mw.tick()
	tick := mw.world.Tick()
	mw.pending = append(mw.pending, sim.Damage(id, 3))
	mw.tick()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	var figures []ui.DamageNumeralState
	for _, n := range mw.view.DamageNumeralStates() {
		if n.Victim == uint32(id) {
			figures = append(figures, n)
		}
	}
	if len(figures) != 2 || figures[0].Tick != tick || figures[0].Amount != 20 || figures[1].Tick != tick+1 || figures[1].Amount != 3 {
		t.Fatalf("same-tick merge/different-tick loss control: %+v", figures)
	}
	e, _ := mw.world.Entity(id)
	numeralWitnessOutput(t, f, "commands", figures, image.Pt(int(e.X), int(e.Y)))
	t.Logf("mission%d victim%d command controls: %+v", mission, id, figures)
}

func TestReleaseDamageNumeralsMergeCommandsWithinRawTick(t *testing.T) {
	numeralCommandControls(t, releaseFront(t), 20)
}
func TestReleaseSecondGameDamageNumeralsMergeCommandsWithinRawTick(t *testing.T) {
	numeralCommandControls(t, secondGameFront(t), 10)
}
