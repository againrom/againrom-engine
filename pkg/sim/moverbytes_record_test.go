package sim

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// moverDigest names 180 Mover bytes by the first eight bytes of their SHA-256.
func moverDigest(m [180]byte) string {
	s := sha256.Sum256(m[:])
	return fmt.Sprintf("%x", s[:8])
}

// checkMoverRecords compares got with the recorded want and prints the whole
// table when any entry differs, so a new record is one paste.
func checkMoverRecords(t *testing.T, got, want map[string]string) {
	t.Helper()
	bad := len(got) != len(want)
	for k, v := range got {
		if want[k] != v {
			bad = true
			t.Errorf("%s = %s, recorded %s", k, v, want[k])
		}
	}
	if bad {
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		for _, k := range keys {
			fmt.Fprintf(&b, "\t%q: %q,\n", k, got[k])
		}
		t.Fatalf("records differ; now:\n%s", b.String())
	}
}

// moverRecordActors returns a World with a ground AI actor, the controlled
// player's actor and an air actor, each with its own facing and rate.
func moverRecordActors(t *testing.T) *World {
	t.Helper()
	w := moverWorld(t)
	for i := range w.entities {
		w.entities[i].Facing = byte(0x30 + 0x21*i)
		w.entities[i].DesiredFacing = byte(0x50 + 0x13*i)
		w.entities[i].RotationSpeed = int32(7 + i)
		w.entities[i].PostX, w.entities[i].PostY = int32(2+i), int32(3+i)
	}
	return w
}

// TestMoverBytesRecordSim records the 180 Mover bytes every pkg/sim writer
// writes for each actor kind it serves, and the hash of a small World built
// through each production path after a few hundred steps.
func TestMoverBytesRecordSim(t *testing.T) {
	got := map[string]string{}
	w := moverRecordActors(t)
	for _, id := range []EntityID{1, 2, 3} {
		e, _ := w.Entity(id)
		got[fmt.Sprintf("placed/%d", id)] = moverDigest(placedMover(e, false))
		got[fmt.Sprintf("placed-joined/%d", id)] = moverDigest(placedMover(e, true))
		got[fmt.Sprintf("constructed/%d", id)] = moverDigest(w.constructedMover(e, false))
		got[fmt.Sprintf("constructed-joined/%d", id)] = moverDigest(w.constructedMover(e, true))
	}
	w.ConstructMissionMovers([]EntityID{2})
	for _, id := range []EntityID{1, 2, 3} {
		var m [180]byte
		for i := range m {
			m[i] = byte(i * 7)
		}
		ok := w.ApplyMover(id, &m)
		got[fmt.Sprintf("apply/%d/%v", id, ok)] = moverDigest(m)
	}
	var claim [180]byte
	putWalkClaim(&claim, walkClaim{Target: 0x1234, Claim: 0x0506, Arrived: true})
	got["walkclaim/arrived"] = moverDigest(claim)
	putWalkClaim(&claim, walkClaim{Target: 0x0102, Claim: 0x0304})
	got["walkclaim/lowered"] = moverDigest(claim)

	// Current motion: fresh and kept, idle, with a stride, unrated, turning,
	// with a turn state and with a route.
	project := func(name string, edit func(*Entity), fresh bool, route [][2]int32) {
		for _, id := range []EntityID{1, 2, 3} {
			e, _ := w.Entity(id)
			if edit != nil {
				edit(&e)
			}
			old := SavedActorMotion{Entity: id}
			if !fresh {
				for i := range old.Mover {
					old.Mover[i] = byte(i*13 + 1)
				}
			} else {
				w.ApplyMover(id, &old.Mover)
			}
			m, err := ProjectActorMotion(e, old, route, fresh)
			key := fmt.Sprintf("motion/%s/%d/%v", name, id, fresh)
			if err != nil {
				got[key] = "error"
				continue
			}
			got[key] = moverDigest(m.Mover) + "/" + moverDigest(m.SavedMover())
		}
	}
	for _, fresh := range []bool{true, false} {
		project("idle", nil, fresh, nil)
		project("target", func(e *Entity) { e.HasTarget, e.TargetX, e.TargetY = true, 9, 10 }, fresh, [][2]int32{{4, 4}, {5, 5}, {5, 5}})
		project("stride", func(e *Entity) {
			e.Transit, e.TransitTotal = 3, 8
			e.Stride = NativeStride{Present: true, FromX: e.X - 1, FromY: e.Y, ToX: e.X, ToY: e.Y, Rate: 32, StepX: 32, Direction: 2}
		}, fresh, [][2]int32{{6, 6}})
		project("unrated", func(e *Entity) { e.Transit, e.TransitTotal = 2, 5 }, fresh, nil)
		project("turning", func(e *Entity) { e.TurnTotal, e.TurnRemaining = 6, 4 }, fresh, nil)
		project("turnstate", func(e *Entity) { e.TurnState = TurnState{Present: true, Active: true, Counter: 3} }, fresh, nil)
		project("turnstate-idle", func(e *Entity) { e.TurnState = TurnState{Present: true, Counter: 9} }, fresh, nil)
	}

	// The imported-motion stepper: a crossing, its arrival, and the turn leaf.
	mw := mustWorld(t, 7, Bounds{32, 32}, []Entity{{ID: 7, Owner: 1, X: 15, Y: 16, HP: 100, MaxHP: 100, Speed: 1000, Facing: 64}})
	o := SavedActorOrder{Entity: 7, State: 0xb}
	o.Raw[8], o.Raw[9], o.Raw[10], o.Raw[11] = 1, 3, 20, 16
	g := SavedGroup{ID: 1, Selector: 1, Members: []SavedGroupMember{{Archive: 1, Entity: 7, Bound: true}}}
	g.AI[0x20], g.AI[0x45] = 4, 1
	if err := mw.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
		t.Fatal(err)
	}
	m := SavedActorMotion{Entity: 7, Position: SavedActorPosition{Cell: 0x100f, PackedCell: 0x100f, FineX: 224, FineY: 128}, StaticRoute: []uint16{0x1010, 0x1011}, DynamicRoute: []uint16{0x1010}}
	m.Mover[0], m.Mover[1], m.Mover[10], m.Mover[0x98] = 64, 64, 100, 1
	for _, at := range []int{6, 0x74, 0x76, 0x80, 0xa6} {
		binary.LittleEndian.PutUint16(m.Mover[at:], 0x1010)
	}
	for at, v := range map[int]uint16{0x70: 0x100f, 0xa8: 32, 0xaa: 8, 0xac: 3, 0xae: 2} {
		binary.LittleEndian.PutUint16(m.Mover[at:], v)
	}
	m.Mover[0xb0] = 32
	cells := []SavedActorCell{{Cell: 0x100f, Ground: SavedActorSlot{Key: 0x1004, Entity: 7, Bound: true}}, {Cell: 0x1010}}
	binary.LittleEndian.PutUint32(cells[0].Payload[4:], 0x1004)
	blocks := []SavedActorBlock{{Cell: 0x100f, Dyn: 0x43, Static: 7}, {Cell: 0x1010, Dyn: 0x45, Static: 9}}
	if err := mw.ImportOriginalActorMotions([]SavedActorMotion{m}, cells, blocks); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 6; i++ {
		Step(mw, nil)
		got[fmt.Sprintf("crossing/%d", i)] = moverDigest(mw.motionFor(7).Mover)
	}
	for i := 7; i <= 300; i++ {
		Step(mw, nil)
	}
	got["crossing/hash"] = fmt.Sprintf("%016x", mw.Hash())
	tw := savedTurnWorldForTest(t, 0, 96, 16, 1, 0)
	for i := 1; i <= 7; i++ {
		Step(tw, nil)
		if mm := tw.motionFor(7); mm != nil {
			got[fmt.Sprintf("turn/%d", i)] = moverDigest(mm.Mover)
		}
	}
	for i := 8; i <= 300; i++ {
		Step(tw, nil)
	}
	got["turn/hash"] = fmt.Sprintf("%016x", tw.Hash())
	pw := savedTurnWorldForTest(t, 64, 64, 100, 0, 0)
	s := pw.entities[0].ActorLoad.Source
	s.MoverSpeed = 42
	pw.publishSource(0, s)
	got["source/rate"] = moverDigest(pw.motionFor(7).Mover)

	// A built World after a walk, and its SAVE bytes after the walk.
	hw := moverRecordActors(t)
	hw.ConstructMissionMovers([]EntityID{2})
	Step(hw, []Command{MoveTo(2, CellPoint{X: 12, Y: 8}), MoveTo(1, CellPoint{X: 9, Y: 14})})
	for i := 0; i < 300; i++ {
		Step(hw, nil)
	}
	got["walk/hash"] = fmt.Sprintf("%016x", hw.Hash())
	for _, id := range []EntityID{1, 2, 3} {
		e, _ := hw.Entity(id)
		old := SavedActorMotion{Entity: id}
		hw.ApplyMover(id, &old.Mover)
		if mm, err := ProjectActorMotion(e, old, hw.Route(id), true); err == nil {
			got[fmt.Sprintf("walk/%d", id)] = moverDigest(mm.SavedMover())
		}
	}
	var step [180]byte
	binary.LittleEndian.PutUint16(step[0x80:], 0x0a0b)
	got["savedmover/claim"] = moverDigest(SavedActorMotion{Mover: step}.SavedMover())

	checkMoverRecords(t, got, simMoverRecords)
}

// simMoverRecords holds the bytes the writers wrote before they had one
// producer.
var simMoverRecords = map[string]string{
	"apply/1/true":                  "ee82e01ea10fe0fd",
	"apply/2/true":                  "f275095d49f9ddb6",
	"apply/3/true":                  "2c3687a7e35e6ab9",
	"constructed-joined/1":          "a33dd6e81e918225",
	"constructed-joined/2":          "a33dd6e81e918225",
	"constructed-joined/3":          "a33dd6e81e918225",
	"constructed/1":                 "99133096157d368e",
	"constructed/2":                 "0713aa671e9ceea3",
	"constructed/3":                 "462f6070f1ffb2ef",
	"crossing/1":                    "9f18bcfd2a49e30b",
	"crossing/2":                    "3cf313af0eedade7",
	"crossing/3":                    "87fa7c1357a1ecfe",
	"crossing/4":                    "245dff59fc2fb1b3",
	"crossing/5":                    "f37ae1826b3692c2",
	"crossing/6":                    "f37ae1826b3692c2",
	"crossing/hash":                 "29d746b19f23c0a5",
	"motion/idle/1/false":           "a4dd43d876278044/a4dd43d876278044",
	"motion/idle/1/true":            "811a34ac9d40ed98/811a34ac9d40ed98",
	"motion/idle/2/false":           "5717e9c9483c9180/5717e9c9483c9180",
	"motion/idle/2/true":            "b438b7209349d10d/b438b7209349d10d",
	"motion/idle/3/false":           "d0ceaea81f196895/d0ceaea81f196895",
	"motion/idle/3/true":            "4f800573d01727e8/4f800573d01727e8",
	"motion/stride/1/false":         "ac358b0d77fb9159/ac358b0d77fb9159",
	"motion/stride/1/true":          "73733d50c4c76a46/73733d50c4c76a46",
	"motion/stride/2/false":         "116db8dcec160f8d/116db8dcec160f8d",
	"motion/stride/2/true":          "02c4bf0205bd024d/02c4bf0205bd024d",
	"motion/stride/3/false":         "991f10ad0eb549ce/991f10ad0eb549ce",
	"motion/stride/3/true":          "d79ae35d83f01f2a/d79ae35d83f01f2a",
	"motion/target/1/false":         "58b6c75de6bb5fe8/58b6c75de6bb5fe8",
	"motion/target/1/true":          "de224aa48a4a204d/de224aa48a4a204d",
	"motion/target/2/false":         "d8d83c4e9516d463/d8d83c4e9516d463",
	"motion/target/2/true":          "c6df587f174bd59b/c6df587f174bd59b",
	"motion/target/3/false":         "e5020793636bb005/e5020793636bb005",
	"motion/target/3/true":          "92d63ed2e2b7ad86/92d63ed2e2b7ad86",
	"motion/turning/1/false":        "df3c449ed90596b8/df3c449ed90596b8",
	"motion/turning/1/true":         "6fd7540140e1f932/6fd7540140e1f932",
	"motion/turning/2/false":        "2b7515cddd66775e/2b7515cddd66775e",
	"motion/turning/2/true":         "6885f44bffd8b3d2/6885f44bffd8b3d2",
	"motion/turning/3/false":        "fa4bfce69827929d/fa4bfce69827929d",
	"motion/turning/3/true":         "585c7231f714f11e/585c7231f714f11e",
	"motion/turnstate-idle/1/false": "ff15147620063edb/ff15147620063edb",
	"motion/turnstate-idle/1/true":  "7d33817249e71aa6/7d33817249e71aa6",
	"motion/turnstate-idle/2/false": "6bf541674ea47dca/6bf541674ea47dca",
	"motion/turnstate-idle/2/true":  "058676269f498ef6/058676269f498ef6",
	"motion/turnstate-idle/3/false": "a1cf539c9bf1ba9c/a1cf539c9bf1ba9c",
	"motion/turnstate-idle/3/true":  "c0f3c585084c96f7/c0f3c585084c96f7",
	"motion/turnstate/1/false":      "5234c43378103fb3/5234c43378103fb3",
	"motion/turnstate/1/true":       "7b8e41bf4a61fb8a/7b8e41bf4a61fb8a",
	"motion/turnstate/2/false":      "c6d54cebf7605b6f/c6d54cebf7605b6f",
	"motion/turnstate/2/true":       "ea4bb1281b08938a/ea4bb1281b08938a",
	"motion/turnstate/3/false":      "3701bbd06350f6db/3701bbd06350f6db",
	"motion/turnstate/3/true":       "d5369eed18184c3a/d5369eed18184c3a",
	"motion/unrated/1/false":        "ea309be3e4ddc63a/ea309be3e4ddc63a",
	"motion/unrated/1/true":         "b409c4cb1e907f60/b409c4cb1e907f60",
	"motion/unrated/2/false":        "6085f6e1dc1b498c/6085f6e1dc1b498c",
	"motion/unrated/2/true":         "3448a7172d0a09ff/3448a7172d0a09ff",
	"motion/unrated/3/false":        "7a96effc61d9321d/7a96effc61d9321d",
	"motion/unrated/3/true":         "f5dfd72ba7f85bec/f5dfd72ba7f85bec",
	"placed-joined/1":               "a33dd6e81e918225",
	"placed-joined/2":               "a33dd6e81e918225",
	"placed-joined/3":               "a33dd6e81e918225",
	"placed/1":                      "1fb38873b30f6f41",
	"placed/2":                      "ec245b70052975f2",
	"placed/3":                      "2ffa83adb1a8179a",
	"savedmover/claim":              "b3f3cb8a04c9e595",
	"source/rate":                   "75cde050d3a03a61",
	"turn/1":                        "d98208ad1440c94d",
	"turn/2":                        "7c7bde90ce0f40d1",
	"turn/3":                        "1c5075f54a60c423",
	"turn/4":                        "90cdafef5ee756c7",
	"turn/5":                        "dd294db5839bcc03",
	"turn/6":                        "e0ccdb3617e91504",
	"turn/7":                        "378bc93fecf6e691",
	"turn/hash":                     "1c64ededda2a47be",
	"walk/1":                        "8536d2a521071fa4",
	"walk/2":                        "ae1ddd969bcdc990",
	"walk/3":                        "4f800573d01727e8",
	"walk/hash":                     "ee3ec3237b0d5667",
	"walkclaim/arrived":             "2b337b2e6ff5306f",
	"walkclaim/lowered":             "a2ca7420394d343a",
}
