package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
)

// A body joins the dead list at its teardown, so the earlier teardown stands
// first. An order other than id order enters the byte form and the hash and
// survives a cold load and LoadDeadList; a malformed order is refused with
// the World unchanged.
func TestSecondDeadRankByteForm(t *testing.T) {
	w := animateWorld(t, fallenBody(2, 16, 16, -15, 6), fallenBody(3, 16, 16, -15, 3))
	w.entities[0].KnownSpells = 0
	for range 8 {
		Step(w, nil)
	}
	// The tick register is non-default after a step and would add its own
	// wrapper; this test reads the dead-rank wrapper outermost.
	w.setRegisterAt(999, 0)
	if got := w.DeadList(); !reflect.DeepEqual(got, []EntityID{3, 2}) {
		t.Fatalf("dead list %v, want [3 2]", got)
	}
	ranked, err := w.MarshalBinary()
	if err != nil || ranked[0] != deadRankFormVersion {
		t.Fatalf("the order did not enter the byte form: %v", err)
	}
	cold := cloneWorld(t, w)
	if !reflect.DeepEqual(cold.DeadList(), w.DeadList()) || cold.Hash() != w.Hash() {
		t.Fatal("the order was lost across a cold load")
	}
	unranked := cloneWorld(t, w)
	unranked.entities[1].deadRank, unranked.entities[2].deadRank = 0, 0
	plain, err := unranked.MarshalBinary()
	if err != nil || plain[0] == deadRankFormVersion || unranked.Hash() == w.Hash() {
		t.Fatalf("id order wrote a form or the order left the hash: %v", err)
	}
	unranked.LoadDeadList([]EntityID{1, 3, 2})
	if unranked.Hash() != w.Hash() {
		t.Fatal("the loaded order differs")
	}
	for _, words := range [][]uint32{{}, {1}, {1, 2}, {3, 1}, {0, 2, 1}, {2, 2}, {2, 1, 1}} {
		ws := beginWrapper(plain)
		bad := bytes.Clone(plain)
		for _, v := range words {
			bad = binary.LittleEndian.AppendUint32(bad, v)
		}
		bad = endWrapper(bad, ws, deadRankFormVersion)
		if err := cold.UnmarshalBinary(bad); err == nil || cold.Hash() != w.Hash() {
			t.Fatalf("order words %v accepted or changed the World: %v", words, err)
		}
	}
	good := endWrapper(binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(bytes.Clone(plain), 2), 1), beginWrapper(plain), deadRankFormVersion)
	if err := cold.UnmarshalBinary(good); err != nil || cold.Hash() != w.Hash() {
		t.Fatalf("the written order was refused: %v", err)
	}
}

// fallenBody is a body lying where it fell at health hp; its teardown comes
// on the tick its dwell of dwell ticks ends.
func fallenBody(id EntityID, x, y, hp int32, dwell uint16) Entity {
	e := animateCorpse(id, DecayFallen)
	e.X, e.Y, e.HP, e.Dwell = x, y, hp, dwell
	return e
}

// goneBody is a body past the ladder's last rung: the next decay pass, or a
// direct removal, leaves its terminal row.
func goneBody(id EntityID, x, y int32) Entity {
	e := animateCorpse(id, decayLast)
	e.X, e.Y, e.HP = x, y, decayGoneHP-100
	return e
}

// actorKinds names each actor by id, decay stage and health.
func actorKinds(w *World) []string {
	var out []string
	for _, e := range w.entities {
		out = append(out, fmt.Sprintf("%d:k%d:hp%d", e.ID, e.Decay, e.HP))
	}
	return out
}

func terminalIDs(w *World) []EntityID {
	var ids []EntityID
	for _, r := range w.CurrentTerminalActors() {
		ids = append(ids, r.ID)
	}
	return ids
}

// Kind-2 bodies 2 and 3 lie on 16,16 and 3 tears down first; kind-3 body 4
// there joins before both. The arm takes 3, and the pool returns 3 itself, so
// the copies read the template (R2-ENGINE-780, R2-ENGINE-781).
func TestSecondAnimateTakesTheFirstCorpseOfTheDeadList(t *testing.T) {
	w := animateWorld(t, fallenBody(2, 16, 16, -15, 6), fallenBody(3, 16, 16, -15, 3), fallenBody(4, 16, 16, -25, 1), fallenBody(5, 18, 16, -15, 2))
	w.entities[0].KnownSpells = 0
	for range 8 {
		Step(w, nil)
	}
	for i, kind := range []DecayStage{2, 2, 3, 2} {
		if e := w.entities[i+1]; e.Decay != kind || e.Alive() {
			t.Fatalf("body %d: kind %d alive %v before the cast", e.ID, e.Decay, e.Alive())
		}
	}
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) {
		t.Fatal("Animate Dead did not raise a corpse")
	}
	if len(w.entities) != 5 || w.entities[1].ID != 2 || w.entities[1].Decay != 2 || w.entities[3].Decay != 3 || w.entities[4].Decay != 2 {
		t.Fatalf("the arm took another body than 3: %+v", actorKinds(w))
	}
	got, template := w.entities[2], w.spellCreatures[0].Actor
	if got.ID != 3 || !got.Alive() || got.TypeID != template.TypeID || got.X != 16 || got.Y != 16 {
		t.Fatalf("raised actor %d type %d at %d,%d, want id 3 type %d at 16,16", got.ID, got.TypeID, got.X, got.Y, template.TypeID)
	}
	if got.MaxHP != template.MaxHP || got.HP != template.MaxHP || got.NativeBasis.Body != template.NativeBasis.Body ||
		got.Reaction != template.Reaction/2+1 || got.Mind != template.Mind || got.Spirit != template.Spirit || got.ToHit != template.ToHit || got.Defence != template.Defence {
		t.Fatalf("the copies did not read the template: health %d/%d body %d reaction %d", got.HP, got.MaxHP, got.NativeBasis.Body, got.Reaction)
	}
	if ids := terminalIDs(w); len(ids) != 0 {
		t.Fatalf("the reused corpse kept a terminal row: %v", ids)
	}
	cloneWorld(t, w)
}

// An older unit of kind 5 stands before the corpse on the global list, so the
// pool returns it: the creature takes its id and copies the corpse, and the
// corpse stays consumed at kind 5.
func TestSecondAnimateReusesTheOlderPooledUnit(t *testing.T) {
	w := animateWorld(t, goneBody(2, 10, 10), animateCorpse(3, 2))
	w.remove([]EntityID{2})
	if ids := terminalIDs(w); !reflect.DeepEqual(ids, []EntityID{2}) {
		t.Fatalf("pooled rows %v, want [2]", ids)
	}
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) {
		t.Fatal("Animate Dead did not raise the corpse")
	}
	if len(w.entities) != 2 {
		t.Fatalf("actors=%d, want caster and raised actor", len(w.entities))
	}
	got := w.entities[1]
	if got.ID != 2 || got.X != 16 || got.Y != 16 || got.MaxHP != 121 || got.HP != 121 || got.Reaction != 24 || got.Mind != 39 || got.NativeBasis.Body != 65 {
		t.Fatalf("raised actor %d at %d,%d health %d/%d reaction %d, want id 2 copying corpse 3 at 16,16", got.ID, got.X, got.Y, got.HP, got.MaxHP, got.Reaction)
	}
	terminal := w.CurrentTerminalActors()
	if len(terminal) != 1 || terminal[0].ID != 3 || terminal[0].HP != controlSpiritCorpseHP || terminal[0].Stage != consumedCorpseStage {
		t.Fatalf("terminal rows %+v, want the consumed corpse 3 alone", terminal)
	}
	cloneWorld(t, w)
}

// A blocked cell fails the placement after the pool took its unit, which
// leaves; the corpse stays consumed unless it was that unit (R2-ENGINE-782).
func TestSecondAnimateFailedPlacementLeavesTheTakenUnitGone(t *testing.T) {
	blocker := spEnt(9, 16, 16)
	blocker.TokenSize = 1
	w := animateWorld(t, animateCorpse(2, 2), blocker)
	w.ordinaryEffectPayload(0, 1, w.spells[0], 17)
	if len(w.entities) != 2 || w.entities[1].ID != 9 {
		t.Fatalf("failed placement left actors %+v", actorKinds(w))
	}
	if ids := terminalIDs(w); len(ids) != 0 {
		t.Fatalf("the corpse the pool took kept a terminal row: %v", ids)
	}
	cloneWorld(t, w)

	w = animateWorld(t, goneBody(2, 10, 10), animateCorpse(3, 2), blocker)
	w.remove([]EntityID{2})
	w.ordinaryEffectPayload(0, 1, w.spells[0], 17)
	if len(w.entities) != 2 || w.entities[1].ID != 9 {
		t.Fatalf("failed placement left actors %+v", actorKinds(w))
	}
	terminal := w.CurrentTerminalActors()
	if len(terminal) != 1 || terminal[0].ID != 3 || terminal[0].HP != controlSpiritCorpseHP || terminal[0].Stage != consumedCorpseStage {
		t.Fatalf("terminal rows %+v, want the consumed corpse 3 alone", terminal)
	}
	cloneWorld(t, w)
}

// Once the first Summon's unit has died and its body finished, the second
// Summon takes it from the pool and keeps its id (R2-ENGINE-455).
func TestSecondSummonReusesTheIdOfAFinishedUnit(t *testing.T) {
	w := summonWorld(t, 0x25, 0)
	if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) || len(w.entities) != 2 || w.entities[1].ID != 2 {
		t.Fatalf("first Summon built %+v", actorKinds(w))
	}
	w.entities[1].setCurrentHealth(decayGoneHP - 100)
	w.entities[1].Decay = decayLast
	w.clearFelled(1)
	w.remove([]EntityID{2})
	if ids := terminalIDs(w); !reflect.DeepEqual(ids, []EntityID{2}) {
		t.Fatalf("pooled rows %v, want [2]", ids)
	}
	if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) {
		t.Fatal("second Summon did not land")
	}
	if len(w.entities) != 2 || w.entities[1].ID != 2 || !w.entities[1].Alive() || w.entities[1].Owner != w.entities[0].Owner {
		t.Fatalf("second Summon built %+v, want id 2 reused", actorKinds(w))
	}
	if ids := terminalIDs(w); len(ids) != 0 {
		t.Fatalf("the reused unit kept a terminal row: %v", ids)
	}
	if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) || len(w.entities) != 3 || w.entities[2].ID != 3 {
		t.Fatalf("third Summon with an empty pool built %+v, want id 3", actorKinds(w))
	}
	cold := cloneWorld(t, w)
	Step(w, nil)
	Step(cold, nil)
	if w.Hash() != cold.Hash() {
		t.Fatal("next tick differs after a cold load")
	}
}

// The pool walks the global list from its head: the lowest id first.
func TestSecondSummonTakesThePoolInListOrder(t *testing.T) {
	w := summonWorld(t, 0x25, 0, goneBody(2, 3, 3), goneBody(3, 4, 3))
	w.remove([]EntityID{3, 2})
	for _, want := range []EntityID{2, 3, 4} {
		if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) || indexOfEntity(w.entities, want) < 0 {
			t.Fatalf("Summon did not build id %d: %+v", want, actorKinds(w))
		}
	}
	if ids := terminalIDs(w); len(ids) != 0 {
		t.Fatalf("pool rows %v left", ids)
	}
	cloneWorld(t, w)
}

// A Summon whose placement fails has taken its pooled unit, which leaves
// play (R2-ENGINE-791).
func TestSecondSummonFailedPlacementLeavesTheTakenUnitGone(t *testing.T) {
	var ring []Entity
	id := EntityID(2)
	for y := int32(13); y <= 19; y++ {
		for x := int32(13); x <= 19; x++ {
			if x != 16 || y != 16 {
				ring = append(ring, spEnt(id, x, y))
				id++
			}
		}
	}
	w := summonWorld(t, 0x25, 0, append(ring, goneBody(id, 3, 3))...)
	w.remove([]EntityID{id})
	if w.ordinaryEffectPayload(0, 0, w.spells[0], 50) {
		t.Fatal("Summon landed with no free cell")
	}
	if len(w.entities) != len(ring)+1 {
		t.Fatalf("aborted Summon left %d actors", len(w.entities))
	}
	if ids := terminalIDs(w); len(ids) != 0 {
		t.Fatalf("the taken unit kept its terminal row: %v", ids)
	}
	cloneWorld(t, w)
}

// The first game ranks no body and pools no unit: teardown, Control Spirit
// and a cheat summon keep the recorded World hash.
func TestFirstGameTeardownAndRaiseKeepTheirRecordedWorld(t *testing.T) {
	w := csItemWorld(t, DomainGround)
	a, b := spEnt(4, 5, 5), spEnt(5, 5, 5)
	a.HP, a.Decay, a.Dwell = -15, DecayFallen, 4
	b.HP, b.Decay, b.Dwell = -15, DecayFallen, 2
	w2, err := NewSummoningWorld(30, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, append(w.Entities(), a, b), nil, Relations{}, nil, nil, []SpellRule{csRule()}, hlGhostTemplate())
	if err != nil {
		t.Fatal(err)
	}
	w2.entities[2].HP, w2.entities[2].Decay = -10, DecayBones
	for range 6 {
		Step(w2, nil)
	}
	if !w2.ordinaryEffectPayload(0, 2, w2.spells[0], 17) {
		t.Fatal("first-game raise refused")
	}
	summoned := spEnt(0, 0, 0)
	summoned.Owner, summoned.TokenSize = 7, 1
	if _, err := w2.CheatSummon(summoned, nil, [EquipSlots]ItemInstance{}, 8, 8); err != nil {
		t.Fatalf("cheat summon: %v", err)
	}
	t.Logf("first-game teardown and raise hash=%016x", w2.Hash())
	if w2.Hash() != firstGameTeardownHash {
		t.Fatal("first-game World hash changed")
	}
}

// firstGameTeardownHash is that World's hash before the dead pool existed.
const firstGameTeardownHash = 0xb0d7af9cfe2af143
