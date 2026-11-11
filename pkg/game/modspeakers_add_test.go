package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/internal/witness"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// modFolderBytes is every file of the folder dir, by path.
func modFolderBytes(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		out[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// checkAddWroteNothing fails unless the mod folder holds the files before and
// its digest is the one the editor compares on the next save.
func checkAddWroteNothing(t *testing.T, s *mapInspectionEdits, dir string, before map[string]string) {
	t.Helper()
	after := modFolderBytes(t, dir)
	for p, b := range before {
		if a, ok := after[p]; !ok || a != b {
			t.Errorf("%s changed: held %d bytes, now %d (present %v)", p, len(b), len(a), ok)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			t.Errorf("%s appeared", p)
		}
	}
	digest, err := mod.ContentDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if digest != s.inspector.mod.digest {
		t.Errorf("the digest the next save compares is stale")
	}
}

func writeModText(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func speakerRecordText(id int32, picture string) string {
	return fmt.Sprintf("[[speaker]]\nid = %d\nname = \"Karl\"\npicture = %q\n", id, picture)
}

func facePNG(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "face.png")
	speakerPNG(t, p, color.RGBA{R: 10, G: 200, B: 30, A: 255})
	return p
}

// A picture the mod reader would refuse is refused before anything is written.
func TestModSpeakerRefusesAPictureOverTheModFileLimit(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	p := facePNG(t)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(b, bytes.Repeat([]byte{7}, 2<<20)...), 0o600); err != nil {
		t.Fatal(err)
	}
	before := modFolderBytes(t, dir)
	if _, err := s.AddSpeaker(speakerLady, "Lady", p); err == nil || !strings.Contains(err.Error(), strconv.Itoa(mod.MaxFileBytes)) {
		t.Fatalf("the add answered %v", err)
	}
	checkAddWroteNothing(t, s, dir, before)
}

// A name holding a no-break space is written so the mod reader reads it back.
func TestModSpeakerNameWithANoBreakSpaceRoundTrips(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	name := "Lady Ann"
	if _, err := s.AddSpeaker(speakerLady, name, facePNG(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile)))
	if err != nil {
		t.Fatal(err)
	}
	d, err := mod.ReadSpeakers("ambush", dir, mod.SpeakersFile, src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Speakers) != 1 || d.Speakers[0].Name != name {
		t.Fatalf("read back %+v", d.Speakers)
	}
	if err := s.inspector.SetMod(dir); err != nil || s.inspector.mod.contextIssue != "" {
		t.Fatalf("the mod does not load after the add: %v %q", err, s.inspector.mod.contextIssue)
	}
}

// When the file write fails the picture the add wrote goes, with the folder
// it made, and the digest stays what it was.
func TestModSpeakerFailedFileWriteLeavesNothing(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	if err := os.Mkdir(filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile)), 0o755); err != nil {
		t.Fatal(err)
	}
	before := modFolderBytes(t, dir)
	if _, err := s.AddSpeaker(speakerLady, "Lady", facePNG(t)); err == nil {
		if _, err := s.SaveIntoMod(); err == nil {
			t.Fatal("save went through a speakers file that cannot be written")
		}
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	checkAddWroteNothing(t, s, dir, before)
	if _, err := os.Stat(filepath.Join(dir, "speakers")); !os.IsNotExist(err) {
		t.Fatalf("the picture folder stayed: %v", err)
	}
	if got, _ := s.UnitSpeaker(speakerLady); got > 0 {
		t.Fatalf("the failed add left record %d in the catalog", got)
	}
}

// A picture already at the new record's path comes back after a failed write.
func TestModSpeakerFailedFileWriteRestoresAnOldPicture(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	writeModText(t, dir, speakerPictureFile(speakerNumberStart("ambush")), "an old file no record names")
	digest, err := mod.ContentDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.inspector.mod.digest = digest
	if err := os.Mkdir(filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile)), 0o755); err != nil {
		t.Fatal(err)
	}
	before := modFolderBytes(t, dir)
	if _, err := s.AddSpeaker(speakerLady, "Lady", facePNG(t)); err == nil {
		if _, err := s.SaveIntoMod(); err == nil {
			t.Fatal("save went through a speakers file that cannot be written")
		}
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	checkAddWroteNothing(t, s, dir, before)
}

// While the mod's records did not load the add is refused at the click and at
// the write, in the read-only line, and the mod's own record keeps its picture.
func TestModSpeakerRefusesWhileTheRecordsDidNotLoad(t *testing.T) {
	start := speakerNumberStart("ambush")
	cases := []struct {
		name   string
		damage func(t *testing.T, dir string)
	}{
		{"a bad picture on one record", func(t *testing.T, dir string) {
			witness.PNG(t, filepath.Join(dir, "speakers", "small.png"), image.NewRGBA(image.Rect(0, 0, 10, 10)))
			writeModText(t, dir, mod.SpeakersFile, speakerRecordText(start, speakerPictureFile(start))+speakerRecordText(start+1, "speakers/small.png"))
		}},
		{"a failing script", func(t *testing.T, dir string) {
			writeModText(t, dir, mod.SpeakersFile, speakerRecordText(start, speakerPictureFile(start)))
			writeModText(t, dir, "main.star", "def init(game, settings):\n    fail(\"no\")\n")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, s, dir := modSpeakerEditor(t)
			if err := os.MkdirAll(filepath.Join(dir, "speakers"), 0o755); err != nil {
				t.Fatal(err)
			}
			speakerPNG(t, filepath.Join(dir, filepath.FromSlash(speakerPictureFile(start))), color.RGBA{R: 200, A: 255})
			c.damage(t, dir)
			if err := s.inspector.SetMod(dir); err != nil {
				t.Fatal(err)
			}
			issue := s.inspector.mod.contextIssue
			if issue == "" {
				t.Fatal("the mod loaded")
			}
			_, miss := s.UnitSpeaker(speakerLady)
			if s.CanAddSpeaker(speakerLady, miss) {
				t.Fatal("a mod whose records did not load offers a record")
			}
			want := s.inspector.word("editor.mod.readonly") + " " + issue
			if got := s.SpeakerAddRefusal(miss); got != want {
				t.Fatalf("the click would show %q", got)
			}
			before := modFolderBytes(t, dir)
			if _, err := s.AddSpeaker(speakerLady, "Lady", facePNG(t)); err == nil || err.Error() != want {
				t.Fatalf("the add answered %v", err)
			}
			checkAddWroteNothing(t, s, dir, before)

			openPageOne(t, e)
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 1, K: 0, Flag: false})
			if !ok {
				t.Fatalf("no button on the speaker row:\n%s", trigLines(e))
			}
			messageClick(t, e, r, ok)
			p := e.CellPoint(pickLadyCell)
			e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Time{})
			messageClick(t, e, image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}, true)
			if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, s.inspector.word("editor.mod.readonly")) {
				t.Fatalf("the click does not state the refusal:\n%s", lines)
			}
			checkAddWroteNothing(t, s, dir, before)
		})
	}
}

// A record that reached the file after the mod was opened is seen: the number
// never rests on the layered catalog alone.
func TestModSpeakerNumberReadsTheModsOwnFile(t *testing.T) {
	_, s, _ := modSpeakerEditor(t)
	start := speakerNumberStart("ambush")
	got, err := s.inspector.freeSpeakerNumber([]byte(speakerRecordText(start, "speakers/old.png")))
	if err != nil {
		t.Fatal(err)
	}
	if got != start+1 {
		t.Fatalf("the number is %d, want %d past the record the file holds", got, start+1)
	}
	if _, err := s.inspector.freeSpeakerNumber([]byte("[[speaker]]\nid = 1\n")); err == nil {
		t.Fatal("a speakers file that does not parse gave a number")
	}
}

func mustModSpeakersFile(t *testing.T, dir string) []byte {
	t.Helper()
	b, err := modSpeakersFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A number whose picture file a record of the mod names is not given to a new
// record: the add would replace that record's picture.
func TestModSpeakerNumberPassesAPictureARecordNames(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	start := speakerNumberStart("ambush")
	other := start + 1
	if other > mod.LastModSpeaker {
		other = mod.FirstModSpeaker
	}
	writeModText(t, dir, mod.SpeakersFile, speakerRecordText(other, speakerPictureFile(start)))
	got, err := s.inspector.freeSpeakerNumber(mustModSpeakersFile(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	if got == start {
		t.Fatalf("number %d would replace record %d's picture", got, other)
	}
}

// The same id starts at the same number, in the band, and ids differ.
func TestSpeakerNumberStartIsStable(t *testing.T) {
	if got := speakerNumberStart("ambush"); got != speakerNumberStart("ambush") || got < mod.FirstModSpeaker || got > mod.LastModSpeaker {
		t.Fatalf("start %d", got)
	}
	if got := speakerNumberStart("ambush"); got != 74321 {
		t.Fatalf("start for ambush is %d; the hash changed, and a mod's next number with it", got)
	}
	seen := map[int32]bool{}
	for _, id := range []string{"mod-a", "mod-b", "mod-c", "ambush", "tavern-rows"} {
		seen[speakerNumberStart(id)] = true
	}
	if len(seen) != 5 {
		t.Fatalf("five ids gave %d starts", len(seen))
	}
}

// Two mods made in the editor in one folder get different numbers, and a
// number the other mod's file holds is passed.
func TestModSpeakerNumberTwoModsInOneFolder(t *testing.T) {
	parent := t.TempDir()
	_, a, _ := modSpeakerEditorIn(t, parent, "mod-a")
	_, b, _ := modSpeakerEditorIn(t, parent, "mod-b")
	ida, err := a.AddSpeaker(speakerLady, "Lady", facePNG(t))
	if err != nil {
		t.Fatal(err)
	}
	idb, err := b.AddSpeaker(speakerLady, "Lady", facePNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if ida == idb {
		t.Fatalf("both mods took %d", ida)
	}
	// mod-c's first number is already mod-a's by hand.
	writeModText(t, filepath.Join(parent, "mod-a"), mod.SpeakersFile, speakerRecordText(speakerNumberStart("mod-c"), "a.png"))
	_, c, dir := modSpeakerEditorIn(t, parent, "mod-c")
	got, err := c.inspector.freeSpeakerNumber(mustModSpeakersFile(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	if got != speakerNumberStart("mod-c")+1 {
		t.Fatalf("mod-c's number is %d, its start %d is the other mod's", got, speakerNumberStart("mod-c"))
	}
}

// Two mods made in separate folders see nothing of each other and still get
// different numbers: each starts at its own.
func TestModSpeakerNumberTwoModsInSeparateFolders(t *testing.T) {
	ids := map[string]int64{}
	for _, name := range []string{"mod-a", "mod-b"} {
		_, s, _ := modSpeakerEditorIn(t, t.TempDir(), name)
		id, err := s.AddSpeaker(speakerLady, "Lady", facePNG(t))
		if err != nil {
			t.Fatal(err)
		}
		if id != int64(speakerNumberStart(name)) {
			t.Fatalf("mod %s took %d, its start is %d", name, id, speakerNumberStart(name))
		}
		ids[name] = id
	}
	if ids["mod-a"] == ids["mod-b"] {
		t.Fatalf("both mods took %d", ids["mod-a"])
	}
}

// A mod that removed its record and adds again starts where it started.
func TestModSpeakerNumberAfterARecordWasRemoved(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	first, err := s.AddSpeaker(speakerLady, "Lady", facePNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile))); err != nil {
		t.Fatal(err)
	}
	if err := s.inspector.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	s.workspace = nil
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	again, err := s.AddSpeaker(speakerLady, "Lady", facePNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatalf("the mod took %d, then %d after the record was removed", first, again)
	}
}

// The walk goes round the top of the band to its bottom.
func TestModSpeakerNumberWrapsToTheBottomOfTheBand(t *testing.T) {
	id := ""
	for i := 0; id == ""; i++ {
		if c := "m" + strconv.Itoa(i); mod.LastModSpeaker-speakerNumberStart(c) < 3 {
			id = c
		}
	}
	_, s, dir := modSpeakerEditorIn(t, t.TempDir(), id)
	var file strings.Builder
	for n := speakerNumberStart(id); n <= mod.LastModSpeaker; n++ {
		file.WriteString(speakerRecordText(n, "a.png"))
	}
	writeModText(t, dir, mod.SpeakersFile, file.String())
	got, err := s.inspector.freeSpeakerNumber(mustModSpeakersFile(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	if got != mod.FirstModSpeaker {
		t.Fatalf("number %d, want the band's bottom %d", got, mod.FirstModSpeaker)
	}
}

// The band's top survives both games' tag readers and the message tag writer,
// and a number one digit longer would not.
func TestSpeakerNumberTopIsReadWholeByBothGames(t *testing.T) {
	number := strconv.Itoa(mod.LastModSpeaker)
	tag := "part=1,npc=" + number
	if got, named := (firstEventTags{}).speaker(tag); !named || got != mod.LastModSpeaker {
		t.Fatalf("the first game read %d %v", got, named)
	}
	if got, named := (secondEventTags{}).speaker(tag); !named || got != mod.LastModSpeaker {
		t.Fatalf("the second game read %d %v", got, named)
	}
	if got, _ := (secondEventTags{}).speaker("part=1,npc=" + number + "9"); got == mod.LastModSpeaker*10+9 {
		t.Fatalf("the second game reads six digits; the band could end higher")
	}
}
