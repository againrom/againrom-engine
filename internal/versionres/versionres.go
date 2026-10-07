// Package versionres writes the Windows version resource of a program as a COFF
// object (.syso) that "go build" links into the executable. Explorer's
// Properties > Details tab reads the resource.
//
// The object holds one resource, RT_VERSION name 1, language US English, built
// from the program's VERSION file and the Program row below. The output is a
// pure function of those two inputs, so a committed object can be checked
// against a fresh one byte for byte.
package versionres

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"

	"againrom/internal/buildinfo"
)

// Program is one executable that carries a version resource.
type Program struct {
	Dir         string // module-relative directory holding VERSION and the .syso
	Exe         string // the executable's file name
	Product     string
	Description string
}

// Programs lists every executable with a version resource.
var Programs = []Program{
	{Dir: "cmd/againrom", Exe: "againrom.exe", Product: "Againrom", Description: "Againrom game"},
	{Dir: "cmd/starter", Exe: "starter.exe", Product: "Againrom", Description: "Againrom starter"},
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
	return coff(blob), nil
}

// ReadVersion returns the validated version in dir's VERSION file.
func ReadVersion(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	if err != nil {
		return "", err
	}
	return buildinfo.ParseVersion(string(data))
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

// coff wraps data in an object file with a single .rsrc section whose resource
// tree is RT_VERSION / 1 / 0x0409 -> data.
func coff(data []byte) []byte {
	const (
		rootDir  = 0
		nameDir  = 24
		langDir  = 48
		entry    = 72
		dataAt   = 88
		sizeHdr  = 20 + 40
		symSize  = 18
		relocLen = 10
	)
	dir := func(id uint32, target uint32) []byte {
		b := make([]byte, 16)
		binary.LittleEndian.PutUint16(b[14:], 1) // one ID entry
		return append(b, le32(id, target)...)
	}
	var rsrc []byte
	rsrc = append(rsrc, dir(16, 0x80000000|nameDir)...)
	rsrc = append(rsrc, dir(1, 0x80000000|langDir)...)
	rsrc = append(rsrc, dir(0x0409, entry)...)
	rsrc = append(rsrc, le32(dataAt, uint32(len(data)), 0, 0)...)
	rsrc = append(rsrc, pad4(append([]byte(nil), data...))...)

	var out []byte
	u16 := func(v uint16) { out = binary.LittleEndian.AppendUint16(out, v) }
	u32 := func(v uint32) { out = binary.LittleEndian.AppendUint32(out, v) }
	symAt := uint32(sizeHdr + len(rsrc) + relocLen)
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
	u16(1)
	u16(0)
	u32(0x40000040)
	out = append(out, rsrc...)
	u32(entry)
	u32(0)
	u16(3) // IMAGE_REL_AMD64_ADDR32NB
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
