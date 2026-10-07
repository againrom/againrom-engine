package sim

// The server type id band, and what crosses a mission boundary.
//
// EVERY ACTOR CARRIES A SERVER TYPE ID (Entity.TypeID). Which arm built it
// decides the value: a zero-mode Humans placement preserves its table TypeID;
// the NPC arm overwrites it into the band only for an exact Hero flag; a minted
// party member is also in the band. A creature keeps its class row's id, and an
// actor built from neither carries zero.
//
// THE BAND IS THE WHOLE OF THE BOUNDARY RULE. At the end of a mission the
// server keeps a surviving participant's actor if and only if its type id is
// inside the band, and destroys it otherwise. There is no flag, no group and no
// script difference between an actor that crosses and one that does not. A
// hand-over changes ownership, not construction mode: mission 20's Sarindar and
// three guards remain low-type and are culled, while mission 40's Hero-flagged
// Brian is retained.
//
// IT IS NOT A ROLE. A map's enemy people are inside the band too. What the band
// names is which creation arm ran, and the boundary test is applied per actor
// per owner rather than per side.
const (
	// PersistLow is the band's inclusive lower bound.
	PersistLow int32 = 0x21
	// PersistHigh is the band's EXCLUSIVE upper bound, and it is strictly
	// below the death-gold gate's own 0x40 (deathGold, step.go): no value
	// this band admits can reach that roll, so giving a person a band type
	// id cannot make him drop gold he did not drop before.
	PersistHigh int32 = 0x40

	// HumanTypeID is the male fighter base of the player-character type range.
	HumanTypeID int32 = PersistLow
)

// HeroTypeID preserves both axes of the player-character constructor type.
func HeroTypeID(mage, female bool) int32 {
	typeID := HumanTypeID
	if female {
		typeID++
	}
	if mage {
		typeID += 2
	}
	return typeID
}

// InPersistBand reports whether typeID is inside the band — whether an actor
// carrying it survives a mission boundary under its owner.
func InPersistBand(typeID int32) bool {
	return typeID >= PersistLow && typeID < PersistHigh
}

// BoundarySurvivors is which of owner's actors cross into the next mission:
// alive, owned by owner, and carrying a band type id. An actor failing any one
// of the three does not cross.
//
// IT WALKS EVERY ACTOR THE WORLD HOLDS and not only the ones the mission was
// started with, which is the whole point of asking the world rather than the
// party: an actor that arrived by a script hand-over is judged by exactly the
// same three tests as one that walked in, and there is nowhere for a rule to
// apply to one and not the other.
//
// THE ANSWER IS IN ASCENDING ID ORDER, so it does not depend on the order
// this world happens to store entities in. Ids are assigned ascending at
// load and at the party mint, so the scan below is already in that order for
// every world this tree builds; the doc states the guarantee the caller may
// rely on rather than the accident that supplies it today.
func (w *World) BoundarySurvivors(owner uint32) []EntityID {
	if w == nil {
		return nil
	}
	var out []EntityID
	for i := range w.entities {
		e := w.entities[i]
		if e.Owner != owner || !e.Alive() || !InPersistBand(e.TypeID) {
			continue
		}
		out = insertAscending(out, e.ID)
	}
	return out
}

// insertAscending puts id in its place in an already ascending slice. An
// insertion sort rather than a sort call: the input is already ascending for
// every world this tree builds, so this walks the tail once and moves nothing,
// and pkg/sim stays free of the comparator plumbing a sort would need.
func insertAscending(ids []EntityID, id EntityID) []EntityID {
	ids = append(ids, id)
	for i := len(ids) - 1; i > 0 && ids[i-1] > ids[i]; i-- {
		ids[i-1], ids[i] = ids[i], ids[i-1]
	}
	return ids
}
