package sim

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"
)

// joinDeadList ranks entity i's body on the dead list when its teardown
// comes: the reap appends the torn-down actor to the list (TRIG-REAP-017),
// so the body takes the rank after every rank the World holds. A profile
// without the list ranks nothing.
func (w *World) joinDeadList(i int) {
	if !w.game.deadList {
		return
	}
	rank := uint32(0)
	for _, e := range w.entities {
		rank = max(rank, e.deadRank)
	}
	w.entities[i].deadRank = rank + 1
}

// corpseList is the indices of the World's torn-down bodies in dead-list
// order, head first: unranked bodies, then ranked ones by rank, equal ranks
// in id order. Animate Dead walks it (R2-ENGINE-780).
func (w *World) corpseList() []int {
	var out []int
	for i := range w.entities {
		if terminalRegistryActor(&w.entities[i]) || w.entities[i].deadRank != 0 {
			out = append(out, i)
		}
	}
	slices.SortStableFunc(out, func(a, b int) int { return cmp.Compare(w.entities[a].deadRank, w.entities[b].deadRank) })
	return out
}

// pooledDead is the first unit of kind 5 or more on the global unit list
// (R2-ENGINE-455, R2-ENGINE-781): a retained terminal row whose body
// finished, lowest id first, as ids are minted in construction order and a
// reused unit keeps its id. A mover that left no body holds no unit.
func (w *World) pooledDead() (EntityID, bool) {
	id, ok := EntityID(0), false
	take := func(c EntityID) {
		if !ok || c < id {
			id, ok = c, true
		}
	}
	for _, r := range w.currentTerminalActors {
		if !r.LeavesNoBody() {
			take(r.ID)
		}
	}
	for _, r := range w.originalDead {
		if r.terminal.Stage == consumedCorpseStage && indexOfEntity(w.entities, r.ID) < 0 {
			take(r.ID)
		}
	}
	return id, ok
}

// takePooledDead takes unit id off the dead list: its terminal row, the
// contents its body held and its retained basis go, as the arm unlinks the
// unit from the corpse list and frees its holdings (R2-ENGINE-781).
func (w *World) takePooledDead(id EntityID) {
	other := func(v EntityID) bool { return v == id }
	w.currentTerminalActors = slices.DeleteFunc(slices.Clone(w.currentTerminalActors), func(r CurrentTerminalActor) bool { return other(r.ID) })
	w.originalDead = slices.DeleteFunc(slices.Clone(w.originalDead), func(r originalDeadRecord) bool { return other(r.ID) })
	w.passiveBodies = slices.DeleteFunc(slices.Clone(w.passiveBodies), func(r PassiveBody) bool { return other(r.ID) })
	w.removedNativeBases = slices.DeleteFunc(slices.Clone(w.removedNativeBases), func(r NativeActorBasisRecord) bool { return other(r.ID) })
}

// poolCreation is p with the pooled unit taken for a profile that reuses
// one: the arm takes it before the build, so a failed placement leaves it
// taken (R2-ENGINE-455, R2-ENGINE-791). An empty pool mints the next id.
func (w *World) poolCreation(p actorCreationProfile) actorCreationProfile {
	if !p.pooled {
		return p
	}
	if id, ok := w.pooledDead(); ok {
		w.takePooledDead(id)
		p.reuse = id
	}
	return p
}

// DeadList is the ids of the dead list's bodies, head first: the order the
// SAV writer gives the dead-actor roots.
func (w *World) DeadList() []EntityID {
	var ids []EntityID
	for _, i := range w.corpseList() {
		ids = append(ids, w.entities[i].ID)
	}
	return ids
}

// LoadDeadList joins the listed bodies to the dead list in the order given,
// as a SAV's dead-actor roots hold them; an id that is no body is skipped.
func (w *World) LoadDeadList(ids []EntityID) {
	for _, id := range ids {
		if i := indexOfEntity(w.entities, id); i >= 0 && terminalRegistryActor(&w.entities[i]) && w.entities[i].deadRank == 0 {
			w.joinDeadList(i)
		}
	}
}

// deadRankFormVersion carries the dead list's order when it is not id order:
// the order alone decides play, so two Worlds with one order have one form.
// A form without it walks its bodies in id order.
const deadRankFormVersion byte = 138

// deadRankIndexLen is one body of the list: its uint32 entity index.
const deadRankIndexLen = 4

// appendDeadRanks writes the list's entity indices, head first.
func (w *World) appendDeadRanks(b []byte) []byte {
	list := w.corpseList()
	if slices.IsSorted(list) {
		return b
	}
	ws := beginWrapper(b)
	for _, i := range list {
		b = binary.LittleEndian.AppendUint32(b, uint32(i))
	}
	return endWrapper(b, ws, deadRankFormVersion)
}

func (w *World) unmarshalDeadRanks(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed dead rank section") }
	env, ok := readWrapper(data, deadRankFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%deadRankIndexLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	for at := 0; at < len(s); at += deadRankIndexLen {
		index := binary.LittleEndian.Uint32(s[at:])
		if uint64(index) >= uint64(len(entities)) || entities[index].Alive() || entities[index].Dwell != 0 || entities[index].deadRank != 0 {
			return fail()
		}
		entities[index].deadRank = uint32(at/deadRankIndexLen) + 1
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
