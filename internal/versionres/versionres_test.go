package versionres

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

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

func TestSysoStructure(t *testing.T) {
	data, err := Syso("3.4.5", Programs[1])
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(data[0:]) != 0x8664 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		t.Fatal("not an amd64 object with one section")
	}
	raw := binary.LittleEndian.Uint32(data[20+16:])
	at := binary.LittleEndian.Uint32(data[20+20:])
	rel := binary.LittleEndian.Uint32(data[20+24:])
	if rel != at+raw || int(rel)+10 > len(data) {
		t.Fatalf("relocation at %d, section %d+%d, file %d", rel, at, raw, len(data))
	}
	sec := data[at : at+raw]
	size := binary.LittleEndian.Uint32(sec[76:])
	off := binary.LittleEndian.Uint32(sec[72:])
	if int(off+size) > len(sec) || binary.LittleEndian.Uint16(sec[off:]) != uint16(size) {
		t.Fatalf("resource data %d+%d of %d, first length %d", off, size, len(sec), binary.LittleEndian.Uint16(sec[off:]))
	}
	for _, s := range []string{"VS_VERSION_INFO", "FileVersion", "3.4.5", "starter.exe", "Againrom starter"} {
		if !bytes.Contains(sec, utf16z(s)[:len(utf16z(s))-2]) {
			t.Errorf("%q missing from the resource", s)
		}
	}
	fixed := sec[off+40:]
	if binary.LittleEndian.Uint32(fixed) != 0xFEEF04BD || binary.LittleEndian.Uint32(fixed[8:]) != 3<<16|4 || binary.LittleEndian.Uint32(fixed[12:]) != 5<<16 {
		t.Fatalf("fixed file info %x", fixed[:20])
	}
}
