// Package versionres writes the Windows resources of a program as a COFF object
// (.syso) that "go build" links into the executable. Explorer's Properties >
// Details tab reads the version resource; Explorer, shortcuts and the taskbar
// read the icon.
//
// The object holds RT_VERSION name 1, built from the program's VERSION file and
// the Program row below, and the program's icon: RT_GROUP_ICON name 1 and one
// RT_ICON per size of the icon package, all in language US English. The 256
// size is the embedded PNG itself, the smaller sizes are 32-bit DIBs. The output
// is a pure function of the VERSION file, the row and the embedded icon, so a
// committed object can be checked against a fresh one byte for byte.
package versionres

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"againrom/internal/appicon"
	"againrom/internal/buildinfo"
)

// Program is one executable that carries a version resource.
type Program struct {
	Dir         string // module-relative directory holding the .syso, and VERSION unless VersionDir names another
	VersionDir  string // module-relative directory whose VERSION the program carries; empty is Dir
	Exe         string // the executable's file name
	Product     string
	Description string
	Icon        appicon.Name
}

// VersionSource is the module-relative directory holding the VERSION file the
// program's resource is built from.
func (p Program) VersionSource() string {
	if p.VersionDir != "" {
		return p.VersionDir
	}
	return p.Dir
}

// Programs lists every executable with a version resource.
var Programs = []Program{
	{Dir: "cmd/againrom", Exe: "againrom.exe", Product: "Againrom", Description: "Againrom game", Icon: appicon.Game},
	{Dir: "cmd/starter", Exe: "starter.exe", Product: "Againrom", Description: "Againrom starter", Icon: appicon.Starter},
	// The map editor is part of the game and carries the game's version.
	{Dir: "cmd/mapedit", VersionDir: "cmd/againrom", Exe: "mapedit.exe", Product: "Againrom", Description: "Againrom map editor", Icon: appicon.MapEditor},
}

// SysoName is the object's file name in a program's directory. The suffix keeps
// the Go toolchain from linking it into any other platform's build.
const SysoName = "rsrc_windows_amd64.syso"

// Syso builds the COFF object for p at version (MAJOR.MINOR.PATCH).
func Syso(version string, p Program) ([]byte, error) {
	version, err := buildinfo.ParseVersion(version)
	if err != nil {
		return nil, err
	}
	var num [3]uint16
	for i, part := range strings.Split(version, ".") {
		n, _ := strconv.Atoi(part)
		num[i] = uint16(n)
	}
	blob := versionInfo(num, [][2]string{
		{"FileDescription", p.Description},
		{"FileVersion", version},
		{"InternalName", strings.TrimSuffix(p.Exe, ".exe")},
		{"OriginalFilename", p.Exe},
		{"ProductName", p.Product},
		{"ProductVersion", version},
	})
	icons, err := iconResources(p.Icon)
	if err != nil {
		return nil, err
	}
	res := append(icons, resource{typ: rtVersion, name: 1, data: blob})
	return coff(res), nil
}

// ReadVersion returns the validated version in dir's VERSION file.
func ReadVersion(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	if err != nil {
		return "", err
	}
	return buildinfo.ParseVersion(string(data))
}

// FileVersion reads the file version, MAJOR.MINOR.PATCH, from the fixed file
// information of an object Syso built.
func FileVersion(syso []byte) (string, error) {
	at := bytes.Index(syso, le32(0xFEEF04BD))
	if at < 0 || at+16 > len(syso) {
		return "", fmt.Errorf("no fixed file information")
	}
	ms := binary.LittleEndian.Uint32(syso[at+8:])
	ls := binary.LittleEndian.Uint32(syso[at+12:])
	return buildinfo.ParseVersion(fmt.Sprintf("%d.%d.%d", ms>>16, ms&0xFFFF, ls>>16))
}

func utf16z(s string) []byte {
	units := append(utf16.Encode([]rune(s)), 0)
	b := make([]byte, 2*len(units))
	for i, u := range units {
		binary.LittleEndian.PutUint16(b[2*i:], u)
	}
	return b
}

func pad4(b []byte) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}

// block is one VS_VERSIONINFO-family node: length, value length, type, key,
// padding, value, padding, children. Each child is already padded to four bytes.
func block(key string, value []byte, valueLen int, typ uint16, children []byte) []byte {
	body := pad4(append(make([]byte, 6), utf16z(key)...))
	body = append(body, value...)
	if len(children) > 0 {
		body = pad4(body)
		body = append(body, children...)
	}
	binary.LittleEndian.PutUint16(body[0:], uint16(len(body)))
	binary.LittleEndian.PutUint16(body[2:], uint16(valueLen))
	binary.LittleEndian.PutUint16(body[4:], typ)
	return body
}

func le32(vs ...uint32) []byte {
	b := make([]byte, 4*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[4*i:], v)
	}
	return b
}

func versionInfo(v [3]uint16, strs [][2]string) []byte {
	ms := uint32(v[0])<<16 | uint32(v[1])
	ls := uint32(v[2]) << 16
	fixed := le32(0xFEEF04BD, 0x00010000, ms, ls, ms, ls,
		0x3F,       // file flags mask
		0,          // file flags
		0x00040004, // VOS_NT_WINDOWS32
		1,          // VFT_APP
		0, 0, 0)
	var table []byte
	for _, s := range strs {
		val := utf16z(s[1])
		table = append(table, pad4(block(s[0], val, len(val)/2, 1, nil))...)
	}
	stringTable := pad4(block("040904B0", nil, 0, 1, table))
	stringInfo := pad4(block("StringFileInfo", nil, 0, 1, stringTable))
	translation := le32(0x04B00409)
	varInfo := pad4(block("VarFileInfo", nil, 0, 1, pad4(block("Translation", translation, 4, 0, nil))))
	return block("VS_VERSION_INFO", fixed, len(fixed), 0, append(stringInfo, varInfo...))
}

const (
	rtIcon      = 3
	rtGroupIcon = 14
	rtVersion   = 16
	langUS      = 0x0409
)

// resource is one leaf of the resource tree: type / name / langUS.
type resource struct {
	typ, name uint32
	data      []byte
}

// iconResources returns the RT_ICON leaves, numbered from 1 in size order, and
// the RT_GROUP_ICON leaf that lists them.
func iconResources(n appicon.Name) ([]resource, error) {
	images := appicon.Images(n)
	if len(images) == 0 {
		return nil, fmt.Errorf("no icon %q", n)
	}
	var out []resource
	group := binary.LittleEndian.AppendUint16(nil, 0)
	group = binary.LittleEndian.AppendUint16(group, 1)
	group = binary.LittleEndian.AppendUint16(group, uint16(len(images)))
	for i, im := range images {
		size := im.Bounds().Dx()
		var data []byte
		if size == 256 {
			data = appicon.PNG(n)
		} else {
			data = dib(im)
		}
		id := uint32(i + 1)
		out = append(out, resource{typ: rtIcon, name: id, data: data})
		dim := byte(size)
		if size >= 256 {
			dim = 0
		}
		group = append(group, dim, dim, 0, 0)
		group = binary.LittleEndian.AppendUint16(group, 1)  // planes
		group = binary.LittleEndian.AppendUint16(group, 32) // bit count
		group = binary.LittleEndian.AppendUint32(group, uint32(len(data)))
		group = binary.LittleEndian.AppendUint16(group, uint16(id))
	}
	return append(out, resource{typ: rtGroupIcon, name: 1, data: group}), nil
}

// dib encodes a square image as the body of a 32-bit icon image: a bitmap
// header with doubled height, bottom-up BGRA rows and an all-zero AND mask.
func dib(im image.Image) []byte {
	b := im.Bounds()
	size := b.Dx()
	out := le32(40, uint32(size), uint32(2*size))
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint16(out, 32)
	out = append(out, le32(0, uint32(4*size*size), 0, 0, 0, 0)...)
	for y := b.Max.Y - 1; y >= b.Min.Y; y-- {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
			out = append(out, c.B, c.G, c.R, c.A)
		}
	}
	maskRow := (size + 31) / 32 * 4
	return append(out, make([]byte, maskRow*size)...)
}

// coff wraps the resources in an object file with a single .rsrc section whose
// resource tree is type / name / language -> data, types and names ascending.
func coff(res []resource) []byte {
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].typ != res[j].typ {
			return res[i].typ < res[j].typ
		}
		return res[i].name < res[j].name
	})
	var types []uint32
	names := map[uint32][]int{}
	for i, r := range res {
		if len(names[r.typ]) == 0 {
			types = append(types, r.typ)
		}
		names[r.typ] = append(names[r.typ], i)
	}
	// Directory offsets: root, one directory per type, one per resource, then
	// the data entries and the data.
	at := 16 + 8*len(types)
	typeDir := map[uint32]int{}
	for _, t := range types {
		typeDir[t] = at
		at += 16 + 8*len(names[t])
	}
	langDir := make([]int, len(res))
	for i := range res {
		langDir[i] = at
		at += 24
	}
	entryAt := make([]int, len(res))
	for i := range res {
		entryAt[i] = at
		at += 16
	}
	dataAt := make([]int, len(res))
	for i, r := range res {
		dataAt[i] = at
		at += (len(r.data) + 3) &^ 3
	}
	dir := func(entries ...[2]uint32) []byte {
		b := make([]byte, 16)
		binary.LittleEndian.PutUint16(b[14:], uint16(len(entries)))
		for _, e := range entries {
			b = append(b, le32(e[0], e[1])...)
		}
		return b
	}
	var rsrc []byte
	var root [][2]uint32
	for _, t := range types {
		root = append(root, [2]uint32{t, 0x80000000 | uint32(typeDir[t])})
	}
	rsrc = append(rsrc, dir(root...)...)
	for _, t := range types {
		var es [][2]uint32
		for _, i := range names[t] {
			es = append(es, [2]uint32{res[i].name, 0x80000000 | uint32(langDir[i])})
		}
		rsrc = append(rsrc, dir(es...)...)
	}
	for i := range res {
		rsrc = append(rsrc, dir([2]uint32{langUS, uint32(entryAt[i])})...)
	}
	for i, r := range res {
		rsrc = append(rsrc, le32(uint32(dataAt[i]), uint32(len(r.data)), 0, 0)...)
	}
	for _, r := range res {
		rsrc = append(rsrc, pad4(append([]byte(nil), r.data...))...)
	}

	const (
		sizeHdr  = 20 + 40
		relocLen = 10
	)
	var out []byte
	u16 := func(v uint16) { out = binary.LittleEndian.AppendUint16(out, v) }
	u32 := func(v uint32) { out = binary.LittleEndian.AppendUint32(out, v) }
	symAt := uint32(sizeHdr + len(rsrc) + relocLen*len(res))
	u16(0x8664)
	u16(1)
	u32(0)
	u32(symAt)
	u32(1)
	u16(0)
	u16(0)
	out = append(out, ".rsrc\x00\x00\x00"...)
	u32(0)
	u32(0)
	u32(uint32(len(rsrc)))
	u32(sizeHdr)
	u32(uint32(sizeHdr + len(rsrc)))
	u32(0)
	u16(uint16(len(res)))
	u16(0)
	u32(0x40000040)
	out = append(out, rsrc...)
	for i := range res {
		u32(uint32(entryAt[i]))
		u32(0)
		u16(3) // IMAGE_REL_AMD64_ADDR32NB
	}
	out = append(out, ".rsrc\x00\x00\x00"...)
	u32(0)
	u16(1)
	u16(0)
	out = append(out, 3, 0)
	u32(4)
	return out
}

// Describe names a generated object for a failure message.
func (p Program) Describe() string { return fmt.Sprintf("%s (%s)", p.Exe, p.Dir) }
