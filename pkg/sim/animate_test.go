package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestSecondAnimateNamePolicyKeepsLivingPersonExperience(t *testing.T) {
	hero, victim := spEnt(1, 14, 14), spEnt(2, 16, 16)
	hero.Humanoid, hero.GainsXP, hero.XPSlot, hero.Owner = true, true, 1, 7
	hero.TypeID = HumanTypeID
	victim.Humanoid, victim.SuppressCorpseLoot, victim.Owner = true, true, 8
	victim.TypeID, victim.SkillXP[1] = 1, 400000
	w := secondWorld(t, 53, Bounds{Width: 40, Height: 40}, nil, hero, victim)
	w.payExperience(0, 1, 20, true)
	if got := w.entities[0].SkillXP; got != ([skillSlots]int32{0, 100}) {
		t.Fatalf("living NPC-named person: SkillXP=%v, want [0 100 0 0 0 0]", got)
	}
}

func TestSecondAnimateRewardMarkByteForm(t *testing.T) {
	w := animateWorld(t, animateCorpse(2, 2))
	plain, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	plainHash := w.Hash()
	w.entities[0].SuppressRewards = true
	marked, err := w.MarshalBinary()
	if err != nil || marked[0] != rewardMarkFormVersion || bytes.Equal(plain, marked) || w.Hash() == plainHash {
		t.Fatalf("reward mark did not enter byte form and hash: %v", err)
	}
	cold := cloneWorld(t, w)
	if !cold.entities[0].SuppressRewards || cold.entities[0].SuppressCorpseLoot || cold.Hash() != w.Hash() {
		t.Fatal("reward mark was lost or became the name policy")
	}
	if err := cold.UnmarshalBinary(plain); err != nil || cold.entities[0].SuppressRewards || cold.Hash() != plainHash {
		t.Fatalf("unmarked form did not clear the reward mark: %v", err)
	}
	for _, indices := range [][]uint32{{2}, {0, 0}, {1, 0}} {
		ws := beginWrapper(plain)
		bad := bytes.Clone(plain)
		for _, index := range indices {
			bad = binary.LittleEndian.AppendUint32(bad, index)
		}
		bad = endWrapper(bad, ws, rewardMarkFormVersion)
		if err := cold.UnmarshalBinary(bad); err == nil || cold.Hash() != plainHash {
			t.Fatalf("invalid reward indices %v accepted or changed World: %v", indices, err)
		}
	}
}

func TestSecondAnimateKeepsItsSourceRowStores(t *testing.T) {
	for kind := DecayStage(2); kind <= 4; kind++ {
		w := animateWorld(t, animateCorpse(2, kind))
		for i := range w.spellCreatures {
			e := &w.spellCreatures[i].Actor
			e.NativeBasis = NativeActorBasis{}
			e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
			e.ActorLoad.Source.Stats[0] = 13
		}
		if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) || len(w.entities) != 2 {
			t.Fatal("source-backed raise failed")
		}
		e := w.entities[1]
		s := e.ActorLoad.Source
		body := uint16(65)
		if kind == 3 {
			body = 33
		}
		if kind == 4 {
			body = 13
		}
		if s.Stats[0] != body || s.Stats[1] != 24 || s.Stats[2] != 39 || s.Stats[3] != 31 || s.Stats[8] != uint16(e.HP) || s.Stats[9] != uint16(e.MaxHP) || binary.LittleEndian.Uint16(s.Attack[:]) != 73 || binary.LittleEndian.Uint16(s.Defence[:]) != 81 {
			t.Fatal("source row did not take the corpse stores")
		}
		cloneWorld(t, w)
	}
}

func animateWorld(t *testing.T, corpses ...Entity) *World {
	t.Helper()
	caster := effectMage(1, 14, 14, 1<<22)
	caster.Owner, caster.Group = 7, 50
	w := secondWorld(t, 53, Bounds{Width: 40, Height: 40}, []SpellRule{secondRow(SpellRule{ID: 22, ManaCost: 2, TargetsUnit: true, MaxRange: 10})}, append([]Entity{caster}, corpses...)...)
	var creatures []SpellCreature
	for i, name := range []string{"F_Zombie.1", "F_Skeleton.1", "Ghost"} {
		e := NewActor(ActorDefinition{Class: int32(71 + i), TypeID: int32(71 + i), HP: 123, MaxHP: 123, Speed: int32(3 + i), TokenSize: 1,
			NativeBasis: NativeActorBasis{}.WithBody(13), Reaction: 14, Mind: 15, Spirit: 16, ToHit: 17, Defence: 18, Absorption: 19,
			DamageBase: 23, DamageSpread: 7, Protection: [5]int32{11, 12, 13, 14, 15}, Resistance: [5]uint8{21, 22, 23, 24, 25},
			XPValue: 9876, GoldChance: 100, TreasureMin: 8, TreasureMax: 8}, ActorPlacement{})
		creatures = append(creatures, SpellCreature{Name: name, Actor: e})
	}
	w.DeclareSpellCreatures(creatures)
	return w
}

func animateCorpse(id EntityID, kind DecayStage) Entity {
	e := spEnt(id, 16, 16)
	e.HP, e.MaxHP, e.Decay = -25, 121, kind
	e.NativeBasis, e.Reaction, e.Mind, e.Spirit = NativeActorBasis{}.WithBody(65), 47, 39, 31
	e.ToHit, e.Defence, e.TypeID = 73, 81, 35
	return e
}

func TestSecondAnimateRaisesEachCorpseKind(t *testing.T) {
	for kind := DecayStage(2); kind <= 4; kind++ {
		t.Run(string(rune('0'+kind)), func(t *testing.T) {
			w := animateWorld(t, animateCorpse(2, kind))
			before := w.entities[0]
			if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) {
				t.Fatal("Animate Dead did not raise the corpse")
			}
			if len(w.entities) != 2 {
				t.Fatalf("actors=%d, want caster and raised actor", len(w.entities))
			}
			got, template := w.entities[1], w.spellCreatures[kind-2].Actor
			body, hp := uint16(65), int32(121)
			if kind == 3 {
				body, hp = 33, 60
			}
			if kind == 4 {
				body, hp = 13, 60
			}
			if got.TypeID != template.TypeID || got.NativeBasis.Body != body || got.HP != hp || got.MaxHP != hp || got.Reaction != 24 || got.Mind != 39 || got.Spirit != 31 || got.ToHit != 73 || got.Defence != 81 {
				t.Fatalf("raised statistics: %+v", got)
			}
			if got.Speed != template.Speed || got.DamageBase != template.DamageBase || got.Absorption != template.Absorption || got.Protection != template.Protection || got.Resistance != template.Resistance {
				t.Fatal("raise lost untouched template statistics")
			}
			if got.X != 16 || got.Y != 16 || got.Owner != before.Owner || got.Group == 0 || got.Group == before.Group || got.summonKey != 0 || !got.SuppressRewards || got.SuppressCorpseLoot != template.SuppressCorpseLoot {
				t.Fatalf("raised placement/owner/group/marks: %+v", got)
			}
			terminal := w.CurrentTerminalActors()
			if w.entities[0] != before || len(terminal) != 1 || terminal[0].HP != -10001 || terminal[0].Stage != 5 {
				t.Fatal("caster or consumed corpse state differs")
			}
			cold := cloneWorld(t, w)
			if cold.Hash() != w.Hash() {
				t.Fatal("raised actor byte form differs")
			}
		})
	}
}

func TestSecondAnimatePicksKindBeforeListOrderAndSkipsUndeadTypes(t *testing.T) {
	a, b, c, d := animateCorpse(2, 3), animateCorpse(3, 2), animateCorpse(4, 2), animateCorpse(5, 2)
	b.TypeID, c.TypeID = 82, 98
	w := animateWorld(t, a, b, c, d)
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) || len(w.entities) != 5 || indexOfEntity(w.entities, 5) >= 0 {
		for _, e := range w.entities {
			t.Logf("id %d type %d kind %d", e.ID, e.TypeID, e.Decay)
		}
		t.Fatal("kind-2 eligible corpse was not chosen")
	}
	if w.entities[1].Decay != 3 || w.entities[2].Decay != 2 || w.entities[3].Decay != 2 || w.entities[4].TypeID != 71 {
		t.Fatal("an excluded corpse changed or the wrong template was built")
	}
	for _, typ := range []int32{81, 99} {
		e := animateCorpse(2, 2)
		e.TypeID = typ
		w = animateWorld(t, e)
		if !w.ordinaryEffectPayload(0, 1, w.spells[0], 1) || len(w.entities) != 2 {
			t.Fatalf("type %d excluded", typ)
		}
	}
}

func TestSecondAnimateFailedPlacementOnlyConsumesCorpse(t *testing.T) {
	blocker := spEnt(3, 16, 16)
	blocker.TokenSize = 1
	w := animateWorld(t, animateCorpse(2, 2), blocker)
	before := w.Entities()
	groups := len(w.groups)
	w.ordinaryEffectPayload(0, 1, w.spells[0], 17)
	before = append(before[:1], before[2:]...)
	terminal := w.CurrentTerminalActors()
	if !reflect.DeepEqual(w.Entities(), before) || len(terminal) != 1 || terminal[0].HP != -10001 || terminal[0].Stage != 5 {
		t.Fatal("failed placement changed actors or lost the consumed corpse")
	}
	if len(w.groups) != groups {
		t.Fatal("failed placement created a group")
	}
}

func TestSecondAnimateClampsWithoutSearching(t *testing.T) {
	e := animateCorpse(2, 2)
	e.X, e.Y = 2, 38
	w := animateWorld(t, e)
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) || len(w.entities) != 2 || w.entities[1].X != 8 || w.entities[1].Y != 31 {
		t.Fatal("clamped raise failed")
	}
}

func TestSecondAnimateStaysOutsideCapAndDropsNothing(t *testing.T) {
	w := animateWorld(t, animateCorpse(2, 2))
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) || len(w.entities) != 2 {
		t.Fatal("no raised actor")
	}
	before := w.entities[1]
	stream := w.rng
	for range 128 {
		w.summonCapPass()
	}
	if w.entities[1] != before || w.rng != stream {
		t.Fatal("raised actor participates in summon cap")
	}
	w.entities[0].Humanoid, w.entities[0].GainsXP, w.entities[0].XPSlot = true, true, 1
	w.entities[0].MaxMana, w.entities[0].KnownSpells, w.entities[0].Owner = 0, 0, 8
	w.payExperience(0, 1, 60, true)
	if w.entities[0].SkillXP != ([skillSlots]int32{}) {
		t.Fatal("human attacker earned experience from raised actor")
	}
	w.carried[1] = []ItemStack{StackItem(ItemInstance{Code: 0x101, Kind: 1, Price: 27}, 1)}
	w.equipment[1][1] = ItemInstance{Code: 0x102, Kind: 1, Price: 29}
	if w.hasTerminalLoot(1) {
		t.Fatal("raised actor reports terminal loot")
	}
	deathStream := w.rng
	w.entities[1].setCurrentHealth(-50)
	w.clearFelled(1)
	w.dropTerminalLoot(1)
	if len(w.sacks) != 0 || len(w.carried[1]) != 0 || !reflect.DeepEqual(w.equipment[1], [EquipSlots]ItemInstance{}) || w.rng != deathStream {
		t.Fatal("raised death dropped loot")
	}
}

func TestFirstControlSpiritKeepsRecordedWorld(t *testing.T) {
	w := csItemWorld(t, DomainGround)
	if !w.ordinaryEffectPayload(0, 2, w.spells[0], 17) {
		t.Fatal("first-game raise refused")
	}
	t.Logf("ROM1 Control Spirit hash=%016x", w.Hash())
	if w.Hash() != 0x038cb620815d4b71 {
		t.Fatal("first-game World hash changed")
	}
	if len(w.entities) != 3 || w.entities[2].Owner != 7 || w.entities[2].Reaction != 21 {
		t.Fatal("first-game result changed")
	}
}
