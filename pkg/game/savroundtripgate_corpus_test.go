//go:build sessioncorpusaudit

package game

// The SAV round-trip gate: GAME, then SAV, then LOAD, then GAME, over the
// save points this project reaches without a window. A mission copy plays
// (ticks, a ground drop, kills, decay) and saves through the player's own SAVE
// dialog route; a town saves through the same route off the map. A second
// front end loads that file. The two must then agree on the hashed World, on
// what the player sees, and over a common advance; a town must agree on its
// whole snapshot.
//
// A file that cannot complete the cycle is a named REFUSAL by stage and
// reason, never a skip. A disagreement is a named MISMATCH by field path. The
// baseline under testdata records, per case, its outcome kind and every key it
// may carry; a key a case did not carry before fails.

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"image"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
	"unsafe"

	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	// savGateKills is how many non-party actors play kills before saving.
	savGateKills = 5
	// savGateDecayCap bounds the ticks play waits for those bodies to decay
	// fully; a body still decaying at the cap is saved mid-decay.
	savGateDecayCap = 24000
	// savGateKitCycles is how many SAVE and LOAD rounds a played owner kit
	// runs: the owner's own sequence of loading a file that already carries
	// the dead, killing more, saving and loading, repeated once more. The
	// corpus runs one round.
	savGateKitCycles = 3
	// savGateCycleKills is how many actors each cycled round kills, enough
	// that three rounds leave dozens of dead, as the owner's own saves carry.
	savGateCycleKills = 12
	// savGateAdvance is the common advance both copies run after LOAD.
	savGateAdvance = 120
	// savGateDecaySample: one corpus file in this many, chosen by a hash of
	// its relative path, plays to full decay; the rest play the dying route.
	savGateDecaySample = 4
	// savGateDyingTicks is how long the dying route advances after its kills:
	// the bodies are saved mid-decay, about an eighth of the full span.
	savGateDyingTicks = 2400
	// savGateWorkers bounds how many cases run at once in one process.
	savGateWorkers = 4
)

// savGateCase is one save point: an original or kit file, or a New Game at a
// chosen mission and tick count.
type savGateCase struct {
	population, name string
	raw              []byte
	// town is a between-mission save: it is restored and saved off the map.
	town            bool
	mission, ticks  int
	kill, spareZero bool
	noDrop          bool
	// dying saves savGateDyingTicks after the kills instead of waiting for
	// full decay.
	dying bool
	// kills overrides savGateKills when positive.
	kills int
	// cycles is how many SAVE and LOAD rounds the case runs; after the first,
	// the loaded copy kills more and saves again. Zero means one.
	cycles int
}

// savGateOutcome is what one case produced. A refusal in a later round keeps
// the mismatches of the rounds before it.
type savGateOutcome struct {
	refusal                string
	unreadable             bool
	mismatches, transients []string
}

// kind names the outcome as the baseline records it.
func (o savGateOutcome) kind() string {
	switch {
	case o.unreadable:
		return "unreadable"
	case o.refusal != "":
		return "refused"
	case len(o.mismatches) > 0:
		return "mismatched"
	}
	return "accepted"
}

// keys are the census keys the outcome carries, sorted and unique.
func (o savGateOutcome) keys() []string {
	seen := map[string]bool{}
	if o.unreadable {
		seen["unreadable original"] = true
	}
	if o.refusal != "" {
		seen["refused "+savGateDigits(o.refusal)] = true
	}
	for _, m := range o.mismatches {
		seen["mismatch "+m] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// savGateCauses names why each baselined key is expected today, by key
// prefix; the longest prefix wins. A baselined key no prefix names fails, so
// no recorded defect goes unnamed. The story document carries the evidence.
var savGateCauses = map[string]string{
	"unreadable original": "EXP-0261-owner-runs/game9000.sav is not a SAV container (magic Bsg&)",
	"refused load restore: sim: restored current actions: saved actor cell ":                               "new owner SAVs game0005/game0006 in oldsaves7 have typed actor cell slots our LOAD refuses; the source and needed reader rule are Unknown (D54)",
	"refused cycle save: project current holdings: current actor manifest has a repeated or absent source": "after a SAVE, LOAD and more kills, SAVE refuses in the actor manifest",
	"mismatch visible.scene.frame":          "fallen actors restart their presentation death clock after LOAD; the view scene clock now resumes from World.Tick",
	"mismatch cycle.visible.scene.frame":    "the same fallen-actor death clock restarts after the next SAVE and LOAD",
	"mismatch visible.unit.body":            "a killed human mid-decay draws unarmed after LOAD; a party member that dropped a worn item draws unarmed after LOAD",
	"mismatch visible.scene.art":            "the same body change, as the scene draws it",
	"mismatch advanced.visible.unit.body":   "the body change persists over the advance",
	"mismatch advanced.visible.scene.art":   "the body change persists over the advance",
	"mismatch world.":                       "hashed state the round trip loses; the story document names each field path's cause",
	"mismatch world.savedGroups.Orders.Raw": "cold LOAD changes retained actor order bytes where a queued victim differs from the active victim, DIV-2352; the field rule remains open",
	"mismatch advanced.world.":              "the same hashed loss after the common advance",
	"mismatch cycle.world.":                 "the same hashed loss in later rounds",
	"mismatch cycle.advanced.world.":        "the same hashed loss in later rounds",
	"mismatch town.":                        "town state the round trip loses; the story document names each field path's cause",
}

// savGateCause returns the cause of the longest prefix naming key.
func savGateCause(key string) (string, bool) {
	best := -1
	cause := ""
	for p, c := range savGateCauses {
		if strings.HasPrefix(key, p) && len(p) > best {
			best, cause = len(p), c
		}
	}
	return cause, best >= 0
}

// savGateDigits normalises a reason so one defect over many files is one key.
func savGateDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return '#'
		}
		return r
	}, s)
}

var savGateCastagnoli = crc32.MakeTable(crc32.Castagnoli)

func savGateImageHash(img image.Image) string {
	if img == nil || reflect.ValueOf(img).IsNil() {
		return "none"
	}
	b := img.Bounds()
	if rgba, ok := img.(*image.RGBA); ok {
		ieee := crc32.ChecksumIEEE([]byte(fmt.Sprintf("%v", b)))
		castagnoli := crc32.Checksum([]byte(fmt.Sprintf("%v", b)), savGateCastagnoli)
		for y := 0; y < b.Dy(); y++ {
			row := rgba.Pix[y*rgba.Stride : y*rgba.Stride+b.Dx()*4]
			ieee = crc32.Update(ieee, crc32.IEEETable, row)
			castagnoli = crc32.Update(castagnoli, savGateCastagnoli, row)
		}
		return fmt.Sprintf("%08x%08x", ieee, castagnoli)
	}
	h := fnv.New64a()
	fmt.Fprintf(h, "%v", b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			h.Write([]byte{byte(r >> 8), byte(g >> 8), byte(bl >> 8), byte(a >> 8)})
		}
	}
	return fmt.Sprintf("%016x", h.Sum64())
}

func savGateFrameHash(fr *terrain.StaticFrame) string {
	if fr == nil {
		return "none"
	}
	b := make([]byte, 0, 24+2*len(fr.Pixels)+4*len(fr.Palette))
	for _, n := range []int{fr.Width, fr.Height, len(fr.Pixels)} {
		b = append(b, byte(n), byte(n>>8), byte(n>>16), byte(n>>24), byte(n>>32), byte(n>>40), byte(n>>48), byte(n>>56))
	}
	for _, px := range fr.Pixels {
		opaque := byte(0)
		if px.Opaque {
			opaque = 1
		}
		b = append(b, px.Index, opaque)
	}
	for _, c := range fr.Palette {
		b = append(b, c.R, c.G, c.B, c.A)
	}
	h := fnv.New64a()
	h.Write(b)
	return fmt.Sprintf("%016x", h.Sum64())
}

// savGateArt names a unit's drawn art by what the player can tell apart: a
// party body or a placed class, and its name. Pointers never cross two front
// ends, so identity is never compared.
func savGateArt(units *terrain.UnitSet, class *terrain.UnitClass) string {
	if class == nil {
		return "none"
	}
	if units != nil {
		for _, body := range units.Bodies {
			if body == class {
				return "body " + class.Name
			}
		}
	}
	return "class " + class.Name
}

// savGateVisible is what the player sees, keyed "<field> <subject>". The
// field half is the census field. The portrait is the picture production
// draws, keyed by the entity's class.
func savGateVisible(a *ui.App, mw *mapWorld) map[string]string {
	out := map[string]string{}
	for _, e := range mw.world.Entities() {
		key := fmt.Sprintf(" %d", e.ID)
		out["unit.body"+key] = savGateArt(mw.units, mw.art[e.ID])
		out["unit.portrait"+key] = savGateImageHash(mw.unitPicture(e.ID, e.Class))
		out["unit.hover"+key] = savGateImageHash(mw.inspectionUnitPicture(uint32(e.ID)))
		if fig, ok := mw.figures[e.ID]; ok {
			out["unit.figure"+key] = fmt.Sprintf("%+v", fig)
		}
	}
	for _, d := range mw.entityDraws() {
		key := fmt.Sprintf(" %d", d.ID)
		out["unit.name"+key] = d.Name
		out["unit.nameIndex"+key] = fmt.Sprint(d.UnitNameIndex)
		out["scene.cell"+key] = fmt.Sprint(d.Cell, d.Step)
		out["scene.art"+key] = savGateArt(mw.units, d.Art)
		out["scene.frame"+key] = fmt.Sprint(d.DrawCategory, d.Mirror, savGateFrameHash(d.Frame))
		out["scene.state"+key] = fmt.Sprint(d.Stone, d.Translucent)
	}
	for _, s := range mw.sackDraws() {
		out[fmt.Sprintf("sack.frame %v", s.Cell)] = fmt.Sprint(s.FrameIndex)
	}
	if cam := mw.view.Camera(); cam != nil {
		out["camera.x"] = fmt.Sprint(cam.X)
		out["camera.y"] = fmt.Sprint(cam.Y)
		out["camera.zoom"] = fmt.Sprint(cam.Zoom)
	}
	if pix, _, err := a.HeadlessCharacterPane(); err != nil {
		out["frame.pane"] = "refused: " + err.Error()
	} else {
		out["frame.pane"] = savGateImageHash(pix)
	}
	return out
}

// savGateTransient names the visible fields a LOAD resets by construction.
// A unit caught mid-step draws interpolated from the cell it left and in its
// walk phase; the view records that previous cell once per tick and no save
// carries it, so after LOAD the unit draws on its cell until the next tick
// moves it. The same comparison after the common advance allows nothing.
var savGateTransient = map[string]string{
	"scene.cell":  "mid-step interpolation offset: the pre-step cell is view-only and no save carries it",
	"scene.frame": "walk phase of a unit mid-step: derived from the same view-only pre-step cell",
}

// savGateCompareVisible reports every visible key that differs, one call per
// field. A key of a unit in moving whose field savGateTransient names goes to
// transient instead, with its reason.
func savGateCompareVisible(va, vb map[string]string, moving map[string]bool, mismatch, transient func(field, detail string)) {
	fields := map[string][]string{}
	transients := map[string][]string{}
	note := func(k, line string) {
		parts := strings.SplitN(k, " ", 2)
		if len(parts) == 2 && moving[parts[1]] && savGateTransient[parts[0]] != "" && transient != nil {
			transients[parts[0]] = append(transients[parts[0]], line)
			return
		}
		fields[parts[0]] = append(fields[parts[0]], line)
	}
	for k, want := range va {
		if got, ok := vb[k]; !ok || got != want {
			note(k, fmt.Sprintf("%s: %q before, %q after", k, want, got))
		}
	}
	for k := range vb {
		if _, ok := va[k]; !ok {
			note(k, k+": absent before SAVE, present after LOAD")
		}
	}
	emit := func(m map[string][]string, to func(field, detail string), reason func(string) string) {
		names := make([]string, 0, len(m))
		for f := range m {
			names = append(names, f)
		}
		sort.Strings(names)
		for _, field := range names {
			lines := m[field]
			sort.Strings(lines)
			detail := strings.Join(lines, "; ")
			if len(lines) > 4 {
				detail = strings.Join(lines[:4], "; ") + fmt.Sprintf("; and %d more", len(lines)-4)
			}
			to("visible."+field, reason(field)+detail)
		}
	}
	emit(fields, mismatch, func(string) string { return "" })
	emit(transients, transient, func(f string) string { return savGateTransient[f] + "; " })
}

// savGateDiffer walks two values of one type, unexported fields included,
// and reports each leaf path whose value differs. Slice and map indices are
// dropped from the path, so one loss over many elements is one path; the
// first detail seen for a path is kept. Functions and channels are not state.
type savGateDiffer struct {
	seen map[[2]uintptr]bool
	add  func(path, detail string)
}

func (d *savGateDiffer) walk(path string, a, b reflect.Value, depth int) {
	if depth > 32 {
		return
	}
	if a.IsValid() != b.IsValid() {
		d.add(path, "present on one side only")
		return
	}
	if !a.IsValid() {
		return
	}
	if a.Type() != b.Type() {
		d.add(path, fmt.Sprintf("type %v before, %v after", a.Type(), b.Type()))
		return
	}
	leaf := func(equal bool) {
		if !equal {
			d.add(path, savGateBrief(fmt.Sprint(a))+" before, "+savGateBrief(fmt.Sprint(b))+" after")
		}
	}
	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				d.add(path, fmt.Sprintf("nil %t before, nil %t after", a.IsNil(), b.IsNil()))
			}
			return
		}
		k := [2]uintptr{a.Pointer(), b.Pointer()}
		if d.seen[k] {
			return
		}
		d.seen[k] = true
		d.walk(path, a.Elem(), b.Elem(), depth+1)
	case reflect.Interface:
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				d.add(path, fmt.Sprintf("nil %t before, nil %t after", a.IsNil(), b.IsNil()))
			}
			return
		}
		d.walk(path, a.Elem(), b.Elem(), depth+1)
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			d.walk(path+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i), depth+1)
		}
	case reflect.Slice, reflect.Array:
		n := min(a.Len(), b.Len())
		if a.Len() != b.Len() {
			d.add(path+".len", fmt.Sprintf("%d before, %d after", a.Len(), b.Len()))
		}
		if a.Kind() == reflect.Slice && a.Type().Elem().Kind() == reflect.Uint8 {
			x, y := a.Bytes()[:n], b.Bytes()[:n]
			if !bytes.Equal(x, y) {
				i := 0
				for x[i] == y[i] {
					i++
				}
				d.add(path, fmt.Sprintf("byte %d of %d: %d before, %d after", i, n, x[i], y[i]))
			}
			return
		}
		for i := 0; i < n; i++ {
			d.walk(path, a.Index(i), b.Index(i), depth+1)
		}
	case reflect.Map:
		for _, k := range a.MapKeys() {
			if bv := b.MapIndex(k); bv.IsValid() {
				d.walk(path, a.MapIndex(k), bv, depth+1)
			} else {
				d.add(path+".key", "key "+savGateBrief(fmt.Sprint(k))+" absent after LOAD")
			}
		}
		for _, k := range b.MapKeys() {
			if !a.MapIndex(k).IsValid() {
				d.add(path+".key", "key "+savGateBrief(fmt.Sprint(k))+" added by LOAD")
			}
		}
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
	case reflect.Bool:
		leaf(a.Bool() == b.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		leaf(a.Int() == b.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		leaf(a.Uint() == b.Uint())
	case reflect.Float32, reflect.Float64:
		leaf(math.Float64bits(a.Float()) == math.Float64bits(b.Float()))
	case reflect.Complex64, reflect.Complex128:
		leaf(a.Complex() == b.Complex())
	case reflect.String:
		leaf(a.String() == b.String())
	}
}

// savGateDiff returns every differing leaf path under prefix, with its first
// detail.
func savGateDiff(prefix string, a, b reflect.Value) map[string]string {
	out := map[string]string{}
	d := &savGateDiffer{seen: map[[2]uintptr]bool{}, add: func(p, detail string) {
		if _, ok := out[p]; !ok {
			out[p] = detail
		}
	}}
	d.walk(prefix, a, b, 0)
	return out
}

// savGateAccess makes an addressable field settable and readable, unexported
// or not.
func savGateAccess(v reflect.Value) reflect.Value {
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

// savGateWorldFields names what differs in the hashed World, by field path.
// Each top-level World field that differs is copied from the saved copy into
// a shallow copy of the loaded World; only a field whose copy moves the hash
// is hashed state, and every differing leaf path under it is reported. So a
// second loss inside a World that already differs at LOAD is its own path.
// If the hashes still differ with every such field copied, "world.other"
// names the remainder.
func savGateWorldFields(a, b *sim.World) map[string]string {
	seen := map[string]string{}
	av, bv := reflect.ValueOf(a).Elem(), reflect.ValueOf(b).Elem()
	probe := reflect.New(av.Type())
	probe.Elem().Set(bv)
	world := probe.Interface().(*sim.World)
	base := b.Hash()
	for i := 0; i < av.NumField(); i++ {
		local := savGateDiff("world."+av.Type().Field(i).Name, av.Field(i), bv.Field(i))
		if len(local) == 0 {
			continue
		}
		dst := savGateAccess(probe.Elem().Field(i))
		dst.Set(savGateAccess(av.Field(i)))
		if world.Hash() == base {
			dst.Set(savGateAccess(bv.Field(i)))
			continue
		}
		for p, detail := range local {
			seen[p] = detail
		}
	}
	if world.Hash() != a.Hash() {
		seen["world.other"] = "the hashes differ with every differing field copied"
	}
	if len(seen) == 0 {
		seen["world.other"] = "no field differs; the rest of the byte form"
	}
	return seen
}

// savGateBrief prints a value short enough for one log line.
func savGateBrief(s string) string {
	if len(s) > 160 {
		s = s[:160] + "..."
	}
	return s
}

// savGateOpen builds one front end and opens a case's mission on it.
func savGateOpen(t *testing.T, assets, title string, c savGateCase, raw []byte) (*FrontEnd, *ui.App, string, bool) {
	t.Helper()
	f, err := decodedInstallFront(assets)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", assets, err)
	}
	cleanupFrontAudio(t, f)
	headlessCensusMusic(t, f)
	f.SetDeterministicFrames(true)
	a := f.App(title)
	a.Layout(1024, 768)
	var open ui.MapOpener
	if raw != nil {
		var town bool
		open, town, err = f.RestoreOriginal(raw)
		if err != nil {
			return nil, nil, "restore: " + err.Error(), false
		}
		if town {
			return nil, nil, "", true
		}
	} else {
		open = f.NewGameOpener(c.mission, ui.ChargenResult{Name: "Gate", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	}
	if err := a.OpenMission(open); err != nil {
		return nil, nil, "open: " + err.Error(), false
	}
	if f.live == nil {
		return nil, nil, "open: no live mission", false
	}
	return f, a, "", false
}

// savGatePlay advances the chosen ticks, then, for a kill case, kills the
// lowest-id living non-party actors and waits for their bodies to decay.
func savGatePlay(t *testing.T, f *FrontEnd, c savGateCase) {
	f.LiveAdvance(c.ticks)
	if c.ticks > 0 && !c.noDrop {
		savGateDrop(t, f, c)
	}
	if !c.kill {
		return
	}
	party := map[sim.EntityID]bool{}
	for _, id := range f.live.mission.ids {
		party[id] = true
	}
	var killed []sim.EntityID
	for _, e := range f.live.world.Entities() {
		if len(killed) == savGateKills && c.kills == 0 || c.kills > 0 && len(killed) == c.kills {
			break
		}
		if party[e.ID] || c.spareZero && e.ID == 0 || e.Owner == sim.SelfSlot || e.HP <= 0 || e.Decay != sim.DecayNone {
			continue
		}
		killed = append(killed, e.ID)
		f.LiveKill(uint32(e.ID))
	}
	if len(killed) == 0 {
		return
	}
	t.Logf("PLAY %s killed %v", c.name, killed)
	if c.dying {
		f.LiveAdvance(savGateDyingTicks)
		return
	}
	// A body is fully decayed once the live world no longer holds it,
	// whichever ledger (terminal or original dead) kept its record. A killed
	// actor the world holds alive and not decaying did not die; waiting on it
	// would only run to the cap, so it is named and not waited for.
	var spared []sim.EntityID
	for n := 0; n < savGateDecayCap; n += 100 {
		f.LiveAdvance(100)
		done := true
		spared = spared[:0]
		for _, id := range killed {
			e, ok := f.live.world.Entity(id)
			switch {
			case !ok:
			case e.HP > 0 && e.Decay == sim.DecayNone:
				spared = append(spared, id)
			default:
				done = false
			}
		}
		if done {
			t.Logf("PLAY %s decayed fully after %d ticks; not killed %v", c.name, n+100, spared)
			return
		}
	}
	t.Logf("PLAY %s still decaying at the %d-tick cap", c.name, savGateDecayCap)
}

// savGateDrop makes the first party member drop one item on its own cell,
// the owner's ground-drop route: a new sack the writer must produce from
// current state rather than carry from loaded bytes. It tries the pack first
// and then each equipment slot, and stops at the first drop the world applies.
func savGateDrop(t *testing.T, f *FrontEnd, c savGateCase) {
	if len(f.live.mission.ids) == 0 {
		return
	}
	hero := f.live.mission.ids[0]
	e, ok := f.live.entity(hero)
	if !ok {
		return
	}
	at := sim.CellPoint{X: e.X, Y: e.Y}
	tries := []sim.Command{sim.DropCarried(hero, 0, at)}
	for slot := sim.EquipSlot(1); slot <= 12; slot++ {
		tries = append(tries, sim.DropWorn(hero, slot, at))
	}
	before := len(f.live.world.Sacks())
	for _, cmd := range tries {
		f.live.pending = append(f.live.pending, cmd)
		f.LiveAdvance(1)
		if n := len(f.live.world.Sacks()); n != before {
			t.Logf("PLAY %s dropped an item at %v: sacks %d -> %d", c.name, at, before, n)
			return
		}
	}
	t.Logf("PLAY %s: no drop applied", c.name)
}

// savGateSave writes f through the player's SAVE dialog route, on or off the
// map, and returns the written bytes.
func savGateSave(t *testing.T, f *FrontEnd, onMap bool) ([]byte, string) {
	dir := t.TempDir()
	seams := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})
	prepared, err := seams.Prepare(ui.SaveRequest{OnMap: onMap, Directory: dir, Name: "gate", Format: ui.SaveSAV})
	if err != nil {
		return nil, "save: " + err.Error()
	}
	if _, err := prepared.Commit(true); err != nil {
		return nil, "save commit: " + err.Error()
	}
	written, err := ReadSaveFile(filepath.Join(dir, "gate.sav"))
	if err != nil {
		return nil, "save read: " + err.Error()
	}
	if _, err := sav.Open(written); err != nil {
		return nil, "save decode: " + err.Error()
	}
	return written, ""
}

// savGateRun takes one case through GAME, SAV, LOAD, GAME.
func savGateRun(t *testing.T, assets string, c savGateCase) (out savGateOutcome) {
	// A panic anywhere in the cycle is a refusal named by its message, so one
	// case cannot end the census for the rest of the population.
	defer func() {
		if r := recover(); r != nil {
			out.refusal = fmt.Sprint("panic: ", r)
		}
	}()
	if c.raw != nil {
		if _, err := sav.Open(c.raw); err != nil {
			t.Logf("SAV-ROUNDTRIP-GATE UNREADABLE %s: not a SAV container: %v", c.name, err)
			return savGateOutcome{unreadable: true}
		}
	}
	if c.town {
		return savGateTownRun(t, assets, c)
	}
	f, a, refusal, town := savGateOpen(t, assets, "gate play "+c.name, c, c.raw)
	if town {
		return savGateOutcome{refusal: "restore: the file restores as a town"}
	}
	if refusal != "" {
		return savGateOutcome{refusal: refusal}
	}
	savGatePlay(t, f, c)
	for cycle := 1; ; cycle++ {
		stage, prefix := "", ""
		if cycle > 1 {
			stage, prefix = "cycle ", "cycle."
		}
		var revivedIdentity uint32
		if c.name == "2027-09-07/game0033.sav dying" && cycle == 1 {
			if got := fmt.Sprintf("%X", sha256.Sum256(c.raw)); got != "B6FFD904F44717C76E22F12D8801A21B3DFCA3DD8563B7E02AAECBF93A4C740A" {
				t.Fatalf("dying corpus source hash=%s", got)
			}
			state := f.live.mission.state.savedDocument
			for _, binding := range state.Actors {
				if binding.EntityID != 36 {
					continue
				}
				e, ok := f.live.world.Entity(binding.EntityID)
				if !ok || binding.ObjectIndex != 83 {
					t.Fatal("dying source actor binding", binding, ok)
				}
				r := &state.Document.Objects[binding.ObjectIndex-1]
				st, _ := savedStructureValue(r, "Stage")
				hp, _ := savedStructureValue(r, "Health")
				revivedIdentity, _ = savedStructureValue(r, "Identity")
				if e.HP != 31 || e.MaxHP != 31 || e.Decay != sim.DecayNone || e.Owner != 2 || e.Group != 11 || hp != 0 || st != 1 || revivedIdentity == 0 {
					t.Fatal("dying route live actor/document stage witness", e, hp, st, revivedIdentity)
				}
				t.Logf("PRE-SAVE entity %d HP %d/%d Decay %d, document Health %d Stage %d", e.ID, e.HP, e.MaxHP, e.Decay, hp, st)
			}
			if revivedIdentity == 0 {
				t.Fatal("dying route lost revived actor binding")
			}
		}
		g, b, written, refusal := savGateSaveLoad(t, assets, c, f)
		if refusal != "" {
			t.Logf("CYCLE %s %d refused", c.name, cycle)
			out.refusal = stage + refusal
			sort.Strings(out.mismatches)
			return out
		}
		if revivedIdentity != 0 {
			doc, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for i := range doc.Objects {
				r := &doc.Objects[i]
				identity, _ := savedStructureValue(r, "Identity")
				if identity != revivedIdentity {
					continue
				}
				hp, _ := savedStructureValue(r, "Health")
				st, _ := savedStructureValue(r, "Stage")
				if hp != 31 || st != 0 {
					t.Fatal("dying route written actor tuple", hp, st)
				}
				found = true
			}
			loaded, ok := g.live.world.Entity(36)
			if !found || !ok || loaded.HP != 31 || loaded.Decay != sim.DecayNone {
				t.Fatal("dying route written actor did not reload", found, loaded, ok)
			}
		}
		savGateCompare(t, c, f, a, g, b, prefix, &out)
		if cycle >= c.cycles {
			break
		}
		// The loaded copy becomes the game, which now carries the dead of the
		// cycles before it: kill more, let them decay, and save again.
		a.StopAudio()
		f, a = g, b
		savGatePlay(t, f, savGateCase{name: fmt.Sprintf("%s cycle %d", c.name, cycle+1), ticks: 30, kill: true, noDrop: c.noDrop, kills: c.kills})
	}
	sort.Strings(out.mismatches)
	return out
}

// savGateTownProvenance names the town snapshot fields that carry the loaded
// document and its object topology rather than town state: our writer
// rebuilds them by design, so their bytes differ after every resave. The
// unexported fields are constructor inputs of the same kind.
var savGateTownProvenance = map[string]bool{"OriginalCity": true, "CityObjects": true}

// savGateTownRun restores a town, saves it through the SAVE dialog off the
// map, restores the written file on a fresh front end, and compares the two
// towns' snapshots field path by field path.
func savGateTownRun(t *testing.T, assets string, c savGateCase) (out savGateOutcome) {
	restore := func(raw []byte) (*FrontEnd, string) {
		f, err := decodedInstallFront(assets)
		if err != nil {
			t.Fatalf("NewFrontEnd(%q): %v", assets, err)
		}
		cleanupFrontAudio(t, f)
		f.SetDeterministicFrames(true)
		_, town, err := f.RestoreOriginal(raw)
		if err != nil {
			return nil, "restore: " + err.Error()
		}
		if !town {
			return nil, "restore: the file restores as a mission"
		}
		return f, ""
	}
	f, refusal := restore(c.raw)
	if refusal != "" {
		out.refusal = refusal
		return out
	}
	before, _, err := currentRoundTripSnapshot(f, false)
	if err != nil {
		out.refusal = "snapshot: " + err.Error()
		return out
	}
	written, refusal := savGateSave(t, f, false)
	if refusal != "" {
		out.refusal = refusal
		return out
	}
	g, refusal := restore(written)
	if refusal != "" {
		out.refusal = "load " + refusal
		return out
	}
	after, _, err := currentRoundTripSnapshot(g, false)
	if err != nil {
		out.refusal = "load snapshot: " + err.Error()
		return out
	}
	fields := map[string]string{}
	bv, av := reflect.ValueOf(&before).Elem(), reflect.ValueOf(&after).Elem()
	for i := 0; i < bv.NumField(); i++ {
		field := bv.Type().Field(i)
		if !field.IsExported() || savGateTownProvenance[field.Name] {
			continue
		}
		for p, detail := range savGateDiff("town."+field.Name, bv.Field(i), av.Field(i)) {
			fields[p] = detail
		}
	}
	names := make([]string, 0, len(fields))
	for k := range fields {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, field := range names {
		out.mismatches = append(out.mismatches, field)
		t.Logf("MISMATCH %s %s: %s", c.name, field, fields[field])
	}
	return out
}

// savGateSaveLoad saves f through the player's SAVE dialog route and restores
// the written file into a second front end.
func savGateSaveLoad(t *testing.T, assets string, c savGateCase, f *FrontEnd) (*FrontEnd, *ui.App, []byte, string) {
	written, refusal := savGateSave(t, f, true)
	if refusal != "" {
		return nil, nil, nil, refusal
	}
	g, b, refusal, town := savGateOpen(t, assets, "gate load "+c.name, c, written)
	if town {
		return nil, nil, nil, "load: our mission save reopened as a town"
	}
	if refusal != "" {
		return nil, nil, nil, "load " + refusal
	}
	return g, b, written, ""
}

// savGateCompare compares the saved copy f with the loaded copy g, advances
// both, and compares again. Each field is recorded under prefix. When LOAD
// agreed on the hash, the hashes must agree on every tick of the advance;
// when it did not, the hashed field paths are named again after the advance,
// so a loss that appears or spreads during it is its own key.
func savGateCompare(t *testing.T, c savGateCase, f *FrontEnd, a *ui.App, g *FrontEnd, b *ui.App, prefix string, out *savGateOutcome) {
	mismatch := func(field, detail string) {
		out.mismatches = append(out.mismatches, prefix+field)
		t.Logf("MISMATCH %s %s%s: %s", c.name, prefix, field, detail)
	}
	worldFields := func(label string) {
		ha, hb := f.live.world.Hash(), g.live.world.Hash()
		if ha == hb {
			return
		}
		fields := savGateWorldFields(f.live.world, g.live.world)
		names := make([]string, 0, len(fields))
		for k := range fields {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, field := range names {
			mismatch(label+field, fmt.Sprintf("world hash %016x saved, %016x loaded; %s", ha, hb, fields[field]))
		}
	}
	worldFields("")
	moving := map[string]bool{}
	for _, d := range f.live.entityDraws() {
		if d.Step != (image.Point{}) {
			moving[fmt.Sprint(d.ID)] = true
		}
	}
	savGateCompareVisible(savGateVisible(a, f.live), savGateVisible(b, g.live), moving, mismatch, func(field, detail string) {
		out.transients = append(out.transients, prefix+field)
		t.Logf("TRANSIENT %s %s%s: %s", c.name, prefix, field, detail)
	})
	equalAtLoad := f.live.world.Hash() == g.live.world.Hash()
	var comparison savGateWorldComparison
	for i := 1; i <= savGateAdvance; i++ {
		f.LiveAdvance(1)
		g.LiveAdvance(1)
		if equalAtLoad && !comparison.equal(f.live.world, g.live.world) {
			mismatch("advance.hash", fmt.Sprintf("first diverged on common tick %d of %d", i, savGateAdvance))
			break
		}
	}
	if !equalAtLoad {
		worldFields("advanced.")
	}
	savGateCompareVisible(savGateVisible(a, f.live), savGateVisible(b, g.live), nil, func(field, detail string) {
		mismatch("advanced."+field, detail)
	}, nil)
}

type savGateWorldComparison struct {
	a, b []byte
}

// Equal byte forms hash equal; a refused byte form keeps the hash fallback.
func (c *savGateWorldComparison) equal(a, b *sim.World) bool {
	x, errA := a.MarshalBinaryInto(c.a)
	y, errB := b.MarshalBinaryInto(c.b)
	c.a, c.b = x, y
	if errA != nil || errB != nil {
		return a.Hash() == b.Hash()
	}
	return bytes.Equal(x, y)
}

// savGateKits discovers the owner kits named by AGAINROM_SAV_ROUNDTRIP_KITS,
// a path list of files or glob patterns. A pattern that matches nothing is
// ABSENT, which fails. A kit is named by its directory and file name, so the
// baseline does not depend on where the seat keeps it.
func savGateKits(t *testing.T) (cases []savGateCase, absent []string) {
	spec := os.Getenv("AGAINROM_SAV_ROUNDTRIP_KITS")
	for _, pattern := range filepath.SplitList(spec) {
		if pattern == "" {
			continue
		}
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			absent = append(absent, pattern)
			continue
		}
		sort.Strings(matches)
		for _, m := range matches {
			raw, err := os.ReadFile(m)
			if err != nil {
				absent = append(absent, m)
				continue
			}
			name := filepath.Base(filepath.Dir(m)) + "/" + filepath.Base(m)
			cases = append(cases, savGateVariants("kit", name, raw, savGateKitCycles, true)...)
		}
	}
	return cases, absent
}

// savGateVariants are the save points one file yields. A file sav.Open cannot
// read is one unreadable case. A town is one case, saved as loaded off the
// map. A mission is resaved as loaded, then saved after play (a short
// advance, a ground drop, kills and full decay, or with full false the dying
// route that saves the bodies mid-decay), and, when cycles is above one, in
// the owner's repeated sequence of kills, SAVE and LOAD with no drop.
func savGateVariants(population, name string, raw []byte, cycles int, full bool) []savGateCase {
	doc, err := sav.Open(raw)
	if err != nil {
		return []savGateCase{{population: population + " resave", name: name + " resave", raw: raw}}
	}
	if doc.World == nil {
		return []savGateCase{{population: population + " town", name: name + " town", raw: raw, town: true}}
	}
	played := savGateCase{population: population + " played", name: name + " played", raw: raw, ticks: 30, kill: true}
	if !full {
		played.population, played.name, played.dying = population+" dying", name+" dying", true
	}
	out := []savGateCase{
		{population: population + " resave", name: name + " resave", raw: raw},
		played,
	}
	if cycles > 1 {
		out = append(out, savGateCase{population: population + " cycled", name: name + " cycled", raw: raw,
			ticks: 30, kill: true, noDrop: true, cycles: cycles, kills: savGateCycleKills})
	}
	return out
}

// savGateDecaySampled picks the corpus files that play to full decay by a
// hash of the relative path, so a file added to the corpus moves no other
// file between the played and dying populations.
func savGateDecaySampled(rel string) bool {
	h := fnv.New32a()
	h.Write([]byte(rel))
	return h.Sum32()%savGateDecaySample == 0
}

// savGateCorpus discovers every original under AGAINROM_SAVE_CORPUS on the
// milestone-2 walkers' own terms, skipping exp*-* directories.
func savGateCorpus(t *testing.T) []savGateCase {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS must name gameversions/saves")
	}
	var cases []savGateCase
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
		rel, _ := filepath.Rel(corpus, path)
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		cases = append(cases, savGateVariants("corpus", rel, raw, 1, savGateDecaySampled(rel))...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return cases
}

// savGateNewGames are the New Game save points: missions ten and twenty at
// tick zero, after a short advance and a ground drop, and after kills and full
// decay, once including and once sparing the actor with entity id zero.
func savGateNewGames() []savGateCase {
	var cases []savGateCase
	for _, m := range []int{10, 20} {
		cases = append(cases,
			savGateCase{population: "newgame", name: fmt.Sprintf("mission %d at tick 0", m), mission: m},
			savGateCase{population: "newgame", name: fmt.Sprintf("mission %d after 300 ticks", m), mission: m, ticks: 300},
			savGateCase{population: "newgame", name: fmt.Sprintf("mission %d after kills and decay", m), mission: m, ticks: 30, kill: true},
			savGateCase{population: "newgame", name: fmt.Sprintf("mission %d after kills and decay sparing actor zero", m), mission: m, ticks: 30, kill: true, spareZero: true})
	}
	return cases
}

// The New Game and kit part tests run the New Game save points and the owner
// kits named by AGAINROM_SAV_ROUNDTRIP_KITS, split over savGateKitParts.
func TestSAVRoundTripGateNewGameKitsPart1(t *testing.T) { savGateNewGameKits(t, 0) }
func TestSAVRoundTripGateNewGameKitsPart2(t *testing.T) { savGateNewGameKits(t, 1) }
func TestSAVRoundTripGateNewGameKitsPart3(t *testing.T) { savGateNewGameKits(t, 2) }

// The corpus part tests run every original under AGAINROM_SAVE_CORPUS: a town
// saved off the map; a mission resaved as loaded, and played to full decay or
// on the dying route. The cases are split over savGateCorpusParts.
func TestSAVRoundTripGateCorpusPart1(t *testing.T) { savGateCorpusPart(t, 0) }
func TestSAVRoundTripGateCorpusPart2(t *testing.T) { savGateCorpusPart(t, 1) }
func TestSAVRoundTripGateCorpusPart3(t *testing.T) { savGateCorpusPart(t, 2) }
func TestSAVRoundTripGateCorpusPart4(t *testing.T) { savGateCorpusPart(t, 3) }

const (
	savGateKitParts    = 3
	savGateCorpusParts = 4
)

func savGateNewGameKits(t *testing.T, part int) {
	kits, absent := savGateKits(t)
	savGateCensus(t, "newgamekits", part, savGateKitParts, []string{"newgame", "kit resave", "kit played", "kit cycled"}, append(savGateNewGames(), kits...), absent)
}

func savGateCorpusPart(t *testing.T, part int) {
	savGateCensus(t, "corpus", part, savGateCorpusParts, []string{"corpus town", "corpus resave", "corpus played", "corpus dying"}, savGateCorpus(t), nil)
}

// savGateShareCost is a case's weight when the cases are shared among parts:
// its file size, a New Game counted as a typical file, times its route's
// cost.
func savGateShareCost(c savGateCase) int64 {
	size := int64(len(c.raw))
	if c.raw == nil {
		size = 65536
	}
	return size * int64(savGateCost(c)+1)
}

// savGateCost orders cases longest first so the workers finish together.
func savGateCost(c savGateCase) int {
	switch {
	case c.cycles > 1:
		return 3
	case c.kill && !c.dying:
		return 2
	case c.kill:
		return 1
	}
	return 0
}

// savGateRecord is one case of a baseline file: its population, outcome kind
// and the keys it may carry.
type savGateRecord struct {
	population, kind string
	keys             map[string]bool
}

// savGateBaselinePath is the recorded baseline of one gate test.
func savGateBaselinePath(name string) string {
	return filepath.Join("testdata", "savroundtripgate-"+name+".txt")
}

// savGateReadBaseline reads "population<TAB>case<TAB>kind<TAB>key" lines; a
// case's kind line carries the key "-".
func savGateReadBaseline(t *testing.T, name string) map[string]*savGateRecord {
	raw, err := os.ReadFile(savGateBaselinePath(name))
	if err != nil {
		t.Fatalf("baseline %s: %v", name, err)
	}
	out := map[string]*savGateRecord{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 4 {
			t.Fatalf("baseline %s: malformed line %q", name, line)
		}
		r := out[f[1]]
		if r == nil {
			r = &savGateRecord{population: f[0], kind: f[2], keys: map[string]bool{}}
			out[f[1]] = r
		}
		if f[3] != "-" {
			r.keys[f[3]] = true
		}
	}
	return out
}

// savGateWriteBaseline writes the observed outcomes as a baseline file under
// dir, for AGAINROM_SAV_ROUNDTRIP_RECORD.
func savGateWriteBaseline(t *testing.T, dir, name string, lines []string) {
	sort.Strings(lines)
	body := "# population\tcase\toutcome\tkey; written by AGAINROM_SAV_ROUNDTRIP_RECORD\n" + strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "savroundtripgate-"+name+".txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// savGateBaselineLines are the baseline lines of the outcomes.
func savGateBaselineLines(cases []savGateCase, outcomes []savGateOutcome) []string {
	var lines []string
	for i, c := range cases {
		o := outcomes[i]
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t-", c.population, c.name, o.kind()))
		for _, k := range o.keys() {
			lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s", c.population, c.name, o.kind(), k))
		}
	}
	return lines
}

// savGateCensusCounts is one population's census.
type savGateCensusCounts struct {
	Discovered, Accepted, Unreadable, Refused, Mismatched, Absent int
}

// savGateCensusSummary is one part's census, as the part that completes the
// set reads it.
type savGateCensusSummary struct {
	Pops     map[string]*savGateCensusCounts
	Keys     map[string]map[string]int
	Lines    []string
	Cases    int
	Workers  int
	Wall     time.Duration
	Recorded bool
}

// savGateCensus runs one part's share of the cases, at most savGateWorkers at once (or
// AGAINROM_SAV_ROUNDTRIP_WORKERS), logs every outcome in case order and the
// census of each named population, and checks each case against the
// baseline:
//
//   - a recorded case fails on a key it did not carry or on a changed town,
//     unreadable or mission kind; a recorded key it no longer carries is
//     logged FIXED;
//   - a case the baseline does not hold fails on any key its population does
//     not already carry somewhere, so a new corpus file that shows only named
//     defects passes;
//   - a recorded case that did not run fails, so an emptied corpus fails.
//
// The first part also checks the baseline's causes, the absent kits and the
// missing cases, over every discovered case. The part that completes the set
// logs the census of the whole and writes a recorded baseline.
// AGAINROM_SAV_ROUNDTRIP_ONLY keeps only the cases whose name contains it and
// then skips the missing-case check.
func savGateCensus(t *testing.T, name string, part, parts int, populations []string, cases []savGateCase, absent []string) {
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install")
	}
	baseline := savGateReadBaseline(t, name)
	only := os.Getenv("AGAINROM_SAV_ROUNDTRIP_ONLY")
	if only != "" {
		var kept []savGateCase
		for _, c := range cases {
			if strings.Contains(c.name, only) {
				kept = append(kept, c)
			}
		}
		cases = kept
	}
	discovered := cases
	names, costs := make([]string, len(cases)), make([]int64, len(cases))
	for i, c := range cases {
		names[i], costs[i] = c.name, savGateShareCost(c)
	}
	share := corpusPartOf(names, costs, parts)
	cases = nil
	for i, c := range discovered {
		if share[i] == part {
			cases = append(cases, c)
		}
	}
	workers := savGateWorkers
	if n, err := strconv.Atoi(os.Getenv("AGAINROM_SAV_ROUNDTRIP_WORKERS")); err == nil && n > 0 {
		workers = n
	}
	order := make([]int, len(cases))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return savGateCost(cases[order[a]]) > savGateCost(cases[order[b]]) })
	outcomes := make([]savGateOutcome, len(cases))
	start := time.Now()
	next := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				c := cases[i]
				t.Run(c.population+"/"+c.name, func(t *testing.T) {
					began := time.Now()
					outcomes[i] = savGateRun(t, assets, c)
					t.Logf("SAV-ROUNDTRIP-GATE TIME %s: %s", c.name, time.Since(began).Round(time.Millisecond))
				})
			}
		}()
	}
	for _, i := range order {
		next <- i
	}
	close(next)
	wg.Wait()
	mine := savGateCensusSummary{Pops: map[string]*savGateCensusCounts{}, Keys: map[string]map[string]int{}, Cases: len(cases), Workers: workers}
	if dir := os.Getenv("AGAINROM_SAV_ROUNDTRIP_RECORD"); dir != "" {
		mine.Lines, mine.Recorded = savGateBaselineLines(cases, outcomes), true
	}

	pops := mine.Pops
	for _, p := range populations {
		pops[p] = &savGateCensusCounts{}
	}
	named := map[string]map[string]bool{}
	for key, r := range baseline {
		if named[r.population] == nil {
			named[r.population] = map[string]bool{}
		}
		for k := range r.keys {
			named[r.population][k] = true
			if _, ok := savGateCause(k); !ok && part == 0 {
				t.Errorf("SAV-ROUNDTRIP-GATE UNNAMED %s: baselined key %q has no cause in savGateCauses", key, k)
			}
		}
	}
	if part == 0 {
		for _, p := range absent {
			t.Errorf("SAV-ROUNDTRIP-GATE ABSENT kit %s", p)
			pops["kit resave"].Absent++
		}
	}
	ran := map[string]bool{}
	for _, c := range discovered {
		ran[c.name] = true
	}
	for i, c := range cases {
		o, p := outcomes[i], pops[c.population]
		p.Discovered++
		switch o.kind() {
		case "unreadable":
			p.Unreadable++
		case "refused":
			p.Refused++
			t.Logf("SAV-ROUNDTRIP-GATE REFUSED %s: %s", c.name, o.refusal)
		case "mismatched":
			p.Mismatched++
		default:
			p.Accepted++
		}
		if len(o.mismatches) > 0 {
			t.Logf("SAV-ROUNDTRIP-GATE MISMATCHED %s: %s", c.name, strings.Join(o.mismatches, ", "))
		}
		rec := baseline[c.name]
		if rec == nil {
			for _, k := range o.keys() {
				if !named[c.population][k] {
					t.Errorf("SAV-ROUNDTRIP-GATE NEW %s: %s is not a defect its population carries", c.name, k)
				}
			}
			t.Logf("SAV-ROUNDTRIP-GATE ADDED %s: %s, not in the baseline", c.name, o.kind())
			continue
		}
		if (rec.kind == "unreadable") != o.unreadable || rec.population != c.population {
			t.Errorf("SAV-ROUNDTRIP-GATE KIND %s: recorded %s in %s, now %s in %s", c.name, rec.kind, rec.population, o.kind(), c.population)
		}
		for _, k := range o.keys() {
			if !rec.keys[k] {
				t.Errorf("SAV-ROUNDTRIP-GATE NEW %s: %s", c.name, k)
			}
		}
		have := map[string]bool{}
		for _, k := range o.keys() {
			have[k] = true
		}
		var fixed []string
		for k := range rec.keys {
			if !have[k] {
				fixed = append(fixed, k)
			}
		}
		sort.Strings(fixed)
		for _, k := range fixed {
			t.Logf("SAV-ROUNDTRIP-GATE FIXED %s: %s no longer occurs; record the baseline again", c.name, k)
		}
	}
	if only == "" && part == 0 {
		var missing []string
		for key := range baseline {
			if !ran[key] {
				missing = append(missing, key)
			}
		}
		sort.Strings(missing)
		for _, key := range missing {
			t.Errorf("SAV-ROUNDTRIP-GATE MISSING %s: recorded in the baseline, not run", key)
		}
	}
	keys := mine.Keys
	for i, c := range cases {
		if keys[c.population] == nil {
			keys[c.population] = map[string]int{}
		}
		for _, k := range outcomes[i].keys() {
			keys[c.population][k]++
		}
	}
	mine.Wall = time.Since(start)
	t.Logf("SAV-ROUNDTRIP-GATE PART %d of %d: %d case(s) in %s", part+1, parts, len(cases), mine.Wall.Round(time.Second))
	set := corpusPartsCollect(t, "savgate-"+name, part, parts, mine)
	if set == nil {
		return
	}
	whole := savGateCensusSummary{Pops: map[string]*savGateCensusCounts{}, Keys: map[string]map[string]int{}}
	for _, p := range populations {
		whole.Pops[p] = &savGateCensusCounts{}
	}
	for _, s := range corpusPartsDecode[savGateCensusSummary](t, set) {
		for p, c := range s.Pops {
			w := whole.Pops[p]
			if w == nil {
				w = &savGateCensusCounts{}
				whole.Pops[p] = w
			}
			w.Discovered += c.Discovered
			w.Accepted += c.Accepted
			w.Unreadable += c.Unreadable
			w.Refused += c.Refused
			w.Mismatched += c.Mismatched
			w.Absent += c.Absent
		}
		for p, ks := range s.Keys {
			if whole.Keys[p] == nil {
				whole.Keys[p] = map[string]int{}
			}
			for k, n := range ks {
				whole.Keys[p][k] += n
			}
		}
		whole.Lines = append(whole.Lines, s.Lines...)
		whole.Recorded = whole.Recorded || s.Recorded
		whole.Cases += s.Cases
		whole.Workers = max(whole.Workers, s.Workers)
		whole.Wall = max(whole.Wall, s.Wall)
	}
	if dir := os.Getenv("AGAINROM_SAV_ROUNDTRIP_RECORD"); dir != "" && whole.Recorded {
		savGateWriteBaseline(t, dir, name, whole.Lines)
	}
	for _, p := range populations {
		c := whole.Pops[p]
		t.Logf("SAV-ROUNDTRIP-GATE CENSUS %s: discovered %d, accepted %d, unreadable %d, refused %d, mismatched %d, absent %d",
			p, c.Discovered, c.Accepted, c.Unreadable, c.Refused, c.Mismatched, c.Absent)
		sorted := make([]string, 0, len(whole.Keys[p]))
		for k := range whole.Keys[p] {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			cause, _ := savGateCause(k)
			t.Logf("SAV-ROUNDTRIP-GATE KEY %s: %s: %d case(s); %s", p, k, whole.Keys[p][k], cause)
		}
	}
	t.Logf("SAV-ROUNDTRIP-GATE RUNTIME %d case(s) in %s on %d worker(s) in each of %d part(s)", whole.Cases, whole.Wall.Round(time.Second), whole.Workers, parts)
}
