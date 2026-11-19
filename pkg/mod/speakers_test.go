package mod

import (
	"fmt"
	"strings"
	"testing"
)

// The speaker writer writes only what the string reader reads: every name comes
// back from the file as it went in.
func TestSpeakerWriterRoundTripsEveryName(t *testing.T) {
	for _, name := range []string{
		"Lady", "Lady Ann", "Карл", `say "hi"`, `back\slash`, "tab\tstop", "line\nbreak", "cr\rend",
		"#not a comment", "it's", "\x01\x7f", "Ё ",
	} {
		s := NewSpeaker(FirstModSpeaker, name, "speakers/"+name+".png")
		s.Bind("maps/"+name+".alm", 7)
		d, err := ParseSpeakers("m", SpeakersFile, appendSpeaker(t, nil, s))
		if err != nil {
			t.Fatalf("name %q: %v", name, err)
		}
		got := d.Speakers[0]
		if got.Name != name || got.Picture != s.Picture || got.Map != s.Map || got.Unit != 7 || !got.Bound {
			t.Fatalf("name %q read back as %+v", name, got)
		}
	}
}

// The speaker writer keeps the file's other text and starts its record on a line
// of its own.
func TestSpeakerWriterKeepsTheFile(t *testing.T) {
	src := []byte("[[speaker]]\nid = 20000\nname = 'Karl'\npicture = \"a.png\"")
	d, err := ParseSpeakers("m", SpeakersFile, appendSpeaker(t, src, NewSpeaker(20001, "Ann", "b.png")))
	if err != nil || len(d.Speakers) != 2 || d.Speakers[0].Name != "Karl" || d.Speakers[1].Name != "Ann" {
		t.Fatalf("%v %+v", err, d.Speakers)
	}
}

// MoveSpeakerMaps rewrites only the map value of a record a move takes,
// matching the path folded, and keeps every other byte.
func TestMoveSpeakerMapsRewritesOnlyAMovedMap(t *testing.T) {
	src := "# speakers\r\n[[speaker]]\r\nid = 20000\r\nname = 'Karl'\r\npicture = 'a.png'\r\nmap   = 'Mission/EN/Map.alm'  # bound\r\nunit = 3\r\n" +
		"[[speaker]]\r\nid = 20001\r\nname = 'Ann'\r\npicture = 'b.png'\r\nmap = 'maps/other.alm'\r\nunit = 4\r\n"
	moves := []FileMove{{From: "mission/text/en/event01.txt", To: "maps/map.text/en/event01.txt"}, {From: "mission/en/map.alm", To: "maps/en/map.alm"}}
	out, moved, err := MoveSpeakerMaps([]byte(src), moves)
	want := strings.Replace(src, "'Mission/EN/Map.alm'", `"maps/en/map.alm"`, 1)
	if err != nil || !moved || string(out) != want {
		t.Fatalf("%v %v\n%q\nwant\n%q", err, moved, out, want)
	}
	if out, moved, err := MoveSpeakerMaps([]byte(src), moves[:1]); err != nil || moved || string(out) != src {
		t.Fatalf("an unmoved map changed the file: %v %v %q", err, moved, out)
	}
}

// A pending picture is read from memory, and a record whose picture is
// neither pending nor in the mod is still refused at its line.
func TestReadSpeakersTakesAPendingPicture(t *testing.T) {
	src := []byte("[[speaker]]\nid = 20000\nname = \"Karl\"\npicture = \"speakers/new.png\"\n")
	d, err := ReadSpeakers("m", t.TempDir(), SpeakersFile, src, func(rel string) ([]byte, bool) {
		return []byte{1, 2}, rel == "speakers/new.png"
	})
	if err != nil || string(d.Speakers[0].PictureBytes) != "\x01\x02" {
		t.Fatalf("%v %+v", err, d.Speakers)
	}
	if _, err := ReadSpeakers("m", t.TempDir(), SpeakersFile, src, nil); err == nil || !strings.Contains(err.Error(), "new.png") {
		t.Fatalf("a missing picture read as %v", err)
	}
}

// The band's top holds and the next number is refused: the file's limit is
// the limit of the number the message tag can carry.
func TestSpeakerNumberBand(t *testing.T) {
	file := func(id int) []byte {
		return []byte(fmt.Sprintf("[[speaker]]\nid = %d\nname = \"Karl\"\npicture = \"a.png\"\n", id))
	}
	for _, id := range []int{FirstModSpeaker, LastModSpeaker} {
		if _, err := ParseSpeakers("m", SpeakersFile, file(id)); err != nil {
			t.Fatalf("id %d: %v", id, err)
		}
	}
	for _, id := range []int{FirstModSpeaker - 1, LastModSpeaker + 1} {
		if _, err := ParseSpeakers("m", SpeakersFile, file(id)); err == nil {
			t.Fatalf("id %d was accepted", id)
		}
	}
}

// appendSpeaker is src with record s added through the speakers writer.
func appendSpeaker(t *testing.T, src []byte, s Speaker) []byte {
	t.Helper()
	var d Data
	if len(src) != 0 {
		parsed, err := ParseSpeakers("m", SpeakersFile, src)
		if err != nil {
			t.Fatal(err)
		}
		d.Speakers = parsed
	}
	d.Speakers.Speakers = append(d.Speakers.Speakers, s)
	out, err := writeSpeakers(SpeakersFile, src, d)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
