//go:build sessioncorpusaudit

// Package game original-SAV byte-identity census (docs/1197/story.md).
//
// THE QUESTION THIS ANSWERS. pipeline/SAV-COMPLETION.md's milestone 2 compares
// thirteen structure rows raw-file-to-live over the discovered corpus, each row
// carrying its own named boundary, and says in as many words that a green run
// "does not prove that every required field or live consumer has an oracle".
// The instrument that looks like it closes that gap does not:
// savroundtrip1195_corpus_test.go builds `want` from before.Snapshot after
// restoring the original file and `got` from after.Snapshot after restoring our
// own re-export, so BOTH SIDES PASS THROUGH OUR READER. A field the reader
// silently drops is absent from both and its census reports zero mismatches.
// Its 94-of-102 with zero mismatches proves that we read the same file the same
// way twice, which is a real regression bound and is not a completeness claim.
//
// pkg/formats/sav's own TestDocument1115LawfulCorpusAudit (build tag
// savdocumentaudit) has the same shape one layer down: it parses, serializes,
// re-parses and re-serializes, and asserts the SECOND encoding equals the first.
// It calls clear(raw) immediately after parsing, so the one comparison that
// would settle completeness -- our bytes against the bytes we read -- is not
// merely absent, it is made impossible. That test measures stability. This one
// measures identity.
//
// WHAT THIS DOES. For every file the milestone2 corpus discovers: decode it to
// the intermediate the reader produces (sav.DecodeDocumentData, whose DTO owns
// no File.Body, Store, TailRest, offset or compressed input -- save_document.go),
// emit a whole container back out of that intermediate with no intended change
// (sav.EncodeDocumentData), and compare the produced bytes against the file they
// came from. NO LIVE WORLD IS INVOLVED on either side: restoring into a Mission
// and re-deriving a file from game state would launder the bytes through exactly
// the reader whose completeness is in question.
//
// EVERY BYTE DIFFERENCE IS CLASSIFIED AND NONE IS DISMISSED. Each differing
// range is reported with its offset, its length, the decoded structure that
// covers it, and one of four classes:
//
//   - "intended"           a difference this project's own code makes on
//     purpose and names at the point it makes it
//   - "modelled-reemitted" a field we do model, written back in a different
//     place or order, with the information preserved --
//     and the preservation is PROVEN, not assumed, by an
//     independent decoder (see sav1197RegDigest)
//   - "unmodelled"         a byte present in the file and absent from the
//     intermediate: information we do not carry
//   - "unknown"            anything no rule below explains, kept with its
//     offset and FAILING its file's subtest
//
// The rule set is deliberately built so that a NEW loss cannot be absorbed by an
// existing rule: every "modelled-reemitted" rule in the state store requires the
// independent registry digest to be equal on both sides, so a store value that
// actually went missing breaks the digest, disqualifies those rules and lands in
// "unknown". An unknown count of zero is the real result; an unknown count that
// was never measured is worthless.
//
// OWNER PRIORITY (recorded here because it orders the census, not only the
// prose). Owner direction: which missions are already completed is not
// important; the campaign POSITION -- which mission the game is on now -- and
// the state of the Valuable Documents journal are. The census therefore reports
// the campaign numbers before anything else, and reports the journal TWICE, on
// purpose:
//
//   - SAV-BYTEID-1197-DOCUMENT-VALUES compares the decoded collection element
//     by element. A seat hotfix added the same comparison to
//     savroundtrip1195_corpus_test.go, where it found nothing.
//   - SAV-BYTEID-1197-DOCUMENT-BYTES locates the document run by its OWN wire
//     encoding inside the source's campaign record and compares that span, at
//     that offset, byte for byte. A value comparison can pass while the bytes
//     differ -- an equal-valued list rebuilt somewhere else in the record --
//     and that is precisely the quiet difference nobody has enumerated.
//
// SAV-BYTEID-1197-CAMPAIGN-TRAILER classifies the 32 bytes between the run and
// the record's end, which pkg/game/save.go's own note calls unattributed.
//
// THE INDEPENDENCE RULE, in this file's own terms. The state store is compared
// two ways. Positionally, byte against byte. Structurally, through
// pkg/formats/reg -- the SEPARATE .reg parser, which shares the store's
// 0x31415926 signature and none of pkg/formats/sav's own parseStateStore code.
// A store rewritten with a value silently dropped would agree with itself
// through sav's parser; it cannot agree with reg's.
//
//	go test -tags sessioncorpusaudit -count=1 ./pkg/game/ -run TestSAVByteIdentity1197 -v
//
// AGAINROM_SAVE_CORPUS names gameversions/saves and AGAINROM_ASSETS names the EN
// or RU lawful root (milestone2Corpus requires it even though this census never
// resumes a game). scripts/check-milestone2-acceptance.sh runs it with its own
// defaults and prints its SAV-BYTEID- lines even on a passing run.
// AGAINROM_SAV1197_RANGES=all prints every individual range instead of one
// summary line per (region, rule); unknown ranges are always printed in full.
//
// NO OWNER SAVE NAME REACHES A TRACKED FILE. Every committed string here is a
// region, structure or rule name; the corpus-relative path stays in memory and
// reaches t.Logf only, on savroundtrip1195_corpus_test.go's own terms.
package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"testing"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
)

// The four classifications. See the file doc comment for what each means.
const (
	sav1197Intended   = "intended"
	sav1197ReEmitted  = "modelled-reemitted"
	sav1197Unmodelled = "unmodelled"
	sav1197Unknown    = "unknown"
)

// sav1197Range is one differing byte range: where it is, how long it is, which
// decoded structure covers it, and why it differs. structure is "" where no
// decoded structure covers the offset, which is itself reportable.
type sav1197Range struct {
	region    string
	off, n    int
	structure string
	class     string
	rule      string
}

// sav1197Result is one file's census row.
//
// documents and docBytes are deliberately SEPARATE. documents is a VALUE
// comparison of the decoded Valuable Documents collection; docBytes is a BYTE
// comparison of the located document run itself, at its own offset inside the
// campaign record. A value comparison passing while the bytes differ would mean
// we rebuilt an equal-valued list somewhere else in the record, which is exactly
// the kind of quiet difference this census exists to name, so the two numbers
// are reported apart and their agreement is stated rather than assumed.
type sav1197Result struct {
	identical bool
	ranges    []sav1197Range
	notes     []string
	campaign  bool // the whole campaign record region re-emitted byte-identically
	documents bool // the decoded document collection compares equal by value
	position  bool // the campaign position re-emitted identically

	docLocated  bool // the document run was uniquely located in the source record
	docBytes    bool // that located span re-emitted byte-identically, at its offset
	tailLocated bool // the 32-byte record trailer after the run was located
	tailBytes   bool // that trailer re-emitted byte-identically
	tailHead    bool // its first dword is the SelectedMission this reader decodes
}

// sav1197Diffs coalesces every differing byte of a and b into ranges over their
// common prefix, and appends one final range for a length difference. Offsets
// are in the SOURCE's coordinate system, which is the only one a reader of this
// census can check against the file on disk.
func sav1197Diffs(a, b []byte) [][2]int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var out [][2]int
	for i := 0; i < n; {
		if a[i] == b[i] {
			i++
			continue
		}
		j := i
		for j < n && a[j] != b[j] {
			j++
		}
		out = append(out, [2]int{i, j - i})
		i = j
	}
	if len(a) != len(b) {
		longer := len(a)
		if len(b) > longer {
			longer = len(b)
		}
		out = append(out, [2]int{n, longer - n})
	}
	return out
}

// sav1197StoreRec is one node record of the embedded state store's own on-disk
// table, read here with this file's own arithmetic rather than through sav's
// decoded cityState -- the independence rule in the file doc comment.
type sav1197StoreRec struct {
	reserved, value, size, kind uint32
	name                        string
	nul                         int // index of the NUL inside the 16-byte name field, or -1
}

// sav1197StoreFrame splits a state store into its record table and its value
// pool. It reports false for a stream that does not frame, which is not an
// error: a file whose tail is not a store has none.
func sav1197StoreFrame(b []byte) (recs []sav1197StoreRec, poolOff int, ok bool) {
	if len(b) < 28 || string(b[:4]) != "&YA1" {
		return nil, 0, false
	}
	count := int(binary.LittleEndian.Uint32(b[16:20]))
	if count < 0 || count > (len(b)-28)/32 {
		return nil, 0, false
	}
	end := 24 + 32*count
	if end+4 > len(b) {
		return nil, 0, false
	}
	recs = make([]sav1197StoreRec, count)
	for i := range recs {
		off := 24 + 32*i
		field := b[off+16 : off+32]
		nul := bytes.IndexByte(field, 0)
		name := string(field)
		if nul >= 0 {
			name = string(field[:nul])
		}
		recs[i] = sav1197StoreRec{
			reserved: binary.LittleEndian.Uint32(b[off : off+4]),
			value:    binary.LittleEndian.Uint32(b[off+4 : off+8]),
			size:     binary.LittleEndian.Uint32(b[off+8 : off+12]),
			kind:     binary.LittleEndian.Uint32(b[off+12 : off+16]),
			name:     name, nul: nul,
		}
	}
	return recs, end + 4, true
}

// sav1197StorePaths gives every record its registry path, so two slots are
// compared by what they ARE and not by what they are called. The store is two
// levels deep: a root directory record owns the contiguous child range its value
// and size words declare. A record no directory claims gets "", which compares
// unequal to every real path and so can never make two slots look alike.
//
// THIS IS NOT COSMETIC. The shipped world store carries TWO distinct nodes named
// "IsOpen", under Inventory and under SpellBook. Matching slots by name alone
// made the writer's reordering of those two look like one node whose value
// changed, which is the difference between "rewritten in canonical order" and
// "a value the player loses".
func sav1197StorePaths(recs []sav1197StoreRec) []string {
	paths := make([]string, len(recs))
	for i, r := range recs {
		if r.kind&1 == 0 {
			continue
		}
		paths[i] = "/" + r.name
	}
	for i, r := range recs {
		if r.kind&1 == 0 || paths[i] == "" {
			continue
		}
		for k := int(r.value); k < int(r.value)+int(r.size) && k >= 0 && k < len(recs); k++ {
			if recs[k].kind&1 == 0 && paths[k] == "" {
				paths[k] = paths[i] + "/" + recs[k].name
			}
		}
	}
	return paths
}

// sav1197RegDigest decodes a state store with pkg/formats/reg -- a parser
// independent of sav's own -- and returns one sorted line per node: its path,
// its raw kind word and its value. Two stores with equal digests carry the same
// registry content whatever order their records are written in. A stream reg
// refuses yields a single PARSE-ERR line, which compares unequal to anything
// else and so cannot silently pass as agreement.
func sav1197RegDigest(b []byte) []string {
	r, err := reg.Parse(b)
	if err != nil {
		return []string{"PARSE-ERR: " + err.Error()}
	}
	var out []string
	var walk func(n *reg.Node, prefix string)
	walk = func(n *reg.Node, prefix string) {
		for _, c := range n.Children {
			path := prefix + "/" + c.Name
			switch {
			case c.Dir:
				out = append(out, fmt.Sprintf("%s DIR kind=%#x", path, c.Kind))
				walk(c, path)
			case c.Type == reg.TypeString:
				out = append(out, fmt.Sprintf("%s STR kind=%#x %q", path, c.Kind, c.Str))
			case c.Type == reg.TypeInt:
				out = append(out, fmt.Sprintf("%s INT kind=%#x %d", path, c.Kind, c.Int))
			case c.Type == reg.TypeIntArray:
				out = append(out, fmt.Sprintf("%s INTS kind=%#x %v", path, c.Kind, c.Ints))
			default:
				out = append(out, fmt.Sprintf("%s TYPE%d kind=%#x", path, c.Type, c.Kind))
			}
		}
	}
	walk(r.Root, "")
	sort.Strings(out)
	return out
}

// sav1197Landmark is one named start offset in the decompressed archive body.
type sav1197Landmark struct {
	off  int
	name string
}

// sav1197BodyMap builds the body's landmark list from the reader's OWN located
// offsets: the exact envelope walk's tagged object starts
// (DocumentObjectLocations), the world half's four structure locators, the
// common trailer block and the alignment byte after it. Attribution takes the
// greatest landmark at or before an offset, so a byte inside a nested object's
// own fields names that object rather than its owner -- stated here because it
// is a real limit of the attribution, not a claim about nesting.
func sav1197BodyMap(f *sav.File) []sav1197Landmark {
	marks := []sav1197Landmark{{0, "document head"}}
	if locs, err := f.DocumentObjectLocations(); err == nil {
		for _, l := range locs {
			marks = append(marks, sav1197Landmark{l.Off, fmt.Sprintf("object %d %s", l.ArchiveIndex, l.Class)})
		}
	}
	if w := f.World; w != nil {
		marks = append(marks,
			sav1197Landmark{w.BuildingsOff, "world Building root list"},
			sav1197Landmark{w.BlocksOff, "world terrain block plane"},
			sav1197Landmark{w.CellRecOff, "world cell records"},
			sav1197Landmark{w.SessionOff, "world session block"})
	}
	if f.TrailerOff > 0 {
		marks = append(marks,
			sav1197Landmark{f.TrailerOff, "common trailer block"},
			sav1197Landmark{f.TrailerOff + 400, "archive alignment pad"})
	}
	sort.SliceStable(marks, func(i, j int) bool { return marks[i].off < marks[j].off })
	return marks
}

// sav1197At names the landmark an offset falls at or after.
func sav1197At(marks []sav1197Landmark, off int) string {
	name := ""
	for _, m := range marks {
		if m.off > off {
			break
		}
		name = m.name
	}
	return name
}

// sav1197StoreAt names the store component an offset lands in, the index of the
// record when it lands in one, and which of that record's five fields it is.
func sav1197StoreAt(recs []sav1197StoreRec, poolOff, off int) (string, int, string) {
	switch {
	case off < 24:
		return "state store header", -1, "header"
	case off < poolOff-4:
		i := (off - 24) / 32
		within := (off - 24) % 32
		field := "name"
		switch {
		case within < 4:
			field = "reserved"
		case within < 8:
			field = "value"
		case within < 12:
			field = "size"
		case within < 16:
			field = "kind"
		}
		name := ""
		if i < len(recs) {
			name = recs[i].name
		}
		return fmt.Sprintf("state store record %d (%s) %s", i, name, field), i, field
	case off < poolOff:
		return "state store pool length", -1, "poollen"
	default:
		return fmt.Sprintf("state store pool +%d", off-poolOff), -1, "pool"
	}
}

// sav1197Inspect runs the whole census for one file. It never writes anything
// anywhere: raw is read-only input and every product stays in memory.
func sav1197Inspect(raw []byte) (sav1197Result, error) {
	var res sav1197Result

	data, err := sav.DecodeDocumentData(raw)
	if err != nil {
		return res, fmt.Errorf("DECODE: %w", err)
	}
	out, err := sav.EncodeDocumentData(data)
	if err != nil {
		return res, fmt.Errorf("ENCODE: %w", err)
	}
	// A byte-identical file still goes through the whole campaign comparison
	// below rather than being short-circuited to "everything matched": the
	// located/unlocatable counts the census reports have to be measured on the
	// same population as the differing files, or they would silently describe a
	// smaller one.
	res.identical = bytes.Equal(out, raw)

	src, err := sav.Open(raw)
	if err != nil {
		return res, fmt.Errorf("REOPEN-SOURCE: %w", err)
	}
	ours, err := sav.Open(out)
	if err != nil {
		return res, fmt.Errorf("REOPEN-OURS: %w", err)
	}

	add := func(region string, off, n int, structure, class, rule string) {
		res.ranges = append(res.ranges, sav1197Range{region, off, n, structure, class, rule})
	}

	// --- container header -------------------------------------------------
	// blobEnd (header+4) and blobBytes (header+12) are RECOMPUTED by every
	// writer in this package (container.go Marshal, save_document.go
	// serializeSaveDocument); they differ exactly when the compressed blob
	// changed length, and the rule below proves ours are the correct
	// recomputation rather than merely different.
	srcEnd, ourEnd := binary.LittleEndian.Uint32(raw[4:]), binary.LittleEndian.Uint32(out[4:])
	srcLen, ourLen := binary.LittleEndian.Uint32(raw[12:]), binary.LittleEndian.Uint32(out[12:])
	if !bytes.Equal(raw[:4], out[:4]) {
		add("header", 0, 4, "container magic", sav1197Unknown, "container magic differs")
	}
	for _, h := range []struct {
		off  int
		name string
		a, b uint32
	}{
		{4, "container blobEnd", srcEnd, ourEnd},
		{8, "container version", src.Version, ours.Version},
		{12, "container blobBytes", srcLen, ourLen},
	} {
		if h.a == h.b {
			continue
		}
		if h.off != 8 && ourEnd == 16+ourLen && int(ourEnd) <= len(out) {
			add("header", h.off, 4, h.name, sav1197Intended,
				"container blob extent recomputed for the re-emitted blob")
			continue
		}
		add("header", h.off, 4, h.name, sav1197Unknown, "container header word differs")
	}

	// --- decompressed archive body ----------------------------------------
	marks := sav1197BodyMap(src)
	bodyEqual := bytes.Equal(src.Body, ours.Body)
	for _, d := range sav1197Diffs(src.Body, ours.Body) {
		off, n := d[0], d[1]
		// SAV-DECPAD-238: serializeArchiveDocument appends one zero byte when
		// the document's own content ends on an odd length. The original's
		// byte at that position is whatever its heap held. The rule proves the
		// position really is the pad -- one byte past the 400-byte trailer and
		// the last byte of an equal-length body -- before accepting it.
		if n == 1 && len(src.Body) == len(ours.Body) && off == len(src.Body)-1 &&
			src.TrailerOff > 0 && src.TrailerOff+400 == off && ours.Body[off] == 0 {
			add("body", off, n, "archive alignment pad", sav1197Intended,
				"archive alignment pad written as zero, not carried (SAV-DECPAD-238)")
			continue
		}
		add("body", off, n, sav1197At(marks, off), sav1197Unknown, "archive body byte differs")
	}

	// --- the compressed blob itself ---------------------------------------
	// The blob is not diffed byte against byte: one changed body byte moves
	// every run boundary after it, so positional ranges inside a compressed
	// stream carry no information. The two cases that DO carry information are
	// separated here.
	switch {
	case bytes.Equal(raw[16:srcEnd], out[16:ourEnd]):
	case bodyEqual:
		add("blob", 16, int(srcEnd)-16, "transport codec stream", sav1197ReEmitted,
			"the transport codec did not re-compress an unchanged body bit-identically")
	default:
		res.notes = append(res.notes, fmt.Sprintf(
			"blob differs (%d -> %d bytes) because the decompressed body differs; the body ranges are the cause",
			srcEnd-16, ourEnd-16))
	}

	// --- label region -----------------------------------------------------
	// SAV-LABELTAIL-236: the original overwrites the 0x100-byte region in
	// place and never clears it, so a short label is followed by the tail of
	// whatever was longer before. serializeSaveDocument writes a fresh region
	// holding only the NUL-terminated name.
	srcNUL := bytes.IndexByte(src.LabelRegion, 0)
	for _, d := range sav1197Diffs(src.LabelRegion, ours.LabelRegion) {
		off, n := d[0], d[1]
		zeroed := off < len(ours.LabelRegion)
		for i := off; i < off+n && i < len(ours.LabelRegion); i++ {
			if ours.LabelRegion[i] != 0 {
				zeroed = false
			}
		}
		if srcNUL >= 0 && off > srcNUL && zeroed {
			add("label", off, n, "slot label region debris", sav1197Intended,
				"label debris after the name's NUL dropped (SAV-LABELTAIL-236)")
			continue
		}
		add("label", off, n, "slot label region", sav1197Unknown, "label region byte differs")
	}

	// --- embedded state store ---------------------------------------------
	srcRecs, srcPool, srcFramed := sav1197StoreFrame(src.Store)
	ourRecs, _, ourFramed := sav1197StoreFrame(ours.Store)
	srcPaths, ourPaths := sav1197StorePaths(srcRecs), sav1197StorePaths(ourRecs)
	srcDigest, ourDigest := sav1197RegDigest(src.Store), sav1197RegDigest(ours.Store)
	digestEqual := len(srcDigest) == len(ourDigest)
	for i := 0; digestEqual && i < len(srcDigest); i++ {
		digestEqual = srcDigest[i] == ourDigest[i]
	}
	if !digestEqual {
		// Name every registry node that changed, so a store loss says WHICH
		// value the player lost rather than only that some byte moved.
		ourSet := map[string]bool{}
		for _, line := range ourDigest {
			ourSet[line] = true
		}
		srcSet := map[string]bool{}
		for _, line := range srcDigest {
			srcSet[line] = true
			if !ourSet[line] {
				res.notes = append(res.notes, "STORE-CONTENT-LOST "+line)
			}
		}
		for _, line := range ourDigest {
			if !srcSet[line] {
				res.notes = append(res.notes, "STORE-CONTENT-INVENTED "+line)
			}
		}
	}
	for _, d := range sav1197Diffs(src.Store, ours.Store) {
		off, n := d[0], d[1]
		structure, idx, field := "state store", -1, ""
		if srcFramed {
			structure, idx, field = sav1197StoreAt(srcRecs, srcPool, off)
		}
		class, rule := sav1197Unknown, "state store byte differs"
		inSlot := srcFramed && ourFramed && idx >= 0 && idx < len(srcRecs) && idx < len(ourRecs)
		sameSlot := inSlot && srcPaths[idx] != "" && srcPaths[idx] == ourPaths[idx]
		if inSlot && srcPaths[idx] != "" {
			structure = fmt.Sprintf("state store record %d (%s) %s", idx, srcPaths[idx], field)
		}
		switch {
		case !digestEqual || !srcFramed || !ourFramed:
			// Every layout rule below asserts the content survived. It did
			// not, so none of them may absorb this range.
		case idx >= 0 && !sameSlot:
			class, rule = sav1197ReEmitted,
				"record table rewritten in the writer's canonical directory order"
		case field == "name" && srcRecs[idx].nul >= 0 && off-(24+32*idx+16) > srcRecs[idx].nul:
			zeroed := true
			for i := off; i < off+n && i < len(ours.Store); i++ {
				if ours.Store[i] != 0 {
					zeroed = false
				}
			}
			if zeroed {
				class, rule = sav1197Unmodelled,
					"node-name field debris after the NUL is not carried by the intermediate"
			}
		case field == "reserved" && srcRecs[idx].reserved != 0 && ourRecs[idx].reserved == 0:
			class, rule = sav1197Unmodelled,
				"node record reserved word is not carried by the intermediate"
		case (field == "value" || field == "size") && srcRecs[idx].kind&1 != 0:
			class, rule = sav1197ReEmitted,
				"directory child range rewritten for the canonical record order"
		case (field == "value" || field == "size") && (srcRecs[idx].kind == 0 || srcRecs[idx].kind == 6):
			class, rule = sav1197ReEmitted, "pool locator rewritten for the rebuilt value pool"
		case field == "pool":
			class, rule = sav1197ReEmitted, "value pool rebuilt in the writer's canonical order"
		}
		add("store", off, n, structure, class, rule)
	}

	// --- campaign record --------------------------------------------------
	res.campaign = bytes.Equal(src.TailRest, ours.TailRest)
	for _, d := range sav1197Diffs(src.TailRest, ours.TailRest) {
		add("campaign", d[0], d[1], "campaign record", sav1197Unknown, "campaign record byte differs")
	}
	res.documents, res.position = sav1197CompareCampaign(src, ours, &res)
	return res, nil
}

// sav1197CompareCampaign reads both sides' decoded CampaignProjection and
// reports, in the owner's own order of priority, whether the Valuable Documents
// journal and the campaign position survived. It names the field that moved, not
// only that a byte moved. The byte comparison above is the stronger statement
// where it holds; this one is the statement a reader can act on.
func sav1197CompareCampaign(src, ours *sav.File, res *sav1197Result) (documents, position bool) {
	a, aok, aerr := src.Campaign()
	b, bok, berr := ours.Campaign()
	if aerr != nil || berr != nil {
		res.notes = append(res.notes, fmt.Sprintf("campaign projection unreadable: source %v, ours %v", aerr, berr))
		return false, false
	}
	if !aok || !bok {
		res.notes = append(res.notes, fmt.Sprintf("campaign projection absent: source=%v ours=%v", aok, bok))
		return false, false
	}
	documents = len(a.Documents) == len(b.Documents)
	if !documents {
		res.notes = append(res.notes, fmt.Sprintf(
			"CAMPAIGN-DOCUMENTS count %d -> %d", len(a.Documents), len(b.Documents)))
	} else {
		for i := range a.Documents {
			if a.Documents[i] != b.Documents[i] {
				documents = false
				res.notes = append(res.notes, fmt.Sprintf(
					"CAMPAIGN-DOCUMENTS record %d {Value:%d Kind:%d} -> {Value:%d Kind:%d}",
					i, a.Documents[i].Value, a.Documents[i].Kind,
					b.Documents[i].Value, b.Documents[i].Kind))
			}
		}
	}
	position = a.Main.Mission == b.Main.Mission && a.Main.Announced == b.Main.Announced &&
		a.SelectedMission == b.SelectedMission && a.AutoGetMission == b.AutoGetMission &&
		a.LastMission == b.LastMission && a.MissionTime == b.MissionTime &&
		a.FirstMapPoint == b.FirstMapPoint && len(a.Children) == len(b.Children)
	for i := 0; position && i < len(a.Children); i++ {
		position = a.Children[i].Mission == b.Children[i].Mission &&
			a.Children[i].Announced == b.Children[i].Announced
	}
	if !position {
		res.notes = append(res.notes, fmt.Sprintf(
			"CAMPAIGN-POSITION main %d->%d selected %d->%d autoget %d->%d last %d->%d children %d->%d",
			a.Main.Mission, b.Main.Mission, a.SelectedMission, b.SelectedMission,
			a.AutoGetMission, b.AutoGetMission, a.LastMission, b.LastMission,
			len(a.Children), len(b.Children)))
	}
	sav1197LocateDocuments(src, ours, a, res)
	return documents, position
}

// sav1197DocumentRun is the document collection's own wire encoding: a count
// dword then that many eight-byte (Value, Kind) pairs and nothing else
// (MISSION-DOC-021, High for the record). Building it from the DECODED
// projection and finding it in the SOURCE's own campaign bytes is what makes
// the byte check independent of where our writer chose to put it.
func sav1197DocumentRun(c sav.CampaignProjection) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(c.Documents)))
	for _, d := range c.Documents {
		out = binary.LittleEndian.AppendUint32(out, d.Value)
		out = binary.LittleEndian.AppendUint32(out, d.Kind)
	}
	return out
}

// sav1197LocateDocuments answers the BYTE half of the Valuable Documents
// question: not "does an equal-valued list come back" but "do the same records
// come back at the same offset, in the same order, with the same bytes around
// them".
//
// The run is located by its own unique encoding inside the source's campaign
// record, never by an offset this file hard-codes. A collection with no
// documents encodes as four zero bytes, which is not unique and is not claimed
// to be located; those files are counted apart and named, because reporting
// them as located would be a measurement this code did not make.
//
// THE 32-BYTE TRAILER. MISSION-DOC-021 is High for the document record and only
// Medium for where the run sits in the original's file, calling "the 32-byte
// trailer between it and EOF" unattributed (pkg/game/save.go's own note on
// Snapshot.Documents). This reader does attribute it: the eight dwords after the
// run are SelectedMission, one scalar read for framing and deliberately left
// unnamed, AutoGetMission, LastMission, FirstMapPoint, MissionTime, ScoreEvents
// and the marker count. The census checks the first of those against the
// SelectedMission this reader decodes, so the attribution is measured on every
// file rather than asserted once here.
func sav1197LocateDocuments(src, ours *sav.File, c sav.CampaignProjection, res *sav1197Result) {
	if len(c.Documents) == 0 {
		res.notes = append(res.notes,
			"CAMPAIGN-DOCUMENTS the collection is empty, so its four-byte count word is not uniquely locatable and no byte result is claimed for it")
		return
	}
	run := sav1197DocumentRun(c)
	if bytes.Count(src.TailRest, run) != 1 {
		res.notes = append(res.notes, fmt.Sprintf(
			"CAMPAIGN-DOCUMENTS the %d-record run occurs %d times in the campaign record, so no byte result is claimed for it",
			len(c.Documents), bytes.Count(src.TailRest, run)))
		return
	}
	off := bytes.Index(src.TailRest, run)
	end := off + len(run)
	res.docLocated = true
	res.docBytes = end <= len(ours.TailRest) && bytes.Equal(src.TailRest[off:end], ours.TailRest[off:end])
	if !res.docBytes {
		res.notes = append(res.notes, fmt.Sprintf(
			"CAMPAIGN-DOCUMENT-BYTES the %d-record run at campaign+%d length %d did not re-emit at its own offset",
			len(c.Documents), off, len(run)))
	}
	if end+32 != len(src.TailRest) {
		res.notes = append(res.notes, fmt.Sprintf(
			"CAMPAIGN-TRAILER the document run at campaign+%d leaves %d bytes to the record's end, not 32",
			off, len(src.TailRest)-end))
		return
	}
	res.tailLocated = true
	res.tailBytes = end+32 <= len(ours.TailRest) && bytes.Equal(src.TailRest[end:end+32], ours.TailRest[end:end+32])
	res.tailHead = binary.LittleEndian.Uint32(src.TailRest[end:end+4]) == c.SelectedMission
	if !res.tailBytes || !res.tailHead {
		res.notes = append(res.notes, fmt.Sprintf(
			"CAMPAIGN-TRAILER at campaign+%d: bytes-identical=%v first-dword %d vs decoded SelectedMission %d",
			end, res.tailBytes, binary.LittleEndian.Uint32(src.TailRest[end:end+4]), c.SelectedMission))
	}
}

// sav1197Baseline is this census's committed regression bound.
//
// WHAT IT ENFORCES. identical is a FLOOR: a file that re-emits byte-identically
// today must still do so. The other five are CEILINGS of zero: no file may fail
// to decode or to re-encode, no file's campaign record region, document records
// or campaign position may stop re-emitting identically, and no differing byte
// range may go unclassified.
//
// WHAT IT CANNOT DISTINGUISH. The corpus is discovered and grows on its own
// (milestone2_acceptance_reader_test.go's own note). A genuinely new original
// save exercising a difference no rule here explains reddens this gate, and that
// is the intended direction of the error: a completeness census that quietly
// absorbs an unexplained byte is the exact failure this story exists to end. The
// per-rule range and byte counts are LOGGED AND NOT GATED, because a larger
// corpus legitimately raises them without anything having regressed.
//
// THE UPDATE PROCEDURE, stated here and nowhere else in this file: a deliberate
// change to production behaviour that alters these numbers updates them in the
// SAME commit as that change, by re-running this test with -v on EN and RU and
// copying the SAV-BYTEID-1197-CENSUS line in below. A baseline edit with no
// matching production change is itself a defect for that commit's review.
type sav1197Baseline struct {
	identical int
	refused   int
	campaign  int
	documents int
	docBytes  int
	tailBytes int
	position  int
	unknown   int
}

// Measured on this story's own tip against gameversions/saves, EN and RU:
// 103 files discovered, 1 unreadable and skipped by milestone2Corpus itself
// (its own "audited N file(s)" line carries that count), 102 attempted.
var sav1197OriginalBaseline = sav1197Baseline{
	identical: 12, // every one is a file this project's own writer produced
	refused:   0,
	campaign:  0,
	documents: 0,
	docBytes:  0,
	tailBytes: 0,
	position:  0,
	unknown:   0,
}

func TestSAVByteIdentity1197OriginalCorpus(t *testing.T) {
	verbose := os.Getenv("AGAINROM_SAV1197_RANGES") == "all"

	total, identical, refused := 0, 0, 0
	campaignDiff, documentsDiff, positionDiff, unknown := 0, 0, 0, 0
	docLocated, docBytesDiff, docUnlocatable := 0, 0, 0
	tailLocated, tailBytesDiff, tailHead := 0, 0, 0
	byClass, byRule, bytesByRule := map[string]int{}, map[string]int{}, map[string]int{}
	firstByRule := map[string]string{}

	milestone2Corpus(t, func(t *testing.T, mf milestone2File, _ *FrontEnd) {
		total++
		res, err := sav1197Inspect(mf.raw)
		if err != nil {
			refused++
			t.Errorf("REFUSED %s: %v", mf.rel, err)
			return
		}
		for _, note := range res.notes {
			t.Logf("%s: %s", mf.rel, note)
		}
		if !res.campaign {
			campaignDiff++
		}
		if !res.documents {
			documentsDiff++
		}
		if !res.position {
			positionDiff++
		}
		switch {
		case !res.docLocated:
			docUnlocatable++
		case res.docBytes:
			docLocated++
		default:
			docLocated++
			docBytesDiff++
		}
		if res.tailLocated {
			tailLocated++
			if !res.tailBytes {
				tailBytesDiff++
			}
			if res.tailHead {
				tailHead++
			}
		}
		if res.identical {
			identical++
			t.Logf("%s: BYTE-IDENTICAL", mf.rel)
			return
		}
		local := map[string][2]int{}
		for _, r := range res.ranges {
			byClass[r.class]++
			key := r.region + " | " + r.class + " | " + r.rule
			byRule[key]++
			bytesByRule[key] += r.n
			if _, ok := firstByRule[key]; !ok {
				firstByRule[key] = fmt.Sprintf("%s+%d (%s)", r.region, r.off, r.structure)
			}
			c := local[key]
			local[key] = [2]int{c[0] + 1, c[1] + r.n}
			if r.class == sav1197Unknown {
				unknown++
				t.Errorf("%s: UNCLASSIFIED %s+%d length %d structure=%q rule=%q",
					mf.rel, r.region, r.off, r.n, r.structure, r.rule)
				continue
			}
			if verbose {
				t.Logf("%s: %s+%d length %d structure=%q class=%s rule=%q",
					mf.rel, r.region, r.off, r.n, r.structure, r.class, r.rule)
			}
		}
		keys := make([]string, 0, len(local))
		for k := range local {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			t.Logf("%s: %d range(s), %d byte(s) -- %s", mf.rel, local[k][0], local[k][1], k)
		}
	})

	if total == 0 {
		t.Fatal("byte-identity corpus is empty; AGAINROM_SAVE_CORPUS is misconfigured")
	}

	// The owner's own order of priority: the Valuable Documents journal and the
	// campaign position first, then everything else.
	t.Logf("SAV-BYTEID-1197-DOCUMENT-VALUES attempted=%d identical=%d differing=%d",
		total, total-documentsDiff, documentsDiff)
	t.Logf("SAV-BYTEID-1197-DOCUMENT-BYTES located=%d byte-identical=%d differing=%d unlocatable-empty-collection=%d",
		docLocated, docLocated-docBytesDiff, docBytesDiff, docUnlocatable)
	t.Logf("SAV-BYTEID-1197-CAMPAIGN-TRAILER located=%d byte-identical=%d differing=%d first-dword-is-SelectedMission=%d",
		tailLocated, tailLocated-tailBytesDiff, tailBytesDiff, tailHead)
	t.Logf("SAV-BYTEID-1197-POSITION attempted=%d identical=%d differing=%d",
		total, total-positionDiff, positionDiff)
	t.Logf("SAV-BYTEID-1197-CAMPAIGN-BYTES attempted=%d identical=%d differing=%d",
		total, total-campaignDiff, campaignDiff)
	t.Logf("SAV-BYTEID-1197-CENSUS attempted=%d byte-identical=%d differing=%d refused=%d unclassified-ranges=%d",
		total, identical, total-identical-refused, refused, unknown)
	for _, c := range []string{sav1197Intended, sav1197ReEmitted, sav1197Unmodelled, sav1197Unknown} {
		t.Logf("SAV-BYTEID-1197-CLASS %-20s %d range(s)", c, byClass[c])
	}
	keys := make([]string, 0, len(byRule))
	for k := range byRule {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("SAV-BYTEID-1197-RULE %d range(s), %d byte(s), first at %s -- %s",
			byRule[k], bytesByRule[k], firstByRule[k], k)
	}

	base := sav1197OriginalBaseline
	if identical < base.identical {
		t.Errorf("byte-identical count regressed: %d, baseline %d", identical, base.identical)
	}
	// The 32-byte trailer's attribution is measured, not asserted: its first
	// dword must be the SelectedMission this reader decodes, on every file where
	// the trailer was located at all.
	if tailHead != tailLocated {
		t.Errorf("campaign record trailer: first dword is the decoded SelectedMission on %d of %d located trailers",
			tailHead, tailLocated)
	}
	for _, c := range []struct {
		name  string
		got   int
		limit int
	}{
		{"refused", refused, base.refused},
		{"campaign-record differing", campaignDiff, base.campaign},
		{"campaign-document value differing", documentsDiff, base.documents},
		{"campaign-document byte differing", docBytesDiff, base.docBytes},
		{"campaign-record trailer byte differing", tailBytesDiff, base.tailBytes},
		{"campaign-position differing", positionDiff, base.position},
		{"unclassified ranges", unknown, base.unknown},
	} {
		if c.got > c.limit {
			t.Errorf("%s rose above its baseline: %d, baseline %d", c.name, c.got, c.limit)
		}
	}
}
