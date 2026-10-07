package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The participant's units that hold a loaded attack cycle in the original saved
// games of the corpus, given the orders whose setters store a destination or a
// state and no attack progress. The cycle resolves its blow first, and nothing
// loads a second cycle behind it that the order did not ask for
// (AI-ORDER-039, AI-RETREAT-272, HERO-CADENCE-112; DIV-1509, DIV-1563,
// DIV-1577). Each save is resumed through the ordinary original-save load and
// stepped by production ticks to the offset the order is given at, so the order
// meets the cycle at every stage of it.
var loadedCycleCorpusSaves = []string{
	"2026-08-02/game0009.sav",
	"2026-08-14/game0013.sav",
	"2026-10-02/projectiles-original-en/game0022.sav",
}

func corpusEntity(w *sim.World, id sim.EntityID) (sim.Entity, bool) {
	for _, e := range w.Entities() {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

// loadedCycleHolder resumes the save, steps offset ticks and returns the world
// with the first participant unit that then holds a loaded cycle on a unit and
// no destination, and a second living unit of the participant when one exists.
func loadedCycleHolder(t *testing.T, f *FrontEnd, raw []byte, offset int) (w *sim.World, holder sim.Entity, companion sim.EntityID, hasCompanion, ok bool) {
	t.Helper()
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
	if err != nil {
		t.Fatalf("ResumeOriginalSave: %v", err)
	}
	w = ms.World
	for range offset {
		sim.Step(w, nil)
	}
	for _, e := range w.Entities() {
		if e.Owner != sim.SelfSlot || !e.Alive() || e.OffMap {
			continue
		}
		if !ok && e.HasAttackTarget && e.AttackTargetKind == sim.AttackTargetUnit && e.AttackPhase != sim.AttackReady && !e.HasTarget && e.Transit == 0 {
			holder, ok = e, true
		} else if !hasCompanion {
			companion, hasCompanion = e.ID, true
		}
	}
	return w, holder, companion, hasCompanion, ok
}

func TestReleaseOriginalSaveLoadedCycleFinishesBeforeStateAndMoveOrders(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("AGAINROM_SAVE_CORPUS is not set")
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Skip("AGAINROM_ASSETS is not set")
	}
	f, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", assets, err)
	}
	f.SetDeterministicFrames(true)
	cleanupFrontAudio(t, f)

	type writer struct {
		name         string
		needsMate    bool
		order        func(h sim.Entity, mate sim.EntityID) sim.Command
		walksAway    bool
		loadsAllowed bool
	}
	writers := []writer{
		{"move", false, func(h sim.Entity, _ sim.EntityID) sim.Command {
			return sim.MoveTo(h.ID, sim.CellPoint{X: h.X - 3, Y: h.Y})
		}, true, false},
		{"defend a companion", true, func(h sim.Entity, mate sim.EntityID) sim.Command {
			return sim.GroupDefend(h.ID, mate, 1)
		}, false, false},
	}
	offsets := []int{0, 2, 4, 6, 8, 10, 12, 14, 16, 20, 24, 30}
	checked := 0
	for _, rel := range loadedCycleCorpusSaves {
		path := filepath.Join(corpus, filepath.FromSlash(rel))
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Logf("%s: not in this corpus, skipped", rel)
			continue
		}
		for _, wr := range writers {
			for _, offset := range offsets {
				w, h, mate, hasMate, ok := loadedCycleHolder(t, f, raw, offset)
				if !ok || wr.needsMate && !hasMate {
					continue
				}
				victim := h.AttackTarget
				// The same save stepped without the order says whether the cycle
				// ends on this very tick, which no order can keep.
				twin, _, _, _, _ := loadedCycleHolder(t, f, raw, offset)
				sim.Step(twin, nil)
				natural, _ := corpusEntity(twin, h.ID)
				sim.Step(w, []sim.Command{wr.order(h, mate)})
				got, _ := corpusEntity(w, h.ID)
				if natural.AttackPhase != sim.AttackReady && (!got.HasAttackTarget || got.AttackTarget != victim || got.AttackPhase == sim.AttackReady) {
					t.Fatalf("%s %s at offset %d: the order dropped the loaded cycle: victim %v/%d phase %d", rel, wr.name, offset, got.HasAttackTarget, got.AttackTarget, got.AttackPhase)
				}
				if natural.AttackPhase == sim.AttackReady {
					continue
				}
				checked++
				// Native continuation: the world decoded from the bytes it would
				// be saved as runs on to the same hash through the cycle's end.
				form, err := w.MarshalBinary()
				if err != nil {
					t.Fatalf("%s %s at offset %d: marshal after the order: %v", rel, wr.name, offset, err)
				}
				var control sim.World
				if err := control.UnmarshalBinary(form); err != nil {
					t.Fatalf("%s %s at offset %d: decode after the order: %v", rel, wr.name, offset, err)
				}
				mapload.BindSourceDerive(&control)
				if control.Hash() != w.Hash() {
					t.Fatalf("%s %s at offset %d: the decoded world's hash differs from the live world's", rel, wr.name, offset)
				}
				// The blow resolves, then nothing loads another cycle on the old
				// victim before the unit leaves its cell.
				home, applied, loads := got, h.AttackPhase != sim.AttackCharging || got.AttackPhase != sim.AttackCharging, 0
				prev := got.AttackPhase
				for n := range 200 {
					sim.Step(w, nil)
					if n < 60 {
						sim.Step(&control, nil)
						if control.Hash() != w.Hash() {
							t.Fatalf("%s %s at offset %d: the decoded world diverged %d ticks after the order", rel, wr.name, offset, n+1)
						}
					}
					now, alive := corpusEntity(w, h.ID)
					if !alive || !now.Alive() {
						break
					}
					if prev == sim.AttackCharging && now.AttackPhase != sim.AttackCharging {
						applied = true
					}
					if prev == sim.AttackReady && now.AttackPhase != sim.AttackReady {
						loads++
					}
					prev = now.AttackPhase
					if now.X != home.X || now.Y != home.Y {
						break
					}
				}
				if !applied {
					t.Errorf("%s %s at offset %d: the loaded blow never resolved", rel, wr.name, offset)
				}
				if loads != 0 {
					t.Errorf("%s %s at offset %d: %d more cycles loaded behind the order", rel, wr.name, offset, loads)
				}
			}
		}
	}
	if checked == 0 {
		t.Skip("the corpus holds none of the listed loaded-cycle saves")
	}
	t.Logf("%d order and offset trials met a loaded cycle", checked)
}
