package game

import (
	"fmt"
	"testing"

	"againrom/pkg/sim"
)

func nativeAreaPulseContinuation(t *testing.T) {
	for _, spell := range []uint16{7, 8, 12, 17} {
		t.Run(fmt.Sprint(spell), func(t *testing.T) {
			_, raw := groundCorpusFile(t, "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd")
			f := releaseFront(t)
			f.Options = OptionsStore{}
			openWorldEffectsTestSave(t, f, raw, "area-pulse-source.sav")
			var target sim.EntityID
			for _, e := range f.live.world.ActiveEffects() {
				if e.Spell == 12 && e.Remaining == 12 {
					target = e.Target
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
			if err := f.live.world.HeadlessPlace(target, 19, 40); err != nil {
				t.Fatal(err)
			}
			runtime := releaseEntity(t, f.live, target).SourceBinding.RuntimeID
			script, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantCastAtCell, Args: [10]int32{19, 41, 19, 40, int32(spell), 30}}}, []sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 999}})
			if err != nil {
				t.Fatal(err)
			}
			installTestScript(t, f, script)
			for tick := 0; ; tick++ {
				books, scrolls, scripts := f.live.world.NativeCastContinuations()
				areas := f.live.world.CellEffects()
				if len(areas) == 1 && books+scrolls+scripts == 0 && f.live.world.PendingSpellDeliveries() == 0 && areas[0].Remaining%16 == 1 {
					break
				}
				if tick > 256 {
					t.Fatal("area did not reach next-pulse save cut")
				}
				f.live.tick()
			}
			installTestScript(t, f, quiet)
			path := saveCorpseMission(t, f, t.TempDir())
			cold := loadAreaContinuation(t, path)
			installTestScript(t, cold, quiet)
			f.live.tick()
			cold.live.tick()
			live := releaseEntity(t, f.live, target)
			var loaded sim.Entity
			for _, e := range cold.live.world.Entities() {
				if e.SourceBinding.RuntimeID == runtime {
					loaded = e
					break
				}
			}
			t.Logf("spell%d next pulse: HP native=%d loaded=%d; speed native=%d loaded=%d; scan native=%d loaded=%d; marker native=%d/%d loaded=%d/%d", spell, live.HP, loaded.HP, live.Speed, loaded.Speed, live.ScanRange, loaded.ScanRange, live.SpellFXSpell, live.SpellFX, loaded.SpellFXSpell, loaded.SpellFX)
			if live.SpellFX != loaded.SpellFX || live.SpellFXSpell != loaded.SpellFXSpell || live.HP != loaded.HP || live.Speed != loaded.Speed || live.ScanRange != loaded.ScanRange {
				t.Error("ordinary SAV LOAD changed the next area's successful-application marker")
			}
		})
	}
}
