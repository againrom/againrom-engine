package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
)

// The town shop: four shelves of generated stock, a five-place table both
// sides trade across, and the two commits.
//
// IT IS FRONT-END STATE AND NOT HASHED SIMULATION STATE. SHOP-SAVE-015
// establishes the original writes no part of a shop's stock to a save. Shop
// entries nevertheless carry sim.ItemInstance-compatible identity so buying
// and selling cannot strip effects or stored value at the town boundary.

// ShopShelf names one of the merchant's four shelves.
type ShopShelf int

const (
	// ShelfArmour is the armour and shields shelf — SHOP-POOL-006's kind 1,
	// the two collections drawn as one pool.
	ShelfArmour ShopShelf = iota
	// ShelfWeapons is kind 2.
	ShelfWeapons
	// ShelfMagic is SHOP-MAGIC-007's mode 6: the union of the other two
	// pools. A second stage attaches one to three eligible effects, prices the
	// result, and redraws candidates that cannot accept an effect.
	ShelfMagic
	// ShelfBooks is the fourth shelf: spell books and consumables.
	ShelfBooks
	numShopShelves
)

// String is the shelf's own name, which is what the room's selector row reads.
func (s ShopShelf) String() string {
	switch s {
	case ShelfArmour:
		return "ARMOUR"
	case ShelfWeapons:
		return "WEAPONS"
	case ShelfMagic:
		return "MAGIC"
	case ShelfBooks:
		return "BOOKS"
	}
	return "unknown"
}

// shopShelfDraws is how many items each shelf takes, verbatim from
// SHOP-GEN-005: 100 armour and shields, 100 weapons, 20 magic items. Books
// draw their variable 1..8 count separately after these fixed shelves.
var shopShelfDraws = [numShopShelves]int{
	ShelfArmour:  100,
	ShelfWeapons: 100,
	ShelfMagic:   20,
	ShelfBooks:   0,
}

// The readable book art codes are not monotonic in installed school order.
// Spells use 1 Fire, 2 Water, 3 Air, 4 Earth and 5 Astral (each school holds
// its own "Protection from" spell in the installed Spells rows); the five
// class-14 codes are the shipped generic school-book pictures and names.
var shopBookCodeBySchool = [6]data.ItemCode{
	0,
	0x0e15, // Fire
	0x0e14, // Water
	0x0e13, // Air
	0x0e16, // Earth
	0x0e17, // Astral
}

const shopBookLabelCode data.ItemCode = 0x0e17

// ShopTablePlaces is how many places the table holds (SHOP-TRAY-024: one strip
// of five 80x80 cells, shared by both sides).
//
// The tray CONTAINER in the original has no capacity test anywhere
// (SHOP-TRAY-025); the widget over it has five cells, bounds its hit test to
// 0..4 and refuses a display-list append past that count. Five is therefore what
// a player can reach, and this model refuses a sixth (provenance A-3).
const ShopTablePlaces = 5

// ShopItem is one element: a code, what one unit of it costs, and how many units
// the element holds.
//
// A GENERATED ELEMENT ALWAYS HOLDS ONE UNIT. SHOP-DUP-028 establishes the
// generator appends with a plain Add and nothing dedups, so two draws of one
// candidate are two elements rather than one at count 2. A count above 1 arrives
// only from the player's own pack.
type ShopItem struct {
	cityID          sim.SavedObjectID
	cityOrigins     []string
	owners          []shopOwnerQuantity
	ObjectID        sim.SavedObjectID
	Code            data.ItemCode
	Kind            uint8
	Effects         []sim.ItemEffect
	Price           int32
	Count           int32
	Weight          int16
	WeightPresent   bool
	SourceEquipment sim.SourceEquipment
}

func shopItemFromInstance(item sim.ItemInstance, count int32) ShopItem {
	item = item.Clone()
	return ShopItem{ObjectID: item.ObjectID, Code: data.ItemCode(item.Code), Kind: item.Kind, Effects: item.Effects, Price: item.Price, Count: count, Weight: item.Weight, WeightPresent: item.WeightPresent, SourceEquipment: item.SourceEquipment}
}

func (i ShopItem) Instance() sim.ItemInstance {
	return sim.ItemInstance{ObjectID: i.ObjectID, Code: uint16(i.Code), Kind: i.Kind,
		Effects: append([]sim.ItemEffect(nil), i.Effects...), Price: i.Price, Weight: i.Weight, WeightPresent: i.WeightPresent, SourceEquipment: i.SourceEquipment}
}

func (i ShopItem) Clone() ShopItem {
	i.Effects = append([]sim.ItemEffect(nil), i.Effects...)
	i.cityOrigins = append([]string(nil), i.cityOrigins...)
	i.owners = append([]shopOwnerQuantity(nil), i.owners...)
	return i
}

func shopItemEqual(a, b ShopItem) bool {
	if a.cityID != b.cityID {
		return false
	}
	// Distinct saved nodes remain separate even when all their values agree.
	// Unidentified items retain the existing value-based folding rule.
	return a.ObjectID == b.ObjectID && a.Price == b.Price && sim.ItemEqual(a.Instance(), b.Instance())
}

func shopItemStateEqual(a, b ShopItem) bool {
	return a.Count == b.Count && a.Code == b.Code && a.Kind == b.Kind && a.Price == b.Price &&
		sim.StackStateEqual(sim.StackItem(a.Instance(), uint32(a.Count)), sim.StackItem(b.Instance(), uint32(b.Count)))
}

// ShopPlace is one of the table's five places: an item, and which side of the
// deal owns it.
//
// Mine IS THE OWNERSHIP STAMP (SHOP-TRAY-025's elem+0x14) AND A BOOL BY
// DESIGN. Every consumer in the original reads it as a zero test: the buy
// commit walks the places where it is clear, the sell commit walks the
// places where it is set, and clearing the table switches on it. Single
// player has one customer, so the field carries nothing the test does not.
//
// From is the shelf a merchant-owned place came off, so clearing the table puts
// it back where it was. It is meaningless on a player-owned place.
type ShopPlace struct {
	ShopItem
	Mine bool
	From ShopShelf
}

// Shop is the merchant: what is on his shelves, and what is on the table between
// him and the player.
type Shop struct {
	inputEpoch *byte // transient click scope, replaced on stock regeneration
	// ceiling is the value window's upper bound in coins — shop+0x70, the
	// original's one per-shop parameter (SHOP-CAP-004, SHOP-MISSION-018).
	ceiling int32

	shelves [numShopShelves][]ShopItem
	table   []ShopPlace

	// tbl supplies definition rows for sale classification and shelf ordering.
	tbl *mapload.Table

	// restocks is the number of explicit tavern Sleep commands applied to
	// this town-arrival shop. The shop is rebuilt on every town arrival, so
	// the counter is session-local shop state and is not persisted.
	restocks uint64

	// drawSeed is the seed of the last Generate. A tavern restock reads it to
	// decide whether a mod's stocked item is offered again (modStockOffered).
	drawSeed int64
}

// NewShop is an empty shop at a ceiling. It stocks nothing: generation is the
// caller's own step, because SHOP-TOWN-022 puts SetCap and Generate together at
// one moment and this type should not decide when that moment is.
func NewShop(ceiling int32) *Shop { return &Shop{ceiling: ceiling, inputEpoch: new(byte)} }

// Isolate stock/table ownership until a source actor callback has admitted
// the paired pack change. This does not create a new click epoch.
func cloneShopMutation(s *Shop) *Shop {
	if s == nil {
		return &Shop{}
	}
	n := *s
	n.table = s.Table()
	for i := range n.shelves {
		n.shelves[i] = s.Shelf(ShopShelf(i))
	}
	return &n
}

// Ceiling is the value window's upper bound this shop stocks under.
func (s *Shop) Ceiling() int32 {
	if s == nil {
		return 0
	}
	return s.ceiling
}

// Shelf is what shelf holds, in the order shopsort.go puts it in (`DIV-319`):
// fighter items, then mage items, then the rest; inside a class by item kind;
// inside a kind by price, cheapest first. A shop generated with no item table
// holds its shelves in the order they were drawn.
func (s *Shop) Shelf(shelf ShopShelf) []ShopItem {
	if s == nil || shelf < 0 || shelf >= numShopShelves {
		return nil
	}
	out := make([]ShopItem, len(s.shelves[shelf]))
	for i := range out {
		out[i] = s.shelves[shelf][i].Clone()
	}
	return out
}

// Table is the five places, in the order they were put down.
func (s *Shop) Table() []ShopPlace {
	if s == nil {
		return nil
	}
	out := make([]ShopPlace, len(s.table))
	for i := range out {
		out[i] = s.table[i]
		out[i].ShopItem = out[i].ShopItem.Clone()
	}
	return out
}

// Generate clears every shelf and refills it from the tables (SHOP-LIFE-013,
// SHOP-GEN-005).
//
// IT CLEARS BEFORE IT FILLS AND THE CLEAR IS AN ASSIGNMENT, not a walk. That is
// SHOP-GEN-005's own shape once its laziness clause is taken out: the routine's
// first act past its gate destroys every element of every shelf and RemoveAlls
// the arrays, so generating twice leaves the shelf counts where one generation
// leaves them rather than doubled.
//
// THE TABLE IS CLEARED TOO. The original cannot reach this state — its gate
// refuses to generate while a customer is inside — and a table left holding
// merchant-owned places from a stock that no longer exists is the one way this
// model could hold an item belonging to nothing.
//
// The seed is the caller's. SHOP-RNG-008 establishes the original seeds the CRT
// rand() from a clock and no assortment is reproducible; this build's stock is
// reproducible from a seed the caller derives from campaign state, which is
// provenance A-1's disclosed divergence.
func (s *Shop) Generate(t *mapload.Table, seed int64) { s.GenerateWith(t, seed, nil) }

// GenerateWith is Generate drawing on stock when it is the shared stream of
// original mode (SHOP-RNG-008); otherwise the stock draws on its own
// generator at seed.
func (s *Shop) GenerateWith(t *mapload.Table, seed int64, stock *random.Stream) {
	s.drawSeed = seed
	var r shopDrawSource = random.NewGo(seed)
	if stock != nil && stock.Shared() {
		r = stock
	}
	s.generate(t, r)
}

// Restock clears and refills an unattended shop with a fresh deterministic
// draw. A non-empty table is refused: leaving the shop normally clears it, and
// destroying a staged player item would turn the tavern convenience into an
// inventory mutation.
func (s *Shop) Restock(t *mapload.Table, seed int64) bool { return s.RestockWith(t, seed, nil) }

// RestockWith is Restock drawing as GenerateWith does.
func (s *Shop) RestockWith(t *mapload.Table, seed int64, stock *random.Stream) bool {
	if s == nil || t == nil || len(s.table) != 0 {
		return false
	}
	s.restocks++
	s.GenerateWith(t, shopRestockSeed(seed, s.restocks), stock)
	return true
}

func (s *Shop) generate(t *mapload.Table, r shopDrawSource) {
	if s == nil {
		return
	}
	s.inputEpoch = new(byte)
	s.table = nil
	s.tbl = t
	for i := range s.shelves {
		s.shelves[i] = nil
	}

	allArmour := shopArmourPool(t, s.ceiling)
	allWeapons := shopWeaponPool(t, s.ceiling)
	armour := shopPlainShelfPool(allArmour)
	weapons := shopPlainShelfPool(allWeapons)
	pools := [numShopShelves][]data.ShopCandidate{
		ShelfArmour:  armour,
		ShelfWeapons: weapons,
		ShelfMagic:   append(append([]data.ShopCandidate{}, allArmour...), allWeapons...),
	}

	for shelf := range pools {
		pool := pools[shelf]
		if len(pool) == 0 {
			continue
		}
		n := shopShelfDraws[shelf]
		items := make([]ShopItem, 0, n)
		attempts := n
		if ShopShelf(shelf) == ShelfMagic {
			attempts = 10*n + 1
		}
		for i := 0; i < attempts && len(items) < n; i++ {
			c := pool[r.Intn(len(pool))]
			if ShopShelf(shelf) == ShelfMagic {
				item, ok := shopEnchantedItem(c, s.ceiling, t, r)
				if !ok {
					continue
				}
				items = append(items, item)
				continue
			}
			items = append(items, shopItemFromInstance(sim.ItemInstance{
				Code: uint16(c.Code), Kind: c.ItemKind, Price: c.Price}, 1))
		}
		items = shopStackShelf(items)
		shopSortShelf(items, t)
		s.shelves[shelf] = items
	}
	// A mod's item that asks for a place is added to the armour shelf after the
	// random draw, without taking a draw (DIV-1862).
	if stock := modShopStock(t, s.drawSeed, s.restocks, len(armour)); len(stock) != 0 {
		items := append(s.shelves[ShelfArmour][:len(s.shelves[ShelfArmour]):len(s.shelves[ShelfArmour])], stock...)
		items = shopStackShelf(items)
		shopSortShelf(items, t)
		s.shelves[ShelfArmour] = items
	}

	books := shopBookPool(t, s.ceiling)
	if r == nil {
		return
	}
	consumables := append(books, shopScrollPool(t, s.ceiling, r)...)
	var items []ShopItem
	if len(consumables) != 0 {
		n, _ := shopDrawInclusive(r, 1, 8)
		for i := 0; i < n; i++ {
			items = append(items, consumables[r.Intn(len(consumables))].Clone())
		}
	}
	// Owner policy (DIV-1025): retain the random stock, then guarantee one
	// book of each eligible spell without consuming another random draw.
	for _, book := range books {
		spell, _ := book.Instance().BookSpell()
		found := false
		for _, item := range items {
			if got, ok := item.Instance().BookSpell(); ok && got == spell {
				found = true
				break
			}
		}
		if !found {
			items = append(items, book.Clone())
		}
	}
	items = append(items, shopPotionStock(t, r)...)
	items = shopStackShelf(items)
	shopSortShelf(items, t)
	s.shelves[ShelfBooks] = items
}

// The Armor class field range the weapons and armour shelves drop (SHOP-118,
// SHOP-121): an Armors row whose Slot is 3, 4 or 5 never enters either shelf.
// The range and the sutableFor bit 0 test beside it are program constants.
const (
	shopPlainShelfDroppedClassLow  = 3
	shopPlainShelfDroppedClassHigh = 5
)

// shopPlainShelfPool is the weapons or armour shelf's candidates: the walk's
// population less a Weapon whose sutableFor bit 0 (Fighter) is clear and an
// Armor whose class field is 3..5 (SHOP-118). A Shield takes no test; its
// class is 2. The Magic Items shelf skips both tests and draws the unfiltered
// population, where a candidate that can take no first effect is rejected by
// shopEnchantedItem (SHOP-119).
func shopPlainShelfPool(pool []data.ShopCandidate) []data.ShopCandidate {
	out := make([]data.ShopCandidate, 0, len(pool))
	for _, candidate := range pool {
		if candidate.ItemKind == 2 && !candidate.Fighter {
			continue
		}
		if candidate.ItemKind != 2 && candidate.EffectSlot >= shopPlainShelfDroppedClassLow &&
			candidate.EffectSlot <= shopPlainShelfDroppedClassHigh {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

// shopBookPool is every readable one-spell book the installed Spells table
// prices positively and no higher than this shop's ceiling. The instance owns
// the complete identity later used by stacking, trade and reading.
func shopBookPool(t *mapload.Table, ceiling int32) []ShopItem {
	if t == nil || t.Spells == nil {
		return nil
	}
	limit := t.Spells.Len()
	if limit > 32 {
		limit = 32
	}
	pool := make([]ShopItem, 0, limit)
	for spell := 1; spell < limit; spell++ {
		if !shopBookSpellAdmitted(spell) {
			continue
		}
		params := t.Spells.EntryParams(spell)
		price, ok := mapload.SpellBookCost(uint16(spell), t)
		if !ok || price <= 0 || price > ceiling || len(params) <= 2 {
			continue
		}
		school := params[2]
		if school <= 0 || int(school) >= len(shopBookCodeBySchool) {
			continue
		}
		item := sim.ItemInstance{
			Code:  uint16(shopBookCodeBySchool[school]),
			Kind:  5,
			Price: price,
			Effects: []sim.ItemEffect{{
				Kind: 42, Mode: 0, Operand: uint32(spell),
			}},
		}
		pool = append(pool, shopItemFromInstance(item, 1))
	}
	return pool
}

// maskTable is t's collection as a data.MaskTable, or nil where it cannot
// answer the raw block the material masks live in.
func maskTable(c data.Collection) data.MaskTable {
	m, ok := c.(data.MaskTable)
	if !ok {
		return nil
	}
	return m
}

// shopArmourPool is SHOP-POOL-006's kind 1: Shields and Armors admitted by one
// window and drawn as one pool. The two are concatenated rather than interleaved
// — the original's own kind dispatches over both collections in turn.
func shopArmourPool(t *mapload.Table, ceiling int32) []data.ShopCandidate {
	if t == nil {
		return nil
	}
	pool := data.ShopPool(maskTable(t.Shields), data.ShieldShopClass, t.Shapes, t.Materials, ceiling)
	return append(pool,
		data.ShopPool(maskTable(t.Armors), data.ArmorShopClass, t.Shapes, t.Materials, ceiling)...)
}

// shopWeaponPool is kind 2.
func shopWeaponPool(t *mapload.Table, ceiling int32) []data.ShopCandidate {
	if t == nil {
		return nil
	}
	return data.ShopPool(maskTable(t.Weapons), data.WeaponShopClass, t.Shapes, t.Materials, ceiling)
}

// TakeFromShelf moves n units of element i onto the table in a NEW place,
// stamped as the merchant's and remembering the shelf they came off
// (SHOP-TRAY-025).
//
// IT IS APPEND-ONLY, FOR PutOnTable's REASON. Whether n units should instead
// join a place the merchant's own goods already stand in is the caller's
// identity rule to state, not this method's to guess — shopFromShelf's own
// shopFindHisPlace is the one this build applies, and GrowTablePlace beside
// RemoveFromShelf is what a caller that found a match uses instead.
//
// n AT OR ABOVE THE ELEMENT'S OWN COUNT TAKES ALL OF IT and the element
// leaves the shelf, exactly as this method always took it before shelf
// elements could hold more than one unit; a smaller n splits the element and
// leaves the remainder on the shelf, which is TakeOffTable's rule one
// container over. SHOP-TRAY-025 establishes the protocol carries a quantity
// and both containers honour a split; SHOP-TRAY-027 grades whether a
// partial-stack move is REACHABLE in the original as Unknown, and the split
// this build reaches is the owner's own default-one convention (`DIV-047`),
// not a claimed original behaviour.
//
// A full table refuses and changes nothing, and so does n <= 0.
func (s *Shop) TakeFromShelf(shelf ShopShelf, i int, n int32) bool {
	if s == nil || len(s.table) >= ShopTablePlaces || n <= 0 {
		return false
	}
	items := s.Shelf(shelf)
	if i < 0 || i >= len(items) {
		return false
	}
	moved, ok := s.RemoveFromShelf(shelf, i, n)
	if !ok {
		return false
	}
	s.table = append(s.table, ShopPlace{ShopItem: moved, From: shelf})
	return true
}

// RemoveFromShelf takes n units of element i OFF THE SHELF, reporting what
// moved — TakeFromShelf's own bounds check and split, restated for a caller
// that is not putting the item on the table at all (1005 round 2,
// `DIV-087`): a direct shelf-to-doll purchase never opens a table place, so
// the table's own five-place limit does not gate it.
//
// The returned item carries exactly what moved: n where the element holds
// more, or the whole element where it does not, in which case the element
// leaves the shelf. n <= 0 is refused, as an index out of range already is.
func (s *Shop) RemoveFromShelf(shelf ShopShelf, i int, n int32) (ShopItem, bool) {
	if s == nil || n <= 0 {
		return ShopItem{}, false
	}
	items := s.Shelf(shelf)
	if i < 0 || i >= len(items) {
		return ShopItem{}, false
	}
	item := items[i]
	if n >= item.Count {
		s.shelves[shelf] = append(items[:i:i], items[i+1:]...)
		return item, true
	}
	moved := item
	moved.Count = n
	s.shelves[shelf][i].Count -= n
	return moved, true
}

// PutOnTable puts an item the player already owns on the table, stamped as his,
// in a NEW place. The caller has already taken it out of the pack; this
// refuses a full table and an empty element so no caller has to unwind one.
//
// IT IS APPEND-ONLY AND STAYS THAT WAY (seat). Whether a caller's own lot
// should instead join an existing place is that caller's identity rule to
// state, not this method's to guess — shopFromPack's own shopFindMinePlace
// is the one this build applies, and GrowTablePlace below is what a caller
// that found a match uses instead of a second call here.
func (s *Shop) PutOnTable(item ShopItem) bool {
	if s == nil || len(s.table) >= ShopTablePlaces || item.Count <= 0 {
		return false
	}
	s.table = append(s.table, ShopPlace{ShopItem: item.Clone(), Mine: true})
	return true
}

// GrowTablePlace adds n units to table place i's own count, in place: no new
// place is opened and no other place is touched. It is the merge half of
// PutOnTable's append-only contract, for a caller that has already found a
// place to join by its own identity rule (shopFindMinePlace, shoproom.go) —
// this method makes no ownership or identity decision of its own.
func (s *Shop) GrowTablePlace(i int, n int32) bool {
	if s == nil || i < 0 || i >= len(s.table) || n <= 0 {
		return false
	}
	s.table[i].Count += n
	return true
}

// TakeOffTable moves up to n units off place i, so the caller can put them
// back where the returned place's stamp says they belong. The returned place
// carries exactly what moved: n itself where the place holds more, or the
// whole original where it does not.
func (s *Shop) TakeOffTable(i int, n int32) (ShopPlace, bool) {
	if s == nil || i < 0 || i >= len(s.table) || n <= 0 {
		return ShopPlace{}, false
	}
	place := s.table[i]
	place.ShopItem = place.ShopItem.Clone()
	if n >= place.Count {
		s.table = append(s.table[:i:i], s.table[i+1:]...)
		return place, true
	}
	moved := place
	moved.Count = n
	s.table[i].Count -= n
	moved.owners, s.table[i].owners = takeShopOwners(place.owners, n)
	return moved, true
}

// ClearTable sends every place home and reports what the PLAYER gets back
// (SHOP-TRAY-025's opcode 0x35).
//
// The merchant's places go back on the shelf they came off, here. The player's
// come back to the caller, because his pack is not this type's to write.
// No coin moves either way.
func (s *Shop) ClearTable() []ShopItem {
	if s == nil {
		return nil
	}
	var mine []ShopItem
	for _, place := range s.table {
		if place.Mine {
			mine = append(mine, place.ShopItem.Clone())
			continue
		}
		s.returnToShelf(place.From, place.ShopItem)
	}
	s.table = nil
	return mine
}

// returnToShelf puts an item back. IT MERGES INTO AN EQUAL ELEMENT, which is
// what SHOP-DUP-028 establishes the original's own return path does.
//
// IT THEN RE-SORTS THE SHELF, so an item returned to a shelf that holds no
// equal element lands where the owner's order puts it rather than at the end
// (shopsort.go, `DIV-319`). A merge moves nothing and the re-sort is a no-op
// on an already-sorted shelf, so this is reached for its own sake only on the
// open-a-new-cell arm.
func (s *Shop) returnToShelf(shelf ShopShelf, item ShopItem) {
	item.owners = nil
	if shelf < 0 || shelf >= numShopShelves {
		shelf = ShelfMagic
	}
	if item.Count <= 0 {
		return
	}
	items := s.shelves[shelf]
	for i := range items {
		if shopItemEqual(items[i], item) {
			items[i].Count += item.Count
			return
		}
	}
	s.shelves[shelf] = append(items, item.Clone())
	shopSortShelf(s.shelves[shelf], s.tbl)
}

// BuyTotal is what the merchant's places on the table cost: the sum of
// quantity x unit price (SHOP-TRAY-026's +0x154, and the same arithmetic
// SHOP-BUY-009's commit debits).
func (s *Shop) BuyTotal() int32 {
	if s == nil {
		return 0
	}
	var total int32
	for _, place := range s.table {
		if !place.Mine {
			total += place.Count * place.Price
		}
	}
	return total
}

// SHOP-TRAY-026
func (s *Shop) SellTotal() int32 {
	if s == nil {
		return 0
	}
	var total int32
	for _, place := range s.table {
		if place.Mine {
			total += ((place.Price + 1) / 2) * place.Count
		}
	}
	return total
}

// SellPayout is what Sell will actually credit: ceil(quantity x price / 2) per
// place (SHOP-SELL-010, the ftol(0.5n + 0.5) the claim reads as exactly
// ceil(n/2)).
func (s *Shop) SellPayout() int32 {
	if s == nil {
		return 0
	}
	var total int32
	for _, place := range s.table {
		if place.Mine {
			total += shopHalfUp(place.Count * place.Price)
		}
	}
	return total
}

// shopHalfUp is ceil(n/2) for a non-negative n — the integer form of
// SHOP-SELL-010's ftol(0.5 x n + 0.5), which that claim establishes is exact
// with no representation error.
func shopHalfUp(n int32) int32 {
	if n < 0 {
		return 0
	}
	return (n + 1) / 2
}

// Buy commits the merchant's places (SHOP-BUY-009, SHOP-TRAY-026).
//
// THE GUARD IS THE CLIENT'S AND IT RUNS FIRST (SHOP-TRAY-026): nothing is sent
// unless the total is non-zero and the purse covers the WHOLE basket. That is
// what makes SHOP-BUY-009's per-element abort unreachable — the commit below
// debits monotonically, so an accepted basket makes every prefix affordable —
// and the abort is written all the same, because it is the routine's own shape
// and a purse that disagreed with the total would otherwise buy on credit.
//
// It reports what the player receives and what he pays. On a refusal it reports
// nothing and changes nothing: the table, the shelves and the purse are all as
// they were. The player's own places stay on the table either way.
func (s *Shop) Buy(purse int32) (bought []ShopItem, spend int32, ok bool) {
	if s == nil {
		return nil, 0, false
	}
	total := s.BuyTotal()
	if total == 0 || purse < total {
		return nil, 0, false
	}

	left := purse
	kept := make([]ShopPlace, 0, len(s.table))
	for _, place := range s.table {
		if place.Mine {
			kept = append(kept, place)
			continue
		}
		cost := place.Count * place.Price
		if cost > left {
			// SHOP-BUY-009's JMP past the iterator: the rest of the basket
			// is not bought either. The guard above makes this unreachable.
			kept = append(kept, place)
			break
		}
		left -= cost
		bought = append(bought, place.ShopItem)
	}
	s.table = kept
	return bought, purse - left, true
}

// Sell commits the player's places (SHOP-SELL-010, SHOP-TRAY-026).
//
// THE GUARD IS ONE TEST AND NOT TWO: the sell arm sends on a non-zero total
// alone, because nothing about handing goods over can fail on the purse.
//
// Each place is credited ceil(quantity x price / 2) — the WHOLE stack halved
// once, not each unit — and goes back on a shelf. The merchant's own places stay
// on the table.
//
// WHICH SHELF IS DECIDED BY THE ITEM'S CLASS, not by where it came from
// (SHOP-SELL-010's return path switches on item+0x44 and ends in a default arm
// that logs and files the item under Magic Items). A sold item may never have
// been on a shelf at all, so there is no origin to put it back to.
func (s *Shop) Sell() (paid int32, ok bool) {
	if s == nil || s.SellTotal() == 0 {
		return 0, false
	}
	kept := make([]ShopPlace, 0, len(s.table))
	for _, place := range s.table {
		if !place.Mine {
			kept = append(kept, place)
			continue
		}
		// Retain unpriced goods on the tray. Fresh Documents carry the
		// installed -1 sentinel (DAT-DOC-021), not the zero used by the
		// older control. A priced neighbour must not make that physical
		// item disappear for no payment. Keep its source price unchanged;
		// positive-price accounting still follows SHOP-SELL-010.
		if place.Price <= 0 {
			kept = append(kept, place)
			continue
		}
		paid += shopHalfUp(place.Count * place.Price)
		s.returnToShelf(shopShelfForItem(place.ShopItem, s.tbl), place.ShopItem)
	}
	s.table = kept
	return paid, true
}

// shopShelfFor is which shelf an item sold to the merchant lands on
// (SHOP-SELL-010): a weapon on the weapon shelf, an armour or a shield on the
// armour shelf, and anything else on the Magic Items shelf — the original's own
// default arm, the one that logs "Item of strange type is returned to shop".
func shopShelfFor(c data.ItemCode) ShopShelf {
	switch c.B() {
	case shopWeaponClass:
		return ShelfWeapons
	case shopShieldClass:
		return ShelfArmour
	}
	if c.B() >= 1 && c.B() <= shopArmourMaxSlot {
		return ShelfArmour
	}
	return ShelfMagic
}

func shopShelfForItem(item ShopItem, table *mapload.Table) ShopShelf {
	if item.Kind == 3 || item.Kind == 4 {
		return ShelfBooks
	}
	if _, readable := item.Instance().BookSpell(); readable {
		return ShelfBooks
	}
	if item.Code.B() == shopWeaponClass {
		var suitability data.Suitability
		known := false
		source := item.SourceEquipment
		if source.Class == sim.SourceWeapon && source.Definition.Present {
			v := source.Definition.Suitable
			suitability, known = data.Suitability{Fighter: v&1 != 0, Mage: v&2 != 0}, true
		} else if table != nil {
			suitability, known = data.SuitabilityFromCode(item.Code, table.Weapons, table.Shields, table.Armors)
		}
		if known && suitability.Mage && !suitability.Fighter {
			return ShelfMagic
		}
	}
	return shopShelfFor(item.Code)
}

// The three class values this file reads off an item code's field B. They are
// pkg/data's own weaponItemClass and shieldItemClass, which that package does
// not export, and the upper bound of an equipment slot — an armour's field B is
// its own Slot column. Spelled here because the shelf a sold item lands on is a
// SHOP rule and not an item rule; nothing else in this file reads a class.
const (
	shopWeaponClass   = 1
	shopShieldClass   = 2
	shopArmourMaxSlot = 12
)

// shopSeed derives the generator's seed from campaign state.
//
// THE THREE INPUTS ARE ALL RECOMPUTED FROM WHAT A SAVE ALREADY CARRIES, which is
// what lets the shop be restocked after a load without the save holding a shelf
// (SHOP-SAVE-015). It is also what makes the stock reproducible where the
// original's is not — provenance A-1, and the one thing about this shop a player
// can see that the original would not have shown him.
//
// The mix is FNV-1a over the three, and the sign bit is dropped so the result is
// a legal seed rather than one that depends on the platform's int64.
func shopSeed(chapter, finished int, ceiling int32) int64 {
	h := uint64(14695981039346656037)
	for _, v := range []uint64{uint64(chapter), uint64(finished), uint64(uint32(ceiling))} {
		for i := 0; i < 8; i++ {
			h ^= (v >> uint(8*i)) & 0xff
			h *= 1099511628211
		}
	}
	return int64(h >> 1)
}

// shopRestockSeed folds the explicit restock ordinal into the town-arrival
// seed. It keeps headless runs reproducible while giving each Sleep command a
// distinct deterministic random stream.
func shopRestockSeed(seed int64, ordinal uint64) int64 {
	h := uint64(seed)
	for i := 0; i < 8; i++ {
		h ^= (ordinal >> uint(8*i)) & 0xff
		h *= 1099511628211
	}
	return int64(h >> 1)
}
