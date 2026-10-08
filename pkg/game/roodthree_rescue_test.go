package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func ownerRoodRecord(t *testing.T, doc sav.DocumentData) *sav.DocumentRecordData {
	t.Helper()
	var found *sav.DocumentRecordData
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		unit, _ := savedStructureValue(r, "T08")
		identity, _ := savedStructureValue(r, "Identity")
		if unit != 443 || identity != 1358956592 {
			continue
		}
		if found != nil {
			t.Fatal("owner SAV repeats Rood's actor identity")
		}
		found = r
	}
	if found == nil {
		t.Fatal("owner SAV has no Rood actor identity")
	}
	return found
}

func ownerRoodPartyEntity(t *testing.T, f *FrontEnd) sim.Entity {
	t.Helper()
	var found sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID != 443 {
			continue
		}
		if found.MapUnitID != 0 {
			t.Fatal("two live Rood placements")
		}
		found = e
	}
	if found.MapUnitID == 0 || found.ID != 124 || !slices.Contains(f.live.mission.state.Start.IDs, found.ID) || found.Owner != 1 {
		t.Fatalf("Rood lost player party binding: id=%d owner=%d party=%v", found.ID, found.Owner, f.live.mission.state.Start.IDs)
	}
	return found
}

// Rood's Defence in saverood.sav and the two values the claims derive from
// it. HERO-DEATH-026: the death arm's first tick halves the defence, a signed
// division by two. HERO-REVIVE-068: the heal that carries health from
// non-positive back above zero doubles it with a left shift. An odd defence
// therefore does not return: 53 reads 26 while fallen and 52 once healed.
const (
	roodHealthyDefence = 53
	roodFallenDefence  = roodHealthyDefence / 2
	roodHealedDefence  = roodFallenDefence << 1
)

// requireRoodSAV reads Rood's Human record from a written SAV: current
// health, corpse stage and the defence word. DeadActors must not root him.
func requireRoodSAV(t *testing.T, phase string, raw []byte, hp int32, stage uint32, defence int16) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actor := ownerRoodRecord(t, doc)
	if slices.Contains(roodRootKeys(t, doc), uint32(1358956592)) {
		t.Fatalf("%s SAV roots Rood in DeadActors", phase)
	}
	for field, want := range map[string]uint32{"Health": uint32(uint16(hp)), "Stage": stage} {
		if got, err := savedStructureValue(actor, field); err != nil || got != want {
			t.Fatalf("%s SAV Rood %s=%d, want %d: %v", phase, field, got, want, err)
		}
	}
	word := savedRecordRawForTest(t, *actor, "UBE")
	if len(word) < 2 || int16(binary.LittleEndian.Uint16(word)) != defence {
		t.Fatalf("%s SAV Rood defence word %x, want %d", phase, word, defence)
	}
}

// roodHealer names the spell table's restorative row and the first party
// member, in party order, whose cast of it at the fallen Rood the world
// admits now.
func roodHealer(t *testing.T, f *FrontEnd, rood sim.EntityID) (sim.EntityID, uint16) {
	t.Helper()
	var heal uint16
	for _, rule := range f.live.world.Spells() {
		if rule.Restorative {
			if heal != 0 {
				t.Fatal("the spell table has two restorative rows")
			}
			heal = rule.ID
		}
	}
	if heal == 0 {
		t.Fatal("the spell table has no restorative row")
	}
	for _, id := range f.live.mission.state.Start.IDs {
		if id != rood && f.live.world.BookSpellRefusal(id, rood, uint32(heal)) == "" {
			return id, heal
		}
	}
	t.Fatal("no party caster can Heal the fallen Rood")
	return 0, 0
}

// roodFallAndHeal fells the healthy party Rood of saverood.sav to fallenHP,
// lifts him with a party caster's Heal, then SAVEs, cold LOADs, moves and
// guards him. His party binding, Group and stock survive every step.
//
// The fall is the mission script's health write (instant 34, selector 6,
// TRIG-PROPERTY-036) that sim.HeadlessDamage performs, the game's route to an
// exact health; it runs the death transition every blow runs. The Heal is the
// player's cast order at the body through the map screen's attack seam. It
// lands while the body is still dying, before any mission-loss rule applies.
// Mana and the current action are not compared once he rises: his own AI acts
// in that tick, and the saved autohealing reserve of 0 percent starts his own
// Heal.
func roodFallAndHeal(t *testing.T, input []byte, fallenHP int32) {
	f := loadRoodMission(t, input)
	healthy := ownerRoodPartyEntity(t, f)
	// The native save's XP repairs the deficient trained base without changing its effective sheet or lifecycle.
	wantTraining := sim.NativeTraining{Present: true, Levels: [6]int32{0, 70, 100, 100, 86, 64}}
	wantXP := [6]int32{0, 788746, 12527069, 12527588, 3406036, 404319}
	items, itemsOK := f.live.world.EquippedItems(healthy.ID)
	bonus := mapload.EquippedSkillBonus(items, false)
	if !itemsOK || healthy.NativeTraining != wantTraining || healthy.SkillXP != wantXP || healthy.Skill != ([6]int32{0, 70, 100, 100, 86, 64}) ||
		f.live.world.Rules().EffectiveSkill(wantTraining.Levels[3], bonus[3]) != healthy.Skill[3] || f.live.world.NativeTrainingNeedsProducer(healthy.ID) {
		policy := f.live.world.CurrentPolicy()
		t.Fatalf("owner Rood lost the repaired trained base, XP or effective sheet: base %v skill %v XP %v worn %v items %t clock %t object carrier %t source %t/%d profile %d", healthy.NativeTraining, healthy.Skill, healthy.SkillXP, bonus, itemsOK, policy.ClockKnown, policy.ObjectCarrier, healthy.ActorLoad.Present, healthy.ActorLoad.Source.Class, healthy.CurrentProfileBasis)
	}
	if !healthy.Alive() || healthy.HP <= 0 || healthy.Decay != sim.DecayNone || healthy.Defence != roodHealthyDefence {
		t.Fatalf("source Rood is not a healthy party member with Defence %d: %+v", roodHealthyDefence, healthy)
	}
	stockOf := func(front *FrontEnd) sim.Stock {
		for _, stock := range front.live.world.Stock() {
			if stock.ID == healthy.ID {
				return stock
			}
		}
		t.Fatal("Rood has no current stock record")
		return sim.Stock{}
	}
	baselineStock := stockOf(f)
	checkStock := func(phase string, got sim.Stock) {
		if !reflect.DeepEqual(got.OrderedStacks, baselineStock.OrderedStacks) ||
			!reflect.DeepEqual(got.Items, baselineStock.Items) ||
			!reflect.DeepEqual(got.ItemInstances, baselineStock.ItemInstances) ||
			got.Equipped != baselineStock.Equipped ||
			!reflect.DeepEqual(got.EquippedItems, baselineStock.EquippedItems) {
			t.Fatalf("%s changed Rood's carried or worn items: before=%+v after=%+v", phase, baselineStock, got)
		}
	}
	checkFallen := func(phase string, got sim.Entity) {
		if got.HP != fallenHP || got.Decay != sim.DecayFallen || got.Defence != roodFallenDefence ||
			got.HasTarget || got.HasAttackTarget || got.CastWait != 0 || got.X != healthy.X || got.Y != healthy.Y ||
			got.SourceBinding != healthy.SourceBinding || got.Group != healthy.Group ||
			got.CommandGroup != healthy.CommandGroup {
			t.Fatalf("%s lost Rood's recoverable party body or kept an action: before=%+v after=%+v", phase, healthy, got)
		}
	}
	checkHealthy := func(phase string, got sim.Entity, hp int32) {
		if got.HP != hp || got.HP > got.MaxHP || got.Defence != roodHealedDefence || !got.Alive() ||
			got.Decay != sim.DecayNone || got.Dwell != 0 ||
			got.ID != healthy.ID || got.MapUnitID != healthy.MapUnitID ||
			got.SourceBinding != healthy.SourceBinding || got.Owner != healthy.Owner ||
			got.Group != healthy.Group || got.CommandGroup != healthy.CommandGroup ||
			got.X != healthy.X || got.Y != healthy.Y || got.Class != healthy.Class ||
			got.TypeID != healthy.TypeID || got.Humanoid != healthy.Humanoid ||
			got.MaxHP != healthy.MaxHP || got.MaxMana != healthy.MaxMana ||
			got.Speed != healthy.Speed || got.HumanMovement != healthy.HumanMovement ||
			got.Load != healthy.Load || got.Capacity != healthy.Capacity ||
			got.ToHit != healthy.ToHit || got.Absorption != healthy.Absorption ||
			got.DamageBase != healthy.DamageBase || got.DamageSpread != healthy.DamageSpread ||
			got.Reach != healthy.Reach || got.AttackCharge != healthy.AttackCharge ||
			got.AttackRelax != healthy.AttackRelax || got.SkillXP != healthy.SkillXP ||
			got.Skill != healthy.Skill || got.NativeTraining != healthy.NativeTraining ||
			got.CurrentProfileBasis != healthy.CurrentProfileBasis {
			t.Fatalf("%s changed Rood: HP %d/%d->%d/%d mana max %d->%d skill %v->%v XP %v->%v base %v->%v defence %d->%d", phase, healthy.HP, healthy.MaxHP, got.HP, got.MaxHP, healthy.MaxMana, got.MaxMana, healthy.Skill, got.Skill, healthy.SkillXP, got.SkillXP, healthy.NativeTraining, got.NativeTraining, healthy.Defence, got.Defence)
		}
		wantLoad := healthy.ActorLoad
		if wantLoad.Present {
			wantLoad.Source.Stats[8] = uint16(got.HP)
			binary.LittleEndian.PutUint16(wantLoad.Source.Defence[:], uint16(got.Defence))
		}
		if got.ActorLoad != wantLoad {
			t.Fatalf("%s changed Rood's source actor/load basis: before=%+v after=%+v", phase, wantLoad, got.ActorLoad)
		}
	}
	checkTraining := func(phase string, got sim.Entity) {
		if got.Skill != healthy.Skill || got.NativeTraining != healthy.NativeTraining || got.SkillXP != healthy.SkillXP || got.MaxHP != healthy.MaxHP || got.MaxMana != healthy.MaxMana {
			t.Fatalf("%s changed the guarded training sheet: skill %v base %v XP %v pools %d/%d", phase, got.Skill, got.NativeTraining, got.SkillXP, got.MaxHP, got.MaxMana)
		}
	}
	cached := f.live.derivedSkills[healthy.ID]
	cached.levels[3]--
	f.live.derivedSkills[healthy.ID] = cached
	f.live.recomputeRaisedSkills()
	checkTraining("cached refresh", ownerRoodPartyEntity(t, f))
	f.live.switchInventorySubject(uint32(healthy.ID))
	if !f.live.invSubjectSet || f.live.invSubject.ID != uint32(healthy.ID) {
		t.Fatal("Rood inventory subject was not selected")
	}
	f.live.invSkill[3]--
	f.live.rearm()
	checkTraining("level-only rearm", ownerRoodPartyEntity(t, f))
	if err := f.live.world.HeadlessDamage(healthy.ID, healthy.HP-fallenHP); err != nil {
		t.Fatal(err)
	}
	fallen := ownerRoodPartyEntity(t, f)
	checkFallen("fall", fallen)
	if !fallen.Dying() || fallenHP == 0 && (!fallen.Downed() || fallen.Dead()) ||
		fallenHP < 0 && (!fallen.Dead() || !fallen.OrdinaryTargetable()) {
		t.Fatalf("HP %d has the wrong recoverable life class: %+v", fallenHP, fallen)
	}
	for _, terminal := range f.live.world.CurrentTerminalActors() {
		if terminal.ID == fallen.ID {
			t.Fatal("recoverable Rood became a terminal actor")
		}
	}
	for _, dead := range f.live.world.OriginalDeadActors() {
		if dead.ID == fallen.ID {
			t.Fatal("recoverable Rood entered the original dead registry")
		}
	}
	requireRoodSAV(t, "fallen", saveRoodMission(t, f), fallenHP, 1, roodFallenDefence)
	checkStock("fall", stockOf(f))
	caster, heal := roodHealer(t, f, fallen.ID)
	f.live.attackOrCast(uint32(caster), uint32(fallen.ID), uint32(heal), 0, 0, false)
	var restored int32
	for landed := false; !landed; {
		var casts []sim.CastEvent
		f.live.tickWithCastSink(func(events []sim.CastEvent) { casts = append(casts, events...) })
		for _, cast := range casts {
			if cast.Target != fallen.ID {
				continue
			}
			if landed || cast.Caster != caster || cast.Spell != heal || cast.HealthRestored <= 0 {
				t.Fatalf("unexpected cast at the fallen Rood: %+v", cast)
			}
			landed, restored = true, cast.HealthRestored
		}
		if body := ownerRoodPartyEntity(t, f); !landed {
			if !body.Dying() {
				t.Fatalf("party caster %d's Heal did not land inside the dying window: %+v", caster, body)
			}
			checkFallen("dying window", body)
		}
	}
	healed := ownerRoodPartyEntity(t, f)
	checkHealthy("heal", healed, fallenHP+restored)
	checkStock("heal", stockOf(f))
	written := saveRoodMission(t, f)
	requireRoodSAV(t, "healed", written, healed.HP, 0, roodHealedDefence)
	cold := loadRoodMission(t, written)
	resumed := ownerRoodPartyEntity(t, cold)
	checkHealthy("cold LOAD", resumed, healed.HP)
	checkStock("cold LOAD", stockOf(cold))
	at := sim.CellPoint{X: resumed.X + 1, Y: resumed.Y}
	liveWalk(t, cold.live, resumed.ID, at.X, at.Y)
	moved := ownerRoodPartyEntity(t, cold)
	checkTraining("cold next move", moved)
	if moved.X != at.X || moved.Y != at.Y || !moved.Alive() || moved.Defence != roodHealedDefence {
		t.Fatalf("recovered Rood did not walk: %+v", moved)
	}
	cold.live.stance(uint32(moved.ID), true)
	cold.live.tick()
	guard := ownerRoodPartyEntity(t, cold)
	checkTraining("cold next Guard", guard)
	order, _, ok := cold.live.world.FrozenGroupAI(guard.Owner, guard.CommandGroup)
	if !ok || order != uint8(sim.OrderGuard) || !guard.Alive() || guard.Defence != roodHealedDefence {
		t.Fatalf("recovered Rood did not accept Guard after walking: order=%d ok=%t actor=%+v", order, ok, guard)
	}
	requireRoodSAV(t, "guard", saveRoodMission(t, cold), guard.HP, 0, roodHealedDefence)
}

func TestRoodThreeOwnerSaveRescueContinuation(t *testing.T) {
	root, original := os.Getenv("AGAINROM_ROOD_THREE_DIR"), os.Getenv("AGAINROM_ROOD_SAV")
	if os.Getenv("AGAINROM_ASSETS") == "" || root == "" || original == "" {
		t.Skip("no AGAINROM_ASSETS, AGAINROM_ROOD_THREE_DIR or AGAINROM_ROOD_SAV: Rood lifecycle inputs are required")
	}
	source, sourceDoc := roodStageHealSource(t)
	requireRoodPlacement(t, sourceDoc, 0, 1, false)
	t.Run("downed_source_then_heal", func(t *testing.T) {
		f := loadRoodMission(t, source)
		before := worldEntityByRuntimeID(t, f, 258)
		if before.ID != 124 || before.HP != 0 || before.MaxHP != 86 || before.Defence != 26 || before.Group != 16 ||
			!before.Downed() || before.Dead() || before.Decay != sim.DecayFallen || before.Owner != 5 || before.SourceBinding.Identity != 302922488 {
			t.Fatalf("downed Rood lost live source identity: %+v", before)
		}
		for _, terminal := range f.live.world.CurrentTerminalActors() {
			if terminal.ID == before.ID {
				t.Fatal("downed Rood became terminal")
			}
		}
		if err := f.live.world.HeadlessHeal(before.ID); err != nil {
			t.Fatal(err)
		}
		alive := worldEntityByRuntimeID(t, f, 258)
		if alive.ID != before.ID || alive.SourceBinding != before.SourceBinding || alive.Group != before.Group || alive.Owner != before.Owner ||
			alive.HP != 86 || alive.Decay != sim.DecayNone || alive.Defence != 52 {
			t.Fatalf("heal lost actor identity, lifecycle or Defence: before=%+v after=%+v", before, alive)
		}
		saved := saveRoodMission(t, f)
		written, err := sav.DecodeDocumentData(saved)
		if err != nil {
			t.Fatal(err)
		}
		requireRoodPlacement(t, written, int16(alive.HP), 0, false)
		cold := loadRoodMission(t, saved)
		cold.live.tick()
		continued := worldEntityByRuntimeID(t, cold, 258)
		if continued.ID != alive.ID || continued.SourceBinding != alive.SourceBinding || continued.Owner != alive.Owner ||
			continued.HP <= 0 || continued.Decay != sim.DecayNone || continued.Defence != alive.Defence {
			t.Fatalf("healed Rood changed across SAVE, cold LOAD and tick: before=%+v after=%+v", alive, continued)
		}
	})
	for _, tc := range []struct {
		name, hash string
		hp         uint32
	}{
		{"saverood.sav", "C3C5B565556C9A90618DF2F281F1D555E897589A233DC8163E39E543F2586AB4", 110},
		{"saveвввв.sav", "4690D07AB3ED1C6DCF806929102FC74C61BE8F08E5A2E742663C9F9123E2C3A4", 115},
		{"saveввввs.sav", "81CA510CD9FB6870BFF9C8736A8F66184EE67AD6604B7553D52F327F86CFA27E", 115},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join(root, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%X", sha256.Sum256(input)); got != tc.hash {
				t.Fatalf("owner SAV hash=%s, want %s", got, tc.hash)
			}
			inputDoc, err := sav.DecodeDocumentData(input)
			if err != nil {
				t.Fatal(err)
			}
			inputRecord := ownerRoodRecord(t, inputDoc)
			if slices.Contains(roodRootKeys(t, inputDoc), uint32(1358956592)) {
				t.Fatal("joined Rood is still a DeadActors root")
			}
			for field, want := range map[string]uint32{"Health": tc.hp, "Stage": 1, "RuntimeID": 125} {
				if got, err := savedStructureValue(inputRecord, field); err != nil || got != want {
					t.Fatalf("input Rood %s=%d, want %d: %v", field, got, want, err)
				}
			}
			if tc.name == "saverood.sav" {
				t.Run("healthy_fall_heal", func(t *testing.T) {
					for _, fallenHP := range []int32{0, -9} {
						t.Run(fmt.Sprintf("hp_%d", fallenHP), func(t *testing.T) {
							roodFallAndHeal(t, input, fallenHP)
						})
					}
				})
			}
			f := loadRoodMission(t, input)
			rood := ownerRoodPartyEntity(t, f)
			if rood.HP != int32(tc.hp) || rood.Decay != sim.DecayNone || rood.Defence <= 0 {
				t.Fatalf("restored Rood health, Decay or Defence: %+v", rood)
			}
			at := sim.CellPoint{X: rood.X + 1, Y: rood.Y}
			liveWalk(t, f.live, rood.ID, at.X, at.Y)
			moved := ownerRoodPartyEntity(t, f)
			if moved.X != at.X || moved.Y != at.Y || moved.HP <= 0 || moved.Decay != sim.DecayNone {
				t.Fatalf("restored Rood did not walk: before=%+v after=%+v", rood, moved)
			}
			f.live.stance(uint32(rood.ID), true)
			f.live.tick()
			guarding := ownerRoodPartyEntity(t, f)
			order, _, ok := f.live.world.FrozenGroupAI(guarding.Owner, guarding.CommandGroup)
			if !ok || order != uint8(sim.OrderGuard) {
				t.Fatalf("restored Rood did not accept Guard: order=%d group=%d ok=%t", order, guarding.CommandGroup, ok)
			}
			written := saveRoodMission(t, f)
			writtenDoc, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			writtenRood := ownerRoodRecord(t, writtenDoc)
			if slices.Contains(roodRootKeys(t, writtenDoc), uint32(1358956592)) {
				t.Fatal("SAVE rooted the living party Rood as dead")
			}
			for field, want := range map[string]uint32{"Health": uint32(guarding.HP), "Stage": 0} {
				if got, err := savedStructureValue(writtenRood, field); err != nil || got != want {
					t.Fatalf("SAVE Rood %s=%d, want %d: %v", field, got, want, err)
				}
			}
			cold := loadRoodMission(t, written)
			resumed := ownerRoodPartyEntity(t, cold)
			if resumed.ID != guarding.ID || resumed.HP != guarding.HP || resumed.Defence != guarding.Defence ||
				resumed.Decay != sim.DecayNone || resumed.X != guarding.X || resumed.Y != guarding.Y {
				t.Fatalf("Rood party identity or lifecycle changed across cold LOAD: before=%+v after=%+v", guarding, resumed)
			}
			cold.live.stance(uint32(resumed.ID), false)
			cold.live.tick()
			next := ownerRoodPartyEntity(t, cold)
			order, _, ok = cold.live.world.FrozenGroupAI(next.Owner, next.CommandGroup)
			if !ok || order != uint8(sim.OrderStandGround) || next.HP <= 0 || next.Decay != sim.DecayNone || next.Defence != resumed.Defence {
				t.Fatalf("Rood's next action or lifecycle failed: order=%d ok=%t actor=%+v", order, ok, next)
			}
			end, err := sav.DecodeDocumentData(saveRoodMission(t, cold))
			if err != nil {
				t.Fatal(err)
			}
			endRood := ownerRoodRecord(t, end)
			for field, want := range map[string]uint32{"Health": uint32(next.HP), "Stage": 0} {
				if got, err := savedStructureValue(endRood, field); err != nil || got != want {
					t.Fatalf("next SAVE Rood %s=%d, want %d: %v", field, got, want, err)
				}
			}
		})
	}
}
