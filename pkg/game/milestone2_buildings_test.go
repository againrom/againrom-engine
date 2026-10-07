package game

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sort"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Only Off, Class and ArchiveIndex come from the archive enumerator. Every
// compared field is read from the decompressed bytes: SAV-TOKEN-034's 37-byte
// head, SAV-BLDG-037's 77-byte base, SAV-CLASSSER-173/176 and SHOP-SAVE-015's
// subclass tails. No decoded Building member is an expected value. A raw root
// tag walk checks the enumeration against the entire counted archive list.
type building1145Want struct {
	live       sim.Structure
	source     sim.SavedStructure
	normalized bool
	next       int
}

func building1145Raw(body []byte, off int, class string, archive uint16) (building1145Want, error) {
	var want building1145Want
	if off < 0 || off > len(body)-77 {
		return want, fmt.Errorf("Building at %d has no complete 77-byte base", off)
	}
	b := body[off:]
	want.next = off + 77
	u16 := func(p int) uint16 { return binary.LittleEndian.Uint16(b[p:]) }
	u32 := func(p int) uint32 { return binary.LittleEndian.Uint32(b[p:]) }
	want.live = sim.Structure{Kind: uint16(b[59]), Field42: u16(60), MaxHealth: u16(62), Col: int32(b[0]), Row: int32(b[1]), Width: b[67], Height: b[68], Attach: u32(73), Blocking: u32(69)}
	s := &want.source
	s.SourceKey, s.ArchiveIndex, s.AuthoredID = u32(29), archive, u32(19)
	copy(s.Position[:], b[:12])
	s.RuntimeID, s.Token0C, s.Token0E, s.Token18, s.Token1C, s.Reference = u32(12), b[16], u16(17), u16(23), u32(25), u32(33)
	copy(s.Base52[:], b[37:59])
	s.Kind, s.Field46, s.Field48, s.Blocking = b[59], u16(64), b[66], u32(69)
	// The later scalar LOAD writes overlap the earlier raw +52 image.
	// Preserve the other 16 raw bytes; the final width/height/blocking win.
	rawBase := s.Base52
	s.Base52[14], s.Base52[15] = b[67], b[68]
	copy(s.Base52[18:22], b[69:73])
	want.normalized = rawBase != s.Base52
	switch class {
	case "Building":
		s.Class = sim.SavedBuilding
	case "Tavern", "Shop":
		if len(b) < 81 {
			return want, fmt.Errorf("%s lacks its trailing dword", class)
		}
		want.next = off + 81
		if class == "Tavern" {
			s.Class, s.Tavern9C = sim.SavedTavern, u32(77)
		} else {
			s.Class, s.Shop70 = sim.SavedShop, u32(77)
		}
	case "Outpost":
		if len(b) < 95 {
			return want, fmt.Errorf("Outpost lacks its fixed tail")
		}
		s.Class = sim.SavedOutpost
		s.OutpostWords = [4]uint32{u32(77), u32(81), u32(85), u32(89)}
		n, p := uint32(u16(93)), 95
		if n == 0xffff {
			if len(b) < 99 {
				return want, fmt.Errorf("Outpost lacks its wide count")
			}
			n, p = u32(95), 99
		}
		if uint64(n)*8 > uint64(len(b)-p) {
			return want, fmt.Errorf("Outpost's %d records exceed its bytes", n)
		}
		for i := uint32(0); i < n; i++ {
			var record [8]byte
			copy(record[:], b[p:p+8])
			s.OutpostRecords = append(s.OutpostRecords, record)
			p += 8
		}
		want.next = off + p
	default:
		return want, fmt.Errorf("unsupported Building class %q", class)
	}
	return want, nil
}

func buildings1145Expected(f *sav.File) ([]building1145Want, map[uint16]uint32, error) {
	rows, present, err := f.Buildings()
	if err != nil || !present || f.World == nil {
		return nil, nil, fmt.Errorf("Building enumeration: present=%v err=%v", present, err)
	}
	byArchive, byOffset := map[uint16]sav.Building{}, map[int]sav.Building{}
	for _, row := range rows {
		if _, exists := byArchive[row.ArchiveIndex]; exists {
			return nil, nil, fmt.Errorf("duplicate Building locator %d", row.ArchiveIndex)
		}
		byArchive[row.ArchiveIndex] = row
		byOffset[row.Off] = row
	}
	p, end := f.World.BuildingsOff, f.World.BuildingsEnd
	if p < 0 || end > len(f.Body) || p > end-4 {
		return nil, nil, fmt.Errorf("invalid Building root span %d..%d", p, end)
	}
	n := binary.LittleEndian.Uint32(f.Body[p:])
	p += 4
	if uint64(n)*2 > uint64(end-p) {
		return nil, nil, fmt.Errorf("Building root count exceeds span")
	}
	var wants []building1145Want
	seen := map[uint16]bool{}
	for i := uint32(0); i < n; i++ {
		if p > end-2 {
			return nil, nil, fmt.Errorf("missing Building root tag %d", i)
		}
		tag := binary.LittleEndian.Uint16(f.Body[p:])
		p += 2
		if tag == 0 {
			continue
		}
		var row sav.Building
		var exists bool
		if tag < 0x8000 {
			row, exists = byArchive[tag]
		} else {
			class := ""
			if tag == 0xffff {
				if p > end-4 {
					return nil, nil, fmt.Errorf("truncated Building class tag")
				}
				size := int(binary.LittleEndian.Uint16(f.Body[p+2:]))
				p += 4 // schema and class-name length
				if size > end-p {
					return nil, nil, fmt.Errorf("truncated Building class name")
				}
				class, p = string(f.Body[p:p+size]), p+size
			}
			row, exists = byOffset[p]
			if class != "" && class != row.Class {
				return nil, nil, fmt.Errorf("Building class %q differs from locator %q", class, row.Class)
			}
		}
		if !exists {
			return nil, nil, fmt.Errorf("Building root %d/tag %#x has no matching locator", i, tag)
		}
		want, err := building1145Raw(f.Body[:end], row.Off, row.Class, row.ArchiveIndex)
		if err != nil {
			return nil, nil, err
		}
		if tag >= 0x8000 {
			p = want.next
		}
		if !seen[row.ArchiveIndex] {
			wants = append(wants, want)
			seen[row.ArchiveIndex] = true
		}
	}
	if p != end || len(wants) != len(rows) {
		return nil, nil, fmt.Errorf("Building locators=%d unique roots=%d endpoint=%d want=%d", len(rows), len(wants), p, end)
	}
	// SAV-CELLLOAD-109/110: a 2-byte key followed by the 52-byte payload;
	// Building identity is payload +0x0c. Repeated keys use the last entry.
	p = f.World.CellRecOff
	if p < 0 || p > len(f.Body)-2 {
		return nil, nil, fmt.Errorf("missing Building cell count")
	}
	cellCount := uint32(binary.LittleEndian.Uint16(f.Body[p:]))
	p += 2
	if cellCount == 0xffff {
		if p > len(f.Body)-4 {
			return nil, nil, fmt.Errorf("missing wide Building cell count")
		}
		cellCount = binary.LittleEndian.Uint32(f.Body[p:])
		p += 4
	}
	if uint64(cellCount)*54 > uint64(len(f.Body)-p) {
		return nil, nil, fmt.Errorf("Building cell count exceeds body")
	}
	links := map[uint16]uint32{}
	for i := uint32(0); i < cellCount; i++ {
		links[binary.LittleEndian.Uint16(f.Body[p:])] = binary.LittleEndian.Uint32(f.Body[p+14:])
		p += 54
	}
	return wants, links, nil
}

func buildings1145Differences(wants []building1145Want, links map[uint16]uint32, structures []sim.Structure, sources []sim.SavedStructure, cells []sim.SavedStructureCell, present bool) []string {
	var differences []string
	add := func(format string, args ...any) { differences = append(differences, fmt.Sprintf(format, args...)) }
	if !present || len(structures) != len(wants) || len(sources) != len(wants) {
		add("roster: present=%v live=%d retained=%d file=%d", present, len(structures), len(sources), len(wants))
	}
	byID, byKey := map[sim.StructureID]sim.Structure{}, map[uint32]sim.SavedStructure{}
	for _, s := range structures {
		if _, exists := byID[s.ID]; exists {
			add("duplicate live structure ID %d", s.ID)
		}
		byID[s.ID] = s
	}
	keyByID := map[sim.StructureID]uint32{}
	for _, s := range sources {
		if _, exists := byKey[s.SourceKey]; exists {
			add("duplicate live source key %#x", s.SourceKey)
		}
		if _, exists := keyByID[s.ID]; exists {
			add("duplicate retained structure ID %d", s.ID)
		}
		byKey[s.SourceKey], keyByID[s.ID] = s, s.SourceKey
	}
	for _, want := range wants {
		got, exists := byKey[want.source.SourceKey]
		if !exists {
			add("missing Building %#x", want.source.SourceKey)
			continue
		}
		// Native handles and ALM-script binding metadata are not SAV fields.
		// Source identity still joins the independent live Structure below.
		want.source.ID, want.source.HasAuthored, want.source.AuthoredIndex = got.ID, got.HasAuthored, got.AuthoredIndex
		want.live.ID = got.ID
		live, exists := byID[got.ID]
		comparison := live
		comparison.UseAmount = 0 // Installed potion metadata is not a SAV Building field.
		if !exists || comparison != want.live {
			add("Building %#x live=%+v file=%+v", got.SourceKey, live, want.live)
		}
		if !reflect.DeepEqual(got, want.source) {
			add("Building %#x retained=%+v file=%+v", got.SourceKey, got, want.source)
		}
	}
	if len(cells) != len(links) {
		add("cell population: live=%d file=%d", len(cells), len(links))
	}
	byCell := map[uint16]sim.SavedStructureCell{}
	for _, c := range cells {
		if _, exists := byCell[c.Cell]; exists {
			add("duplicate live cell %#x", c.Cell)
		}
		byCell[c.Cell] = c
	}
	keys := make([]int, 0, len(links))
	for key := range links {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)
	for _, key := range keys {
		want := links[uint16(key)]
		got, exists := byCell[uint16(key)]
		actual := uint32(0)
		if got.HasStructure {
			var bound bool
			actual, bound = keyByID[got.ID]
			if !bound {
				add("cell %#x names absent live structure %d", key, got.ID)
			}
		}
		if !exists || got.HasStructure != (want != 0) || actual != want {
			add("cell %#x: present=%v live Building=%#x file=%#x", key, exists, actual, want)
		}
	}
	return differences
}
