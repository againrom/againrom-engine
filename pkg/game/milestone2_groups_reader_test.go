package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type group1155Raw struct {
	player           uint16
	inline           int
	off, end         int
	selector         uint32
	ai               []byte
	words, path      []uint16
	members          []uint16 // Raw slots, including repeats and nulls, before LOAD.
	current          []uint16 // Independently normalized LOAD membership.
	reference, owner sim.SavedGroupReference
}

type group1155ActorRaw struct {
	class    string
	identity uint32
	order    []byte
	patrol   []uint16
	state    uint32
	player   uint16 // Final membership's enclosing Player, not Group+44.
}

type groups1155Source struct {
	players players1154Source
	groups  []group1155Raw
	actors  map[uint16]group1155ActorRaw
}

func group1155Words(b []byte) []uint16 {
	var out []uint16
	for p := 0; p < len(b); p += 2 {
		out = append(out, binary.LittleEndian.Uint16(b[p:]))
	}
	return out
}

func group1155Bytes(words []uint16) []byte {
	var out []byte
	for _, word := range words {
		out = binary.LittleEndian.AppendUint16(out, word)
	}
	return out
}

// Only decompression and structural starts come from production. The Player
// reader supplies raw root identities/counts/Slots/This, never decoded fields.
// SAV-GRPSAVENEXT-572 and SAV-WLIST-040 define both counted u16 lists,80 AI
// bytes, the u32 member count and trailing G1C/G40/G44. The following Group or
// Player raw-tail START locates the suffix; no decoded Group end or member
// list supplies a count, value or expected traversal order.
func groups1155Expected(f *sav.File) (groups1155Source, error) {
	s := groups1155Source{actors: map[uint16]group1155ActorRaw{}}
	var err error
	s.players, err = players1154Expected(f)
	if err != nil {
		return s, err
	}
	locations, err := f.DocumentPlayerGroupLocations()
	if err != nil {
		return s, err
	}
	objects, err := f.DocumentObjectLocations()
	if err != nil {
		return s, err
	}
	orders, err := f.DocumentActorLocations()
	if err != nil {
		return s, err
	}
	r := player1154Reader{body: f.Body}
	byOff, byIndex := map[int]sav.DocumentObjectLocation{}, map[uint16]sav.DocumentObjectLocation{}
	keys := map[uint16]uint32{}
	for _, loc := range objects {
		byOff[loc.Off], byIndex[loc.ArchiveIndex] = loc, loc
		p := loc.Off
		switch loc.Class {
		case "Player":
			keys[loc.ArchiveIndex] = s.players.players[loc.ArchiveIndex].values["This"]
			continue
		case "Spell":
			p += 5 // SAV-SPELL-044: this follows u8,u8,u8,u16.
		case "Token", "Unit", "Human", "Humanoid", "Building", "Outpost", "Tavern", "Shop", "Sack", "Item", "Weapon", "Shield", "Armor", "Effect", "Effect_DirectDamage", "VirtualCaster", "SpellEffect", "PointEffect", "AreaEffect", "SpellTransport":
			p += 29 // SAV-TOKEN-034: Token's eighth field is its own identity.
		case "Diary", "Spellbook":
			continue // Neither class registers its own key.
		default:
			return s, fmt.Errorf("Group identity domain lacks class %s at %d", loc.Class, loc.Off)
		}
		keys[loc.ArchiveIndex] = r.number(&p, 4)
	}
	resolve := func(key uint32, end int) (sim.SavedGroupReference, error) {
		ref := sim.SavedGroupReference{Key: key}
		if key == 0 {
			return ref, nil
		}
		for _, loc := range objects {
			if keys[loc.ArchiveIndex] != key || loc.Off >= end {
				continue
			}
			if ref.Class != 0 {
				return ref, fmt.Errorf("raw Group reference %#x has several earlier identities", key)
			}
			ref.Archive, ref.Class = loc.ArchiveIndex, 2
			if loc.Class == "Player" {
				ref.Class, ref.Owner = 1, s.players.players[loc.ArchiveIndex].values["Slot"]
			}
		}
		return ref, nil // SAV-GRPLOAD-560: a missing key is null, without fallback.
	}
	seen := map[uint16]bool{}
	for _, root := range s.players.roots {
		if root == 0 || seen[root] {
			continue
		}
		seen[root] = true
		player := s.players.players[root]
		var own []sav.DocumentPlayerGroupLocation
		for _, loc := range locations {
			if loc.PlayerOff == player.off {
				own = append(own, loc)
			}
		}
		if uint64(len(own)) != uint64(player.groupCount) {
			return s, fmt.Errorf("raw Player %d Group count differs from starts", root)
		}
		for i, loc := range own {
			g := group1155Raw{player: root, inline: i, off: loc.Off, end: player.diary.location.Off - 32}
			if i+1 < len(own) {
				g.end = own[i+1].Off
			}
			p := g.off
			g.words = group1155Words(r.array(&p, 2))
			g.ai = slices.Clone(r.take(&p, 80))
			g.path = group1155Words(r.array(&p, 2))
			n := r.number(&p, 4)
			if r.err != nil {
				return s, r.err
			}
			if uint64(n) != uint64(len(loc.ActorRefOffs)) || p > g.end-12 {
				return s, fmt.Errorf("Group at %d raw member count/layout disagrees with tag starts", g.off)
			}
			for j, off := range loc.ActorRefOffs {
				next := g.end - 12
				if j+1 < len(loc.ActorRefOffs) {
					next = loc.ActorRefOffs[j+1]
				}
				if off != p || next < off+2 {
					return s, fmt.Errorf("Group at %d member tag %d has inconsistent extent", g.off, j)
				}
				index := r.referenceAt(off, byOff, byIndex)
				tag := r.number(&p, 2)
				if tag < 0x8000 && p != next {
					return s, fmt.Errorf("Group at %d null/back-reference has non-tag payload", g.off)
				}
				p = next
				g.members = append(g.members, index)
				if index != 0 {
					loc := byIndex[index]
					if loc.Class != "Unit" && loc.Class != "Human" && loc.Class != "Humanoid" {
						return s, fmt.Errorf("Group raw member %d is %s", index, loc.Class)
					}
					s.actors[index] = group1155ActorRaw{class: loc.Class, identity: keys[index]}
				}
			}
			if p != g.end-12 {
				return s, fmt.Errorf("Group at %d members do not reach the three-dword suffix", g.off)
			}
			g.selector = r.number(&p, 4)
			if g.reference, err = resolve(r.number(&p, 4), g.end); err != nil {
				return s, err
			}
			if g.owner, err = resolve(r.number(&p, 4), g.end); err != nil {
				return s, err
			}
			s.groups = append(s.groups, g)
		}
	}
	// SAV-GRPLOAD-560 detach-before-append has an independent closed-form
	// oracle: keep only each non-null actor's LAST raw occurrence globally.
	// Null slots remain distinct. SAV-GRPOWNER-561's Player suffix changes the
	// actor owner separately from the Group's restored independent owner.
	type occurrence struct{ group, slot int }
	last := map[uint16]occurrence{}
	for i, g := range s.groups {
		for j, index := range g.members {
			if index != 0 {
				last[index] = occurrence{i, j}
			}
		}
	}
	for i := range s.groups {
		g := &s.groups[i]
		for j, index := range g.members {
			if index == 0 || last[index] == (occurrence{i, j}) {
				g.current = append(g.current, index)
				if index != 0 {
					a := s.actors[index]
					a.player = g.player
					s.actors[index] = a
				}
			}
		}
	}
	for _, loc := range orders {
		a, relevant := s.actors[loc.ArchiveIndex]
		if !relevant {
			continue
		}
		// SAV-UNITPROG-156: six raw blocks 24,22,24,64,180,148;
		// SAV-PATROLCURSOR-571: the148-byte block precedes its counted ring.
		p := loc.RawBlocksOff + 24 + 22 + 24 + 64 + 180
		a.order = slices.Clone(r.take(&p, 148))
		a.patrol = group1155Words(r.array(&p, 2))
		if r.err != nil || p != loc.ControlOff {
			return s, fmt.Errorf("raw actor %d order/ring does not reach control run: %v", loc.ArchiveIndex, r.err)
		}
		p += 4 // Four u8 scalars precede U50 in the control run.
		a.state = r.number(&p, 4)
		s.actors[loc.ArchiveIndex] = a
	}
	for index, a := range s.actors {
		if len(a.order) != 148 {
			return s, fmt.Errorf("raw Group actor %d lacks an order structure", index)
		}
	}
	return s, r.err
}
