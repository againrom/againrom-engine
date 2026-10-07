package game

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// This synthetic stream is written from literal operands, with no SAV codec
// or decoder. The companion DTO is the supplied test input, not a conversion
// of the independent reader's output. It includes every supported class,
// nonzero opaque fields, repeated/shared child edges, a null Weapon Spell,
// an owned Spell, an empty Sack and deliberately stale signed load.
type sack1151Fixture struct {
	body      []byte
	locations []sav.DocumentObjectLocation
	origins   map[uint16]uint16
	doc       sav.DocumentData
	next      uint16
	classes   map[string]uint16
	tail      int
}

func (f *sack1151Fixture) number(n uint32, width int) {
	for i := 0; i < width; i++ {
		f.body = append(f.body, byte(n>>uint(i*8)))
	}
}

func (f *sack1151Fixture) object(class string) (uint16, uint16) {
	if ci := f.classes[class]; ci != 0 {
		f.number(uint32(ci|0x8000), 2)
	} else {
		f.number(0xffff, 2)
		f.number(1, 2)
		f.number(uint32(len(class)), 2)
		f.body = append(f.body, class...)
		f.classes[class] = f.next
		f.next++
	}
	index := f.next
	f.next++
	f.locations = append(f.locations, sav.DocumentObjectLocation{ArchiveIndex: index, Class: class, Off: len(f.body)})
	f.doc.Objects = append(f.doc.Objects, sav.DocumentRecordData{Class: class})
	local := uint16(len(f.doc.Objects))
	f.origins[index] = local
	return index, local
}

func (f *sack1151Fixture) value(local uint16, name string, n uint32, width int) {
	f.doc.Objects[local-1].Values = append(f.doc.Objects[local-1].Values, sav.DocumentValueData{Name: name, Value: n})
	f.number(n, width)
}

func (f *sack1151Fixture) raw(local uint16, name string, b []byte) {
	f.doc.Objects[local-1].Raw = append(f.doc.Objects[local-1].Raw, sav.DocumentRawData{Name: name, Bytes: slices.Clone(b)})
	f.body = append(f.body, b...)
}

func (f *sack1151Fixture) token(local uint16, seed uint32) {
	f.raw(local, "Block12", []byte{17, 18, byte(seed), 9, 128, 128, 19, 20, 21, 22, 23, 24})
	f.value(local, "RuntimeID", 0x12345678+seed, 4)
	f.value(local, "T0C", seed, 1)
	f.value(local, "T0E", 0x9876, 2)
	f.value(local, "T08", 0xabcdef12, 4)
	f.value(local, "T18", 2, 2)
	f.value(local, "T1C", 0xffffffe9, 4)
	f.value(local, "Identity", 0x11223300+seed, 4)
	f.value(local, "Reference", 0x99887700+seed, 4)
}

func (f *sack1151Fixture) refs(local uint16, name string, refs []uint16, count bool) {
	f.doc.Objects[local-1].RefSlots = append(f.doc.Objects[local-1].RefSlots, sav.DocumentRefsData{Name: name, Objects: refs})
	if count {
		f.doc.Objects[local-1].Counts = append(f.doc.Objects[local-1].Counts, sav.DocumentCountData{Name: name, Count: uint32(len(refs))})
	}
}

func sack1151Literal() sack1151Fixture {
	f := sack1151Fixture{next: 1, classes: map[string]uint16{}, origins: map[uint16]uint16{}, doc: sav.DocumentData{World: &sav.DocumentWorldData{}}}
	f.number(2, 4)
	_, sack := f.object("Sack")
	f.token(sack, 5)
	f.value(sack, "S3C", 7123, 4)
	f.number(5, 4)
	var contents []uint16
	var effectIndex, effectLocal uint16
	for ordinal, class := range []string{"Item", "Weapon", "Armor", "Shield", "Weapon"} {
		_, item := f.object(class)
		contents = append(contents, item)
		f.token(item, uint32(20+ordinal))
		var effects []uint16
		if ordinal == 0 {
			f.number(3, 4)
			for j := 0; j < 2; j++ {
				index, effect := f.object("Effect")
				f.token(effect, uint32(30+j))
				f.value(effect, "E3C", uint32(3+j), 1)
				f.value(effect, "E3D", 8, 1)
				f.value(effect, "E40", 0xfffffff1+uint32(j), 4)
				f.value(effect, "E0C", 0, 1)
				effects = append(effects, effect)
				if j == 0 {
					effectIndex, effectLocal = index, effect
				}
			}
			f.number(uint32(effectIndex), 2)
			effects = append(effects, effectLocal)
		} else if ordinal == 1 {
			f.number(1, 4)
			f.number(uint32(effectIndex), 2)
			effects = []uint16{effectLocal}
		} else {
			f.number(0, 4)
		}
		f.refs(item, "Effects", effects, true)
		f.value(item, "F40", 0x8101+uint32(ordinal)*0x100, 2)
		f.value(item, "F42", uint32(1+ordinal), 2)
		f.value(item, "F44", 0, 1)
		f.value(item, "F45", 0x56, 1)
		f.value(item, "F46", 0x67, 1)
		f.value(item, "F48", 0xabcd, 2)
		f.value(item, "F4A", 0xfffd, 2)
		f.value(item, "F47", 0x78, 1)
		switch class {
		case "Weapon":
			f.raw(item, "W52", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24})
			f.raw(item, "W6A", []byte{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46})
			f.value(item, "W50", 2, 1)
			if ordinal == 1 {
				_, spell := f.object("Spell")
				f.value(spell, "S08", 7, 1)
				f.value(spell, "S09", 13, 1)
				f.value(spell, "S0A", 1, 1)
				f.value(spell, "S0C", 0x1234, 2)
				f.value(spell, "This", 0x11223344, 4)
				f.refs(item, "WeaponSpell", []uint16{spell}, false)
			} else {
				f.number(0, 2)
				f.refs(item, "WeaponSpell", []uint16{0}, false)
			}
		case "Armor":
			f.raw(item, "A52", []byte{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29, 31, 33, 35, 37, 39, 41, 43})
			f.value(item, "A50", 3, 1)
		case "Shield":
			f.raw(item, "S50", []byte{2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44})
		}
	}
	f.refs(sack, "Contents", contents, true)
	f.tail = len(f.body)
	f.value(sack, "Contents1C", 0xffffffff, 4)
	f.value(sack, "Contents20", 0xffffffef, 4)
	_, empty := f.object("Sack")
	f.token(empty, 6)
	f.value(empty, "S3C", 0, 4)
	f.number(0, 4)
	f.refs(empty, "Contents", nil, true)
	f.value(empty, "Contents1C", 10000, 4)
	f.value(empty, "Contents20", 0, 4)
	f.doc.World.Sacks = []uint16{sack, empty}
	return f
}

func (f *sack1151Fixture) live(t *testing.T) (*SnapshotSAVObjectBindings, *sim.SavedObjects, []sim.Sack) {
	t.Helper()
	b := &SnapshotSAVObjectBindings{Version: 2}
	r := &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: sim.SavedObjectID(len(f.doc.Objects) + 1)}
	var sacks []sim.Sack
	for i := range f.doc.Objects {
		local := uint16(i + 1)
		id := sim.SavedObjectID(local)
		row := &f.doc.Objects[i]
		binding := SnapshotSAVObjectBinding{ID: id, ObjectIndex: local}
		switch row.Class {
		case "Sack":
			token, gold, c, err := savedSackRecord(row)
			if err != nil {
				t.Fatal(err)
			}
			r.Sacks = append(r.Sacks, sim.SavedSackObject{ID: id, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal}, Token: token, Gold: gold})
			r.SackRoots = append(r.SackRoots, id)
			c.Owner = sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: id}
			refs, _ := savedObjectRefs(row, "Contents")
			s := sim.Sack{ObjectID: id, X: int32(token.Position[2]), Y: int32(token.Position[3]), Gold: gold}
			for _, ref := range refs {
				c.Items = append(c.Items, sim.SavedObjectID(ref))
				item, err := savedItemRecord(&f.doc, ref)
				if err != nil {
					t.Fatal(err)
				}
				item.Value.ObjectID = sim.SavedObjectID(ref)
				for range item.Value.Count {
					s.ItemInstances = append(s.ItemInstances, item.Value.Instance())
					s.Items = append(s.Items, item.Value.Code)
				}
			}
			r.Containers = append(r.Containers, c)
			b.Sacks = append(b.Sacks, binding)
			sacks = append(sacks, s)
		case "Item", "Weapon", "Armor", "Shield":
			item, err := savedItemRecord(&f.doc, local)
			if err != nil {
				t.Fatal(err)
			}
			item.ID, item.Value.ObjectID = id, id
			refs, _ := savedObjectRefs(row, "Effects")
			for _, ref := range refs {
				item.Effects = append(item.Effects, sim.SavedObjectID(ref))
			}
			refs, _ = savedObjectRefs(row, "WeaponSpell")
			if len(refs) == 1 {
				item.Spell = sim.SavedObjectID(refs[0])
			}
			r.Items = append(r.Items, item)
			b.Items = append(b.Items, binding)
		case "Effect":
			e, err := savedEffectRecord(row)
			if err != nil {
				t.Fatal(err)
			}
			e.ID = id
			r.Effects = append(r.Effects, e)
			b.Effects = append(b.Effects, binding)
		case "Spell":
			s, err := savedSpellRecord(row)
			if err != nil {
				t.Fatal(err)
			}
			s.ID = id
			r.Spells = append(r.Spells, s)
			b.Spells = append(b.Spells, binding)
		}
	}
	return b, r, sacks
}

func TestSacks1151IndependentReaderAndControls(t *testing.T) {
	fixture := sack1151Literal()
	source, err := sacks1151Read(fixture.body, 0, fixture.locations)
	if err != nil {
		t.Fatal(err)
	}
	if source.count != 2 || len(source.rows) != 10 || source.rows[source.roots[0]].values["Contents20"] != 0xffffffef {
		t.Fatal("literal count/population/signed tail changed", source.count, len(source.rows))
	}
	if differences := sacks1151DocumentDifferences(source, fixture.origins, &fixture.doc); len(differences) != 0 {
		t.Fatal(differences)
	}
	b, r, sacks := fixture.live(t)
	if differences, gaps := sacks1151LiveDifferences(source, fixture.origins, b, r, sacks); len(differences)+len(gaps) != 0 {
		t.Fatal(differences, gaps)
	}
	for _, tc := range []struct {
		name, fragment string
		edit           func(*sack1151Fixture)
	}{
		{"omitted Sack", "Sack count", func(f *sack1151Fixture) { f.doc.World.Sacks = f.doc.World.Sacks[1:] }},
		{"root order", "root order/aliases", func(f *sack1151Fixture) { slices.Reverse(f.doc.World.Sacks) }},
		{"root alias", "root order/aliases", func(f *sack1151Fixture) { f.doc.World.Sacks[1] = f.doc.World.Sacks[0] }},
		{"omitted Item", "Contents order/aliases", func(f *sack1151Fixture) {
			f.doc.Objects[0].RefSlots[0].Objects = f.doc.Objects[0].RefSlots[0].Objects[1:]
		}},
		{"item order", "Contents order/aliases", func(f *sack1151Fixture) { slices.Reverse(f.doc.Objects[0].RefSlots[0].Objects) }},
		{"omitted tail", "omitted scalar Contents20", func(f *sack1151Fixture) {
			v := &f.doc.Objects[0].Values
			*v = slices.DeleteFunc(*v, func(x sav.DocumentValueData) bool { return x.Name == "Contents20" })
		}},
		{"container count", "Contents count", func(f *sack1151Fixture) { f.doc.Objects[0].Counts[0].Count++ }},
		{"item count", "F42", func(f *sack1151Fixture) { newGroupSetValue1115(t, &f.doc.Objects[1], "F42", 9) }},
		{"Effect order", "Effects order/aliases", func(f *sack1151Fixture) { v := f.doc.Objects[1].RefSlots[0].Objects; v[0], v[1] = v[1], v[0] }},
		{"Effect alias", "Effects order/aliases", func(f *sack1151Fixture) { v := f.doc.Objects[1].RefSlots[0].Objects; v[2] = v[1] }},
		{"opaque Token", "T0E", func(f *sack1151Fixture) { newGroupSetValue1115(t, &f.doc.Objects[0], "T0E", 0) }},
		{"owned Spell", "S0C", func(f *sack1151Fixture) {
			for i := range f.doc.Objects {
				if f.doc.Objects[i].Class == "Spell" {
					newGroupSetValue1115(t, &f.doc.Objects[i], "S0C", 0)
				}
			}
		}},
	} {
		t.Run("Document/"+tc.name, func(t *testing.T) {
			f := sack1151Literal()
			tc.edit(&f)
			d := sacks1151DocumentDifferences(source, f.origins, &f.doc)
			if !strings.Contains(strings.Join(d, "\n"), tc.fragment) {
				t.Fatal("control escaped", d)
			}
		})
	}
	for _, tc := range []struct {
		name string
		edit func(*SnapshotSAVObjectBindings, *sim.SavedObjects, []sim.Sack)
	}{
		{"omitted Sack", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) { r.Sacks = r.Sacks[1:] }},
		{"root order", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) { slices.Reverse(r.SackRoots) }},
		{"omitted Item", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) { r.Items = r.Items[1:] }},
		{"omitted tail", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) { r.Containers = r.Containers[1:] }},
		{"stale tail", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) { r.Containers[0].Accumulator = 0 }},
		{"item count", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) { r.Items[0].Value.Count++ }},
		{"item order", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) {
			slices.Reverse(r.Containers[0].Items)
		}},
		{"item alias", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) {
			r.Containers[0].Items[1] = r.Containers[0].Items[0]
		}},
		{"Effect order", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) {
			r.Items[0].Effects[0], r.Items[0].Effects[1] = r.Items[0].Effects[1], r.Items[0].Effects[0]
		}},
		{"Effect alias", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) {
			r.Items[0].Effects[2] = r.Items[0].Effects[1]
		}},
		{"owned Spell", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ []sim.Sack) { r.Spells[0].Value.ManaCost = 0 }},
		{"expanded live item", func(_ *SnapshotSAVObjectBindings, _ *sim.SavedObjects, s []sim.Sack) {
			s[0].ItemInstances[0].Weight = 0
		}},
		{"unexplained adoption gap", func(b *SnapshotSAVObjectBindings, _ *sim.SavedObjects, _ []sim.Sack) { b.Sacks = b.Sacks[1:] }},
	} {
		t.Run("World/"+tc.name, func(t *testing.T) {
			f := sack1151Literal()
			b, r, s := f.live(t)
			tc.edit(b, r, s)
			d, _ := sacks1151LiveDifferences(source, f.origins, b, r, s)
			if len(d) == 0 {
				t.Fatal("control escaped")
			}
		})
	}
	for _, tc := range []struct {
		name string
		edit func(*sack1151Fixture)
	}{
		{"raw root count", func(f *sack1151Fixture) { binary.LittleEndian.PutUint32(f.body, 1) }},
		{"unbounded raw count", func(f *sack1151Fixture) { binary.LittleEndian.PutUint32(f.body, 0xffffffff) }},
		{"omitted locator", func(f *sack1151Fixture) { f.locations = f.locations[1:] }},
		{"truncated tail", func(f *sack1151Fixture) { f.body = f.body[:len(f.body)-1] }},
		{"unsupported class", func(f *sack1151Fixture) { f.locations[len(f.locations)-1].Class = "Unknown" }},
	} {
		t.Run("source/"+tc.name, func(t *testing.T) {
			f := sack1151Literal()
			tc.edit(&f)
			if _, err := sacks1151Read(f.body, 0, f.locations); err == nil {
				t.Fatal("malformed source accepted")
			}
		})
	}
}

// Equal values do not make two archive objects the same object. The original
// controls kept the origin join fixed, so a decoder could lose an Effect and
// collapse that join without either comparator detecting the changed graph.
func TestSacks1151RejectsCollapsedEqualEffectOrigins(t *testing.T) {
	equalFixture := func() (sack1151Fixture, sackByteSource) {
		f := sack1151Literal()
		var first, second int
		for _, loc := range f.locations {
			switch loc.ArchiveIndex {
			case 6:
				first = loc.Off
			case 7:
				second = loc.Off
			}
		}
		if first == 0 || second == 0 || f.origins[6] != 3 || f.origins[7] != 4 {
			t.Fatal("distinct Effect fixture identities changed")
		}
		copy(f.body[second:second+44], f.body[first:first+44])
		f.doc.Objects[3] = f.doc.Objects[2]
		source, err := sacks1151Read(f.body, 0, f.locations)
		if err != nil {
			t.Fatal(err)
		}
		return f, source
	}
	validate := func(r *sim.SavedObjects) {
		t.Helper()
		for i := range r.Items {
			r.Items[i].Origin.Kind = sim.SavedObjectOriginal
		}
		for i := range r.Effects {
			r.Effects[i].Origin.Kind = sim.SavedObjectOriginal
		}
		for i := range r.Spells {
			r.Spells[i].Origin.Kind = sim.SavedObjectOriginal
		}
		if err := r.Validate(); err != nil {
			t.Fatal("control is not an admissible native graph", err)
		}
	}
	f, source := equalFixture()
	b, r, sacks := f.live(t)
	validate(r)
	if len(source.rows) != 10 || len(f.doc.Objects) != 10 || len(r.Effects) != 2 {
		t.Fatal("distinct equal-valued baseline population changed")
	}
	if d := sacks1151DocumentDifferences(source, f.origins, &f.doc); len(d) != 0 {
		t.Fatal("distinct equal-valued Document baseline", d)
	}
	if d, gaps := sacks1151LiveDifferences(source, f.origins, b, r, sacks); len(d)+len(gaps) != 0 {
		t.Fatal("distinct equal-valued World baseline", d, gaps)
	}
	t.Log("baseline: raw10/Document10/nativeEffects2; distinct equal-valued objects and repeated references both preserved")
	t.Run("archive-to-DTO collapse", func(t *testing.T) {
		f, source := equalFixture()
		remap := func(index uint16) uint16 {
			if index == 4 {
				return 3
			}
			if index > 4 {
				return index - 1
			}
			return index
		}
		f.doc.Objects = slices.Delete(f.doc.Objects, 3, 4)
		for i := range f.doc.Objects {
			for j := range f.doc.Objects[i].RefSlots {
				for k, ref := range f.doc.Objects[i].RefSlots[j].Objects {
					f.doc.Objects[i].RefSlots[j].Objects[k] = remap(ref)
				}
			}
		}
		for i, index := range f.doc.World.Sacks {
			f.doc.World.Sacks[i] = remap(index)
		}
		for index, local := range f.origins {
			f.origins[index] = remap(local)
		}
		b, r, sacks := f.live(t)
		validate(r)
		if len(source.rows) != 10 || len(f.doc.Objects) != 9 || len(r.Effects) != 1 || f.origins[6] != f.origins[7] {
			t.Fatal("collapse control did not remove the distinct Effect")
		}
		document := sacks1151DocumentDifferences(source, f.origins, &f.doc)
		world, gaps := sacks1151LiveDifferences(source, f.origins, b, r, sacks)
		if !strings.Contains(strings.Join(document, "\n"), "archive-to-DTO identity collapse") || !strings.Contains(strings.Join(world, "\n"), "archive-to-DTO identity collapse") || len(gaps) != 0 {
			t.Fatalf("false acceptance: raw10/Document9/nativeEffects1; Document=%v World=%v gaps=%v", document, world, gaps)
		}
		t.Log("raw10/Document9/nativeEffects1 rejected independently:", document, world)
	})
	t.Run("DTO-to-native collapse", func(t *testing.T) {
		f, source := equalFixture()
		b, r, sacks := f.live(t)
		r.Effects = slices.Delete(r.Effects, 1, 2)
		for i := range r.Items {
			for j, id := range r.Items[i].Effects {
				if id == 4 {
					r.Items[i].Effects[j] = 3
				}
			}
		}
		b.Effects[1].ID = 3
		validate(r)
		if d := sacks1151DocumentDifferences(source, f.origins, &f.doc); len(d) != 0 {
			t.Fatal("native-only collapse changed the complete Document", d)
		}
		world, gaps := sacks1151LiveDifferences(source, f.origins, b, r, sacks)
		if !strings.Contains(strings.Join(world, "\n"), "DTO-to-native identity collapse") || len(gaps) != 0 {
			t.Fatalf("false acceptance: raw10/Document10/nativeEffects1; World=%v gaps=%v", world, gaps)
		}
		t.Log("raw10/Document10/nativeEffects1 rejected:", world)
	})
}
