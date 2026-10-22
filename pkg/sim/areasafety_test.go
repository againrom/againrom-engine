package sim

import "testing"

func safeAreaWorld(t *testing.T, far bool) *World {
	t.Helper()
	mage := acCaster(1, 4, 8, 500, 1<<2|1<<18, 2)
	mage.Owner, mage.MaxMana = SelfSlot, 500
	mage.AttackCharge, mage.AttackRelax = 1, 1
	mage.HP, mage.MaxHP = 1000, 1000
	near := spEnt(2, 5, 8)
	near.Owner, near.HP, near.MaxHP = 2, 5000, 5000
	ents := []Entity{mage, near}
	if far {
		other := near
		other.ID, other.X = 3, 10
		ents = append(ents, other)
	}
	rule := SpellRule{ID: 2, Area: true, Damaging: true, Radius: 1, AreaHits: AreaHitsAll, DamageMin: 20, DamageMax: 20, ManaCost: 3, MaxRange: 12}
	rel := acEnemies(t)
	rel.Set(2, SelfSlot, relationLocked)
	return hlWorld(t, 5, rel, []SpellRule{rule, acbShield()}, ents...)
}

func TestUnorderedAreaCastSparesItsSide(t *testing.T) {
	for _, attack := range []bool{false, true} {
		t.Run(map[bool]string{false: "idle", true: "acquired-attack"}[attack], func(t *testing.T) {
			w := safeAreaWorld(t, true)
			if attack {
				w.entities[0].HasAttackTarget = true
				w.entities[0].AttackTarget = 2
				w.entities[0].ActorState = actorStateGuard
			}
			event := safeAreaFirstCast(t, w)
			t.Logf("target=%d tick=%d World=%x", event.Target, w.Tick(), w.Hash())
			if spAt(t, w, 1).HP != 1000 || spAt(t, w, 2).HP != 5000 || spAt(t, w, 3).HP >= 5000 {
				t.Fatalf("unsafe selection: caster=%d near=%d far=%d", spAt(t, w, 1).HP, spAt(t, w, 2).HP, spAt(t, w, 3).HP)
			}
		})
	}
}

func TestUnorderedAreaCastWaitsAndTriesSupport(t *testing.T) {
	w := safeAreaWorld(t, false)
	w.entities[1].X = 6
	w.spells[0].Radius = 2
	w.autoCast(0)
	if spAt(t, w, 1).Mana != 500 || len(w.bookCasts) != 0 {
		t.Fatal("unsafe area cast began")
	}
	w.SetAutoCast(SelfSlot, AutoCastMinimum|AutoCastOwn)
	w.autoCast(0)
	if len(w.bookCasts) != 1 || w.bookCasts[0].Spell != 18 {
		t.Fatal("unsafe area row blocked offered point Shield")
	}
	safeAreaFirstCast(t, w)
	if _, ok := effectIndex(w.attached, 1, 18); !ok {
		t.Fatal("Shield did not land")
	}
}

func safeAreaFirstCast(t *testing.T, w *World) CastEvent {
	t.Helper()
	for range 80 {
		for _, event := range StepObserved(w, nil) {
			if event.Caster == 1 {
				return event
			}
		}
	}
	t.Fatal("mage did not cast")
	return CastEvent{}
}

func TestUnorderedAreaCastProtectsOwnAndAlliedFootprints(t *testing.T) {
	for _, owner := range []uint32{SelfSlot, 3} {
		w := safeAreaWorld(t, true)
		w.entities[0].X = 1
		friend := spEnt(4, 4, 7)
		friend.Owner = owner
		friend.TokenSize = 2
		w.entities = append(w.entities, friend)
		w.relations.Set(SelfSlot, 3, relationLocked)
		rule, _ := w.findSpell(2)
		if target, ok := w.autoCastTarget(0, rule); !ok || target != 3 {
			t.Fatalf("owner %d selected %d/%v", owner, target, ok)
		}
	}
}

func TestAreaSafetyLeavesOrderedAndComputerCasts(t *testing.T) {
	for _, kind := range []string{"unit-order", "cell-order", "attack-order", "computer"} {
		t.Run(kind, func(t *testing.T) {
			w := safeAreaWorld(t, true)
			var commands []Command
			switch kind {
			case "unit-order":
				commands = []Command{Cast(1, 2, 2)}
			case "cell-order":
				commands = []Command{CastAt(1, 2, CellPoint{X: 5, Y: 8})}
			case "attack-order":
				commands = []Command{Attack(1, 2)}
			case "computer":
				w.relations = acEnemies(t)
				w.entities[0].Owner = 2
				w.entities[1].Owner = SelfSlot
				w.entities[2].Owner = SelfSlot
			}
			Step(w, commands)
			for range 16 {
				Step(w, nil)
			}
			if spAt(t, w, 1).HP >= 1000 || spAt(t, w, 2).HP >= 5000 {
				t.Fatalf("control stopped adjacent cast: caster=%d near=%d", spAt(t, w, 1).HP, spAt(t, w, 2).HP)
			}
		})
	}
}

func TestUnorderedWeaponAreaCastSparesItsSide(t *testing.T) {
	w := safeAreaWorld(t, true)
	w.entities[0].WeaponSpell = 2
	w.entities[0].Reach = 6
	w.entities[0].AutoSpell = 0
	w.entities[0].KnownSpells = 0
	w.entities[0].Book.State = BookAbsent
	w.entities[0].HasAttackTarget = true
	w.entities[0].AttackTarget = 2
	w.entities[0].ActorState = actorStateGuard
	safeAreaFirstCast(t, w)
	if spAt(t, w, 1).HP != 1000 || spAt(t, w, 2).HP != 5000 || spAt(t, w, 3).HP >= 5000 {
		t.Fatalf("weapon unsafe selection: caster=%d near=%d far=%d", spAt(t, w, 1).HP, spAt(t, w, 2).HP, spAt(t, w, 3).HP)
	}
}

func TestAreaSafetyUsesHitPolicyAndDoesNotDraw(t *testing.T) {
	for _, policy := range []AreaHits{AreaHitsAll, AreaHitsNotOwn, AreaHitsHostile} {
		w := safeAreaWorld(t, true)
		rule, _ := w.findSpell(2)
		rule.AreaHits = policy
		before := w.Hash()
		got := w.unsafeUnorderedArea(0, 1, rule)
		if got != (policy == AreaHitsAll) || w.Hash() != before {
			t.Fatalf("policy %d: unsafe=%v or state changed", policy, got)
		}
		friend := spEnt(4, 5, 7)
		friend.Owner = 3
		w.entities = append(w.entities, friend)
		w.relations.Set(SelfSlot, 3, relationLocked)
		got = w.unsafeUnorderedArea(0, 1, rule)
		if got != (policy != AreaHitsHostile) {
			t.Fatalf("ally policy %d: unsafe=%v", policy, got)
		}
	}
	w := safeAreaWorld(t, true)
	rule, _ := w.findSpell(2)
	w.rom2 = &rom2ScriptState{}
	if w.unsafeUnorderedArea(0, 1, rule) {
		t.Fatal("second-game cast was filtered")
	}
}

func TestAreaSafetyUsesLandingShapes(t *testing.T) {
	for _, arm := range []uint16{2, 3, 4, 9, 19, 21} {
		w := safeAreaWorld(t, true)
		w.bounds = Bounds{Width: 48, Height: 48}
		w.entities[0].X, w.entities[0].Y = 20, 20
		w.entities[1].X, w.entities[1].Y = 21, 20
		rule, _ := w.findSpell(2)
		rule.ID = arm
		if arm == 3 || arm == 19 {
			rule.AreaDuration = 8
			rule.Distribution = distributionWall
		}
		if arm == 4 || arm == 9 || arm == 21 {
			rule.Distribution = distributionStaged
		}
		if arm == 3 || arm == 19 || arm == 9 {
			friend := spEnt(4, 21, 19)
			friend.Owner = SelfSlot
			if arm == 9 {
				friend.X, friend.Y = 22, 20
			}
			w.entities = append(w.entities, friend)
		}
		w.spells = []SpellRule{rule}
		before := w.Hash()
		got := w.unsafeUnorderedArea(0, 1, rule)
		if w.Hash() != before {
			t.Fatalf("arm %d changed state", arm)
		}
		if arm != 19 && !got {
			t.Fatalf("arm %d missed protected caster", arm)
		}
		if arm == 19 && got {
			t.Fatal("non-hitting earth wall protected an actor")
		}
	}
}

func TestAreaSafetyCoversSupportWalk(t *testing.T) {
	w := safeAreaWorld(t, false)
	w.entities = w.entities[:1]
	w.entities[0].AutoSpell = 0
	w.spells[1].Area = true
	w.spells[1].Radius = 1
	w.SetAutoCast(SelfSlot, AutoCastMinimum|AutoCastOwn)
	w.supportCast(0)
	if len(w.bookCasts) != 0 || w.entities[0].Mana != 500 {
		t.Fatal("support walk admitted area over protected caster")
	}
}

func TestAreaSafetyControlsWeaponAttack(t *testing.T) {
	for _, kind := range []string{"unsafe", "ordered", "computer"} {
		t.Run(kind, func(t *testing.T) {
			w := safeAreaWorld(t, false)
			w.entities[0].WeaponSpell = 2
			w.entities[0].AutoSpell = 0
			w.entities[0].Reach = 6
			w.entities[0].HasAttackTarget = true
			w.entities[0].AttackTarget = 2
			w.entities[0].ActorState = actorStateGuard
			if kind == "ordered" {
				w.entities[0].ActorState = actorStateEngage
			}
			if kind == "computer" {
				w.entities[0].Owner = 2
				w.entities[1].Owner = SelfSlot
				w.relations = acEnemies(t)
			}
			if kind == "unsafe" {
				for range 80 {
					if events := StepObserved(w, nil); len(events) != 0 {
						t.Fatal("unsafe weapon area released", events)
					}
				}
				if w.entities[0].HP != 1000 || w.entities[1].HP != 5000 {
					t.Fatal("unsafe weapon area dealt damage")
				}
			} else {
				event := safeAreaFirstCast(t, w)
				if !event.Weapon || event.Target != 2 || w.entities[0].HP >= 1000 {
					t.Fatal("weapon control changed", event)
				}
			}
		})
	}
}
