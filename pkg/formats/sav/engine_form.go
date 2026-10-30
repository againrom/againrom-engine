package sav

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

// documentForm is how a SAVE writes an engine document in a profile's
// records: the trailer's two words and the field changes, record by record.
type documentForm struct {
	marker, global uint32
	fields         []formField
	// emptyArrays are the state store's array leaves the profile writes
	// with no element.
	emptyArrays []string
}

type formSite uint8

const (
	// formRaw drops a raw field and writes each of into in its place.
	formRaw formSite = iota + 1
	// formInline drops an inline record and its count.
	formInline
	// formSlot drops a reference slot.
	formSlot
	// formWord keeps the low 16 bits of a value.
	formWord
)

type formField struct {
	site formSite
	name string
	into []formAdd
}

// formAdd is a raw field the profile writes, filled from the engine
// field it replaces.
type formAdd struct {
	name string
	fill func(engine []byte) []byte
}

func zeroFill(n int) func([]byte) []byte {
	return func([]byte) []byte { return make([]byte, n) }
}

// preferenceFill is the second game's 36 preference bytes: the engine's 32,
// whose byte 0x1f is the formation, then 0xff and three zero bytes, the
// values of every EN original (DIV-4305).
func preferenceFill(engine []byte) []byte {
	out := make([]byte, 36)
	copy(out, engine)
	out[32] = 0xff
	return out
}

// DocumentInProfile writes an engine document in p's records, the document
// the original of p's game reads. It answers false, and data unchanged, when
// p's records are the engine's own. The Againrom supplement leaves are left
// out: the engine's own document travels whole beside it
// (StoreEngineDocument). A document with a world half has no second-game
// form yet (DIV-3079).
func DocumentInProfile(data DocumentData, p *Profile) (DocumentData, bool, error) {
	if p == nil || p.form == nil {
		return data, false, nil
	}
	if data.Game != "" || data.SecondTail != nil {
		return DocumentData{}, false, fmt.Errorf("sav: the document is already in a profile's records")
	}
	if data.World != nil {
		return DocumentData{}, false, fmt.Errorf("sav: a document with a world half has no %s form", p.Game)
	}
	out, err := CloneDocumentData(data)
	if err != nil {
		return DocumentData{}, false, err
	}
	for i := range out.Objects {
		if _, ok := p.programmes[out.Objects[i].Class]; !ok {
			return DocumentData{}, false, fmt.Errorf("sav: a %s record has no %s form", out.Objects[i].Class, p.Game)
		}
		out.Objects[i] = p.form.record(out.Objects[i])
	}
	out.Game = p.Game
	out.Campaign = CityCampaignData{}
	out.Marker, out.GlobalDWord = p.form.marker, p.form.global
	for _, d := range supplementLeaves {
		d.remove(&out.State)
	}
	for i, r := range out.State.ValueRecords {
		for _, path := range p.form.emptyArrays {
			if r.Value.Kind == 6 && StateRecordIs(out.State.RootKind, out.State.DirectoryRecords, r.Path, path) {
				out.State.ValueRecords[i].Value.Bytes = nil
			}
		}
	}
	return out, true, nil
}

// record applies every field change to r; group records hold no field a
// form changes.
func (f *documentForm) record(r DocumentRecordData) DocumentRecordData {
	for _, field := range f.fields {
		switch field.site {
		case formRaw:
			at := slices.IndexFunc(r.Raw, func(v DocumentRawData) bool { return v.Name == field.name })
			if at < 0 {
				continue
			}
			engine := r.Raw[at].Bytes
			r.Raw = slices.Delete(r.Raw, at, at+1)
			for _, add := range field.into {
				r.Raw = append(r.Raw, DocumentRawData{Name: add.name, Bytes: add.fill(engine)})
			}
			slices.SortFunc(r.Raw, func(a, b DocumentRawData) int { return strings.Compare(a.Name, b.Name) })
		case formInline:
			if slices.ContainsFunc(r.Inline, func(v DocumentInlineData) bool { return v.Name == field.name }) {
				r.Inline = slices.DeleteFunc(r.Inline, func(v DocumentInlineData) bool { return v.Name == field.name })
				r.Counts = slices.DeleteFunc(r.Counts, func(v DocumentCountData) bool { return v.Name == field.name })
			}
		case formSlot:
			r.RefSlots = slices.DeleteFunc(r.RefSlots, func(v DocumentRefsData) bool { return v.Name == field.name })
		case formWord:
			for i := range r.Values {
				if r.Values[i].Name == field.name {
					r.Values[i].Value &= 0xffff
				}
			}
		}
	}
	return r
}

// StoreEngineDocument is a profile document holding the engine's own encoded
// document in its AgainromForm leaf, which EngineDocument answers on LOAD
// (DIV-4304). EncodeDocumentData encodes it.
func StoreEngineDocument(data DocumentData, engine []byte) (DocumentData, error) {
	if p, ok := ProfileOf(engine); !ok || p != FirstGame {
		return DocumentData{}, fmt.Errorf("sav: the engine document is not an Asg& document")
	}
	if data.Game == "" {
		return DocumentData{}, fmt.Errorf("sav: a profile document needs a profile")
	}
	state := data.State
	if err := nativeFormLeaf.storePayload(&state, engine); err != nil {
		return DocumentData{}, err
	}
	data.State = state
	return data, nil
}

// EngineDocument is the engine document a profile file carries in its
// AgainromForm leaf, and saved itself for any other file. A file with no
// value record at the leaf's path is the original's. A leaf that does not
// validate, or does not hold an Asg& document, refuses.
func EngineDocument(saved []byte) ([]byte, error) {
	if p, ok := ProfileOf(saved); !ok || p.form == nil {
		return saved, nil
	}
	doc, err := DecodeDocumentData(saved)
	if err != nil {
		if f, openErr := Open(saved); openErr == nil && f.carriesLeaf(nativeFormLeaf) {
			return nil, fmt.Errorf("sav: the engine document leaf does not decode: %w", err)
		}
		return saved, nil
	}
	engine, found, err := nativeFormLeaf.lookupPayload(doc.State)
	if err != nil {
		return nil, err
	}
	if !found {
		return saved, nil
	}
	if p, ok := ProfileOf(engine); !ok || p != FirstGame {
		return nil, fmt.Errorf("sav: the engine document leaf holds no Asg& document")
	}
	return bytes.Clone(engine), nil
}
