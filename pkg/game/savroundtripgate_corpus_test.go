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
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	// savGateKills is how many non-party actors play kills before saving.
	savGateKills = 5
	// savGateDecayCap bounds the ticks play waits for those bodies to decay
	// fully; a body still decaying at the cap is saved mid-decay.
	savGateDecayCap = 24000
	// savGateAdvance is the common advance both copies run after LOAD.
	savGateAdvance = 120
	// savGateDyingTicks is how long the dying route advances after its kills:
	// the bodies are saved mid-decay, about an eighth of the full span.
	savGateDyingTicks = 2400
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

// savGateDigits normalises a reason so one defect over many files is one key.
func savGateDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return '#'
		}
		return r
	}, s)
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

// savGateBrief prints a value short enough for one log line.
func savGateBrief(s string) string {
	if len(s) > 160 {
		s = s[:160] + "..."
	}
	return s
}

// savGateOpen builds one front end and opens a case's mission on it.
func savGateOpen(t testing.TB, assets, title string, c savGateCase, raw []byte) (*FrontEnd, *ui.App, string, bool) {
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
func savGateSave(t testing.TB, f *FrontEnd, onMap bool) ([]byte, string) {
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

// savGateTownProvenance names the town snapshot fields that carry the loaded
// document and its object topology rather than town state: our writer
// rebuilds them by design, so their bytes differ after every resave. The
// unexported fields are constructor inputs of the same kind.
var savGateTownProvenance = map[string]bool{"OriginalCity": true, "CityObjects": true}

// savGateSaveLoad saves f through the player's SAVE dialog route and restores
// the written file into a second front end.
func savGateSaveLoad(t testing.TB, assets string, c savGateCase, f *FrontEnd) (*FrontEnd, *ui.App, []byte, string) {
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
