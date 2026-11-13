package alm_test

// The texts record. Every fixture is assembled here; nothing reads a file.

import (
	"bytes"
	"encoding/binary"
	"maps"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
)

// textsFixture is a page tree in both languages and one shared page.
func textsFixture() map[string][]byte {
	return map[string][]byte{
		"en/briefing.txt": []byte("Hold the bridge.\r\n"),
		"ru/briefing.txt": []byte("Удержите мост.\r\n"),
		"en/event01.txt":  []byte("<part=1>\r\nThey come.\r\n"),
		"ru/event01.txt":  []byte("<part=1>\r\nОни идут.\r\n"),
		"title.txt":       {},
	}
}

func TestTextsRecordLayoutAndRoundTrip(t *testing.T) {
	pages := textsFixture()
	b, err := alm.EncodeTextsRecord(pages)
	if err != nil {
		t.Fatal(err)
	}
	le := binary.LittleEndian
	if le.Uint32(b) != 1 || le.Uint32(b[4:]) != 5 {
		t.Fatal("head words", b[:8])
	}
	// The first page in name order: its name length, name, text length, text.
	first := "en/briefing.txt"
	if le.Uint32(b[8:]) != uint32(len(first)) || string(b[12:12+len(first)]) != first ||
		le.Uint32(b[12+len(first):]) != uint32(len(pages[first])) {
		t.Fatal("first page", b[8:40])
	}
	got, err := alm.DecodeTextsRecord(b)
	if err != nil || !maps.EqualFunc(got, pages, bytes.Equal) {
		t.Fatalf("decode: %v %q", err, got)
	}
	if again, _ := alm.EncodeTextsRecord(got); !bytes.Equal(again, b) {
		t.Fatal("re-encode differs")
	}
	for n := range len(b) {
		if _, err := alm.DecodeTextsRecord(b[:n]); err == nil {
			t.Fatalf("a payload cut to %d bytes decoded", n)
		}
	}
}

func TestTextsRecordRidesBesideTheScriptRecordAndComesOut(t *testing.T) {
	plain := recordMap()
	if _, err := alm.AppendTextsRecord(plain, textsFixture()); err == nil {
		t.Fatal("an original map took the texts record")
	}
	extended, err := alm.AppendScriptRecord(plain, recordFixture())
	if err != nil {
		t.Fatal(err)
	}
	stream, err := alm.AppendTextsRecord(extended, textsFixture())
	if err != nil {
		t.Fatal(err)
	}
	m := openOK(t, stream)
	if m.RecordCount != 12 || m.Records[11].TypeID != alm.TextsRecordTypeID || !m.HasTextsRecord() || !m.HasScriptRecord() {
		t.Fatalf("record frame: count %d", m.RecordCount)
	}
	if s, err := m.Script(); err != nil || !sameRecordTriggers(s.Triggers, recordFixture().Triggers) {
		t.Fatalf("script beside the texts: %v", err)
	}
	pages, ok, err := alm.TextsOf(stream)
	if err != nil || !ok || !maps.EqualFunc(pages, textsFixture(), bytes.Equal) {
		t.Fatalf("texts: %v %v %q", ok, err, pages)
	}
	if _, err := alm.AppendTextsRecord(stream, textsFixture()); err == nil {
		t.Fatal("a second texts record was appended")
	}
	back, had := alm.WithoutTextsRecord(stream)
	if !had || !bytes.Equal(back, extended) {
		t.Fatal("taking the record out did not give the extended map back")
	}
	if same, had := alm.WithoutTextsRecord(extended); had || !bytes.Equal(same, extended) {
		t.Fatal("a map without the record changed")
	}
	if _, ok, err := alm.TextsOf(extended); ok || err != nil {
		t.Fatal("a map without the record reads texts", ok, err)
	}
	if openOK(t, plain).HasTextsRecord() {
		t.Fatal("an original map carries the texts record")
	}
}

func TestTextsRecordRefusesWhatTheFolderCannotHold(t *testing.T) {
	for name, pages := range map[string]map[string][]byte{
		"empty name":   {"": nil},
		"three parts":  {"en/a/b.txt": nil},
		"dot part":     {"../b.txt": nil},
		"backslash":    {`en\b.txt`: nil},
		"control":      {"b\n.txt": nil},
		"long name":    {strings.Repeat("a", alm.TextsRecordName+1): nil},
		"long page":    {"b.txt": make([]byte, alm.TextsRecordText+1)},
		"not utf-8":    {"\xff.txt": nil},
		"leading root": {"/b.txt": nil},
	} {
		if _, err := alm.EncodeTextsRecord(pages); err == nil {
			t.Errorf("%s: encoded", name)
		}
	}
	le := binary.LittleEndian
	head := func(version, n uint32) []byte {
		return le.AppendUint32(le.AppendUint32(nil, version), n)
	}
	page := func(b []byte, name, text string) []byte {
		b = le.AppendUint32(b, uint32(len(name)))
		b = append(b, name...)
		b = le.AppendUint32(b, uint32(len(text)))
		return append(b, text...)
	}
	for name, payload := range map[string][]byte{
		"version":   head(2, 0),
		"count":     head(1, alm.TextsRecordPages+1),
		"short":     head(1, 1),
		"twice":     page(page(head(1, 2), "a.txt", "x"), "a.txt", "y"),
		"order":     page(page(head(1, 2), "b.txt", "x"), "a.txt", "y"),
		"trailing":  append(page(head(1, 1), "a.txt", "x"), 0),
		"bad name":  page(head(1, 1), "../a.txt", "x"),
		"huge text": le.AppendUint32(append(le.AppendUint32(head(1, 1), 5), "a.txt"...), 0xffffffff),
	} {
		if _, err := alm.DecodeTextsRecord(payload); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
}
