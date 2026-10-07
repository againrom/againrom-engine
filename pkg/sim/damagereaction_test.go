package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func distantSpellWorld(t *testing.T) (*World, SpellRule) {
	t.Helper()
	caster := spMage(1, 16, 24, 20, 200, 200, 1<<1)
	caster.Owner = SelfSlot
	victim := engFighter(2, 2, 24, 24)
	victim.ScanRange, victim.HP, victim.MaxHP = 1, 200, 200
	rule := SpellRule{ID: 1, MaxRange: 12, TargetsUnit: true, Damaging: true, DamageMin: 4, DamageMax: 4}
	w, err := NewSpelledWorld(1, engBounds, ModeCanonical, nil, []Entity{caster, victim}, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	w.relations = engRel(t, [3]uint32{1, 1, 2}, [3]uint32{2, 2, 2})
	return w, rule
}

func TestDistantSpellHitLetsVictimAcquireCaster(t *testing.T) {
	w, _ := distantSpellWorld(t)
	if w.actorSees(1, cellOf(w.entities[0])) {
		t.Fatal("caster must start outside victim sight")
	}
	if events := spRunCast(w, spCast(1, 2, 1)); len(events) == 0 || w.entities[1].HP >= 200 {
		t.Fatal("spell did not hit")
	}
	for range 32 {
		Step(w, nil)
	}
	if target, ok := engVictim(w, 2); !ok || target != 1 {
		t.Fatalf("victim did not acquire distant caster: %d %t", target, ok)
	}
}

func TestDistantAreaDamageReachesOrdinaryGroupAcquisition(t *testing.T) {
	w, rule := distantSpellWorld(t)
	if !w.ordinaryAreaEffect(0, 1, rule, 0) || w.entities[1].HP >= 200 {
		t.Fatal("area damage did not land")
	}
	if w.entities[1].HasAttackTarget {
		t.Fatal("damage hook assigned a target directly")
	}
	engRun(w, 1)
	if target, ok := engVictim(w, 2); !ok || target != 1 {
		t.Fatal("area victim did not acquire caster")
	}
}

func TestAttackNoticeExpiresByCandidateBuildsAndDoesNotRevealFog(t *testing.T) {
	w, _ := distantSpellWorld(t)
	fog := w.Sight(SelfSlot)
	w.flipOnBlow(0, 1)
	before := w.Hash()
	for range 100 {
		if !w.actorSees(1, cellOf(w.entities[0])) {
			t.Fatal("AI query lost the attacker cell")
		}
	}
	if w.Hash() != before || !bytes.Equal(fog, w.Sight(SelfSlot)) {
		t.Fatal("perception query consumed state or revealed fog")
	}
	g := aiGroup{owner: 2, members: []int{1}}
	for scan := 1; scan <= 21; scan++ {
		if got := w.candidates(g); len(got) != 1 || got[0] != 0 {
			t.Fatal("notice expired before its last scan", scan, got)
		}
		if got := w.attackNoticeAt(1); got.Scans != uint8(scan) || (got.Cell == 0) != (scan == 21) {
			t.Fatal("notice lifetime", scan, got)
		}
	}
	if got := w.candidates(g); len(got) != 0 {
		t.Fatal("expired cell remained visible", got)
	}
	w.entities[0].X = 15
	w.flipOnBlow(0, 1)
	if got := w.attackNoticeAt(1); got != (attackNotice{Cell: 0x180f}) {
		t.Fatal("new blow did not replace and reset notice", got)
	}
}

func TestAttackNoticeUsesOriginalOrderFieldsAndTransfersNativeOwnership(t *testing.T) {
	w, _ := distantSpellWorld(t)
	w.flipOnBlow(0, 1)
	if err := w.ImportSavedGroups(nil, nil); err != nil {
		t.Fatal(err)
	}
	w.ensureSavedOrder(1)
	o := w.savedOrder(2)
	if w.entities[1].attackNotice != (attackNotice{}) || binary.LittleEndian.Uint16(o.Raw[0x58:]) != 0x1810 {
		t.Fatal("native notice was duplicated or dropped")
	}
	o.Raw[0x5a] = 17
	before := o.Raw
	w.entities[0].X = 14
	w.flipOnBlow(0, 1)
	want := before
	want[0x54], want[0x58], want[0x59], want[0x5a] = 1, 14, 24, 0
	if o.Raw != want || o.State != 0 || w.entities[1].HasAttackTarget {
		t.Fatal("reaction changed order fields beyond the alarm")
	}
	data := mustMarshal(t, w)
	if binary.LittleEndian.Uint32(data[len(data)-4:]) != 0 {
		t.Fatal("original order also acquired a native notice record")
	}
	var back World
	if err := back.UnmarshalBinary(data); err != nil || back.Hash() != w.Hash() {
		t.Fatal("original notice resume", err)
	}
	for range 21 {
		w.candidates(aiGroup{owner: 2, members: []int{1}})
		back.candidates(aiGroup{owner: 2, members: []int{1}})
		if w.Hash() != back.Hash() {
			t.Fatal("original notice aged differently after resume")
		}
	}
}

func TestAttackNoticeKeepsAcquisitionLimits(t *testing.T) {
	for _, name := range []string{"allied", "invisible", "off-map", "owner-zero", "moved"} {
		t.Run(name, func(t *testing.T) {
			w, _ := distantSpellWorld(t)
			if name == "allied" {
				w.relations.Set(2, 1, 2)
			}
			if name == "owner-zero" {
				w.entities[0].Owner = 0
			}
			w.flipOnBlow(0, 1)
			switch name {
			case "invisible":
				w.attached = []attachedEffect{{Target: 1, Spell: 15, Remaining: 32}}
			case "off-map":
				w.entities[0].OffMap = true
			case "moved":
				w.entities[0].X = 15
			}
			if got := w.candidates(aiGroup{owner: 2, members: []int{1}}); len(got) != 0 {
				t.Fatal("notice bypassed acquisition filter", got)
			}
		})
	}
}

func TestPoisonDamageAlertsOnlyWithLivingKnownSource(t *testing.T) {
	for _, name := range []string{"known", "casterless", "dead-source", "zero-damage"} {
		t.Run(name, func(t *testing.T) {
			w, rule := distantSpellWorld(t)
			rule.ID = 8
			w.spells = []SpellRule{rule}
			w.attached = []attachedEffect{{Target: 2, Caster: 1, HasCaster: true, Spell: 8}}
			damage := int64(4)
			switch name {
			case "casterless":
				w.attached[0].HasCaster, w.attached[0].Caster = false, 0
			case "dead-source":
				w.entities[0].HP = -1
			case "zero-damage":
				damage = 0
			}
			w.awardPoisonTick(0, 1, damage)
			if (w.attackNoticeAt(1).Cell != 0) != (name == "known") {
				t.Fatal("poison invented or lost an attacker")
			}
		})
	}
}

func TestAttackNoticeDoesNotOverrideStandGroundOrHealing(t *testing.T) {
	w, rule := distantSpellWorld(t)
	for i := range w.groups {
		w.groups[i].order = orderStandGround
	}
	w.ordinaryAreaEffect(0, 1, rule, 0)
	engRun(w, 1)
	if w.entities[1].HasAttackTarget || w.entities[1].HasTarget {
		t.Fatal("notice bypassed stand-ground reach")
	}
	w, _ = distantSpellWorld(t)
	w.entities[1].HP = 100
	if !w.ordinaryEffect(0, 1, SpellRule{ID: 12, TargetsUnit: true, Restorative: true, DamageMin: 4, DamageMax: 4}, 0) {
		t.Fatal("heal did not apply")
	}
	if w.entities[1].HP <= 100 || w.attackNoticeAt(1) != (attackNotice{}) || w.relations.Hostile(2, 1) {
		t.Fatal("healing created hostility")
	}
}
