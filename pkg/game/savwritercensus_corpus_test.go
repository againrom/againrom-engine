//go:build sessioncorpusaudit

// Writer census over changed worlds (pipeline/SAV-ENDGAME.md, execution order
// item eight, second part). The round-trip instrument resaves an unchanged
// load, where carried original bytes hide writer gaps. This census loads each
// corpus original, changes the world, saves through the player's SAV dialog
// route, and checks every persisted field it covers.
//
// Every field the census covers is compared with the loaded original. A
// difference is accepted only when a published rule fixed the written value
// exactly in that file, or when writerCensusDebt names it with its cause; any
// other difference fails. The rules describe state the session creates or
// changes, not what SAVE writes for carried state. The expected side of every
// rule comes from a claim or from the original's own bytes, never from this
// engine's reader. Bytes are read through the sack byte walker the Sacks
// acceptance instrument owns, the generic document decoder, and the generic
// registry parser.
//
// Each file runs as its own parallel subtest with its own front end, bounded
// by writerCensusWorkers. The corpus is split over writerCensusParts part
// tests, each a separate process in the milestone-2 gate; the census over the
// whole corpus is checked once, by the part that completes the set.
package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// writerCensusViewFloor is SESS-VIEW-030's lower scroll bound: the origin of
// a world camera never sits in the outermost eight cells.
const writerCensusViewFloor = 8

// writerCensusDecayTicks bounds how long a killed actor is advanced toward
// its terminal stage; the full-decay regression test uses the same span.
const writerCensusDecayTicks = 24000

// writerCensusWorkers bounds how many files are loaded and changed at once.
const writerCensusWorkers = 6

// writerCensusParts is how many part tests share the corpus.
const writerCensusParts = 6

type writerCensusBaselineRow struct {
	ceiling int
	cause   string
}

// writerCensusBaseline names every refusal reason and rule mismatch the
// census may carry, each with its cause. A reason or mismatch key absent here
// has ceiling zero; a count above its ceiling fails.
var writerCensusBaseline = map[string]writerCensusBaselineRow{}

// writerCensusChangeFloor is how many world files each change must reach on
// the discovered corpus. A change that stops taking effect lowers its count
// and fails; a grown corpus raises it and the floor may be raised with it.
var writerCensusChangeFloor = map[string]int{
	"camera":          65,
	"drop-new-sack":   63,
	"pickup":          61,
	"party-move":      65,
	"kill-full-decay": 65,
}

// writerCensusDebt names fields that the writer produces from current state
// and that differ from the loaded original with no published rule for their
// value. Keys are "family:field"; the family is sack, sack-item, registry (a
// world file's state store) or town (a town file's).
var writerCensusDebt = map[string]string{
	"registry:/CurrentState/AgainromActions": "engine continuation supplement, DIV-1369; rewritten from current state on every save; no ROM1 rule",
	"registry:/CurrentState/AgainromRng":     "engine random-stream leaf, DIV-1202; advances with the simulation; no ROM1 rule",
	"registry:/CurrentState/AgainromSeed":    "engine random-session leaf, DIV-2736; an original SAV's session seed is derived from its bytes; no ROM1 rule",
	"registry:/Fog/Data": "explored plane written from current state after the party moved; SAV-FOG-061 fixes the encoding and a LOAD that only ORs, " +
		"checked here as runs covering the loaded extent with no explored cell lost; no rule fixes which new cells are explored",
	"registry:/Objects/Selection":        "current selection written from state; no rule fixes its value",
	"registry:/Projectiles/FreeIndex":    "projectile allocator advanced by each shot a ranged swing released after the census's orders; SAV-1131 fixes its step, no rule fixes its value",
	"registry:/Projectiles/IDs":          "projectile store written from the projectiles in flight; SAV-PROJSTORE-428 fixes the layout, no rule fixes a value",
	"registry:/SpellBook/Pressed":        "the viewer's current spell after the census's orders, written from state; no rule fixes its value",
	"town:/CurrentState/AgainromActions": "engine continuation supplement, DIV-1369; rewritten from current state on every save; no ROM1 rule",
	"town:/CurrentState/AgainromSeed":    "engine random-session leaf, DIV-2736; an original SAV's session seed is derived from its bytes; no ROM1 rule",
	"town:/Objects/Selection": "the town writer writes an empty selection where the loaded file names objects by index; SAV-914 has LOAD consume Objects, " +
		"but no claim fixes what a town selection index names in a rebuilt document, so the loaded bytes are not carried",
}

func init() {
	for _, leaf := range []string{"action", "actiondir", "actionphase", "actionsegments", "actionspell", "actiontarget",
		"actionx", "actiony", "actionz", "dir", "lastaction", "phase", "picture", "x", "y", "z"} {
		writerCensusDebt["registry:/Prj#/"+leaf] = "projectile in flight written from state; SAV-PROJSTORE-428 fixes the layout, no rule fixes a value"
	}
}

// writerCensusRules names the fields an exact rule can accept: a difference
// counts as ruled only in a file where the rule fixed the written value and
// the value matched it.
var writerCensusRules = map[string]string{
	"registry:/View/X": "SESS-VIEW-030 floor over the camera after the change",
	"registry:/View/Y": "SESS-VIEW-030 floor over the camera after the change",
	"sack:T1C":         "ITEM-SACK-010 over contained item values each equal to the original's",
	"sack-item:T08":    "ITEM-GROUNDMOVE-130's pickup stamp, the original flags OR 1",
	"sack-item:F42":    "ITEM-MERGE-129: an item put onto an original stack of equal code joins that record and adds its count; checked as a grown count on a record keeping the original identity, the incoming count not counted independently",
	"sack-item:split":  "a record split off an original stack repeats its fields but key and count, and the counts sum to the original",
}

const (
	writerCensusSackKey    = "T1C|ITEM-SACK-010"
	writerCensusViewKey    = "/View|SESS-VIEW-030"
	writerCensusFogKey     = "/Fog/Data|SAV-FOG-061"
	writerCensusRosterKey  = "party supplement|PARTY-ROSTER-002"
	writerCensusMissingKey = "sack|missing original"
	writerCensusIDKey      = "constructed sack runtime id|MOVE-ID-016"
	writerCensusMaskKey    = "constructed sack T18|SAV-655"
	writerCensusItemKey    = "sack item|no original record"
)

// writerCensusItemClasses are the item classes a Sack's contents hold.
var writerCensusItemClasses = map[string]bool{"Item": true, "Weapon": true, "Armor": true, "Shield": true}

// writerCensusWrite is one original's changed and written state: the written
// bytes and what the change fixes about them.
type writerCensusWrite struct {
	raw        []byte
	town       bool
	changes    []string
	view       [2]int32
	viewSet    bool
	picked     [2]int32
	pickedDone bool
}

// writerCensusResult is one file's outcome, aggregated after every file ran.
type writerCensusResult struct {
	rel        string
	town       bool
	changes    []string
	stage      string
	err        error
	mismatch   map[string][]string
	fields     map[string]int
	verified   map[string]bool
	known      map[string][]string
	knownIssue string
	took       time.Duration
}

type writerCensusTally struct {
	discovered, changed, written int
	refused                      map[string]int
	mismatches                   map[string]int
	known                        map[string]int
	debt                         map[string]int
	unnamed                      map[string]int
	changes                      map[string]int
	ruled                        map[string]int
}

func newWriterCensusTally() *writerCensusTally {
	return &writerCensusTally{refused: map[string]int{}, mismatches: map[string]int{}, known: map[string]int{}, debt: map[string]int{},
		unnamed: map[string]int{}, changes: map[string]int{}, ruled: map[string]int{}}
}

// writerCensusFiles walks AGAINROM_SAVE_CORPUS with the milestone-2 walker's
// directory rule and returns every readable .sav, logging each unreadable one.
func writerCensusFiles(t *testing.T) []milestone2File {
	t.Helper()
	return writerCensusWalk(t, true)
}

// writerCensusWalk is writerCensusFiles with the unreadable lines optional,
// so the parts of one census name each unreadable file once.
func writerCensusWalk(t *testing.T, logUnreadable bool) []milestone2File {
	t.Helper()
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS must name gameversions/saves")
	}
	var out []milestone2File
	unreadable := 0
	err := filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".sav" {
			return nil
		}
		rel, relErr := filepath.Rel(corpus, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		f, err := sav.Open(raw)
		if err != nil {
			unreadable++
			if logUnreadable {
				t.Logf("%s: unreadable, skipped: %v", rel, err)
			}
			return nil
		}
		out = append(out, milestone2File{rel: rel, raw: raw, body: f.Body, f: f, present: f.World != nil})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// writerCensusEach runs fn once per file, each in its own parallel subtest,
// at most writerCensusWorkers at a time, and returns the results in file
// order once every subtest has finished. fn shares no mutable state.
func writerCensusEach(t *testing.T, files []milestone2File, fn func(t *testing.T, mf milestone2File) writerCensusResult) []writerCensusResult {
	t.Helper()
	out := make([]writerCensusResult, len(files))
	slots := make(chan struct{}, writerCensusWorkers)
	// Larger files start first: a blocked channel send is served in order,
	// so the long worlds do not trail the short ones.
	order := make([]int, len(files))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return len(files[order[a]].raw) > len(files[order[b]].raw) })
	t.Run("files", func(t *testing.T) {
		for _, i := range order {
			mf := files[i]
			t.Run(mf.rel, func(t *testing.T) {
				t.Parallel()
				slots <- struct{}{}
				defer func() { <-slots }()
				start := time.Now()
				r := fn(t, mf)
				r.rel, r.took = mf.rel, time.Since(start)
				out[i] = r
			})
		}
	})
	return out
}

// writerCensusSackRule applies ITEM-SACK-010: a Sack's token value +0x1c is
// its gold (+0x3c) plus the sum of each contained item record's own +0x1c, in
// the field's own 32-bit width.
func writerCensusSackRule(src sackByteSource) map[string][]string {
	out := map[string][]string{}
	for _, index := range src.indices() {
		r := src.rows[index]
		if r.class != "Sack" {
			continue
		}
		want := r.values["S3C"]
		var parts []string
		for _, ref := range r.refs["Contents"] {
			item := src.rows[ref]
			if item == nil {
				continue
			}
			want += item.values["T1C"]
			parts = append(parts, fmt.Sprintf("%s code %d count %d value %d", item.class, item.values["F40"], item.values["F42"], int32(item.values["T1C"])))
		}
		if got := r.values["T1C"]; got != want {
			out[writerCensusSackKey] = append(out[writerCensusSackKey], fmt.Sprintf("sack archive %d runtime id %d T1C=%d, gold %d plus contents gives %d; contents %v",
				index, r.values["RuntimeID"], int32(got), r.values["S3C"], int32(want), parts))
		}
	}
	return out
}

// writerCensusLeaves flattens a registry into path -> printed value.
func writerCensusLeaves(r *reg.Reg) map[string]string {
	out := map[string]string{}
	if r == nil || r.Root == nil {
		return out
	}
	var walk func(prefix string, n *reg.Node)
	walk = func(prefix string, n *reg.Node) {
		for _, c := range n.Children {
			p := prefix + "/" + c.Name
			if c.Dir {
				walk(p, c)
				continue
			}
			switch c.Type {
			case reg.TypeInt:
				out[p] = fmt.Sprint(c.Int)
			case reg.TypeFloat:
				out[p] = fmt.Sprint(c.Float)
			case reg.TypeString:
				out[p] = c.Str
			case reg.TypeIntArray:
				out[p] = fmt.Sprint(c.Ints)
			default:
				out[p] = fmt.Sprintf("kind %d", c.Kind)
			}
		}
	}
	walk("", r.Root)
	return out
}

// writerCensusNode returns the registry leaf at path, or nil.
func writerCensusNode(r *reg.Reg, path string) *reg.Node {
	if r == nil || r.Root == nil {
		return nil
	}
	n := r.Root
	for _, name := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		var next *reg.Node
		for _, c := range n.Children {
			if c.Name == name {
				next = c
				break
			}
		}
		if next == nil {
			return nil
		}
		n = next
	}
	return n
}

// writerCensusViewFloorRule applies SESS-VIEW-030's lower bound to a world
// save's camera origin. The reference test applies it to the originals.
func writerCensusViewFloorRule(leaves map[string]string) []string {
	var out []string
	for _, axis := range []string{"/View/X", "/View/Y"} {
		v, ok := leaves[axis]
		if !ok {
			out = append(out, axis+" absent from a world save")
			continue
		}
		var n int
		if _, err := fmt.Sscan(v, &n); err != nil || n < writerCensusViewFloor {
			out = append(out, fmt.Sprintf("%s=%s below SESS-VIEW-030's floor %d", axis, v, writerCensusViewFloor))
		}
	}
	return out
}

// writerCensusViewExact checks a world save's /View against the origin the
// camera held when the census saved, floored by SESS-VIEW-030.
func writerCensusViewExact(leaves map[string]string, want [2]int32) []string {
	var out []string
	for i, axis := range []string{"/View/X", "/View/Y"} {
		if got, w := leaves[axis], fmt.Sprint(want[i]); got != w {
			out = append(out, fmt.Sprintf("%s=%s, the camera after the change gives %s", axis, got, w))
		}
	}
	return out
}

// writerCensusFogRule applies SAV-FOG-061: /Fog/Data is runs of tile bit 15
// over the whole plane, the first in /Fog/FirstState's state. A LOAD only
// ORs the bit in, so the written runs must cover the loaded plane's extent and
// keep every cell the loaded file had explored.
func writerCensusFogRule(original, written *reg.Reg) []string {
	plane := func(r *reg.Reg) ([]bool, bool) {
		first, data := writerCensusNode(r, "/Fog/FirstState"), writerCensusNode(r, "/Fog/Data")
		if first == nil || data == nil {
			return nil, false
		}
		var cells []bool
		state := first.Int != 0
		for _, run := range data.Ints {
			for range run {
				cells = append(cells, state)
			}
			state = !state
		}
		return cells, true
	}
	before, okBefore := plane(original)
	after, okAfter := plane(written)
	switch {
	case !okBefore:
		return nil
	case !okAfter:
		return []string{"the loaded file carries a fog plane the written file lacks"}
	case len(before) != len(after):
		return []string{fmt.Sprintf("written runs cover %d cells, the loaded plane %d", len(after), len(before))}
	}
	lost := 0
	for i := range before {
		if before[i] && !after[i] {
			lost++
		}
	}
	if lost > 0 {
		return []string{fmt.Sprintf("%d explored cell(s) of the loaded plane written unexplored", lost)}
	}
	return nil
}

// writerCensusRosterRule applies PARTY-ROSTER-002 (membership is
// containment, so one actor belongs to one place in the roster) to this
// engine's own party supplement: no actor is named twice across its party
// and roster lists.
func writerCensusRosterRule(raw []byte) ([]string, error) {
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		return nil, err
	}
	b, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		return nil, err
	}
	var a struct {
		Party, Roster []struct{ Entity uint32 }
	}
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	seen := map[uint32]bool{}
	var out []string
	for _, rows := range [][]struct{ Entity uint32 }{a.Party, a.Roster} {
		for _, p := range rows {
			if seen[p.Entity] {
				out = append(out, fmt.Sprintf("actor %d named twice in the party supplement", p.Entity))
			}
			seen[p.Entity] = true
		}
	}
	return out, nil
}

// writerCensusRecordValues maps a decoded record's named values and raw
// blocks to printed values for comparison.
func writerCensusRecordValues(r *sav.DocumentRecordData) map[string]string {
	out := map[string]string{}
	for _, v := range r.Values {
		out[v.Name] = fmt.Sprint(v.Value)
	}
	for _, v := range r.Raw {
		out[v.Name] = fmt.Sprintf("%x", v.Bytes)
	}
	return out
}

func writerCensusRuntimeID(r *sav.DocumentRecordData) uint32 {
	for _, v := range r.Values {
		if v.Name == "RuntimeID" {
			return v.Value
		}
	}
	return 0
}

// writerCensusSackCell is a Sack's cell, from its Token position block.
func writerCensusSackCell(r *sackByteRecord) [2]int32 {
	b := r.raw["Block12"]
	if len(b) < 4 {
		return [2]int32{-1, -1}
	}
	return [2]int32{int32(b[2]), int32(b[3])}
}

// writerCensusSacks compares the written Sacks with the original's. An
// original sack joins a written one on cell and runtime id and every field is
// compared. Every contained item record is compared field by field with the
// original record of the same class and Identity key; a record with a new key
// must be split off an original stack whose other fields it repeats, with the
// count conserved. An original sack absent from the written file is a
// mismatch unless it is the one the party picked up. A written sack with no
// original is constructed by the session and must carry a nonzero runtime id
// no other record holds (MOVE-ID-016) and mask 2 at Token+0x18 (SAV-655).
// verified records the item fields whose changes a rule confirmed, and
// whether every sack's ITEM-SACK-010 value rests on item values equal to the
// original's.
func writerCensusSacks(before, after sackByteSource, origDoc, writtenDoc *sav.DocumentData, w writerCensusWrite, mismatch map[string][]string, fields map[string]int, verified map[string]bool) {
	type key struct {
		cell [2]int32
		id   uint32
	}
	// An item record joins its original on class and Identity key, which a
	// record keeps wherever it moves; runtime ids repeat across item records.
	byIdentity := func(d *sav.DocumentData) (map[string]map[string]string, []map[string]string) {
		out, all := map[string]map[string]string{}, []map[string]string(nil)
		for i := range d.Objects {
			r := &d.Objects[i]
			if values := writerCensusRecordValues(r); writerCensusItemClasses[r.Class] {
				values["class"] = r.Class
				all = append(all, values)
				if values["Identity"] != "0" {
					out[r.Class+" "+values["Identity"]] = values
				}
			}
		}
		return out, all
	}
	origItems, origAll := byIdentity(origDoc)
	writtenItems, _ := byIdentity(writtenDoc)
	holders := map[uint32]int{}
	for i := range writtenDoc.Objects {
		if id := writerCensusRuntimeID(&writtenDoc.Objects[i]); id != 0 {
			holders[id]++
		}
	}
	identities := func(r *sackByteRecord, src sackByteSource) []uint32 {
		var out []uint32
		for _, ref := range r.refs["Contents"] {
			if item := src.rows[ref]; item != nil {
				out = append(out, item.values["Identity"])
			}
		}
		return out
	}
	origSacks := map[key]*sackByteRecord{}
	var origKeys []key
	for _, index := range before.indices() {
		if r := before.rows[index]; r.class == "Sack" {
			k := key{writerCensusSackCell(r), r.values["RuntimeID"]}
			origSacks[k] = r
			origKeys = append(origKeys, k)
		}
	}
	valuesExact, stampExact, splitExact, mergeExact := true, true, true, true
	matched := map[key]bool{}
	for _, index := range after.indices() {
		r := after.rows[index]
		if r.class != "Sack" {
			continue
		}
		k := key{writerCensusSackCell(r), r.values["RuntimeID"]}
		if old := origSacks[k]; old != nil {
			matched[k] = true
			for name, v := range r.values {
				if old.values[name] != v {
					fields["sack:"+name]++
				}
			}
			for name, b := range r.raw {
				if string(old.raw[name]) != string(b) {
					fields["sack:"+name]++
				}
			}
			if fmt.Sprint(identities(old, before)) != fmt.Sprint(identities(r, after)) {
				fields["sack:Contents"]++
			}
		} else {
			id := r.values["RuntimeID"]
			if id == 0 || holders[id] > 1 {
				mismatch[writerCensusIDKey] = append(mismatch[writerCensusIDKey], fmt.Sprintf("sack at %v runtime id %d held by %d record(s)", k.cell, id, holders[id]))
			}
			if r.values["T18"] != 2 {
				mismatch[writerCensusMaskKey] = append(mismatch[writerCensusMaskKey], fmt.Sprintf("sack at %v runtime id %d T18=%d", k.cell, id, r.values["T18"]))
			}
		}
		for _, ref := range r.refs["Contents"] {
			item := after.rows[ref]
			if item == nil {
				continue
			}
			have := map[string]string{}
			for name, v := range item.values {
				have[name] = fmt.Sprint(v)
			}
			for name, b := range item.raw {
				have[name] = fmt.Sprintf("%x", b)
			}
			orig := origItems[fmt.Sprint(item.class, " ", item.values["Identity"])]
			if item.values["Identity"] == 0 || orig == nil {
				// A record split off a stack: every field but the key and the
				// count repeats an original record, whose written remainder
				// and this record's count sum to the original count.
				split := false
				for _, src := range origAll {
					same := src["class"] == item.class
					for name, v := range have {
						same = same && (name == "Identity" || name == "F42" || src[name] == v)
					}
					if !same {
						continue
					}
					var want, rest, got uint32
					fmt.Sscan(src["F42"], &want)
					fmt.Sscan(have["F42"], &got)
					if remainder := writtenItems[item.class+" "+src["Identity"]]; remainder != nil {
						fmt.Sscan(remainder["F42"], &rest)
					}
					if rest+got == want {
						split = true
						break
					}
				}
				if split {
					fields["sack-item:split"]++
					continue
				}
				splitExact, valuesExact = false, false
				mismatch[writerCensusItemKey] = append(mismatch[writerCensusItemKey], fmt.Sprintf("sack at %v holds %s code %d identity %d with no original record of that identity and no stack it splits from",
					k.cell, item.class, item.values["F40"], item.values["Identity"]))
				continue
			}
			for name, v := range have {
				if orig[name] == v {
					continue
				}
				fields["sack-item:"+name]++
				switch name {
				case "F42":
					var was, now uint32
					fmt.Sscan(orig[name], &was)
					fmt.Sscan(v, &now)
					mergeExact = mergeExact && now > was
				case "T1C":
					valuesExact = false
				case "T08":
					// ITEM-GROUNDMOVE-130: pickup writes Token+0x08 = 1 on every
					// incoming Item; a merge keeps the old flags OR 1.
					var before uint32
					fmt.Sscan(orig[name], &before)
					stampExact = stampExact && item.values["T08"] == before|1
				}
			}
		}
	}
	for _, k := range origKeys {
		if matched[k] || w.pickedDone && k.cell == w.picked {
			continue
		}
		r := origSacks[k]
		mismatch[writerCensusMissingKey] = append(mismatch[writerCensusMissingKey], fmt.Sprintf("original sack at %v runtime id %d gold %d T1C %d is absent from the written file",
			k.cell, k.id, r.values["S3C"], int32(r.values["T1C"])))
	}
	verified["sack:T1C"] = valuesExact
	verified["sack-item:T08"] = stampExact
	verified["sack-item:split"] = splitExact
	verified["sack-item:F42"] = mergeExact
}

// writerCensusCheck runs every rule over one written file and diffs the
// families it covers against the loaded original. It returns rule mismatches
// keyed "field|rule", differing fields keyed "family:field", and the fields
// whose written value an exact rule fixed and confirmed in this file.
func writerCensusCheck(original *sav.File, originalRaw []byte, w writerCensusWrite, writtenRaw []byte) (map[string][]string, map[string]int, map[string]bool, error) {
	written, err := sav.Open(writtenRaw)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse: %w", err)
	}
	mismatch := map[string][]string{}
	fields := map[string]int{}
	verified := map[string]bool{}
	if written.World != nil && original.World != nil {
		src, err := sackByteWalk(written)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("sack walk: %w", err)
		}
		before, err := sackByteWalk(original)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("original sack walk: %w", err)
		}
		origDoc, err := sav.DecodeDocumentData(originalRaw)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("original document: %w", err)
		}
		writtenDoc, err := sav.DecodeDocumentData(writtenRaw)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("written document: %w", err)
		}
		sackRule := writerCensusSackRule(src)
		for key, lines := range sackRule {
			mismatch[key] = lines
		}
		writerCensusSacks(before, src, &origDoc, &writtenDoc, w, mismatch, fields, verified)
		verified["sack:T1C"] = verified["sack:T1C"] && len(sackRule[writerCensusSackKey]) == 0
	} else if written.World != nil {
		return nil, nil, nil, fmt.Errorf("a town original was written with a world half")
	}
	store, _ := written.StateStore()
	origStore, _ := original.StateStore()
	leaves := writerCensusLeaves(store)
	if written.World != nil {
		if w.viewSet {
			lines := writerCensusViewExact(leaves, w.view)
			mismatch[writerCensusViewKey] = lines
			for i, axis := range []string{"/View/X", "/View/Y"} {
				verified["registry:"+axis] = leaves[axis] == fmt.Sprint(w.view[i])
			}
		} else {
			mismatch[writerCensusViewKey] = writerCensusViewFloorRule(leaves)
		}
		mismatch[writerCensusFogKey] = writerCensusFogRule(origStore, store)
	}
	// A town file's leaves are their own family: the town writer has no
	// world to take current state from.
	family := "registry:"
	if written.World == nil {
		family = "town:"
	}
	origLeaves := writerCensusLeaves(origStore)
	for p, v := range leaves {
		if origLeaves[p] != v {
			fields[family+writerCensusRegistryField(p)]++
		}
	}
	for p := range origLeaves {
		if _, ok := leaves[p]; !ok {
			fields[family+writerCensusRegistryField(p)]++
		}
	}
	roster, err := writerCensusRosterRule(writtenRaw)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("party supplement: %w", err)
	}
	mismatch[writerCensusRosterKey] = roster
	return mismatch, fields, verified, nil
}

// writerCensusClassify splits one file's differing fields into ruled (an
// exact rule confirmed the written value), debt (named with a cause) and
// unnamed (a failure).
func writerCensusClassify(fields map[string]int, verified map[string]bool) (ruled, debt, unnamed map[string]int) {
	ruled, debt, unnamed = map[string]int{}, map[string]int{}, map[string]int{}
	for field, n := range fields {
		switch {
		case verified[field]:
			ruled[field] += n
		case writerCensusDebt[field] != "":
			debt[field] += n
		default:
			unnamed[field] += n
		}
	}
	return ruled, debt, unnamed
}

// writerCensusRegistryField folds numbered registry roots into one field
// name so a debt row covers the family, not one id.
func writerCensusRegistryField(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		trimmed := strings.TrimRight(s, "0123456789")
		if trimmed != s && trimmed != "" {
			parts[i] = trimmed + "#"
		}
	}
	return strings.Join(parts, "/")
}

// writerCensusCameraOrigin is the origin the SAV writer reads from the
// camera, floored by SESS-VIEW-030's lower bound.
func writerCensusCameraOrigin(v *ui.Viewer) [2]int32 {
	app := v.SaveApplication()
	var out [2]int32
	for i, c := range []float64{app.ViewX, app.ViewY} {
		n := int32(c + 0.5)
		if c < 0 {
			n = int32(c - 0.5)
		}
		out[i] = max(n, writerCensusViewFloor)
	}
	return out
}

// writerCensusChange changes one opened mission: a drop from a pack into a
// new sack, a pick-up of the nearest sack, party movement, a kill advanced to
// full decay, and last the camera. It returns the changes that took effect.
func writerCensusChange(f *FrontEnd, loadedView [2]int32, loadedViewSet bool) (w writerCensusWrite) {
	mw := f.live
	if mw == nil || mw.world == nil {
		return w
	}
	defer writerCensusCamera(mw, loadedView, loadedViewSet, &w)
	party := append([]sim.EntityID(nil), mw.mission.ids...)
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range mw.world.Entities() {
		byID[e.ID] = e
	}
	var alive []sim.Entity
	inParty := map[sim.EntityID]bool{}
	for _, id := range party {
		inParty[id] = true
		if e, ok := byID[id]; ok && e.Alive() && !e.OffMap {
			alive = append(alive, e)
		}
	}
	if len(alive) == 0 {
		return w
	}
	startCells := map[sim.EntityID][2]int32{}
	for _, e := range alive {
		startCells[e.ID] = [2]int32{e.X, e.Y}
	}
	sacksBefore := len(mw.world.Sacks())
	// A drop from the first member's pack, at its own cell, is a new sack.
	for slot := 0; slot < 4; slot++ {
		mw.pending = append(mw.pending, sim.DropCarried(alive[0].ID, sim.ItemSlot(slot), sim.CellPoint{X: alive[0].X, Y: alive[0].Y}))
		f.LiveAdvance(1)
		if len(mw.world.Sacks()) > sacksBefore {
			w.changes = append(w.changes, "drop-new-sack")
			break
		}
	}
	// The member nearest an original sack picks it up.
	picker, best, bestD := alive[0], sim.Sack{}, int32(-1)
	for _, e := range alive {
		for _, sk := range mw.world.Sacks() {
			if sk.X == alive[0].X && sk.Y == alive[0].Y {
				continue
			}
			if d := max(abs32(sk.X-e.X), abs32(sk.Y-e.Y)); bestD < 0 || d < bestD {
				picker, best, bestD = e, sk, d
			}
		}
	}
	// Every other member walks three cells. The pick-up order follows in a
	// later tick, because a move order issued in the same tick joins one
	// group order and cancels a standing pick-up.
	for _, e := range alive {
		if e.ID == picker.ID {
			continue
		}
		f.LiveOrder(uint32(e.ID), int(e.X+3), int(e.Y+3))
	}
	f.LiveAdvance(1)
	if bestD >= 0 {
		mw.orderPickup(picker.ID, best.X, best.Y)
	}
	owner := alive[0].Owner
	victim, found := sim.Entity{}, false
	for _, e := range mw.world.Entities() {
		if !inParty[e.ID] && e.Alive() && !e.OffMap && e.Owner != owner {
			victim, found = e, true
			break
		}
	}
	if found {
		f.LiveKill(uint32(victim.ID))
	} else {
		w.changes = append(w.changes, "not:kill-no-living-opponent")
	}
	taken := func() bool {
		for _, sk := range mw.world.Sacks() {
			if sk.X == best.X && sk.Y == best.Y && sk.ObjectID == best.ObjectID {
				return false
			}
		}
		return true
	}
	// Advance until the victim reaches full decay and the pick-up is done,
	// checking every 100 ticks; a file with no victim advances 1600 ticks for
	// the pick-up walk.
	limit := writerCensusDecayTicks
	if !found {
		limit = 1600
	}
	for t := 0; t < limit; t += 100 {
		f.LiveAdvance(100)
		if found {
			// Full decay ends in the terminal-actor list or, for an actor with
			// no document root, in removal from the living entity list.
			_, present := mw.world.Entity(victim.ID)
			terminal := !present
			for _, a := range mw.world.CurrentTerminalActors() {
				terminal = terminal || a.ID == victim.ID
			}
			if terminal {
				w.changes = append(w.changes, "kill-full-decay")
				found = false
			}
		}
		if !found && (t+100 >= 1600 || bestD >= 0 && taken()) {
			break
		}
	}
	if found {
		w.changes = append(w.changes, "not:kill-no-full-decay")
	}
	switch {
	case bestD < 0:
		w.changes = append(w.changes, "not:pickup-no-original-sack")
	case taken():
		w.changes = append(w.changes, "pickup")
		w.picked, w.pickedDone = [2]int32{best.X, best.Y}, true
	default:
		w.changes = append(w.changes, "not:pickup-incomplete")
	}
	moved := false
	for _, e := range mw.world.Entities() {
		if c, ok := startCells[e.ID]; ok && (c[0] != e.X || c[1] != e.Y) {
			moved = true
		}
	}
	if moved {
		w.changes = append(w.changes, "party-move")
	}
	return w
}

// writerCensusCamera pans the camera, as a player scrolls, through the
// viewer's clamp to the top-left corner, or on toward the centre when that
// leaves the loaded origin unchanged, and records the origin the save must
// carry.
func writerCensusCamera(mw *mapWorld, loaded [2]int32, loadedSet bool, w *writerCensusWrite) {
	if mw.view == nil || mw.view.Camera() == nil {
		return
	}
	cam := mw.view.Camera()
	cam.Pan(-cam.WorldW(), -cam.WorldH())
	w.view = writerCensusCameraOrigin(mw.view)
	if loadedSet && w.view == loaded {
		cam.Pan(cam.WorldW()/2, cam.WorldH()/2)
		w.view = writerCensusCameraOrigin(mw.view)
	}
	w.viewSet = true
	if loadedSet && w.view == loaded {
		w.changes = append(w.changes, "not:camera-unmoved")
		return
	}
	w.changes = append(w.changes, "camera")
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// writerCensusSave writes through the SAV dialog's Prepare and Commit, the
// player's own save route, and reads the file back.
func writerCensusSave(f *FrontEnd, onMap bool, dir string) ([]byte, string, error) {
	seams := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})
	prepared, err := seams.Prepare(ui.SaveRequest{OnMap: onMap, Directory: dir, Name: "census", Format: ui.SaveSAV})
	if err != nil {
		return nil, "SAVE-PREPARE", err
	}
	if _, err := prepared.Commit(true); err != nil {
		return nil, "SAVE-COMMIT", err
	}
	raw, err := ReadSaveFile(filepath.Join(dir, "census.sav"))
	if err != nil {
		return nil, "READ", err
	}
	return raw, "", nil
}

// writerCensusWriteOne loads, changes and writes one original on its own
// front end. It returns the refusal stage and error when no file is written.
func writerCensusWriteOne(t *testing.T, assets string, mf milestone2File) (writerCensusWrite, string, error) {
	f, err := decodedInstallFront(assets)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", assets, err)
	}
	cleanupFrontAudio(t, f)
	headlessCensusMusic(t, f)
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(mf.raw)
	if err != nil {
		return writerCensusWrite{}, "LOAD", err
	}
	w := writerCensusWrite{town: town}
	if !town {
		a := f.App("writer census")
		defer a.StopAudio()
		a.Layout(1024, 768)
		if err := a.OpenMission(open); err != nil {
			return w, "OPEN", err
		}
		store, _ := mf.f.StateStore()
		leaves := writerCensusLeaves(store)
		var loaded [2]int32
		_, errX := fmt.Sscan(leaves["/View/X"], &loaded[0])
		_, errY := fmt.Sscan(leaves["/View/Y"], &loaded[1])
		w = writerCensusChange(f, loaded, errX == nil && errY == nil)
	}
	raw, stage, err := writerCensusSave(f, !town, t.TempDir())
	if err != nil {
		return w, stage, err
	}
	w.raw = raw
	return w, "", nil
}

// writerCensusRun writes and checks one original, mutate standing in for a
// writer with one fix disabled.
func writerCensusRun(t *testing.T, assets string, mf milestone2File) writerCensusResult {
	w, stage, err := writerCensusWriteOne(t, assets, mf)
	r := writerCensusResult{town: w.town, changes: w.changes, stage: stage, err: err}
	if err != nil {
		return r
	}
	r.mismatch, r.fields, r.verified, err = writerCensusCheck(mf.f, mf.raw, w, w.raw)
	if err != nil {
		r.stage, r.err = "CHECK", err
	} else {
		r.known, r.knownIssue = writerCensusKnownNewInputs(mf, w.raw, r.mismatch)
	}
	return r
}

var writerCensusNewInputSHA = map[string]string{
	"2026-09-27/oldsaves7/game0005.sav": "715d7f9d20b90a966ab6e9f14fb8fa4cea90097a1a41f2488ed8d5d38da414a4",
}

func writerCensusExtraMismatch(mismatch map[string][]string, allowed ...string) string {
	for _, key := range writerCensusSortedLines(mismatch) {
		if len(mismatch[key]) > 0 && !slices.Contains(allowed, key) {
			return key
		}
	}
	return ""
}

func writerCensusKnownNewInputs(mf milestone2File, written []byte, mismatch map[string][]string) (map[string][]string, string) {
	sha, named := writerCensusNewInputSHA[mf.rel]
	if !named {
		return nil, ""
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(mf.raw)); got != sha {
		return nil, fmt.Sprintf("%s source SHA %s, want %s", mf.rel, got, sha)
	}
	return writerCensusOriginalSackAnomalies(mf, written, mismatch)
}

func writerCensusOriginalSackAnomalies(mf milestone2File, written []byte, mismatch map[string][]string) (map[string][]string, string) {
	before, err := sackByteWalk(mf.f)
	if err != nil {
		return nil, err.Error()
	}
	writtenFile, err := sav.Open(written)
	if err != nil {
		return nil, err.Error()
	}
	after, err := sackByteWalk(writtenFile)
	if err != nil {
		return nil, err.Error()
	}
	lines := mismatch[writerCensusSackKey]
	sourceAnomalies := writerCensusSackRule(before)[writerCensusSackKey]
	if extra := writerCensusExtraMismatch(mismatch, writerCensusSackKey); extra != "" {
		return nil, fmt.Sprintf("source/produced T1C anomaly has extra mismatch key %s", extra)
	}
	if len(sourceAnomalies) != 5 || len(lines) != 4 {
		return nil, fmt.Sprintf("source/produced T1C anomaly counts changed: %d/%d", len(sourceAnomalies), len(lines))
	}
	type sackKey struct {
		cell [2]int32
		id   uint32
	}
	source := map[sackKey]*sackByteRecord{}
	for _, index := range before.indices() {
		r := before.rows[index]
		if r.class != "Sack" {
			continue
		}
		key := sackKey{writerCensusSackCell(r), r.values["RuntimeID"]}
		if source[key] != nil {
			return nil, fmt.Sprintf("source Sack key %+v repeated", key)
		}
		source[key] = r
	}
	seen := map[sackKey]bool{}
	for _, line := range lines {
		var archive uint16
		if n, err := fmt.Sscanf(line, "sack archive %d", &archive); err != nil || n != 1 {
			return nil, "produced T1C line has no archive index: " + line
		}
		got := after.rows[archive]
		if got == nil || got.class != "Sack" {
			return nil, "produced T1C line names no Sack: " + line
		}
		key := sackKey{writerCensusSackCell(got), got.values["RuntimeID"]}
		old := source[key]
		if old == nil || seen[key] || got.values["Identity"] != old.values["Identity"] || got.values["T1C"] != 0 || old.values["T1C"] != 0 ||
			got.values["S3C"] != old.values["S3C"] || len(got.refs["Contents"]) != len(old.refs["Contents"]) {
			return nil, "produced T1C anomaly differs from source Sack: " + line
		}
		seen[key] = true
		for i, ref := range got.refs["Contents"] {
			item, sourceItem := after.rows[ref], before.rows[old.refs["Contents"][i]]
			if item == nil || sourceItem == nil || item.class != sourceItem.class {
				return nil, "produced T1C anomaly differs from source contents: " + line
			}
			for _, name := range []string{"Identity", "F40", "F42", "T1C"} {
				if item.values[name] != sourceItem.values[name] {
					return nil, "produced T1C anomaly differs from source contents: " + line
				}
			}
		}
	}
	return map[string][]string{writerCensusSackKey: slices.Clone(lines)}, ""
}

// The census's part tests. Each checks its own files; the whole census runs
// in the part that completes the set (writerCensusChangedWorlds).
func TestSAVWriterCensusChangedWorldsPart1(t *testing.T) { writerCensusChangedWorlds(t, 0) }
func TestSAVWriterCensusChangedWorldsPart2(t *testing.T) { writerCensusChangedWorlds(t, 1) }
func TestSAVWriterCensusChangedWorldsPart3(t *testing.T) { writerCensusChangedWorlds(t, 2) }
func TestSAVWriterCensusChangedWorldsPart4(t *testing.T) { writerCensusChangedWorlds(t, 3) }
func TestSAVWriterCensusChangedWorldsPart5(t *testing.T) { writerCensusChangedWorlds(t, 4) }
func TestSAVWriterCensusChangedWorldsPart6(t *testing.T) { writerCensusChangedWorlds(t, 5) }

// writerCensusPartSummary is one part's tally and timing, as the part that
// completes the set reads it.
type writerCensusPartSummary struct {
	Discovered, Changed, Written                              int
	Refused, Mismatches, Known, Debt, Unnamed, Changes, Ruled map[string]int
	Wall, Sum, SlowestTook                                    time.Duration
	SlowestRel                                                string
}

// writerCensusChangedWorlds runs one part's share of the corpus, larger files
// spread first, and logs each file's lines. The part that completes the set
// adds every part's tally and checks the whole census, as one process over
// the whole corpus did. AGAINROM_CENSUS_ONLY narrows each part to the files
// whose path contains it and reports that part alone, unchecked.
func writerCensusChangedWorlds(t *testing.T, part int) {
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}
	only := os.Getenv("AGAINROM_CENSUS_ONLY")
	start := time.Now()
	all := writerCensusWalk(t, part == 0)
	names, sizes := make([]string, len(all)), make([]int64, len(all))
	for i, mf := range all {
		names[i], sizes[i] = mf.rel, int64(len(mf.raw))
	}
	share := corpusPartOf(names, sizes, writerCensusParts)
	var files []milestone2File
	matched := 0
	for i, mf := range all {
		if only == "" || strings.Contains(mf.rel, only) {
			matched++
			if share[i] == part {
				files = append(files, mf)
			}
		}
	}
	if matched == 0 {
		t.Fatal("writer census discovered no file; AGAINROM_SAVE_CORPUS is misconfigured")
	}
	results := writerCensusEach(t, files, func(t *testing.T, mf milestone2File) writerCensusResult {
		return writerCensusRun(t, assets, mf)
	})
	tally := newWriterCensusTally()
	var sum time.Duration
	for _, r := range results {
		tally.discovered++
		sum += r.took
		for _, c := range r.changes {
			tally.changes[c]++
		}
		for _, c := range r.changes {
			if !strings.HasPrefix(c, "not:") {
				tally.changed++
				break
			}
		}
		if r.err != nil {
			reason := r.stage + ": " + r.err.Error()
			tally.refused[reason]++
			t.Logf("SAV-WRITERCENSUS-REFUSED %s %s", r.rel, reason)
			continue
		}
		if r.knownIssue != "" {
			t.Errorf("SAV-WRITERCENSUS-KNOWN %s: %s", r.rel, r.knownIssue)
		}
		tally.written++
		for _, key := range writerCensusSortedLines(r.mismatch) {
			for _, line := range r.mismatch[key] {
				tally.mismatches[key]++
				t.Logf("SAV-WRITERCENSUS-MISMATCH %s %s: %s", r.rel, key, line)
				if slices.Contains(r.known[key], line) {
					tally.known[key]++
					cause := "original-source T1C anomaly; ITEM-SACK-010 conflict Unknown"
					if key != writerCensusSackKey {
						cause = "D56 file-bound writer debt"
					}
					t.Logf("SAV-WRITERCENSUS-KNOWN %s %s: %s; %s", r.rel, key, line, cause)
				}
			}
		}
		ruled, debt, unnamed := writerCensusClassify(r.fields, r.verified)
		for k, n := range ruled {
			tally.ruled[k] += n
		}
		for k, n := range debt {
			tally.debt[k] += n
		}
		for _, k := range writerCensusSorted(unnamed) {
			tally.unnamed[k] += unnamed[k]
			t.Logf("SAV-WRITERCENSUS-UNNAMED %s %s differs from the original %d time(s) with no confirmed rule and no debt row", r.rel, k, unnamed[k])
		}
	}
	mine := writerCensusPartSummary{Discovered: tally.discovered, Changed: tally.changed, Written: tally.written,
		Refused: tally.refused, Mismatches: tally.mismatches, Known: tally.known, Debt: tally.debt, Unnamed: tally.unnamed,
		Changes: tally.changes, Ruled: tally.ruled, Wall: time.Since(start), Sum: sum}
	for _, r := range results {
		if r.took > mine.SlowestTook {
			mine.SlowestTook, mine.SlowestRel = r.took, r.rel
		}
	}
	t.Logf("SAV-WRITERCENSUS-PART %d of %d: %d file(s) in %s", part+1, writerCensusParts, len(results), mine.Wall.Round(time.Second))
	if only != "" {
		writerCensusReport(t, tally, mine.Wall)
		return
	}
	set := corpusPartsCollect(t, "writercensus", part, writerCensusParts, mine)
	if set == nil {
		return
	}
	whole, slowest := newWriterCensusTally(), writerCensusPartSummary{}
	for _, p := range corpusPartsDecode[writerCensusPartSummary](t, set) {
		whole.discovered += p.Discovered
		whole.changed += p.Changed
		whole.written += p.Written
		for _, add := range []struct{ to, from map[string]int }{{whole.refused, p.Refused}, {whole.mismatches, p.Mismatches},
			{whole.known, p.Known}, {whole.debt, p.Debt}, {whole.unnamed, p.Unnamed}, {whole.changes, p.Changes}, {whole.ruled, p.Ruled}} {
			for k, n := range add.from {
				add.to[k] += n
			}
		}
		slowest.Sum += p.Sum
		slowest.Wall = max(slowest.Wall, p.Wall)
		if p.SlowestTook > slowest.SlowestTook {
			slowest.SlowestTook, slowest.SlowestRel = p.SlowestTook, p.SlowestRel
		}
	}
	t.Logf("SAV-WRITERCENSUS-TIME wall=%s per-file sum=%s slowest=%s %s workers=%d parts=%d",
		slowest.Wall.Round(time.Second), slowest.Sum.Round(time.Second), slowest.SlowestTook.Round(100*time.Millisecond), slowest.SlowestRel, writerCensusWorkers, writerCensusParts)
	writerCensusReport(t, whole, slowest.Wall)
	if whole.known[writerCensusSackKey] != 4 || whole.known[writerCensusItemKey] != 0 || whole.known[writerCensusMissingKey] != 0 {
		t.Errorf("SAV-WRITERCENSUS-KNOWN expected source T1C 4 and item/missing 0/0; got %d/%d/%d",
			whole.known[writerCensusSackKey], whole.known[writerCensusItemKey], whole.known[writerCensusMissingKey])
	}
	for _, f := range writerCensusFailures(whole) {
		t.Error(f)
	}
}

func writerCensusSorted(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writerCensusSortedLines(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writerCensusReport(t *testing.T, tally *writerCensusTally, took time.Duration) {
	t.Helper()
	refused, mismatched, unnamed := 0, 0, 0
	for _, n := range tally.refused {
		refused += n
	}
	for _, n := range tally.mismatches {
		mismatched += n
	}
	for _, n := range tally.unnamed {
		unnamed += n
	}
	for _, k := range writerCensusSorted(tally.changes) {
		t.Logf("SAV-WRITERCENSUS-CHANGE %s=%d", k, tally.changes[k])
	}
	for _, k := range writerCensusSorted(tally.refused) {
		t.Logf("SAV-WRITERCENSUS-REASON %d: %s (%s)", tally.refused[k], k, writerCensusBaseline[k].cause)
	}
	for _, k := range writerCensusSorted(tally.mismatches) {
		t.Logf("SAV-WRITERCENSUS-FIELD %s=%d known=%d unexpected=%d (%s)", k, tally.mismatches[k], tally.known[k], tally.mismatches[k]-tally.known[k], writerCensusBaseline[k].cause)
	}
	for _, k := range writerCensusSorted(tally.ruled) {
		t.Logf("SAV-WRITERCENSUS-RULED %s=%d confirmed by %s", k, tally.ruled[k], writerCensusRules[k])
	}
	for _, k := range writerCensusSorted(tally.debt) {
		t.Logf("SAV-WRITERCENSUS-DEBT %s=%d: %s", k, tally.debt[k], writerCensusDebt[k])
	}
	t.Logf("SAV-WRITERCENSUS-CENSUS discovered=%d changed=%d written=%d refused=%d mismatched=%d known=%d unexpected=%d unnamed=%d runtime=%s",
		tally.discovered, tally.changed, tally.written, refused, mismatched, tally.known[writerCensusSackKey]+tally.known[writerCensusItemKey]+tally.known[writerCensusMissingKey],
		mismatched-tally.known[writerCensusSackKey]-tally.known[writerCensusItemKey]-tally.known[writerCensusMissingKey], unnamed, took.Round(time.Second))
}

// writerCensusFailures returns every way a tally breaks the census: a
// refusal or mismatch count above its baseline ceiling, a change below its
// floor, or an unnamed differing field.
func writerCensusFailures(tally *writerCensusTally) []string {
	var out []string
	for _, k := range writerCensusSorted(tally.refused) {
		if row := writerCensusBaseline[k]; tally.refused[k] > row.ceiling {
			out = append(out, fmt.Sprintf("SAV-WRITERCENSUS-RISE %q now %d, baseline %d", k, tally.refused[k], row.ceiling))
		}
	}
	for _, k := range writerCensusSorted(tally.mismatches) {
		unexpected := tally.mismatches[k] - tally.known[k]
		if row := writerCensusBaseline[k]; unexpected < 0 || unexpected > row.ceiling {
			out = append(out, fmt.Sprintf("SAV-WRITERCENSUS-RISE %q now %d unexpected of %d total, baseline %d", k, unexpected, tally.mismatches[k], row.ceiling))
		}
	}
	for _, k := range writerCensusSorted(writerCensusChangeFloor) {
		if n := tally.changes[k]; n < writerCensusChangeFloor[k] {
			out = append(out, fmt.Sprintf("SAV-WRITERCENSUS-FLOOR change %q reached %d file(s), floor %d", k, n, writerCensusChangeFloor[k]))
		}
	}
	for _, k := range writerCensusSorted(tally.unnamed) {
		out = append(out, fmt.Sprintf("SAV-WRITERCENSUS-UNNAMED %s differs %d time(s) with no confirmed rule and no debt row", k, tally.unnamed[k]))
	}
	return out
}

// TestSAVWriterCensusReference applies the census's rules to every world
// original, the original client's own writing, as a check that each rule is
// read correctly. A break is logged; the owner's settling save
// game0002-bigsack.sav must pass every rule, and the corpus must hold world
// originals.
func TestSAVWriterCensusReference(t *testing.T) {
	results := writerCensusEach(t, writerCensusFiles(t), func(t *testing.T, mf milestone2File) writerCensusResult {
		if !mf.present {
			return writerCensusResult{town: true}
		}
		src, err := sackByteWalk(mf.f)
		if err != nil {
			return writerCensusResult{stage: "sack walk", err: err}
		}
		stacked := 0
		for _, index := range src.indices() {
			if r := src.rows[index]; r.class == "Sack" {
				for _, ref := range r.refs["Contents"] {
					if item := src.rows[ref]; item != nil && item.values["F42"] > 1 {
						stacked++
					}
				}
			}
		}
		store, _ := mf.f.StateStore()
		rules := writerCensusSackRule(src)
		rules[writerCensusViewKey] = writerCensusViewFloorRule(writerCensusLeaves(store))
		return writerCensusResult{mismatch: rules, fields: map[string]int{"stacked": stacked}}
	})
	counts := map[string]int{}
	worlds, stacked := 0, 0
	for _, r := range results {
		if r.err != nil {
			t.Errorf("%s: %s: %v", r.rel, r.stage, r.err)
			continue
		}
		if r.town {
			continue
		}
		worlds++
		stacked += r.fields["stacked"]
		for _, key := range writerCensusSortedLines(r.mismatch) {
			for _, line := range r.mismatch[key] {
				counts[key]++
				t.Logf("SAV-WRITERCENSUS-REFERENCE %s %s: %s", r.rel, key, line)
				if strings.HasSuffix(r.rel, "game0002-bigsack.sav") {
					t.Errorf("the owner's original-written reference breaks %s", key)
				}
			}
		}
	}
	if worlds == 0 {
		t.Fatal("the reference found no world original; AGAINROM_SAVE_CORPUS is misconfigured")
	}
	t.Logf("SAV-WRITERCENSUS-REFERENCE-CENSUS world originals=%d stacked items in sacks=%d breaks=%v", worlds, stacked, counts)
}

// writerCensusRewrite decodes a written file, applies edit, and encodes it
// again, standing in for a writer with one fix disabled.
func writerCensusRewrite(t *testing.T, raw []byte, edit func(*sav.DocumentData) error) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := edit(&doc); err != nil {
		t.Fatal(err)
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func writerCensusSetState(d *sav.DocumentData, path string, v int32) error {
	for i := range d.State.ValueRecords {
		if d.State.ValueRecords[i].Path == path {
			d.State.ValueRecords[i].Value.Int32 = v
			return nil
		}
	}
	return fmt.Errorf("no %s leaf", path)
}

// TestSAVWriterCensusSensitivity writes game0002-bigsack.sav once through
// the census's route and checks the unedited write and one edited copy per
// writer gap. Each edit reproduces a gap the way an unfixed writer wrote it,
// and each must be caught by the rule or field class named for it.
func TestSAVWriterCensusSensitivity(t *testing.T) {
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}
	// Read directly, failing rather than skipping: this family runs only
	// where its gate supplies the corpus.
	payload, err := os.ReadFile(filepath.Join(os.Getenv("AGAINROM_SAVE_CORPUS"), "2026-09-24", "game0002-bigsack.sav"))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(payload)); got != "8842212e16cbb97ad05af059bdd637df2ab777b4c7f5bb5430353b68b5312c51" {
		t.Fatalf("the owner's settling save changed: %s", got)
	}
	f, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	mf := milestone2File{rel: "2026-09-24/game0002-bigsack.sav", raw: payload, body: f.Body, f: f, present: f.World != nil}
	w, stage, err := writerCensusWriteOne(t, assets, mf)
	if err != nil {
		t.Fatalf("%s: %v", stage, err)
	}
	for _, c := range []string{"camera", "drop-new-sack", "kill-full-decay"} {
		if !strings.Contains(strings.Join(w.changes, " "), c) {
			t.Fatalf("the settling save's change did not reach %s: %v", c, w.changes)
		}
	}
	sackValue := func(d *sav.DocumentData, pick func(r *sav.DocumentRecordData) bool, set func(r *sav.DocumentRecordData) error) error {
		n := 0
		for i := range d.Objects {
			if pick(&d.Objects[i]) {
				n++
				if err := set(&d.Objects[i]); err != nil {
					return err
				}
			}
		}
		if n == 0 {
			return fmt.Errorf("no record picked")
		}
		return nil
	}
	isSack := func(r *sav.DocumentRecordData) bool { return r.Class == "Sack" }
	for _, tc := range []struct {
		name string
		// mismatch are the rule keys that must catch the gap; unnamed the
		// fields that must fail as unnamed. Nothing else may fire.
		mismatch, unnamed []string
		edit              func(*sav.DocumentData) error
	}{
		{"unedited", nil, nil, nil},
		{"sack value zero", []string{writerCensusSackKey}, []string{"sack:T1C"}, func(d *sav.DocumentData) error {
			return sackValue(d, isSack, func(r *sav.DocumentRecordData) error { return savedStructureSetValue(r, "T1C", 0) })
		}},
		{"contained item counts lowered", nil, []string{"sack-item:F42"}, func(d *sav.DocumentData) error {
			return sackValue(d, func(r *sav.DocumentRecordData) bool { return writerCensusItemClasses[r.Class] }, func(r *sav.DocumentRecordData) error {
				return savedStructureSetValue(r, "F42", 0)
			})
		}},
		{"item prices zeroed consistently", nil, []string{"sack:T1C", "sack-item:T1C"}, func(d *sav.DocumentData) error {
			if err := sackValue(d, func(r *sav.DocumentRecordData) bool { return writerCensusItemClasses[r.Class] }, func(r *sav.DocumentRecordData) error {
				return savedStructureSetValue(r, "T1C", 0)
			}); err != nil {
				return err
			}
			return sackValue(d, isSack, func(r *sav.DocumentRecordData) error {
				gold, err := savedStructureValue(r, "S3C")
				if err != nil {
					return err
				}
				return savedStructureSetValue(r, "T1C", gold)
			})
		}},
		{"original sack re-identified as runtime id 0", []string{writerCensusMissingKey, writerCensusIDKey}, nil, func(d *sav.DocumentData) error {
			return sackValue(d, func(r *sav.DocumentRecordData) bool { return r.Class == "Sack" && writerCensusRuntimeID(r) == 10 }, func(r *sav.DocumentRecordData) error {
				gold, err := savedStructureValue(r, "S3C")
				if err != nil {
					return err
				}
				value, err := savedStructureValue(r, "T1C")
				if err != nil {
					return err
				}
				for _, set := range []struct {
					name  string
					value uint32
				}{{"RuntimeID", 0}, {"S3C", 0}, {"T1C", value - gold}} {
					if err := savedStructureSetValue(r, set.name, set.value); err != nil {
						return err
					}
				}
				return nil
			})
		}},
		{"constructed sack with runtime id 0 and mask 0", []string{writerCensusIDKey, writerCensusMaskKey}, nil, func(d *sav.DocumentData) error {
			origIDs := map[uint32]bool{}
			src, err := sackByteWalk(mf.f)
			if err != nil {
				return err
			}
			for _, r := range src.rows {
				if r.class == "Sack" {
					origIDs[r.values["RuntimeID"]] = true
				}
			}
			return sackValue(d, func(r *sav.DocumentRecordData) bool { return r.Class == "Sack" && !origIDs[writerCensusRuntimeID(r)] }, func(r *sav.DocumentRecordData) error {
				if err := savedStructureSetValue(r, "RuntimeID", 0); err != nil {
					return err
				}
				return savedStructureSetValue(r, "T18", 0)
			})
		}},
		{"view row seven", []string{writerCensusViewKey}, []string{"registry:/View/Y"}, func(d *sav.DocumentData) error {
			return writerCensusSetState(d, "/View/Y", 7)
		}},
		{"view carried from the loaded file", []string{writerCensusViewKey}, nil, func(d *sav.DocumentData) error {
			store, _ := mf.f.StateStore()
			for _, axis := range []string{"/View/X", "/View/Y"} {
				n := writerCensusNode(store, axis)
				if n == nil {
					return fmt.Errorf("no loaded %s", axis)
				}
				if err := writerCensusSetState(d, axis, n.Int); err != nil {
					return err
				}
			}
			return nil
		}},
		{"an option with no rule", nil, []string{"registry:/GameOptions/Speed"}, func(d *sav.DocumentData) error {
			store, _ := mf.f.StateStore()
			n := writerCensusNode(store, "/GameOptions/Speed")
			if n == nil {
				return fmt.Errorf("no loaded /GameOptions/Speed")
			}
			return writerCensusSetState(d, "/GameOptions/Speed", n.Int^1)
		}},
		{"fog plane re-fogged", []string{writerCensusFogKey}, nil, func(d *sav.DocumentData) error {
			for i := range d.State.ValueRecords {
				if r := &d.State.ValueRecords[i]; r.Path == "/Fog/Data" {
					total := uint32(0)
					for i := 0; i+4 <= len(r.Value.Bytes); i += 4 {
						total += binary.LittleEndian.Uint32(r.Value.Bytes[i:])
					}
					// Every cell unexplored: one run in the first state when
					// that state is unexplored, else an empty first run.
					r.Value.Bytes = binary.LittleEndian.AppendUint32(nil, total)
					for _, f := range d.State.ValueRecords {
						if f.Path == "/Fog/FirstState" && f.Value.Int32 != 0 {
							r.Value.Bytes = binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(nil, 0), total)
						}
					}
					return nil
				}
			}
			return fmt.Errorf("no /Fog/Data leaf")
		}},
		{"party restated into the roster", []string{writerCensusRosterKey}, nil, func(d *sav.DocumentData) error {
			b, present, err := sav.NativeActions(d.State)
			if err != nil || !present {
				return fmt.Errorf("party supplement absent: %v", err)
			}
			var a map[string]json.RawMessage
			if err := json.Unmarshal(b, &a); err != nil {
				return err
			}
			var party, roster []json.RawMessage
			if err := json.Unmarshal(a["Party"], &party); err != nil {
				return err
			}
			if len(a["Roster"]) > 0 && string(a["Roster"]) != "null" {
				if err := json.Unmarshal(a["Roster"], &roster); err != nil {
					return err
				}
			}
			if a["Roster"], err = json.Marshal(append(roster, party...)); err != nil {
				return err
			}
			if b, err = json.Marshal(a); err != nil {
				return err
			}
			return sav.SetNativeActions(&d.State, b)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := w.raw
			if tc.edit != nil {
				raw = writerCensusRewrite(t, w.raw, tc.edit)
			}
			mismatch, fields, verified, err := writerCensusCheck(mf.f, mf.raw, w, raw)
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range writerCensusSortedLines(mismatch) {
				if lines := mismatch[key]; len(lines) > 0 && !slices.Contains(tc.mismatch, key) {
					t.Errorf("unexpected %s: %v", key, lines)
				}
			}
			for _, key := range tc.mismatch {
				if len(mismatch[key]) == 0 {
					t.Errorf("the census did not catch the gap under %s", key)
				} else {
					t.Logf("SAV-WRITERCENSUS-SENSITIVE %s caught under %s: %s", tc.name, key, mismatch[key][0])
				}
			}
			ruled, debt, unnamed := writerCensusClassify(fields, verified)
			for _, k := range writerCensusSorted(unnamed) {
				if !slices.Contains(tc.unnamed, k) {
					t.Errorf("unexpected unnamed field %s", k)
				}
			}
			for _, k := range tc.unnamed {
				if unnamed[k] == 0 {
					t.Errorf("the census did not fail %s as unnamed", k)
				} else {
					t.Logf("SAV-WRITERCENSUS-SENSITIVE %s fails field %s", tc.name, k)
				}
			}
			if tc.edit == nil {
				// The unedited write passes, and it passes because the
				// camera rule confirmed the changed view and every sack value
				// rests on original item records, not because nothing changed.
				for _, k := range []string{"registry:/View/X", "registry:/View/Y"} {
					if ruled[k] == 0 {
						t.Errorf("the unedited write did not confirm %s by rule: ruled %v", k, ruled)
					}
				}
				if !verified["sack:T1C"] {
					t.Errorf("the unedited write's sack values do not rest on original item records")
				}
				for k := range debt {
					if writerCensusDebt[k] == "" {
						t.Errorf("debt field %s has no debt row", k)
					}
				}
				t.Logf("SAV-WRITERCENSUS-SENSITIVE unedited passes: ruled %v, debt %v", ruled, debt)
			}
		})
	}
}

// TestSAVWriterCensusFailures proves the census's enforcement returns a
// failure for each way a tally can break it: a mismatch or refusal with no
// baseline row, a change below its floor, and an unnamed field.
func TestSAVWriterCensusFailures(t *testing.T) {
	if got := writerCensusExtraMismatch(map[string][]string{writerCensusSackKey: {"known"}, "empty": nil}, writerCensusSackKey); got != "" {
		t.Fatalf("empty mismatch category counted as extra: %s", got)
	}
	if got := writerCensusExtraMismatch(map[string][]string{writerCensusSackKey: {"known"}, "extra": {"new"}}, writerCensusSackKey); got != "extra" {
		t.Fatalf("extra mismatch category escaped: %s", got)
	}
	passing := func() *writerCensusTally {
		tally := newWriterCensusTally()
		for k, n := range writerCensusChangeFloor {
			tally.changes[k] = n
		}
		return tally
	}
	if got := writerCensusFailures(passing()); len(got) != 0 {
		t.Fatalf("a tally at every floor with nothing unnamed fails: %v", got)
	}
	known := passing()
	for key, count := range map[string]int{writerCensusSackKey: 4} {
		known.mismatches[key], known.known[key] = count, count
	}
	if got := writerCensusFailures(known); len(got) != 0 {
		t.Fatalf("the four exact, file-bound observations fail: %v", got)
	}
	known.mismatches[writerCensusSackKey]++
	if got := writerCensusFailures(known); len(got) != 1 || !strings.Contains(got[0], writerCensusSackKey) {
		t.Fatalf("an extra T1C mismatch escaped: %v", got)
	}
	known.mismatches[writerCensusSackKey]--
	known.mismatches["new mismatch key"] = 1
	if got := writerCensusFailures(known); len(got) != 1 || !strings.Contains(got[0], "new mismatch key") {
		t.Fatalf("a new mismatch key escaped: %v", got)
	}
	for _, tc := range []struct {
		name, want string
		edit       func(*writerCensusTally)
	}{
		{"mismatch", "SAV-WRITERCENSUS-RISE", func(x *writerCensusTally) { x.mismatches[writerCensusSackKey] = 1 }},
		{"refusal", "SAV-WRITERCENSUS-RISE", func(x *writerCensusTally) { x.refused["SAVE-PREPARE: refused"] = 1 }},
		{"floor", "SAV-WRITERCENSUS-FLOOR", func(x *writerCensusTally) { x.changes["pickup"]-- }},
		{"unnamed", "SAV-WRITERCENSUS-UNNAMED", func(x *writerCensusTally) { x.unnamed["registry:/GameOptions/Speed"] = 1 }},
	} {
		tally := passing()
		tc.edit(tally)
		got := writerCensusFailures(tally)
		if len(got) != 1 || !strings.HasPrefix(got[0], tc.want) {
			t.Errorf("%s: failures %v, want one %s", tc.name, got, tc.want)
		}
	}
	ruled, debt, unnamed := writerCensusClassify(map[string]int{"registry:/View/X": 1, "registry:/View/Y": 1, "registry:/Fog/Data": 1, "registry:/GameOptions/Speed": 1},
		map[string]bool{"registry:/View/X": true})
	if ruled["registry:/View/X"] != 1 || debt["registry:/Fog/Data"] != 1 || unnamed["registry:/View/Y"] != 1 || unnamed["registry:/GameOptions/Speed"] != 1 || len(ruled)+len(debt)+len(unnamed) != 4 {
		t.Errorf("classification: ruled %v debt %v unnamed %v", ruled, debt, unnamed)
	}
	for _, k := range []string{"town:/GameOptions/FlyingHP", "town:/GameOptions/ShowHP", "town:/GameOptions/ShowTimeFlow",
		"town:/GameOptions/Speed", "town:/Inventory/IsOpen", "town:/SpellBook/Pressed", "town:/View/X", "town:/View/Y",
		"registry:/GameOptions/Speed", "registry:/View/X", "sack:T1C", "sack-item:T1C"} {
		if writerCensusDebt[k] != "" {
			t.Errorf("%s is a debt row; the writer takes it from current state or the loaded file", k)
		}
	}
}
