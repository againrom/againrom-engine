package sim

import "testing"

func TestFirstGameRefusedReacquisitionStoresOrderStateAndTurnsComputerUnit(t *testing.T) {
	t.Parallel()
	const width, height = 24, 16
	b := Bounds{Width: width, Height: height}
	human := engFighter(1, SelfSlot, 4, 4)
	human.Facing = 64
	human.DesiredFacing = 64
	comp := engFighter(2, 2, 10, 4)
	comp.Facing = 64
	comp.DesiredFacing = 64
	foe := engFighter(3, 3, 20, 4)
	rel := engRel(t,
		[3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 1},
		[3]uint32{2, 3, 1}, [3]uint32{3, 2, 1},
	)
	w, err := NewRelatedWorld(7, b, ModeCanonical, Terrain{}, []Entity{human, comp, foe}, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreCurrentPlayers(
		[]SavedGroupPlayer{{ID: 1, Slot: SelfSlot}, {ID: 2, Slot: 2}, {ID: 3, Slot: 3}},
		[]PlayerParticipant{{PlayerID: 1, Value: 0}, {PlayerID: 2, Value: 1}, {PlayerID: 3, Value: 1}},
		true,
	); err != nil {
		t.Fatal(err)
	}
	if w.savedOrder(1) == nil || w.savedOrder(2) == nil {
		t.Fatal("fixture: missing saved orders")
	}
	w.savedOrder(1).Raw[8] = 5
	w.savedOrder(2).Raw[8] = 5
	w.savedOrder(2).Raw[0x54] = 1

	w.entities[0].HasAttackTarget, w.entities[0].AttackTarget, w.entities[0].AttackTargetKind = true, 3, AttackTargetUnit
	w.entities[1].HasAttackTarget, w.entities[1].AttackTarget, w.entities[1].AttackTargetKind = true, 3, AttackTargetUnit

	w.answerRefusedRoute(0, refusedPursuit)
	if o := w.savedOrder(1); o.Raw[8] != 0 {
		t.Fatalf("human refused reacquisition ord+8 = %#x, want 0", o.Raw[8])
	}
	if e := w.entities[0]; !e.PursuitIdle || !e.HasAttackTarget || e.AttackTarget != 3 || e.Turning() || e.Facing != 64 {
		t.Fatalf("human refused state: idle=%v target=%v/%d turning=%v facing=%d",
			e.PursuitIdle, e.HasAttackTarget, e.AttackTarget, e.Turning(), e.Facing)
	}

	probe := w.rng
	wantHeading := uint8(64) + 0x21 + uint8(probe.aiRange(190))
	w.answerRefusedRoute(1, refusedPursuit)
	if o := w.savedOrder(2); o.Raw[8] != 0xb || o.Raw[0x54] != 0 {
		t.Fatalf("computer alarmed refused reacquisition ord+8=%#x ord+0x54=%d, want 0xb and 0", o.Raw[8], o.Raw[0x54])
	}
	if e := w.entities[1]; !e.PursuitIdle || !e.HasAttackTarget || e.AttackTarget != 3 || (e.DesiredFacing != wantHeading && e.Facing != wantHeading) {
		t.Fatalf("computer alarmed refused state: idle=%v target=%v/%d facing=%d desired=%d want=%d",
			e.PursuitIdle, e.HasAttackTarget, e.AttackTarget, e.Facing, e.DesiredFacing, wantHeading)
	}

	turns, stands := 0, 0
	for n := 0; n < 20000 && (turns < 3 || stands < 3); n++ {
		e := &w.entities[1]
		e.clearTurn()
		e.PursuitIdle = false
		o := w.savedOrder(2)
		o.Raw[8], o.Raw[0x54] = 5, 0
		p := w.rng
		draw := p.raw()
		want := e.Facing + 0x21 + uint8(p.aiRange(190))
		stood := w.rng
		stood.raw()
		w.answerRefusedRoute(1, refusedPursuit)
		e = &w.entities[1]
		if o.Raw[8] != 0xb {
			t.Fatalf("step %d: ord+8 = %#x, want 0xb", n, o.Raw[8])
		}
		if draw >= 0xcd {
			if w.rng != stood || e.Turning() {
				t.Fatalf("draw %d: turned or drew again", draw)
			}
			stands++
			continue
		}
		if w.rng != p || (e.DesiredFacing != want && e.Facing != want) {
			t.Fatalf("draw %d: facing %d/%d, want %d", draw, e.Facing, e.DesiredFacing, want)
		}
		e.Facing = want
		turns++
	}
	if turns < 3 || stands < 3 {
		t.Fatalf("turns=%d stands=%d, want both >= 3", turns, stands)
	}
}

func TestComputerHealListExcludesDomainThreeActorsInBothGames(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		name := "first_game"
		if second {
			name = "second_game"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			airAlly := ownUnit(2, 3, 2, 40)
			airAlly.Domain = DomainAir
			groundAlly := ownUnit(3, 5, 2, 40)
			groundAlly.Domain = DomainGround
			w, heal, _, _ := drawWorld(t, second, 11, airAlly, groundAlly)
			casts := mageCasts(w, 100)
			if len(casts) != 1 || casts[0].Spell != heal || casts[0].Target != 3 {
				t.Fatalf("case A: casts %+v, want one Heal at ground ally 3 (skipping domain-3 ally 2)", casts)
			}

			onlyAir := ownUnit(2, 3, 2, 40)
			onlyAir.Domain = DomainAir
			wB, _, _, _ := drawWorld(t, second, 11, onlyAir)
			if casts := mageCasts(wB, 100); len(casts) != 0 {
				t.Fatalf("case B: casts %+v, want none for domain-3 ally alone", casts)
			}

			wC, healC, _, _ := drawWorld(t, second, 11)
			wC.entities[0].Domain = DomainAir
			wC.entities[0].HP = 40
			castsC := mageCasts(wC, 100)
			if len(castsC) != 1 || castsC[0].Spell != healC || castsC[0].Target != 1 {
				t.Fatalf("case C: casts %+v, want self-Heal on domain-3 caster", castsC)
			}

			buildBuffWorld := func(domain Domain) (*World, uint16) {
				healRow, protRow := hlHeal(), acsBuff(5)
				healRow.ManaCost, protRow.ManaCost = 3, 5
				if second {
					healRow.ID, protRow.ID = 24, 4
					healRow, protRow = secondRow(healRow), secondRow(protRow)
				}
				rows := []SpellRule{healRow, protRow}
				mage := spMage(1, 2, 2, 30, 60, 60, 0)
				mage.Owner, mage.Book.State, mage.Domain = 7, BookPresent, domain
				for _, r := range rows {
					mage.KnownSpells |= 1 << r.ID
					mage.Book.Slots[r.ID-1] = BookSpell{Range: uint8(r.MaxRange), Defensive: 1, ManaCost: uint16(r.ManaCost)}
				}
				var script *Script
				if second {
					var err error
					if script, err = newScript(ScriptROM2, nil, nil, nil); err != nil {
						t.Fatal(err)
					}
				}
				companion := spEnt(99, 10, 10)
				companion.Owner = SelfSlot
				world, err := NewStockedSpelledWorld(31, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
					[]Entity{mage, companion}, script, engRel(t,
						[3]uint32{7, SelfSlot, relationLocked}, [3]uint32{SelfSlot, 7, relationLocked}),
					nil, nil, rows)
				if err != nil {
					t.Fatal(err)
				}
				if err := world.RestoreCurrentPlayers(
					[]SavedGroupPlayer{{ID: 91, Slot: 7}, {ID: 92, Slot: SelfSlot}},
					[]PlayerParticipant{{PlayerID: 91, Value: 1}, {PlayerID: 92, Value: 0}}, true); err != nil {
					t.Fatal(err)
				}
				holdGroups(world)
				return world, protRow.ID
			}
			wAir, _ := buildBuffWorld(DomainAir)
			if casts := mageCasts(wAir, 120); len(casts) != 0 {
				t.Fatalf("case D (air caster): casts %+v, want none", casts)
			}
			wGround, prot := buildBuffWorld(DomainGround)
			castsG := mageCasts(wGround, 120)
			if len(castsG) == 0 || castsG[0].Spell != prot || castsG[0].Target != 1 {
				t.Fatalf("case D (ground caster): casts %+v, want self-buff %d", castsG, prot)
			}
		})
	}
}

func TestDarknessPulseRunsBlowHookOnCasterInBothGames(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		name := "first_game"
		if second {
			name = "second_game"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			build := func(diag byte) (*World, SpellRule) {
				rule := SpellRule{ID: 17, Area: true, Distribution: distributionDiamond, Radius: 1, AreaDuration: 2,
					TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 200}
				var script *Script
				if second {
					rule.ID = 14
					rule = secondRow(rule)
					script = secondGameScript(t, nil, nil, nil)
				}
				caster := effectMage(1, 20, 20, 1<<rule.ID)
				caster.Owner = 2
				caster.TypeID = 10
				rel := engRel(t, [3]uint32{2, 2, uint32(diag)})
				w, err := NewStockedSpelledWorld(5, Bounds{Width: 40, Height: 40}, ModeCanonical, Terrain{},
					[]Entity{caster}, script, rel, nil, nil, []SpellRule{rule})
				if err != nil {
					t.Fatal(err)
				}
				if w.savedOrder(1) == nil {
					t.Fatal("fixture: missing caster saved order")
				}
				return w, rule
			}

			w, rule := build(0x12)
			key := cellKey(20, 20)
			w.applyAreaCells(cellEffect{Key: key, Spell: rule.ID, Mode: areaModeCloud, Caster: 1, HasCaster: true}, rule, []uint16{key})
			if n := w.attackNoticeAt(0); n.Cell != key || n.Scans != 0 {
				t.Fatalf("live: attack notice = %+v, want cell %#x scans 0", n, key)
			}
			if o := w.savedOrder(1); o.Raw[0x54] != 1 {
				t.Fatalf("live: ord+0x54 = %d, want 1", o.Raw[0x54])
			}
			if w.relations.Hostile(2, 2) {
				t.Fatal("live: locked diagonal flipped hostile")
			}
			if second && w.entities[0].AreaHitCount != 0 {
				t.Fatalf("live: locked diagonal incremented AreaHitCount to %d, want 0", w.entities[0].AreaHitCount)
			}
			if !second && (!w.entities[0].HasKillCredit || w.entities[0].KillCreditSource != 1) {
				t.Fatalf("live ROM1: kill credit = %v/%d, want true/1", w.entities[0].HasKillCredit, w.entities[0].KillCreditSource)
			}

			wOpen, ruleOpen := build(0)
			wOpen.applyAreaCells(cellEffect{Key: key, Spell: ruleOpen.ID, Mode: areaModeCloud, Caster: 1, HasCaster: true}, ruleOpen, []uint16{key})
			if !wOpen.relations.Hostile(2, 2) {
				t.Fatal("live: unlocked diagonal did not flip hostile")
			}

			wSaved, ruleSaved := build(0x12)
			payload := SavedEffect{Class: "Effect", E0C: uint8(ruleSaved.ID), E3C: 19, E3D: 2, E40: uint32(uint16(200))<<16 | uint32(uint16(2))}
			wSaved.applySavedAreaPayload(SavedAreaDriver{Key: key, Spell: ruleSaved.ID, Mode: areaModeCloud},
				SavedSpellEffect{AE44: &payload}, []uint16{key}, 1)
			if n := wSaved.attackNoticeAt(0); n.Cell != key || n.Scans != 0 {
				t.Fatalf("saved: attack notice = %+v, want cell %#x scans 0", n, key)
			}
			if o := wSaved.savedOrder(1); o.Raw[0x54] != 1 {
				t.Fatalf("saved: ord+0x54 = %d, want 1", o.Raw[0x54])
			}
		})
	}
}

func TestSecondGameAreaHitStateSixteenRequiresTerminalHealth(t *testing.T) {
	t.Parallel()
	downed, _, fire := areaHostilityWorld(t, true, 1, 1, Relations{})
	pulseAreaAtVictim(downed, fire, areaModeBlast)
	if downed.entities[1].HP != 0 || downed.entities[1].Alive() {
		t.Fatalf("fixture: victim hp=%d alive=%v, want 0 and false", downed.entities[1].HP, downed.entities[1].Alive())
	}
	if downed.relations.Hostile(2, SelfSlot) || downed.entities[1].AreaHitCount != 1 {
		t.Fatalf("downed victim (HP=0, MaxHP=1): hostile=%v count=%d, want false and 1",
			downed.relations.Hostile(2, SelfSlot), downed.entities[1].AreaHitCount)
	}

	terminal, _, _ := areaHostilityWorld(t, true, -10, -20, Relations{})
	terminal.flipOnBlow(0, 1, blowArea)
	if !terminal.relations.Hostile(2, SelfSlot) || terminal.entities[1].AreaHitCount != 1 {
		t.Fatalf("terminal victim (HP=-10): hostile=%v count=%d, want true and 1",
			terminal.relations.Hostile(2, SelfSlot), terminal.entities[1].AreaHitCount)
	}
}

func TestCoverEngageRanksDownedHostileWithoutHealthSplitInBothGames(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		name := "first_game"
		if second {
			name = "second_game"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tornDown := engFighter(3, SelfSlot, 20, 20)
			tornDown.HP = -10
			downed := engFighter(4, SelfSlot, 21, 20)
			downed.HP = 0
			living := engFighter(5, SelfSlot, 24, 20)
			living.HP = 100
			if got := coverTarget(t, second, 1, 0, tornDown, downed, living); got != 4 {
				t.Fatalf("cover engaged %d, want closer downed hostile 4 (ignoring torn-down 3)", got)
			}
		})
	}
}

func TestTargetingAdoptionReplayHashAndBinaryRoundTrip(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		name := "first_game"
		if second {
			name = "second_game"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			build := func() *World {
				healRow := hlHeal()
				darkRow := SpellRule{ID: 17, Area: true, Distribution: distributionDiamond, Radius: 1, AreaDuration: 2,
					TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 200}
				var script *Script
				if second {
					healRow.ID, darkRow.ID = 24, 14
					healRow, darkRow = secondRow(healRow), secondRow(darkRow)
					script = secondGameScript(t, nil, nil, nil)
				}
				mage := spMage(1, 6, 6, 30, 60, 20, 1<<healRow.ID|1<<darkRow.ID)
				mage.Owner, mage.Book.State = 2, BookPresent
				mage.Book.Slots[healRow.ID-1] = BookSpell{Range: 6, Defensive: 1, ManaCost: 3}
				airAlly := ownUnit(2, 7, 6, 40)
				airAlly.Owner, airAlly.Domain = 2, DomainAir
				groundAlly := ownUnit(3, 9, 6, 40)
				groundAlly.Owner = 2
				defender := engFighter(4, 2, 12, 12)
				subject := engFighter(5, 2, 12, 11)
				downed := engFighter(6, SelfSlot, 13, 11)
				downed.HP = 0
				living := engFighter(7, SelfSlot, 16, 11)
				rel := engRel(t,
					[3]uint32{2, 2, 0x12}, [3]uint32{SelfSlot, SelfSlot, 0x12},
					[3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 1},
				)
				w, err := NewStockedSpelledWorld(19, Bounds{Width: 24, Height: 24}, ModeCanonical, Terrain{},
					[]Entity{mage, airAlly, groundAlly, defender, subject, downed, living}, script, rel,
					nil, nil, []SpellRule{healRow, darkRow})
				if err != nil {
					t.Fatal(err)
				}
				if err := w.RestoreCurrentPlayers(
					[]SavedGroupPlayer{{ID: 1, Slot: SelfSlot}, {ID: 2, Slot: 2}},
					[]PlayerParticipant{{PlayerID: 1, Value: 0}, {PlayerID: 2, Value: 1}}, true); err != nil {
					t.Fatal(err)
				}
				w.landArea(darkRow, 0, 1, true, 6, 6, 6, 6, nil)
				w.coverEngage(3, 4)
				return w
			}

			a, b := build(), build()
			var mid *World
			for tick := 1; tick <= 48; tick++ {
				Step(a, nil)
				Step(b, nil)
				if a.Hash() != b.Hash() {
					t.Fatalf("tick %d: replay hashes diverge %x != %x", tick, a.Hash(), b.Hash())
				}
				if tick == 16 {
					mid = worldRoundTripForTest(t, a)
					if a.Hash() != mid.Hash() {
						t.Fatalf("tick %d: immediate round-trip hash diverged %x != %x", tick, a.Hash(), mid.Hash())
					}
				} else if mid != nil {
					Step(mid, nil)
					if a.Hash() != mid.Hash() {
						t.Fatalf("tick %d: round-trip hash diverged %x != %x", tick, a.Hash(), mid.Hash())
					}
				}
			}
		})
	}
}
