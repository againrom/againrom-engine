package sav

import (
	"encoding/binary"
	"fmt"
)

const maxArchiveOutput = 32 << 20

// archiveWriter runs the inverse of programmes. It emits object tags from its
// own traversal; Record.Off, End and Index are never inputs. RefSlots must be
// complete: the compact Refs projection cannot recover nullable list order.
// This is a wire component, not an admission or live-world export policy.
type archiveWriter struct {
	b       []byte
	next    uint16
	classes map[string]uint16
	objects map[*Record]uint16
	err     error
}

func newArchiveWriter() *archiveWriter {
	return &archiveWriter{next: 1, classes: map[string]uint16{}, objects: map[*Record]uint16{}}
}

func serializeArchiveReferences(roots []*Record) ([]byte, error) {
	if len(roots) > maxListElements {
		return nil, fmt.Errorf("sav: too many archive roots")
	}
	w := newArchiveWriter()
	for _, root := range roots {
		if err := w.reference(root, 0); err != nil {
			return nil, err
		}
	}
	return w.b, nil
}

func (w *archiveWriter) put(b []byte) {
	if w.err != nil {
		return
	}
	if len(b) > maxArchiveOutput-len(w.b) {
		w.err = fmt.Errorf("sav: archive output exceeds %d bytes", maxArchiveOutput)
		return
	}
	w.b = append(w.b, b...)
}

func (w *archiveWriter) word(v uint16)  { w.put(binary.LittleEndian.AppendUint16(nil, v)) }
func (w *archiveWriter) dword(v uint32) { w.put(binary.LittleEndian.AppendUint32(nil, v)) }

func (w *archiveWriter) takeIndex() (uint16, error) {
	if w.next == 0 || w.next&instanceTagBit != 0 {
		return 0, fmt.Errorf("sav: archive index space exhausted")
	}
	index := w.next
	w.next++
	return index, nil
}

func (w *archiveWriter) reference(r *Record, depth int) error {
	if w.err != nil {
		return w.err
	}
	if depth > maxWalkDepth {
		return fmt.Errorf("sav: archive nesting past %d", maxWalkDepth)
	}
	if r == nil {
		w.word(0)
		return w.err
	}
	if index, ok := w.objects[r]; ok {
		w.word(index)
		return w.err
	}
	prog, ok := programmes[r.Class]
	if !ok || len(r.Class) == 0 || len(r.Class) > maxClassName {
		return fmt.Errorf("sav: no archive programme for class %q", r.Class)
	}
	if index, ok := w.classes[r.Class]; ok {
		w.word(instanceTagBit | index)
	} else {
		index, err := w.takeIndex()
		if err != nil {
			return err
		}
		w.classes[r.Class] = index
		w.word(classIntro)
		w.word(1)
		w.word(uint16(len(r.Class)))
		w.put([]byte(r.Class))
	}
	index, err := w.takeIndex()
	if err != nil {
		return err
	}
	// Publish before nested references: a backreference can name its owner.
	w.objects[r] = index
	return w.programme(prog, r, depth)
}

func (w *archiveWriter) programme(prog []step, r *Record, depth int) error {
	for _, s := range prog {
		if err := w.step(s, r, depth); err != nil {
			return fmt.Errorf("sav: %s.%s: %w", r.Class, s.name, err)
		}
		if w.err != nil {
			return w.err
		}
	}
	return nil
}

func (w *archiveWriter) refs(r *Record, name string, n, depth int) error {
	refs := r.RefSlots[name]
	if n < 0 || n > maxListElements || len(refs) != n {
		return fmt.Errorf("complete reference slots %d do not match count %d", len(refs), n)
	}
	for _, ref := range refs {
		if err := w.reference(ref, depth+1); err != nil {
			return err
		}
	}
	return w.err
}

func archiveCount(r *Record, name string) (int, error) {
	n, ok := r.Counts[name]
	if !ok || n < 0 || n > maxListElements {
		return 0, fmt.Errorf("missing or invalid count for %s", name)
	}
	return n, nil
}

func (w *archiveWriter) value(r *Record, name string) error {
	v, ok := r.Value[name]
	if !ok {
		return fmt.Errorf("missing scalar %s", name)
	}
	w.dword(v)
	return w.err
}

func (w *archiveWriter) step(s step, r *Record, depth int) error {
	switch s.op {
	case stpRun:
		// The straight-run encoder already owns the Player-only XOR/clamp.
		// Unlike its old decode/reencode use, a live producer can exceed a
		// plain field's width; reject rather than silently truncating it.
		for _, m := range s.members {
			v := r.Value[m.Name]
			if (m.Kind == KindU8 && v > 0xff) || (m.Kind == KindU16 && v > 0xffff) {
				return fmt.Errorf("%s value %d exceeds its field width", m.Name, v)
			}
		}
		b, err := (Class{Name: r.Class, Members: s.members}).encode(Fields{Value: r.Value, Text: r.Text, Raw: r.Raw})
		if err != nil {
			return err
		}
		w.put(b)
	case stpClass:
		prog, ok := programmes[s.class]
		if !ok {
			return fmt.Errorf("unknown base %q", s.class)
		}
		return w.programme(prog, r, depth)
	case stpInline:
		refs := r.Refs[s.name]
		if len(refs) != 1 || refs[0] == nil || refs[0].Class != s.class {
			return fmt.Errorf("inline %s needs one %s", s.name, s.class)
		}
		return w.programme(programmes[s.class], refs[0], depth)
	case stpObjRef:
		return w.refs(r, s.name, 1, depth)
	case stpRefRun:
		return w.refs(r, s.name, s.n, depth)
	case stpList, stpContainer:
		n, err := archiveCount(r, s.name)
		if err != nil {
			return err
		}
		w.dword(uint32(n))
		if err := w.refs(r, s.name, n, depth); err != nil {
			return err
		}
		if s.op == stpContainer {
			for _, suffix := range []string{"1C", "20"} {
				if err := w.value(r, s.name+suffix); err != nil {
					return err
				}
			}
		}
	case stpU16List, stpDWordArray, stpWordArray, stpRawArray:
		n, err := archiveCount(r, s.name)
		if err != nil {
			return err
		}
		width := 2
		if s.op == stpDWordArray {
			width = 4
		} else if s.op == stpRawArray {
			width = s.n
		}
		b, ok := r.Raw[s.name]
		if width <= 0 || !ok || len(b) != n*width {
			return fmt.Errorf("array has %d bytes, want %d elements of width %d", len(b), n, width)
		}
		count, err := cityAppendCount(nil, n)
		if err != nil {
			return err
		}
		w.put(count)
		w.put(b)
	case stpFlagged:
		flag, ok := r.Value[s.name]
		if !ok || flag > 1 {
			return fmt.Errorf("missing or invalid presence flag")
		}
		w.put([]byte{byte(flag)})
		if flag == 1 {
			return w.programme(s.sub, r, depth)
		}
	case stpSpellbook:
		n, err := archiveCount(r, s.name)
		if err != nil {
			return err
		}
		if err := w.value(r, s.name+"Header"); err != nil {
			return err
		}
		w.dword(uint32(n))
		return w.refs(r, s.name, max(0, n-1), depth)
	case stpGroups:
		n, err := archiveCount(r, s.name)
		if err != nil {
			return err
		}
		if len(r.Groups) != n {
			return fmt.Errorf("Group count does not match records")
		}
		w.dword(uint32(n))
		for _, group := range r.Groups {
			if group == nil || group.Class != "Group" {
				return fmt.Errorf("invalid inline Group")
			}
			if err := w.programme(groupProgramme, group, depth); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown archive operation %d", s.op)
	}
	return w.err
}

// detachArchiveRecords owns every scalar/array/reference and preserves aliases.
// Source locations and archive indices disappear; only the field graph remains.
func detachArchiveRecords(roots []*Record) []*Record {
	owned, _ := detachArchiveRecordsWithOrigins(roots)
	return owned
}

// Capture the original tagged record identity while assigning its detached
// pointer, never by joining saved address keys or guessing shared indices.
// Inline records have Index zero and are deliberately absent from this map.
func detachArchiveRecordsWithOrigins(roots []*Record) ([]*Record, map[*Record]uint16) {
	seen := map[*Record]*Record{}
	origins := map[*Record]uint16{}
	var clone func(*Record) *Record
	var refs func([]*Record) []*Record
	refs = func(in []*Record) []*Record {
		out := make([]*Record, len(in))
		for i, r := range in {
			out[i] = clone(r)
		}
		return out
	}
	clone = func(r *Record) *Record {
		if r == nil {
			return nil
		}
		if out, ok := seen[r]; ok {
			return out
		}
		out := newRecord(r.Class, 0, 0)
		seen[r] = out
		if r.Index != 0 {
			origins[out] = r.Index
		}
		for k, v := range r.Value {
			out.Value[k] = v
		}
		for k, v := range r.Text {
			out.Text[k] = v
		}
		for k, v := range r.Raw {
			out.Raw[k] = append([]byte(nil), v...)
		}
		for k, v := range r.Counts {
			out.Counts[k] = v
		}
		for k, v := range r.Refs {
			out.Refs[k] = refs(v)
		}
		for k, v := range r.RefSlots {
			out.RefSlots[k] = refs(v)
		}
		out.Groups = refs(r.Groups)
		out.SpellSlots = refs(r.SpellSlots)
		out.WornSlots = refs(r.WornSlots)
		return out
	}
	return refs(roots), origins
}
