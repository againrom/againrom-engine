package game

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Installed columns feed the production constructor, command and native SAVE
// envelope. The actor placement is synthetic; no install or owner save writes.
func TestReleaseSpellDelivery1183InstalledRulesAndNativeSave(t *testing.T) {
	f := releaseFront(t)
	rules := mapload.SpellRules(f.Table)
	if len(rules) != 28 {
		t.Fatal("installed spell population", len(rules))
	}
	for _, r := range rules {
		want := int32(1)
		if r.ID == 1 || r.ID == 2 || r.ID == 13 || r.ID == 14 {
			want = 2
		}
		if r.Delivery != want {
			t.Fatal("installed delivery column", r.ID, r.Delivery)
		}
	}
	for _, id := range []uint16{1, 2, 13} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			rule := rules[id-1]
			caster := sim.Entity{ID: 1, Owner: sim.SelfSlot, X: 1, Y: 1, HP: 100, MaxHP: 100, Mana: 200, MaxMana: 200, ScanRange: 20, KnownSpells: uint32(1) << id, AttackCharge: 8}
			target := sim.Entity{ID: 2, Owner: 2, X: 5, Y: 1, HP: 1000, MaxHP: 1000, DyingTime: 200}
			w, err := sim.NewSpelledWorld(1183, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{caster, target}, nil, rules)
			if err != nil {
				t.Fatal(err)
			}
			cmd := sim.Command{Kind: sim.KindCast, Entity: 1, X: 2, Y: int32(id)}
			if rule.Area {
				cmd = sim.Command{Kind: sim.KindCastAt, Entity: 1, X: 5, Y: 1, Spell: id}
			}
			sim.Step(w, []sim.Command{cmd})
			if w.Entities()[0].Mana != 200-rule.ManaCost {
				t.Fatal("admission mana", w.Entities()[0].Mana, rule.ManaCost)
			}
			for step := 0; w.PendingSpellDeliveries() == 0 && step < 128; step++ {
				sim.Step(w, nil)
			}
			if w.PendingSpellDeliveries() != 1 || w.Entities()[1].HP != 1000 || w.Entities()[1].SpellFX != 0 {
				t.Fatal("pending delivery or early impact", w.PendingSpellDeliveries(), w.Entities()[1].HP)
			}
			state := &SnapshotSAVDocument{Version: 1}
			if err := projectSavedWorldEffects(state, w); err != nil {
				t.Fatal(err)
			}
			if err := admitEffects(state, w); err != nil {
				t.Fatal("released delivery refused", err)
			}
			b, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			payload, err := EncodeSave(Snapshot{World: b}, "in flight")
			if err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			name, err := store.Write(time.Unix(1183, 0), payload)
			if err != nil {
				t.Fatal(err)
			}
			read, err := (SaveStore{Dir: store.Dir}).Read(name)
			if err != nil || !bytes.Equal(payload, read) {
				t.Fatal("native file", err)
			}
			snapshot, _, err := DecodeSave(read)
			if err != nil {
				t.Fatal(err)
			}
			var back sim.World
			if err := back.UnmarshalBinary(snapshot.World); err != nil {
				t.Fatal(err)
			}
			steps := int(4 * 256 / rule.EffectSpeed)
			if id == 13 {
				steps = 10
			}
			steps++ // A tail transport appends its child for the next list pass.
			for step := 1; step <= steps; step++ {
				sim.Step(w, nil)
				sim.StepObserved(&back, nil)
				if w.Hash() != back.Hash() || step < steps && w.Entities()[1].HP != 1000 {
					t.Fatal("LOAD or early payload", step)
				}
			}
			if w.Entities()[1].HP >= 1000 || w.PendingSpellDeliveries() != 0 {
				t.Fatal("missing delivered damage")
			}
			if err := projectSavedWorldEffects(state, w); err != nil || strings.Contains(state.WorldEffects.Unavailable, "pending native spell deliveries") {
				t.Fatal("stale delivery refusal", err)
			}
			t.Logf("installed spell%d: mana=%d, flight=%d, native file=%s, final HP=%d", id, rule.ManaCost, steps, name, w.Entities()[1].HP)
		})
	}
}
