package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func outpostActor(id EntityID, owner, group uint32, x, y int32) Entity {
	return Entity{ID: id, TypeID: 1, HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1, Owner: owner, Group: group, X: x, Y: y}
}

// outpostWorld is a 32x32 second-game World: the human's unit 1 (group 3),
// owner 2's group 7 (units 2 and 3) and group 8 (unit 4), owner 3's group 7
// (unit 5).
func outpostWorld(t *testing.T) *World {
	t.Helper()
	return secondWorld(t, 1, Bounds{Width: 32, Height: 32}, nil,
		outpostActor(1, SelfSlot, 3, 9, 20), outpostActor(2, 2, 7, 10, 10), outpostActor(3, 2, 7, 13, 11),
		outpostActor(4, 2, 8, 20, 20), outpostActor(5, 3, 7, 20, 12))
}

func outpostOf(t *testing.T, w *World, owner, group uint32) Outpost {
	t.Helper()
	for _, o := range w.Outposts() {
		if o.Owner == owner && o.Group == group {
			return o
		}
	}
	t.Fatalf("no Outpost for owner %d group %d in %d", owner, group, len(w.outposts))
	return Outpost{}
}

// worldPasses steps w until n world passes have run.
func worldPasses(w *World, n int) {
	for n > 0 {
		if w.tick%scriptCycle == worldPassPhase {
			n--
		}
		Step(w, nil)
	}
}

func TestOutpostTypeTenPassBuildsOneOutpostPerComputerGroup(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true, Spread: true, Entries: []OutpostEntry{
		{Group: 7, Percent: 50, Flags: 2}, {Group: 8, Percent: 30, Flags: 0}, {Group: 7, Percent: -4, Flags: 3},
	}})
	if len(w.Outposts()) != 3 {
		t.Fatalf("Outposts = %+v, want owner 2 groups 7 and 8 and owner 3 group 7", w.Outposts())
	}
	// The last matching entry wins; flag 2 keeps the spread, max(13-10, 11-10).
	o := outpostOf(t, w, 2, 7)
	if o.Reload != -4 || o.Countdown != -4 || o.Radius != 3 || o.X != 12 || o.Y != 11 || o.Counting || len(o.Members) != 2 {
		t.Fatalf("owner 2 group 7 = %+v", o)
	}
	if o.Members[0].ID != 2 || o.Members[1].ID != 3 || o.Members[1].Body == nil || o.Members[1].Body.X != 13 {
		t.Fatalf("templates = %+v", o.Members)
	}
	if o := outpostOf(t, w, 2, 8); o.Reload != 30 || o.Radius != 0 {
		t.Fatalf("owner 2 group 8 = %+v, want reload 30 radius 0", o)
	}
	// A group no entry names takes 120 × S / 100; the build without the
	// spread keeps radius 0.
	w = outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true})
	if o := outpostOf(t, w, 3, 7); o.Reload != 120 || o.Countdown != 120 || o.Radius != 0 {
		t.Fatalf("owner 3 group 7 = %+v, want the defaults", o)
	}
}

func TestOutpostShutGateAndFirstGameBuildNone(t *testing.T) {
	w := outpostWorld(t)
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	w.DeclareOutposts(OutpostMap{Spread: true, Entries: []OutpostEntry{{Group: 7, Percent: 5, Flags: 2}}})
	after, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Outposts()) != 0 || !bytes.Equal(before, after) {
		t.Fatalf("shut gate: %d Outposts, form equal %v", len(w.Outposts()), bytes.Equal(before, after))
	}
	first := scriptWorld(t, nil, []Entity{outpostActor(2, 2, 7, 10, 10)})
	first.DeclareOutposts(OutpostMap{Runs: true, Entries: []OutpostEntry{{Group: 7, Percent: 5}}})
	if len(first.Outposts()) != 0 {
		t.Fatalf("first game built %+v", first.Outposts())
	}
}

func TestOutpostGroupComesBackOnPassCountdownPlusOne(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true, Entries: []OutpostEntry{{Group: 7, Percent: 2}}})
	worldPasses(w, 2)
	if o := outpostOf(t, w, 2, 7); o.Counting || o.Countdown != 2 {
		t.Fatalf("with its members standing the Outpost = %+v", o)
	}
	if !w.remove([]EntityID{2, 3}) {
		t.Fatal("remove")
	}
	// Countdown 2: pass 1 starts it, passes 1 and 2 count, pass 3 fires.
	worldPasses(w, 2)
	if o := outpostOf(t, w, 2, 7); !o.Counting || o.Countdown != 0 || indexOfEntity(w.entities, 2) >= 0 {
		t.Fatalf("after 2 passes Outpost = %+v, member 2 at %d", o, indexOfEntity(w.entities, 2))
	}
	worldPasses(w, 1)
	o := outpostOf(t, w, 2, 7)
	if o.Counting || o.Countdown != 2 {
		t.Fatalf("after 3 passes Outpost = %+v", o)
	}
	for _, want := range []Entity{outpostActor(2, 2, 7, 10, 10), outpostActor(3, 2, 7, 13, 11)} {
		i := indexOfEntity(w.entities, want.ID)
		if i < 0 {
			t.Fatalf("member %d did not come back", want.ID)
		}
		e := w.entities[i]
		if e.X != want.X || e.Y != want.Y || e.Owner != 2 || effectiveGroup(&e) != 7 || !w.scriptCountsMember(e) {
			t.Fatalf("member %d = cell %d,%d owner %d group %d", e.ID, e.X, e.Y, e.Owner, effectiveGroup(&e))
		}
		if g := w.savedGroupFor(want.ID); g == nil || g.Selector != 7 || g.Owner.Owner != 2 {
			t.Fatalf("member %d Group = %+v", want.ID, g)
		}
	}
	// The re-formed World keeps its byte form.
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.Outposts(), w.Outposts()) || back.Hash() != w.Hash() {
		t.Fatal("Outposts do not survive the byte form")
	}
}

// A body starts the countdown from decay stage 3 (byte +0x13c of at least 3,
// R2-ENGINE-510); the return takes the body away.
func TestOutpostCountdownStartsAtDecayStageThree(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true, Entries: []OutpostEntry{{Group: 7, Percent: 0}}})
	killGroupMember(t, w, 2)
	killGroupMember(t, w, 3)
	worldPasses(w, 2)
	if o := outpostOf(t, w, 2, 7); o.Counting {
		t.Fatalf("bodies at stage 2 started the countdown: %+v", o)
	}
	for _, id := range []EntityID{2, 3} {
		i := indexOfEntity(w.entities, id)
		w.entities[i].HP, w.entities[i].Decay = decayThirdHP-1, decayDarkStage
	}
	worldPasses(w, 1)
	for _, id := range []EntityID{2, 3} {
		i := indexOfEntity(w.entities, id)
		if i < 0 || w.entities[i].Decay != DecayNone || w.entities[i].HP != 20 {
			t.Fatalf("member %d was not re-created", id)
		}
	}
	if o := outpostOf(t, w, 2, 7); o.Counting || o.Countdown != 0 {
		t.Fatalf("Outpost = %+v, want the reload 0 and no count", o)
	}
}

func TestOutpostWithSpreadPlacesAroundItsCentre(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true, Spread: true, Entries: []OutpostEntry{{Group: 7, Percent: 0, Flags: 2}}})
	if !w.remove([]EntityID{2, 3}) {
		t.Fatal("remove")
	}
	worldPasses(w, 1)
	// Centre 12,11, radius 3: the search draws centre - 1 + 0..3 per axis.
	for _, id := range []EntityID{2, 3} {
		i := indexOfEntity(w.entities, id)
		if i < 0 {
			t.Fatalf("member %d did not come back", id)
		}
		if e := w.entities[i]; e.X < 11 || e.X > 14 || e.Y < 10 || e.Y > 13 {
			t.Fatalf("member %d at %d,%d, outside the search", id, e.X, e.Y)
		}
	}
}

func TestOutpostWaitsWhileTheGroupListsAMember(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true, Entries: []OutpostEntry{{Group: 7, Percent: 0}}})
	if !w.remove([]EntityID{2, 3}) {
		t.Fatal("remove")
	}
	// A unit that joins owner 2's group 7 keeps the Group listed.
	newcomer := outpostActor(9, 2, 7, 16, 16)
	id, err := w.createActor(newcomer, nil, [EquipSlots]ItemInstance{}, 16, 16, nearestFreeCell(16, 16), false,
		actorCreationProfile{ungrouped: true})
	if err != nil {
		t.Fatal(err)
	}
	w.reformOutpostGroup(2, 7, []EntityID{id})
	worldPasses(w, 3)
	if indexOfEntity(w.entities, 2) >= 0 || !outpostOf(t, w, 2, 7).Counting {
		t.Fatal("the Outpost re-created members while its Group listed one")
	}
}

func TestOutpostActionThirtyOneSetsMatchingCountdowns(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true})
	w.runInstant(ScriptInstant{Op: ScriptInstantOutpostCountdown, Group: 7, HasGroup: true, Args: [10]int32{9}})
	if a, b, c := outpostOf(t, w, 2, 7), outpostOf(t, w, 3, 7), outpostOf(t, w, 2, 8); a.Countdown != 9 || b.Countdown != 9 || c.Countdown != 120 {
		t.Fatalf("countdowns %d %d %d, want 9 9 120", a.Countdown, b.Countdown, c.Countdown)
	}
	// p1 is not scaled and the reload stays; a plain p0 names the group too.
	w.runInstant(ScriptInstant{Op: ScriptInstantOutpostCountdown, Args: [10]int32{8, -3}})
	if c := outpostOf(t, w, 2, 8); c.Countdown != -3 || c.Reload != 120 || c.Counting {
		t.Fatalf("group 8 = %+v", c)
	}
	if s := secondGameScript(t, nil, []ScriptInstant{{Op: ScriptInstantOutpostCountdown}}, nil); len(s.Unsupported()) != 0 {
		t.Fatalf("action 31 gaps %+v", s.Unsupported())
	}
}

func TestOutpostLoadedStateHoldsNoMember(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true, Entries: []OutpostEntry{{Group: 7, Percent: 0}}})
	w.AdoptLoadedOutposts()
	for _, o := range w.Outposts() {
		if o.Group != 0 || o.Reload != 120 || o.Countdown != 0 || o.Counting || o.Radius != 0 || len(o.Members) != 0 || o.Owner == 0 {
			t.Fatalf("loaded Outpost = %+v", o)
		}
	}
	if !w.remove([]EntityID{2, 3}) {
		t.Fatal("remove")
	}
	before := w.Outposts()
	worldPasses(w, 3)
	if indexOfEntity(w.entities, 2) >= 0 || !reflect.DeepEqual(before, w.Outposts()) {
		t.Fatal("a loaded Outpost acted")
	}
}

func TestOutpostActionThirtyOneLateGroupActsOnGroupZero(t *testing.T) {
	w := outpostWorld(t)
	w.DeclareOutposts(OutpostMap{Runs: true})
	w.runInstant(ScriptInstant{Op: ScriptInstantOutpostCountdown, LateGroup: true, Args: [10]int32{7}})
	if g7, g8 := outpostOf(t, w, 2, 7), outpostOf(t, w, 2, 8); g7.Countdown != 120 || g8.Countdown != 120 {
		t.Fatalf("countdowns %d %d, a late group reference must not name group 7 or 8", g7.Countdown, g8.Countdown)
	}
}

func newOutpostGroupActor(t *testing.T, w *World, group uint32) EntityID {
	t.Helper()
	id, err := w.createActor(outpostActor(9, 2, group, 16, 16), nil, [EquipSlots]ItemInstance{}, 16, 16, nearestFreeCell(16, 16), false,
		actorCreationProfile{ungrouped: true})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestOutpostNewGroupTakesTheNextIdentityAndOwner(t *testing.T) {
	w := outpostWorld(t)
	id := newOutpostGroupActor(t, w, 9)
	want := max(w.savedGroups.HighWater, maxSavedGroupID(w.savedGroups.Groups)) + 1
	container := w.uniqueSavedPlayerSlot(2)
	n := len(w.savedGroups.Groups)
	w.reformOutpostGroup(2, 9, []EntityID{id})
	g := w.outpostGroup(2, 9)
	if g == nil || len(w.savedGroups.Groups) != n+1 || g.ID != want || w.savedGroups.HighWater != want || g.ContainerID != container {
		t.Fatalf("new Outpost group %+v, want id %d container %d", g, want, container)
	}
	if wantOwner := map[bool]uint32{true: container}[w.hasSavedFormations()]; g.OwnerID != wantOwner {
		t.Fatalf("owner %d under formations %v", g.OwnerID, w.hasSavedFormations())
	}
}

func TestOutpostNewGroupIsRefusedAtTheIdentityCeiling(t *testing.T) {
	w := outpostWorld(t)
	id := newOutpostGroupActor(t, w, 9)
	w.savedGroups.HighWater = ^uint32(0)
	n := len(w.savedGroups.Groups)
	w.reformOutpostGroup(2, 9, []EntityID{id})
	if len(w.savedGroups.Groups) != n || w.savedGroups.HighWater != ^uint32(0) || w.outpostGroup(2, 9) != nil {
		t.Fatal("an Outpost group was added past the identity ceiling")
	}
}
