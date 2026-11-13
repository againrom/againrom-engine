package alm

// The Againrom texts record: an extended map's pages, carried in an .almx map
// as one more record of the Againrom record family, beside the script record.
// The original steps over its id as over every id past its record switch.
//
// Every word is a little-endian uint32. The payload is the head, then the
// pages back to back in name order, and it ends exactly after the last one:
//
//	head  +0x00 version (1), +0x04 page count
//	page  +0x00 name length L, then L bytes of name: the page's path in the
//	      map's text folder, "name.txt" or "<lang>/name.txt", UTF-8 with "/"
//	      +4+L text length T, then T bytes of text, as the page file holds them

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// The texts record's numbers.
const (
	TextsRecordTypeID   = 101
	TextsRecordVersion  = 1
	TextsRecordHeadSize = 8
	// TextsRecordPages bounds the page count, TextsRecordName a page name's
	// bytes and TextsRecordText a page's bytes.
	TextsRecordPages = 1024
	TextsRecordName  = 128
	TextsRecordText  = 1 << 20
)

func textsRefusal(format string, args ...any) error {
	return fmt.Errorf("alm: the texts record: "+format, args...)
}

// checkTextsName refuses a page name the folder could not hold: empty, over
// TextsRecordName bytes, not UTF-8, with a control character, a backslash or
// a colon, or not one or two path parts each of which names a file.
func checkTextsName(name string) error {
	if name == "" || len(name) > TextsRecordName || !utf8.ValidString(name) ||
		strings.ContainsAny(name, "\\:") || strings.ContainsFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return textsRefusal("page name %q is not a page path", name)
	}
	parts := strings.Split(name, "/")
	if len(parts) > 2 || slices.ContainsFunc(parts, func(p string) bool { return p == "" || p == "." || p == ".." }) {
		return textsRefusal("page name %q is not a page path", name)
	}
	return nil
}

// EncodeTextsRecord is the one encoder of the texts record's payload: pages,
// keyed by their path in the map's text folder, in name order. It refuses
// what DecodeTextsRecord refuses.
func EncodeTextsRecord(pages map[string][]byte) ([]byte, error) {
	if len(pages) > TextsRecordPages {
		return nil, textsRefusal("%d pages, at most %d", len(pages), TextsRecordPages)
	}
	names := make([]string, 0, len(pages))
	for name := range pages {
		names = append(names, name)
	}
	slices.Sort(names)
	le := binary.LittleEndian
	out := make([]byte, TextsRecordHeadSize)
	le.PutUint32(out, TextsRecordVersion)
	le.PutUint32(out[4:], uint32(len(names)))
	for _, name := range names {
		if err := checkTextsName(name); err != nil {
			return nil, err
		}
		text := pages[name]
		if len(text) > TextsRecordText {
			return nil, textsRefusal("page %q is %d bytes, at most %d", name, len(text), TextsRecordText)
		}
		out = le.AppendUint32(out, uint32(len(name)))
		out = append(out, name...)
		out = le.AppendUint32(out, uint32(len(text)))
		out = append(out, text...)
	}
	return out, nil
}

// DecodeTextsRecord is the one decoder of the texts record's payload. Every
// length is compared with the bytes left before anything is allocated; a
// name out of order or twice, a bad name, a count or length past its bound,
// a payload cut short and a byte after the last page are refused.
func DecodeTextsRecord(payload []byte) (map[string][]byte, error) {
	le := binary.LittleEndian
	total := len(payload)
	if total < TextsRecordHeadSize {
		return nil, textsRefusal("%d bytes, under its %d-byte head", total, TextsRecordHeadSize)
	}
	if v := le.Uint32(payload); v != TextsRecordVersion {
		return nil, textsRefusal("version %d, want %d", v, TextsRecordVersion)
	}
	n := int64(le.Uint32(payload[4:]))
	if n > TextsRecordPages {
		return nil, textsRefusal("%d pages, at most %d", n, TextsRecordPages)
	}
	at := TextsRecordHeadSize
	if need := n * 8; need > int64(total-at) {
		return nil, textsRefusal("%d pages need %d bytes, %d remain", n, need, total-at)
	}
	pages := make(map[string][]byte, n)
	word := func(what string) (int, error) {
		if total-at < 4 {
			return 0, textsRefusal("%s needs 4 bytes, %d remain", what, total-at)
		}
		v := int64(le.Uint32(payload[at:]))
		at += 4
		return int(min(v, int64(total)+1)), nil
	}
	last := ""
	for i := range n {
		l, err := word("a page name's length")
		if err != nil {
			return nil, err
		}
		if l == 0 || l > TextsRecordName || l > total-at {
			return nil, textsRefusal("page %d's name is %d bytes; %d remain, at most %d", i, l, total-at, TextsRecordName)
		}
		name := string(payload[at : at+l])
		at += l
		if err := checkTextsName(name); err != nil {
			return nil, err
		}
		if i > 0 && name <= last {
			return nil, textsRefusal("page %q is out of name order or named twice", name)
		}
		last = name
		t, err := word("a page's length")
		if err != nil {
			return nil, err
		}
		if t > TextsRecordText || t > total-at {
			return nil, textsRefusal("page %q is %d bytes; %d remain, at most %d", name, t, total-at, TextsRecordText)
		}
		pages[name] = bytes.Clone(payload[at : at+t])
		at += t
	}
	if at != total {
		return nil, textsRefusal("%d byte(s) follow the last page", total-at)
	}
	return pages, nil
}

// FrameTextsRecord is pages as one record of the stream: the record header,
// then the payload EncodeTextsRecord writes.
func FrameTextsRecord(pages map[string][]byte) ([]byte, error) {
	payload, err := EncodeTextsRecord(pages)
	if err != nil {
		return nil, err
	}
	return frameRecord(TextsRecordTypeID, payload), nil
}

// AppendTextsRecord returns stream with pages as one more record after its
// last one, the file header's record count raised by one. stream must be a
// map Open accepts that carries the script record and no texts record: the
// texts ride only in an extended map.
func AppendTextsRecord(stream []byte, pages map[string][]byte) ([]byte, error) {
	m, err := Open(stream)
	if err != nil {
		return nil, err
	}
	if !m.HasScriptRecord() {
		return nil, textsRefusal("the map carries no script record; texts ride only in an extended map")
	}
	if m.textsRecords > 0 {
		return nil, textsRefusal("the map already carries one")
	}
	framed, err := FrameTextsRecord(pages)
	if err != nil {
		return nil, err
	}
	return appendRecord(stream, m, framed), nil
}

// WithoutTextsRecord returns stream with every texts record taken out and
// the file header's record count lowered to match, and whether it carried
// one. A stream the record walk refuses comes back as it is.
func WithoutTextsRecord(stream []byte) ([]byte, bool) {
	f, err := walkRecords(stream)
	if err != nil || f.textsRecords == 0 {
		return stream, false
	}
	out := make([]byte, 0, len(stream))
	out = append(out, stream[:FileHeaderSize]...)
	at := FileHeaderSize
	for _, rec := range f.records {
		end := at + RecordHeaderSize + int(rec.PayloadSize)
		if rec.TypeID != TextsRecordTypeID {
			out = append(out, stream[at:end]...)
		}
		at = end
	}
	binary.LittleEndian.PutUint32(out[FileRecordCount:], f.fh.recordCount-uint32(f.textsRecords))
	return append(out, stream[at:]...), true
}

// TextsOf is the pages the texts record of stream holds, and whether stream
// carries one. A stream the record walk refuses carries none; two texts
// records, or one DecodeTextsRecord refuses, is an error.
func TextsOf(stream []byte) (map[string][]byte, bool, error) {
	f, err := walkRecords(stream)
	if err != nil || f.textsRecords == 0 {
		return nil, false, nil
	}
	if f.textsRecords > 1 {
		return nil, true, textsRefusal("the map carries %d of them", f.textsRecords)
	}
	pages, err := DecodeTextsRecord(f.textsRecord)
	return pages, true, err
}

// HasTextsRecord reports whether the map carries the Againrom texts record.
func (m *Map) HasTextsRecord() bool { return m.textsRecords > 0 }
