package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// archiveDocument is the complete decoded wire document, independent of File
// and its source bytes. Its graph is NOT proof that gameplay restored every
// object. The live-world producer must replace mutable state before export.
// SAV-DOC-053's amended top-level order, SAV-TERRKEY-056 and SAV-SESS-031
// govern this envelope. The container's state store/campaign follow separately.
type archiveDocument struct {
	head           Head
	players, dead  []*Record
	world          *archiveWorld
	marker, global uint32
	trailer        TrailerBody
	// prof is the game whose programmes and widths the document follows;
	// nil is the first game's.
	prof *Profile
	// Import-only pointer join, captured by the same graph clone which erases
	// source indices. Neither the archive writer nor persistence DTO emits it.
	origins map[*Record]uint16
}

func (d *archiveDocument) profile() *Profile {
	if d == nil || d.prof == nil {
		return FirstGame
	}
	return d.prof
}

// TrailerBody is world+0x118's 400-byte block (SAV-646, SAV-790): a
// heap-allocated, exclusively-owned raw block transferred whole by one
// non-virtual call site, unconditionally on load regardless of the stream's
// marker/global arm (SAV-790, SAV-TRAIL-026). Of its 100 dwords, a complete
// static-instruction-form search finds exactly two with a located consumer
// (SAV-791, Medium — a consumer reached only through an argument or an alias
// would not be found): TurnTracing (dword 0) and ScriptTracing (dword 1) are
// paired debug-trace verbosity toggles, each read-branch-flip-log against its
// own confirmation string (SAV-647, amended; SAV-694), with no gameplay
// meaning and no located producer in this engine. Residual is dwords 2..99,
// carried opaque; no consumer is asserted for them and none is invented here.
type TrailerBody struct {
	TurnTracing, ScriptTracing uint32
	Residual                   [98]uint32
}

// trailerBodyToArray and trailerBodyFromArray convert between TrailerBody and
// DocumentData's own flat [100]uint32 wire field, which stays an array (not
// TrailerBody itself) because gob distinguishes the two encodings and a
// frozen native envelope this project already wrote used the array form.
func trailerBodyToArray(t TrailerBody) [100]uint32 {
	var out [100]uint32
	out[0], out[1] = t.TurnTracing, t.ScriptTracing
	copy(out[2:], t.Residual[:])
	return out
}

func trailerBodyFromArray(a [100]uint32) TrailerBody {
	var t TrailerBody
	t.TurnTracing, t.ScriptTracing = a[0], a[1]
	copy(t.Residual[:], a[2:])
	return t
}

// readTrailerBody consumes the block's 100 dwords in file order, whatever the
// caller already decided about the preceding marker/global arm (SAV-790: the
// block itself is unconditional).
func (w *walker) readTrailerBody() (TrailerBody, error) {
	var t TrailerBody
	var err error
	if t.TurnTracing, err = w.u32(); err != nil {
		return TrailerBody{}, err
	}
	if t.ScriptTracing, err = w.u32(); err != nil {
		return TrailerBody{}, err
	}
	for i := range t.Residual {
		if t.Residual[i], err = w.u32(); err != nil {
			return TrailerBody{}, err
		}
	}
	return t, nil
}

type archiveWorld struct {
	buildings, effects, sacks []*Record
	// extra is the profile's Token list after the Sacks.
	extra           []*Record
	blocks          []BlockRecord
	cells           []archiveCell
	terrainIdentity uint32
	session         archiveSession
	// cellRest is each entry's bytes past the profile's cell prefix, entry
	// after entry; wideResults and wideDiplomacy the session fields the first
	// game's typed block has no room for (Profile.wideResults).
	cellRest      []byte
	wideResults   []int32
	wideDiplomacy []byte
}

// narrowCell reads one profile entry through the first game's cell fields up
// to the profile's prefix; the bytes after it are returned whole.
func narrowCell(p *Profile, rec []byte) (archiveCell, []byte, error) {
	var full [cellRecLen]byte
	n := 2 + p.cellPrefix()
	if len(rec) != p.cellLen {
		return archiveCell{}, nil, fmt.Errorf("sav: cell entry has %d bytes, want %d", len(rec), p.cellLen)
	}
	copy(full[:], rec[:n])
	var c archiveCell
	if err := binary.Read(bytes.NewReader(full[:]), binary.LittleEndian, &c); err != nil {
		return archiveCell{}, nil, err
	}
	return c, rec[n:], nil
}

// widenCell is narrowCell's inverse. A first-game field past the prefix
// must be zero: the profile's entry has no such field.
func widenCell(p *Profile, c archiveCell, rest []byte) ([]byte, error) {
	b, err := binary.Append(nil, binary.LittleEndian, c)
	if err != nil {
		return nil, err
	}
	n := 2 + p.cellPrefix()
	if len(rest) != p.cellRest() {
		return nil, fmt.Errorf("sav: cell rest has %d bytes, want %d", len(rest), p.cellRest())
	}
	for _, v := range b[n:] {
		if v != 0 {
			return nil, fmt.Errorf("sav: a %s cell entry has no first-game trigger tail", p.Magic)
		}
	}
	return append(b[:n], rest...), nil
}

// narrowSession reads a profile's session block into the first game's
// typed block, the result slots past its hundred and, row-major, the matrix
// entries outside its 50x50 top-left (DIV-4170).
func narrowSession(p *Profile, raw []byte) (archiveSession, []int32, []byte, error) {
	l := p.session
	if len(raw) != p.sessionLen {
		return archiveSession{}, nil, nil, fmt.Errorf("sav: session has %d bytes, want %d", len(raw), p.sessionLen)
	}
	first := make([]byte, 0, sessionLen)
	first = append(first, raw[l.results:l.results+4*triggerResultCount]...)
	first = append(first, raw[l.latches:l.diplomacy]...)
	var results []int32
	for i := triggerResultCount; i < l.resultCount; i++ {
		results = append(results, int32(u32(raw, l.results+4*i)))
	}
	var diplomacy []byte
	side := l.diplomacySide
	for r := range side {
		row := raw[l.diplomacy+r*side : l.diplomacy+(r+1)*side]
		if r < diplomacySide {
			first = append(first, row[:diplomacySide]...)
			diplomacy = append(diplomacy, row[diplomacySide:]...)
		} else {
			diplomacy = append(diplomacy, row...)
		}
	}
	first = append(first, raw[l.diplomacy+side*side:]...)
	if len(first) != sessionLen {
		return archiveSession{}, nil, nil, fmt.Errorf("sav: the %s session layout does not narrow", p.Magic)
	}
	var s archiveSession
	if err := binary.Read(bytes.NewReader(first), binary.LittleEndian, &s); err != nil {
		return archiveSession{}, nil, nil, err
	}
	return s, results, diplomacy, nil
}

// widenSession is narrowSession's inverse.
func widenSession(p *Profile, s archiveSession, results []int32, diplomacy []byte) ([]byte, error) {
	l := p.session
	if len(results) != p.wideResults() || len(diplomacy) != p.wideDiplomacy() {
		return nil, fmt.Errorf("sav: session holds %d wide result slots and %d wide stances, want %d and %d",
			len(results), len(diplomacy), p.wideResults(), p.wideDiplomacy())
	}
	first, err := binary.Append(nil, binary.LittleEndian, s)
	if err != nil {
		return nil, err
	}
	out := make([]byte, p.sessionLen)
	copy(out[l.results:], first[:4*triggerResultCount])
	for i, v := range results {
		put32(out, l.results+4*(triggerResultCount+i), uint32(v))
	}
	copy(out[l.latches:l.diplomacy], first[4*triggerResultCount:sessDiplomacy])
	side, at := l.diplomacySide, 0
	for r := range side {
		row := out[l.diplomacy+r*side : l.diplomacy+(r+1)*side]
		if r < diplomacySide {
			copy(row, first[sessDiplomacy+r*diplomacySide:sessDiplomacy+(r+1)*diplomacySide])
			at += copy(row[diplomacySide:], diplomacy[at:])
		} else {
			at += copy(row, diplomacy[at:])
		}
	}
	copy(out[l.diplomacy+side*side:], first[sessDiplomacy+diplomacySide*diplomacySide:])
	return out, nil
}

// archiveCell names every member of SAV-CELLLOAD-111's 52-byte payload. The
// ten keys are still numeric wire keys, not native IDs or repaired pointers.
// Record order matters: duplicates and zero-valued overlays are not collapsed.
type archiveCell struct {
	Cell                                  uint16
	Cost, Static, LayerCount, Residue03   uint8
	GroundActor, AirActor, Building, Sack uint32
	Layers                                [6]uint32
	Operation, Power, SourceX, SourceY    uint8
	TargetX, TargetY                      uint8
	Residue32                             uint16
}

// Unnamed session regions retain their measured widths and source names. They
// have no invented semantic/default interpretation. Native current-state
// producers for these regions remain necessary; retaining an imported region
// alone does not establish LOAD-to-current-SAVE fidelity.
type archiveSession struct {
	Results          [100]int32
	Latches          [1000]uint8
	Raw08            [48]uint8
	RawA828          [400]uint8
	DiplomacyHeader  [8]uint8
	Diplomacy        [50][50]uint8
	FlagA48, FlagA49 uint8
	ValueA4C, Won    uint32
	ValueB3B0, Lost  uint32
}

func parseArchiveDocument(body []byte) (*archiveDocument, error) {
	return parseArchiveDocumentFor(body, FirstGame)
}

// parseArchiveDocumentFor reads body with the profile's programmes and widths.
func parseArchiveDocumentFor(body []byte, prof *Profile) (*archiveDocument, error) {
	if len(body) > maxArchiveOutput {
		return nil, fmt.Errorf("sav: decoded archive exceeds %d bytes", maxArchiveOutput)
	}
	if prof == nil {
		prof = FirstGame
	}
	f := &File{Body: body, Profile: prof}
	if err := f.readHead(); err != nil {
		return nil, err
	}
	w := &walker{b: body, p: f.Head.End, next: 1, classes: map[uint16]string{}, objects: map[uint16]*Record{}, retainGraph: true, prof: prof}
	d := &archiveDocument{head: f.Head, prof: prof}
	var err error
	d.players, err = w.completeRoots("Players", f.Head.PlayerCount, "Player", true)
	if err != nil {
		return nil, err
	}
	d.dead, err = w.completeCountedRoots("dead actors", "Unit")
	if err != nil {
		return nil, err
	}
	if w.p >= len(body) {
		return nil, fmt.Errorf("sav: archive lacks world selector")
	}
	present := body[w.p] != 0
	w.p++
	if present {
		world := &archiveWorld{}
		d.world = world
		world.buildings, err = w.completeCountedRoots("Buildings", "Building")
		if err != nil {
			return nil, err
		}
		world.effects, err = w.completeCountedRoots("SpellEffects", "SpellEffect")
		if err != nil {
			return nil, err
		}
		half, err := w.worldHalf()
		if err != nil {
			return nil, err
		}
		world.blocks = half.Blocks
		world.cells = make([]archiveCell, half.CellRecCount)
		for i := range world.cells {
			off := half.CellRecDataOff + i*prof.cellLen
			cell, rest, err := narrowCell(prof, body[off:off+prof.cellLen])
			if err != nil {
				return nil, fmt.Errorf("sav: cell %d: %w", i, err)
			}
			world.cells[i] = cell
			world.cellRest = append(world.cellRest, rest...)
		}
		world.terrainIdentity = u32(body, half.SessionOff-4)
		world.session, world.wideResults, world.wideDiplomacy, err = narrowSession(prof, body[half.SessionOff:half.SessionOff+prof.sessionLen])
		if err != nil {
			return nil, fmt.Errorf("sav: session: %w", err)
		}
		world.sacks, err = w.completeCountedRoots("Sacks", "Sack")
		if err != nil {
			return nil, err
		}
		if prof.extraList {
			if world.extra, err = w.completeCountedRoots("extra Tokens", "Token"); err != nil {
				return nil, err
			}
		}
	}
	d.marker, err = w.u32()
	if err != nil {
		return nil, err
	}
	if d.marker == 0xbadface1 {
		d.global, err = w.u32()
		if err != nil {
			return nil, err
		}
	}
	d.trailer, err = w.readTrailerBody()
	if err != nil {
		return nil, err
	}
	if len(body) != w.p+(w.p&1) {
		return nil, fmt.Errorf("sav: archive ends at %d, decoded size %d", w.p, len(body))
	}
	// Source coordinates are deliberately absent from the detached document.
	d.head.MapNameOff, d.head.MissionOff, d.head.DifficultyOff, d.head.End = 0, 0, 0, 0
	d.detachRoots()
	return d, nil
}

// Unlike playerRecords, this preserves every Player reference slot, including
// nulls and repeated objects. All five roots share one archive state.
func (w *walker) completeRoots(name string, n uint32, class string, nullable bool) ([]*Record, error) {
	if n > maxListElements || uint64(n) > uint64((len(w.b)-w.p)/2) {
		return nil, fmt.Errorf("sav: %s count %d exceeds bounded remaining data", name, n)
	}
	out := make([]*Record, int(n))
	for i := range out {
		r, err := w.object(0)
		if err != nil {
			return nil, fmt.Errorf("sav: %s %d: %w", name, i, err)
		}
		if (r == nil && !nullable) || (r != nil && !groundClass(r.Class, class)) {
			return nil, fmt.Errorf("sav: %s %d is not an admitted %s reference", name, i, class)
		}
		out[i] = r
	}
	return out, nil
}

func (w *walker) completeCountedRoots(name, class string) ([]*Record, error) {
	n, err := w.u32()
	if err != nil {
		return nil, err
	}
	return w.completeRoots(name, n, class, false)
}

func (d *archiveDocument) rootLists() []*[]*Record {
	lists := []*[]*Record{&d.players, &d.dead}
	if d.world != nil {
		lists = append(lists, &d.world.buildings, &d.world.effects, &d.world.sacks)
		if d.profile().extraList {
			lists = append(lists, &d.world.extra)
		}
	}
	return lists
}

func (d *archiveDocument) detachRoots() {
	var roots []*Record
	lists := d.rootLists()
	for _, list := range lists {
		roots = append(roots, (*list)...)
	}
	owned, origins := detachArchiveRecordsWithOrigins(roots)
	d.origins = origins
	at := 0
	for _, list := range lists {
		n := len(*list)
		*list = owned[at : at+n : at+n]
		at += n
	}
}

func serializeArchiveDocument(d *archiveDocument) ([]byte, error) {
	b, _, err := writeArchiveDocument(d, false)
	return b, err
}

// serializeArchiveDocumentRows also writes Token rows above a byte, as their
// low byte, and returns them for the document's rows leaf.
func serializeArchiveDocumentRows(d *archiveDocument) ([]byte, []tokenRow, error) {
	return writeArchiveDocument(d, true)
}

func writeArchiveDocument(d *archiveDocument, carryRows bool) ([]byte, []tokenRow, error) {
	if d == nil {
		return nil, nil, fmt.Errorf("sav: missing archive document")
	}
	w := newArchiveWriter()
	w.prof, w.carryRows = d.profile(), carryRows
	b, err := w.archiveDocument(d)
	if err != nil {
		return nil, nil, err
	}
	return b, w.rows, nil
}

func (w *archiveWriter) archiveDocument(d *archiveDocument) ([]byte, error) {
	if len(d.head.MapName) >= 0xff {
		return nil, fmt.Errorf("sav: extended map CString is unsupported")
	}
	w.dword(d.head.CounterA)
	w.dword(d.head.CounterB)
	name, err := cityAppendCString(nil, d.head.MapName)
	if err != nil {
		return nil, err
	}
	w.put(name)
	for _, v := range d.head.Reserved {
		w.dword(v)
	}
	w.dword(d.head.Mission)
	w.dword(d.head.Difficulty)
	w.dword(d.head.PlayerListField)
	if err := w.rootList(d.players, "Player", true); err != nil {
		return nil, err
	}
	if err := w.rootList(d.dead, "Unit", false); err != nil {
		return nil, err
	}
	if d.world == nil {
		w.put([]byte{0})
	} else {
		w.put([]byte{1})
		if err := w.worldDocument(d.world); err != nil {
			return nil, err
		}
	}
	w.dword(d.marker)
	if d.marker == 0xbadface1 {
		w.dword(d.global)
	} else if d.global != 0 {
		return nil, fmt.Errorf("sav: trailer global has no marker arm")
	}
	w.dword(d.trailer.TurnTracing)
	w.dword(d.trailer.ScriptTracing)
	for _, v := range d.trailer.Residual {
		w.dword(v)
	}
	if len(w.b)&1 != 0 {
		w.put([]byte{0}) // SAV-DECPAD-238: new alignment, not source provenance.
	}
	if w.err != nil {
		return nil, w.err
	}
	return w.b, nil
}

func (w *archiveWriter) rootList(roots []*Record, class string, nullable bool) error {
	if len(roots) > maxListElements {
		return fmt.Errorf("sav: too many %s roots", class)
	}
	w.dword(uint32(len(roots)))
	for i, r := range roots {
		if (r == nil && !nullable) || (r != nil && !groundClass(r.Class, class)) {
			return fmt.Errorf("sav: %s root %d has invalid class or null", class, i)
		}
		if err := w.reference(r, 0); err != nil {
			return err
		}
	}
	return w.err
}

func (w *archiveWriter) countedLength(n int) error {
	if n < 0 || n > maxListElements {
		return fmt.Errorf("sav: archive array length %d exceeds bound", n)
	}
	b, err := cityAppendCount(nil, n)
	if err != nil {
		return err
	}
	w.put(b)
	return w.err
}

func (w *archiveWriter) worldDocument(world *archiveWorld) error {
	if err := w.rootList(world.buildings, "Building", false); err != nil {
		return err
	}
	if err := w.rootList(world.effects, "SpellEffect", false); err != nil {
		return err
	}
	if err := w.countedLength(len(world.blocks)); err != nil {
		return err
	}
	for i, block := range world.blocks {
		if i > 0 && block.Cell <= world.blocks[i-1].Cell {
			return fmt.Errorf("sav: block keys are not strictly increasing at %d", i)
		}
		w.dword(uint32(block.Cell)<<16 | uint32(block.Dyn)<<8 | uint32(block.Static))
	}
	if err := w.countedLength(len(world.cells)); err != nil {
		return err
	}
	rest := w.prof.cellRest()
	if len(world.cellRest) != rest*len(world.cells) {
		return fmt.Errorf("sav: cell rest has %d bytes for %d entries", len(world.cellRest), len(world.cells))
	}
	for i, cell := range world.cells {
		b, err := widenCell(w.prof, cell, world.cellRest[i*rest:(i+1)*rest])
		if err != nil {
			return fmt.Errorf("sav: cell %d: %w", i, err)
		}
		w.put(b)
	}
	w.dword(world.terrainIdentity)
	b, err := widenSession(w.prof, world.session, world.wideResults, world.wideDiplomacy)
	if err != nil {
		return err
	}
	w.put(b)
	if err := w.rootList(world.sacks, "Sack", false); err != nil {
		return err
	}
	if w.prof.extraList {
		return w.rootList(world.extra, "Token", false)
	} else if len(world.extra) != 0 {
		return fmt.Errorf("sav: a %s document has no Token list after the Sacks", w.prof.Magic)
	}
	return nil
}
