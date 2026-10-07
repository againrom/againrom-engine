package game

import (
	"os"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// partyOwnerFirst and partyOwnerLast bound the mission-141 group whose owner
// the script's trigger 3 hands to the player once a player unit comes within
// three cells of (15,59); partyOwnerLast is the one the party kills.
const partyOwnerFirst, partyOwnerLast = 8, 12

type partyOwnerMember struct {
	ID     sim.EntityID
	X, Y   int32
	HP     int32
	Owner  uint32
	Worn   []uint16
	Pack   []uint16
	Unit   uint16
	Living bool
}

func partyOwnerMembers(t *testing.T, f *FrontEnd) map[sim.EntityID]partyOwnerMember {
	t.Helper()
	out := map[sim.EntityID]partyOwnerMember{}
	for _, e := range f.live.world.Entities() {
		if e.Owner != sim.SelfSlot {
			continue
		}
		m := partyOwnerMember{ID: e.ID, X: e.X, Y: e.Y, HP: e.HP, Owner: e.Owner, Unit: e.MapUnitID, Living: e.Alive()}
		worn, _ := f.live.world.EquippedItems(e.ID)
		for _, item := range worn {
			m.Worn = append(m.Worn, item.Code)
		}
		pack, _ := f.live.world.CarriedItems(e.ID)
		for _, item := range pack {
			m.Pack = append(m.Pack, item.Code)
		}
		out[e.ID] = m
	}
	return out
}

// partyOwnerReferences requires every placed actor record's owner Reference
// to be the identity key of the Player whose slot is that actor's current
// owner, a Group member's to be its enclosing Player's key, and the party
// walk of the file to read without refusal.
func partyOwnerReferences(t *testing.T, f *FrontEnd, store SaveStore, name string, doc sav.DocumentData) {
	t.Helper()
	bySlot := map[uint32]uint32{}
	for _, index := range doc.Players {
		player := &doc.Objects[index-1]
		key, err := savedStructureValue(player, "This")
		if err != nil {
			t.Fatal(err)
		}
		slot, err := savedStructureValue(player, "Slot")
		if err != nil {
			t.Fatal(err)
		}
		bySlot[uint32(uint16(slot))] = key
		for _, group := range player.Groups {
			refs, _ := savedObjectRefs(&group, "Actors")
			for _, member := range refs {
				if ref, _ := savedStructureValue(&doc.Objects[member-1], "Reference"); ref != key {
					t.Errorf("Group member record %d names owner %#x inside the Player keyed %#x", member, ref, key)
				}
			}
		}
	}
	owners := map[uint16]uint32{}
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID != 0 {
			owners[e.MapUnitID] = e.Owner
		}
	}
	checked := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
			continue
		}
		unit, err := savedStructureValue(r, "T08")
		if err != nil {
			t.Fatal(err)
		}
		owner, placed := owners[uint16(unit)]
		if !placed {
			continue
		}
		checked++
		if ref, _ := savedStructureValue(r, "Reference"); ref != bySlot[owner] {
			t.Errorf("map unit %d names owner %#x, want slot %d's Player %#x", uint16(unit), ref, owner, bySlot[owner])
		}
	}
	if checked == 0 {
		t.Fatal("no placed actor record was checked")
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := file.PartyWalk(); err != nil {
		t.Fatalf("party walk of %s: %v", name, err)
	}
}

// Mission 141 on a world that runs on a loaded engine SAV: the party walks to
// the strangers at (15,59), the script hands their group to the player, and
// the party kills one of them through its own attack order. SAVE writes every
// actor's owner Reference as a Player key of the same file, and a cold LOAD
// from the main menu restores the party: every member at its cell with its
// health, worn and carried items, the killed stranger as a body the player
// owns. The fresh session without the first LOAD is the second case.
func TestReleaseTransferredDeadAllySAVReloadsTheParty(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the mission-141 hand-over needs a lawful install")
	}
	for _, loaded := range []bool{true, false} {
		name := "fresh session"
		if loaded {
			name = "loaded session"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			if err := f.App("party owner load").OpenMission(f.MissionOpener(141)); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			if loaded {
				f.live.tick()
				opened, _ := deadPatrolSave(t, f, store)
				f = loadLocalLegacySave(t, store, opened)
			}
			for _, id := range f.live.mission.ids {
				f.live.enqueue(uint32(id), 15, 57)
			}
			victim := deadPatrolEntity(t, f, partyOwnerLast)
			for i := 0; victim.Owner != sim.SelfSlot; i++ {
				if i == 3000 {
					t.Fatalf("map unit %d still owned by %d after %d ticks", partyOwnerLast, victim.Owner, i)
				}
				f.live.tick()
				victim = deadPatrolEntity(t, f, partyOwnerLast)
			}
			for id := uint16(partyOwnerFirst); id <= partyOwnerLast; id++ {
				if e := deadPatrolEntity(t, f, id); e.Owner != sim.SelfSlot {
					t.Fatalf("map unit %d owner %d after the hand-over", id, e.Owner)
				}
			}
			for _, id := range f.live.mission.ids {
				f.live.strike(uint32(id), uint32(victim.ID))
			}
			for i := 0; victim.Alive() || victim.Decay < sim.DecayBones; i++ {
				if i == 4000 {
					t.Fatalf("map unit %d HP %d stage %d after %d ticks of the party's attack", partyOwnerLast, victim.HP, victim.Decay, i)
				}
				f.live.tick()
				victim = deadPatrolEntity(t, f, partyOwnerLast)
			}
			want := partyOwnerMembers(t, f)
			saved, doc := deadPatrolSave(t, f, store)
			partyOwnerReferences(t, f, store, saved, doc)
			bodyObject := uint16(0)
			for i := range doc.Objects {
				r := &doc.Objects[i]
				if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
					continue
				}
				if unit, err := savedStructureValue(r, "T08"); err == nil && unit == partyOwnerLast {
					if bodyObject != 0 {
						t.Fatal("repeated torn-down actor records")
					}
					bodyObject = uint16(i + 1)
				}
			}
			if bodyObject == 0 || !slices.Contains(doc.DeadActors, bodyObject) {
				t.Fatal("torn-down actor has no DeadActors root", bodyObject)
			}
			for _, player := range doc.Players {
				for _, group := range doc.Objects[player-1].Groups {
					members, _ := savedObjectRefs(&group, "Actors")
					if slices.Contains(members, bodyObject) {
						t.Fatal("torn-down actor remains in a saved Group")
					}
				}
			}

			g := loadLocalLegacySave(t, store, saved)
			got := partyOwnerMembers(t, g)
			if len(got) != len(want) {
				t.Fatalf("LOAD restored %d player-owned actors, want %d", len(got), len(want))
			}
			for id, w := range want {
				m, ok := got[id]
				if !ok || m.X != w.X || m.Y != w.Y || m.HP != w.HP || m.Living != w.Living || m.Unit != w.Unit ||
					!slices.Equal(m.Worn, w.Worn) || !slices.Equal(m.Pack, w.Pack) {
					t.Fatalf("LOAD restored actor %d as %+v (present %t), want %+v", id, m, ok, w)
				}
			}
			if body := got[victim.ID]; body.Living || body.Unit != partyOwnerLast {
				t.Fatalf("map unit %d restored as %+v, want a body the player owns", partyOwnerLast, body)
			}
			for range 64 {
				g.live.tick()
			}
		})
	}
}
