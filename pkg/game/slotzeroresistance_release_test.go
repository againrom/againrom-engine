package game

import (
	"bytes"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The corpus save carries two Humans whose six damage-kind bytes
// (live +0xce..+0xd3 and modifier +0x10e..+0x113) are all 100, the output of
// the `+god` order (HERO-MODDK-161). HERO-DKIDX-162: an attacker with active
// slot 0 reads the target byte at +0xce, and 100 leaves no damage.
const slotZeroCorpusSAV = "2026-08-15/game0018.sav"
const slotZeroCorpusSHA256 = "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b"

func TestReleaseSlotZeroStrikeOnCorpusGodActorThroughSaveAndColdLoad(t *testing.T) {
	_, raw := groundCorpusFile(t, slotZeroCorpusSAV, slotZeroCorpusSHA256)
	f, _ := coldMission(t, raw)
	target, attacker := slotZeroWitnessPair(t, f.live.world)
	inputBlocks := slotZeroSavedBlocks(t, raw)
	if len(inputBlocks) != 2 {
		t.Fatalf("%s actor records with +0xce at 100 = %d, want the 2 corpus Humans", slotZeroCorpusSAV, len(inputBlocks))
	}

	type result struct {
		tick          int
		dealt, contrl int32
	}
	strike := func(label string, target, attacker sim.Entity, value uint8) result {
		t.Helper()
		target.ActorLoad.Source.Defence[16] = value
		tick, dealt, control := slotZeroLockstepBlow(t, attacker, target)
		want := (control*(100-int32(value)) + 75) / 100
		if control <= 0 || dealt != want {
			t.Fatalf("%s: slot-0 strike on +0xce=%d dealt %d against control %d, want ftol(%d*%d/100+0.75)=%d",
				label, value, dealt, control, control, 100-int32(value), want)
		}
		t.Logf("%s: attacker %d (slot %d) on actor %d, +0xce=%d: tick %d, control %d, dealt %d",
			label, attacker.ID, attacker.XPSlot, target.ID, value, tick, control, dealt)
		return result{tick, dealt, control}
	}
	loaded := strike("loaded", target, attacker, 100)
	half := strike("loaded, byte at 50", target, attacker, 50)
	if loaded.dealt != 0 || half.dealt == 0 || half.dealt >= half.contrl {
		t.Fatalf("slot-0 results loaded=%+v half=%+v, want 0 at 100 and a reduced nonzero blow at 50", loaded, half)
	}

	written, doc, _ := saveCurrentEffect(t, f)
	writtenBlocks := slotZeroSavedBlocks(t, written)
	if len(writtenBlocks) != len(inputBlocks) {
		t.Fatalf("written SAV carries %d actor records with +0xce at 100, input %d", len(writtenBlocks), len(inputBlocks))
	}
	for i := range inputBlocks {
		if !bytes.Equal(inputBlocks[i], writtenBlocks[i]) {
			t.Fatalf("damage-kind bytes did not round-trip: input %x, written %x", inputBlocks[i], writtenBlocks[i])
		}
	}
	if len(doc.Objects) == 0 {
		t.Fatal("written SAV has no objects")
	}

	cold, _ := coldMission(t, written)
	coldTarget, coldAttacker := slotZeroWitnessPair(t, cold.live.world)
	if coldTarget.SourceNow().Defence != target.SourceNow().Defence || coldTarget.SourceNow().Modifier != target.SourceNow().Modifier {
		t.Fatal("cold LOAD changed the target's damage-kind blocks")
	}
	after := strike("cold LOAD", coldTarget, coldAttacker, 100)
	if after != loaded {
		t.Fatalf("cold LOAD strike %+v, before SAVE %+v", after, loaded)
	}
}

// slotZeroWitnessPair names the first living actor whose +0xce byte is 100
// and the living loaded actor with active slot 0 and the largest physical pair.
func slotZeroWitnessPair(t *testing.T, w *sim.World) (target, attacker sim.Entity) {
	t.Helper()
	var haveTarget, haveAttacker bool
	for _, e := range w.Entities() {
		s := e.SourceNow()
		if !haveTarget && e.HP > 0 && s.Class != 0 && s.Defence[16] == 100 && s.Modifier[58] == 100 {
			target, haveTarget = e, true
		}
		if e.HP > 0 && e.XPSlot == 0 && e.ActorLoad.Source.Class != 0 && s.Defence[16] == 0 && e.DamageBase > 0 &&
			(!haveAttacker || e.DamageBase+e.DamageSpread > attacker.DamageBase+attacker.DamageSpread) {
			attacker, haveAttacker = e, true
		}
	}
	if !haveTarget || !haveAttacker {
		t.Fatalf("witness pair: target %v attacker %v", haveTarget, haveAttacker)
	}
	return target, attacker
}

// slotZeroSavedBlocks returns, in record order, the live defence block (UBE)
// and modifier damage-kind bytes of each actor record whose +0xce is 100.
func slotZeroSavedBlocks(t *testing.T, raw []byte) [][]byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var out [][]byte
	for i := range doc.Objects {
		live, err := savedActorRaw(&doc.Objects[i], "UBE", 22)
		if err != nil || live[16] != 100 {
			continue
		}
		modifier, err := savedActorRaw(&doc.Objects[i], "UD4", 64)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, slices.Concat(live, modifier[46:64]))
	}
	return out
}

// slotZeroLockstepBlow strikes target with attacker in two isolated worlds,
// one with the target's +0xce byte as given and one with it at zero, and
// returns the first blow's damage in each.
func slotZeroLockstepBlow(t *testing.T, loadedAttacker, loadedTarget sim.Entity) (tick int, dealt, control int32) {
	t.Helper()
	attacker := cleanResistanceWitnessEntity(loadedAttacker, 1, 5, 5)
	attacker.Owner = sim.SelfSlot
	target := cleanResistanceWitnessEntity(loadedTarget, 2, 6, 5)
	target.Owner = 0
	controlTarget := target
	controlTarget.ActorLoad.Source.Defence[16] = 0
	build := func(victim sim.Entity) *sim.World {
		t.Helper()
		w, err := sim.NewWorld(1, sim.Bounds{Width: 20, Height: 20}, sim.ModeCanonical, nil, []sim.Entity{attacker, victim})
		if err != nil {
			t.Fatalf("isolated slot-0 world: %v", err)
		}
		mapload.BindSourceDerive(w)
		return w
	}
	resisted, plain := build(target), build(controlTarget)
	start := target.HP
	for step := 1; step <= 4096; step++ {
		var cmds []sim.Command
		if step == 1 {
			cmds = []sim.Command{sim.Attack(attacker.ID, target.ID)}
		}
		sim.Step(resisted, cmds)
		sim.Step(plain, cmds)
		if hp := entityByID(t, plain, target.ID).HP; hp != start {
			return step, start - entityByID(t, resisted, target.ID).HP, start - hp
		}
	}
	t.Fatal("slot-0 attacker landed no blow in 4096 ticks")
	return 0, 0, 0
}
