package game

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/sim"
)

func cityMutationFixture() *cityObjectTopology {
	g := &cityObjectTopology{
		Version: cityObjectTopologyVersion, NextID: 100,
		Items: []cityItemTopology{
			{ID: 10, Effects: []sim.SavedObjectID{30, 30, 0, 31}, Spell: 40},
			{ID: 11, Effects: []sim.SavedObjectID{30}, Spell: 40},
			{ID: 12, Effects: []sim.SavedObjectID{0}},
			{ID: 13},
		},
		Effects: []sim.SavedObjectID{30, 31}, Spells: []sim.SavedObjectID{40, 41},
		Roots: []cityPartyObjectRoots{
			{PartyID: []byte("alpha"), Pack: []sim.SavedObjectID{0, 10, 10, 11, 0, 12}},
			{PartyID: []byte("beta"), Pack: []sim.SavedObjectID{0, 11}},
		},
		Books: []cityBookTopology{{PartyID: []byte("alpha")}, {PartyID: []byte("beta")}},
	}
	g.Roots[0].Worn[4], g.Roots[1].Worn[2] = 11, 10
	g.Books[0].Slots[0], g.Books[0].Slots[2], g.Books[0].Slots[3] = 40, 40, 41
	g.Books[1].Slots[0] = 40
	return g
}

func requireCityMutationUnchanged(t *testing.T, g, before *cityObjectTopology) {
	t.Helper()
	if !reflect.DeepEqual(g, before) {
		t.Fatal("mutation changed its input or allocator floor")
	}
}

func TestCityItemRootMoveKeepsExactAliasesAndNulls(t *testing.T) {
	g := cityMutationFixture()
	before := g.Clone()
	from := cityItemLocation{PartyID: "alpha", Kind: cityItemPack, Index: 2}
	to := cityItemLocation{PartyID: "beta", Kind: cityItemWorn, Index: 0}
	n, err := moveCityItemRoot(g, 10, from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := before.Clone()
	want.Roots[0].Pack = []sim.SavedObjectID{0, 10, 11, 0, 12}
	want.Roots[1].Worn[0] = 10
	if !reflect.DeepEqual(n, want) {
		t.Fatal("whole move changed unselected aliases, children, nulls or allocator", n)
	}
	requireCityMutationUnchanged(t, g, before)
	// All returned slices are detached, including ones the operation did not edit.
	n.Items[0].Effects[0] = 0
	n.Roots[0].PartyID[0] = 'X'
	n.Roots[1].Pack[0] = 10
	n.Books[0].PartyID[0] = 'Y'
	n.Effects[0], n.Spells[0] = 99, 98
	requireCityMutationUnchanged(t, g, before)
}

func TestCityItemRootMoveReordersAndDetachesWithoutOwners(t *testing.T) {
	g := cityMutationFixture()
	from := cityItemLocation{PartyID: "alpha", Kind: cityItemPack, Index: 1}
	to := cityItemLocation{PartyID: "alpha", Kind: cityItemPack, Index: 4}
	n, err := moveCityItemRoot(g, 10, from, to)
	if err != nil || !slices.Equal(n.Roots[0].Pack, []sim.SavedObjectID{0, 10, 11, 0, 10, 12}) {
		t.Fatal("pack destination did not use its post-removal index", err, n)
	}
	to = cityItemLocation{PartyID: "beta", Kind: cityItemWorn, Index: 11}
	n, err = moveCityItemRoot(g, 13, cityItemLocation{}, to)
	if err != nil || n.Roots[1].Worn[11] != 13 || n.NextID != g.NextID {
		t.Fatal("detached item failed to retain identity", err, n)
	}
	back, err := moveCityItemRoot(n, 13, to, cityItemLocation{})
	if err != nil || !reflect.DeepEqual(back, g) {
		t.Fatal("detached roundtrip left a fake owner or pruned nodes", err, back)
	}
}

func TestCityItemRootMoveRoundTripsAliasedDetachedOccurrence(t *testing.T) {
	g := cityMutationFixture()
	before := g.Clone()
	from := cityItemLocation{PartyID: "alpha", Kind: cityItemPack, Index: 1}
	onTable, err := moveCityItemRoot(g, 10, from, cityItemLocation{})
	if err != nil {
		t.Fatal(err)
	}
	wantTable := before.Clone()
	wantTable.Roots[0].Pack = []sim.SavedObjectID{0, 10, 11, 0, 12}
	if !reflect.DeepEqual(onTable, wantTable) {
		t.Fatal("table move changed surviving aliases or children", onTable)
	}
	back, err := moveCityItemRoot(onTable, 10, cityItemLocation{}, from)
	if err != nil || !reflect.DeepEqual(back, before) {
		t.Fatal("table occurrence could not return beside its surviving alias", err, back)
	}
	worn := cityItemLocation{PartyID: "beta", Kind: cityItemWorn, Index: 0}
	equipped, err := moveCityItemRoot(onTable, 10, cityItemLocation{}, worn)
	wantWorn := wantTable.Clone()
	wantWorn.Roots[1].Worn[0] = 10
	if err != nil || !reflect.DeepEqual(equipped, wantWorn) {
		t.Fatal("table occurrence could not equip while another worn alias survived", err, equipped)
	}
	if candidate, err := moveCityItemRoot(onTable, 10, cityItemLocation{}, cityItemLocation{PartyID: "beta", Kind: cityItemWorn, Index: 2}); err == nil || candidate != nil {
		t.Fatal("attested table source bypassed occupied-destination check", err, candidate)
	}
	partial, id, err := splitCityItemRoot(onTable, 10, cityItemLocation{}, worn)
	if err != nil || id != 100 || partial.Roots[1].Worn[0] != id ||
		!reflect.DeepEqual(partial.Items[:len(onTable.Items)], onTable.Items) ||
		!slices.Equal(partial.Roots[0].Pack, onTable.Roots[0].Pack) {
		t.Fatal("partial table occurrence changed a surviving source alias", id, err, partial)
	}
	for _, badID := range []sim.SavedObjectID{0, 40, 99} {
		if candidate, err := moveCityItemRoot(onTable, badID, cityItemLocation{}, worn); err == nil || candidate != nil {
			t.Fatal("attested detached source bypassed exact Item ID check", badID, err)
		}
		if candidate, id, err := splitCityItemRoot(onTable, badID, cityItemLocation{}, worn); err == nil || candidate != nil || id != 0 {
			t.Fatal("attested detached split bypassed exact Item ID check", badID, id, err)
		}
	}
	requireCityMutationUnchanged(t, onTable, wantTable)
	requireCityMutationUnchanged(t, g, before)
}

func TestCityItemSplitClonesRepeatedOccurrencesAndKeepsSurvivors(t *testing.T) {
	g := cityMutationFixture()
	before := g.Clone()
	from := cityItemLocation{PartyID: "alpha", Kind: cityItemPack, Index: 1}
	to := cityItemLocation{PartyID: "beta", Kind: cityItemPack, Index: 1}
	n, id, err := splitCityItemRoot(g, 10, from, to)
	if err != nil || id != 100 || n.NextID != 105 {
		t.Fatal("split allocation", id, err, n)
	}
	want := before.Clone()
	want.NextID = 105
	want.Items = append(want.Items, cityItemTopology{ID: 100, Effects: []sim.SavedObjectID{101, 102, 0, 103}, Spell: 104})
	want.Effects = append(want.Effects, 101, 102, 103)
	want.Spells = append(want.Spells, 104)
	want.Roots[1].Pack = []sim.SavedObjectID{0, 100, 11}
	if !reflect.DeepEqual(n, want) {
		t.Fatal("split lost source aliases, ordered nulls or independent child occurrences", n)
	}
	requireCityMutationUnchanged(t, g, before)
	next, nextID, err := splitCityItemRoot(n, id, to, cityItemLocation{})
	if err != nil || nextID != 105 || next.NextID != 110 || next.Items[5].Spell != 109 ||
		!slices.Equal(next.Items[5].Effects, []sim.SavedObjectID{106, 107, 0, 108}) {
		t.Fatal("a later split reused an issued identity", nextID, err, next)
	}
	if !reflect.DeepEqual(n, want) {
		t.Fatal("second split mutated its input")
	}
}

func TestCityItemSplitPreservesAbsentChildren(t *testing.T) {
	for _, test := range []struct {
		id   sim.SavedObjectID
		from cityItemLocation
		want []sim.SavedObjectID
	}{
		{12, cityItemLocation{PartyID: "alpha", Kind: cityItemPack, Index: 5}, []sim.SavedObjectID{0}},
		{13, cityItemLocation{}, nil},
	} {
		t.Run(fmt.Sprint(test.id), func(t *testing.T) {
			g := cityMutationFixture()
			n, id, err := splitCityItemRoot(g, test.id, test.from, cityItemLocation{})
			if err != nil || id != 100 || n.NextID != 101 || n.Items[4].Spell != 0 || !reflect.DeepEqual(n.Items[4].Effects, test.want) ||
				!slices.Equal(n.Effects, g.Effects) || !slices.Equal(n.Spells, g.Spells) || !reflect.DeepEqual(n.Roots, g.Roots) {
				t.Fatal("absent children acquired nodes or null source roots changed", id, err, n)
			}
		})
	}
}

func TestCityItemMutationsRejectMalformedSourcesAndDestinations(t *testing.T) {
	validFrom := cityItemLocation{PartyID: "alpha", Kind: cityItemPack, Index: 1}
	validTo := cityItemLocation{PartyID: "beta", Kind: cityItemWorn, Index: 0}
	for _, test := range []struct {
		name string
		edit func(*cityObjectTopology, *sim.SavedObjectID, *cityItemLocation, *cityItemLocation)
	}{
		{"zero item", func(_ *cityObjectTopology, id *sim.SavedObjectID, _, _ *cityItemLocation) { *id = 0 }},
		{"missing item", func(_ *cityObjectTopology, id *sim.SavedObjectID, _, _ *cityItemLocation) { *id = 99 }},
		{"wrong node kind", func(_ *cityObjectTopology, id *sim.SavedObjectID, _, _ *cityItemLocation) { *id = 40 }},
		{"stale item", func(_ *cityObjectTopology, id *sim.SavedObjectID, _, _ *cityItemLocation) { *id = 11 }},
		{"null source", func(_ *cityObjectTopology, _ *sim.SavedObjectID, from, _ *cityItemLocation) { from.Index = 0 }},
		{"source underflow", func(_ *cityObjectTopology, _ *sim.SavedObjectID, from, _ *cityItemLocation) { from.Index = -1 }},
		{"source past pack", func(g *cityObjectTopology, _ *sim.SavedObjectID, from, _ *cityItemLocation) {
			from.Index = len(g.Roots[0].Pack)
		}},
		{"source past worn", func(_ *cityObjectTopology, _ *sim.SavedObjectID, from, _ *cityItemLocation) {
			from.Kind, from.Index = cityItemWorn, sim.EquipSlots
		}},
		{"unknown source party", func(_ *cityObjectTopology, _ *sim.SavedObjectID, from, _ *cityItemLocation) { from.PartyID = "alph" }},
		{"empty source party", func(_ *cityObjectTopology, _ *sim.SavedObjectID, from, _ *cityItemLocation) { from.PartyID = "" }},
		{"fake detached owner", func(_ *cityObjectTopology, _ *sim.SavedObjectID, from, _ *cityItemLocation) {
			from.Kind = cityItemDetached
		}},
		{"invalid source kind", func(_ *cityObjectTopology, _ *sim.SavedObjectID, from, _ *cityItemLocation) { from.Kind = 255 }},
		{"destination underflow", func(_ *cityObjectTopology, _ *sim.SavedObjectID, _, to *cityItemLocation) { to.Index = -1 }},
		{"destination past pack", func(g *cityObjectTopology, _ *sim.SavedObjectID, _, to *cityItemLocation) {
			to.Kind, to.Index = cityItemPack, len(g.Roots[1].Pack)+1
		}},
		{"destination past worn", func(_ *cityObjectTopology, _ *sim.SavedObjectID, _, to *cityItemLocation) { to.Index = sim.EquipSlots }},
		{"occupied destination", func(_ *cityObjectTopology, _ *sim.SavedObjectID, _, to *cityItemLocation) { to.Index = 2 }},
		{"unknown destination party", func(_ *cityObjectTopology, _ *sim.SavedObjectID, _, to *cityItemLocation) { to.PartyID = "missing" }},
		{"invalid destination kind", func(_ *cityObjectTopology, _ *sim.SavedObjectID, _, to *cityItemLocation) { to.Kind = 255 }},
		{"detached index", func(_ *cityObjectTopology, _ *sim.SavedObjectID, _, to *cityItemLocation) {
			*to = cityItemLocation{Index: 1}
		}},
		{"bad header", func(g *cityObjectTopology, _ *sim.SavedObjectID, _, _ *cityItemLocation) { g.Version++ }},
		{"zero floor", func(g *cityObjectTopology, _ *sim.SavedObjectID, _, _ *cityItemLocation) { g.NextID = 0 }},
		{"reused floor", func(g *cityObjectTopology, _ *sim.SavedObjectID, _, _ *cityItemLocation) { g.NextID = 41 }},
		{"dangling child", func(g *cityObjectTopology, _ *sim.SavedObjectID, _, _ *cityItemLocation) { g.Items[0].Effects[0] = 99 }},
		{"wrong child kind", func(g *cityObjectTopology, _ *sim.SavedObjectID, _, _ *cityItemLocation) { g.Items[0].Spell = 30 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			g, id, from, to := cityMutationFixture(), sim.SavedObjectID(10), validFrom, validTo
			test.edit(g, &id, &from, &to)
			before := g.Clone()
			if n, err := moveCityItemRoot(g, id, from, to); err == nil || n != nil {
				t.Fatal("invalid whole move produced a candidate", err, n)
			}
			requireCityMutationUnchanged(t, g, before)
			if n, id, err := splitCityItemRoot(g, id, from, to); err == nil || n != nil || id != 0 {
				t.Fatal("invalid split produced a candidate or issued ID", id, err, n)
			}
			requireCityMutationUnchanged(t, g, before)
		})
	}
	if n, err := moveCityItemRoot(nil, 10, validFrom, validTo); err == nil || n != nil {
		t.Fatal("absent topology moved an item")
	}
	if n, id, err := splitCityItemRoot(nil, 10, validFrom, validTo); err == nil || n != nil || id != 0 {
		t.Fatal("absent topology split an item")
	}
}

func TestCityItemSplitAllocationFailureIsAtomicAtEveryChild(t *testing.T) {
	from := cityItemLocation{PartyID: "alpha", Kind: cityItemPack, Index: 1}
	for remaining := sim.SavedObjectID(0); remaining < 5; remaining++ {
		g := cityMutationFixture()
		g.NextID = ^sim.SavedObjectID(0) - remaining
		before := g.Clone()
		if n, id, err := splitCityItemRoot(g, 10, from, cityItemLocation{}); err == nil || n != nil || id != 0 {
			t.Fatal("exhausted split returned partial candidate", remaining, id, err, n)
		}
		requireCityMutationUnchanged(t, g, before)
	}
	g := cityMutationFixture()
	g.NextID = ^sim.SavedObjectID(0) - 5
	n, id, err := splitCityItemRoot(g, 10, from, cityItemLocation{})
	if err != nil || id != g.NextID || n.NextID != ^sim.SavedObjectID(0) {
		t.Fatal("last complete allocation wrapped or skipped the floor", id, err, n)
	}
	before := n.Clone()
	if next, nextID, err := splitCityItemRoot(n, 10, from, cityItemLocation{}); err == nil || next != nil || nextID != 0 {
		t.Fatal("exhausted floor reused a successfully issued ID", nextID, err)
	}
	requireCityMutationUnchanged(t, n, before)
}

func TestCityBookMutationSeparatesSharedSpellAndKeepsUniqueIdentity(t *testing.T) {
	for _, test := range []struct {
		name string
		slot int
		old  sim.SavedObjectID
		want sim.SavedObjectID
	}{
		{"null slot", 1, 0, 100},
		{"shared weapon and book", 0, 40, 100},
		{"unique slot", 3, 41, 41},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := cityMutationFixture()
			before := g.Clone()
			n, id, err := prepareCityBookMutation(g, "alpha", test.slot, test.old)
			if err != nil || id != test.want {
				t.Fatal("book mutation identity", id, err)
			}
			want := before.Clone()
			want.Books[0].Slots[test.slot] = test.want
			if test.want == 100 {
				want.Spells = append(want.Spells, 100)
				want.NextID = 101
			}
			if !reflect.DeepEqual(n, want) {
				t.Fatal("book mutation changed other book/weapon roots or values", n)
			}
			requireCityMutationUnchanged(t, g, before)
			next, again, err := prepareCityBookMutation(n, "alpha", test.slot, id)
			if err != nil || again != id || !reflect.DeepEqual(next, n) {
				t.Fatal("independent spell acquired another unnecessary node", again, err, next)
			}
			n.Books[0].PartyID[0], n.Roots[0].PartyID[0] = 'X', 'Y'
			n.Items[0].Effects[0] = 0
			requireCityMutationUnchanged(t, g, before)
		})
	}
	// A second book slot alone is enough to keep the old Spell alive.
	g := cityMutationFixture()
	g.Items[0].Spell, g.Items[1].Spell, g.Books[1].Slots[0] = 0, 0, 0
	n, id, err := prepareCityBookMutation(g, "alpha", 0, 40)
	if err != nil || id != 100 || n.Books[0].Slots[2] != 40 || !slices.Contains(n.Spells, 40) {
		t.Fatal("repeated book edge was changed or its child pruned", id, err, n)
	}
	// A retained detached weapon also owns its Spell child.
	g = cityMutationFixture()
	g.Roots = []cityPartyObjectRoots{{PartyID: []byte("alpha")}}
	g.Books = []cityBookTopology{{PartyID: []byte("alpha")}}
	g.Books[0].Slots[0] = 40
	n, id, err = prepareCityBookMutation(g, "alpha", 0, 40)
	if err != nil || id != 100 || n.Items[0].Spell != 40 {
		t.Fatal("detached weapon child lost identity", id, err, n)
	}
}

func TestCityBookMutationCreatesOnlyExplicitPartyBook(t *testing.T) {
	g := cityMutationFixture()
	g.Books = g.Books[:1]
	n, id, err := prepareCityBookMutation(g, "beta", 27, 0)
	if err != nil || id != 100 || len(n.Books) != 2 || string(n.Books[1].PartyID) != "beta" || n.Books[1].Slots[27] != 100 {
		t.Fatal("missing book did not use the explicit party", id, err, n)
	}
	for _, ref := range n.Books[1].Slots[:27] {
		if ref != 0 {
			t.Fatal("creating one book slot filled another null root")
		}
	}
	if len(g.Books) != 1 || g.NextID != 100 {
		t.Fatal("book creation mutated input")
	}
}

func TestCityBookMutationRejectsStaleRootsBoundsAndExhaustion(t *testing.T) {
	for _, test := range []struct {
		name, party string
		slot        int
		expected    sim.SavedObjectID
		edit        func(*cityObjectTopology)
	}{
		{"underflow", "alpha", -1, 40, nil},
		{"past end", "alpha", 28, 40, nil},
		{"missing party", "absent", 0, 40, nil},
		{"empty party", "", 0, 40, nil},
		{"stale spell", "alpha", 0, 41, nil},
		{"stale null", "alpha", 0, 0, nil},
		{"missing spell", "alpha", 0, 99, nil},
		{"wrong node kind", "alpha", 0, 10, nil},
		{"absent book stale source", "beta", 0, 40, func(g *cityObjectTopology) { g.Books = g.Books[:1] }},
		{"malformed graph", "alpha", 0, 40, func(g *cityObjectTopology) { g.Books[0].Slots[0] = 30 }},
		{"exhausted COW", "alpha", 0, 40, func(g *cityObjectTopology) { g.NextID = ^sim.SavedObjectID(0) }},
		{"exhausted new node", "alpha", 1, 0, func(g *cityObjectTopology) { g.NextID = ^sim.SavedObjectID(0) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := cityMutationFixture()
			if test.edit != nil {
				test.edit(g)
			}
			before := g.Clone()
			if n, id, err := prepareCityBookMutation(g, test.party, test.slot, test.expected); err == nil || n != nil || id != 0 {
				t.Fatal("invalid book mutation produced candidate", id, err, n)
			}
			requireCityMutationUnchanged(t, g, before)
		})
	}
	if n, id, err := prepareCityBookMutation(nil, "alpha", 0, 0); err == nil || n != nil || id != 0 {
		t.Fatal("absent topology created book")
	}
	g := cityMutationFixture()
	g.NextID = ^sim.SavedObjectID(0)
	if n, id, err := prepareCityBookMutation(g, "alpha", 3, 41); err != nil || id != 41 || n.NextID != g.NextID {
		t.Fatal("unique book spuriously needs an allocation", id, err, n)
	}
}

func TestCityObjectMutationValidatesCandidateBudgetBeforeReturn(t *testing.T) {
	g := &cityObjectTopology{
		Version: cityObjectTopologyVersion, NextID: 4,
		Items:   []cityItemTopology{{ID: 1, Effects: []sim.SavedObjectID{2}, Spell: 3}},
		Effects: []sim.SavedObjectID{2}, Spells: []sim.SavedObjectID{3},
		Roots: []cityPartyObjectRoots{{PartyID: []byte("a"), Pack: make([]sim.SavedObjectID, sim.MaxSavedObjects-18)}},
	}
	if err := g.Validate(); err != nil {
		t.Fatal("fixture must fit the input budget", err)
	}
	before := g.Clone()
	if n, err := moveCityItemRoot(g, 1, cityItemLocation{}, cityItemLocation{PartyID: "a", Kind: cityItemPack}); err == nil || n != nil {
		t.Fatal("move returned oversized roots", err)
	}
	requireCityMutationUnchanged(t, g, before)
	if n, id, err := splitCityItemRoot(g, 1, cityItemLocation{}, cityItemLocation{}); err == nil || n != nil || id != 0 {
		t.Fatal("split returned oversized children", id, err)
	}
	requireCityMutationUnchanged(t, g, before)
	if n, id, err := prepareCityBookMutation(g, "a", 0, 0); err == nil || n != nil || id != 0 {
		t.Fatal("book edit returned oversized roots", id, err)
	}
	requireCityMutationUnchanged(t, g, before)
}
