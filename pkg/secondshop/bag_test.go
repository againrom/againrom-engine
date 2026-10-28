package secondshop

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

type bagTestCollection map[int][]int32

func (c bagTestCollection) Len() int                   { return 128 }
func (c bagTestCollection) EntryParams(i int) []int32  { return c[i] }
func (c bagTestCollection) EntryRaw(int) []byte        { return nil }
func (c bagTestCollection) EntryDoubles(int) []float64 { return nil }

// TestBagUnitsRoundTrip codes each item kind as bag units and reads them
// back: the armour ID is material<<12|part<<8|class<<5|row, the enchanted
// spell a castspell effect and a book a teachspell effect.
func TestBagUnitsRoundTrip(t *testing.T) {
	tables := Tables{
		Armors:  bagTestCollection{7: {0, 0, 0, 0, 9}},
		Shields: bagTestCollection{2: {0, 0, 0, 0, 2}},
		Weapons: bagTestCollection{4: {0, 0, 0, 0, 1}, 5: {0, 0, 0, 0, -1}},
	}
	items := []Item{
		{Table: Armor, Row: 7, Class: 3, Material: 5, Count: 1, Price: 4000},
		{Table: Weapon, Row: 4, Class: 1, Material: 2, Spell: 9, Level: 3, Count: 1, Price: 7000},
		{Table: Shield, Row: 2, Class: 0, Material: 1, Count: 1, Price: 300},
		{Table: Magic, Row: 70, Count: 3, Price: 83},
		{Table: Book, Spell: 12, Count: 1, Price: 1500},
	}
	var bag []sim.ItemInstance
	for _, it := range items {
		units, ok := tables.Units(it)
		if !ok {
			t.Fatalf("%+v has no bag form", it)
		}
		bag = append(bag, units...)
	}
	if bag[0].Code != 5<<12|9<<8|3<<5|7 || bag[0].Kind != 1 || bag[1].Kind != 2 || bag[1].Effects[0] != (sim.ItemEffect{Kind: 41, Operand: 9 | 3<<16}) {
		t.Fatalf("armour %#04x kind %d, weapon %+v", bag[0].Code, bag[0].Kind, bag[1])
	}
	got, ok := Stacks(bag)
	if !ok || !reflect.DeepEqual(got, items) {
		t.Fatalf("stacks %+v, want %+v", got, items)
	}
	left, taken, stack, ok := TakeUnits(bag, 3)
	if !ok || len(taken) != 3 || len(left) != 4 || !reflect.DeepEqual(stack, items[3]) {
		t.Fatalf("take: %d left, %d taken, %+v", len(left), len(taken), stack)
	}
	// A weapon row with no body part has no ID form.
	if _, ok := tables.Instance(Item{Table: Weapon, Row: 5, Count: 1}); ok {
		t.Fatal("a row with part -1 was coded")
	}
	// Loss control: a unit with an identity the bag never writes is refused.
	bad := bag[0]
	bad.ObjectID = 7
	if _, ok := FromInstance(bad); ok {
		t.Fatal("a unit with an object identity decoded")
	}
	// A worn copy of each unit, with the fields the bag never writes, comes
	// back as the unit Instance wrote.
	for _, u := range bag {
		worn := u.Clone()
		worn.ObjectID, worn.Weight, worn.WeightPresent, worn.Kind = 7, 9, true, 3
		worn.Effects = append(worn.Effects, sim.ItemEffect{Kind: 7})
		if got := Unit(worn); !reflect.DeepEqual(got, u) {
			t.Fatalf("the unit of %+v is %+v, want %+v", worn, got, u)
		}
	}
}

// TestUnitTakesTheBagForm puts worn items into the bag's form: a bag unit is
// kept as it is, and a worn item built from a row loses the fields the bag
// does not keep and takes the type byte of its part.
func TestUnitTakesTheBagForm(t *testing.T) {
	book := sim.ItemInstance{Code: 0x0e8c, Price: 1500, Effects: []sim.ItemEffect{{Kind: 42, Operand: 12}}}
	if got := Unit(book); !reflect.DeepEqual(got, book) {
		t.Fatalf("the book unit became %+v", got)
	}
	for _, tc := range []struct {
		in   sim.ItemInstance
		kind uint8
	}{
		{sim.ItemInstance{ObjectID: 3, Code: 0x2101, Kind: 1, Price: 40, Weight: 9, WeightPresent: true}, 2},
		{sim.ItemInstance{Code: 0x4201, Kind: 1, Price: 15}, 0},
		{sim.ItemInstance{Code: 0x5321, Kind: 3, Price: 80, Effects: []sim.ItemEffect{{Kind: 41, Operand: 9 | 2<<16}, {Kind: 7}}}, 1},
	} {
		got := Unit(tc.in)
		if got.Code != tc.in.Code || got.Kind != tc.kind || got.Price != tc.in.Price || got.ObjectID != 0 || got.WeightPresent {
			t.Fatalf("Unit(%+v) = %+v, want kind %d", tc.in, got, tc.kind)
		}
		if _, ok := FromInstance(got); !ok {
			t.Fatalf("Unit(%+v) = %+v has no bag form", tc.in, got)
		}
	}
	if got := Unit(sim.ItemInstance{Code: 0x5321, Effects: []sim.ItemEffect{{Kind: 41, Operand: 9 | 2<<16}, {Kind: 7}}}); len(got.Effects) != 1 || got.Effects[0] != (sim.ItemEffect{Kind: 41, Operand: 9 | 2<<16}) {
		t.Fatalf("the enchanted armour's effects are %+v, want its castspell only", got.Effects)
	}
}
