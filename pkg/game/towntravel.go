package game

import (
	"fmt"
	"maps"
	"slices"

	"againrom/pkg/town"
)

// The first game's party belongs to one chapter town at a time: the town it
// travelled to last, and the town it left for a mission until it travels
// again. Each chapter town of the campaign's catalog keeps its own state,
// built by newTownState, the party's in Town's own fields and the rest in
// others.

// townEntry is the catalog entry of the party's town.
func (t *Town) townEntry() town.Entry {
	if t == nil {
		return chapterTownEntry
	}
	id := t.townID()
	for _, e := range t.camp.Towns.chapterTowns() {
		if e.ID == id {
			return e
		}
	}
	return chapterTownEntry
}

// manyTowns reports whether the campaign has more than one chapter town;
// with one, every offer is the town's and no rule below filters any.
func (t *Town) manyTowns() bool { return t != nil && len(t.others) > 0 }

// townOpen reports whether entry e's town is open to travel: from the start,
// or once the mission it opens after is done.
func (t *Town) townOpen(e town.Entry) bool { return e.OpensAfter == 0 || t.Done(e.OpensAfter) }

// travelTowns are the open chapter towns the party can travel to, in catalog
// order: every one but its own.
func (t *Town) travelTowns() []town.Entry {
	if !t.manyTowns() || !t.Open() {
		return nil
	}
	var out []town.Entry
	for _, e := range t.camp.Towns.chapterTowns() {
		if e.ID != t.townID() && t.townOpen(e) {
			out = append(out, e)
		}
	}
	return out
}

// travelTo makes chapter town id the party's town: the party's town state
// goes to others and id's comes out. It refuses its own town, a town the
// campaign does not hold and a town not yet open.
func (t *Town) travelTo(id int) bool {
	if t == nil || id == t.townID() {
		return false
	}
	next, ok := t.others[id]
	if !ok || !slices.ContainsFunc(t.travelTowns(), func(e town.Entry) bool { return e.ID == id }) {
		return false
	}
	others := maps.Clone(t.others)
	delete(others, id)
	others[t.townID()] = t.townState
	t.townState, t.others = next, others
	return true
}

// checkTowns refuses a catalog whose chapter town opens after a mission the
// campaign does not hold: that town would never open.
func (c Campaign) checkTowns() error {
	for _, e := range c.Towns.chapterTowns() {
		if _, ok := c.Chapters[e.OpensAfter]; e.OpensAfter != 0 && !ok {
			return fmt.Errorf("town catalog: town %d opens after mission %d, which the campaign does not hold", e.ID, e.OpensAfter)
		}
	}
	return nil
}

// offeredHere reports whether the party's town offers mission m: a town
// whose entry lists missions offers those, and a town that lists none offers
// every mission no town lists, the tavern's empty row among them.
func (t *Town) offeredHere(m int) bool {
	if !t.manyTowns() {
		return true
	}
	if own := t.townEntry().Missions; len(own) > 0 {
		return slices.Contains(own, m)
	}
	for _, e := range t.camp.Towns.chapterTowns() {
		if slices.Contains(e.Missions, m) {
			return false
		}
	}
	return true
}

// currentTowns is the first game's town record in the SAV's session leaf:
// the party's town and the state of each other chapter town that differs
// from its fresh build, in catalog order. The original format holds one
// town (DIV-3853); the party's town's own state stays in the original
// campaign record and session fields.
type currentTowns struct {
	Town   int
	Others []currentTownState `json:",omitempty"`
}

// currentTownState is one other town's state. A hired flag names the town
// that hired the squad the party carries; the campaign's mercenary unlocks
// are one list, in the original campaign record.
type currentTownState struct {
	ID       int
	Taken    []SnapshotOffer `json:",omitempty"`
	Pool     [15]int
	Capacity [15]int
	Hired    [15]bool
}

func (r *currentTowns) clone() *currentTowns {
	if r == nil {
		return nil
	}
	out := &currentTowns{Town: r.Town}
	for _, o := range r.Others {
		o.Taken = slices.Clone(o.Taken)
		out.Others = append(out.Others, o)
	}
	return out
}

// townsRecord is the town record a SAVE writes: nil while the party
// belongs to town 1 and every other town keeps its fresh state, so a game of
// one town writes none.
func (t *Town) townsRecord() *currentTowns {
	if t == nil {
		return nil
	}
	r := &currentTowns{Town: t.townID()}
	for _, e := range t.camp.Towns.chapterTowns() {
		s, ok := t.others[e.ID]
		if !ok || sameTownState(s, newTownState(e, t.camp)) {
			continue
		}
		o := currentTownState{ID: e.ID}
		for ref := range s.taken {
			o.Taken = append(o.Taken, SnapshotOffer{Chapter: ref.chapter, Building: int(ref.building), Index: ref.index})
		}
		sortOffers(o.Taken)
		for i := 1; i < 16; i++ {
			o.Pool[i-1], o.Capacity[i-1] = s.mercPool[i], s.mercCapacity[i]
			o.Hired[i-1] = s.mercHired[i]
		}
		r.Others = append(r.Others, o)
	}
	if r.Town == chapterTown && len(r.Others) == 0 {
		return nil
	}
	return r
}

func sameTownState(a, b townState) bool {
	return a.id == b.id && maps.Equal(a.taken, b.taken) && a.mercPool == b.mercPool &&
		a.mercCapacity == b.mercCapacity && a.mercHired == b.mercHired
}

// restoreTowns applies a LOAD's town record: the party's town state, which
// the original record and session fields filled, becomes town r.Town's, and
// each other town takes its recorded state or a fresh one. No record is
// town 1 with every other town fresh. A town the campaign does not hold, a
// record that names one town twice, a pool below zero or above its capacity,
// a hire of a type the town holds no squad of, and one type hired in two
// towns are refused.
func (t *Town) restoreTowns(r *currentTowns) error {
	if r == nil {
		return nil
	}
	entries := t.camp.Towns.chapterTowns()
	entry := func(id int) (town.Entry, bool) {
		i := slices.IndexFunc(entries, func(e town.Entry) bool { return e.ID == id })
		if i < 0 {
			return town.Entry{}, false
		}
		return entries[i], true
	}
	if _, ok := entry(r.Town); !ok {
		return fmt.Errorf("save names town %d, which the campaign does not hold", r.Town)
	}
	others := map[int]townState{}
	for _, e := range entries {
		if e.ID != r.Town {
			others[e.ID] = newTownState(e, t.camp)
		}
	}
	seen := map[int]bool{}
	var hiredBy [16]int
	for _, o := range r.Others {
		e, ok := entry(o.ID)
		if !ok || o.ID == r.Town || seen[o.ID] {
			return fmt.Errorf("save holds a state of town %d it cannot restore", o.ID)
		}
		seen[o.ID] = true
		s := newTownState(e, t.camp)
		for _, ref := range o.Taken {
			s.taken[offerRef{chapter: ref.Chapter, building: TownBuilding(ref.Building), index: ref.Index}] = true
		}
		for i := 1; i < 16; i++ {
			pool, capacity, hired := o.Pool[i-1], o.Capacity[i-1], o.Hired[i-1]
			switch {
			case pool < 0 || pool > capacity:
				return fmt.Errorf("save holds town %d's mercenary type %d at %d of %d", o.ID, i, pool, capacity)
			case hired && capacity == 0:
				return fmt.Errorf("save holds town %d's hire of mercenary type %d, which it has no squad of", o.ID, i)
			case hired && (t.mercHired[i] || hiredBy[i] != 0):
				return fmt.Errorf("save holds mercenary type %d hired in two towns", i)
			}
			if hired {
				hiredBy[i] = o.ID
			}
			s.mercPool[i], s.mercCapacity[i], s.mercHired[i] = pool, capacity, hired
		}
		others[o.ID] = s
	}
	t.townState.id = r.Town
	if len(others) == 0 {
		others = nil
	}
	t.others = others
	return nil
}
