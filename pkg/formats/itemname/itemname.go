// Package itemname parses the shipped item-name table: main.res's
// text/itemname.bin, a packed u16 key per entry, paired positionally with
// text/itemname.txt, one line per entry (research ITEM-DISPNAME-036,
// ITEM-NAMEKEY-037, TEXT-ITEMNAME-019).
//
// A LINE IS STORED AS THE SHIPPED BYTES, UNCHANGED. text/itemname.txt is
// not a code page this package converts: pkg/render/text's Font.index
// selects a glyph record from EACH BYTE of a drawn string and applies the
// install's own language conversion there (text.Convert under
// text.SelectorConverting), so the string this package hands back has to
// carry the game's own byte arrangement, not a transcoded one. A caller
// that decoded the line first would hand the drawing path two bytes per
// converted character instead of one, and no caller here does.
//
// It is a leaf. Both byte streams are read by a caller off the container
// filesystem and handed in whole; this package opens no archive and
// imports no other againrom package.
package itemname

import (
	"bytes"
	"encoding/binary"
)

// Parse builds the shipped table from bin's key array and txt's line array.
// Entry i's key is the little-endian u16 at bin[2i : 2i+2]; entry i's name
// is line i of txt, split over CRLF or bare LF, stored as that line's own
// bytes with no code-page conversion applied (package doc: the drawing
// path converts, this package does not). A trailing newline in txt adds no
// final line, the same rule pkg/data's ParseBodyList applies to the
// shipped body list.
//
// THE TWO LENGTHS NEED NOT AGREE: only as many entries as bin and txt both
// carry are walked, the shorter of the two bounding the loop, so a short
// read of either file loses entries rather than reading past the other.
//
// AN EMPTY LINE IS DROPPED rather than stored as its key's name: a map
// lookup cannot tell a present empty string from an absent key, and an
// empty display name is never what a caller wants in place of its own
// fallback.
func Parse(bin, txt []byte) map[uint16]string {
	lines := splitLines(txt)
	n := len(bin) / 2
	if len(lines) < n {
		n = len(lines)
	}
	out := make(map[uint16]string, n)
	for i := 0; i < n; i++ {
		if s := lines[i]; s != "" {
			out[binary.LittleEndian.Uint16(bin[i*2:i*2+2])] = s
		}
	}
	return out
}

// splitLines is txt split into lines, one per shipped row, over CRLF or
// bare LF, each line's bytes carried into its string unchanged (package
// doc).
func splitLines(txt []byte) []string {
	parts := bytes.Split(txt, []byte("\n"))
	if n := len(parts); n > 0 && len(parts[n-1]) == 0 {
		parts = parts[:n-1]
	}
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = string(bytes.TrimSuffix(p, []byte("\r")))
	}
	return out
}
