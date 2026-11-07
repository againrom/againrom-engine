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
	// sightBit is the bit of the matrix cell M[owner][viewer] that lets
	// viewer see by owner's bodies; zero where the profile stores none.
	sightBit uint8
	// order is the profile's rule for a Unit record's attack order; nil
	// writes the engine's order code unchanged.
	order *formOrder
}

// formOrder writes a Unit record's attack order with its target: an
// engine order attack whose command key at target names another record of
// the document writes that key into the profile's command block at
// target; any other attack order writes order idle with state and phase
// 0. A Unit record is the one that holds the code, state, phase and from
// raw fields; into is the command block the profile writes.
type formOrder struct {
	code, state, phase, from, into string
	attack, idle                   uint32
	target                         int
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
// game holds as a Pointer, by Token Identity (R2-ENGINE-603), and the
// sight its session matrix holds.
type ProfileKeeps struct {
	Pointers map[uint32]uint32
	// Commands is each second-game Unit command's blow record and stamp,
	// by the record's Token Identity.
	Commands map[uint32]CommandStrike
	// Sees answers whether Player slot viewer sees by the bodies of slot
	// owner, the World's fog; nil sees nothing.
	Sees func(viewer, owner uint32) bool
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

// commandWord is word +0x16 of the second game's Unit command, 65500 in
// every original-made record (R2-SESSION-339) and the constructor's value
// (R2-ENGINE-846); writeCommandStrikes writes an order's stamp over it
// (DIV-4479).
const commandWord = 65500

// commandFill is the second game's 184-byte Unit command: word +0, the
// command post the Group's mode-1 arm moves to (R2-ENGINE-824), is the
// engine order's post, its word +0; word +0x0a, the goal the walk stores
// and the Move programme walks a member to (R2-ENGINE-844, AI-MOVE-023),
// is the engine order's goal; word +0x16 is commandWord; the alarm
// +0x54 and the notice +0x58..+0x5a are the engine order's (R2-ENGINE-845);
// the three creature spell slots are the engine order's, in the second
// game's window; every other byte is zero (DIV-4305, DIV-4490).
// writeCommandStrikes writes the blow record and the stamp over it.
func commandFill(engine []byte) []byte {
	out := make([]byte, secondCreatureSpells.width)
	copy(out[:2], engine)
	if len(engine) >= commandGoal+2 {
		copy(out[commandGoal:commandGoal+2], engine[commandGoal:])
	}
	put16(out, 0x16, commandWord)
	if len(engine) > commandNoticeEnd {
		copy(out[commandAlarm:commandNoticeEnd], engine[commandAlarm:commandNoticeEnd])
	}
	if slots, ok := firstCreatureSpells.slots(engine); ok {
		secondCreatureSpells.put(out, slots)
	}
	return out
}

const (
	// commandGoal is the walk's goal cell word, at the same place in both
	// games' blocks.
	commandGoal = 0x0a
	// commandAlarm..commandNoticeEnd are the alarm +0x54 and the notice
	// cell and count +0x58..+0x5a, at the same place in both games' blocks.
	commandAlarm     = 0x54
	commandNoticeEnd = 0x5b
)

// CommandStrike is the blow record and pursuit stamp of a second-game Unit
// command (R2-ENGINE-845, R2-ENGINE-846): the stamp +0x16, the striker's
// record key +0x94, the fine clock at the blow +0x98 and the cell +0x9c.
type CommandStrike struct {
	Stamp   uint16
	Striker uint32
	Clock   uint32
	Away    uint16
}

// readCommandStrike is b's blow record and stamp; false for a block that is
// not the second game's command.
func readCommandStrike(b []byte) (CommandStrike, bool) {
	if len(b) != secondCreatureSpells.width {
		return CommandStrike{}, false
	}
	return CommandStrike{Stamp: u16(b, 0x16), Striker: u32(b, 0x94), Clock: u32(b, 0x98), Away: u16(b, 0x9c)}, true
}

// writeCommandStrikes writes each keeps entry into the command block of
// the record its Token Identity names. A record without the block keeps
// what it holds.
func writeCommandStrikes(out *DocumentData, keeps map[uint32]CommandStrike) {
	if len(keeps) == 0 {
		return
	}
	for i := range out.Objects {
		r := &out.Objects[i]
		at := slices.IndexFunc(r.Values, func(v DocumentValueData) bool { return v.Name == "Identity" })
		if at < 0 {
			continue
		}
		k, ok := keeps[r.Values[at].Value]
		if !ok {
			continue
		}
		for j := range r.Raw {
			if b := r.Raw[j].Bytes; r.Raw[j].Name == secondCreatureSpells.block && len(b) == secondCreatureSpells.width {
				put16(b, 0x16, k.Stamp)
				put32(b, 0x94, k.Striker)
				put32(b, 0x98, k.Clock)
				put16(b, 0x9c, k.Away)
			}
		}
	}
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
		p.form.sight(&out, keeps.Sees)
	}
	held := map[uint32]bool{}
	for _, r := range out.Objects {
		if at := slices.IndexFunc(r.Values, func(v DocumentValueData) bool { return v.Name == "Identity" }); at >= 0 && !slices.Contains(p.form.dropClasses, r.Class) {
			held[r.Values[at].Value] = true
		}
	}
	for i := range out.Objects {
		if _, ok := p.programmes[out.Objects[i].Class]; !ok && !slices.Contains(p.form.dropClasses, out.Objects[i].Class) {
			return DocumentData{}, false, fmt.Errorf("sav: a %s record has no %s form", out.Objects[i].Class, p.Game)
		}
		out.Objects[i] = p.form.record(out.Objects[i], held)
	}
	if err := p.form.pointer.retype(&out, keeps.Pointers); err != nil {
		return DocumentData{}, false, err
	}
	writeCommandStrikes(&out, keeps.Commands)
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

// sight ORs the form's sight bit into M[owner][viewer] of the session
// matrix for each pair of the document's Player slots in which sees lets
// viewer see by owner's bodies. A slot past the 50x50 the World's relation
// holds has no cell there.
func (f *documentForm) sight(data *DocumentData, sees func(viewer, owner uint32) bool) {
	if f.sightBit == 0 || sees == nil || data.World == nil {
		return
	}
	var slots []uint32
	for _, p := range data.Players {
		if p == 0 || int(p) > len(data.Objects) {
			continue
		}
		for _, v := range data.Objects[p-1].Values {
			if v.Name == "Slot" && v.Value < diplomacySide {
				slots = append(slots, v.Value)
			}
		}
	}
	for _, owner := range slots {
		for _, viewer := range slots {
			if sees(viewer, owner) {
				data.World.Session.Diplomacy[owner][viewer] |= f.sightBit
			}
		}
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

// record applies every field change to r and the attack order rule, whose
// targets are the Token Identities held; group records hold no field a
// form changes.
func (f *documentForm) record(r DocumentRecordData, held map[uint32]bool) DocumentRecordData {
	attack, key := f.order.read(r)
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
	f.order.write(&r, attack, key, held)
	return r
}

// raw is r's raw field name when it is width bytes or more.
func (o *formOrder) raw(r DocumentRecordData, name string, width int) []byte {
	at := slices.IndexFunc(r.Raw, func(v DocumentRawData) bool { return v.Name == name })
	if at < 0 || len(r.Raw[at].Bytes) < width {
		return nil
	}
	return r.Raw[at].Bytes
}

// read reports whether r is a Unit record under the engine's attack order,
// and the key its engine command holds at target.
func (o *formOrder) read(r DocumentRecordData) (bool, uint32) {
	if o == nil {
		return false, 0
	}
	code, from := o.raw(r, o.code, 4), o.raw(r, o.from, o.target+4)
	if code == nil || from == nil || o.raw(r, o.state, 4) == nil || o.raw(r, o.phase, 4) == nil || u32(code, 0) != o.attack {
		return false, 0
	}
	return true, u32(from, o.target)
}

// write gives the profile's record of an attack order its target key, or
// the idle order when the key names no other record held.
func (o *formOrder) write(r *DocumentRecordData, attack bool, key uint32, held map[uint32]bool) {
	if !attack {
		return
	}
	self := slices.IndexFunc(r.Values, func(v DocumentValueData) bool { return v.Name == "Identity" })
	if into := o.raw(*r, o.into, o.target+4); key != 0 && held[key] && (self < 0 || r.Values[self].Value != key) && into != nil {
		put32(into, o.target, key)
		return
	}
	put32(o.raw(*r, o.code, 4), 0, o.idle)
	put32(o.raw(*r, o.state, 4), 0, 0)
	put32(o.raw(*r, o.phase, 4), 0, 0)
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
