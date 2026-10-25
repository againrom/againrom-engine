package versionres

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/appicon"
	"againrom/internal/archtest"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := archtest.FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestCommittedObjectsMatchTheirVersionFiles fails when a VERSION file was
// edited without regenerating its object:
// go run ./internal/versionres/cmd/versionres
func TestCommittedObjectsMatchTheirVersionFiles(t *testing.T) {
	root := moduleRoot(t)
	for _, p := range Programs {
		dir := filepath.Join(root, filepath.FromSlash(p.Dir))
		version, err := ReadVersion(filepath.Join(root, filepath.FromSlash(p.VersionSource())))
		if err != nil {
			t.Fatalf("%s: %v", p.Describe(), err)
		}
		want, err := Syso(version, p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(dir, SysoName))
		if err != nil {
			t.Fatalf("%s: %v", p.Describe(), err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: %s does not match VERSION %s; run go run ./internal/versionres/cmd/versionres", p.Describe(), SysoName, version)
		}
	}
}

func TestFileVersionReadsTheFixedInformation(t *testing.T) {
	for _, v := range []string{"0.118.0", "3.4.5", "65535.0.65535"} {
		data, err := Syso(v, Programs[0])
		if err != nil {
			t.Fatal(err)
		}
		if got, err := FileVersion(data); err != nil || got != v {
			t.Errorf("%s read back as %q %v", v, got, err)
		}
	}
	if _, err := FileVersion([]byte("no resource")); err == nil {
		t.Fatal("an object without fixed file information was read")
	}
}

func TestMapEditorCarriesTheGameVersion(t *testing.T) {
	for _, p := range Programs {
		if p.Exe == "mapedit.exe" {
			if p.VersionSource() != "cmd/againrom" || p.Dir != "cmd/mapedit" {
				t.Fatalf("%+v", p)
			}
			return
		}
	}
	t.Fatal("no mapedit.exe row")
}

func TestSysoIsDeterministicAndFollowsTheVersion(t *testing.T) {
	p := Programs[0]
	a, err := Syso("1.2.3", p)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Syso("1.2.3", p)
	c, _ := Syso("1.2.4", p)
	if !bytes.Equal(a, b) || bytes.Equal(a, c) {
		t.Fatal("object is not a pure function of the version")
	}
	if _, err := Syso("1.2", p); err == nil {
		t.Fatal("bad version accepted")
	}
}

// objectResources walks the resource tree of an object Syso built and returns
// every leaf by type and name. Each leaf must carry a relocation.
func objectResources(t *testing.T, data []byte) map[[2]uint32][]byte {
	t.Helper()
	if binary.LittleEndian.Uint16(data[0:]) != 0x8664 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		t.Fatal("not an amd64 object with one section")
	}
	raw := binary.LittleEndian.Uint32(data[20+16:])
	at := binary.LittleEndian.Uint32(data[20+20:])
	rel := binary.LittleEndian.Uint32(data[20+24:])
	nrel := binary.LittleEndian.Uint16(data[20+32:])
	if rel != at+raw || int(rel)+10*int(nrel) > len(data) {
		t.Fatalf("relocation at %d, section %d+%d, file %d", rel, at, raw, len(data))
	}
	sec := data[at : at+raw]
	out := map[[2]uint32][]byte{}
	relocs := 0
	var walk func(off uint32, depth int, ids [3]uint32)
	walk = func(off uint32, depth int, ids [3]uint32) {
		n := int(binary.LittleEndian.Uint16(sec[off+14:]))
		for k := 0; k < n; k++ {
			id := binary.LittleEndian.Uint32(sec[off+16+8*uint32(k):])
			target := binary.LittleEndian.Uint32(sec[off+20+8*uint32(k):])
			ids[depth] = id
			if target&0x80000000 != 0 {
				walk(target&0x7FFFFFFF, depth+1, ids)
				continue
			}
			if depth != 2 || ids[2] != 0x0409 {
				t.Fatalf("leaf at depth %d language %#x", depth, ids[2])
			}
			dataOff := binary.LittleEndian.Uint32(sec[target:])
			size := binary.LittleEndian.Uint32(sec[target+4:])
			for r := 0; r < int(nrel); r++ {
				if binary.LittleEndian.Uint32(data[rel+10*uint32(r):]) == target {
					relocs++
				}
			}
			out[[2]uint32{ids[0], ids[1]}] = sec[dataOff : dataOff+size]
		}
	}
	walk(0, 0, [3]uint32{})
	if relocs != len(out) || int(nrel) != len(out) {
		t.Fatalf("%d leaves, %d relocations matched, %d declared", len(out), relocs, nrel)
	}
	return out
}

func TestSysoStructure(t *testing.T) {
	data, err := Syso("3.4.5", Programs[1])
	if err != nil {
		t.Fatal(err)
	}
	res := objectResources(t, data)
	sec := res[[2]uint32{rtVersion, 1}]
	if first := int(binary.LittleEndian.Uint16(sec)); first > len(sec) || len(sec)-first >= 4 {
		t.Fatalf("version block length %d of %d", binary.LittleEndian.Uint16(sec), len(sec))
	}
	for _, s := range []string{"VS_VERSION_INFO", "FileVersion", "3.4.5", "starter.exe", "Againrom starter"} {
		if !bytes.Contains(sec, utf16z(s)[:len(utf16z(s))-2]) {
			t.Errorf("%q missing from the resource", s)
		}
	}
	fixed := sec[40:]
	if binary.LittleEndian.Uint32(fixed) != 0xFEEF04BD || binary.LittleEndian.Uint32(fixed[8:]) != 3<<16|4 || binary.LittleEndian.Uint32(fixed[12:]) != 5<<16 {
		t.Fatalf("fixed file info %x", fixed[:20])
	}
}

// TestEachObjectHoldsItsProgramsIcon checks the icon group of every program: it
// lists every size of the icon package, its 256 entry is the embedded PNG byte
// for byte, and each smaller entry is a 32-bit DIB of its size.
func TestEachObjectHoldsItsProgramsIcon(t *testing.T) {
	root := moduleRoot(t)
	icons := map[string]bool{}
	for _, p := range Programs {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p.Dir), SysoName))
		if err != nil {
			t.Fatal(err)
		}
		icons[string(p.Icon)] = true
		res := objectResources(t, data)
		group := res[[2]uint32{rtGroupIcon, 1}]
		if len(group) < 6 || binary.LittleEndian.Uint16(group[2:]) != 1 {
			t.Fatalf("%s: no icon group", p.Describe())
		}
		n := int(binary.LittleEndian.Uint16(group[4:]))
		if n != len(appicon.Sizes) || len(group) != 6+14*n {
			t.Fatalf("%s: group lists %d icons in %d bytes", p.Describe(), n, len(group))
		}
		for i := 0; i < n; i++ {
			e := group[6+14*i:]
			want := appicon.Sizes[i]
			dim := int(e[0])
			if dim == 0 {
				dim = 256
			}
			id := binary.LittleEndian.Uint16(e[12:])
			body := res[[2]uint32{rtIcon, uint32(id)}]
			if dim != want || uint32(len(body)) != binary.LittleEndian.Uint32(e[8:]) {
				t.Fatalf("%s: entry %d is %d with %d bytes", p.Describe(), i, dim, len(body))
			}
			if want == 256 {
				if !bytes.Equal(body, appicon.PNG(p.Icon)) {
					t.Errorf("%s: the 256 entry is not the embedded PNG", p.Describe())
				}
				if _, err := png.Decode(bytes.NewReader(body)); err != nil {
					t.Errorf("%s: %v", p.Describe(), err)
				}
				continue
			}
			if binary.LittleEndian.Uint32(body) != 40 || int(binary.LittleEndian.Uint32(body[4:])) != want ||
				int(binary.LittleEndian.Uint32(body[8:])) != 2*want || binary.LittleEndian.Uint16(body[14:]) != 32 {
				t.Errorf("%s: entry %d header %x", p.Describe(), want, body[:20])
			}
			if wantLen := 40 + 4*want*want + (want+31)/32*4*want; len(body) != wantLen {
				t.Errorf("%s: entry %d is %d bytes, want %d", p.Describe(), want, len(body), wantLen)
			}
		}
	}
	if len(icons) != len(Programs) {
		t.Fatalf("programs share an icon: %v", icons)
	}
}
