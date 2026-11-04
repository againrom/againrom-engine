package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// secondGuardMission is a mission 10 holding two knights of Player 2 in
// Group 5, at cells 48/48 and 50/48: a computer Group the engine runs as its
// guard, which the second game holds at programme byte 18 (DIV-4285).
func secondGuardMission() secondMissionContent {
	p := humansParams(40, 0, 52)
	p[16], p[18], p[24] = 9, 0, 7301
	knight := func(x uint32, id uint16) alm.Unit {
		return alm.Unit{X: x<<8 | 0x80, Y: 48<<8 | 0x80, ClassID: 9, ClassSubID: 1, Flags: 0x10, ServerID: 7301, Owner: 2, UnitID: id, GroupID: 5}
	}
	return secondMissionContent{side: 128, humans: map[int]dbEntry{256: {name: "Knight", params: p}},
		units: []alm.Unit{knight(48, 7), knight(50, 8)}}
}

// secondGuardKnightPost is the guard setter's group post of the two
// knights: the cell of their fine centroid, 49/48.
const secondGuardKnightPost = 49 | 48<<8

func secondRecordRaw(r *sav.DocumentRecordData, name string) []byte {
	for _, v := range r.Raw {
		if v.Name == name {
			return v.Bytes
		}
	}
	return nil
}

// secondGuardGroup is the programme block of Player 2's Group 5 in doc.
func secondGuardGroup(t *testing.T, doc sav.DocumentData) []byte {
	t.Helper()
	for _, p := range doc.Players {
		r := &doc.Objects[p-1]
		if slot, _ := savedStructureValue(r, "Slot"); slot != 2 {
			continue
		}
		for i := range r.Groups {
			if sel, _ := savedStructureValue(&r.Groups[i], "G1C"); sel == 5 {
				return secondRecordRaw(&r.Groups[i], "G3C")
			}
		}
	}
	t.Fatal("no Group 5 of Player 2")
	return nil
}

// secondMemberPosts is the command posts of each Player's Group members'
// records, sorted, by the Player's slot, and the knights' by map unit ID.
// Every command block byte past words +0 and +0x16 is the zero it was
// before the post had a source.
func secondMemberPosts(t *testing.T, doc sav.DocumentData) (map[uint32][]uint16, map[uint16]uint16) {
	t.Helper()
	bySlot, knights := map[uint32][]uint16{}, map[uint16]uint16{}
	for _, p := range doc.Players {
		r := &doc.Objects[p-1]
		slot, _ := savedStructureValue(r, "Slot")
		for i := range r.Groups {
			for _, refs := range r.Groups[i].RefSlots {
				if refs.Name != "Actors" {
					continue
				}
				for _, m := range refs.Objects {
					a := &doc.Objects[m-1]
					block := secondRecordRaw(a, "URaw184")
					if len(block) != 184 {
						t.Fatalf("%s record %d holds a %d-byte command block", a.Class, m, len(block))
					}
					for at, b := range block {
						if at >= 2 && (at < 0x16 || at > 0x17) && b != 0 {
							t.Fatalf("%s record %d command block byte %#x is %#x; only words +0 and +0x16 have a source", a.Class, m, at, b)
						}
					}
					post := binary.LittleEndian.Uint16(block)
					bySlot[slot] = append(bySlot[slot], post)
					if unit, _ := savedStructureValue(a, "T08"); slot == 2 {
						knights[uint16(unit)] = post
					}
				}
			}
		}
		slices.Sort(bySlot[slot])
	}
	return bySlot, knights
}

// secondEntityPost is e's command post as a cell word.
func secondEntityPost(e sim.Entity) uint16 { return uint16(e.PostX) | uint16(e.PostY)<<8 }

// secondWorldPosts is the posts of the World's living owned actors, sorted,
// by owner.
func secondWorldPosts(w *sim.World) map[uint32][]uint16 {
	out := map[uint32][]uint16{}
	for _, e := range w.Entities() {
		if e.Alive() && e.Owner != 0 {
			out[e.Owner] = append(out[e.Owner], secondEntityPost(e))
		}
	}
	for owner := range out {
		slices.Sort(out[owner])
	}
	return out
}

// secondGuardKnights is the World's two knights' posts, by map unit ID.
func secondGuardKnights(t *testing.T, w *sim.World) map[uint16]uint16 {
	t.Helper()
	out := map[uint16]uint16{}
	for _, e := range w.Entities() {
		if e.MapUnitID == 7 || e.MapUnitID == 8 {
			out[uint16(e.MapUnitID)] = secondEntityPost(e)
		}
	}
	if len(out) != 2 {
		t.Fatalf("the World holds %d knights, want 2", len(out))
	}
	return out
}

// A second-game mission SAVE writes each Group's programme from the Group's
// current state and each member's command post from its order: the knights'
// guard Group holds programme byte 18 (DIV-4285) and the guard setter's post,
// and each member, the party among them, the post the World holds for it
// (R2-ENGINE-507, R2-ENGINE-824). The engine's LOAD of the file and the
// LOAD of its records alone, as of the original's file, restore both. Every
// command block byte past words +0 and +0x16 stays zero and the Group's
// replaced list pointer stays zero. Fails while the SAVE writes programme
// byte 1, group post 0 and command post 0 (R2-SESSION-337).
func TestSecondMissionGroupWritesItsProgrammeAndPosts(t *testing.T) {
	c := secondGuardMission()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	raw := secondPageSave(t, f, app)
	doc := checkSecondMissionFile(t, raw, 10)
	ai := secondGuardGroup(t, doc)
	if ai[0x20] != 18 || binary.LittleEndian.Uint16(ai) != secondGuardKnightPost {
		t.Fatalf("the knights' Group holds programme byte %d and post %#04x, want 18 and %#04x", ai[0x20], binary.LittleEndian.Uint16(ai), secondGuardKnightPost)
	}
	if binary.LittleEndian.Uint32(ai[0x4c:]) != 0 {
		t.Fatalf("the Group's list pointer is %#x, want the zero LOAD replaces", binary.LittleEndian.Uint32(ai[0x4c:]))
	}
	posts, knightPosts := secondMemberPosts(t, doc)
	world := secondWorldPosts(f.live.world)
	if len(world[sim.SelfSlot]) == 0 || len(world[2]) != 2 {
		t.Fatalf("the World holds %d party actors and %d knights", len(world[sim.SelfSlot]), len(world[2]))
	}
	for owner, want := range world {
		if !slices.Equal(posts[owner], want) || slices.Contains(want, 0) {
			t.Fatalf("Player %d's members write command posts %#04x, want their posts %#04x", owner, posts[owner], want)
		}
	}
	knights := secondGuardKnights(t, f.live.world)
	for id, post := range knights {
		if knightPosts[id] != post {
			t.Fatalf("knight %d writes command post %#04x, want %#04x", id, knightPosts[id], post)
		}
	}

	cold := secondMissionLoadBytesOf(t, c, raw)
	if cold.live == nil {
		t.Fatal("the file does not LOAD")
	}
	if got := secondGuardKnights(t, cold.live.world); got[7] != knights[7] || got[8] != knights[8] {
		t.Fatalf("LOAD: knight posts %#04x, want %#04x", got, knights)
	}

	// The file's records alone, one knight's post planted away from its
	// cell: LOAD reads the Group's programme and post and the planted post.
	const planted = 43 | 44<<8
	bareDoc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	bareDoc.State.ValueRecords = slices.DeleteFunc(slices.Clone(bareDoc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
		return strings.EqualFold(r.Path, sav.NativeFormPath)
	})
	plantedAt := false
	for i := range bareDoc.Objects {
		r := &bareDoc.Objects[i]
		if v, _ := savedStructureValue(r, "T08"); r.Class == "Human" && uint16(v) == 7 {
			binary.LittleEndian.PutUint16(secondRecordRaw(r, "URaw184"), planted)
			plantedAt = true
		}
	}
	if !plantedAt {
		t.Fatal("no record of knight 7")
	}
	bare, err := sav.EncodeDocumentData(bareDoc)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), bare, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, _ := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if loaded.live == nil {
		t.Fatal("the file's records alone do not LOAD")
	}
	groups, _, present := loaded.live.world.SavedGroups()
	at := slices.IndexFunc(groups, func(g sim.SavedGroup) bool { return g.Selector == 5 && g.Owner.Owner == 2 })
	if !present || at < 0 {
		t.Fatal("the records' LOAD holds no Group 5 of Player 2")
	}
	if g := groups[at]; g.AI[0x20] != 18 || binary.LittleEndian.Uint16(g.AI[:]) != secondGuardKnightPost {
		t.Fatalf("the records' LOAD holds programme byte %d and post %#04x, want 18 and %#04x", g.AI[0x20], binary.LittleEndian.Uint16(g.AI[:]), secondGuardKnightPost)
	}
	if got := secondGuardKnights(t, loaded.live.world); got[7] != planted || got[8] != knights[8] {
		t.Fatalf("the records' LOAD: knight posts %#04x, want 7 at %#04x and 8 at %#04x", got, planted, knights[8])
	}
}
