package mod

import (
	"encoding/binary"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
)

// recordedMap is the smallest record frame the record walk accepts, the
// type-0, type-1 and type-2 records empty, with pages as its texts record
// when pages is not nil.
func recordedMap(t *testing.T, pages map[string]string) string {
	t.Helper()
	le := binary.LittleEndian
	count := uint32(3)
	var tail []byte
	if pages != nil {
		held := map[string][]byte{}
		for name, body := range pages {
			held[name] = []byte(body)
		}
		var err error
		if tail, err = alm.FrameTextsRecord(held); err != nil {
			t.Fatal(err)
		}
		count++
	}
	b := le.AppendUint32(nil, 0x0052374D)
	b = le.AppendUint32(b, alm.FileHeaderSize)
	b = le.AppendUint32(b, 0)
	b = le.AppendUint32(b, count)
	b = le.AppendUint32(b, 1)
	for typeID := range uint32(3) {
		for _, w := range []uint32{7, alm.RecordHeaderSize, 0, typeID, 0} {
			b = le.AppendUint32(b, w)
		}
	}
	return string(append(b, tail...))
}

// insideFixture is a map whose texts record and text folder both hold pages:
// event01 in both, event02 only in the folder, event04 only in the record's
// Russian pages, event05 in the Russian pages of both.
func insideFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "inside")
	files := map[string]string{
		"maps/east.almx": recordedMap(t, map[string]string{
			"en/event01.txt": "Inside one",
			"ru/event01.txt": "Внутри один",
			"ru/event04.txt": "Внутри четыре",
			"ru/event05.txt": "Внутри пять",
			"title.txt":      "Inside title",
		}),
		"maps/east.text/en/event01.txt": "Folder one",
		"maps/east.text/en/event02.txt": "Folder two",
		"maps/east.text/ru/event05.txt": "Папка пять",
	}
	for name, body := range files {
		writeTreeFile(t, dir, name, body)
	}
	return dir
}

func pagesText(files map[string][]byte) map[string]string {
	out := map[string]string{}
	for name, b := range files {
		out[name] = string(b)
	}
	return out
}

// The texts record is read before the folders; a page only the folder holds
// is still read; the record's other-language pages fill before the folder's.
func TestMissionPagesComeFromTheMapsTextsRecordFirst(t *testing.T) {
	dir := insideFixture(t)
	files, other, err := (Mission{Map: "maps/east.almx"}).ReadTexts(dir, "en")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"event01.txt": "Inside one", "event02.txt": "Folder two", "title.txt": "Inside title",
		"event04.txt": "Внутри четыре", "event05.txt": "Внутри пять",
	}
	if got := pagesText(files); !maps.Equal(got, want) {
		t.Fatalf("pages %q", got)
	}
	if !reflect.DeepEqual(other, map[string]bool{"event04.txt": true, "event05.txt": true}) {
		t.Fatalf("other-language names %v", other)
	}
	files, other, err = (Mission{Map: "maps/east.almx"}).ReadTexts(dir, "ru")
	if err != nil || string(files["event01.txt"]) != "Внутри один" || string(files["event02.txt"]) != "Folder two" ||
		string(files["event05.txt"]) != "Внутри пять" || !other["event02.txt"] || other["event01.txt"] || len(files) != 5 {
		t.Fatalf("ru pages %q other %v: %v", pagesText(files), other, err)
	}
}

// A map that holds its pages in the record needs no folder, also where the
// mission row declares one; a map with neither still reads as it did.
func TestMissionPagesFromTheRecordNeedNoFolder(t *testing.T) {
	for _, texts := range []string{"", "mission/text/{lang}"} {
		dir := t.TempDir()
		writeTreeFile(t, dir, "maps/east.almx", recordedMap(t, map[string]string{"en/briefing.txt": "Hold.", "ru/briefing.txt": "Держать."}))
		m := Mission{Map: "maps/east.almx", Texts: texts}
		for lang, want := range map[string]string{"en": "Hold.", "ru": "Держать."} {
			files, other, err := m.ReadTexts(dir, lang)
			if err != nil || string(files["briefing.txt"]) != want || len(files) != 1 || len(other) != 0 {
				t.Fatalf("texts %q %s: %q %v %v", texts, lang, pagesText(files), other, err)
			}
		}
	}
	dir := t.TempDir()
	writeTreeFile(t, dir, "maps/east.almx", recordedMap(t, nil))
	if files, _, err := (Mission{Map: "maps/east.almx"}).ReadTexts(dir, "en"); err != nil || len(files) != 0 {
		t.Fatalf("a map without the record or a folder: %q %v", files, err)
	}
	if _, _, err := (Mission{Map: "maps/east.almx", Texts: "mission/text/{lang}"}).ReadTexts(dir, "en"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a declared folder missing beside a map without the record: %v", err)
	}
	m := Mission{Map: "maps/east.almx", MapBytes: []byte(recordedMap(t, map[string]string{"en/briefing.txt": "Bytes."}))}
	if files, _, err := m.ReadTexts(dir, "en"); err != nil || string(files["briefing.txt"]) != "Bytes." {
		t.Fatalf("the map bytes a mission row holds: %q %v", files, err)
	}
	writeTreeFile(t, dir, "bad/map.alm", recordedMap(t, map[string]string{"en/Briefing.txt": "x"}))
	if _, _, err := (Mission{Map: "bad/map.alm"}).ReadTexts(dir, "en"); err == nil {
		t.Fatal("a record page that is no mission text file name was read")
	}
}

// A mod whose map holds its pages in the record reads the same pages packed
// as from its folder.
func TestPackedModReadsTheMapsTextsRecordAsItsFolder(t *testing.T) {
	dir := insideFixture(t)
	writeTreeFile(t, dir, "mod.toml", "id = \"inside\"\ntitle = \"Inside\"\nversion = \"1\"\n")
	packed := packInto(t, dir, t.TempDir())
	for _, lang := range []string{"en", "ru"} {
		folder, folderOther, err := (Mission{Map: "maps/east.almx"}).ReadTexts(dir, lang)
		if err != nil {
			t.Fatal(err)
		}
		res, resOther, err := (Mission{Map: "maps/east.almx"}).ReadTexts(packed, lang)
		if err != nil || !reflect.DeepEqual(folder, res) || !reflect.DeepEqual(folderOther, resOther) || len(folder) != 5 {
			t.Fatalf("%s: folder %q packed %q: %v", lang, pagesText(folder), pagesText(res), err)
		}
	}
}
