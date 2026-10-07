package game

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func actor1161Fixture(t *testing.T) []byte {
	t.Helper()
	shared := &poolFixtureSpell{id: 1, rangeByte: 19, defensive: 2, cost: 65535}
	a := actorBookFixture(91, shared, nil, &poolFixtureSpell{id: 3, rangeByte: 7, defensive: 1, cost: 32768})
	a.human = true
	a.holdings = &holdingFixture{weapon: &holdingFixtureItem{class: "Weapon", code: 0x810e, count: 1, kind: 2}, shield: &holdingFixtureItem{class: "Shield", code: 0x8201, count: 1, kind: 1}, items: []*holdingFixtureItem{{class: "Item", code: 0xe06, count: 3, kind: 3, weight: 7, effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 7 | 13<<16}}}}}
	a.holdings.worn[11] = &holdingFixtureItem{class: "Armor", code: 0xc01, count: 1, kind: 1, weight: 11}
	b := actorBookFixture(92)
	b.book = []*poolFixtureSpell{}
	c := actorBookFixture(93, shared)
	c.humanoid = true
	d := actorBookFixture(94)
	body := poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{a, b, c, d, a}}}}, nil)
	// Literal writer positions, independent of the SAV locator: a newly
	// written Spell ends at its writer offset plus9; empty/shared/absent
	// books end after their literal header/tag widths.
	for i, start := range []int{a.book[2].off + 9, b.bookOff + 9, c.bookOff + 11, d.bookOff + 1} {
		for word := range 4 {
			binary.LittleEndian.PutUint32(body[start+4*word:], 0x89abc001+uint32(i*17+word))
		}
		body[start+16] = byte(0xd1 + i)
	}
	return completeDocumentTail1115(t, poolFixtureFront(t, 91, 92, 93, 94), savedContainer(body))
}

func TestActorRoots1161DocumentLossControls(t *testing.T) {
	raw := actor1161Fixture(t)
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	roots, r, origins, err := actor1161Read(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d := actor1161DocumentDifferences(roots, r, origins, &doc); len(d) != 0 {
		t.Fatal("positive rich root fixture", d)
	}
	if roots[0].values["U5C"] != 0x89abc001 || roots[0].values["U40"] != 0x89abc004 || roots[0].values["U48"] != 0xd1 || roots[3].values["U48"] != 0xd4 {
		t.Fatal("independent final17 literal anchors differ")
	}
	fields := 0
	for _, root := range roots {
		for name := range root.values {
			fields++
			t.Run(fmt.Sprintf("actor%d-value-%s", root.loc.ArchiveIndex, name), func(t *testing.T) {
				bad, _ := sav.CloneDocumentData(doc)
				row := &bad.Objects[origins[root.loc.ArchiveIndex]-1]
				savedObjectSetValue(row, name, root.values[name]^1)
				if d := actor1161DocumentDifferences(roots, r, origins, &bad); len(d) == 0 {
					t.Fatal("accepted scalar/tail loss")
				}
			})
		}
		for name := range root.counts {
			fields++
			t.Run(fmt.Sprintf("actor%d-count-%s", root.loc.ArchiveIndex, name), func(t *testing.T) {
				bad, _ := sav.CloneDocumentData(doc)
				row := &bad.Objects[origins[root.loc.ArchiveIndex]-1]
				group1155SetCount(t, row, name, int(root.counts[name])+1)
				if d := actor1161DocumentDifferences(roots, r, origins, &bad); len(d) == 0 {
					t.Fatal("accepted count loss")
				}
			})
		}
		for name, refs := range root.refs {
			fields++
			t.Run(fmt.Sprintf("actor%d-edge-%s", root.loc.ArchiveIndex, name), func(t *testing.T) {
				bad, _ := sav.CloneDocumentData(doc)
				row := &bad.Objects[origins[root.loc.ArchiveIndex]-1]
				if len(refs) == 0 {
					row.RefSlots = slices.DeleteFunc(row.RefSlots, func(v sav.DocumentRefsData) bool { return v.Name == name })
				} else {
					for i := range row.RefSlots {
						if row.RefSlots[i].Name == name {
							row.RefSlots[i].Objects[0] ^= 1
						}
					}
				}
				if d := actor1161DocumentDifferences(roots, r, origins, &bad); len(d) == 0 {
					t.Fatal("accepted reference/presence/slot loss")
				}
			})
		}
	}
	children := 0
	for _, index := range r.source.indices() {
		row := r.source.rows[index]
		for name, value := range row.values {
			children++
			t.Run(fmt.Sprintf("child%d-%s", index, name), func(t *testing.T) {
				bad, _ := sav.CloneDocumentData(doc)
				savedObjectSetValue(&bad.Objects[origins[index]-1], name, value^1)
				if d := actor1161DocumentDifferences(roots, r, origins, &bad); len(d) == 0 {
					t.Fatal("accepted child field/identity loss")
				}
			})
		}
	}
	t.Logf("actor roots controls: %d actor field/count/edge mutations and %d child scalar/identity mutations rejected; 4 actors include Unit/Human/Humanoid, repeated references, sparse/shared/present-empty/absent books", fields, children)
}

func TestActorRoots1161ReaderTrustAndBounds(t *testing.T) {
	raw := actor1161Fixture(t)
	source, _ := sav.Open(raw)
	actors, err := source.DocumentActorLocations()
	if err != nil {
		t.Fatal(err)
	}
	objects, _ := source.DocumentObjectLocations()
	_, origins, _ := sav.DecodeDocumentDataWithOrigins(raw)
	for _, name := range []string{"omitted actor", "duplicate actor", "duplicate object", "duplicate origin", "effect count", "truncated body", "held endpoint", "XP endpoint", "book count"} {
		t.Run(name, func(t *testing.T) {
			body := slices.Clone(source.Body)
			as := slices.Clone(actors)
			os := slices.Clone(objects)
			join := slices.Clone(origins)
			switch name {
			case "omitted actor":
				as = as[1:]
			case "duplicate actor":
				as[1] = as[0]
			case "duplicate object":
				os[1] = os[0]
			case "duplicate origin":
				join[1] = join[0]
			case "effect count":
				binary.LittleEndian.PutUint32(body[as[0].Off+37:], 0xffffffff)
			case "truncated body":
				body = body[:as[0].StateOff+2]
			case "held endpoint":
				as[0].StateOff++
			case "XP endpoint":
				as[0].HumanoidXPOff++
			case "book count":
				// Find the independent original reader's nonempty sparse book
				// tag span by its uniquely literal header/count and first class tag.
				found := false
				for p := as[0].StateOff; p < as[0].HumanoidXPOff-10; p++ {
					if body[p] == 1 && binary.LittleEndian.Uint32(body[p+1:]) == 0 && binary.LittleEndian.Uint32(body[p+5:]) == 4 && binary.LittleEndian.Uint16(body[p+9:]) == 0xffff {
						binary.LittleEndian.PutUint32(body[p+5:], 0xffffffff)
						found = true
						break
					}
				}
				if !found {
					t.Fatal("literal sparse book count absent")
				}
			}
			if _, _, _, err := actor1161ReadLocations(body, as, os, join); err == nil {
				t.Fatal("accepted invalid raw input")
			}
		})
	}
}

func TestActorRoots1161LiveLossControls(t *testing.T) {
	raw := unit1158AppFixture(t)
	source, _ := sav.Open(raw)
	roots, r, origins, err := actor1161Read(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	f := unit1158FixtureFront(t)
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	if d, p := actor1161WorldDifferences(roots, r, origins, ms.savedDocument, ms.World); len(d) != 0 || p.live != 2 {
		t.Fatal("positive live baseline", d, p)
	}
	observed := ms.World.Entities()
	for _, name := range []string{"missing living", "wrong identity", "wrong book", "wrong class"} {
		t.Run(name, func(t *testing.T) {
			bad := slices.Clone(observed)
			switch name {
			case "missing living":
				bad = bad[1:]
			case "wrong identity":
				bad[0].SourceBinding.Identity ^= 1
			case "wrong book":
				bad[0].Book.State = sim.BookPresent
			case "wrong class":
				bad[0].SourceBinding.Class = 3
			}
			d, _ := actor1161EntityDifferences(roots, r, origins, ms.savedDocument, ms.World, bad)
			if len(d) == 0 {
				t.Fatal("accepted live root loss")
			}
			if name == "missing living" && !strings.Contains(strings.Join(d, ";"), "eligible raw actor missing") {
				t.Fatal("living omission was hidden as raw-only", d)
			}
		})
	}
	// The independent stage, not the imported survivor set, decides this case.
	roots[0].stage = 1
	roots[0].hp = 0
	if d, _ := actor1161EntityDifferences(roots, r, origins, ms.savedDocument, ms.World, observed[1:]); !strings.Contains(strings.Join(d, ";"), "eligible raw actor missing") {
		t.Fatal("dying omission was hidden as raw-only", d)
	}
}
