package sav

import (
	"fmt"
	"slices"
)

// NewDocumentRecord allocates the grammar of a new semantic object. It does
// not supply gameplay defaults: scalar/raw zeroes are explicit uninitialised
// constructor fields which the caller must fill from its construction policy.
// Flags name the optional programmes to include. The ordinary Document encoder
// remains the only serializer, and object references remain DTO-local indices.
func NewDocumentRecord(class string, flags ...string) (DocumentRecordData, error) {
	prog, ok := programmes[class]
	if !ok {
		return DocumentRecordData{}, fmt.Errorf("sav: no constructor grammar for %q", class)
	}
	out := DocumentRecordData{Class: class}
	used := make(map[string]bool)
	var walk func([]step) error
	walk = func(steps []step) error {
		for _, s := range steps {
			switch s.op {
			case stpRun:
				for _, m := range s.members {
					switch m.Kind {
					case KindCString:
						out.Texts = append(out.Texts, DocumentTextData{Name: m.Name})
					case KindRaw:
						out.Raw = append(out.Raw, DocumentRawData{Name: m.Name, Bytes: make([]byte, m.Len)})
					default:
						out.Values = append(out.Values, DocumentValueData{Name: m.Name})
					}
				}
			case stpClass:
				if err := walk(programmes[s.class]); err != nil {
					return err
				}
			case stpFlagged:
				v := uint32(0)
				if slices.Contains(flags, s.name) {
					v, used[s.name] = 1, true
					if err := walk(s.sub); err != nil {
						return err
					}
				}
				out.Values = append(out.Values, DocumentValueData{Name: s.name, Value: v})
			case stpObjRef:
				out.RefSlots = append(out.RefSlots, DocumentRefsData{Name: s.name, Objects: make([]uint16, 1)})
			case stpList, stpContainer, stpSpellbook, stpRefRun:
				n := 0
				if s.op == stpRefRun {
					n = s.n
				}
				count := n
				if s.op == stpSpellbook {
					count = 1
					out.Values = append(out.Values, DocumentValueData{Name: s.name + "Header"})
				}
				if s.op == stpContainer {
					out.Values = append(out.Values, DocumentValueData{Name: s.name + "1C"}, DocumentValueData{Name: s.name + "20"})
				}
				out.Counts = append(out.Counts, DocumentCountData{Name: s.name, Count: uint32(count)})
				out.RefSlots = append(out.RefSlots, DocumentRefsData{Name: s.name, Objects: make([]uint16, n)})
			case stpU16List, stpDWordArray, stpWordArray, stpRawArray:
				out.Counts = append(out.Counts, DocumentCountData{Name: s.name})
				out.Raw = append(out.Raw, DocumentRawData{Name: s.name})
			case stpGroups:
				out.Counts = append(out.Counts, DocumentCountData{Name: s.name}, DocumentCountData{Name: "Actors"})
			case stpInline:
				r, err := NewDocumentRecord(s.class)
				if err != nil {
					return err
				}
				out.Counts = append(out.Counts, DocumentCountData{Name: s.name, Count: 1})
				out.Inline = append(out.Inline, DocumentInlineData{Name: s.name, Record: r})
			default:
				return fmt.Errorf("sav: unsupported constructor operation %d", s.op)
			}
		}
		return nil
	}
	if err := walk(prog); err != nil {
		return DocumentRecordData{}, err
	}
	for _, flag := range flags {
		if !used[flag] {
			return DocumentRecordData{}, fmt.Errorf("sav: inactive constructor flag %q", flag)
		}
	}
	slices.SortFunc(out.Values, func(a, b DocumentValueData) int { return compareDocumentName(a.Name, b.Name) })
	slices.SortFunc(out.Texts, func(a, b DocumentTextData) int { return compareDocumentName(a.Name, b.Name) })
	slices.SortFunc(out.Raw, func(a, b DocumentRawData) int { return compareDocumentName(a.Name, b.Name) })
	slices.SortFunc(out.Counts, func(a, b DocumentCountData) int { return compareDocumentName(a.Name, b.Name) })
	slices.SortFunc(out.RefSlots, func(a, b DocumentRefsData) int { return compareDocumentName(a.Name, b.Name) })
	slices.SortFunc(out.Inline, func(a, b DocumentInlineData) int { return compareDocumentName(a.Name, b.Name) })
	return out, nil
}

func compareDocumentName(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
