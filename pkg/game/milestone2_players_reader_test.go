package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// SAV-PLAYER-028/SAV-OBF-029 specify the seventeen-field straight run;
// SAV-662 specifies the literal 32-byte block immediately before the inline
// Diary; SAV-DIARY-042 specifies both independent arrays and D2C. This reader
// consumes original body bytes. Production supplies only structural starts,
// archive indices and reference-tag starts. Direct Player containment is read
// here from raw Group counts/tags, never from decoded Actors/Refs lists.
// Neither Field nor a decoded DTO supplies any expected value or count.
type player1154Raw struct {
	index       uint16
	off, end    int
	name        string
	values      map[string]uint32
	raw10, tail []byte
	diary       diary1154Raw
	groupCount  uint32
}

type diary1154Raw struct {
	location      sav.DocumentDiaryLocation
	dwords, words []byte
	self          uint32
	end           int
}

type players1154Source struct {
	end              int // independently consumed Player root stream
	roots            []uint16
	players          map[uint16]*player1154Raw
	diaries          []diary1154Raw
	members          []sav.DocumentObjectLocation
	memberContainers map[uint16][]int
}

type player1154Reader struct {
	body []byte
	err  error
}

func (r *player1154Reader) take(p *int, n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || *p < 0 || n > len(r.body)-*p {
		r.err = fmt.Errorf("Player byte walk: %d bytes at %d exceed body %d", n, *p, len(r.body))
		return nil
	}
	b := r.body[*p : *p+n]
	*p += n
	return b
}

func (r *player1154Reader) number(p *int, n int) uint32 {
	b := r.take(p, n)
	if r.err != nil {
		return 0
	}
	switch n {
	case 1:
		return uint32(b[0])
	case 2:
		return uint32(binary.LittleEndian.Uint16(b))
	default:
		return binary.LittleEndian.Uint32(b)
	}
}

// CArchive counts use a u16, extended by u32 after 0xffff. Bound bytes before
// allocating; the two Diary counts are read separately, never made equal.
func (r *player1154Reader) array(p *int, width int) []byte {
	n := r.number(p, 2)
	if n == 0xffff {
		n = r.number(p, 4)
	}
	if r.err != nil {
		return nil
	}
	if n > 1<<20 || uint64(n)*uint64(width) > uint64(len(r.body)-*p) {
		r.err = fmt.Errorf("Diary byte walk: count %d width %d at %d exceeds bounds", n, width, *p)
		return nil
	}
	return slices.Clone(r.take(p, int(n)*width))
}

func (r *player1154Reader) diary(loc sav.DocumentDiaryLocation) diary1154Raw {
	d := diary1154Raw{location: loc}
	if loc.Off < 0 {
		return d
	}
	p := loc.Off
	d.dwords, d.words = r.array(&p, 4), r.array(&p, 2)
	d.self = r.number(&p, 4)
	d.end = p
	return d
}

func players1154Expected(f *sav.File) (players1154Source, error) {
	s := players1154Source{players: map[uint16]*player1154Raw{}, memberContainers: map[uint16][]int{}}
	locations, err := f.DocumentDiaryLocations()
	if err != nil {
		return s, err
	}
	objects, err := f.DocumentObjectLocations()
	if err != nil {
		return s, err
	}
	r := player1154Reader{body: f.Body}
	objectsByOff := map[int]sav.DocumentObjectLocation{}
	objectsByIndex := map[uint16]sav.DocumentObjectLocation{}
	for _, loc := range objects {
		objectsByOff[loc.Off], objectsByIndex[loc.ArchiveIndex] = loc, loc
		if loc.Class == "Unit" || loc.Class == "Human" || loc.Class == "Humanoid" {
			s.members = append(s.members, loc)
		}
	}
	byOff := map[int]uint16{}
	for _, loc := range locations {
		if loc.OwnerClass != "Player" {
			index := r.referenceAt(loc.RefOff, objectsByOff, objectsByIndex)
			if index != loc.ArchiveIndex || (index == 0) != (loc.Off < 0) || index != 0 && objectsByIndex[index].Off != loc.Off {
				return s, fmt.Errorf("Humanoid archive %d Diary raw tag disagrees with structural location", loc.OwnerArchiveIndex)
			}
		}
		d := r.diary(loc)
		s.diaries = append(s.diaries, d)
		if loc.OwnerClass != "Player" {
			continue
		}
		if loc.Off < 32 {
			return s, fmt.Errorf("Player archive %d lacks inline Diary start", loc.OwnerArchiveIndex)
		}
		p := loc.OwnerOff
		n := r.number(&p, 1)
		if n == 0xff {
			return s, fmt.Errorf("Player at %d has unsupported CString", p-1)
		}
		row := &player1154Raw{index: loc.OwnerArchiveIndex, off: loc.OwnerOff, end: d.end, name: string(r.take(&p, int(n))), values: map[string]uint32{}, diary: d}
		v := func(name string, width int) { row.values[name] = r.number(&p, width) }
		v("Slot", 2)
		v("SlotAgain", 4)
		row.raw10 = slices.Clone(r.take(&p, 8))
		v("F44", 1)
		v("Participant", 4)
		v("F2C", 2)
		v("Money", 4)
		row.values["Money"] ^= 0x5c073f4d
		v("Outcome", 1)
		v("F3D", 1)
		v("F48", 4)
		row.values["F48"] ^= 0x5c073f4d
		v("F50", 4)
		v("F54", 2)
		v("F4C", 2)
		v("F58", 4)
		v("Hero", 4)
		v("This", 4)
		row.groupCount = r.number(&p, 4)
		p = loc.Off - 32 // SAV-662: own 32-byte structure immediately before Diary.
		row.tail = slices.Clone(r.take(&p, 32))
		s.players[row.index], byOff[row.off] = row, row.index
	}
	groups, err := f.DocumentPlayerGroupLocations()
	if err != nil {
		return s, err
	}
	if err := s.readGroups(&r, groups, byOff, objectsByOff, objectsByIndex); err != nil {
		return s, err
	}
	// SAV-HEAD-025/SAV-STREAM-013: Head.End locates the root stream. Its
	// preceding dword is the raw reference-slot count, including null/repeats.
	p := f.Head.End - 4
	n := r.number(&p, 4)
	if r.err != nil {
		return s, r.err
	}
	if n > 65536 || uint64(n)*2 > uint64(len(f.Body)-p) {
		return s, fmt.Errorf("Player root count %d exceeds bounds", n)
	}
	for range n {
		tag := uint16(r.number(&p, 2))
		index := tag
		if tag >= 0x8000 {
			if tag == 0xffff {
				schema, count := r.number(&p, 2), r.number(&p, 2)
				if schema != 1 || count != 6 || string(r.take(&p, int(count))) != "Player" {
					return s, fmt.Errorf("Player root has wrong class at %d", p)
				}
			}
			index = byOff[p]
			row := s.players[index]
			if row == nil {
				return s, fmt.Errorf("Player root lacks structural start at %d", p)
			}
			p = row.end
		} else if tag != 0 {
			if row := s.players[tag]; row == nil || row.off >= p {
				return s, fmt.Errorf("Player root has unknown/forward archive %d", tag)
			}
		}
		s.roots = append(s.roots, index)
	}
	s.end = p
	return s, r.err
}

func (r *player1154Reader) referenceAt(off int, byOff map[int]sav.DocumentObjectLocation, byIndex map[uint16]sav.DocumentObjectLocation) uint16 {
	p := off
	tag := uint16(r.number(&p, 2))
	if r.err != nil || tag == 0 {
		return 0
	}
	if tag < 0x8000 {
		if loc, ok := byIndex[tag]; !ok || loc.Off >= off {
			r.err = fmt.Errorf("unknown/forward archive reference %d at %d", tag, off)
		}
		return tag
	}
	name := ""
	if tag == 0xffff {
		schema, n := r.number(&p, 2), r.number(&p, 2)
		if schema != 1 || n == 0 || n > 32 {
			r.err = fmt.Errorf("unsupported class header at %d", off)
			return 0
		}
		name = string(r.take(&p, int(n)))
	}
	loc, ok := byOff[p]
	if !ok || name != "" && loc.Class != name {
		r.err = fmt.Errorf("no object structure at %d (%q)", p, name)
		return 0
	}
	return loc.ArchiveIndex
}

// SAV-GRPSAVENEXT-572 supplies Group's two word lists,80-byte AI block and
// u32 member count. The locator supplies only starts so a missing decoded
// actor/reference cannot silently shrink the expected Diary/F58 live subset.
func (s *players1154Source) readGroups(r *player1154Reader, groups []sav.DocumentPlayerGroupLocation, playersByOff map[int]uint16, byOff map[int]sav.DocumentObjectLocation, byIndex map[uint16]sav.DocumentObjectLocation) error {
	counts := map[int]uint32{}
	for _, group := range groups {
		player := s.players[playersByOff[group.PlayerOff]]
		if player == nil || group.Off <= player.off || group.Off >= player.diary.location.Off-32 {
			return fmt.Errorf("invalid Group start %d in Player start %d", group.Off, group.PlayerOff)
		}
		counts[group.PlayerOff]++
		p := group.Off
		r.array(&p, 2)
		r.take(&p, 80)
		r.array(&p, 2)
		n := r.number(&p, 4)
		if r.err != nil {
			return r.err
		}
		if uint64(n) != uint64(len(group.ActorRefOffs)) {
			return fmt.Errorf("Group at %d raw actor count %d != %d tag locations", group.Off, n, len(group.ActorRefOffs))
		}
		for i, off := range group.ActorRefOffs {
			if i == 0 && off != p || off < p || i > 0 && off <= group.ActorRefOffs[i-1] {
				return fmt.Errorf("invalid Group actor tag start %d", off)
			}
			index := r.referenceAt(off, byOff, byIndex)
			if index == 0 {
				continue
			}
			class := byIndex[index].Class
			if class != "Unit" && class != "Human" && class != "Humanoid" {
				return fmt.Errorf("Group actor tag %d names %s", off, class)
			}
			if !slices.Contains(s.memberContainers[index], group.PlayerOff) {
				s.memberContainers[index] = append(s.memberContainers[index], group.PlayerOff)
			}
		}
	}
	for _, player := range s.players {
		if player.groupCount != counts[player.off] {
			return fmt.Errorf("Player archive %d raw Group count %d != %d structural starts", player.index, player.groupCount, counts[player.off])
		}
	}
	return r.err
}

func (d diary1154Raw) entries() []sim.SavedDiaryEntry {
	var out []sim.SavedDiaryEntry
	for i := 0; i < len(d.dwords)/4 && i < len(d.words)/2; i++ {
		count, remaining := binary.LittleEndian.Uint32(d.dwords[4*i:]), binary.LittleEndian.Uint16(d.words[2*i:])
		if count != 0 || remaining != 1024 {
			out = append(out, sim.SavedDiaryEntry{Index: i, Count: count, Remaining: remaining})
		}
	}
	return out
}

// Only the decoder's transient index correspondence is used. Its values are
// discarded. The comparison below separately rejects many-to-one mappings.
func players1154Origins(raw []byte) (map[uint16]uint16, error) {
	_, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		return nil, err
	}
	join := map[uint16]uint16{}
	for _, row := range origins {
		if _, exists := join[row.ArchiveIndex]; exists {
			return nil, fmt.Errorf("duplicate source origin %d", row.ArchiveIndex)
		}
		join[row.ArchiveIndex] = row.ObjectIndex
	}
	return join, nil
}
