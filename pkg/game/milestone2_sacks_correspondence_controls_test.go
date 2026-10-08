package game

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestSackCurrentRetiredRowsAreNotLivePopulation(t *testing.T) {
	f := sack1151Literal()
	source, err := sackRead(f.body, 0, f.locations)
	if err != nil {
		t.Fatal(err)
	}
	b, r, ground := f.live(t)
	for _, id := range []sim.SavedObjectID{468, 474} {
		r.Sacks = append(r.Sacks, sim.SavedSackObject{
			ID: id, Retired: true, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal},
		})
		b.Sacks = append(b.Sacks, SnapshotSAVObjectBinding{ID: id})
	}
	r.NextID = 475
	if d, gaps := sackLiveDifferences(source, f.origins, b, r, ground); len(d)+len(gaps) != 0 {
		t.Fatal("two live roots and two retired history rows", d, gaps)
	}
}

func TestSackCurrentRepeatedRootSlotsKeepDistinctPopulation(t *testing.T) {
	f := sack1151Literal()
	binary.LittleEndian.PutUint32(f.body[:4], 3)
	f.number(uint32(f.locations[0].ArchiveIndex), 2)
	f.doc.World.Sacks = append(f.doc.World.Sacks, f.doc.World.Sacks[0])
	source, err := sackRead(f.body, 0, f.locations)
	if err != nil {
		t.Fatal(err)
	}
	b, r, ground := f.live(t)
	r.SackRoots = append(r.SackRoots, r.SackRoots[0])
	if source.count != 3 || len(ground) != 2 || len(r.Sacks) != 2 {
		t.Fatal("three raw slots and two physical objects fixture changed")
	}
	if d := sackDocumentDifferences(source, f.origins, &f.doc); len(d) != 0 {
		t.Fatal("raw repeated root must remain in Document", d)
	}
	if d, gaps := sackLiveDifferences(source, f.origins, b, r, ground); len(d)+len(gaps) != 0 {
		t.Fatal("root multiplicity is distinct from live object population", d, gaps)
	}
	r.SackRoots = r.SackRoots[:2]
	if d, _ := sackLiveDifferences(source, f.origins, b, r, ground); !strings.Contains(strings.Join(d, "\n"), "root order/aliases") {
		t.Fatal("repeated native root loss escaped", d)
	}
}

func TestSackCurrentRetiredLifecycleAndDuplicateRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*SnapshotSAVObjectBindings, *sim.SavedObjects, *[]sim.Sack)
	}{
		{"live row with Object0", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ *[]sim.Sack) { r.Sacks[2].Retired = false }},
		{"retired nonzero binding", func(b *SnapshotSAVObjectBindings, _ *sim.SavedObjects, _ *[]sim.Sack) { b.Sacks[2].ObjectIndex = 1 }},
		{"retired Gold", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ *[]sim.Sack) { r.Sacks[2].Gold = 1 }},
		{"retired root", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ *[]sim.Sack) {
			r.SackRoots = append(r.SackRoots, 468)
		}},
		{"retired container", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ *[]sim.Sack) {
			r.Containers = append(r.Containers, sim.SavedObjectContainer{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: 468}, Present: true})
		}},
		{"retired ground carrier", func(_ *SnapshotSAVObjectBindings, _ *sim.SavedObjects, s *[]sim.Sack) {
			*s = append(*s, sim.Sack{ObjectID: 468})
		}},
		{"missing retired binding", func(b *SnapshotSAVObjectBindings, _ *sim.SavedObjects, _ *[]sim.Sack) { b.Sacks = b.Sacks[:2] }},
		{"duplicate live row", func(_ *SnapshotSAVObjectBindings, r *sim.SavedObjects, _ *[]sim.Sack) {
			r.Sacks = append(r.Sacks, r.Sacks[0])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := sack1151Literal()
			source, err := sackRead(f.body, 0, f.locations)
			if err != nil {
				t.Fatal(err)
			}
			b, r, ground := f.live(t)
			r.Sacks = append(r.Sacks, sim.SavedSackObject{ID: 468, Retired: true, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal}})
			b.Sacks = append(b.Sacks, SnapshotSAVObjectBinding{ID: 468})
			tc.edit(b, r, &ground)
			if d, _ := sackLiveDifferences(source, f.origins, b, r, ground); len(d) == 0 {
				t.Fatal("lifecycle/duplicate control escaped")
			}
		})
	}
}

func TestSackCurrentUnboundCoverageUsesExactCurrentObject(t *testing.T) {
	f := sack1151Literal()
	source, err := sackRead(f.body, 0, f.locations)
	if err != nil {
		t.Fatal(err)
	}
	b, r, ground := f.live(t)
	b.Sacks, b.Items, b.Effects, b.Spells = b.Sacks[1:], nil, nil, nil
	r.Sacks, r.SackRoots, r.Containers = r.Sacks[1:], r.SackRoots[1:], r.Containers[1:]
	r.Items, r.Effects, r.Spells = nil, nil, nil
	ground[0].ObjectID = 0
	for i := range ground[0].ItemInstances {
		ground[0].ItemInstances[i].ObjectID = 0
	}
	b.Unavailable = []SnapshotSAVObjectCoverage{{ObjectIndex: 1, Reason: "fixture native Token/container carrier absent"}}
	if d, gaps := sackLiveDifferences(source, f.origins, b, r, ground); len(d) != 0 || len(gaps) != 1 {
		t.Fatal("explicit ID0 Token/container gap was lost", d, gaps)
	}
	b.Unavailable[0].ObjectIndex = 2
	if d, _ := sackLiveDifferences(source, f.origins, b, r, ground); !strings.Contains(strings.Join(d, "\n"), "unadopted without named coverage") {
		t.Fatal("stale coverage coordinate escaped", d)
	}
}

func sackReindexedFixture() sack1151Fixture {
	f := sack1151Literal()
	order := []int{9, 5, 0, 2, 8, 1, 7, 3, 6, 4}
	objects, remap := slices.Clone(f.doc.Objects), map[uint16]uint16{0: 0}
	for current, original := range order {
		remap[uint16(original+1)] = uint16(current + 1)
	}
	for current, original := range order {
		row := objects[original]
		row.RefSlots = slices.Clone(row.RefSlots)
		for j := range row.RefSlots {
			row.RefSlots[j].Objects = slices.Clone(row.RefSlots[j].Objects)
			for k, ref := range row.RefSlots[j].Objects {
				row.RefSlots[j].Objects[k] = remap[ref]
			}
		}
		f.doc.Objects[current] = row
	}
	for i, ref := range f.doc.World.Sacks {
		f.doc.World.Sacks[i] = remap[ref]
	}
	return f
}

func TestSackCurrentCurrentIdentityJoinPreservesRawChecks(t *testing.T) {
	f := sackReindexedFixture()
	source, err := sackRead(f.body, 0, f.locations)
	if err != nil {
		t.Fatal(err)
	}
	if stale := sackDocumentDifferences(source, f.origins, &f.doc); len(stale) == 0 {
		t.Fatal("stale decoder coordinate witness did not fail")
	}
	join, err := sackCurrentOrigins(source, &f.doc)
	if err != nil {
		t.Fatal(err)
	}
	if join[source.roots[0]] == f.origins[source.roots[0]] {
		t.Fatal("current join reused decoder ordinal")
	}
	if d := sackDocumentDifferences(source, join, &f.doc); len(d) != 0 {
		t.Fatal("key-preserving reindex changed raw correspondence", d)
	}
	f.doc.Objects = append(f.doc.Objects, sav.DocumentRecordData{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Identity", Value: source.rows[source.roots[0]].values["Identity"]}}})
	if _, err := sackCurrentOrigins(source, &f.doc); err != nil {
		t.Fatal("class-scoped key rejected unrelated typed decoy", err)
	}
	for _, tc := range []struct {
		name string
		edit func(*sav.DocumentData, map[uint16]uint16, sackByteSource)
	}{
		{"gold", func(d *sav.DocumentData, j map[uint16]uint16, s sackByteSource) {
			newGroupSetValue1115(t, &d.Objects[j[s.roots[0]]-1], "S3C", 0)
		}},
		{"code", func(d *sav.DocumentData, j map[uint16]uint16, s sackByteSource) {
			ref := s.rows[s.roots[0]].refs["Contents"][0]
			newGroupSetValue1115(t, &d.Objects[j[ref]-1], "F40", 0)
		}},
		{"service", func(d *sav.DocumentData, j map[uint16]uint16, s sackByteSource) {
			ref := s.rows[s.roots[0]].refs["Contents"][0]
			newGroupSetValue1115(t, &d.Objects[j[ref]-1], "F47", 0)
		}},
		{"opaque Token", func(d *sav.DocumentData, j map[uint16]uint16, s sackByteSource) {
			d.Objects[j[s.roots[0]]-1].Raw[0].Bytes = make([]byte, 12)
		}},
		{"root alias", func(d *sav.DocumentData, _ map[uint16]uint16, _ sackByteSource) { d.World.Sacks[1] = d.World.Sacks[0] }},
		{"child alias", func(d *sav.DocumentData, j map[uint16]uint16, s sackByteSource) {
			ref := s.rows[s.roots[0]].refs["Contents"][0]
			v := d.Objects[j[ref]-1].RefSlots[0].Objects
			v[2] = v[1]
		}},
		{"coherent Effect key and edge swap", func(d *sav.DocumentData, j map[uint16]uint16, s sackByteSource) {
			item := s.rows[s.roots[0]].refs["Contents"][0]
			refs := s.rows[item].refs["Effects"]
			first, second := j[refs[0]], j[refs[1]]
			a, _ := savedStructureValue(&d.Objects[first-1], "Identity")
			b, _ := savedStructureValue(&d.Objects[second-1], "Identity")
			newGroupSetValue1115(t, &d.Objects[first-1], "Identity", b)
			newGroupSetValue1115(t, &d.Objects[second-1], "Identity", a)
			for i := range d.Objects {
				for j := range d.Objects[i].RefSlots {
					for k, ref := range d.Objects[i].RefSlots[j].Objects {
						if ref == first {
							d.Objects[i].RefSlots[j].Objects[k] = second
						} else if ref == second {
							d.Objects[i].RefSlots[j].Objects[k] = first
						}
					}
				}
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := sackReindexedFixture()
			join, err := sackCurrentOrigins(source, &current.doc)
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(&current.doc, join, source)
			join, err = sackCurrentOrigins(source, &current.doc)
			if err != nil {
				t.Fatal("non-identity values affected key selection", err)
			}
			if d := sackDocumentDifferences(source, join, &current.doc); len(d) == 0 {
				t.Fatal("raw value/alias loss escaped")
			}
		})
	}
	f = sackReindexedFixture()
	binary.LittleEndian.PutUint32(f.body[f.locations[0].Off+37:], 0)
	changed, err := sackRead(f.body, 0, f.locations)
	if err != nil {
		t.Fatal(err)
	}
	join, err = sackCurrentOrigins(changed, &f.doc)
	if err != nil {
		t.Fatal("raw Gold affected identity correspondence", err)
	}
	if d := sackDocumentDifferences(changed, join, &f.doc); len(d) == 0 {
		t.Fatal("independently changed raw Gold escaped")
	}
}

func TestSackCurrentCurrentIdentityJoinRejectsMissingAndAmbiguousMetadata(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*sack1151Fixture)
	}{
		{"missing current key", func(f *sack1151Fixture) {
			f.doc.Objects[0].Values = slices.DeleteFunc(f.doc.Objects[0].Values, func(v sav.DocumentValueData) bool { return v.Name == "Identity" })
		}},
		{"zero current key", func(f *sack1151Fixture) { newGroupSetValue1115(t, &f.doc.Objects[0], "Identity", 0) }},
		{"duplicate member", func(f *sack1151Fixture) {
			f.doc.Objects[0].Values = append(f.doc.Objects[0].Values, sav.DocumentValueData{Name: "Identity", Value: 0x11223305})
		}},
		{"same-class collision", func(f *sack1151Fixture) { f.doc.Objects = append(f.doc.Objects, f.doc.Objects[0]) }},
		{"wrong-class replacement", func(f *sack1151Fixture) { f.doc.Objects[0].Class = "Unit" }},
		{"zero raw key", func(f *sack1151Fixture) { binary.LittleEndian.PutUint32(f.body[f.locations[0].Off+29:], 0) }},
		{"duplicate raw Effect key", func(f *sack1151Fixture) {
			var first, second int
			for _, loc := range f.locations {
				if loc.Class == "Effect" {
					if first == 0 {
						first = loc.Off
					} else {
						second = loc.Off
					}
				}
			}
			copy(f.body[second+29:second+33], f.body[first+29:first+33])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := sack1151Literal()
			tc.edit(&f)
			source, err := sackRead(f.body, 0, f.locations)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := sackCurrentOrigins(source, &f.doc); err == nil {
				t.Fatal("invalid independent identity metadata accepted")
			}
		})
	}
}

func TestSackCurrentReindexedUnboundCoverageCoordinate(t *testing.T) {
	f := sackReindexedFixture()
	source, err := sackRead(f.body, 0, f.locations)
	if err != nil {
		t.Fatal(err)
	}
	join, err := sackCurrentOrigins(source, &f.doc)
	if err != nil {
		t.Fatal(err)
	}
	local := join[source.roots[0]]
	if local == f.origins[source.roots[0]] {
		t.Fatal("coverage coordinate fixture did not reindex")
	}
	id := sim.SavedObjectID(local)
	b, r, ground := f.live(t)
	b.Sacks = slices.DeleteFunc(b.Sacks, func(v SnapshotSAVObjectBinding) bool { return v.ID == id })
	b.Items, b.Effects, b.Spells = nil, nil, nil
	r.Sacks = slices.DeleteFunc(r.Sacks, func(v sim.SavedSackObject) bool { return v.ID == id })
	r.SackRoots = nil
	for _, object := range f.doc.World.Sacks {
		if object != local {
			r.SackRoots = append(r.SackRoots, sim.SavedObjectID(object))
		}
	}
	r.Containers = slices.DeleteFunc(r.Containers, func(v sim.SavedObjectContainer) bool { return v.Owner.Object == id })
	r.Items, r.Effects, r.Spells = nil, nil, nil
	for i := range ground {
		if ground[i].ObjectID == id {
			ground[i].ObjectID = 0
			for j := range ground[i].ItemInstances {
				ground[i].ItemInstances[j].ObjectID = 0
			}
		}
	}
	b.Unavailable = []SnapshotSAVObjectCoverage{{ObjectIndex: local, Reason: "fixture native Token/container carrier absent"}}
	if d, gaps := sackLiveDifferences(source, join, b, r, ground); len(d) != 0 || len(gaps) != 1 {
		t.Fatal("exact current coverage coordinate failed", d, gaps)
	}
	b.Unavailable[0].ObjectIndex = f.origins[source.roots[0]]
	if d, _ := sackLiveDifferences(source, join, b, r, ground); !strings.Contains(strings.Join(d, "\n"), "unadopted without named coverage") {
		t.Fatal("stale original coverage coordinate escaped", d)
	}
}
