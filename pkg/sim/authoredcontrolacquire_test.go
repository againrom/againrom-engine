package sim

import "testing"

type acquireControlCase struct {
	name           string
	owner, word    uint32
	present, human bool
}

func acquireControlCases() []acquireControlCase {
	return []acquireControlCase{
		{"human-other-slot", 7, 0, true, true},
		{"computer-other-slot", 7, 1, true, false},
		{"computer-zero-low-byte", 7, 0x100, true, false},
		{"computer-self-slot", SelfSlot, 1, true, false},
		{"legacy-self-slot", SelfSlot, 0, false, true},
		{"legacy-other-slot", 7, 0, false, false},
	}
}

func installAcquireControl(t *testing.T, w *World, tc acquireControlCase) {
	t.Helper()
	var rows []PlayerParticipant
	if tc.present {
		rows = []PlayerParticipant{{PlayerID: 91, Value: tc.word}, {PlayerID: 103, Value: 1}}
	}
	if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: tc.owner}, {ID: 103, Slot: 9}}, rows, tc.present); err != nil {
		t.Fatal(err)
	}
}

func TestAuthoredControlUnownedAcquisitionKeepsLegacyRule(t *testing.T) {
	mage := engFighter(1, 0, 5, 5)
	mage.MaxMana, mage.Reach = 10, 1
	w := engWorld(t, engRel(t), mage, engFighter(2, 9, 6, 5))
	if !w.orderAttack(0, 2) {
		t.Fatal("unowned actor acquired a human control predicate")
	}
	w.entities[0].clearAttack()
	w.entities[0].TargetX, w.entities[0].TargetY, w.entities[0].HasTarget = 12, 5, true
	for i := range w.groups {
		w.groups[i].order = orderStandGround
	}
	w.decide(aiGroup{owner: 0, group: w.entities[0].Group, members: []int{0}})
	if !w.entities[0].HasTarget {
		t.Fatal("unowned empty decision gained the human stand-down branch")
	}
}

func TestAuthoredControlRefusedAcquisitionAndColdRestore(t *testing.T) {
	for _, tc := range acquireControlCases() {
		t.Run(tc.name, func(t *testing.T) {
			mage := engFighter(1, tc.owner, 5, 5)
			mage.MaxMana, mage.Reach = 10, 1
			w := engWorld(t, engRel(t, [3]uint32{tc.owner, 9, relationHostile}),
				mage, engFighter(2, 9, 6, 5), engFighter(3, 9, 12, 5))
			installAcquireControl(t, w, tc)
			if !w.attachAttack(0, 3, AttackTargetUnit, true) {
				t.Fatal("fixture could not retain the distant victim")
			}
			cold := worldRoundTripForTest(t, w)
			for _, live := range []*World{w, cold} {
				live.answerRefusedPursuit(0, searchNear)
				e := live.entities[0]
				if tc.human {
					if !e.PursuitIdle || e.AcquirePursuit || e.AttackTarget != 3 {
						t.Fatalf("zero control acquired a target: idle %v acquire %v target %d", e.PursuitIdle, e.AcquirePursuit, e.AttackTarget)
					}
				} else if !e.AcquirePursuit || e.PursuitIdle || !e.HasAttackTarget || e.AttackTarget != 2 {
					t.Fatalf("nonzero control did not acquire the adjacent target: %+v", e)
				}
			}
			if cold.Hash() != w.Hash() {
				t.Fatal("cold restore changed refused acquisition")
			}
		})
	}
}

func TestAuthoredControlMageBookChoiceAndColdRestore(t *testing.T) {
	for _, tc := range acquireControlCases() {
		t.Run(tc.name, func(t *testing.T) {
			mage := spMage(1, 2, 2, 30, 100, 100, 1<<1)
			mage.Owner, mage.Book.State = tc.owner, BookPresent
			mage.Book.Slots[0] = BookSpell{Range: 5, ManaCost: 3}
			target := engFighter(2, 9, 4, 2)
			rule := SpellRule{ID: 1, School: 1, ManaCost: 3, MaxRange: 5, TargetsUnit: true, Damaging: true, DamageMin: 10, DamageMax: 10}
			w := hlWorld(t, 31, engRel(t, [3]uint32{tc.owner, 9, relationHostile}), []SpellRule{rule}, mage, target)
			installAcquireControl(t, w, tc)
			cold := worldRoundTripForTest(t, w)
			for _, live := range []*World{w, cold} {
				before := live.rng
				if got := live.aiCast(0, nil); got == tc.human {
					t.Fatalf("book choice admitted %v for human %v", got, tc.human)
				}
				k, cast := live.bookCastIndex(1)
				if tc.human {
					if cast || live.rng != before {
						t.Fatal("zero control drew or queued a computer book spell")
					}
				} else if !cast || live.bookCasts[k].Spell != 1 || live.bookCasts[k].Target != 2 {
					t.Fatalf("computer book choice lost spell or target: %+v", live.bookCasts)
				}
			}
			if cold.Hash() != w.Hash() {
				t.Fatal("cold restore changed book choice")
			}
		})
	}
}

func authoredCreatureWorld(t *testing.T, tc acquireControlCase, mage bool, reach uint8, spellRange uint8) *World {
	t.Helper()
	book, known := creatureBookOf(map[uint32]uint8{1: spellRange})
	caster := engFighter(1, tc.owner, 5, 5)
	caster.Book, caster.KnownSpells, caster.CreatureSpells = book, known, creatureCertainSlot(1)
	caster.Reach = reach
	if mage {
		caster.MaxMana, caster.Mana = 100, 100
	}
	victim := engFighter(2, 9, 9, 5)
	victim.HP, victim.MaxHP, victim.DamageBase = 5000, 5000, 0
	rule := hlArrow()
	rule.MaxRange = spellRange
	w, err := NewStockedSpelledWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{caster, victim}, nil,
		engRel(t, [3]uint32{tc.owner, 9, relationHostile}, [3]uint32{9, tc.owner, relationLocked}), nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	installAcquireControl(t, w, tc)
	wakeGroups(w)
	return w
}

func TestAuthoredControlCreatureDrawUsesMageAndReach(t *testing.T) {
	for _, tc := range acquireControlCases() {
		for _, actor := range []struct {
			name  string
			mage  bool
			reach uint8
		}{{"short-mage", true, 1}, {"long-mage", true, 2}, {"non-mage", false, 1}} {
			t.Run(tc.name+"/"+actor.name, func(t *testing.T) {
				w := authoredCreatureWorld(t, tc, actor.mage, actor.reach, 7)
				cold := worldRoundTripForTest(t, w)
				want := !tc.human || !actor.mage || actor.reach >= 2
				for _, live := range []*World{w, cold} {
					before := live.rng
					if got := live.creatureEngageCast(0, 2); got != want {
						t.Fatalf("creature slot draw admitted %v, want %v", got, want)
					}
					k, cast := live.bookCastIndex(1)
					if cast != want || cast && (live.bookCasts[k].Spell != 1 || live.bookCasts[k].Target != 2) {
						t.Fatalf("creature cast %+v, want spell 1 at actor 2 present %v", live.bookCasts, want)
					}
					if !want && live.rng != before {
						t.Fatal("suppressed short mage consumed the slot draw")
					}
				}
				if cold.Hash() != w.Hash() {
					t.Fatal("cold restore changed creature draw")
				}
			})
		}
	}
}

func TestAuthoredControlCreatureApproachAndColdRestore(t *testing.T) {
	for _, tc := range []acquireControlCase{
		{"human-other-slot", 7, 0, true, true},
		{"human-self-slot", SelfSlot, 0, true, true},
		{"computer-other-slot", 7, 1, true, false},
		{"computer-zero-low-byte", 7, 0x100, true, false},
		{"computer-self-slot", SelfSlot, 1, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := authoredCreatureWorld(t, tc, false, 1, 3)
			if !w.creatureEngageCast(0, 2) {
				t.Fatal("eligible creature did not arm its out-of-range cast")
			}
			if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Phase != bookApproach || !w.bookCasts[k].Retained {
				t.Fatalf("cast did not retain its approach: %+v", w.bookCasts)
			}
			w.tick = scriptPassPhase + 1
			cold := worldRoundTripForTest(t, w)
			charged := false
			for tick := 0; tick < 160; tick++ {
				Step(w, nil)
				Step(cold, nil)
				if cold.Hash() != w.Hash() {
					t.Fatalf("cold approach diverged at tick %d", tick)
				}
				if k, ok := w.bookCastIndex(1); ok && w.bookCasts[k].Phase == bookCharging {
					if !w.bookCasts[k].Retained || w.entities[0].X <= 5 {
						t.Fatal("approached cast lost retention or never walked")
					}
					charged = true
					break
				}
			}
			if !charged {
				t.Fatal("eligible creature never completed its cast approach")
			}
		})
	}
}

func TestAuthoredControlEmptyNativeDecisionAndColdRestore(t *testing.T) {
	for _, tc := range acquireControlCases() {
		t.Run(tc.name, func(t *testing.T) {
			w := engWorld(t, engRel(t), engFighter(1, tc.owner, 5, 5))
			installAcquireControl(t, w, tc)
			for i := range w.groups {
				w.groups[i].order = orderStandGround
			}
			w.entities[0].TargetX, w.entities[0].TargetY, w.entities[0].HasTarget = 12, 5, true
			cold := worldRoundTripForTest(t, w)
			for _, live := range []*World{w, cold} {
				live.decide(aiGroup{owner: tc.owner, group: live.entities[0].Group, members: []int{0}})
				if got := live.entities[0].HasTarget; got == tc.human {
					t.Fatalf("empty decision kept destination %v for human %v", got, tc.human)
				}
			}
			if cold.Hash() != w.Hash() {
				t.Fatal("cold restore changed empty decision")
			}
		})
	}
}
