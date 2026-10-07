package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
)

func TestFallenBodyRestartsItsSpriteJoltOnEveryDamageReport(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 7, X: 2, Y: 2, HP: 0, MaxHP: 20},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, died: map[sim.EntityID]int{}, hurt: map[sim.EntityID]int{}}

	hit := func(amount int32) image.Point {
		report := sim.StepReported(w, []sim.Command{{Kind: sim.KindDamage, Entity: 7, X: amount}})
		mw.observeDamage(report.Damages)
		mw.scene++
		draws := mw.entityDraws()
		if len(draws) != 1 {
			t.Fatalf("entityDraws returned %d entries, want 1", len(draws))
		}
		if !draws[0].Selectable {
			t.Fatalf("damaged body at HP %d did not cross selectable", draws[0].HP)
		}
		return draws[0].DamageJolt
	}

	if got := hit(1); got != fallenDamageJolt[0] {
		t.Fatalf("first body hit starts at %v, want %v", got, fallenDamageJolt[0])
	}
	// A second hit before the first pattern finishes restarts phase zero rather
	// than continuing phase one.
	if got := hit(1); got != fallenDamageJolt[0] {
		t.Fatalf("second body hit restarted at %v, want %v", got, fallenDamageJolt[0])
	}

	for phase := 1; phase < len(fallenDamageJolt); phase++ {
		report := sim.StepReported(w, nil)
		mw.observeDamage(report.Damages)
		mw.scene++
		if got := mw.damageJolt(7); got != fallenDamageJolt[phase] {
			t.Fatalf("jolt phase %d = %v, want %v", phase, got, fallenDamageJolt[phase])
		}
	}
	report := sim.StepReported(w, nil)
	mw.observeDamage(report.Damages)
	mw.scene++
	if got := mw.damageJolt(7); got != (image.Point{}) {
		t.Fatalf("expired body jolt = %v, want zero", got)
	}
	if len(mw.hurt) != 0 {
		t.Fatalf("expired body jolt retained %d clock entries", len(mw.hurt))
	}
}

func TestPassiveCorpseDecayNeverStartsTheDamageJolt(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 7, X: 2, Y: 2, HP: -1, MaxHP: 20, Decay: sim.DecayFallen},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, died: map[sim.EntityID]int{}, hurt: map[sim.EntityID]int{}}
	lastHP := w.Entities()[0].HP
	decayed := false
	for step := 0; step < 64; step++ {
		report := sim.StepReported(w, nil)
		mw.observeDamage(report.Damages)
		mw.scene++
		if len(report.Damages) != 0 || mw.damageJolt(7) != (image.Point{}) {
			t.Fatalf("passive step %d produced damage/jolt: events=%+v jolt=%v",
				step, report.Damages, mw.damageJolt(7))
		}
		hp := w.Entities()[0].HP
		if hp < lastHP {
			decayed = true
		}
		lastHP = hp
	}
	if !decayed {
		t.Fatal("fixture never reached a passive corpse-decay decrement")
	}
}

func TestFinishingBlowStillPlaysTheFallenBodysLastJolt(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 7, X: 2, Y: 2, HP: -9, MaxHP: 20, Decay: sim.DecayFallen},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, died: map[sim.EntityID]int{}, hurt: map[sim.EntityID]int{}}
	report := sim.StepReported(w, []sim.Command{{Kind: sim.KindDamage, Entity: 7, X: 1}})
	mw.observeDamage(report.Damages)
	mw.scene++
	draw := mw.entityDraws()[0]
	if draw.HP != -10 || draw.Selectable {
		t.Fatalf("finishing blow left HP/selectable at %d/%v, want -10/false", draw.HP, draw.Selectable)
	}
	if draw.DamageJolt != fallenDamageJolt[0] {
		t.Fatalf("finishing blow jolt = %v, want %v", draw.DamageJolt, fallenDamageJolt[0])
	}
}
