package winicon

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"unicode/utf16"
)

const (
	resourceTypeIcon      = 3
	resourceTypeGroupIcon = 14

	// MaxSide is the largest side an icon image has: an icon directory stores a
	// side of 256 as the byte 0 and no side is larger.
	MaxSide = 256

	// MaxImages bounds the images one group may list. Shipped icons carry a
	// handful; the bound keeps what a crafted group can ask for small.
	MaxImages = 64

	groupHeaderLen = 6
	groupEntryLen  = 14
	dibHeaderLen   = 40
	highBit        = 1 << 31
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// Image is one image of an icon.
type Image struct {
	// Depth is the bits per pixel the image was stored at: 1, 4, 8, 24 or 32. A
	// PNG image reports 32.
	Depth int

	// Pix holds the pixels with straight alpha, origin top-left.
	Pix *image.NRGBA
}

// Group is one icon group.
type Group struct {
	// Name is the group's resource name: "#119" for the numeric id 119, the
	// string itself for a named group.
	Name string

	// Language is the group's language id.
	Language uint16

	// Images lists the group's images in the order the group names them.
	Images []Image
}

// FromExecutable reads the first icon group of the executable r holds, in
// resource-directory order (named groups first, then numeric ids ascending, the
// order Windows takes an executable's own icon in), and decodes all its images.
// The first entry of a group name's language directory is the one read; each of
// its images is looked up in that language, or in the first the image has.
func FromExecutable(r io.ReaderAt) (*Group, error) {
	res, err := readResources(r)
	if err != nil {
		return nil, err
	}
	types, err := res.directory(0)
	if err != nil {
		return nil, err
	}
	groups, ok := findDir(types, resourceTypeGroupIcon)
	if !ok {
		return nil, errors.New("winicon: the executable has no icon group")
	}
	icons, ok := findDir(types, resourceTypeIcon)
	if !ok {
		return nil, errors.New("winicon: the executable has icon groups but no icons")
	}
	names, err := res.directory(groups.offset)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, errors.New("winicon: the icon group type lists no group")
	}
	first := names[0]
	if !first.dir {
		return nil, errors.New("winicon: an icon group name holds data where a language directory belongs")
	}
	group := &Group{Name: first.label()}
	langs, err := res.directory(first.offset)
	if err != nil {
		return nil, err
	}
	if len(langs) == 0 {
		return nil, fmt.Errorf("winicon: group %s has no language", group.Name)
	}
	group.Language = uint16(langs[0].id)
	listing, err := res.leaf(langs[0])
	if err != nil {
		return nil, fmt.Errorf("winicon: group %s: %w", group.Name, err)
	}
	ids, err := parseGroup(listing)
	if err != nil {
		return nil, fmt.Errorf("winicon: group %s: %w", group.Name, err)
	}
	iconNames, err := res.directory(icons.offset)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		data, err := res.icon(iconNames, id, langs[0].id)
		if err != nil {
			return nil, fmt.Errorf("winicon: group %s, icon %d: %w", group.Name, id, err)
		}
		img, err := decodeImage(data)
		if err != nil {
			return nil, fmt.Errorf("winicon: group %s, icon %d: %w", group.Name, id, err)
		}
		group.Images = append(group.Images, img)
	}
	return group, nil
}

// parseGroup reads a group's directory and returns the icon resource ids it
// lists, in order.
func parseGroup(b []byte) ([]uint16, error) {
	if len(b) < groupHeaderLen {
		return nil, fmt.Errorf("directory of %d bytes is shorter than its %d-byte header", len(b), groupHeaderLen)
	}
	if kind := binary.LittleEndian.Uint16(b[2:]); binary.LittleEndian.Uint16(b) != 0 || kind != 1 {
		return nil, fmt.Errorf("directory is not an icon group (reserved %d, type %d)", binary.LittleEndian.Uint16(b), kind)
	}
	count := int(binary.LittleEndian.Uint16(b[4:]))
	switch {
	case count == 0:
		return nil, errors.New("directory lists no image")
	case count > MaxImages:
		return nil, fmt.Errorf("directory lists %d images, more than the %d allowed", count, MaxImages)
	case len(b) < groupHeaderLen+count*groupEntryLen:
		return nil, fmt.Errorf("directory of %d bytes cannot hold its %d entries", len(b), count)
	}
	ids := make([]uint16, count)
	for i := range ids {
		ids[i] = binary.LittleEndian.Uint16(b[groupHeaderLen+i*groupEntryLen+12:])
	}
	return ids, nil
}

// resources is the resource section of one executable.
type resources struct {
	data []byte // the section as the file stores it
	va   uint32 // the section's virtual address
	root uint32 // offset of the resource directory within data
}

// entry is one entry of a resource directory.
type entry struct {
	named  bool
	name   string
	id     uint32
	offset uint32 // from the resource directory's start
	dir    bool   // the entry leads to a directory rather than to data
}

func (e entry) label() string {
	if e.named {
		return e.name
	}
	return fmt.Sprintf("#%d", e.id)
}

func findDir(entries []entry, id uint32) (entry, bool) {
	for _, e := range entries {
		if !e.named && e.id == id && e.dir {
			return e, true
		}
	}
	return entry{}, false
}

func readResources(r io.ReaderAt) (*resources, error) {
	f, err := pe.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("winicon: not a Windows executable: %w", err)
	}
	var dirs []pe.DataDirectory
	var listed uint32
	switch h := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		dirs, listed = h.DataDirectory[:], h.NumberOfRvaAndSizes
	case *pe.OptionalHeader64:
		dirs, listed = h.DataDirectory[:], h.NumberOfRvaAndSizes
	default:
		return nil, errors.New("winicon: the executable has no optional header")
	}
	if listed <= pe.IMAGE_DIRECTORY_ENTRY_RESOURCE || dirs[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress == 0 {
		return nil, errors.New("winicon: the executable has no resource directory")
	}
	at := dirs[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress
	for _, s := range f.Sections {
		if at < s.VirtualAddress || at-s.VirtualAddress >= max(s.VirtualSize, s.Size) {
			continue
		}
		data, err := s.Data()
		if err != nil {
			return nil, fmt.Errorf("winicon: reading section %s: %w", s.Name, err)
		}
		if uint64(at-s.VirtualAddress) >= uint64(len(data)) {
			return nil, errors.New("winicon: the resource directory lies beyond its section's stored bytes")
		}
		return &resources{data: data, va: s.VirtualAddress, root: at - s.VirtualAddress}, nil
	}
	return nil, errors.New("winicon: the resource directory lies in no section")
}

// slice returns n bytes of the section from start, or an error when they are
// not all stored.
func (r *resources) slice(start, n uint64) ([]byte, error) {
	if start+n < start || start+n > uint64(len(r.data)) {
		return nil, errors.New("the resource directory reaches beyond the section")
	}
	return r.data[start : start+n], nil
}

// directory reads the resource directory at off and returns its entries: the
// named ones first, then the numeric ones, as the file lists them.
func (r *resources) directory(off uint32) ([]entry, error) {
	start := uint64(r.root) + uint64(off)
	head, err := r.slice(start, 16)
	if err != nil {
		return nil, fmt.Errorf("winicon: %w", err)
	}
	named := int(binary.LittleEndian.Uint16(head[12:]))
	count := named + int(binary.LittleEndian.Uint16(head[14:]))
	body, err := r.slice(start+16, uint64(count)*8)
	if err != nil {
		return nil, fmt.Errorf("winicon: %w", err)
	}
	entries := make([]entry, count)
	for i := range entries {
		key := binary.LittleEndian.Uint32(body[8*i:])
		next := binary.LittleEndian.Uint32(body[8*i+4:])
		e := entry{offset: next &^ highBit, dir: next&highBit != 0}
		if i < named {
			if key&highBit == 0 {
				return nil, errors.New("winicon: a named resource entry carries no name offset")
			}
			name, err := r.text(key &^ highBit)
			if err != nil {
				return nil, err
			}
			e.named, e.name = true, name
		} else {
			e.id = key
		}
		entries[i] = e
	}
	return entries, nil
}

// text reads the UTF-16 resource name at off.
func (r *resources) text(off uint32) (string, error) {
	start := uint64(r.root) + uint64(off)
	head, err := r.slice(start, 2)
	if err != nil {
		return "", fmt.Errorf("winicon: %w", err)
	}
	n := uint64(binary.LittleEndian.Uint16(head))
	raw, err := r.slice(start+2, n*2)
	if err != nil {
		return "", fmt.Errorf("winicon: %w", err)
	}
	units := make([]uint16, n)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(raw[2*i:])
	}
	return string(utf16.Decode(units)), nil
}

// leaf returns the bytes a data entry points at. The entry's RVA is read
// against the resource section itself: every linker puts the data there.
func (r *resources) leaf(e entry) ([]byte, error) {
	if e.dir {
		return nil, errors.New("a resource directory lies where data belongs")
	}
	head, err := r.slice(uint64(r.root)+uint64(e.offset), 16)
	if err != nil {
		return nil, err
	}
	rva, size := binary.LittleEndian.Uint32(head), binary.LittleEndian.Uint32(head[4:])
	if rva < r.va {
		return nil, errors.New("a resource's data lies before its section")
	}
	return r.slice(uint64(rva-r.va), uint64(size))
}

// icon returns the bytes of icon resource id, in language lang when the icon
// has it and in its first language otherwise.
func (r *resources) icon(names []entry, id uint16, lang uint32) ([]byte, error) {
	for _, n := range names {
		if n.named || n.id != uint32(id) {
			continue
		}
		if !n.dir {
			return nil, errors.New("an icon name holds data where a language directory belongs")
		}
		langs, err := r.directory(n.offset)
		if err != nil {
			return nil, err
		}
		if len(langs) == 0 {
			return nil, errors.New("the icon has no language")
		}
		pick := langs[0]
		for _, l := range langs {
			if !l.named && l.id == lang {
				pick = l
				break
			}
		}
		return r.leaf(pick)
	}
	return nil, errors.New("the group names an icon the executable does not hold")
}

func decodeImage(b []byte) (Image, error) {
	if bytes.HasPrefix(b, pngMagic) {
		return decodePNG(b)
	}
	return decodeBitmap(b)
}

func decodePNG(b []byte) (Image, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return Image{}, fmt.Errorf("PNG image: %w", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxSide || cfg.Height > MaxSide {
		return Image{}, fmt.Errorf("PNG image of %dx%d is outside 1 to %d a side", cfg.Width, cfg.Height, MaxSide)
	}
	src, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return Image{}, fmt.Errorf("PNG image: %w", err)
	}
	pix := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	draw.Draw(pix, pix.Bounds(), src, src.Bounds().Min, draw.Src)
	return Image{Depth: 32, Pix: pix}, nil
}

// decodeBitmap reads one bitmap image: a BITMAPINFOHEADER, the colour table,
// the colour bitmap and the AND mask, both bottom-up with rows padded to four
// bytes.
func decodeBitmap(b []byte) (Image, error) {
	if len(b) < dibHeaderLen {
		return Image{}, fmt.Errorf("bitmap of %d bytes is shorter than its %d-byte header", len(b), dibHeaderLen)
	}
	le := binary.LittleEndian
	if size := le.Uint32(b); size != dibHeaderLen {
		return Image{}, fmt.Errorf("bitmap header of %d bytes, want %d", size, dibHeaderLen)
	}
	width, height := int64(int32(le.Uint32(b[4:]))), int64(int32(le.Uint32(b[8:])))
	planes, depth := le.Uint16(b[12:]), int(le.Uint16(b[14:]))
	compression, used := le.Uint32(b[16:]), le.Uint32(b[32:])
	switch {
	case planes != 1:
		return Image{}, fmt.Errorf("bitmap with %d planes, want 1", planes)
	case compression != 0:
		return Image{}, fmt.Errorf("bitmap compression %d, want 0", compression)
	case depth != 1 && depth != 4 && depth != 8 && depth != 24 && depth != 32:
		return Image{}, fmt.Errorf("bitmap of %d bits per pixel, want 1, 4, 8, 24 or 32", depth)
	case width < 1 || width > MaxSide:
		return Image{}, fmt.Errorf("bitmap width %d is outside 1 to %d", width, MaxSide)
	case height < 2 || height%2 != 0 || height/2 > MaxSide:
		return Image{}, fmt.Errorf("bitmap height %d does not hold a colour bitmap and a mask of 1 to %d rows", height, MaxSide)
	}
	entries := int(used)
	if depth <= 8 && used == 0 {
		entries = 1 << depth
	}
	if used > 256 || (depth <= 8 && entries > 1<<depth) {
		return Image{}, fmt.Errorf("bitmap colour table of %d entries at %d bits per pixel", used, depth)
	}
	w, rows := int(width), int(height/2)
	xorStride := (w*depth + 31) / 32 * 4
	andStride := (w + 31) / 32 * 4
	tableAt := dibHeaderLen
	xorAt := tableAt + entries*4
	andAt := xorAt + rows*xorStride
	if need := andAt + rows*andStride; len(b) < need {
		return Image{}, fmt.Errorf("bitmap holds %d bytes, its header needs %d", len(b), need)
	}
	table, xor, and := b[tableAt:xorAt], b[xorAt:andAt], b[andAt:]

	hasAlpha := false
	if depth == 32 {
		for y := 0; y < rows && !hasAlpha; y++ {
			for x := 0; x < w; x++ {
				if xor[y*xorStride+4*x+3] != 0 {
					hasAlpha = true
					break
				}
			}
		}
	}
	pix := image.NewNRGBA(image.Rect(0, 0, w, rows))
	for y := 0; y < rows; y++ {
		file := rows - 1 - y
		xr := xor[file*xorStride : (file+1)*xorStride]
		ar := and[file*andStride : (file+1)*andStride]
		for x := 0; x < w; x++ {
			var c color.NRGBA
			switch depth {
			case 1, 4, 8:
				i := paletteIndex(xr, x, depth)
				if i >= entries {
					return Image{}, fmt.Errorf("bitmap pixel %d,%d names colour %d of a %d-entry table", x, y, i, entries)
				}
				c = color.NRGBA{R: table[4*i+2], G: table[4*i+1], B: table[4*i], A: 255}
			case 24:
				c = color.NRGBA{R: xr[3*x+2], G: xr[3*x+1], B: xr[3*x], A: 255}
			default:
				c = color.NRGBA{R: xr[4*x+2], G: xr[4*x+1], B: xr[4*x], A: xr[4*x+3]}
			}
			switch {
			case depth == 32 && hasAlpha:
			case ar[x/8]>>(7-uint(x%8))&1 == 1:
				c = color.NRGBA{}
			default:
				c.A = 255
			}
			pix.SetNRGBA(x, y, c)
		}
	}
	return Image{Depth: depth, Pix: pix}, nil
}

// paletteIndex is pixel x of a row of 1, 4 or 8 bits per pixel, the leftmost
// pixel in the most significant bits.
func paletteIndex(row []byte, x, depth int) int {
	switch depth {
	case 1:
		return int(row[x/8] >> (7 - uint(x%8)) & 1)
	case 4:
		return int(row[x/2] >> (4 * (1 - uint(x%2))) & 15)
	}
	return int(row[x])
}
