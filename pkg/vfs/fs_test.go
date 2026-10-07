package vfs_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"againrom/pkg/vfs"
)

// ---------------------------------------------------------------- fixtures

// resFile is one file to place in a synthetic .res archive. Path is either a
// bare name or "dir/name"; each segment must fit the format's 16-byte name
// field, and no fixture here needs deeper nesting than that.
type resFile struct {
	path string
	data []byte
}

// buildArchive assembles a fully synthetic .res archive from the documented
// container layout:
//
//	[ header (0x18) ][ data ][ registry: one 32-byte node per directory and file ]
//
// Nodes are emitted as: every directory, in first-seen order; then that
// directory's files, contiguously, in the order given; then the files at the
// archive root. A directory node's child range is therefore contiguous, and the
// root files, referenced by no directory, are the archive's file roots.
func buildArchive(files []resFile) []byte {
	type leaf struct {
		name string
		data []byte
	}

	var dirNames []string
	dirFiles := make(map[string][]resFile)
	var rootFiles []resFile
	for _, f := range files {
		i := strings.IndexByte(f.path, '/')
		if i < 0 {
			rootFiles = append(rootFiles, f)
			continue
		}
		dir, name := f.path[:i], f.path[i+1:]
		if _, seen := dirFiles[dir]; !seen {
			dirNames = append(dirNames, dir)
		}
		dirFiles[dir] = append(dirFiles[dir], resFile{path: name, data: f.data})
	}

	var leaves []leaf
	for _, dir := range dirNames {
		for _, f := range dirFiles[dir] {
			leaves = append(leaves, leaf{name: f.path, data: f.data})
		}
	}
	for _, f := range rootFiles {
		leaves = append(leaves, leaf{name: f.path, data: f.data})
	}

	var data []byte
	offsets := make([]uint32, len(leaves))
	for i, l := range leaves {
		offsets[i] = uint32(0x18 + len(data))
		data = append(data, l.data...)
	}

	le := binary.LittleEndian
	out := make([]byte, 0x18)
	le.PutUint32(out[0x00:], 0x31415926) // signature
	le.PutUint32(out[0x10:], uint32(0x18+len(data)))
	le.PutUint32(out[0x14:], uint32(len(dirNames)+len(leaves)))
	le.PutUint32(out[0x08:], uint32(len(dirNames)+len(rootFiles))) // root count
	out = append(out, data...)

	node := func(a, b, typ uint32, name string) {
		rec := make([]byte, 0x20)
		le.PutUint32(rec[0x04:], a)
		le.PutUint32(rec[0x08:], b)
		le.PutUint32(rec[0x0C:], typ)
		for i := 0x10; i < 0x20; i++ {
			rec[i] = 0xCD // the on-disk padding convention, past the NUL
		}
		copy(rec[0x10:], name)
		if len(name) < 16 {
			rec[0x10+len(name)] = 0x00
		}
		out = append(out, rec...)
	}

	child := len(dirNames)
	for _, dir := range dirNames {
		n := len(dirFiles[dir])
		node(uint32(child), uint32(n), 1, dir)
		child += n
	}
	for i, l := range leaves {
		node(offsets[i], uint32(len(l.data)), 0, l.name)
	}
	return out
}

// writeArchive writes a synthetic archive under dir and returns its host path.
func writeArchive(t *testing.T, dir, name string, files ...resFile) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buildArchive(files), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestOpenFileBackedKeepsOnlyTheIndexAndReadsTheCurrentPayload(t *testing.T) {
	host := writeArchive(t, t.TempDir(), "music.res", resFile{path: "menu.wav", data: []byte("old!")})
	f, err := vfs.OpenFileBacked([]string{host}, nil)
	if err != nil {
		t.Fatalf("OpenFileBacked: %v", err)
	}
	h, err := os.OpenFile(host, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.WriteAt([]byte("new!"), 0x18); err != nil {
		_ = h.Close()
		t.Fatal(err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := f.ReadFile("music/menu.wav")
	if err != nil || string(got) != "new!" {
		t.Fatalf("ReadFile = %q, %v; want current on-disk payload", got, err)
	}
}

// writeLoose writes a loose host file at the slash-separated rel under root, and
// returns its host path.
func writeLoose(t *testing.T, root, rel string, data []byte) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// ---------------------------------------------------------------- assertions

func mustOpen(t *testing.T, archives, dirs []string) *vfs.FS {
	t.Helper()
	f, err := vfs.Open(archives, dirs)
	if err != nil {
		t.Fatalf("Open(%v, %v): %v", archives, dirs, err)
	}
	if f == nil {
		t.Fatalf("Open(%v, %v) returned a nil *FS and no error", archives, dirs)
	}
	return f
}

func assertBytes(t *testing.T, f *vfs.FS, address string, want []byte) {
	t.Helper()
	got, err := f.ReadFile(address)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", address, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("ReadFile(%q) = %q, want %q", address, got, want)
	}
}

// assertAbsent is the miss assertion, and it carries SC-2's second half: every
// refusal returns no bytes, satisfies errors.Is(err, fs.ErrNotExist), and names
// the address it refused as the caller wrote it.
func assertAbsent(t *testing.T, f *vfs.FS, address string) {
	t.Helper()
	got, err := f.ReadFile(address)
	if got != nil {
		t.Errorf("ReadFile(%q) returned %d bytes, want none", address, len(got))
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadFile(%q) error = %v, want one satisfying fs.ErrNotExist", address, err)
	}
	if !strings.Contains(err.Error(), address) {
		t.Errorf("ReadFile(%q) error %q does not name the address it refused", address, err)
	}
}

func assertOpenFails(t *testing.T, archives, dirs []string) error {
	t.Helper()
	f, err := vfs.Open(archives, dirs)
	if err == nil {
		t.Fatalf("Open(%v, %v) succeeded, want a failure", archives, dirs)
	}
	if f != nil {
		t.Fatalf("Open(%v, %v) failed and still returned a non-nil *FS", archives, dirs)
	}
	return err
}

// ---------------------------------------------------------------- AC-1, AC-2

// TestOpenEmptyListsReadsAbsent - AC-1.
func TestOpenEmptyListsReadsAbsent(t *testing.T) {
	for _, lists := range []struct {
		name     string
		archives []string
		dirs     []string
	}{
		{"both nil", nil, nil},
		{"both empty", []string{}, []string{}},
	} {
		t.Run(lists.name, func(t *testing.T) {
			f := mustOpen(t, lists.archives, lists.dirs)
			for _, address := range []string{
				"graphics/x.bin",
				"x.bin",
				`graphics\terrain.3d\dirt.bmp`,
				"",
			} {
				assertAbsent(t, f, address)
			}
		})
	}
}

// TestIdentityFromHostPath - AC-2.
func TestIdentityFromHostPath(t *testing.T) {
	for _, c := range []struct {
		host string
		want string
	}{
		{"Graphics.RES", "graphics"},
		{"sub.d/graphics.res", "graphics"},
		{"ABCDEFGHIJKLMNOPQRST.res", "abcdefghijklmno"},
		{`sub.d\graphics.res`, "graphics"},
		{"/a/b/main.res", "main"},
		{`C:\games\Allods\Scenario.res`, "scenario"},
		{"graphics", "graphics"},
		{"abcdefghijklmno.res", "abcdefghijklmno"},
	} {
		got, err := vfs.Identity(c.host)
		if err != nil {
			t.Errorf("Identity(%q): %v", c.host, err)
			continue
		}
		if got != c.want {
			t.Errorf("Identity(%q) = %q, want %q", c.host, got, c.want)
		}
	}
}

// ---------------------------------------------------------------- AC-8, AC-9

func TestOpenFailsOnMalformedArchive(t *testing.T) {
	dir := t.TempDir()
	hosts := []string{
		writeArchive(t, dir, "graphics.res", resFile{path: "x.bin", data: []byte("GRAPHICS")}),
		filepath.Join(dir, "main.res"),
		writeArchive(t, dir, "scenario.res", resFile{path: "y.bin", data: []byte("SCENARIO")}),
	}
	if err := os.WriteFile(hosts[1], make([]byte, 0x18), 0o600); err != nil {
		t.Fatalf("write %s: %v", hosts[1], err)
	}

	err := assertOpenFails(t, hosts, nil)
	if want := "open " + hosts[1] + ": "; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("Open error %q does not start with %q", err, want)
	}
}

func TestOpenFailsOnMissingArchive(t *testing.T) {
	dir := t.TempDir()
	host := filepath.Join(dir, "graphics.res")
	err := assertOpenFails(t, []string{host}, nil)
	if want := "open " + host + ": "; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("Open error %q does not start with %q", err, want)
	}
}

func TestOpenRefusesEmptyIdentity(t *testing.T) {
	dir := t.TempDir()
	host := writeArchive(t, dir, ".res", resFile{path: "x.bin", data: []byte("XX")})

	if got, err := vfs.Identity(host); err == nil {
		t.Errorf("Identity(%q) = %q, want a refusal", host, got)
	}
	err := assertOpenFails(t, []string{host}, nil)
	if !strings.Contains(err.Error(), host) {
		t.Errorf("Open error %q does not name the host path it refused", err)
	}
}

func TestOpenDoesNotValidateDirectories(t *testing.T) {
	dir := t.TempDir()
	f := mustOpen(t, nil, []string{filepath.Join(dir, "nosuch"), dir})
	assertAbsent(t, f, "world/data.bin")
}

// ---------------------------------------------------------------- AC-3

// TestReadFileFoldsAndRequiresIdentitySegment - AC-3.
func TestReadFileFoldsAndRequiresIdentitySegment(t *testing.T) {
	dirt := []byte("DIRT")
	host := writeArchive(t, t.TempDir(), "graphics.res",
		resFile{path: "terrain.3d/dirt.bmp", data: dirt})
	f := mustOpen(t, []string{host}, nil)

	assertBytes(t, f, `graphics\terrain.3d\dirt.bmp`, dirt)
	assertBytes(t, f, "GRAPHICS/TERRAIN.3D/DIRT.BMP", dirt)
	assertAbsent(t, f, "terrain.3d/dirt.bmp")
}

func TestReadFileDistinctIdentitiesIgnoreOrder(t *testing.T) {
	dir := t.TempDir()
	fromGraphics := []byte("GRAPHICS-Y")
	fromMain := []byte("MAIN-Y")
	graphics := writeArchive(t, dir, "graphics.res", resFile{path: "x/y.bmp", data: fromGraphics})
	main := writeArchive(t, dir, "main.res", resFile{path: "x/y.bmp", data: fromMain})

	for _, order := range [][]string{
		{graphics, main},
		{main, graphics},
	} {
		f := mustOpen(t, order, nil)
		assertBytes(t, f, "graphics/x/y.bmp", fromGraphics)
		assertBytes(t, f, "main/x/y.bmp", fromMain)
		assertBytes(t, f, `MAIN\X\Y.BMP`, fromMain)
		assertAbsent(t, f, "x/y.bmp")
		assertAbsent(t, f, "nosuch/x/y.bmp")
		assertAbsent(t, f, "graphics/x/nosuch.bmp")
	}
}

// ---------------------------------------------------------------- AC-5

// TestReadFileSameIdentityFirstInListAnswers - AC-5. The two host basenames are
// equal, so both archives carry one identity and list order is the only thing
// that decides — the bytes swap when the list does.
func TestReadFileSameIdentityFirstInListAnswers(t *testing.T) {
	root := t.TempDir()
	first := []byte("FIRST")
	second := []byte("SECOND")
	a := writeArchive(t, filepath.Join(root, "a"), "graphics.res", resFile{path: "x.bin", data: first})
	b := writeArchive(t, filepath.Join(root, "b"), "graphics.res", resFile{path: "x.bin", data: second})

	assertBytes(t, mustOpen(t, []string{a, b}, nil), "graphics/x.bin", first)
	assertBytes(t, mustOpen(t, []string{b, a}, nil), "graphics/x.bin", second)
}

func TestReadFileSameIdentityFallsPastAnArchiveWithoutTheEntry(t *testing.T) {
	root := t.TempDir()
	want := []byte("SECOND")
	a := writeArchive(t, filepath.Join(root, "a"), "graphics.res", resFile{path: "other.bin", data: []byte("OTHER")})
	b := writeArchive(t, filepath.Join(root, "b"), "graphics.res", resFile{path: "x.bin", data: want})

	assertBytes(t, mustOpen(t, []string{a, b}, nil), "graphics/x.bin", want)
}

// ---------------------------------------------------------------- AC-6

// TestReadFileRefusesEmptyLeadingSegment - AC-6. The listed directory holds both
// addresses under their folded names and the archive holds x.bin, so the control
// read at the end is what makes the two refusals mean something: the fixture
// resolves the moment a leading segment is there.
func TestReadFileRefusesEmptyLeadingSegment(t *testing.T) {
	root := t.TempDir()
	inArchive := []byte("IN-ARCHIVE")
	host := writeArchive(t, filepath.Join(root, "arc"), "graphics.res",
		resFile{path: "x.bin", data: inArchive})

	loose := filepath.Join(root, "loose")
	writeLoose(t, loose, "x.bin", []byte("LOOSE-BARE"))
	writeLoose(t, loose, "graphics/x.bin", []byte("LOOSE-UNDER-GRAPHICS"))

	f := mustOpen(t, []string{host}, []string{loose})
	assertAbsent(t, f, `\graphics\x.bin`)
	assertAbsent(t, f, "/x.bin")
	assertAbsent(t, f, "/")
	assertBytes(t, f, "graphics/x.bin", inArchive)
}

// TestReadFileUnregisteredIdentityIsAbsent - AC-7.
func TestReadFileUnregisteredIdentityIsAbsent(t *testing.T) {
	host := writeArchive(t, t.TempDir(), "graphics.res",
		resFile{path: "x.bin", data: []byte("XX")})
	f := mustOpen(t, []string{host}, nil)
	assertAbsent(t, f, `nosuch\x.bin`)
}

func TestReadFileIdentityMustEqualLeadingSegment(t *testing.T) {
	host := writeArchive(t, t.TempDir(), "graphics.res",
		resFile{path: "x.bin", data: []byte("XX")})
	f := mustOpen(t, []string{host}, nil)

	for _, address := range []string{
		"graphicsx/x.bin",  // the leading segment extends the identity
		"graphic/x.bin",    // the leading segment is a prefix of the identity
		"graphics2/x.bin",  // and one byte differs, at the end
		"graphics",         // a bare identity: no separator, no remainder
		"graphics/",        // a separator, and an empty remainder
		"GRAPHICSX/X.BIN",  // the same, through the fold
		"x.bin/graphics",   // the identity is there, but not leading
		"a/graphics/x.bin", // the same, one segment in
	} {
		assertAbsent(t, f, address)
	}
	assertBytes(t, f, "graphics/x.bin", []byte("XX"))
}

// ---------------------------------------------------------------- AC-10

// denyRead makes path unreadable to this process and reports whether the host
// honoured it. A read must then fail with something that is not "not found",
// which the loose tier is the only tier that can be made to produce: the archive
// reader fails not-found alone.
//
// On Windows: a deny-read ACE for the current user. os.Stat still answers, since
// it reads the parent directory's metadata rather than opening the file, so the
// tier still holds the address and then fails to read it. Elsewhere: mode 0.
func denyRead(t *testing.T, path string) bool {
	t.Helper()
	if runtime.GOOS == "windows" {
		who, err := user.Current()
		if err != nil {
			t.Logf("denyRead: user.Current: %v", err)
			return false
		}
		if out, err := exec.Command("icacls", path, "/deny", who.Username+":(R)").CombinedOutput(); err != nil {
			t.Logf("denyRead: icacls /deny: %v: %s", err, out)
			return false
		}
		// Cleanups run last in, first out, so this precedes t.TempDir's own and
		// the temporary tree can still be removed.
		t.Cleanup(func() {
			if out, err := exec.Command("icacls", path, "/remove:d", who.Username).CombinedOutput(); err != nil {
				t.Logf("denyRead: icacls /remove:d: %v: %s", err, out)
			}
		})
	} else {
		if err := os.Chmod(path, 0); err != nil {
			t.Logf("denyRead: chmod: %v", err)
			return false
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	}
	// The mechanism counts as honoured only if a direct read now fails with
	// something other than "not found". A user the host does not restrain — root,
	// or an account reading through backup privilege — gets the bytes anyway, and
	// the case has to say so rather than pass on nothing.
	if _, err := os.ReadFile(path); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Logf("denyRead: a direct read of %s still yields error %v", path, err)
		return false
	}
	return true
}

// TestReadFileSurfacesUnreadableLooseFile - AC-10. The earlier directory holds
// the address as a regular file that cannot be read; the later holds a readable
// copy, and must not be reached.
func TestReadFileSurfacesUnreadableLooseFile(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "d1")
	second := filepath.Join(root, "d2")
	denied := writeLoose(t, first, "world/data.bin", []byte("DENIED"))
	readable := []byte("READABLE")
	writeLoose(t, second, "world/data.bin", readable)

	if !denyRead(t, denied) {
		t.Skip("this host does not express an unreadable regular file for this user; see the log above")
	}

	f := mustOpen(t, nil, []string{first, second})
	got, err := f.ReadFile("world/data.bin")
	if err == nil {
		t.Fatalf("ReadFile succeeded with %q; the unreadable earlier source was masked", got)
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile error = %v, want a source failure rather than not-found", err)
	}
	if got != nil {
		t.Errorf("ReadFile returned %d bytes alongside its error, want none", len(got))
	}
	if bytes.Equal(got, readable) {
		t.Errorf("ReadFile returned the later source's bytes")
	}

	// The later copy is readable, and reachable on its own: the refusal above is
	// the earlier source failing, not the address being unresolvable.
	assertBytes(t, mustOpen(t, nil, []string{second}), "world/data.bin", readable)
}

func TestReadFileWalksPastANonRegularFile(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "d1")
	second := filepath.Join(root, "d2")
	if err := os.MkdirAll(filepath.Join(first, "world", "data.bin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	want := []byte("READABLE")
	writeLoose(t, second, "world/data.bin", want)

	assertBytes(t, mustOpen(t, nil, []string{first, second}), "world/data.bin", want)
}

// ---------------------------------------------------------------- AC-13

// TestReadFileArchiveTierPrecedesDirectories - AC-13.
func TestReadFileArchiveTierPrecedesDirectories(t *testing.T) {
	root := t.TempDir()
	inArchive := []byte("IN-ARCHIVE")
	host := writeArchive(t, filepath.Join(root, "arc"), "graphics.res",
		resFile{path: "x.bin", data: inArchive})

	first := filepath.Join(root, "d1")
	second := filepath.Join(root, "d2")
	writeLoose(t, first, "graphics/x.bin", []byte("LOOSE-X"))
	looseY := []byte("LOOSE-Y")
	writeLoose(t, second, "graphics/y.bin", looseY)

	f := mustOpen(t, []string{host}, []string{first, second})
	assertBytes(t, f, "graphics/x.bin", inArchive)
	assertBytes(t, f, "graphics/y.bin", looseY)
}

func TestReadFileSeparatorlessAddressReachesTheLooseTier(t *testing.T) {
	root := t.TempDir()
	host := writeArchive(t, filepath.Join(root, "arc"), "graphics.res",
		resFile{path: "x.bin", data: []byte("IN-ARCHIVE")})
	loose := filepath.Join(root, "loose")
	want := []byte("LOOSE-BEAST")
	writeLoose(t, loose, "beast.alm", want)

	f := mustOpen(t, []string{host}, []string{loose})
	assertBytes(t, f, "BEAST.ALM", want)
	assertAbsent(t, f, "nosuch.alm")
}

func TestReadFileReturnsCallerOwnedBytes(t *testing.T) {
	root := t.TempDir()
	inArchive := []byte("IN-ARCHIVE")
	host := writeArchive(t, filepath.Join(root, "arc"), "graphics.res",
		resFile{path: "x.bin", data: inArchive})
	loose := filepath.Join(root, "loose")
	looseData := []byte("LOOSE-DATA")
	writeLoose(t, loose, "world/data.bin", looseData)

	f := mustOpen(t, []string{host}, []string{loose})
	for _, c := range []struct {
		address string
		want    []byte
	}{
		{"graphics/x.bin", inArchive},
		{"world/data.bin", looseData},
	} {
		got, err := f.ReadFile(c.address)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", c.address, err)
		}
		for i := range got {
			got[i] ^= 0xFF
		}
		assertBytes(t, f, c.address, c.want)
	}
}
