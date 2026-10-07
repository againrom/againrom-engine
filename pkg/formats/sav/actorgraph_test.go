package sav

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func TestActorGraphSavedOrderListsAndNullMembership(t *testing.T) {
	p := graphPlayer1111(2, 10, 900, 0x1111, 1)
	a := graphActor1111(4, 200, 0xaaaa, 0x1111)
	a.Raw["U158"], a.Raw["U50"] = make([]byte, 148), []byte{0x0a, 0, 0, 0}
	a.Raw["U158"][2], a.Raw["U158"][3] = 8, 8
	binary.LittleEndian.PutUint32(a.Raw["U158"][4:], 0x12345678)
	a.Raw["U158_90"], a.Counts["U158_90"] = []byte{8, 8, 9, 9, 8, 8, 10, 10}, 4
	g := graphGroup1111(p, 100, 500, 9, 0, a, nil)
	g.Raw["G4C"], g.Counts["G4C"] = []byte{11, 11, 12, 12}, 2
	graph, err := savedActorGraph([]*Record{p})
	if err != nil {
		t.Fatal(err)
	}
	o := graph.Actors[0]
	if !o.HasOrder || o.ActorState != 0xa || binary.LittleEndian.Uint32(o.Order[4:]) != 0x12345678 || !reflect.DeepEqual(o.Patrol, []uint16{0x0808, 0x0909, 0x0808, 0x0a0a}) {
		t.Fatal("actor operands lost", o)
	}
	if !reflect.DeepEqual(graph.Groups[0].Members, []uint16{4, 0}) || !reflect.DeepEqual(graph.Groups[0].AIWords, []uint16{0x0b0b, 0x0c0c}) {
		t.Fatal("null member or separate path lost", graph.Groups)
	}
}

func TestActorGraphKeepsInlineGroupsAndSourceNamespaces(t *testing.T) {
	var chars []wantChar
	for i := 1; i <= 3; i++ {
		c := wantChar{name: "saved", runtimeID: uint32(i), defRow: 13, mapUnitID: 0, journal: -1}
		c.stats[StatHealth], c.stats[StatHealthMax] = 7, 31
		chars = append(chars, c)
	}
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: chars, groups: 2})
	graph, err := f.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Actors) != 3 || len(graph.Groups) != 2 {
		t.Fatalf("population actors=%d groups=%d", len(graph.Actors), len(graph.Groups))
	}
	// Player class/object take1/2; Human class takes3, then the three bare
	// objects take4/5/6. Inline Groups and Diary must consume no index.
	for i, want := range []struct {
		index          uint16
		runtime, group uint32
	}{{4, 1, 1}, {5, 3, 1}, {6, 2, 2}} {
		a := graph.Actors[i]
		if a.ArchiveIndex != want.index || a.RuntimeID != want.runtime || a.Identity != 0xa0000000|want.runtime || a.Group != want.group ||
			a.MapUnitID != 0 || a.TokenOwnerSlot != 0 || a.OwnerSlot != 1 {
			t.Fatalf("actor%d: %+v", i, a)
		}
		if row, err := a.DefinitionRow(); err != nil || row != 5 || a.DefRow != 13 || a.TypeID != 33 {
			t.Fatalf("row=%d err=%v selectors=%d/%d", row, err, a.DefRow, a.TypeID)
		}
		if a.TokenSize != 0x49 || a.Domain != 0x4a || a.Face != 0x4b || a.ClassSelector != 0x4c || a.Facing != 0x54 {
			t.Fatalf("raw selector projection: %+v", a)
		}
	}
	if !reflect.DeepEqual(graph.Groups[0].Members, []uint16{4, 5}) || !reflect.DeepEqual(graph.Groups[1].Members, []uint16{6}) {
		t.Fatalf("members=%v/%v", graph.Groups[0].Members, graph.Groups[1].Members)
	}
	for _, group := range graph.Groups {
		if group.Selector != 0x1c1c1c1c || group.Owner.Key != 0x44444444 || group.Owner.Resolved || group.Reference.Key != 0x40404040 || group.AI[0] != 0x3c {
			t.Fatalf("group=%+v", group)
		}
	}
	graph.Groups[0].Members[0], graph.Groups[0].AI[0] = 99, 0
	fresh, err := f.ActorGraph()
	if err != nil || fresh.Groups[0].Members[0] != 4 || fresh.Groups[0].AI[0] != 0x3c {
		t.Fatalf("graph aliases source: %+v %v", fresh.Groups, err)
	}
}

func graphActor1111(index uint16, off int, key, tokenOwner uint32) *Record {
	r := newRecord("Unit", off, index)
	r.End = off + 20
	r.Value["Identity"], r.Value["Reference"], r.Value["RuntimeID"] = key, tokenOwner, uint32(index)
	r.Value["Health"], r.Value["HealthMax"], r.Value["T0C"] = 7, 31, 4
	r.Raw["Block12"], r.Raw["U154"], r.Raw["UA6"] = make([]byte, 12), make([]byte, 180), make([]byte, 24)
	r.Raw["UBE"], r.Raw["U114"], r.Raw["UD4"] = make([]byte, 22), make([]byte, 24), make([]byte, 64)
	return r
}

func graphPlayer1111(index uint16, off, end int, key, slot uint32) *Record {
	r := newRecord("Player", off, index)
	r.End, r.Value["This"], r.Value["Slot"] = end, key, slot
	return r
}

func graphGroup1111(player *Record, off, end int, selector, owner uint32, members ...*Record) *Record {
	r := newRecord("Group", off, 0)
	r.End, r.Value["G1C"], r.Value["G44"] = end, selector, owner
	r.Raw["G3C"] = make([]byte, 80)
	r.Refs["Actors"], r.Counts["Actors"] = members, len(members)
	player.Groups = append(player.Groups, r)
	player.Refs["Actors"] = append(player.Refs["Actors"], members...)
	return r
}

func TestActorGraphReplaysDetachAndPlayerSuffixWithoutChangingGroupOwner(t *testing.T) {
	p := graphPlayer1111(2, 10, 900, 0x1111, 9)
	q := graphPlayer1111(6, 1000, 1900, 0x2222, 1)
	p.Value["F58"], q.Value["F58"] = 17, 63
	a := graphActor1111(4, 200, 0xaaaa, 0x1111)
	b := graphActor1111(5, 300, 0xbbbb, 0x2222) // forward Token reference
	first := graphGroup1111(p, 100, 500, 0x12345678, 0x2222, a, b, a)
	first.Raw["G20"], first.Counts["G20"] = []byte{7, 0, 9, 0}, 2
	first.Raw["G4C"], first.Counts["G4C"] = []byte{0x34, 0x12}, 1
	second := graphGroup1111(q, 1100, 1500, 0x12345678, 0x1111, a)
	second.Value["G40"] = 0xaaaa
	graph, err := savedActorGraph([]*Record{p, q, p})
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Actors) != 2 || len(graph.Groups) != 2 {
		t.Fatalf("aliases duplicated graph: %+v", graph)
	}
	if graph.Actors[0].OwnerSlot != 1 || graph.Actors[0].TokenOwnerSlot != 9 || graph.Actors[0].Group != 2 ||
		graph.Actors[1].OwnerSlot != 9 || graph.Actors[1].TokenOwnerSlot != 0 || graph.Actors[1].Group != 1 {
		t.Fatalf("owner/membership precedence: %+v", graph.Actors)
	}
	if graph.Actors[0].Owner.ArchiveIndex != 6 || graph.Actors[0].Owner.PlayerF58 != 63 ||
		graph.Actors[1].Owner.ArchiveIndex != 2 || graph.Actors[1].Owner.PlayerF58 != 17 {
		t.Fatalf("effective owner inputs: %+v", graph.Actors)
	}
	if !reflect.DeepEqual(graph.Groups[0].Members, []uint16{5}) || !reflect.DeepEqual(graph.Groups[1].Members, []uint16{4}) ||
		graph.Groups[0].Owner.Resolved || graph.Groups[1].Owner.PlayerSlot != 9 || graph.Groups[1].Reference.ArchiveIndex != 4 {
		t.Fatalf("group remap or append order: %+v", graph.Groups)
	}
	if !reflect.DeepEqual(graph.Groups[0].Words, []uint16{7, 9}) || !reflect.DeepEqual(graph.Groups[0].AIWords, []uint16{0x1234}) {
		t.Fatalf("word lists: %+v", graph.Groups[0])
	}
}

func TestActorGraphRejectsLateIdentityAndMissingHumanoidFields(t *testing.T) {
	for _, kind := range []string{"duplicate identity", "missing Humanoid XP"} {
		t.Run(kind, func(t *testing.T) {
			p := graphPlayer1111(2, 10, 900, 0x1111, 1)
			a, b := graphActor1111(4, 200, 0xaaaa, 0x1111), graphActor1111(5, 300, 0xbbbb, 0x1111)
			if kind == "duplicate identity" {
				b.Value["Identity"] = a.Value["Identity"]
			} else {
				b.Class = "Humanoid"
			}
			graphGroup1111(p, 100, 500, 7, 0x1111, a, b)
			got, err := savedActorGraph([]*Record{p})
			if err == nil || len(got.Actors) != 0 || len(got.Groups) != 0 {
				t.Fatalf("partial publication: %+v %v", got, err)
			}
		})
	}
}

func TestSavedActorDefinitionUsesExactClassAndUnsignedType(t *testing.T) {
	for _, test := range []struct {
		class     string
		typ       uint16
		row, want uint8
	}{
		{"Unit", 65535, 7, 7}, {"Human", 0, 7, 7}, {"Human", 32, 7, 7},
		{"Human", 33, 7, 5}, {"Human", 32768, 7, 5}, {"Human", 65535, 7, 5},
	} {
		a := ActorRecord{Actor: Actor{Class: test.class}, TypeID: test.typ, DefRow: test.row}
		if got, err := a.DefinitionRow(); got != test.want || err != nil {
			t.Fatalf("%s type%d row%d -> %d %v", test.class, test.typ, test.row, got, err)
		}
	}
	if _, err := (ActorRecord{Actor: Actor{Class: "Humanoid"}}).DefinitionRow(); err == nil || !strings.Contains(err.Error(), "unsupported") && !strings.Contains(err.Error(), "no supported") {
		t.Fatalf("exact Humanoid: %v", err)
	}
}

// wantMover builds a 180-byte mover block with only its named byte fields
// set: DesiredFacing at +1 (MOVE-TURN-031), the passability mask at +5
// (TERR-PASS-051), RotationSpeed at +0xa (MOVE-TURN-031). Every other byte
// is fill, matching the raw-block marker-fill convention every other field
// in this fixture already uses; a caller comparing two movers byte-for-byte
// should pass different fill values so an unwritten byte is not mistaken for
// one the write left alone.
func wantMover(fill, desiredFacing, mask, rotationSpeed uint8, turnInProgress bool) []byte {
	m := make([]byte, 180)
	for i := range m {
		m[i] = fill
	}
	m[1], m[5], m[0xa] = desiredFacing, mask, rotationSpeed
	if turnInProgress {
		m[0xa0] = 1
	} else {
		m[0xa0] = 0
	}
	return m
}

func TestActorGraphMoverAndRouteFromArchive(t *testing.T) {
	h := hero()
	h.mover = wantMover(0x54, 0x60, 0x44, 12, true)
	h.staticRoute = []uint16{0x0c14, 0x0c15, 0x0d16}
	h.dynamicRoute = []uint16{0x0a0a}
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{h}})
	graph, err := f.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Actors) != 1 {
		t.Fatalf("population actors=%d", len(graph.Actors))
	}
	a := graph.Actors[0]
	if !a.HasMover || !bytes.Equal(a.Mover[:], h.mover) {
		t.Fatalf("mover block lost: hasMover=%v", a.HasMover)
	}
	if a.DesiredFacing() != 0x60 || a.PassabilityMask() != 0x44 || a.RotationSpeed() != 12 || !a.TurnInProgress() {
		t.Fatalf("named mover fields: facing=%#x mask=%#x speed=%d turning=%v",
			a.DesiredFacing(), a.PassabilityMask(), a.RotationSpeed(), a.TurnInProgress())
	}
	if a.Facing != 0x54 {
		// mover[0] (current facing) is the fixture's 0x54 filler in wantMover,
		// but Actor.Facing already reads it independently of this story
		// (actors.go); this only guards the two bytes are not swapped.
		t.Fatalf("Actor.Facing should read mover[0]=0x54, got %#x", a.Facing)
	}
	if !reflect.DeepEqual(a.StaticRoute, h.staticRoute) || !reflect.DeepEqual(a.DynamicRoute, h.dynamicRoute) {
		t.Fatalf("route lists: static=%v dynamic=%v", a.StaticRoute, a.DynamicRoute)
	}
	if x, y := RouteCell(0x0c14); x != 0x14 || y != 0x0c {
		t.Fatalf("RouteCell(0x0c14) = (%d,%d), want (20,12)", x, y)
	}
	// A fresh graph aliases the same file bytes rather than copying them.
	fresh, err := f.ActorGraph()
	if err != nil || !reflect.DeepEqual(fresh.Actors[0].StaticRoute, h.staticRoute) {
		t.Fatalf("second ActorGraph call: %v %v", err, fresh.Actors[0].StaticRoute)
	}
}

func TestActorGraphEmptyRouteListsDecodeAsNil(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	graph, err := f.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	a := graph.Actors[0]
	if len(a.StaticRoute) != 0 || len(a.DynamicRoute) != 0 {
		t.Fatalf("a fixture that never set a route should decode empty: %+v %+v", a.StaticRoute, a.DynamicRoute)
	}
	if !a.HasMover || a.PassabilityMask() != 0x54 {
		// The fixture's default mover fill (0x54) is not a real passability
		// mask value (SAV, TERR-PASS-051 ships 0x41/0x44/0x82); this only
		// confirms the accessor reads the byte the fixture actually wrote.
		t.Fatalf("default mover fill lost: hasMover=%v mask=%#x", a.HasMover, a.PassabilityMask())
	}
}

// mover180LayoutBytes are every Mover byte offset this story or an earlier
// one names. A future change to any of them must update this set, which is
// half of what TestMoverAndOrderLayoutAccountsForEveryByte checks; the rest
// of the 180 bytes is the other half, one explicit raw span, not a claim
// that it is unimportant, only that it is untyped here.
var mover180LayoutBytes = map[int]string{
	0:    "Actor.Facing (current facing, actors.go, MOVE-TURN-031)",
	1:    "DesiredFacing (MOVE-TURN-031)",
	5:    "PassabilityMask (TERR-PASS-051)",
	0xa:  "RotationSpeed (MOVE-TURN-031)",
	0xa0: "TurnInProgress (MOVE-TURN-031)",
}

// TestMoverAndOrderLayoutAccountsForEveryByte proves both raw blocks story
// 1134 (and 1115 before it) touch are accounted for in full, named field or
// explicit raw span, on TestSessionBlockLayoutAccountsForEveryByte's own
// shape (session1130_test.go): every named byte plus the untyped remainder
// must sum to exactly the block's own live width, so a widened or narrowed
// named field silently opening or closing a gap fails here instead of only
// in a byte-level round trip elsewhere.
//
// The 180-byte Mover block names five single bytes (mover180LayoutBytes);
// the other 175 are one explicit raw span carried whole as
// ActorRecord.Mover (SAV-CROSSNEXT-585, DIV-950). The 148-byte U158/Order
// block names none at all: program.go declares it KindRaw, Len 148, and
// this story adds no accessor into it — it is carried whole as
// ActorRecord.Order, with HasOrder marking presence. The distinct
// "*(*(+0x158)+0x90)" list (SAV-633) is U158_90, a separately appended u16
// list outside these 148 bytes, not a sub-span of them (ActorRecord.Patrol);
// it plays no part in this accounting.
func TestMoverAndOrderLayoutAccountsForEveryByte(t *testing.T) {
	moverLen := len(ActorRecord{}.Mover)
	for off := range mover180LayoutBytes {
		if off < 0 || off >= moverLen {
			t.Fatalf("named mover offset %#x outside the %d-byte block", off, moverLen)
		}
	}
	if len(mover180LayoutBytes) != 5 {
		t.Fatalf("%d named mover bytes, want exactly 5 (the rest is an explicit raw span, SAV-CROSSNEXT-585)", len(mover180LayoutBytes))
	}
	if named, raw, want := len(mover180LayoutBytes), moverLen-len(mover180LayoutBytes), 180; named+raw != want || moverLen != want {
		t.Fatalf("accounted mover bytes = %d named + %d raw = %d of %d, want %d of %d", named, raw, named+raw, moverLen, want, want)
	}

	orderLen := len(ActorRecord{}.Order)
	const orderNamed, want = 0, 148
	if orderLen != want {
		t.Fatalf("ActorRecord.Order is %d bytes (%d named + %d raw), want %d", orderLen, orderNamed, orderLen-orderNamed, want)
	}
}

func TestSetActorMoverRouteLeavesUnrelatedBytesAlone(t *testing.T) {
	h := hero()
	h.mover = wantMover(0x20, 0x20, 0x41, 8, false)
	h.staticRoute = []uint16{0x0c14, 0x0c15}
	h.dynamicRoute = []uint16{0x0a0a, 0x0b0b, 0x0c0c}
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{h, merc()}})
	before := append([]byte(nil), f.Body...)
	graph, err := f.ActorGraph()
	if err != nil {
		t.Fatalf("ActorGraph: %v", err)
	}
	a := graph.Actors[0]
	// A fill distinct from 0x20 above (not just the four named bytes) so
	// every one of the 180 mover bytes actually changes value below.
	newMover := wantMover(0x77, 0x60, 0x44, 12, true)
	var mover [180]byte
	copy(mover[:], newMover)
	newStatic := []uint16{0x0d16, 0x0d17}
	newDynamic := []uint16{0x0909, 0x0808, 0x0707}
	if err := f.SetActorMoverRoute(a.ArchiveIndex, mover, newStatic, newDynamic); err != nil {
		t.Fatalf("SetActorMoverRoute: %v", err)
	}
	if len(f.Body) != len(before) {
		t.Fatalf("SetActorMoverRoute changed the body length: %d -> %d", len(before), len(f.Body))
	}
	n := 0
	for i := range before {
		if before[i] != f.Body[i] {
			n++
		}
	}
	// Exactly the 180 mover bytes plus 2 static and 3 dynamic route u16s (10
	// bytes) moved; nothing outside actor 0's own two spans.
	if want := 180 + 2*2 + 3*2; n != want {
		t.Fatalf("%d bytes moved, want exactly %d (mover, static route, dynamic route)", n, want)
	}
	after, err := f.ActorGraph()
	if err != nil {
		t.Fatalf("ActorGraph after write: %v", err)
	}
	got := after.Actors[0]
	if !bytes.Equal(got.Mover[:], newMover) || !reflect.DeepEqual(got.StaticRoute, newStatic) || !reflect.DeepEqual(got.DynamicRoute, newDynamic) {
		t.Fatalf("patched values did not round-trip: mover=%v static=%v dynamic=%v", got.Mover[:6], got.StaticRoute, got.DynamicRoute)
	}
	if got.RuntimeID != a.RuntimeID || after.Actors[1].RuntimeID != graph.Actors[1].RuntimeID {
		t.Fatalf("the write disturbed unrelated actor identity: %+v / %+v", got, after.Actors[1])
	}
	if f.World == nil || len(f.World.Blocks) != 0 {
		t.Fatalf("the world half after the mover/route write was disturbed: %+v", f.World)
	}
	chars, err := f.Party()
	if err != nil || len(chars) != 2 {
		t.Fatalf("the party after the mover/route write was disturbed: %v, %d chars", err, len(chars))
	}
}

func TestSetActorMoverRouteRefusesAShapeMismatch(t *testing.T) {
	h := hero()
	h.staticRoute = []uint16{0x0c14, 0x0c15}
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{h}})
	graph, err := f.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	a := graph.Actors[0]
	// F-5: a refused call must leave the file exactly as it found it, mover
	// bytes included, rather than patch the 180-byte mover block before
	// reaching the route-length check that then refuses. before/f.Body is
	// TestSetActorMoverRouteLeavesUnrelatedBytesAlone's own byte-level
	// pattern, here proving zero bytes move rather than exactly the right
	// ones.
	before := append([]byte(nil), f.Body...)
	newMover := a.Mover
	newMover[0] ^= 0xff // a mover distinct from the file's own, so a mistaken write is not masked by coincidence
	if err := f.SetActorMoverRoute(a.ArchiveIndex, newMover, []uint16{1, 2, 3}, nil); err == nil {
		t.Fatal("a static route with the wrong element count should be refused")
	}
	if !bytes.Equal(f.Body, before) {
		t.Fatal("a refused static route should not have patched the mover block first")
	}
	if err := f.SetActorMoverRoute(a.ArchiveIndex, newMover, nil, []uint16{1}); err == nil {
		t.Fatal("a dynamic route the file declared empty should refuse a nonempty replacement")
	}
	if !bytes.Equal(f.Body, before) {
		t.Fatal("a refused dynamic route should not have patched the mover block first")
	}
	if err := f.SetActorMoverRoute(9999, a.Mover, nil, nil); err == nil {
		t.Fatal("an unknown archive index should be refused")
	}
	if err := f.SetActorMoverRoute(a.ArchiveIndex, a.Mover, a.StaticRoute, nil); err != nil {
		t.Fatalf("the file's own unchanged shape should be accepted: %v", err)
	}
}
