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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/intmath"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// writerCensusViewFloor is SESS-VIEW-030's lower scroll bound: the origin of
// a world camera never sits in the outermost eight cells.
const writerCensusViewFloor = 8

// writerCensusDecayTicks bounds how long a killed actor is advanced toward
// its terminal stage; the full-decay regression test uses the same span.
const writerCensusDecayTicks = 24000

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

// writerCensusFiles walks AGAINROM_SAVE_CORPUS with the milestone-2 walker's
// walk and returns every readable .sav, logging each unreadable one.
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
	err := walkFirstGameSaves(corpus, func(path, rel string) error {
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

// writerCensusSacks compares the written Sacks with the original's. A sack
// joins the original of its cell and runtime id field by field. Each item
// record a sack holds is compared with the original of its class and Identity
// key; a record with a new key must be split off an original stack whose
// other fields it repeats, with the count conserved. An original sack absent
// from the written file is a mismatch unless the party picked it up; the
// pick-up freed its runtime id (SAV-1370), so any written sack in its cell has
// no original. A written sack with no original is constructed by the session
// and must carry a nonzero runtime id no other record holds (MOVE-ID-016) and
// mask 2 at Token+0x18 (SAV-655). verified records the item fields a rule
// confirmed and whether each ITEM-SACK-010 value rests on original item
// values.
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
	// The original's Reference misses: a key no record at or before the
	// record holds as Identity or This.
	missed, seen := map[string]bool{}, map[string]bool{}
	for i := range origDoc.Objects {
		values := writerCensusRecordValues(&origDoc.Objects[i])
		for _, name := range []string{"Identity", "This"} {
			if v, ok := values[name]; ok && v != "0" {
				seen[v] = true
			}
		}
		if ref := values["Reference"]; writerCensusItemClasses[origDoc.Objects[i].Class] && ref != "" && ref != "0" && !seen[ref] {
			missed[origDoc.Objects[i].Class+" "+values["Identity"]] = true
		}
	}
	referenceExact := true
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
		if old := origSacks[k]; old != nil && !(w.pickedDone && k.cell == w.picked) {
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
				case "Reference":
					referenceExact = referenceExact && v == "0" && missed[fmt.Sprint(item.class, " ", item.values["Identity"])]
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
	verified["sack-item:Reference"] = referenceExact
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
			if d := max(intmath.Abs(sk.X-e.X), intmath.Abs(sk.Y-e.Y)); bestD < 0 || d < bestD {
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
