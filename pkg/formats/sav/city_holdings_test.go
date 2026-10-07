package sav

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func cityHoldingsFixture(t *testing.T) *CityItemGraph {
	t.Helper()
	// Independent concrete Item -> two distinct equal-valued Effects and a
	// Weapon -> owned Spell. Nonzero retained fields make stale defaults fail.
	item := func(class string, key uint32) *cityObject {
		fields := make([]byte, 12)
		binary.LittleEndian.PutUint16(fields, 0x0111)
		binary.LittleEndian.PutUint16(fields[2:], 3)
		fields[4], fields[5], fields[6], fields[11] = 2, 19, 23, 29
		binary.LittleEndian.PutUint16(fields[7:], 0x3125)
		binary.LittleEndian.PutUint16(fields[9:], 0xfff9)
		return &cityObject{class: class, item: &cityItem{token: cityTestToken(key, 0, 3), fields: fields}}
	}
	a := item("Weapon", 0x71000000)
	a.item.derived = make([]byte, 47)
	a.item.derived[3], a.item.derived[27], a.item.derived[46] = 41, 53, 7
	for i := 0; i < 2; i++ {
		a.item.effects = append(a.item.effects, &cityObject{class: "Effect", effect: &cityEffect{token: cityTestToken(0x71000010+uint32(i)*16, 0, 0), fields: []byte{4, 2, 3, 5, 7, 11, 0}}})
	}
	s := make([]byte, 9)
	s[0], s[1], s[2] = 12, 8, 1
	binary.LittleEndian.PutUint16(s[3:], 17)
	binary.LittleEndian.PutUint32(s[5:], 0x71000030)
	a.item.weaponExtra = &cityObject{class: "Spell", spell: &citySpell{fields: s}}
	w := newCityArchiveWriter(nil)
	if err := w.reference(a); err != nil {
		t.Fatal(err)
	}
	roots := readArchive1115(t, w.b, 1, true)
	b := documentDataBuilder{ids: map[*Record]uint16{}, inline: map[*Record]bool{}}
	ref, err := b.ref(roots[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	return &CityItemGraph{Present: true, InsertIndex: 9, Accumulator: -17, Objects: b.objects, Inventory: []uint16{ref}}
}

func TestCityHoldings1172AtomicGraftAndPrune(t *testing.T) {
	raw := cityTestSource(t)
	f, err := Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	before := p.Data()
	u := CityUpdate{Label: []byte("current pack"), Money: 100}
	for _, c := range p.Roster() {
		u.Characters = append(u.Characters, CityCharacterUpdate{Identity: c.Identity, Name: c.Name, Stats: c.Stats, SkillLevels: c.SkillLevels, SkillXP: c.SkillXP, Experience: c.Experience})
	}
	g := cityHoldingsFixture(t)
	u.Characters[0].Holdings = g
	encoded, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Data(), before) {
		t.Fatal("current graft rewrote immutable source")
	}
	opened, err := Open(encoded)
	if err != nil {
		t.Fatal(err)
	}
	current, err := opened.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := current.DocumentData()
	if err != nil || doc.World != nil {
		t.Fatal("city was fabricated as a world", err)
	}
	var weapon *DocumentRecordData
	var effectCount, spellCount int
	for i := range doc.Objects {
		r := &doc.Objects[i]
		switch r.Class {
		case "Weapon":
			weapon = r
		case "Effect":
			effectCount++
		case "Spell":
			spellCount++
		}
	}
	if weapon == nil || effectCount != 2 || spellCount != 1 {
		t.Fatal("current ownership graph lost nodes")
	}
	if *documentValue1115(t, weapon, "F42") != 3 || *documentValue1115(t, weapon, "F45") != 19 || *documentValue1115(t, weapon, "F4A") != 0xfff9 {
		t.Fatal("current retained item fields lost")
	}
	refs := *documentRefsForField(t, weapon, "Effects")
	if len(refs) != 2 || refs[0] == refs[1] {
		t.Fatal("distinct equal-valued child identities collapsed")
	}
	_, err = p.DocumentData(CityCharacterUpdate{Identity: u.Characters[0].Identity, Holdings: g})
	if err != nil {
		t.Fatal(err)
	}
	g.Inventory = []uint16{uint16(len(g.Objects) + 1)}
	if b, err := p.Marshal(u); err == nil || b != nil {
		t.Fatal("missing root produced output")
	}
	if !reflect.DeepEqual(p.Data(), before) {
		t.Fatal("failed graft mutated source")
	}
	if bytes.Equal(encoded, raw) {
		t.Fatal("mutation did not change output")
	}
	second := CityUpdate{Label: []byte("empty current pack"), Money: 100}
	for _, c := range current.Roster() {
		second.Characters = append(second.Characters, CityCharacterUpdate{Identity: c.Identity, Name: c.Name, Stats: c.Stats, SkillLevels: c.SkillLevels, SkillXP: c.SkillXP, Experience: c.Experience})
	}
	second.Characters[0].Holdings = &CityItemGraph{Present: true, InsertIndex: 9}
	b, err := current.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	d, err := DecodeDocumentData(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range d.Objects {
		if r.Class == "Weapon" || r.Class == "Effect" || r.Class == "Spell" {
			t.Fatal("second SAV retained the removed Item/child graph", r.Class)
		}
	}
}

func TestCityHoldings1172RefusesSingleCauseGraphLosses(t *testing.T) {
	for name, edit := range map[string]func(*CityItemGraph){
		"omitted item":             func(g *CityItemGraph) { g.Inventory = nil },
		"foreign root":             func(g *CityItemGraph) { g.Inventory[0] = 65535 },
		"duplicate item ownership": func(g *CityItemGraph) { g.Inventory = append(g.Inventory, g.Inventory[0]) },
		"collapsed equal Effects": func(g *CityItemGraph) {
			refs := documentRefsForField(t, &g.Objects[g.Inventory[0]-1], "Effects")
			(*refs)[1] = (*refs)[0]
		},
		"stale owned Spell": func(g *CityItemGraph) {
			r := &g.Objects[g.Inventory[0]-1]
			(*documentRefsForField(t, r, "WeaponSpell"))[0] = (*documentRefsForField(t, r, "Effects"))[0]
		},
		"orphan object":     func(g *CityItemGraph) { g.Objects = append(g.Objects, g.Objects[1]) },
		"missing container": func(g *CityItemGraph) { g.Present = false },
	} {
		t.Run(name, func(t *testing.T) {
			g := cityHoldingsFixture(t)
			if err := ValidateCityItemGraph(g); err != nil {
				t.Fatal("unmodified control is invalid", err)
			}
			edit(g)
			if err := ValidateCityItemGraph(g); err == nil {
				t.Fatal("single-cause graph loss was admitted")
			}
		})
	}
}

func TestCityHoldings1172RefusesOutputBeyondColdImportBounds(t *testing.T) {
	for _, extent := range []string{"expanded source loadout", "aggregate city objects"} {
		t.Run(extent, func(t *testing.T) {
			f, err := Open(cityTestSource(t))
			if err != nil {
				t.Fatal(err)
			}
			p, err := f.CityProvenance()
			if err != nil {
				t.Fatal(err)
			}
			before := p.Data()
			g := cityHoldingsFixture(t)
			u := CityUpdate{Label: []byte("oversize current pack")}
			for _, c := range p.Roster() {
				u.Characters = append(u.Characters, CityCharacterUpdate{Identity: c.Identity, Name: c.Name, Stats: c.Stats, SkillLevels: c.SkillLevels, SkillXP: c.SkillXP, Experience: c.Experience})
			}
			if extent == "expanded source loadout" {
				*documentValue1115(t, &g.Objects[g.Inventory[0]-1], "F42") = 65535
			} else {
				item := &g.Objects[g.Inventory[0]-1]
				refs := append([]uint16(nil), (*documentRefsForField(t, item, "Effects"))...)
				prototype := g.Objects[refs[0]-1]
				for len(g.Objects) < maxCityDataObjects {
					effect := prototype
					effect.Values = append([]DocumentValueData(nil), prototype.Values...)
					*documentValue1115(t, &effect, "Identity") = 0x72000000 + uint32(len(g.Objects))*16
					g.Objects = append(g.Objects, effect)
					refs = append(refs, uint16(len(g.Objects)))
				}
				item = &g.Objects[g.Inventory[0]-1]
				*documentRefsForField(t, item, "Effects") = refs
				for i := range item.Counts {
					if item.Counts[i].Name == "Effects" {
						item.Counts[i].Count = uint32(len(refs))
					}
				}
			}
			if err := ValidateCityItemGraph(g); err != nil {
				t.Fatal("individual graph should fit before full-city admission", err)
			}
			u.Characters[0].Holdings = g
			if out, err := p.Marshal(u); err == nil || out != nil {
				t.Fatal("SAV beyond cold import bounds was emitted")
			}
			if !reflect.DeepEqual(before, p.Data()) {
				t.Fatal("refused output changed immutable source")
			}
		})
	}
}

// TestCityGraphOwnerReferenceKeepsAResolvingKeyAndZeroesAMissingOne writes a
// current pack whose Weapon owner Reference first names the hero, a key the
// written document holds, and then a key it does not hold. The first is
// translated with every other identity; the second is written as 0, the value
// the original's load gives a missing key.
func TestCityGraphOwnerReferenceKeepsAResolvingKeyAndZeroesAMissingOne(t *testing.T) {
	f, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	hero := p.Roster()[0].Identity
	value := func(r *DocumentRecordData, name string) *uint32 { return documentValue1115(t, r, name) }
	written := func(owner uint32) (reference uint32, humans map[uint32]bool) {
		t.Helper()
		u := CityUpdate{Label: []byte("owner reference"), Money: 100}
		for _, c := range p.Roster() {
			u.Characters = append(u.Characters, CityCharacterUpdate{Identity: c.Identity, Name: c.Name, Stats: c.Stats, SkillLevels: c.SkillLevels, SkillXP: c.SkillXP, Experience: c.Experience})
		}
		g := cityHoldingsFixture(t)
		*value(&g.Objects[g.Inventory[0]-1], "Reference") = owner
		u.Characters[0].Holdings = g
		b, err := p.Marshal(u)
		if err != nil {
			t.Fatal(err)
		}
		d, err := DecodeDocumentData(b)
		if err != nil {
			t.Fatal(err)
		}
		humans = make(map[uint32]bool)
		for i := range d.Objects {
			r := &d.Objects[i]
			switch r.Class {
			case "Weapon":
				reference = *value(r, "Reference")
			case "Human":
				humans[*value(r, "Identity")] = true
			}
		}
		return reference, humans
	}
	if got, _ := written(0x51000b50); got != 0 {
		t.Fatalf("missing owner key written as %#x, want 0", got)
	}
	if got, humans := written(hero); got == 0 || !humans[got] {
		t.Fatalf("resolving owner key written as %#x, want a written Human identity", got)
	}
}
