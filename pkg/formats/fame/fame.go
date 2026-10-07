// Package fame reads the installed hall of fame without changing record order.
package fame

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// Record preserves the declared name span and both uninterpreted trailing words.
// Name is the prefix before the first NUL, in the installation's code page.
type Record struct {
	Name     string
	NameSpan []byte
	Score    int32
	Tail     [2]uint32
}

// Marshal writes the CString name prefix and three words (FAME-WRITE-007).
// Read spans normalize to Name plus one NUL, as the original writer does;
// signed score bits, both tail words and stored order remain unchanged.
func Marshal(records []Record) ([]byte, error) {
	if len(records) > 4096 {
		return nil, fmt.Errorf("FAME: too many records")
	}
	b := binary.LittleEndian.AppendUint32(nil, uint32(len(records)))
	for i, r := range records {
		if len(r.Name) > 1023 || strings.IndexByte(r.Name, 0) >= 0 {
			return nil, fmt.Errorf("FAME: invalid name at record %d", i)
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(len(r.Name)+1))
		b = append(b, r.Name...)
		b = append(b, 0)
		b = binary.LittleEndian.AppendUint32(b, uint32(r.Score))
		b = binary.LittleEndian.AppendUint32(b, r.Tail[0])
		b = binary.LittleEndian.AppendUint32(b, r.Tail[1])
	}
	return b, nil
}

// Insert follows FAME-INSERT-011: the new row precedes the first signed score
// less than or equal to it, then the resulting tail is trimmed to limit.
// Existing disorder and duplicate names are retained; no full sort occurs.
func Insert(records []Record, record Record, limit int) ([]Record, error) {
	if len(records) > 4096 || limit < 0 || limit > 4096 {
		return nil, fmt.Errorf("FAME: invalid table limit")
	}
	at := len(records)
	for i, r := range records {
		if record.Score >= r.Score {
			at = i
			break
		}
	}
	out := make([]Record, 0, len(records)+1)
	out = append(out, records[:at]...)
	out = append(out, record)
	out = append(out, records[at:]...)
	if len(out) > limit {
		out = out[:limit]
	}
	for i := range out {
		out[i].NameSpan = bytes.Clone(out[i].NameSpan)
	}
	return out, nil
}

// Parse bounds counts and name spans before slicing. Its finite admission policy
// is stricter than the original unbounded name buffer (FAME-STRING-010).
// Trailing input is ignored, as in FAME-READER-009. No sorting or trimming occurs.
func Parse(b []byte) ([]Record, error) {
	if len(b) < 4 {
		return nil, fmt.Errorf("FAME: truncated count")
	}
	n := uint64(binary.LittleEndian.Uint32(b))
	if n > 4096 || n > uint64((len(b)-4)/17) {
		return nil, fmt.Errorf("FAME: invalid record count %d", n)
	}
	b = b[4:]
	out := make([]Record, 0, n)
	for i := uint64(0); i < n; i++ {
		if len(b) < 4 {
			return nil, fmt.Errorf("FAME: truncated record %d", i)
		}
		span := uint64(binary.LittleEndian.Uint32(b))
		b = b[4:]
		if span < 1 || span > 1024 || span+12 > uint64(len(b)) {
			return nil, fmt.Errorf("FAME: invalid name span at record %d", i)
		}
		name := b[:span]
		end := bytes.IndexByte(name, 0)
		if end < 0 {
			return nil, fmt.Errorf("FAME: unterminated name at record %d", i)
		}
		b = b[span:]
		out = append(out, Record{Name: string(name[:end]), NameSpan: bytes.Clone(name),
			Score: int32(binary.LittleEndian.Uint32(b)), Tail: [2]uint32{binary.LittleEndian.Uint32(b[4:]), binary.LittleEndian.Uint32(b[8:])}})
		b = b[12:]
	}
	return out, nil
}
