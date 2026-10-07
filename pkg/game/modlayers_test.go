package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

func layerRow(key, slot, standIn, placement, anchor string) mod.ItemRow {
	r := testRow(key, slot, standIn)
	r.Layer = placement
	r.Anchor, r.AnchorNo = anchor, mod.AnchorSlots[anchor]
	return r
}

func TestSetModItemsCarriesTheLayerPlacement(t *testing.T) {
	tab := fixtureItemTable(30, 5, 5)
	under := layerRow("g", "body", "Soft Mail", mod.LayerUnder, "body")
	cloak := layerRow("c", "shield", "Buckler", mod.LayerOver, "body")
	plain := testRow("p", "head", "Cap")
	if err := itemFront(tab).SetModItems(mod.ItemData{Rows: []mod.ItemRow{under, cloak, plain}}); err != nil {
		t.Fatal(err)
	}
	items := tab.Mods.Items
	if len(items) != 3 {
		t.Fatalf("%+v", items)
	}
	if items[0].Layer != mapload.LayerUnder || items[0].Anchor != uint8(mod.AnchorSlots["body"]) ||
		items[1].Layer != mapload.LayerOver || items[1].Anchor != uint8(mod.AnchorSlots["body"]) ||
		items[2].Layer != 0 || items[2].Anchor != 0 {
		t.Fatalf("placement %+v", items)
	}
	if got, ok := mapload.LayerItem(tab, items[1].Code); !ok || got.Key != "c" {
		t.Fatalf("LayerItem of the cloak: %+v %v", got, ok)
	}
	if _, ok := mapload.LayerItem(tab, items[2].Code); ok {
		t.Fatal("an ordinary item answers as a layer")
	}
	if _, ok := mapload.LayerItem(tab, 0x0a0f); ok {
		t.Fatal("an original item answers as a layer")
	}
	if _, ok := mapload.LayerItem(nil, items[0].Code); ok {
		t.Fatal("a nil table answers a layer")
	}
}

func TestActiveLayersKeepsOnlyWhatThePackBacks(t *testing.T) {
	for _, c := range []struct {
		name            string
		layers, carried []uint16
		want            []uint16
	}{
		{"none worn", nil, []uint16{1, 2}, nil},
		{"all backed", []uint16{1, 2}, []uint16{2, 1, 9}, []uint16{1, 2}},
		{"one sold", []uint16{1, 2}, []uint16{2}, []uint16{2}},
		{"every unit counted once", []uint16{1, 1}, []uint16{1}, []uint16{1}},
		{"two units two layers", []uint16{1, 1}, []uint16{1, 1}, []uint16{1, 1}},
		{"empty pack", []uint16{1}, nil, []uint16{}},
	} {
		got := mapload.ActiveLayers(c.layers, c.carried)
		if !slices.Equal(got, c.want) || (c.want == nil) != (got == nil) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestWearLayerReplacesTheLayerAtTheSameAnchorAndDepth(t *testing.T) {
	tab := fixtureItemTable(30, 5, 5)
	rows := []mod.ItemRow{
		layerRow("a", "body", "Soft Mail", mod.LayerUnder, "body"),
		layerRow("b", "body", "Soft Mail", mod.LayerUnder, "body"),
		layerRow("c", "shield", "Buckler", mod.LayerOver, "body"),
		layerRow("d", "shield", "Buckler", mod.LayerOver, "legs"),
	}
	if err := itemFront(tab).SetModItems(mod.ItemData{Rows: rows}); err != nil {
		t.Fatal(err)
	}
	it := tab.Mods.Items
	worn := wearLayer(nil, it[0], tab)
	worn = wearLayer(worn, it[2], tab)
	worn = wearLayer(worn, it[3], tab)
	if !slices.Equal(worn, []uint16{it[0].Code, it[2].Code, it[3].Code}) {
		t.Fatalf("three different places: %v", worn)
	}
	worn = wearLayer(worn, it[1], tab)
	if !slices.Equal(worn, []uint16{it[2].Code, it[3].Code, it[1].Code}) {
		t.Fatalf("a second under layer at the body replaces the first: %v", worn)
	}
}

func TestTheModMarkCarriesLayersAndRefusesBadOnes(t *testing.T) {
	doc, err := sav.DecodeDocumentData(modTestSAV(t))
	if err != nil {
		t.Fatal(err)
	}
	set := modTestSet(150)
	layers := []modMarkLayer{{Member: "hero", Code: 0x071f}, {Member: "hero", Code: 0x0210}}
	if err := markDocumentForMods(&doc, set, nil, layers); err != nil {
		t.Fatal(err)
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present || !slices.Equal(mark.Layers, layers) {
		t.Fatalf("the mark holds %+v %v %v", mark.Layers, present, err)
	}
	plain, err := sav.DecodeDocumentData(modTestSAV(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := markDocumentForMods(&plain, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	if m, _, _ := readModMark(plain); len(m.Layers) != 0 {
		t.Fatalf("layers recorded without any: %+v", m.Layers)
	}

	items := []mapload.ModItem{
		{Code: 0x071f, Layer: mapload.LayerUnder, Anchor: 7},
		{Code: 0x0210, Layer: mapload.LayerOver, Anchor: 7},
		{Code: 0x0720},
	}
	party := []mapload.PartyMember{{ID: "hero"}, {ID: "player:ann"}}
	if err := applyModLayers(party, layers, items); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(party[0].Layers, []uint16{0x071f, 0x0210}) || len(party[1].Layers) != 0 {
		t.Fatalf("applied %+v", party)
	}
	for _, c := range []struct {
		name   string
		layers []modMarkLayer
		want   string
	}{
		{"an item that is no layer", []modMarkLayer{{Member: "hero", Code: 0x0720}}, "no active mod adds as a layer"},
		{"an item no mod adds", []modMarkLayer{{Member: "hero", Code: 0x0a0f}}, "no active mod adds as a layer"},
		{"a member the save lacks", []modMarkLayer{{Member: "ghost", Code: 0x071f}}, `"ghost"`},
	} {
		err := applyModLayers([]mapload.PartyMember{{ID: "hero"}}, c.layers, items)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if err := applyModLayers(nil, nil, nil); err != nil {
		t.Fatalf("no layers: %v", err)
	}
}

func TestModMarkLayersListsEveryWornLayer(t *testing.T) {
	tab := fixtureItemTable(30, 5, 5)
	got := modMarkLayers([]mapload.PartyMember{
		{ID: "hero", Layers: []uint16{1, 2}, Carried: []uint16{1, 2}},
		{ID: "b"},
		{ID: "c", Layers: []uint16{3, 4}, Carried: []uint16{3}},
	}, tab)
	want := []modMarkLayer{{"hero", 1}, {"hero", 2}, {"c", 3}}
	if !slices.Equal(got, want) {
		t.Fatalf("%+v", got)
	}
	if modMarkLayers(nil, tab) != nil {
		t.Fatal("layers of no party")
	}
}

func TestClonedPartyOwnsItsLayers(t *testing.T) {
	in := []mapload.PartyMember{{ID: "hero", Layers: []uint16{1, 2}}}
	out := mapload.CloneParty(in)
	out[0].Layers[0] = 9
	if in[0].Layers[0] != 1 {
		t.Fatal("a clone shares the layers of its source")
	}
}
