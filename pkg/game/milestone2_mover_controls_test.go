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

func mover1160Literal(t *testing.T, identity uint32) ([]byte, mover1160Source, *SnapshotSAVDocument) {
	t.Helper()
	a, b := &poolFixtureActor{name: "equal"}, &poolFixtureActor{name: "equal"}
	c, d := &poolFixtureActor{name: "Human", human: true}, &poolFixtureActor{name: "Humanoid", humanoid: true}
	actors := []*poolFixtureActor{a, b, c, d}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a, b, a, nil, c, d, b}}}}, nil)
	lists := [3][]uint16{{0x0201, 0x0302, 0x0201}, {0xffff}, {0, 0xfffe, 0, 0x8001}}
	words := func(list []uint16) []byte {
		b := binary.LittleEndian.AppendUint16(nil, uint16(len(list)))
		for _, v := range list {
			b = binary.LittleEndian.AppendUint16(b, v)
		}
		return b
	}
	splice := func(at, n int, value []byte) {
		b := append([]byte(nil), body[:at]...)
		b = append(b, value...)
		body = append(b, body[at+n:]...)
	}
	// Fixture-owned offsets: Token37, empty Effects4, empty route counts4,
	// combat134, mover180, order148, order-path count2. Work backwards so
	// earlier literal starts remain stable while inserting the three lists.
	for i := len(actors) - 1; i >= 0; i-- {
		at := actors[i].off
		binary.LittleEndian.PutUint32(body[at+29:], identity)
		for j := range 12 {
			body[at+j] = byte(0x31 + j)
		}
		for j := range 180 {
			body[at+179+j] = byte(0x41 + j)
		}
		for j := range 148 {
			body[at+359+j] = byte(0x81 + j)
		}
		splice(at+507, 2, words(lists[2]))
		splice(at+41, 4, append(words(lists[0]), words(lists[1])...))
	}
	raw := completeDocumentTail1115(t, unit1158FixtureFront(t), savedContainer(body))
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := mover1160Expected(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := decodeSavedDocument(raw)
	if diffs := want.documentDifferences(state); len(diffs) != 0 {
		t.Fatal("literal positive Document", diffs)
	}
	locs, _ := f.DocumentActorLocations()
	if len(want.records) != 4 || len(locs) != 4 {
		t.Fatal("literal tagged population changed")
	}
	for i, r := range want.records {
		// Each earlier actor added exactly16 element bytes, no new object.
		at := actors[i].off + i*16
		// The literal writer numbered Player class1/object2, Unit class3,
		// objects4/5, Human class6/object7, Humanoid class8/object9.
		if r.archive != [...]uint16{4, 5, 7, 9}[i] || r.class != [...]string{"Unit", "Unit", "Human", "Humanoid"}[i] || r.off != at || locs[i].RoutesOff != at+41 || locs[i].RawBlocksOff != at+53 || locs[i].ControlOff != at+525 || r.position[11] != 0x3c || r.mover[179] != 0xf4 || r.order[147] != 0x14 {
			t.Fatal("fixture-owned start/byte anchor changed", i, locs[i], r)
		}
		for j := range lists {
			if !slices.Equal(r.routes[j], lists[j]) {
				t.Fatal("three literal lists changed", r.routes)
			}
		}
	}
	return raw, want, state
}

func TestMover1160DocumentLossControls(t *testing.T) {
	for _, identity := range []uint32{0, 0x12345678} {
		t.Run(fmt.Sprintf("identity-%x", identity), func(t *testing.T) {
			_, want, state := mover1160Literal(t, identity)
			for _, r := range want.records {
				index := want.origins[r.archive] - 1
				for _, block := range r.blocks() {
					for i := range block.Bytes {
						t.Run(fmt.Sprintf("%s-%d-%s-byte%d", r.class, r.archive, block.Name, i), func(t *testing.T) {
							bad := unit1156Clone(t, state)
							for _, field := range bad.Document.Objects[index].Raw {
								if field.Name == block.Name {
									field.Bytes[i] ^= 0x80
								}
							}
							if len(want.documentDifferences(bad)) == 0 {
								t.Fatal("accepted decoded block-byte loss")
							}
						})
					}
				}
			}
			index := want.origins[want.records[0].archive] - 1
			for _, name := range mover1160Lists {
				for _, control := range []string{"count", "missing-count", "duplicate-count", "missing-raw", "duplicate-raw", "reverse"} {
					t.Run(name+"-"+control, func(t *testing.T) {
						bad := unit1156Clone(t, state)
						r := &bad.Document.Objects[index]
						for i, f := range r.Counts {
							if f.Name == name {
								switch control {
								case "count":
									r.Counts[i].Count++
								case "missing-count":
									r.Counts = slices.Delete(r.Counts, i, i+1)
								case "duplicate-count":
									r.Counts = append(r.Counts, f)
								}
								break
							}
						}
						for i, f := range r.Raw {
							if f.Name == name {
								switch control {
								case "missing-raw":
									r.Raw = slices.Delete(r.Raw, i, i+1)
								case "duplicate-raw":
									r.Raw = append(r.Raw, f)
								case "reverse":
									slices.Reverse(r.Raw[i].Bytes)
								}
								break
							}
						}
						// The one-element ffff list is unchanged by reversal.
						if name == "U178" && control == "reverse" {
							return
						}
						if len(want.documentDifferences(bad)) == 0 {
							t.Fatal("accepted list loss")
						}
					})
				}
			}
			for _, control := range []string{"nil", "unavailable", "drop-equal", "class", "alias-DTO", "zero-DTO"} {
				t.Run(control, func(t *testing.T) {
					bad := unit1156Clone(t, state)
					changed := want
					changed.origins = map[uint16]uint16{}
					for k, v := range want.origins {
						changed.origins[k] = v
					}
					switch control {
					case "nil":
						bad.Document = nil
					case "unavailable":
						bad.Unavailable = "control"
					case "drop-equal":
						bad.Document.Objects = slices.Delete(bad.Document.Objects, int(index), int(index)+1)
					case "class":
						bad.Document.Objects[index].Class = "Humanoid"
					case "alias-DTO":
						changed.origins[want.records[1].archive] = uint16(index + 1)
					case "zero-DTO":
						changed.origins[want.records[1].archive] = 0
					}
					if len(changed.documentDifferences(bad)) == 0 {
						t.Fatal("accepted population loss")
					}
				})
			}
		})
	}
}

func TestMover1160ReaderControls(t *testing.T) {
	raw, _, _ := mover1160Literal(t, 0)
	f, _ := sav.Open(raw)
	locs, _ := f.DocumentActorLocations()
	objects, _ := f.DocumentObjectLocations()
	_, origins, _ := sav.DecodeDocumentDataWithOrigins(raw)
	for _, control := range []string{"missing-locator", "duplicate-locator", "overlap-routes", "shift-routes", "shift-block", "shift-control", "short-body", "short-count", "huge-count", "duplicate-origin", "alias-origin"} {
		t.Run(control, func(t *testing.T) {
			b, l, ids := slices.Clone(f.Body), slices.Clone(locs), slices.Clone(origins)
			switch control {
			case "missing-locator":
				l = l[1:]
			case "duplicate-locator":
				l = append(l, l[0])
			case "overlap-routes":
				l[0].RoutesOff = l[0].Off
			case "shift-routes":
				l[0].RoutesOff++
			case "shift-block":
				l[0].RawBlocksOff++
			case "shift-control":
				l[0].ControlOff++
			case "short-body":
				b = b[:l[0].RawBlocksOff+461]
			case "short-count":
				l[0].RoutesOff = l[0].RawBlocksOff - 1
			case "huge-count":
				binary.LittleEndian.PutUint16(b[l[0].RoutesOff:], 0xffff)
				binary.LittleEndian.PutUint32(b[l[0].RoutesOff+2:], 0xffffffff)
			case "duplicate-origin":
				ids = append(ids, ids[0])
			case "alias-origin":
				ids[0].ObjectIndex = ids[1].ObjectIndex
			}
			if _, err := mover1160Read(b, l, objects, ids); err == nil {
				t.Fatal("accepted broken independent span/population")
			}
		})
	}
	for _, n := range []int{0, 1, 65535, 65536} {
		b := []byte{0xff, 0xff}
		b = binary.LittleEndian.AppendUint32(b, uint32(n))
		for i := range n {
			b = binary.LittleEndian.AppendUint16(b, uint16(i))
		}
		got, end, err := mover1160Words(b, 0, len(b))
		if err != nil || len(got) != n || end != len(b) || n > 0 && got[n-1] != uint16(n-1) {
			t.Fatal("extended count", n, end, err)
		}
		if _, _, err := mover1160Words(b[:len(b)-1], 0, len(b)-1); err == nil {
			t.Fatal("accepted truncated extended list", n)
		}
	}
}

func mover1160LiveFixture(t *testing.T, stage byte) (*Mission, mover1160Source) {
	t.Helper()
	f := unit1158FixtureFront(t)
	hp := uint16(31)
	if stage == 1 {
		hp = 65505
	}
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: hp, maxHP: 101, mana: 23, maxMana: 103, stage: stage, name: "target", profile: literalProfile1107()}
	b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 31, maxHP: 101, mana: 23, maxMana: 103, name: "survivor", profile: literalProfile1107()}
	raw := completeDocumentTail1115(t, f, savedContainer(poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{a, b}}}}, nil)))
	source, _ := sav.Open(raw)
	want, err := mover1160Expected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	if diff, excluded, p := want.worldDifferences(ms.World, ms.savedDocument); len(diff) != 0 || len(excluded) != 0 || p.compared != 2 {
		t.Fatal("two-source baseline", diff, excluded, p)
	}
	return ms, want
}

func TestMover1160MissingCarrierControls(t *testing.T) {
	for _, stage := range []byte{0, 1} {
		t.Run(fmt.Sprintf("stage%d", stage), func(t *testing.T) {
			ms, want := mover1160LiveFixture(t, stage)
			for _, control := range []string{"entity", "source-binding", "motion", "order", "duplicate-motion", "duplicate-order", "duplicate-entity", "class", "retired", "wrong-DTO"} {
				t.Run(control, func(t *testing.T) {
					entities := ms.World.Entities()
					motions, _, _, _ := ms.World.SavedActorMotions()
					_, orders, _ := ms.World.SavedGroups()
					doc := unit1156Clone(t, ms.savedDocument)
					switch control {
					case "entity":
						entities = entities[1:]
					case "source-binding":
						entities[0].SourceBinding = sim.SourceBinding{}
					case "motion":
						motions = motions[1:]
					case "order":
						orders = orders[1:]
					case "duplicate-motion":
						motions = append(motions, motions[0])
					case "duplicate-order":
						orders = append(orders, orders[0])
					case "duplicate-entity":
						entities = append(entities, entities[0])
					case "class":
						entities[0].SourceBinding.Class = 3
					case "retired":
						doc.Actors[0].Retired = true
					case "wrong-DTO":
						doc.Actors[0].ObjectIndex = want.records[0].archive
					}
					diff, excluded, p := want.carrierDifferences(entities, motions, orders, doc)
					if len(diff) == 0 || len(excluded) != 0 || p.compared == 0 {
						t.Fatal("missing source accepted while survivor remained", diff, excluded, p)
					}
					if (control == "entity" || control == "source-binding") && !strings.Contains(strings.Join(diff, " "), "expected live mover/order source") {
						t.Fatal("missing source not named", diff)
					}
				})
			}
			for _, block := range []struct {
				name string
				size int
			}{{"position", 12}, {"mover", 180}, {"order", 144}} {
				for i := range block.size {
					t.Run(fmt.Sprintf("%s-byte%d", block.name, i), func(t *testing.T) {
						motions, _, _, _ := ms.World.SavedActorMotions()
						_, orders, _ := ms.World.SavedGroups()
						switch block.name {
						case "mover":
							motions[0].Mover[i] ^= 0x80
						case "order":
							orders[0].Raw[i] ^= 0x80
						case "position":
							p := &motions[0].Position
							switch {
							case i < 2:
								p.Cell ^= 0x80 << uint(8*i)
							case i < 4:
								p.PackedCell ^= 0x80 << uint(8*(i-2))
							case i == 4:
								p.FineX ^= 0x80
							case i == 5:
								p.FineY ^= 0x80
							case i < 8:
								p.Residue ^= 0x80 << uint(8*(i-6))
							default:
								p.TerrainKey ^= 0x80 << uint(8*(i-8))
							}
						}
						if diff, _, _ := want.carrierDifferences(ms.World.Entities(), motions, orders, ms.savedDocument); len(diff) == 0 {
							t.Fatal("accepted live byte loss")
						}
					})
				}
			}
		})
	}
}

func TestMover1160LiveListLossControls(t *testing.T) {
	f := unit1158FixtureFront(t)
	raw := mover1160AppFixture(t)
	source, _ := sav.Open(raw)
	want, err := mover1160Expected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	mover1160Initial(t, want, ms)
	for list := range 3 {
		for _, control := range []string{"drop", "truncate", "append", "low-byte", "high-byte"} {
			t.Run(mover1160Lists[list]+"-"+control, func(t *testing.T) {
				motions, _, _, _ := ms.World.SavedActorMotions()
				_, orders, _ := ms.World.SavedGroups()
				var words *[]uint16
				switch list {
				case 0:
					words = &motions[0].StaticRoute
				case 1:
					words = &motions[0].DynamicRoute
				case 2:
					words = &orders[0].Patrol
				}
				if len(*words) == 0 {
					t.Fatal("missing nonempty literal list subject")
				}
				switch control {
				case "drop":
					*words = nil
				case "truncate":
					*words = (*words)[:len(*words)-1]
				case "append":
					*words = append(*words, (*words)[0])
				case "low-byte":
					(*words)[0] ^= 0x80
				case "high-byte":
					(*words)[0] ^= 0x8000
				}
				if diff, _, _ := want.carrierDifferences(ms.World.Entities(), motions, orders, ms.savedDocument); len(diff) == 0 {
					t.Fatal("accepted separate live list loss")
				}
			})
		}
	}
	// Matching corruption in two projected Documents must not pass merely
	// because a paired continuation agrees with the same broken projector.
	bad := unit1156Clone(t, ms.savedDocument)
	for _, r := range bad.Document.Objects {
		if !unit1156Class(r.Class) {
			continue
		}
		for _, field := range r.Raw {
			if field.Name == "U154" {
				field.Bytes[83] ^= 0x80
			}
		}
		break
	}
	if diff, _ := mover1160ContinuationDifferences(bad, bad, ms.World); len(diff) != 0 {
		t.Fatal("paired control should agree with itself", diff)
	}
	if diff := mover1160CurrentMotionDifferences(bad, ms.World); len(diff) == 0 {
		t.Fatal("matching Document corruption escaped current World comparison")
	}
}
