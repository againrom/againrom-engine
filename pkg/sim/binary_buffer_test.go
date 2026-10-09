package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"slices"
	"testing"
)

func init() {
	worldMethods = append(worldMethods, "MarshalBinaryInto")
	slices.Sort(worldMethods)
	worldArgReaders["MarshalBinaryInto"] = []reflect.Value{reflect.ValueOf([]byte(nil))}
}

func binaryBufferWorld(t testing.TB, count int) *World {
	t.Helper()
	entities := make([]Entity, count)
	for i := range entities {
		entities[i] = Entity{
			ID: EntityID(i + 1), X: int32(i%56 + 1), Y: int32(i/56 + 1),
			HP: int32(50 + i%50), MaxHP: 100, Speed: 256,
			Class: int32(i % 7), Owner: uint32(i % 4), Facing: uint8(i % 256),
			Resistance: [5]uint8{1, 3, 5, 7, 9},
		}
	}
	w, err := NewWorld(0x123456789abcdef, Bounds{Width: 64, Height: 64}, ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	for i := range w.cost {
		w.cost[i] = byte(i%15 + 1)
		w.height[i] = byte(i % 128)
	}
	for i := range w.entities {
		e := &w.entities[i]
		if i%2 == 0 {
			e.HasTarget, e.TargetX, e.TargetY = true, e.X+1, e.Y
			w.routes[i] = []cell{{x: e.TargetX, y: e.TargetY}}
			e.ActionClock = ActionClock{Known: true, End: uint32(900 + i)}
		}
		if i%3 == 0 {
			w.carried[i] = []ItemStack{PlainStack(0x801, uint32(i%5+1)), PlainStack(0x802, 2)}
		}
	}
	w.sacks = []Sack{{X: 10, Y: 10, Gold: 1234,
		Items: []uint16{0x803, 0x804}, ItemInstances: []ItemInstance{PlainItem(0x803), PlainItem(0x804)}}}
	w.tick, w.hasSessionClock, w.fullTick = 125, true, 0x12345
	w.ImportAutoHealing(SelfSlot, 50)
	return w
}

func requireBinaryBufferEqual(t *testing.T, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		at := 0
		for at < len(got) && at < len(want) && got[at] == want[at] {
			at++
		}
		t.Fatalf("canonical bytes differ at %d; lengths %d/%d", at, len(got), len(want))
	}
}

func TestWorldMarshalBinaryIntoCanonicalBytes(t *testing.T) {
	for _, test := range []struct {
		name string
		make func(*testing.T) *World
	}{
		{"empty", func(t *testing.T) *World { return mustWorld(t, 7, Bounds{Width: 2, Height: 2}, nil) }},
		{"offsets", offsetWorld},
		{"populated", func(t *testing.T) *World { return binaryBufferWorld(t, 143) }},
		{"safe mode", func(t *testing.T) *World { w := binaryBufferWorld(t, 143); w.SetSafeMode(true); return w }},
		{"pending release", func(t *testing.T) *World { w := lcPlayerWorld(t); w.releaseAttack(0); return w }},
		{"pending completion", func(t *testing.T) *World { w := lcPlayerWorld(t); w.CompleteSackPickup(1); return w }},
		{"admitted row with retained carrier", func(t *testing.T) *World {
			w := manualCastWorld(t)
			w.orderAttack(0, 2)
			w.entities[0].ActorState = actorStateRetreat
			w.entities[0].Retreat = RetreatContinuation{Known: true}
			w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 3
			if !w.beginManualCast(0, CastAt(1, 26, CellPoint{X: 6, Y: 3})) {
				t.Fatal("row fixture refused")
			}
			return w
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := test.make(t)
			want := legacyBinaryBufferEncoding(w)
			fresh, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			requireBinaryBufferEqual(t, fresh, want)
			for _, size := range []int{0, 1, len(want) - 1, len(want), len(want) + 37} {
				storage := bytes.Repeat([]byte{0xa5}, size)
				got, err := w.MarshalBinaryInto(storage[:0])
				if err != nil {
					t.Fatal(err)
				}
				requireBinaryBufferEqual(t, got, want)
				if size >= len(want) && &got[0] != &storage[0] {
					t.Fatal("sufficient caller capacity was not reused")
				}
				if size > len(want) && !bytes.Equal(storage[len(want):], bytes.Repeat([]byte{0xa5}, size-len(want))) {
					t.Fatal("write beyond the canonical byte length")
				}
			}
		})
	}
}

func TestWorldMarshalBinaryIntoClearsOldFlagsAndPadding(t *testing.T) {
	w := binaryBufferWorld(t, 4)
	w.entities[0].HasTarget = false
	w.entities[0].TargetX, w.entities[0].TargetY = 0, 0
	w.routes[0] = nil
	w.entities[0].ActionClock = ActionClock{}
	w.autoHealing = [relationSlots]autoHealingPolicy{}
	w.hasSessionClock, w.fullTick = false, 0
	want := legacyBinaryBufferEncoding(w)
	storage := bytes.Repeat([]byte{0xff}, len(want)+64)
	got, err := w.MarshalBinaryInto(storage)
	if err != nil {
		t.Fatal(err)
	}
	requireBinaryBufferEqual(t, got, want)
	entityStart := 34 + 3*64*64
	if got[entityStart+24] != 0 {
		t.Fatal("absent target retained a poisoned presence byte")
	}
	if &got[0] != &storage[0] {
		t.Fatal("poisoned buffer was replaced instead of cleared")
	}
}

func TestWorldMarshalBinaryIntoCallerOwnsStorage(t *testing.T) {
	w := binaryBufferWorld(t, 8)
	want := legacyBinaryBufferEncoding(w)
	storage := make([]byte, 0, len(want)+64)
	got, err := w.MarshalBinaryInto(storage)
	if err != nil {
		t.Fatal(err)
	}
	first, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		got[i] ^= 0xff
	}
	requireBinaryBufferEqual(t, legacyBinaryBufferEncoding(w), want)
	requireBinaryBufferEqual(t, first, want)
	requireBinaryBufferEqual(t, second, want)
	first[0] ^= 0xff
	requireBinaryBufferEqual(t, second, want)
	copy(got, want)
	w.entities[0].HP--
	w.grid[0] = 1
	w.cost[0]++
	w.carried[0][0].Count++
	w.tick++
	if bytes.Equal(legacyBinaryBufferEncoding(w), want) {
		t.Fatal("live mutation did not change the fixture")
	}
	requireBinaryBufferEqual(t, got, want)
	requireBinaryBufferEqual(t, second, want)
	if _, err := w.MarshalBinaryInto(nil); err != nil {
		t.Fatal(err)
	}
	requireBinaryBufferEqual(t, got, want)
}

func TestWorldMarshalBinaryIntoReusesAfterShrinkAndGrow(t *testing.T) {
	w := binaryBufferWorld(t, 143)
	large := legacyBinaryBufferEncoding(w)
	storage := bytes.Repeat([]byte{0xdd}, len(large)+64)
	var buf []byte = storage[:0]
	for _, count := range []int{143, 2, 143, 400, 0, 400} {
		*w = *binaryBufferWorld(t, count)
		want := legacyBinaryBufferEncoding(w)
		previous := buf[:cap(buf)]
		for i := range previous {
			previous[i] = 0xdd
		}
		got, err := w.MarshalBinaryInto(buf)
		if err != nil {
			t.Fatal(err)
		}
		requireBinaryBufferEqual(t, got, want)
		if cap(buf) >= len(want) && &got[0] != &previous[0] {
			t.Fatal("reallocation despite sufficient capacity")
		}
		buf = got
	}
}

func TestWorldMarshalBinaryIntoValidationLeavesBufferUntouched(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*World)
		want   string
	}{
		{"identity-floor", func(w *World) { w.entityIDFloor = entityIDLimit + 1 }, "sim: entity identity floor exceeds namespace"},
		{"session-clock", func(w *World) { w.hasSessionClock = false }, "sim: absent session clock carries a full counter"},
		{"action-clock", func(w *World) { w.entities[0].ActionClock = ActionClock{End: 1} }, "sim: absent action clock carries a deadline"},
		{"actor-load", func(w *World) { w.entities[0].ActorLoad.OwnWeight = 1 }, "sim: absent actor load has residue"},
		{"stride", func(w *World) { w.entities[0].Stride.Rate = 1 }, "sim: entity 1: absent native stride carries state"},
		{"first-fault", func(w *World) { w.entityIDFloor = entityIDLimit + 1; w.hasSessionClock = false }, "sim: entity identity floor exceeds namespace"},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := binaryBufferWorld(t, 4)
			if _, err := w.MarshalBinary(); err != nil {
				t.Fatal("invalid control world:", err)
			}
			test.mutate(w)
			old, oldErr := w.MarshalBinary()
			storage := bytes.Repeat([]byte{0x5a}, 32768)
			got, err := w.MarshalBinaryInto(storage[:3])
			if old != nil || got != nil || oldErr == nil || err == nil || oldErr.Error() != test.want || err.Error() != test.want {
				t.Fatalf("errors fresh=%v reuse=%v; want %q; lengths %d/%d", oldErr, err, test.want, len(old), len(got))
			}
			if !bytes.Equal(storage, bytes.Repeat([]byte{0x5a}, len(storage))) {
				t.Fatal("invalid World changed caller storage")
			}
		})
	}
}

func BenchmarkWorldMarshalBinaryBuffer(b *testing.B) {
	w := binaryBufferWorld(b, 143)
	initial, err := w.MarshalBinary()
	if err != nil {
		b.Fatal(err)
	}
	for _, reuse := range []bool{false, true} {
		name := "fresh"
		if reuse {
			name = "reuse"
		}
		b.Run(name, func(b *testing.B) {
			var buf []byte
			if reuse {
				buf = make([]byte, 0, len(initial))
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(initial)))
			b.ResetTimer()
			for range b.N {
				var out []byte
				var err error
				if reuse {
					out, err = w.MarshalBinaryInto(buf)
					buf = out
				} else {
					out, err = w.MarshalBinary()
				}
				if err != nil || len(out) != len(initial) {
					b.Fatalf("marshal length %d: %v", len(out), err)
				}
			}
		})
	}
}

// This frozen base encoder keeps the fresh-allocation byte writer independent.
func legacyBinaryBufferEncoding(w *World) []byte {
	n := headerLen + len(w.grid) + len(w.cost) + len(w.height) + relationLen +
		entityLen*len(w.entities)
	for _, r := range w.routes {
		n += routeCountLen + cellLen*len(r)
	}
	n += groupCountLen + groupRecordLen*len(w.groups)
	n += sackCountLen
	for _, s := range w.sacks {
		n += sackHeaderLen + 2*len(s.Items)
	}
	for _, stacks := range w.carried {
		n += carryCountLen + 2*int(containerUnits(stacks))
	}
	n += equipRecordLen * len(w.equipment)
	n += treasureRecordLen * len(w.entities)
	n += purseLen
	n += spellCountLen + spellRecordLen*len(w.spells)
	n += itemWeightCountLen + itemWeightRecordLen*len(w.itemWeights)
	n += w.castingSectionLen()
	n += w.scrollSectionLen()
	n += w.scriptStateSectionLen()
	n += w.structureSectionLen()
	n += w.itemStateSectionLen()
	n += w.scriptSectionLen()
	n += w.originalDeadSectionLen()
	n += w.actorLoadSectionLen()
	n += w.instanceWeightSectionLen()
	b := make([]byte, n)

	b[0] = formatVersion
	binary.LittleEndian.PutUint64(b[1:9], w.tick)
	binary.LittleEndian.PutUint64(b[9:17], w.rng.state)
	binary.LittleEndian.PutUint32(b[17:21], uint32(w.bounds.Width))
	binary.LittleEndian.PutUint32(b[21:25], uint32(w.bounds.Height))
	binary.LittleEndian.PutUint32(b[25:29], uint32(len(w.entities)))
	b[29] = byte(w.mode)
	if w.safeMode {
		b[29] |= 0x80
	}
	binary.LittleEndian.PutUint32(b[30:34], uint32(len(w.grid)))
	copy(b[headerLen:], w.grid)
	copy(b[headerLen+len(w.grid):], w.cost)
	copy(b[headerLen+len(w.grid)+len(w.cost):], w.height)

	records := headerLen + len(w.grid) + len(w.cost) + len(w.height)
	for i, e := range w.entities {
		o := records + entityLen*i
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(e.ID))
		binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(e.X))
		binary.LittleEndian.PutUint32(b[o+8:o+12], uint32(e.Y))
		binary.LittleEndian.PutUint32(b[o+12:o+16], uint32(e.TargetX))
		binary.LittleEndian.PutUint32(b[o+16:o+20], uint32(e.TargetY))
		binary.LittleEndian.PutUint32(b[o+20:o+24], uint32(e.Class))
		if e.HasTarget {
			b[o+24] = 1
		}
		b[o+25] = e.Stall
		binary.LittleEndian.PutUint32(b[o+26:o+30], uint32(e.HP))
		binary.LittleEndian.PutUint32(b[o+30:o+34], uint32(e.MaxHP))
		b[o+34] = byte(e.Domain)
		binary.LittleEndian.PutUint32(b[o+35:o+39], uint32(e.Speed))
		binary.LittleEndian.PutUint16(b[o+39:o+41], e.Transit)
		binary.LittleEndian.PutUint16(b[o+41:o+43], e.TransitTotal)
		b[o+43] = e.GroupSpeed
		binary.LittleEndian.PutUint32(b[o+44:o+48], uint32(e.AttackTarget))
		if e.HasAttackTarget {
			b[o+48] = byte(e.AttackTargetKind) + 1
		}
		b[o+49] = byte(e.AttackPhase)
		binary.LittleEndian.PutUint32(b[o+50:o+54], uint32(e.AttackCountdown))
		binary.LittleEndian.PutUint32(b[o+54:o+58], uint32(e.AttackCharge))
		binary.LittleEndian.PutUint32(b[o+58:o+62], uint32(e.AttackRelax))
		binary.LittleEndian.PutUint32(b[o+62:o+66], uint32(e.ToHit))
		binary.LittleEndian.PutUint32(b[o+66:o+70], uint32(e.Defence))
		binary.LittleEndian.PutUint32(b[o+70:o+74], uint32(e.Absorption))
		binary.LittleEndian.PutUint32(b[o+74:o+78], uint32(e.DamageBase))
		binary.LittleEndian.PutUint32(b[o+78:o+82], uint32(e.DamageSpread))
		if e.AlwaysHits {
			b[o+82] = 1
		}
		binary.LittleEndian.PutUint32(b[o+83:o+87], e.Group)
		binary.LittleEndian.PutUint32(b[o+87:o+91], e.Owner)
		b[o+91] = e.Facing
		b[o+92] = byte(e.Decay)
		binary.LittleEndian.PutUint16(b[o+93:o+95], e.Dwell)
		binary.LittleEndian.PutUint32(b[o+95:o+99], uint32(e.DyingTime))
		b[o+99] = e.ScanRange
		b[o+100] = e.ActorState
		if e.PendingOrder.Kind == PendingPickupComplete {
			b[o+100] = actorStateGuard
		}
		if e.PendingOrder.Kind != PendingNone && e.Retreat.Known {
			b[o+100] = actorStateRetreat
		}
		binary.LittleEndian.PutUint32(b[o+101:o+105], uint32(e.PatrolHeadX))
		binary.LittleEndian.PutUint32(b[o+105:o+109], uint32(e.PatrolHeadY))
		binary.LittleEndian.PutUint32(b[o+109:o+113], uint32(e.PatrolTailX))
		binary.LittleEndian.PutUint32(b[o+113:o+117], uint32(e.PatrolTailY))
		b[o+117] = e.PatrolLeg
		b[o+118] = e.Reach
		binary.LittleEndian.PutUint32(b[o+119:o+123], uint32(e.PostX))
		binary.LittleEndian.PutUint32(b[o+123:o+127], uint32(e.PostY))
		binary.LittleEndian.PutUint32(b[o+127:o+131], uint32(e.Mana))
		binary.LittleEndian.PutUint32(b[o+131:o+135], uint32(e.MaxMana))
		binary.LittleEndian.PutUint32(b[o+135:o+139], uint32(e.HealthRegenPeriod))
		binary.LittleEndian.PutUint32(b[o+139:o+143], uint32(e.ManaRegenPeriod))
		b[o+143] = e.HealthHundredths
		b[o+144] = e.ManaHundredths
		binary.LittleEndian.PutUint32(b[o+145:o+149], e.CommandGroup)
		for k, xp := range e.SkillXP {
			binary.LittleEndian.PutUint32(b[o+149+4*k:o+153+4*k], uint32(xp))
		}
		binary.LittleEndian.PutUint32(b[o+173:o+177], uint32(e.Mind))
		binary.LittleEndian.PutUint32(b[o+177:o+181], uint32(e.XPValue))
		b[o+181] = e.XPSlot
		if e.GainsXP {
			b[o+182] = 1
		}
		binary.LittleEndian.PutUint32(b[o+183:o+187], e.KnownSpells)
		for k, lvl := range e.Skill {
			binary.LittleEndian.PutUint32(b[o+187+4*k:o+191+4*k], uint32(lvl))
		}
		binary.LittleEndian.PutUint16(b[o+211:o+213], e.WeaponSpell)
		binary.LittleEndian.PutUint32(b[o+213:o+217], uint32(e.WeaponSpellLevel))
		binary.LittleEndian.PutUint16(b[o+217:o+219], e.AutoSpell)
		b[o+219] = e.CastWait
		binary.LittleEndian.PutUint16(b[o+220:o+222], e.SpellFX)
		b[o+222] = e.SpellFXSpell
		if e.OffMap {
			b[o+223] = 1
		}
		binary.LittleEndian.PutUint32(b[o+224:o+228], uint32(e.EscortTarget))
		if e.HasEscortTarget {
			b[o+228] = 1
		}
		b[o+229] = e.EscortRange
		for k, p := range e.Protection {
			binary.LittleEndian.PutUint32(b[o+230+4*k:o+234+4*k], uint32(p))
		}
		b[o+250] = e.TokenSize
		b[o+251] = e.SeeInvisible
		binary.LittleEndian.PutUint32(b[o+252:o+256], uint32(e.Reaction))
		binary.LittleEndian.PutUint32(b[o+256:o+260], uint32(e.Spirit))
		if e.SuppressCorpseLoot {
			b[o+260] = 1
		}
		binary.LittleEndian.PutUint32(b[o+261:o+265], uint32(e.KillCreditSource))
		if e.HasKillCredit {
			b[o+265] = 1
		}
		b[o+266] = byte(e.KillCreditSpell)
		binary.LittleEndian.PutUint32(b[o+267:o+271], uint32(e.Load))
		binary.LittleEndian.PutUint32(b[o+271:o+275], uint32(e.Capacity))
		binary.LittleEndian.PutUint16(b[o+275:o+277], e.MapUnitID)
		copy(b[o+277:o+282], e.Resistance[:])
		binary.LittleEndian.PutUint32(b[o+282:o+286], uint32(e.Withdraw))
		binary.LittleEndian.PutUint32(b[o+286:o+290], uint32(e.Wimpy))
		if e.Humanoid {
			b[o+290] = 1
		}
		b[o+291] = e.DesiredFacing
		b[o+292] = e.TurnRemaining
		b[o+293] = e.TurnTotal
		for k := 0; k < 4; k++ {
			binary.LittleEndian.PutUint32(b[o+294+4*k:], uint32(e.PotionStats[k]))
			binary.LittleEndian.PutUint32(b[o+310+4*k:], uint32(e.PotionHeadroom[k]))
		}
		if e.HumanMovement.Present {
			b[o+326] = 1
		}
		binary.LittleEndian.PutUint16(b[o+327:], uint16(e.HumanMovement.RawSpeed))
		binary.LittleEndian.PutUint32(b[o+329:], uint32(e.HumanMovement.NativeSpeed))
		binary.LittleEndian.PutUint32(b[o+333:], uint32(e.HumanMovement.Load))
		binary.LittleEndian.PutUint32(b[o+337:], uint32(e.HumanMovement.Capacity))
		encodeSpellbook(b[o+341:o+entityLenV71], e.Book)
		b[o+454], b[o+455] = e.SecondBase, e.SecondSpread
		b[o+456] = byte(e.CurrentProfileBasis)
		_, _ = binary.Encode(b[o+entityLenV75:o+entityLen], binary.LittleEndian, e.SourceBinding)
	}
	o := records + entityLen*len(w.entities)
	for _, r := range w.routes {
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(len(r)))
		o += routeCountLen
		for _, c := range r {
			binary.LittleEndian.PutUint32(b[o:o+4], uint32(c.x))
			binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(c.y))
			o += cellLen
		}
	}
	binary.LittleEndian.PutUint32(b[o:o+4], uint32(len(w.groups)))
	o += groupCountLen
	for _, g := range w.groups {
		binary.LittleEndian.PutUint32(b[o:o+4], g.owner)
		binary.LittleEndian.PutUint32(b[o+4:o+8], g.group)
		b[o+8] = g.base
		b[o+9] = g.order
		binary.LittleEndian.PutUint32(b[o+10:o+14], uint32(g.commandedX))
		binary.LittleEndian.PutUint32(b[o+14:o+18], uint32(g.commandedY))
		o += groupRecordLen
	}
	binary.LittleEndian.PutUint32(b[o:o+4], uint32(len(w.sacks)))
	o += sackCountLen
	for _, s := range w.sacks {
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(s.X))
		binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(s.Y))
		binary.LittleEndian.PutUint32(b[o+8:o+12], s.Gold)
		binary.LittleEndian.PutUint32(b[o+12:o+16], uint32(len(s.Items)))
		o += sackHeaderLen
		for _, code := range s.Items {
			binary.LittleEndian.PutUint16(b[o:o+2], code)
			o += 2
		}
	}
	for _, stacks := range w.carried {
		codes := expandContainer(stacks)
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(len(codes)))
		o += carryCountLen
		for _, code := range codes {
			binary.LittleEndian.PutUint16(b[o:o+2], code)
			o += 2
		}
	}
	for _, eq := range w.equipment {
		for _, item := range eq {
			binary.LittleEndian.PutUint16(b[o:o+2], item.Code)
			o += 2
		}
	}
	for _, e := range w.entities {
		binary.LittleEndian.PutUint32(b[o:o+4], uint32(e.TypeID))
		binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(e.GoldChance))
		binary.LittleEndian.PutUint32(b[o+8:o+12], uint32(e.TreasureMin))
		binary.LittleEndian.PutUint32(b[o+12:o+16], uint32(e.TreasureMax))
		o += treasureRecordLen
	}
	for i, p := range w.purses {
		binary.LittleEndian.PutUint32(b[o+4*i:o+4*i+4], p)
	}
	o += purseLen
	binary.LittleEndian.PutUint16(b[o:o+2], uint16(len(w.spells)))
	o += spellCountLen
	for _, sp := range w.spells {
		binary.LittleEndian.PutUint16(b[o:o+2], sp.ID)
		binary.LittleEndian.PutUint32(b[o+2:o+6], uint32(sp.ManaCost))
		binary.LittleEndian.PutUint32(b[o+6:o+10], uint32(sp.DamageMin))
		binary.LittleEndian.PutUint32(b[o+10:o+14], uint32(sp.DamageMax))
		b[o+14] = sp.School
		b[o+15] = sp.MaxRange
		var flags uint8
		if sp.TargetsUnit {
			flags |= spellFlagTargetsUnit
		}
		if sp.Damaging {
			flags |= spellFlagDamaging
		}
		if sp.Restorative {
			flags |= spellFlagRestorative
		}
		if sp.Area {
			flags |= spellFlagArea
		}
		if sp.Defensive {
			flags |= spellFlagDefensive
		}
		b[o+16] = flags
		binary.LittleEndian.PutUint32(b[o+17:o+21], uint32(sp.AreaDuration))
		b[o+21] = sp.Distribution
		b[o+22] = sp.Radius
		binary.LittleEndian.PutUint32(b[o+23:o+27], uint32(sp.SpellDuration))
		b[o+27] = byte(sp.EffectKind)
		b[o+28] = byte(sp.EffectMode)
		binary.LittleEndian.PutUint32(b[o+29:o+33], uint32(sp.EffectMagnitude))
		binary.LittleEndian.PutUint16(b[o+33:o+35], sp.EffectDuration)
		b[o+35] = sp.Complication
		o += spellRecordLen
	}
	binary.LittleEndian.PutUint16(b[o:o+2], uint16(len(w.itemWeights)))
	o += itemWeightCountLen
	for _, iw := range w.itemWeights {
		binary.LittleEndian.PutUint16(b[o:o+2], iw.Code)
		binary.LittleEndian.PutUint32(b[o+2:o+6], uint32(iw.Weight))
		o += itemWeightRecordLen
	}
	o = w.encodeCasting(b, o)
	o = w.encodeScriptState(b, o)
	o = w.encodeStructures(b, o)
	o = w.encodeItemState(b, o)
	o = w.encodeScrolls(b, o)
	o = w.encodeScript(b, o)
	o = w.encodeOriginalDead(b, o)
	o = w.encodeInstanceWeights(b, o)
	w.encodeActorLoads(b, o)
	w.relations.encodeInto(b[len(b)-relationLen:])
	payload := w.appendAttackNotices(w.appendSavedWorldEffects(w.appendSavedFormations(w.appendStructureUses(w.appendScorched(w.appendGroupRoam(w.appendActionClocks(w.appendCarriedResumeState(w.appendSavedObjects(w.appendSavedCellPlanes(w.appendSavedMotions(w.appendNativeStrides(w.appendSavedGroupPlayerSection(w.appendSavedStructureSection(w.appendSavedGroups(w.appendSessionClock(b))))))))))))))))
	return independentBookSelectionEncoding(w, independentPendingOrderEncoding(w, w.appendTactical(w.appendStructureBlocking(w.appendCurrentTerminalActors(w.appendSavedSpellGraph(w.appendAutoHealing(w.appendSpellDeliveries(w.appendEntityIDFloor(payload)))))))))
}

func independentBookSelectionEncoding(w *World, base []byte) []byte {
	rows := []byte{}
	count := uint32(0)
	for _, e := range w.entities {
		if e.AdmittedBookSpell != 0 {
			rows = binary.LittleEndian.AppendUint32(rows, uint32(e.ID))
			rows = binary.LittleEndian.AppendUint16(rows, e.AdmittedBookSpell)
			count++
		}
	}
	if count == 0 {
		return base
	}
	previous := base[0]
	base[0] = bookSelectionFormVersion
	base = binary.LittleEndian.AppendUint32(base, count)
	base = append(base, rows...)
	base = binary.LittleEndian.AppendUint32(base, uint32(4+len(rows)))
	return append(base, previous, 'B', 'S', 'L', '1')
}

func independentPendingOrderEncoding(w *World, base []byte) []byte {
	var rows []byte
	var count uint32
	for _, e := range w.entities {
		p := e.PendingOrder
		if p.Kind == 0 {
			continue
		}
		row := make([]byte, 20)
		binary.LittleEndian.PutUint32(row, uint32(e.ID))
		row[4] = p.Kind
		if e.Retreat.Known && e.ActorState != actorStateRetreat {
			row[5] = e.ActorState + 1
		}
		if p.RowAdmitted {
			row[5] |= 0x80
		}
		binary.LittleEndian.PutUint16(row[6:], p.Spell)
		binary.LittleEndian.PutUint32(row[8:], uint32(p.Target))
		binary.LittleEndian.PutUint32(row[12:], uint32(p.X))
		binary.LittleEndian.PutUint32(row[16:], uint32(p.Y))
		rows = append(rows, row...)
		count++
	}
	if count == 0 {
		return base
	}
	version := base[0]
	base[0] = 102
	base = binary.LittleEndian.AppendUint32(base, count)
	base = append(base, rows...)
	base = binary.LittleEndian.AppendUint32(base, 4+uint32(len(rows)))
	return append(base, version, 'O', 'R', 'D', '1')
}
