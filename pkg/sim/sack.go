package sim

import (
	"fmt"
	"sort"
)

// Sack is one ground sack: a cell, a purse and the item codes it carries.
//
// It is not an entity: it does not tick, holds no timer, takes no order and
// is in no actor list, so it is a value the world holds in a list of its own
// rather than a shape that would hand every entity invariant — health,
// movement domain, facing, decay — to something none of them describes.
//
// Gold and Items are CARRIED and interpreted nowhere in this tier: an item
// code's class and index are the format leaf's business (alm.ItemClass,
// alm.ItemIndex), and this package reads neither.
type Sack struct {
	ObjectID SavedObjectID
	X, Y     int32
	Gold     uint32
	// Items is the code projection retained for callers that only draw item
	// classes. ItemInstances is the canonical payload; constructors accept
	// either, but reject conflicting double representations.
	Items         []uint16
	ItemInstances []ItemInstance
}

// SackTokenValue is Sack's own `+0x1c` rule (ITEM-SACK-010): gold plus the
// sum of each contained item record's own `+0x1c`, its unit price
// (SHOP-PRICE-011; SHOP-SELL-010 multiplies a count only when selling). items
// holds one entry per written record. The sum is the field's own dword
// arithmetic, so an item priced -1 subtracts one and the result wraps; no
// claim establishes a clamp.
func SackTokenValue(gold uint32, items []ItemInstance) uint32 {
	value := gold
	for _, item := range items {
		value += uint32(item.Price)
	}
	return value
}

// sackRecordSum is gold plus the unit price of each record a sack's
// container writes: a stored stack contributes its unit price once, whatever
// its count, and each unbound instance is its own record. SackTokenValue's
// dword is its low 32 bits.
func (w *World) sackRecordSum(s Sack) int64 {
	sum := func(items []ItemInstance) int64 {
		value := int64(s.Gold)
		for _, item := range items {
			value += int64(item.Price)
		}
		return value
	}
	c := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID})
	if s.ObjectID == 0 || c == nil {
		return sum(s.ItemInstances)
	}
	records := make([]ItemInstance, 0, len(c.Items))
	next := 0
	for _, id := range c.Items {
		if id == 0 {
			if next < len(s.ItemInstances) {
				records = append(records, s.ItemInstances[next])
			}
			next++
			continue
		}
		row := w.savedObjects.item(id)
		if row == nil {
			return sum(s.ItemInstances)
		}
		records = append(records, row.Value.Instance())
		next += int(row.Value.Count)
	}
	return sum(records)
}

func (w *World) sackRecordValue(s Sack) uint32 { return uint32(w.sackRecordSum(s)) }

// SackValue is the value of the sack at a cell, gold plus its records' unit
// prices, the input of the sack's frame (ITEM-137).
func (w *World) SackValue(x, y int32) (int64, bool) {
	for _, s := range w.sacks {
		if s.X == x && s.Y == y {
			return w.sackRecordSum(s), true
		}
	}
	return 0, false
}

// constructedSackRuntimeID is the id a Sack this session creates draws from
// the runtime-id allocator (ITEM-SACK-010): the lowest id no record this
// world knows holds, with 0 never allocated (MOVE-ID-016). An item or effect
// record can carry a nonzero id too, and whether LOAD re-marks it is not
// established, so those ids are reserved as well. Placeable ids are 16-bit on
// the wire, so a wider id also reserves its low half.
func (w *World) constructedSackRuntimeID() uint32 {
	used := map[uint32]bool{0: true}
	reserve := func(id uint32) {
		if id != 0 {
			used[id], used[uint32(uint16(id))] = true, true
		}
	}
	if w.savedObjects != nil {
		for _, s := range w.savedObjects.Sacks {
			reserve(s.Token.RuntimeID)
		}
		for _, item := range w.savedObjects.Items {
			reserve(item.Token.RuntimeID)
		}
		for _, effect := range w.savedObjects.Effects {
			reserve(effect.Token.RuntimeID)
		}
	}
	for _, e := range w.entities {
		reserve(e.SourceBinding.RuntimeID)
	}
	for _, r := range w.originalDead {
		reserve(r.Source.State.RuntimeID)
		reserve(r.terminal.RuntimeID)
	}
	for _, s := range w.savedStructures {
		reserve(s.RuntimeID)
	}
	id := uint32(1)
	for used[id] {
		id++
	}
	return id
}

func sackItems(s Sack) ([]ItemInstance, error) {
	if len(s.Items) != 0 && len(s.ItemInstances) != 0 {
		if len(s.Items) != len(s.ItemInstances) {
			return nil, fmt.Errorf("sack at (%d,%d) has conflicting item representations", s.X, s.Y)
		}
		for i := range s.Items {
			if s.Items[i] != s.ItemInstances[i].Code {
				return nil, fmt.Errorf("sack at (%d,%d) has conflicting item representations", s.X, s.Y)
			}
		}
	}
	if len(s.ItemInstances) != 0 {
		return cloneItems(s.ItemInstances), nil
	}
	return rawPlainItems(s.Items), nil
}

func makeSack(x, y int32, gold uint32, items []ItemInstance) Sack {
	items = cloneItems(items)
	return Sack{X: x, Y: y, Gold: gold, Items: itemCodes(items), ItemInstances: items}
}

func publicSack(s Sack) Sack {
	items := cloneItems(s.ItemInstances)
	out := Sack{ObjectID: s.ObjectID, X: s.X, Y: s.Y, Gold: s.Gold, Items: itemCodes(items)}
	if itemsHaveMetadata(items) {
		out.ItemInstances = items
	}
	return out
}

// sackFault names what is wrong with s over a world of bounds b, or nil when
// it is a state this package can produce.
//
// It is the ONE per-sack predicate, called by both normaliseSacks — to
// learn which entries to refuse — and the decoder — to refuse the same
// shape on the way in, which is the split patrolFault and decayFault already
// model: a constructor that could build what the decoder will not read back
// would build worlds this package cannot marshal and read again.
//
// The one thing it refuses is a cell outside b: NOT clamped, NOT wrapped and
// NOT dropped, on the rule the entity's own X and Y take — nothing here is
// a state this package repairs on a caller's behalf.
func sackFault(b Bounds, s Sack) error {
	if _, in := cellIndexIn(b, s.X, s.Y); !in {
		return fmt.Errorf("sack at (%d,%d) is outside %dx%d", s.X, s.Y, b.Width, b.Height)
	}
	return nil
}

// normaliseSacks is the constructor's own act: it turns a caller's list into
// the one legal representation a world may hold, which the decoder then only
// ever has to refuse a departure from.
//
// ONE SACK PER CELL: entries naming the same cell are folded into one, in
// the order they were given — the original's own sack maker looks the cell
// up first and pours into what is already there rather than building a
// second, so this fold reproduces exactly that. The purses ADD, wrapping at
// the 32-bit width as the original's own addition does (D-8), and the item
// lists JOIN in argument order — the order the caller named the entries
// in, not any order this function chooses.
//
// THE RESULT IS SORTED ascending by (Y, X), so a world's contents depend on
// the sacks named and not on the order they were named in — and an entry
// whose cell falls outside b is REFUSED, on sackFault's own ground.
//
// It is a linear scan and not a map, on groupKeys' own ground: the counts
// this package will ever see are small — the whole shipped corpus authors
// 137 ground sacks across thirty maps — and a map here would put Go's
// randomised iteration order on a path that builds canonical state, for no
// benefit a scan does not already give at this size.
func normaliseSacks(b Bounds, in []Sack) ([]Sack, error) {
	var out []Sack
	for _, s := range in {
		if err := sackFault(b, s); err != nil {
			return nil, err
		}
		items, err := sackItems(s)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := itemWeightFault(item.Code, item.WeightPresent, item.Weight); err != nil {
				return nil, err
			}
		}
		found := false
		for i := range out {
			if out[i].X == s.X && out[i].Y == s.Y {
				if out[i].ObjectID != 0 || s.ObjectID != 0 {
					return nil, fmt.Errorf("sack at (%d,%d) has conflicting object identities", s.X, s.Y)
				}
				out[i].Gold += s.Gold // wraps at 32 bits (D-8)
				out[i].ItemInstances = append(out[i].ItemInstances, items...)
				out[i].Items = itemCodes(out[i].ItemInstances)
				found = true
				break
			}
		}
		if !found {
			entry := makeSack(s.X, s.Y, s.Gold, items)
			entry.ObjectID = s.ObjectID
			out = append(out, entry)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Y < out[j].Y || (out[i].Y == out[j].Y && out[i].X < out[j].X)
	})
	return out, nil
}

// sackAt reports whether a ground sack occupies (x, y).
//
// It is a BINARY SEARCH over the world's own ordered list: the list is
// ascending by (Y, X), which is the same key the search steps on, so no
// index, no map and no plane is needed to answer it. The original keeps a
// per-cell flag beside a hash bucket; this is a different mechanism for the
// same answer, and the difference is not observable through the sack query
// (D-2) — this build's only reader of a sack's presence, not of anything
// it carries.
func (w *World) sackAt(x, y int32) bool {
	i := sort.Search(len(w.sacks), func(i int) bool {
		s := w.sacks[i]
		return s.Y > y || (s.Y == y && s.X >= x)
	})
	return i < len(w.sacks) && w.sacks[i].X == x && w.sacks[i].Y == y
}

// pourSack merges gold and items onto the ground sack at (x, y), or plants a
// fresh one there when the cell holds none — every ground-placing caller's
// shared primitive (the death drop, dropAll, and the player's own ground
// drop; see the doc below for all three).
//
// IT IS sackAt's OWN SEARCH, restated here rather than reused because this
// call needs the INDEX to either merge at or insert at, exactly as
// TakeSack (carry.go) restates it for the index it needs to remove.
// MERGING APPENDS items AT THE FOUND SACK'S TAIL and adds gold with uint32
// wrapping, the same one-sack-per-cell rule normaliseSacks applies at load.
// PLANTING A FRESH
// SACK INSERTS AT THE SEARCH'S OWN INDEX, which preserves the (Y, X) order
// by construction: nothing here re-sorts and nothing re-scans, and a
// sort.Slice in its place would be a second place deciding what the
// list's order is, free to disagree with this one.
//
// ITEMS IS DEEP-COPIED at the boundary. Every caller supplies complete item
// instances now, and their Effects slices are mutable values; the sack must
// not alias the corpse, container or caller after ownership moves.
func (w *World) pourSack(x, y int32, gold uint32, items []ItemInstance) {
	j := sort.Search(len(w.sacks), func(j int) bool {
		s := w.sacks[j]
		return s.Y > y || (s.Y == y && s.X >= x)
	})
	if j < len(w.sacks) && w.sacks[j].X == x && w.sacks[j].Y == y {
		w.sacks[j].Gold += gold
		w.sacks[j].ItemInstances = append(w.sacks[j].ItemInstances, cloneItems(items)...)
		w.sacks[j].Items = itemCodes(w.sacks[j].ItemInstances)
		w.noteSavedSackAddition(w.sacks[j], len(items) != 0)
		return
	}
	w.sacks = append(w.sacks, Sack{})
	copy(w.sacks[j+1:], w.sacks[j:])
	w.sacks[j] = makeSack(x, y, gold, items)
}

// Sacks returns the world's ground sacks, in the world's own ascending
// order, as a fresh copy with each item list copied too: mutating the
// result, or any item list in it, cannot reach the world it came from.
func (w *World) Sacks() []Sack {
	out := make([]Sack, len(w.sacks))
	for i, s := range w.sacks {
		out[i] = publicSack(s)
	}
	return out
}

// ReplaceGroundSacks installs an authoritative saved ground population. Empty
// input clears fresh-map loot. Validation, same-cell merging and deep copying
// finish before the world changes, so a rejected import cannot partly replace it.
func (w *World) ReplaceGroundSacks(sacks []Sack) error {
	if w == nil {
		return fmt.Errorf("replace ground sacks: no world")
	}
	if w.savedObjects != nil {
		return fmt.Errorf("replace ground sacks cannot replace bound object ownership")
	}
	next, err := normaliseSacks(w.bounds, sacks)
	if err != nil {
		return err
	}
	w.sacks = next
	return nil
}
