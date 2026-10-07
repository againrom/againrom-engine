package sim

import (
	"fmt"
	"sort"
)

// foldContainer is THE ONE PLACE a container is normalised: equal codes
// collapse into the FIRST place the code occupies, their counts summing, and
// an element whose count is 0 is dropped rather than kept as a place holding
// nothing.
//
// Every act that puts an item into a container ends here — construction from
// an authored loadout, a decode of the byte form, a take, a give-all, and an
// item displaced out of an equipment slot — so there is no act that can
// leave two elements naming one code behind. It is chosen over a merging
// add(code, count) called at each site (D-2) because equip's write-back puts
// a code back at a NAMED INDEX rather than appending, so an add-only helper
// would leave that one path to merge by hand, and that is exactly the path
// that would rot.
//
// FOLDING IS IDEMPOTENT: a container already holding at most one element per
// code is returned with every element, count and place exactly where it was,
// because the inner scan then never finds a match.
//
// IT IS A LINEAR SCAN AND NOT A MAP, on normaliseSacks' own ground and
// normaliseHoldings' own rule: no map belongs on any path that builds a
// world, because Go's iteration order is randomised and a container's order
// is canonical state that reaches the byte form and the digest. That makes
// it O(n²) over ELEMENTS in the worst case, and n is one actor's container.
//
// COUNTS SUM WITH NO LIMIT TEST OF ANY KIND. There is no capacity and no
// slot count in this build, and the 32-bit width is itself the argument that
// no container this package can construct can reach the count's own top: the
// flat expansion is written with a 32-bit length, so it can hold no more
// units than a 32-bit count can carry.
//
// IT FOLDS BY COMPLETE INSTANCE IDENTITY. `ITEM-STACK-003` makes stackable a
// virtual whose base body accepts a Potion or an item with no effect list.
// ItemEqual supplies the corresponding instance predicate here: ordinary
// items compare code, kind and ordered effects, while Potion also includes
// stored value under the owner's retention rule. Enchanted and plain copies
// of one code therefore remain distinct container elements.
//
// THE RESULT IS A FRESH SLICE, never the argument's own backing array, so a
// caller that folds one container cannot write through into another.
func foldContainer(c []ItemStack) []ItemStack {
	var out []ItemStack
	for _, st := range c {
		if st.Count == 0 {
			continue
		}
		merged := false
		for k := range out {
			// A value fold cannot retire an object or preserve its child
			// edges. Bound objects merge only through SavedObjects.Insert.
			if st.ObjectID != 0 || out[k].ObjectID != 0 {
				continue
			}
			// ItemEqual deliberately excludes transaction price for every
			// non-Potion class. A container cell cannot: one Count has only
			// one Price to preserve, so equal objects bought for different
			// amounts remain separate cells. Kind is retained for the same
			// reason (DIV-747): one cell cannot preserve two concrete kinds.
			if CanMergeItemValues(out[k].Instance(), st.Instance()) {
				out[k].Count += st.Count
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, st.Clone())
		}
	}
	return out
}

// appendUnits appends one element of count 1 per code in codes, at c's tail
// and in codes' own order, and DOES NOT FOLD: it is the half of "put these
// items in this container" that knows how a flat code list becomes elements,
// and the caller runs foldContainer over the result.
//
// The split exists because two of the three callers append more than one
// list before anything is folded — normaliseHoldings walks every Stock
// naming an entity, and TakeSack appends a sack onto what the taker already
// holds — and folding between the appends would merge exactly the same
// elements at exactly the same places for strictly more work. decodeCarried
// preserves its flat projection until the later item sections restore and
// validate complete instance identities and quantities.
//
// Appending AT THE TAIL is normaliseHoldings' own note restated: it is what
// the original's Add does whenever the next-insert index has reached the
// count, which for a container this package only ever grows at the tail is
// every insert. Where the code is already held, the fold then moves the
// units back into the element that holds it and the tail place disappears.
func appendItems(c []ItemStack, items []ItemInstance) []ItemStack {
	for _, item := range items {
		c = append(c, StackItem(item, 1))
	}
	return c
}

// FoldItems is the container a flat unit list becomes when its units are
// inserted in order: each joins the first element foldContainer accepts
// (ITEM-MERGE-129). The city graph, a SAV and mission entry all fold a flat
// party pack with it, so all three show the same cells.
func FoldItems(items []ItemInstance) []ItemStack {
	return foldContainer(appendItems(nil, items))
}

func appendCodes(c []ItemStack, codes []uint16) []ItemStack {
	return appendItems(c, plainItems(codes))
}

// appendUnits is the legacy spelling retained for code-only producers.
func appendUnits(c []ItemStack, codes []uint16) []ItemStack { return appendCodes(c, codes) }

// containerUnits is Σ Count over a container: how many UNITS it holds,
// which is exactly the length of its flat expansion and exactly the code
// count the carry section writes.
//
// It is stated once and read by both halves of the encoder — the pass that
// SIZES the buffer and the pass that WRITES it — because those two agreeing
// is what keeps the section it sizes the same length as the section it
// writes (D-5). len(carried[i]) is the element count and answers a different
// question; before this story the two were the same number, which is the one
// way this could go wrong unnoticed.
func containerUnits(c []ItemStack) uint32 {
	var n uint32
	for _, st := range c {
		n += st.Count
	}
	return n
}

// expandContainer is the FLAT EXPANSION of a container: one code per unit
// held, an element of count n contributing its code n times, in element
// order. It is what a container WAS before this story, and it is the single
// definition every reader that answers in codes goes through — Carried,
// Stock and the carry section's own encoder — so none of them can come to
// disagree with another about what expanding means.
//
// AN EMPTY CONTAINER EXPANDS TO nil rather than to an empty slice, which is
// what every one of those callers already handed back for an actor holding
// nothing.
func expandContainer(c []ItemStack) []uint16 {
	n := containerUnits(c)
	if n == 0 {
		return nil
	}
	out := make([]uint16, 0, n)
	for _, st := range c {
		for k := uint32(0); k < st.Count; k++ {
			out = append(out, st.Code)
		}
	}
	return out
}

// expandItems returns one deep-copied item instance per unit, in canonical
// container order.
func expandItems(c []ItemStack) []ItemInstance {
	n := containerUnits(c)
	if n == 0 {
		return nil
	}
	out := make([]ItemInstance, 0, n)
	for _, st := range c {
		for k := uint32(0); k < st.Count; k++ {
			out = append(out, st.Instance())
		}
	}
	return out
}

// Stock is one actor's authored starting HOLDINGS: an entity id, the item
// codes construction puts into that entity's container, and the twelve
// equipment slots it is wearing, all before the world takes its first tick.
//
// Items remains the code-only compatibility projection established by 0138.
// ItemInstances is the canonical population used by current producers and
// transfers. Stock is both constructor input and output: either representation
// may be supplied, while a double representation is accepted only when every
// code agrees. Counts remain an internal ItemStack concern; both public lists
// are flat, one unit per entry.
//
// EQUIPPED TRAVELS WITH ITEMS RATHER THAN IN A SEVENTH CONSTRUCTOR, and that
// is the load-bearing choice of the starting-equipment hotfix. Both mission
// rebuilds in pkg/mapload already name Stock() — they had to be taught to,
// twice, after a state was added behind them (the sack list in 0103, these
// containers in 0112, each silently dropped for stories) — so a starting
// LOADOUT carried inside the record they already carry cannot become the
// third. One type says what an actor begins with; there is no second thing
// for a rebuild to forget.
type Stock struct {
	ID EntityID
	// A nonnil order supplies exact cells, including empty slots. It is
	// independent of whether the actor has retained load bookkeeping.
	OrderedStacks []ItemStack
	LoadState     *ActorLoadSnapshot
	// Items is the code-only compatibility input. ItemInstances is the
	// canonical population; a record must not populate both.
	Items         []uint16
	ItemInstances []ItemInstance
	// Equipped is what the actor is WEARING, slot 1 at index 0, the zero
	// code meaning an empty slot (equip.go's own spelling of empty). A
	// wholly zero array is an actor wearing nothing, which is what every
	// world built before this field existed held, so an omitted Equipped
	// is materialised to exactly the behaviour such a world had.
	Equipped      [EquipSlots]uint16
	EquippedItems [EquipSlots]ItemInstance
}

// carryFault names what is wrong with codes, or nil when it is a state this
// package can produce.
//
// It is the ONE predicate, called by the constructor's normaliser — to learn
// which codes to drop — and, from the byte form's own task, by the decoder —
// to refuse the same shape on the way in: sackFault's and patrolFault's own
// split, restated for the same reason. A constructor that built what the
// decoder refuses would build worlds this package cannot marshal and read
// again.
//
// The one thing it refuses is a code of ZERO: class is bits 8..11 and a
// class of 0 resolves to a null the map-load caller SKIPS (ITEM-CODE-029),
// so zero is never an item and never a claim a decoded record can make. THE
// CONSTRUCTOR DROPS a zero rather than refusing it — every zero a caller
// hands this package is the map's own null travelling with the rest of a
// record's codes, not a mistake worth an error a reader could learn nothing
// from. THE DECODER REFUSES one instead, on reachFault's own ground: a
// decoded record is a claim about a saved actor, and zero is not a claim
// this package can have written.
func carryFault(items []ItemInstance) error {
	for i, item := range items {
		if item.Code == 0 {
			return fmt.Errorf("carried item %d has code zero, which is not an item", i)
		}
	}
	return nil
}

// normaliseHoldings is the constructor's own act: it turns a caller's list
// of Stock records into the one legal representation a world may hold —
// one container slice AND one equipment record per entity, in entity order,
// exactly the shape routes already has — which a later task's decoder then
// only ever has to refuse a departure from.
//
// IT ANSWERS BOTH HALVES IN ONE WALK because a Stock states both, and two
// functions over one record is exactly how a world could come to hold a
// container from a record whose loadout it dropped.
//
// A Stock naming an id ents does not hold is REFUSED, on sackFault's own
// ground for a cell outside bounds: it is a caller's claim about an entity,
// not a value to fold away. ents must already be sorted by ascending id —
// newWorld's own cp, after its sort — because the lookup is indexOfEntity's
// binary search and not a map, on that search's own ground: no map belongs
// on any path that builds a world.
//
// TWO STOCK NAMING THE SAME ID APPEND, in argument order — the order the
// caller named the entries in, not any order this function chooses,
// normaliseSacks' own rule for two entries landing on one cell restated for
// an id instead of a cell. Within one Stock the codes keep their own order
// too: appended at the tail is what the original's Add does whenever the
// next-insert index has reached the count, which for a container this
// package only ever builds at the tail is every insert.
//
// ZEROES ARE DROPPED wherever carryFault finds one, and every surviving
// slice is a fresh one built by this function's own appends, so mutating a
// caller's Items after this returns cannot reach the world it built.
//
// A ZERO EQUIPMENT SLOT IS NOT DROPPED, IT IS THE EMPTY SLOT — carryFault is
// asked about the container alone. The container is a list, so a zero in it
// is a hole with no meaning; the equipment record is twelve fixed places, so
// a zero in it is the statement that the place is free, and it is the value
// every world built before this field existed holds at all twelve.
//
// TWO STOCK NAMING ONE ID: a non-zero slot of the later record OVERWRITES,
// slot by slot, in the same argument order the containers append in. A zero
// slot of the later record leaves what the earlier one wore, so a record
// naming a container alone never strips a loadout an earlier record put on.
//
// A CONTAINER IS FOLDED, AND IT IS FOLDED ONCE, AT THE END: a Stock states a
// flat list of codes, every code appends one unit at the tail, and
// foldContainer then collapses equal codes into the first place each one
// occupies with their counts summed. Three of one code authored on one actor
// is ONE element at count 3 and not three places (AC-1), and the element
// order is the order the codes were FIRST SEEN — across every Stock naming
// the entity, which is why the fold runs after the walk over `in` rather
// than inside it: folding per record would still be correct, and would
// merely do the same work more often.
func normaliseHoldings(ents []Entity, in []Stock) ([][]ItemStack, [][EquipSlots]ItemInstance, error) {
	out := make([][]ItemStack, len(ents))
	worn := make([][EquipSlots]ItemInstance, len(ents))
	ordered := make([]bool, len(ents))
	for _, st := range in {
		i := indexOfEntity(ents, st.ID)
		if i < 0 {
			return nil, nil, fmt.Errorf("sim: stock names entity %d, which this world does not hold", st.ID)
		}
		items := cloneItems(st.ItemInstances)
		for _, item := range items {
			if err := itemWeightFault(item.Code, item.WeightPresent, item.Weight); err != nil {
				return nil, nil, err
			}
		}
		if len(items) != 0 && len(st.Items) != 0 {
			if len(items) != len(st.Items) {
				return nil, nil, fmt.Errorf("sim: stock for entity %d has conflicting item representations", st.ID)
			}
			for k := range items {
				if items[k].Code != st.Items[k] {
					return nil, nil, fmt.Errorf("sim: stock for entity %d has conflicting item representations", st.ID)
				}
			}
		} else if len(items) == 0 {
			items = plainItems(st.Items)
		}
		if carryFault(items) != nil {
			kept := make([]ItemInstance, 0, len(items))
			for _, item := range items {
				if item.Code != 0 {
					kept = append(kept, item.Clone())
				}
			}
			items = kept
		}
		if st.OrderedStacks != nil {
			if err := orderedStockFault(st.OrderedStacks, items); err != nil {
				return nil, nil, fmt.Errorf("sim: entity %d: %w", st.ID, err)
			}
			ordered[i] = true
			out[i] = append(out[i], cloneStacks(st.OrderedStacks)...)
		} else {
			out[i] = appendItems(out[i], items)
		}
		wornItems := cloneEquipment(st.EquippedItems)
		for _, item := range wornItems {
			if err := itemWeightFault(item.Code, item.WeightPresent, item.Weight); err != nil {
				return nil, nil, err
			}
		}
		if !equipmentItemsEmpty(wornItems) && st.Equipped != ([EquipSlots]uint16{}) {
			if equipmentCodes(wornItems) != st.Equipped {
				return nil, nil, fmt.Errorf("sim: stock for entity %d has conflicting equipment representations", st.ID)
			}
		} else if equipmentItemsEmpty(wornItems) {
			wornItems = plainEquipment(st.Equipped)
		}
		for k, item := range wornItems {
			if item.Code != 0 {
				worn[i][k] = item.Clone()
			}
		}
	}
	for i := range out {
		if !ordered[i] {
			out[i] = foldContainer(out[i])
		}
	}
	for _, st := range in {
		if st.LoadState == nil {
			continue
		}
		i := indexOfEntity(ents, st.ID)
		if err := st.LoadState.Validate(); err != nil {
			return nil, nil, err
		}
		if !st.LoadState.Inventory.ContainerPresent && len(out[i]) != 0 {
			return nil, nil, fmt.Errorf("sim: absent stock container carries items")
		}
	}
	return out, worn, nil
}

// Explicit order names current cells, including null positions. The flat
// projection must account for every unit and every complete instance value.
func orderedStockFault(stacks []ItemStack, items []ItemInstance) error {
	index := 0
	for _, stack := range stacks {
		if stack.Code == 0 || stack.Count == 0 {
			if !emptyOrderedStack(stack) {
				return fmt.Errorf("invalid null ordered stock slot")
			}
			continue
		}
		if uint64(stack.Count) > uint64(len(items)-index) {
			return fmt.Errorf("ordered container quantity exceeds stock projection")
		}
		one := stack
		one.Count = 1
		for range stack.Count {
			if !StackStateEqual(one, StackItem(items[index], 1)) {
				return fmt.Errorf("ordered container disagrees with stock instance %d", index)
			}
			index++
		}
	}
	if index != len(items) {
		return fmt.Errorf("ordered container omits stock units")
	}
	return nil
}

func emptyOrderedStack(stack ItemStack) bool {
	return stack.Effects == nil && StackStateEqual(stack, ItemStack{})
}

func hasNullOrderedSlot(stacks []ItemStack) bool {
	for _, stack := range stacks {
		if emptyOrderedStack(stack) {
			return true
		}
	}
	return false
}

// The two BASE-ACTOR equipment slots, as indices into an entity's twelve
// (ITEM-EQUIP-006): slot 1 is `actor+0x74` and slot 2 is `actor+0x78`.
// ITEM-DEATH-012 first read these as the only two a body unequips on death;
// 0132 read the four instructions after the weapon arm and found a call that
// empties the other ten — the humanoid array at `actor+0x198` — into the
// same container, so a body now gives up all twelve. Slots 1 and 2 keep
// their own named constants and their own two lines at the death path rather
// than folding into its loop over the other ten, because which slot is the
// weapon is a decoded fact and an index is not — the ten at `actor+0x198`
// are a plain range and are walked as one loop, ascending.
const (
	slotWeapon = 0
	slotSecond = 1
)

// holdsSomething is "entity index i has anything at all on it" — a
// container with a code in it, or any of the twelve slots not at the zero
// code. It was Stock()'s question and Stock()'s alone until 0132: before
// that story a body could hold armour it would never drop, so the death path
// asked the narrower dropsSomething instead, answering about the container
// and the two hand slots only. Every slot is droppable now, so "does this
// body hold anything" and "would this body drop anything" are one question
// with one answer, and the death path asks this one too. dropsSomething
// retired rather than staying on as a one-line forwarder, which would read
// as a live distinction and invite a future edit to make the two diverge
// again with no fact behind it.
func (w *World) holdsSomething(i int) bool {
	if len(w.carried[i]) > 0 {
		return true
	}
	for _, item := range w.equipment[i] {
		if item.Code != 0 {
			return true
		}
	}
	return false
}

// Carried returns id's container as its FLAT EXPANSION: the item codes that
// entity holds, one per unit, in element order, as a fresh copy —
// Sacks()'s own rule and for its own reason: mutating the result cannot
// reach the world it came from. The second result is false when id names no
// entity this world holds, and the first is then nil.
//
// ITS SHAPE AND ITS ANSWER ARE UNCHANGED BY 0138. An element of count 3
// gives its code back three times, adjacent and at the element's own place,
// which is exactly the list this reader answered when a container was a code
// list — so pkg/mapload and every other caller outside this package reads
// what it always read, and D-3 exists so that none of them has to change. A
// caller that needs the counts asks CarriedStacks instead.
func (w *World) Carried(id EntityID) ([]uint16, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return nil, false
	}
	return expandContainer(w.carried[i]), true
}

// CarriedStacks returns id's container as its ELEMENTS: each an item code
// and the count of it held there, in element order, as a fresh copy —
// Carried's own rule and for its own reason: mutating the result cannot
// reach the world it came from. The second result is false when id names no
// entity this world holds, and the first is then nil.
//
// It is shaped exactly like Carried and sits beside it because the two
// answer ONE question at two grains, and the grain is the whole difference:
// Carried is what a container has always answered and is what a rebuild
// feeds back in, this is what a reader that has to DRAW or NAME the places
// needs — the inventory window's pack area, the equip command's own gate and
// the mission drive, none of which can get the element count back out of a
// flat expansion once two adjacent codes are equal.
func (w *World) CarriedStacks(id EntityID) ([]ItemStack, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return nil, false
	}
	return cloneStacks(w.carried[i]), true
}

// CarriedItems returns one complete item instance per unit. It is the
// transfer and campaign-boundary reader; Carried remains the code projection
// used by older display and diagnostic callers.
func (w *World) CarriedItems(id EntityID) ([]ItemInstance, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return nil, false
	}
	return expandItems(w.carried[i]), true
}

// Stock returns every HOLDING this world has that is not empty — a container
// with something in it, an equipment record with something worn, or both — in
// ascending entity id, as a fresh copy with each item list copied too —
// Sacks()'s own shape, and it exists for Sacks()'s own reason: A REBUILD
// NAMES WHAT IT CARRIES, so a world that has to be built again out of an
// existing one needs to read this back in ONE call rather than walking the
// entity list and asking Carried per id.
//
// It is the exact inverse of the stock argument NewStockedWorld takes, and
// that is the property worth stating: feeding Stock() back into
// NewStockedWorld over the same entities reproduces the same containers, so
// a rebuild cannot silently hold a subset of what it was rebuilt from.
// pkg/mapload's two mission-start rebuilds are the callers, and the reason
// they are is written down there.
//
// AN ACTOR HOLDING NOTHING AT ALL PRODUCES NO ENTRY, because it would say
// nothing: every entity a world holds already has a materialised list and a
// materialised equipment record, so an entry naming two empty ones is not a
// claim the constructor could act on differently from its absence. That keeps
// the result the size of what is actually held rather than the size of the
// entity list. The test is BOTH halves — a corpse whose container was poured
// but whose armour is still on it, and a fresh actor wearing a class weapon
// and carrying nothing, are each a holding, and reading the container alone
// would drop one of them at every rebuild.
//
// IT ANSWERS BOTH REPRESENTATIONS WHEN METADATA EXISTS. Items remains the
// flat code expansion for older observers; ItemInstances carries the complete
// identity required for the inverse property. Plain instances omit the latter
// because their code projection is already complete.
func (w *World) Stock() []Stock {
	var out []Stock
	for i, stacks := range w.carried {
		if !w.holdsSomething(i) && !w.entities[i].ActorLoad.Present {
			continue
		}
		items := expandItems(stacks)
		worn := cloneEquipment(w.equipment[i])
		stock := Stock{ID: w.entities[i].ID, Items: itemCodes(items), Equipped: equipmentCodes(worn)}
		stock.LoadState = w.entities[i].CurrentActorLoad()
		stock.OrderedStacks = cloneStacks(stacks)
		if itemsHaveMetadata(items) {
			stock.ItemInstances = items
		}
		if equipmentHasMetadata(worn) {
			stock.EquippedItems = worn
		}
		out = append(out, stock)
	}
	return out
}

// ReplaceStock atomically replaces one existing actor's carried and equipped
// item state with a decoded stock record. It is the narrow restore door used
// after an original-save actor has been rebound to the freshly constructed
// map entity. Other actors, sacks and the purse are untouched.
func (w *World) ReplaceStock(s Stock) bool {
	i := indexOfEntity(w.entities, s.ID)
	if i < 0 || w.actorHasSavedItems(i) {
		return false
	}
	if s.LoadState == nil && w.entities[i].ActorLoad.Source.Class != 0 {
		// A restore supplies its coupled source sheet. An ordinary command
		// must use the item virtuals, never replace a loadout by summed deltas.
		return false
	}
	held, worn, err := normaliseHoldings([]Entity{{ID: s.ID}}, []Stock{s})
	if err != nil {
		return false
	}
	if s.LoadState == nil && !w.hasActorContainer(i) && len(held[0]) != 0 {
		return false
	}
	if s.LoadState != nil {
		if s.LoadState.Validate() != nil || !s.LoadState.Inventory.ContainerPresent && len(held[0]) != 0 {
			return false
		}
	} else if !w.sourceMutationReady(i) {
		return false
	}
	before := w.beginLoadMutation(i)
	live := w
	if w.savedObjects != nil || w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		w = &n
	}
	w.carried[i] = held[0]
	w.equipment[i] = worn[0]
	if s.LoadState != nil {
		if err := w.RestoreActorLoad(s.ID, *s.LoadState); err != nil {
			return false
		}
	} else {
		if !w.finishLoadMutation(i, before) {
			return false
		}
	}
	if !w.savedMutationValid() {
		return false
	}
	if w != live {
		*live = *w
	}
	return true
}

// Purse returns roster slot slot's gold: purses[slot], indexed exactly as
// Relations is. A slot at or past relationSlots has no cell and Purse
// answers zero for it — the array's own bound, not a special case carved
// out here. UNLIKE a lookup into Relations, slot 0 is a REAL cell: see the
// purses field's own doc, on World, for why.
func (w *World) Purse(slot uint32) uint32 {
	if slot >= relationSlots {
		return 0
	}
	return w.purses[slot]
}

// SetPurse installs a participant's carried money at a mission boundary. It
// reports false and changes nothing for an out-of-range slot. Mission play
// itself credits this same field through TakeSack.
func (w *World) SetPurse(slot, gold uint32) bool {
	if slot >= relationSlots {
		return false
	}
	w.purses[slot] = gold
	return true
}

// MoveCarried moves count units of code from one actor's container to
// another's. Both actors and the complete requested count are validated before
// either container changes, so a refusal is atomic. The destination is folded
// through the same container normaliser used by pickup and construction.
func (w *World) MoveCarried(from, to EntityID, code uint16, count uint32) error {
	fi := indexOfEntity(w.entities, from)
	if fi < 0 {
		return fmt.Errorf("sim: MoveCarried names source entity %d, which this world does not hold", from)
	}
	ti := indexOfEntity(w.entities, to)
	if ti < 0 {
		return fmt.Errorf("sim: MoveCarried names destination entity %d, which this world does not hold", to)
	}
	if code == 0 || count == 0 {
		return fmt.Errorf("sim: MoveCarried needs a nonzero item code and count")
	}
	index := -1
	for i, st := range w.carried[fi] {
		if st.Code == code {
			index = i
			if st.Count < count {
				return fmt.Errorf("sim: entity %d carries %d of code %#04x, fewer than the requested %d",
					from, st.Count, code, count)
			}
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("sim: entity %d carries no code %#04x", from, code)
	}
	if fi == ti {
		return nil
	}
	if !w.hasActorContainer(ti) {
		return fmt.Errorf("sim: destination actor %d has no container", to)
	}
	fromLoad, toLoad := w.beginLoadMutation(fi), w.beginLoadMutation(ti)
	if !w.sourceMutationReady(fi) || !w.sourceMutationReady(ti) {
		return fmt.Errorf("sim: source load producer has no applicable derive rule")
	}
	live := w
	if w.savedObjects != nil || w.entities[fi].ActorLoad.Source.Class != 0 || w.entities[ti].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(fi)
		n.carried[ti] = cloneStacks(w.carried[ti])
		w = &n
	}
	if w.carried[fi][index].ObjectID == 0 && count < w.carried[fi][index].Count {
		// Unbound native stacks retain their constant-time quantity move.
		// Only an actual object split dispatches one-unit clone virtuals.
		item := w.carried[fi][index].Clone()
		item.Count = count
		w.carried[fi][index].Count -= count
		if !w.addCarried(ti, item, fi) {
			return fmt.Errorf("sim: native item transfer has no destination")
		}
	} else if count == w.carried[fi][index].Count {
		item, ok := w.takeCarriedObject(fi, index, true, ti)
		if !ok || !w.addCarried(ti, item, fi) {
			return fmt.Errorf("sim: whole item transfer is unsupported")
		}
	} else {
		for range count {
			item, ok := w.takeCarriedObject(fi, index, false, ti)
			if !ok || !w.addCarried(ti, item, fi) {
				return fmt.Errorf("sim: split item transfer is unsupported")
			}
		}
	}
	if !w.finishLoadMutation(fi, fromLoad) || !w.finishLoadMutation(ti, toLoad) {
		return fmt.Errorf("sim: transferred load cannot be derived")
	}
	if !w.savedMutationValid() {
		return fmt.Errorf("sim: item transfer ownership differs")
	}
	if w != live {
		*live = *w
	}
	return nil
}

// TakeSack is the transfer primitive: the whole of the sack at (x, y)
// becomes what id carries, in one act with no partial outcome and no caller
// behind it — no order, no walk, no command.
//
// IT MERGES RATHER THAN APPENDS: the sack's codes arrive as one unit each at
// the taker's tail, in the sack's own order, and the whole container is then
// folded — so a taker already holding a potion who picks up two more holds
// ONE element at count 3 (AC-4) rather than three places. Nothing else about
// the transfer changes: not its all-or-nothing outcome, not the gold, not
// what it refuses. A SACK ITSELF DOES NOT STACK and is not touched here: it
// is a flat code list on the ground and stays one.
//
// It finds the sack through the world's own (Y, X)-ascending list —
// sackAt's own binary search, restated here rather than reused because this
// call needs the INDEX to remove, not sackAt's bool — merges every one of
// the sack's codes into id's container IN THE SACK'S OWN ORDER, adds the
// sack's gold to purses[owner], WRAPPING AT 32 BITS exactly as
// normaliseSacks' own merge already does for two sacks landing on one cell,
// and removes the sack from the world's list entirely. The list stays
// ascending and duplicate-free by construction: removing one entry from an
// ordered, duplicate-free list leaves it ordered and duplicate-free, so
// nothing here has to re-sort or re-scan it. NOTHING ELSE MOVES — not the
// entity's position, its order, its state byte, its health or the tick —
// because the routine being reproduced touches none of them.
//
// IT APPLIES carryFault TO NOTHING IT MOVES, unlike normaliseHoldings. A
// code sitting in a world's sack list has already passed the one predicate
// that could refuse it — sackFault's own split, at construction through
// normaliseSacks or on the way in through the decoder — so a zero cannot be
// there to find: this transfer is not a second gate on a value the first
// one already settled, it is a plain copy of what the world already holds
// to be a claim it can carry.
//
// THREE THINGS REFUSE IT, and each leaves the world byte-for-byte what it
// found: an id this world does not hold, a cell holding no sack, and an
// owner at or past relationSlots. THE OWNER CHECK RUNS BEFORE ANYTHING
// MOVES, not after the codes are appended and the sack is gone: past
// relationSlots there is no purse cell to credit, and an all-or-nothing
// transfer may not leave a sack consumed with its gold unaccounted for —
// refusing early is the only way this method can keep that promise, not a
// shortcut around it.
func (w *World) TakeSack(id EntityID, x, y int32) error {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return fmt.Errorf("sim: TakeSack names entity %d, which this world does not hold", id)
	}
	if !w.hasActorContainer(i) {
		return fmt.Errorf("sim: actor %d has no container", id)
	}
	j := sort.Search(len(w.sacks), func(j int) bool {
		s := w.sacks[j]
		return s.Y > y || (s.Y == y && s.X >= x)
	})
	if j >= len(w.sacks) || w.sacks[j].X != x || w.sacks[j].Y != y {
		return fmt.Errorf("sim: TakeSack names (%d,%d), which holds no sack", x, y)
	}
	owner := w.entities[i].Owner
	if owner >= relationSlots {
		return fmt.Errorf("sim: entity %d's owner %d is at or past relationSlots (%d), which has no purse to credit",
			id, owner, relationSlots)
	}

	s := w.sacks[j]
	items, err := sackItems(s)
	if err != nil {
		return err
	}
	if !w.sourceMutationReady(i) {
		return fmt.Errorf("sim: source load producer has no applicable derive rule")
	}
	live := w
	if w.entities[i].ActorLoad.Source.Class != 0 || w.savedObjects != nil {
		n := w.sourceMutationCopy(i)
		w = &n
	}
	if s.ObjectID != 0 {
		if err := w.validateSavedObjects(); err != nil {
			return err
		}
		if err := w.savedObjects.StampPickup(s.ObjectID); err != nil {
			return err
		}
	}
	before := w.beginLoadMutation(i)
	for len(items) != 0 {
		item := items[0]
		items = items[1:]
		value := StackItem(item, 1)
		if item.ObjectID != 0 {
			row := w.savedObjects.item(item.ObjectID)
			old := row.Value.Clone()
			id, err := w.savedObjects.TakeOneAt(item.ObjectID, SavedItemLocation{Owner: SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID}, Index: 0}, old)
			if err != nil || !w.fillSplitObjectSpell(id) {
				return fmt.Errorf("sim: Sack item split: %v", err)
			}
			value = w.savedObjects.item(id).Value.Clone()
			if old.Count > 1 {
				if !w.syncSavedItemViews(item.ObjectID, old, i) {
					return fmt.Errorf("sim: shared Sack split views differ")
				}
			} else {
				w.sacks[j].ItemInstances = w.sacks[j].ItemInstances[1:]
			}
		} else if s.ObjectID != 0 {
			c := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID})
			if len(c.Items) > 0 && c.Items[0] == 0 {
				c.Items = c.Items[1:]
				w.sacks[j].ItemInstances = w.sacks[j].ItemInstances[1:]
			}
		}
		value = w.joinForm(i, value)
		merged := false
		if value.ObjectID == 0 && w.savedObjects != nil {
			if k := w.firstMergeableHeld(i, value); k >= 0 {
				if !w.mergeIntoHeld(i, k, value, 1) {
					return fmt.Errorf("sim: picked up item cannot merge into its equal element")
				}
				merged = true
			} else if constructed, ok := w.constructSavedAcquisition(value); ok {
				value = constructed
				// ITEM-GROUNDMOVE-130 stamps literal one before insertion,
				// including a newly adopted native ground Item.
				w.savedObjects.item(value.ObjectID).Token.T08 = 1
			}
		}
		if !merged && !w.addCarried(i, value) {
			return fmt.Errorf("sim: picked up item has no destination container")
		}
		if s.ObjectID != 0 {
			items = w.sacks[j].ItemInstances
		}
	}
	if !w.finishLoadMutation(i, before) {
		return fmt.Errorf("sim: picked up load cannot be derived")
	}
	w.purses[owner] += s.Gold // wraps at 32 bits, normaliseSacks' own rule
	w.sacks = append(w.sacks[:j], w.sacks[j+1:]...)
	w.publishActorItems(i, SelfSlot, false)
	if s.ObjectID != 0 {
		row := w.savedObjects.sack(s.ObjectID)
		if row.Origin.Kind == SavedObjectOriginal {
			if _, issue := w.removeSavedSackCell(uint16(x) | uint16(y)<<8); issue != "" {
				return fmt.Errorf("sim: Sack removal: %s", issue)
			}
		}
		if err := w.savedObjects.RetireSack(s.ObjectID, s.Gold); err != nil {
			return err
		}
		if err := w.validateSavedObjects(); err != nil {
			return err
		}
	}
	if w != live {
		*live = *w
	}
	return nil
}

// dropAll is instant 20's whole arm.
//
// The unit's WHOLE container goes to the ground at the cell the unit itself
// stands on, and the unit is left holding an empty one. The node carries no
// coordinate at all: the decoded arm forwards the unit's own cell, so the drop
// is always at its feet.
//
// IT MERGES RATHER THAN PLANTING A SECOND SACK, and that is pourSack's whole
// body — the same primitive the corpse drop takes, and the same
// one-sack-per-cell rule normaliseSacks applies at load.
//
// GOLD IS ZERO. The decoded arm forwards a literal 0 and does not call the
// death-gold rule, so a unit's purse is untouched. Nothing else moves
// either: not the twelve worn places, which are different state from the
// container and which the corpse drop DOES move, not the position, the
// owner, the group, the health, the order or the tick.
//
// THE LIST IS THE FLAT EXPANSION AND NOT THE CONTAINER SLICE (0138 D-8's own
// rule): pourSack takes what it is given by reference and copies nothing, so
// handing over the container itself and then nilling the field would leave the
// sack aliasing a slice this arm has just given up.
//
// A UNIT HOLDING NOTHING PLANTS NO SACK, and a unit whose cell lies outside the
// map keeps its container: a sack outside the bounds is a state the decoder
// refuses, so an arm that planted one would build a world this package cannot
// read back. That guard is the corpse drop's own and is asked before anything is
// taken away, so the refusal leaves the world exactly as it was found.
//
// THE CONTAINER BECOMES NIL rather than an empty slice, which is the state every
// entity that has never held anything is already in.
func (w *World) dropAll(i int) {
	if w.savedObjects != nil && w.actorHasSavedItems(i) {
		n := w.sourceMutationCopy(i)
		before := n.beginLoadMutation(i)
		if n.sourceMutationReady(i) && n.dropSavedPack(i, 0) && n.finishLoadMutation(i, before) && n.savedMutationValid() {
			*w = n
		}
		return
	}
	if !w.sourceMutationReady(i) {
		return
	}
	if !w.holdsSomething(i) {
		return
	}
	x, y := w.entities[i].X, w.entities[i].Y
	if sackFault(w.bounds, Sack{X: x, Y: y}) != nil {
		return
	}
	before := w.beginLoadMutation(i)
	items := expandItems(w.carried[i])
	live := w
	if w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		w = &n
	}
	w.carried[i] = nil
	if !w.finishLoadMutation(i, before) {
		return
	}
	if w != live {
		*live = *w
		w = live
	}
	w.pourSack(x, y, 0, items)
}
