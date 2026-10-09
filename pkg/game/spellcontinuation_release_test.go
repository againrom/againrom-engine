package game

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

type spellVictimWitness struct {
	Target        uint32
	Before, After int32
	Payload       sim.SavedEffect
}

type spellContinuationWitness struct {
	Victims    []spellVictimWitness
	Remaining  uint16
	ImpactTick uint16
	Spell      uint16
}

func TestReleaseSpellSAVContinuation(t *testing.T) {
	for _, spell := range []uint16{1, 2, 13, 14} {
		t.Run(fmt.Sprint(spell), func(t *testing.T) { spellSAVContinuation(t, spell) })
	}
}

func spellSAVContinuation(t *testing.T, spell uint16) {
	if path := os.Getenv("AGAINROM_SPELL_SAV_INPUT"); path != "" {
		var want spellContinuationWitness
		proof, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(proof, &want); err != nil {
			t.Fatal(err)
		}
		f := loadAreaContinuation(t, path)
		g := f.live.world.SavedSpellGraph()
		if g != nil {
			if len(g.Roots) != len(want.Victims) {
				t.Fatal("cold transport graph population changed", g)
			}
			for i, id := range g.Roots {
				n := g.Nodes[id-1]
				victim := want.Victims[i]
				if n.Value.ST4C != want.Remaining || n.Primary == 0 {
					t.Fatal("current countdown/child lost", n)
				}
				child := g.Nodes[n.Primary-1]
				payload := child.Value.PE48
				if payload == nil {
					payload = child.Value.AE44
				}
				if payload == nil || *payload != victim.Payload {
					t.Fatal("frozen payload changed", payload, victim.Payload)
				}
				if child.Value.Class == "PointEffect" && releaseEntity(t, f.live, child.Target).SourceBinding.RuntimeID != victim.Target {
					t.Fatal("fan target/order changed")
				}
				if got := world1170Entity(t, f, victim.Target).HP; got != victim.Before {
					t.Fatal("LOAD applied spell", got, victim.Before)
				}
			}
		} else {
			// A current SAV may carry this continuation as a native delivery.
			// Its ordinary transport/child nodes are retired during LOAD.
			deliveries := f.live.world.NativeSpellDeliverySaveStates()
			if len(deliveries) != len(want.Victims) {
				t.Fatalf("cold native delivery population %d, want %d", len(deliveries), len(want.Victims))
			}
			for i, d := range deliveries {
				victim := want.Victims[i]
				if d.Remaining != want.Remaining || d.Area.Payload != victim.Payload {
					t.Fatalf("cold native delivery %d changed: timer %d/%d payload %+v/%+v", i, d.Remaining, want.Remaining, d.Area.Payload, victim.Payload)
				}
				if !d.AtCell && releaseEntity(t, f.live, d.Target).SourceBinding.RuntimeID != victim.Target {
					t.Fatal("native fan target/order changed")
				}
				if got := world1170Entity(t, f, victim.Target).HP; got != victim.Before {
					t.Fatal("LOAD applied spell", got, victim.Before)
				}
			}
		}
		for tick := uint16(1); tick <= want.ImpactTick; tick++ {
			f.live.tick()
			for _, v := range want.Victims {
				got := world1170Entity(t, f, v.Target).HP
				if tick < want.ImpactTick && got != v.Before {
					t.Fatal("early effect", tick, got)
				}
				if tick == want.ImpactTick && got != v.After {
					t.Fatal("payload differs from uninterrupted native play", got, v.After)
				}
			}
			if tick == want.Remaining && tick < want.ImpactTick {
				checkpoint := saveCorpseMission(t, f, t.TempDir())
				f = loadAreaContinuation(t, checkpoint)
				for _, effect := range f.live.world.SavedSpellEffects() {
					if effect.Class == "SpellTransport" {
						t.Fatal("handed-off child was reconstructed as a transport")
					}
				}
				if got := f.live.world.NativeSpellDeliverySaveStates(); len(got) != len(want.Victims) {
					t.Fatalf("handed-off native delivery population %d, want %d", len(got), len(want.Victims))
				}
			}
		}
		for _, effect := range f.live.world.SavedSpellEffects() {
			if effect.Class == "SpellTransport" || effect.Class == "PointEffect" {
				t.Fatal("delivered roots retained")
			}
		}
		second := saveCorpseMission(t, f, t.TempDir())
		f = loadAreaContinuation(t, second)
		for range 3 {
			f.live.tick()
		}
		for _, v := range want.Victims {
			if got := world1170Entity(t, f, v.Target).HP; got != v.After {
				t.Fatal("second LOAD replayed payload", got, v.After)
			}
		}
		t.Logf("spell%d: current timer %d, cold targets/payloads %v; second SAV and no replay", spell, want.Remaining, want.Victims)

		return
	}
	f := spellWitnessSource(t)
	var target sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.Alive() && e.HP > 100 && e.SourceBinding.RuntimeID != 0 {
			target = e
			break
		}
	}
	x, y := int32(19), int32(40)
	if spell == 14 {
		for _, e := range f.live.world.Entities() {
			count := 0
			for _, v := range f.live.world.Entities() {
				if v.Owner == e.Owner && v.Alive() && v.HP > 30 && v.SourceBinding.RuntimeID != 0 {
					count++
				}
			}
			if e.Alive() && e.HP > 30 && e.SourceBinding.RuntimeID != 0 && count >= 3 &&
				f.live.world.Relations().Hostile(sim.SelfSlot, e.Owner) {
				target = e
				break
			}
		}
	}
	if target.SourceBinding.RuntimeID == 0 {
		t.Fatal("positive fixture target absent")
	}
	if err := f.live.world.HeadlessPlace(target.ID, x, y); err != nil {
		t.Fatal(err)
	}
	var sprayCandidates []sim.EntityID
	if spell == 14 {
		// The player's other units stand clear: the group's members close on
		// the nearest of them and settle on its contact ring (MOVE-ALT-020), which would walk
		// the row's actors off the cells the fan is laid on.
		for _, e := range f.live.world.Entities() {
			if e.Owner == sim.SelfSlot && e.Alive() && (e.MaxMana == 0 || !e.Book.HasInstances()) {
				placed := false
				for _, far := range [][2]int32{{x + 25, y}, {x - 25, y}, {x, y - 25}, {x + 25, y - 25}, {x - 25, y - 25}, {x, y + 25}} {
					if f.live.world.HeadlessPlace(e.ID, far[0], far[1]) == nil {
						placed = true
						break
					}
				}
				if !placed {
					t.Fatal("no cell clear of the fan for player unit", e.ID)
				}
			}
		}
		// The row is laid from the cell the placement chose.
		placed, _ := f.live.entity(target.ID)
		x, y = placed.X, placed.Y
		count := 0
		for _, e := range f.live.world.Entities() {
			if e.ID == target.ID || e.Owner != target.Owner || !e.Alive() || e.HP <= 30 || e.SourceBinding.RuntimeID == 0 {
				continue
			}
			// Caster at (x-1, y), primary at (x, y), both candidates at edge
			// distance 2: the first behind, the second ahead. The turn term
			// picks the second; list order picked the first.
			at := [2][2]int32{{x - 1, y + 2}, {x - 3, y + 1}}[count]
			sprayRowWalk(t, f.live, e.ID, at[0], at[1])
			if e, _ := f.live.entity(e.ID); e.X != at[0] || e.Y != at[1] {
				t.Fatal("fan secondary did not reach its cell", e.ID, e.X, e.Y)
			}
			sprayCandidates = append(sprayCandidates, e.ID)
			count++
			if count == 2 {
				break
			}
		}
		if count != 2 {
			t.Fatal("fan fixture lacks two secondary actors")
		}
	}
	instant := sim.ScriptInstant{Op: sim.ScriptInstantCastAtUnit, Args: [10]int32{x - 6, y, int32(spell), 30}, HasUnit: true, Unit: target.ID}
	if spell == 2 {
		instant = sim.ScriptInstant{Op: sim.ScriptInstantCastAtCell, Args: [10]int32{x - 6, y, x, y, int32(spell), 30}}
	}
	script, err := sim.NewScript(nil, []sim.ScriptInstant{instant}, []sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 999}})
	if err != nil {
		t.Fatal(err)
	}
	if spell == 14 {
		// A caster-less script spray selects the primary alone; the party
		// mage casts instead, so its group's view supplies the fan.
		script, _ = sim.NewScript(nil, nil, nil)
		castSprayFromPartyMage(t, f, target.ID, x-1, y)
	}
	installTestScript(t, f, script)
	for tick := 0; f.live.world.PendingSpellDeliveries() == 0; tick++ {
		if tick > 256 {
			t.Fatal("cast did not release delivery")
		}
		f.live.tick()
	}
	// The spray's fan is prepared at admission while its caster still charges.
	books, scrolls, scripts := f.live.world.NativeCastContinuations()
	if books+scrolls+scripts != 0 && spell != 14 {
		t.Fatal("fixture remains in an unreleased action", books, scrolls, scripts)
	}
	deliveries := f.live.world.NativeSpellDeliverySaveStates()
	if spell == 14 && len(deliveries) != 2 {
		t.Fatal("fan release count", len(deliveries))
	}
	if spell == 14 {
		chosen := releaseEntity(t, f.live, deliveries[1].Target).ID
		if chosen != sprayCandidates[1] {
			t.Fatalf("fan secondary %d, want the equally distant candidate nearest the caster's heading %d (list order gives %d)", chosen, sprayCandidates[1], sprayCandidates[0])
		}
	}
	want := spellContinuationWitness{Remaining: deliveries[0].Remaining, ImpactTick: deliveries[0].Remaining + 1, Spell: spell}
	if len(deliveries) > 1 {
		want.ImpactTick = want.Remaining
	}
	for _, d := range deliveries {
		runtime := target.SourceBinding.RuntimeID
		if !d.AtCell {
			runtime = releaseEntity(t, f.live, d.Target).SourceBinding.RuntimeID
		}
		want.Victims = append(want.Victims, spellVictimWitness{Target: runtime, Before: world1170Entity(t, f, runtime).HP, Payload: d.Area.Payload})
	}
	path := saveCorpseMission(t, f, t.TempDir())
	for range int(want.ImpactTick) {
		f.live.tick()
	}
	for i := range want.Victims {
		v := &want.Victims[i]
		v.After = world1170Entity(t, f, v.Target).HP
		if v.After >= v.Before {
			t.Fatal("native positive witness did no damage", *v)
		}
	}
	proof, _ := json.MarshalIndent(want, "", "  ")
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	emitSpellWitness(t, path, fmt.Sprintf("spell%d", spell), proof)
	runSpellWitnessChild(t, path, "AGAINROM_SPELL_SAV_INPUT")

}

func spellWitnessSource(t *testing.T) *FrontEnd {
	t.Helper()
	_, raw := groundCorpusFile(t, "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd")
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.App("native delivery SAV").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if !f.live.world.SetAutoHealing(sim.SelfSlot, 0) {
		t.Fatal("disable ambient healing")
	}
	quiet, _ := sim.NewScript(nil, nil, nil)
	installTestScript(t, f, quiet)
	for range 40 {
		f.live.tick()
	}

	return f
}

func runSpellWitnessChild(t *testing.T, path, variable string) {
	t.Helper()
	parts := strings.Split(t.Name(), "/")
	for i := range parts {
		parts[i] = "^" + regexp.QuoteMeta(parts[i]) + "$"
	}
	cmd := exec.Command(os.Args[0], "-test.run="+strings.Join(parts, "/"), "-test.v")
	cmd.Env = append(os.Environ(), variable+"="+path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh-process delivery: %v\n%s", err, output)
	}
	t.Logf("%s", output)
}

func emitSpellWitness(t *testing.T, path, name string, proof []byte) {
	t.Helper()
	dir := os.Getenv("AGAINROM_SPELL_WITNESS_DIR")
	if dir == "" {
		return
	}
	if !filepath.IsAbs(dir) {
		t.Fatal("spell witness directory must be absolute")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), proof, 0600); err != nil {
		t.Fatal(err)
	}
}

type originalSpellWitness struct {
	Projectile sim.SavedProjectile
	Counter    uint16
}

func originalSpellDeadHP(t *testing.T, f *FrontEnd, runtime uint32) int32 {
	t.Helper()
	for _, d := range f.live.world.OriginalDeadActors() {
		if d.Source.State.RuntimeID == runtime {
			return int32(d.Current.HP)
		}
	}
	t.Fatalf("original dead actor runtime %d missing", runtime)
	return 0
}

func originalSpellSAVContinuation(t *testing.T) {
	if path := os.Getenv("AGAINROM_ORIGINAL_SPELL_INPUT"); path != "" {
		var want originalSpellWitness
		proof, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(proof, &want); err != nil {
			t.Fatal(err)
		}
		f := loadAreaContinuation(t, path)
		got := f.live.world.SavedProjectiles()
		graph := f.live.world.SavedSpellGraph()
		if len(got.Items) != 1 || got.Items[0] != want.Projectile || graph == nil || graph.Nodes[graph.Roots[0]-1].Value.ST4C != want.Counter {
			t.Fatal("original current graphics/counter lost")
		}
		if got := originalSpellDeadHP(t, f, 157); got != -57 {
			t.Fatal("dead-target fixture changed", got)
		}
		for range 4 {
			f.live.tick()
		}
		if len(f.live.world.SavedProjectiles().Items) != 0 || len(f.live.world.SavedSpellEffects()) != 0 {
			t.Fatal("original delivery/graphics did not retire")
		}
		second := saveCorpseMission(t, f, t.TempDir())
		f = loadAreaContinuation(t, second)
		f.live.tick()
		if got := originalSpellDeadHP(t, f, 157); got != -57 {
			t.Fatal("dead target received replay", got)
		}
		t.Log("original0018: current counter2 and projectile segment1, cold completion, dead HP-57 unchanged, second SAV")
		return
	}
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.App("projectile continuation").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		f.live.tick()
	}
	current := f.live.world.SavedProjectiles()
	if len(current.Items) != 1 || current.Items[0].ActionSegments != 1 {
		t.Fatal("current projectile", current)
	}
	path := saveCorpseMission(t, f, t.TempDir())
	proof, _ := json.MarshalIndent(originalSpellWitness{Projectile: current.Items[0], Counter: 2}, "", "  ")
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	emitSpellWitness(t, path, "original0018-current", proof)
	runSpellWitnessChild(t, path, "AGAINROM_ORIGINAL_SPELL_INPUT")
}

// castSprayFromPartyMage teaches the fixture's party mage Prismatic Spray at
// the installed cost and orders it at target from (x, y).
func castSprayFromPartyMage(t *testing.T, f *FrontEnd, target sim.EntityID, x, y int32) {
	t.Helper()
	rule := f.live.world.Spells()[13]
	for _, e := range f.live.world.Entities() {
		if e.Owner != sim.SelfSlot || !e.Alive() || e.MaxMana == 0 || !e.Book.HasInstances() {
			continue
		}
		book := e.Book
		book.Slots[13] = sim.BookSpell{Range: 10, ManaCost: uint16(rule.ManaCost)}
		if err := f.live.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: e.ID, KnownSpells: e.KnownSpells | 1<<14, Book: book}}); err != nil {
			t.Fatal(err)
		}
		sprayRowWalk(t, f.live, e.ID, x, y)
		f.live.pending = append(f.live.pending, sim.Cast(e.ID, target, 14))
		return
	}
	t.Fatal("fixture has no party mage")
}

// sprayRowWalk places the actor near (x, y) and walks it there, returning on
// the tick it first stands on the cell. The shared walk helpers also wait out
// the crossing; those ticks let the spray's primary start its return to its
// distant post before the cast releases, and the fan then loses it.
func sprayRowWalk(t *testing.T, mw *mapWorld, id sim.EntityID, x, y int32) {
	t.Helper()
	if err := mw.world.HeadlessPlace(id, x, y); err != nil {
		t.Fatal(err)
	}
	if e, _ := mw.entity(id); e.X == x && e.Y == y {
		return
	}
	mw.enqueue(uint32(id), int(x), int(y))
	for n := 0; ; n++ {
		mw.tick()
		e, ok := mw.entity(id)
		if !ok || e.X == x && e.Y == y || !e.HasTarget && e.Transit == 0 {
			return
		}
		if n >= headlessReachLimit {
			t.Fatalf("actor %d still walking at (%d,%d) toward (%d,%d)", id, e.X, e.Y, x, y)
		}
	}
}
