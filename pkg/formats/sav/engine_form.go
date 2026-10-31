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
	// dropClasses are the classes the profile has no record of.
	dropClasses []string
	// pointer is the class a Building the World holds as a Pointer takes in
	// the profile's records, and the word that class adds.
	pointer formRetype
	// terrain is the profile's key list and entry map rule; nil keeps the
	// engine's own.
	terrain *terrainForm
	// results and diplomacyHead are session values a rule or the session
	// constructor gives: a result slot past the hundred by index, and the
	// diplomacy block's first bytes.
	results       map[int]int32
	diplomacyHead []byte
}

// terrainForm is how the profile's terrain serializer writes the key list
// and the entry map (R2-SESSION-019, R2-SESSION-310, R2-SESSION-311).
type terrainForm struct {
	// A cell is listed when from <= cell < to and its plane-two byte
	// exceeds above.
	from, to uint16
	above    uint8
	// buckets is the entry map's hash size: entries are written bucket by
	// bucket, (key >> 4) mod buckets.
	buckets int
	// keyAt is where in an entry's rest the entry's own cell key lies.
	keyAt int
}

// formRetype writes a record of class from, named in a ProfileKeeps, as
// class to with one more value, word.
type formRetype struct{ from, to, word string }

// ProfileKeeps are values a profile's records hold that the engine's
// records keep in the World: the +0x6c word of each Building the second
// game holds as a Pointer, by Token Identity (R2-ENGINE-603).
type ProfileKeeps struct {
	Pointers map[uint32]uint32
}

// retype writes each record keeps names as the Pointer class with its word.
func (f formRetype) retype(out *DocumentData, keeps map[uint32]uint32) error {
	if len(keeps) == 0 {
		return nil
	}
	if f.from == "" {
		return fmt.Errorf("sav: the profile has no Pointer record")
	}
	found := 0
	for i := range out.Objects {
		r := &out.Objects[i]
		at := slices.IndexFunc(r.Values, func(v DocumentValueData) bool { return v.Name == "Identity" })
		if r.Class != f.from || at < 0 {
			continue
		}
		word, ok := keeps[r.Values[at].Value]
		if !ok {
			continue
		}
		found++
		r.Class = f.to
		r.Values = append(r.Values, DocumentValueData{Name: f.word, Value: word})
		slices.SortFunc(r.Values, func(a, b DocumentValueData) int { return strings.Compare(a.Name, b.Name) })
	}
	if found != len(keeps) {
		return fmt.Errorf("sav: %d Pointers name %d Building records", len(keeps), found)
	}
	return nil
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
// (StoreEngineDocument). A world half is written in p's widths
// (worldForm); keeps gives the records p holds what the World keeps.
func DocumentInProfile(data DocumentData, p *Profile, keeps ProfileKeeps) (DocumentData, bool, error) {
	if p == nil || p.form == nil {
		return data, false, nil
	}
	if data.Game != "" || data.SecondTail != nil {
		return DocumentData{}, false, fmt.Errorf("sav: the document is already in a profile's records")
	}
	out, err := CloneDocumentData(data)
	if err != nil {
		return DocumentData{}, false, err
	}
	if out.World != nil {
		p.worldForm(out.World)
	}
	for i := range out.Objects {
		if _, ok := p.programmes[out.Objects[i].Class]; !ok && !slices.Contains(p.form.dropClasses, out.Objects[i].Class) {
			return DocumentData{}, false, fmt.Errorf("sav: a %s record has no %s form", out.Objects[i].Class, p.Game)
		}
		out.Objects[i] = p.form.record(out.Objects[i])
	}
	if err := p.form.pointer.retype(&out, keeps.Pointers); err != nil {
		return DocumentData{}, false, err
	}
	dropClassRecords(&out, p.form.dropClasses)
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

// dropClassRecords removes every record of the named classes, whose sites
// the form has dropped, and renumbers the references to the rest.
func dropClassRecords(data *DocumentData, classes []string) {
	if len(classes) == 0 {
		return
	}
	next := make([]uint16, len(data.Objects)+1)
	kept := data.Objects[:0]
	for i, r := range data.Objects {
		if !slices.Contains(classes, r.Class) {
			kept = append(kept, r)
			next[i+1] = uint16(len(kept))
		}
	}
	data.Objects = kept
	ids := func(list []uint16) {
		for i, id := range list {
			list[i] = next[id]
		}
	}
	var record func(r *DocumentRecordData)
	record = func(r *DocumentRecordData) {
		for i := range r.RefSlots {
			ids(r.RefSlots[i].Objects)
		}
		for i := range r.Inline {
			record(&r.Inline[i].Record)
		}
		for i := range r.Groups {
			record(&r.Groups[i])
		}
	}
	for i := range data.Objects {
		record(&data.Objects[i])
	}
	ids(data.Players)
	ids(data.DeadActors)
	if w := data.World; w != nil {
		for _, list := range [][]uint16{w.Buildings, w.Effects, w.Sacks, w.Extra} {
			ids(list)
		}
	}
}

// worldForm writes a world half in p's widths. An engine document holds
// no terrain entry bytes past the first game's cell fields, no result slot
// past the hundred and no stance outside 50x50: the profile's entries get
// zero there, and its cells no first-game trigger tail (DIV-4350).
func (p *Profile) worldForm(w *DocumentWorldData) {
	for i := range w.Cells {
		c := &w.Cells[i]
		c.Operation, c.Power, c.SourceX, c.SourceY, c.TargetX, c.TargetY, c.Residue32 = 0, 0, 0, 0, 0, 0, 0
	}
	if n := p.cellRest() * len(w.Cells); len(w.CellRest) != n {
		w.CellRest = make([]byte, n)
	}
	if len(w.SessionResults) != p.wideResults() {
		w.SessionResults = make([]int32, p.wideResults())
	}
	if len(w.SessionDiplomacy) != p.wideDiplomacy() {
		w.SessionDiplomacy = make([]byte, p.wideDiplomacy())
	}
	for slot, v := range p.form.results {
		w.SessionResults[slot-triggerResultCount] = v
	}
	copy(w.Session.DiplomacyHeader[:], p.form.diplomacyHead)
	if t := p.form.terrain; t != nil {
		t.write(w, p.cellRest())
	}
}

// write lists the cells the rule keeps, stores each entry's own key and
// orders the entries by bucket. The order inside a bucket is not claimed:
// the key falls inside it (DIV-4351).
func (t *terrainForm) write(w *DocumentWorldData, rest int) {
	w.Blocks = slices.DeleteFunc(w.Blocks, func(b BlockRecord) bool {
		return b.Cell < t.from || b.Cell >= t.to || b.Dyn <= t.above
	})
	type entry struct {
		cell DocumentCellData
		rest []byte
	}
	entries := make([]entry, len(w.Cells))
	for i, c := range w.Cells {
		r := slices.Clone(w.CellRest[i*rest : (i+1)*rest])
		r[t.keyAt], r[t.keyAt+1] = byte(c.Cell), byte(c.Cell>>8)
		entries[i] = entry{c, r}
	}
	bucket := func(key uint16) int { return int(key>>4) % t.buckets }
	slices.SortStableFunc(entries, func(a, b entry) int {
		if d := bucket(a.cell.Cell) - bucket(b.cell.Cell); d != 0 {
			return d
		}
		return int(b.cell.Cell) - int(a.cell.Cell)
	})
	w.CellRest = w.CellRest[:0]
	for i, e := range entries {
		w.Cells[i] = e.cell
		w.CellRest = append(w.CellRest, e.rest...)
	}
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
