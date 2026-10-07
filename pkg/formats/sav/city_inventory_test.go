package sav

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func citySaleFixture(t *testing.T) (*CityProvenance, uint32) {
	t.Helper()
	f, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	var id uint32
	for _, c := range p.Roster() {
		if c.Hero {
			id = c.Identity
		}
	}
	u := p.document.objects[p.characterSourceIndex[id]].unit
	u.containerFlag, u.containerTails = 1, [2]uint32{1234, 35}
	item := &cityObject{sourceIndex: 100, class: "Item", item: &cityItem{
		token: cityTestToken(0x10293847, 0, 13),
		// code=0xe0d, count=5, kind=3, +45=71,+46=72,+48=0x7354,
		// per-unit weight=7, unknown +47=0x99. Literal serializer order.
		fields: []byte{13, 14, 5, 0, 3, 71, 72, 0x54, 0x73, 7, 0, 0x99},
	}}
	binary.LittleEndian.PutUint32(item.item.token[25:], 5)
	u.container = []*cityObject{item}
	p.document.objects[100] = item
	return p, id
}

func TestCitySalesPreserveLiteralRemainingObjectAndStoredLoad(t *testing.T) {
	p, id := citySaleFixture(t)
	before := p.Data()
	u := CityUpdate{Label: []byte("sale"), Money: 131, Characters: cityTestUpdates(p.Roster())}
	for i := range u.Characters {
		if u.Characters[i].Identity == id {
			u.Characters[i].Sales = []CityItemSale{{0, 3}}
		}
	}
	raw, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	q, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Data(), before) {
		t.Fatal("sale mutated source provenance")
	}
	var got *cityItem
	for _, o := range q.document.objects {
		if o.item != nil {
			if got != nil {
				t.Fatal("invented sold split object")
			}
			got = o.item
		}
	}
	if got == nil || !bytes.Equal(got.fields, []byte{13, 14, 2, 0, 3, 71, 72, 0x54, 0x73, 7, 0, 0x99}) {
		t.Fatalf("remaining item %+v", got)
	}
	for _, c := range q.Roster() {
		if c.Hero {
			actor := q.document.objects[q.characterSourceIndex[c.Identity]].unit
			if actor.containerTails != [2]uint32{1234, 14} {
				t.Fatal(actor.containerTails)
			}
			items, err := q.Inventory(c.Identity)
			if err != nil || len(items) != 1 || items[0].Piece.Stack != 2 || items[0].Piece.Price != 5 || items[0].Weight != 7 || !items[0].Exclusive {
				t.Fatal(items, err)
			}
		}
	}
	// The next take is relative to the new source, not an obsolete archive tag.
	for i := range u.Characters {
		u.Characters[i].Sales = nil
	}
	u.Characters = cityTestUpdates(q.Roster())
	u.Money = 134
	for _, c := range q.Roster() {
		if c.Hero {
			id = c.Identity
		}
	}
	for i := range u.Characters {
		if u.Characters[i].Identity == id {
			u.Characters[i].Sales = []CityItemSale{{0, 1}}
		}
	}
	raw, err = q.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	f, _ = Open(raw)
	r, _ := f.CityProvenance()
	for _, c := range r.Roster() {
		if c.Hero {
			v := r.document.objects[r.characterSourceIndex[c.Identity]].unit
			if v.containerTails[1] != 7 || binary.LittleEndian.Uint16(v.container[0].item.fields[2:]) != 1 {
				t.Fatal("later sale lost remainder")
			}
		}
	}
}

type cityTestNativeBinding struct {
	ID        uint32
	Object    uint16
	Structure bool
	Missing   bool
}

type cityTestNativeActions struct {
	Version  uint32
	Bindings []cityTestNativeBinding
}

func addCityTestBoundSaleItem(t *testing.T, p *CityProvenance, owner uint32, objectIndex uint16, itemIdentity uint32) {
	t.Helper()
	actor := p.document.objects[p.characterSourceIndex[owner]]
	if actor == nil || actor.unit == nil || p.document.objects[objectIndex] != nil {
		t.Fatal("city sale fixture actor or item index is invalid", owner, objectIndex)
	}
	fields := []byte{0x1c, 0x0e, 1, 0, 3, 71, 72, 0x54, 0x73, 7, 0, 0x99}
	token := cityTestToken(itemIdentity, 0, 13)
	binary.LittleEndian.PutUint32(token[25:], 1)
	item := &cityObject{sourceIndex: objectIndex, class: "Item", item: &cityItem{token: token, fields: fields}}
	actor.unit.containerFlag = 1
	actor.unit.containerTails[1] += 7
	actor.unit.container = append(actor.unit.container, item)
	p.document.objects[objectIndex] = item
}

func cityTestCurrentActorBinding(t *testing.T, p *CityProvenance, identity uint32) uint16 {
	t.Helper()
	index := cityObjectLocalOrdinals(p.document)[p.characterSourceIndex[identity]]
	if index == 0 {
		t.Fatal("city sale fixture actor has no document index", identity)
	}
	leaf, err := json.Marshal(cityTestNativeActions{Version: 1, Bindings: []cityTestNativeBinding{{
		ID: 1, Object: index,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	data := p.Data()
	state := documentStateData(data.State)
	if err := SetNativeActions(&state, leaf); err != nil {
		t.Fatal(err)
	}
	data.State.RootKind = state.RootKind
	data.State.DirectoryRecords = state.DirectoryRecords
	data.State.ValueRecords = state.ValueRecords
	next, err := cityStateFromData(data.State, CityDataVersion)
	if err != nil {
		t.Fatal(err)
	}
	p.state = next
	return index
}

func cityTestSaleWithCurrentBinding(t *testing.T, saleOwner string) (*CityProvenance, uint32, uint16, bool) {
	t.Helper()
	file, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	var reniesta, danath uint32
	for _, actor := range p.Roster() {
		switch actor.Name {
		case "Reniesta":
			reniesta = actor.Identity
		case "Danath":
			danath = actor.Identity
		}
	}
	if reniesta == 0 || danath == 0 {
		t.Fatal("city sale fixture roster changed", p.Roster())
	}
	seller := reniesta
	if saleOwner == "Danath" {
		seller = danath
	}
	itemIndex := uint16(3) // before both actors
	if saleOwner == "Danath" {
		itemIndex = 6 // no-shift
	}
	addCityTestBoundSaleItem(t, p, seller, itemIndex, 0x10293847)
	if saleOwner == "Reniesta" {
		// Survives at the stale index post-compaction.
		addCityTestBoundSaleItem(t, p, danath, 7, 0x20394857)
	}
	oldIndex := cityTestCurrentActorBinding(t, p, danath)
	return p, seller, oldIndex, saleOwner == "Reniesta"
}

func TestCitySaleRemapsCurrentActorBindingAcrossReindex(t *testing.T) {
	for _, tc := range []struct {
		name      string
		saleOwner string
		shifted   bool
	}{
		{name: "sale_before_bound_actor", saleOwner: "Reniesta", shifted: true},
		{name: "sale_after_bound_actor_negative_control", saleOwner: "Danath"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, seller, oldIndex, shifted := cityTestSaleWithCurrentBinding(t, tc.saleOwner)
			if shifted != tc.shifted {
				t.Fatal("sale fixture shift flag", shifted, "want", tc.shifted)
			}
			updates := cityTestUpdates(p.Roster())
			for i := range updates {
				if updates[i].Identity == seller {
					updates[i].Sales = []CityItemSale{{Position: 0, Quantity: 1}}
				}
			}
			raw, err := p.Marshal(CityUpdate{Label: []byte("physical loss"), Money: 123, Characters: updates})
			if err != nil {
				t.Fatal("marshal physical loss", err)
			}
			data, err := DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			leaf, present, err := NativeActions(data.State)
			if err != nil || !present {
				t.Fatal("current actor binding lost", present, err)
			}
			var actions cityTestNativeActions
			if err := json.Unmarshal(leaf, &actions); err != nil || len(actions.Bindings) != 1 {
				t.Fatal("current actor binding payload", err, actions)
			}
			newIndex := actions.Bindings[0].Object
			if shifted && newIndex == oldIndex || !shifted && newIndex != oldIndex {
				t.Fatalf("current actor index = %d after sale, before %d, shifted=%v", newIndex, oldIndex, shifted)
			}
			characters, err := ReadDocumentCharacters(data, []uint16{newIndex})
			if err != nil || len(characters) != 1 || characters[0].Character.Name != "Danath" {
				t.Fatal("current binding no longer resolves to Danath", characters, err)
			}
			if shifted {
				if _, err := ReadDocumentCharacters(data, []uint16{oldIndex}); err == nil || !strings.Contains(err.Error(), "has class Item") {
					t.Fatal("stale-index negative control did not identify the surviving Item", err)
				}
			}
		})
	}
}

func TestCityInventoryProjectsSignedInstanceWeight(t *testing.T) {
	for _, tc := range []struct {
		bytes [2]byte
		want  int16
	}{{[2]byte{0, 0}, 0}, {[2]byte{0xfd, 0xff}, -3}, {[2]byte{7, 0}, 7}} {
		p, id := citySaleFixture(t)
		copy(p.document.objects[100].item.fields[9:11], tc.bytes[:])
		got, err := p.Inventory(id)
		if err != nil || len(got) != 1 || got[0].Piece.Weight != tc.want || got[0].Weight != tc.want || got[0].Piece.Stack != 5 {
			t.Fatalf("signed city item projection: %+v %v; want %d", got, err, tc.want)
		}
	}
}

func TestCityInventoryProjectsLiteralEquipmentTailsAndOwnedSpell(t *testing.T) {
	attack := [24]byte{11, 0, 13, 0, 14, 0, 15, 0, 16, 0, 17, 0, 18, 0, 7, 9, 3, 6, 8, 2, 4, 1, 93, 94}
	defence := [22]byte{19, 0, 21, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}
	for _, class := range []string{"Weapon", "Armor", "Shield"} {
		t.Run(class, func(t *testing.T) {
			p, id := citySaleFixture(t)
			o := p.document.objects[100]
			o.class = class
			want := Piece{Class: class, Code: 0x0e0d, Row: 13, Stack: 5, Kind: 3, Price: 5, Weight: 7}
			switch class {
			case "Weapon":
				o.item.derived = append(append(append([]byte(nil), attack[:]...), defence[:]...), 4)
				o.item.weaponExtra = &cityObject{class: "Spell", sourceIndex: 101, spell: &citySpell{fields: []byte{5, 17, 9, 0xfd, 0xff, 0x44, 0x33, 0x22, 0x11}}}
				want.W52, want.W6A, want.W50 = attack, defence, 4
				want.WeaponSpell = &SavedSpell{ID: 5, Range: 17, Defensive: 9, ManaCost: 65533, ArchiveIndex: 101, Key: 0x11223344}
			case "Armor":
				o.item.derived = append(append([]byte(nil), defence[:]...), 7)
				want.A52, want.A50 = defence, 7
			case "Shield":
				o.item.derived = append([]byte(nil), defence[:]...)
				want.S50 = defence
			}
			got, err := p.Inventory(id)
			if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0].Piece, want) {
				t.Fatalf("literal city equipment projection %+v: %v", got, err)
			}
			o.item.derived = o.item.derived[:len(o.item.derived)-1]
			if got, err := p.Inventory(id); err == nil || got != nil {
				t.Fatal("truncated equipment tail returned partial inventory")
			}
		})
	}
}

func TestCitySalesRefuseAliasesAndInvalidOperationsAtomically(t *testing.T) {
	for _, test := range []struct {
		name  string
		sales []CityItemSale
		alias bool
	}{
		{"zero", []CityItemSale{{0, 0}}, false}, {"overdraw", []CityItemSale{{0, 6}}, false},
		{"foreign", []CityItemSale{{1, 1}}, false}, {"cumulative", []CityItemSale{{0, 3}, {0, 3}}, false},
		{"alias", []CityItemSale{{0, 1}}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			p, id := citySaleFixture(t)
			if test.alias {
				for _, c := range p.Roster() {
					if !c.Hero {
						p.document.objects[p.characterSourceIndex[c.Identity]].unit.reference68 = p.document.objects[100]
					}
				}
			}
			before := p.Data()
			u := CityUpdate{Characters: cityTestUpdates(p.Roster())}
			for i := range u.Characters {
				if u.Characters[i].Identity == id {
					u.Characters[i].Sales = test.sales
				}
			}
			if _, err := p.Marshal(u); err == nil {
				t.Fatal("invalid sale accepted")
			}
			if !reflect.DeepEqual(before, p.Data()) {
				t.Fatal("refusal mutated source")
			}
		})
	}
}

func TestCitySalesWholeObjectAbsentAndNegativeWeightUsesStoredDword(t *testing.T) {
	p, id := citySaleFixture(t)
	binary.LittleEndian.PutUint16(p.document.objects[100].item.fields[9:], 0xfffd) // signed -3
	u := p.document.objects[p.characterSourceIndex[id]].unit
	u.containerTails = [2]uint32{0xffffffff, 100} // deliberately not the item sum
	update := CityUpdate{Characters: cityTestUpdates(p.Roster())}
	for i := range update.Characters {
		if update.Characters[i].Identity == id {
			update.Characters[i].Sales = []CityItemSale{{0, 5}}
		}
	}
	raw, err := p.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := Open(raw)
	q, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range q.document.objects {
		if o.item != nil {
			t.Fatal("sold object remains reachable")
		}
	}
	for _, c := range q.Roster() {
		if c.Hero {
			v := q.document.objects[q.characterSourceIndex[c.Identity]].unit
			if len(v.container) != 0 || v.containerTails != [2]uint32{0xffffffff, 115} {
				t.Fatal(v.containerTails)
			}
		}
	}
}

func TestCitySalesRemintOnlySurvivingReachableObjects(t *testing.T) {
	for _, quantity := range []uint16{3, 5} {
		p, id := citySaleFixture(t)
		// A child reached by the sold object only disappears with its owner;
		// a child shared by a surviving actor remains exactly once.
		effect := &cityObject{sourceIndex: 101, class: "Effect", effect: &cityEffect{token: cityTestToken(0x88992211, 0, 0), fields: []byte{1, 2, 3, 4, 5, 6, 0}}}
		p.document.objects[101] = effect
		p.document.objects[100].item.effects = []*cityObject{effect}
		for _, c := range p.Roster() {
			if !c.Hero {
				p.document.objects[p.characterSourceIndex[c.Identity]].unit.effects = []*cityObject{effect}
			}
		}
		u := CityUpdate{Label: []byte("stable identities"), Money: 123, Characters: cityTestUpdates(p.Roster())}
		for i := range u.Characters {
			if u.Characters[i].Identity == id {
				u.Characters[i].Sales = []CityItemSale{{0, quantity}}
			}
		}
		raw, err := p.Marshal(u)
		if err != nil {
			t.Fatal(err)
		}
		f, err := Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		q, err := f.CityProvenance()
		if err != nil {
			t.Fatal(err)
		}
		u.Characters = cityTestUpdates(q.Roster())
		back, err := q.Marshal(u)
		if err != nil || !bytes.Equal(raw, back) {
			t.Fatal("surviving identity changed on next SAVE", quantity, err)
		}
		children := 0
		for _, o := range q.document.objects {
			if o.effect != nil {
				children++
			}
		}
		if children != 1 {
			t.Fatal("shared reachable child dropped/duplicated", children)
		}
	}
}
