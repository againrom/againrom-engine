package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
)

func TestPresentationSeedKeepsEachRuleSeed(t *testing.T) {
	native := &mapWorld{world: visualIdentityWorld(t, []sim.EntityID{81, 84}, 84)}
	live := liveCastIdentity(sim.CastEvent{Caster: 84, Target: 81, Spell: 6})
	if got := native.presentationSeed(live); got != 1238302146 {
		t.Fatalf("live seed = %d", got)
	}
	point := liveCastIdentity(sim.CastEvent{Caster: 84, Target: 0, Spell: 2, AtCell: true})
	if got := native.presentationSeed(point); got != castSeed(84, 0, 2) {
		t.Fatalf("point cast seed = %d", got)
	}

	rows, next := native.currentVisualIdentities()
	for i := range rows {
		rows[i].Entity -= 14
	}
	cold := &mapWorld{world: visualIdentityWorld(t, []sim.EntityID{67, 70}, 70)}
	cold.restoreVisualIdentities(rows, next)
	if got := cold.presentationSeed(liveCastIdentity(sim.CastEvent{Caster: 70, Target: 67, Spell: 6})); got != 1238302146 {
		t.Fatalf("relabelled live seed = %d", got)
	}
	if got := cold.presentationSeed(presentationIdentity{rule: seedTargetMark, target: 67, spell: 3}); got != castSeed(0, 81, 3) {
		t.Fatalf("target mark seed = %d", got)
	}

	for i := range rows {
		if rows[i].Entity == 67 {
			rows[i].Entity = 0
		}
	}
	zero := &mapWorld{world: visualIdentityWorld(t, []sim.EntityID{0, 70}, 70)}
	zero.restoreVisualIdentities(rows, next)
	if got := zero.presentationSeed(presentationIdentity{rule: seedTargetMark, target: 70, spell: 1}); got != castSeed(0, 84, 1) {
		t.Fatalf("target mark took actor zero's label: %d", got)
	}

	from, to := image.Pt(3, 200), image.Pt(17, 9)
	script := presentationIdentity{rule: seedScript, from: from, to: to, index: 4}
	want := uint32(3|200<<8) ^ uint32(17|9<<8)<<16 ^ uint32(5)
	if got := native.presentationSeed(script); got != want {
		t.Fatalf("script seed = %d, want %d", got, want)
	}

	for _, record := range []uint16{0, 1, 41, 65535} {
		got := native.presentationSeed(presentationIdentity{rule: seedRestored, record: record})
		if got != uint32(record)+1 || got != restoredDefaultSeed(record) {
			t.Fatalf("restored seed of record %d = %d", record, got)
		}
	}
	if (&mapWorld{}).presentationSeed(presentationIdentity{rule: seedRestored, record: 9}) != 10 {
		t.Fatal("restored rule needs a World")
	}
}
