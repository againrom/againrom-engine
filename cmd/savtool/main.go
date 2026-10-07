// Command savtool reads, checks and edits a save file the ORIGINAL GAME wrote
// (developer tool).
//
//	savtool info    FILE           the container, the head, the roster, the world half
//	savtool blocks  FILE [-n N]    the block-plane delta records
//	savtool actors  FILE [-n N]    the placeable-object heads
//	savtool session FILE           the trigger latches, the diplomacy matrix, the counters
//	savtool sacks   FILE...        the authoritative top-level Sack population
//	savtool campaign FILE...       the embedded state store's decoded campaign projection
//	savtool fog     FILE...        the tail's state store and the explored-terrain record
//	savtool verify  FILE...        read each and re-emit it, reporting byte-identity
//	savtool set     FILE -out PATH [edits]   read, modify, write
//
// verify is the tool's centre. It reads a save, re-emits it with the body
// RE-COMPRESSED — not with the blob it read carried across — and reports whether
// the result is the same bytes. That is the strongest check available to a tree
// with no running original: if every byte this tree did not understand comes
// back where it was, the parts it did understand are the only parts that can be
// wrong.
//
// A SAVE THIS TREE WRITES IS ONE WHOSE LINEAGE BEGAN AT A SAVE THE GAME WROTE.
// The set verb retains unedited source members, including opaque regions; it
// does not author the state of a live mission. The sacks verb walks the full
// document framing but projects only ground-loot values, not those opaque bytes.
//
// set NEVER WRITES IN PLACE. -out is required, because the only saves that exist
// are evidence and a tool that could default to overwriting its input is one
// slip from destroying it.
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"againrom/pkg/formats/sav"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "savtool:", err)
		os.Exit(1)
	}
}

const usage = "savtool {info|blocks|actors|party|objects|session|sacks|campaign|verify|lead|fog|set} FILE..."

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", usage)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "sacks":
		if len(rest) == 0 {
			return fmt.Errorf("sacks takes at least one file")
		}
		for _, path := range rest {
			f, err := open(path)
			if err != nil {
				return err
			}
			sacks, present, err := f.GroundSacks()
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			fmt.Fprintf(out, "%s world=%t sacks=%d\n", path, present, len(sacks))
			for i, sack := range sacks {
				fmt.Fprintf(out, "  %d identity=%d cell=(%d,%d) fine=(%d,%d) gold=%d t1c=%d items=%+v\n", i, sack.Identity, sack.Cell&255, sack.Cell>>8, sack.FineX, sack.FineY, sack.Gold, sack.Token1C, sack.Items)
			}
		}
		return nil
	case "campaign":
		if len(rest) == 0 {
			return fmt.Errorf("campaign takes at least one file")
		}
		for _, path := range rest {
			f, err := open(path)
			if err != nil {
				return err
			}
			if err := campaignVerb(out, path, f); err != nil {
				return err
			}
		}
		return nil
	case "objects":
		fs := flag.NewFlagSet("objects", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		class := fs.String("class", "Building", "which class to chain")
		n := fs.Int("n", 8, "how many rows to print (0 for all)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("objects takes exactly one file")
		}
		f, err := open(fs.Arg(0))
		if err != nil {
			return err
		}
		return objects(out, f, *class, *n)
	case "info", "blocks", "actors", "party", "session":
		fs := flag.NewFlagSet(verb, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		n := fs.Int("n", 20, "how many rows to print (0 for all)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("%s takes exactly one file", verb)
		}
		f, err := open(fs.Arg(0))
		if err != nil {
			return err
		}
		switch verb {
		case "info":
			return info(out, fs.Arg(0), f)
		case "blocks":
			return blocks(out, f, *n)
		case "actors":
			return actors(out, f, *n)
		case "party":
			return party(out, f, *n)
		default:
			return session(out, f)
		}
	case "verify":
		if len(rest) == 0 {
			return fmt.Errorf("verify takes at least one file")
		}
		return verify(out, rest)
	case "lead":
		if len(rest) == 0 {
			return fmt.Errorf("lead takes at least one file")
		}
		return lead(out, rest)
	case "fog":
		if len(rest) == 0 {
			return fmt.Errorf("fog takes at least one file")
		}
		return fog(out, rest)
	case "set":
		return set(out, rest)
	}
	return fmt.Errorf("unknown verb %q: %s", verb, usage)
}

func open(path string) (*sav.File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := sav.Open(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

func info(out io.Writer, path string, f *sav.File) error {
	h := f.Head
	fmt.Fprintf(out, "%s\n", path)
	fmt.Fprintf(out, "  version   %#08x   body %d bytes   label %q   store %d bytes   rest %d bytes\n",
		f.Version, len(f.Body), string(f.Label), len(f.Store), len(f.TailRest))
	fmt.Fprintf(out, "  head      counters %d/%d   map %q   mission %d   difficulty %d   players %d\n",
		h.CounterA, h.CounterB, h.MapName, h.Mission, h.Difficulty, h.PlayerCount)
	if h.Mission == 0 {
		fmt.Fprintf(out, "  BETWEEN MISSIONS: mission number 0. The map name above is the PREVIOUS\n"+
			"            mission's and does not say a mission is in progress.\n")
	}
	for i, p := range f.Players {
		who := "scenario"
		if p.Participant == 0 {
			who = "HUMAN"
		}
		fmt.Fprintf(out, "  player %d  slot %2d  %-16q %-8s money %-8d outcome %d (%s)\n",
			i, p.Slot, p.Name, who, p.Money, p.Outcome, outcomeName(p.Outcome))
	}
	if f.World == nil {
		fmt.Fprintf(out, "  world     NONE — no block plane, no cell records, no session block\n")
	} else {
		w := f.World
		won, lost, err := f.Counters()
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "  world     blocks %d @%#x   cell records %d @%#x   session @%#x   won %d lost %d\n",
			len(w.Blocks), w.BlocksOff, w.CellRecCount, w.CellRecOff, w.SessionOff, won, lost)
	}
	live, dead, ided := 0, 0, 0
	for _, a := range f.Actors {
		if a.Dead() {
			dead++
		} else {
			live++
		}
		if a.MapUnitID != 0 {
			ided++
		}
	}
	fmt.Fprintf(out, "  actors    %d owner-graph actors: %d living, %d dead, %d carrying a map unit id\n",
		len(f.Actors), live, dead, ided)
	return nil
}

func campaignVerb(out io.Writer, path string, f *sav.File) error {
	proj, ok, err := f.Campaign()
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if !ok {
		fmt.Fprintf(out, "%s no campaign (tail did not frame an embedded state store)\n", path)
		return nil
	}
	fmt.Fprintf(out, "%s head_mission=%d main=%d selected=%d mercenaries=%d permanent=%d documents=%d\n",
		path, f.Head.Mission, proj.Main.Mission, proj.SelectedMission,
		len(proj.Mercenaries), len(proj.PermanentMercenaries), len(proj.Documents))
	if proj.CarrierRecords != 0 {
		if proj.Payload != nil {
			fmt.Fprintf(out, "  payload   %d carrier records, envelope version %d, %d bytes, sha256 %x\n",
				proj.CarrierRecords, proj.Payload.Version, len(proj.Payload.Data), sha256.Sum256(proj.Payload.Data))
		} else {
			fmt.Fprintf(out, "  payload   %d carrier records ignored: %s\n", proj.CarrierRecords, proj.PayloadError)
		}
	}
	return nil
}

func outcomeName(v uint8) string {
	switch v {
	case 0:
		return "in progress"
	case 1:
		return "COMPLETE"
	default:
		return "failed"
	}
}

func blocks(out io.Writer, f *sav.File, n int) error {
	if f.World == nil {
		return fmt.Errorf("this save has no world half")
	}
	rs := f.World.Blocks
	fmt.Fprintf(out, "%d block-plane DELTA records (a delta over the map's own terrain, not a plane)\n", len(rs))
	for i, r := range rs {
		if n > 0 && i >= n {
			fmt.Fprintf(out, "  ... %d more\n", len(rs)-n)
			break
		}
		fmt.Fprintf(out, "  %4d  cell %#04x  (col %3d,row %3d)  dyn %#02x  static %#02x\n",
			i, r.Cell, r.Col(), r.Row(), r.Dyn, r.Static)
	}
	return nil
}

func actors(out io.Writer, f *sav.File, n int) error {
	fmt.Fprintf(out, "%d owner-graph actors\n", len(f.Actors))
	for i, a := range f.Actors {
		if n > 0 && i >= n {
			fmt.Fprintf(out, "  ... %d more\n", len(f.Actors)-n)
			break
		}
		state := "living"
		if a.Dead() {
			state = "DEAD"
		}
		rest := ""
		if !a.AtRest() {
			rest = "  mid-walk"
		}
		fmt.Fprintf(out, "  %4d  @%#06x  cell (%3d,%3d)  fine %#02x,%#02x  id %-6d unit %-5d %s%s\n",
			i, a.Off, a.Col(), a.Row(), a.FineX, a.FineY, a.RuntimeID, a.MapUnitID, state, rest)
	}
	return nil
}

// objects chains a fixed-length class and re-encodes each record FROM ITS
// DECODED FORM, which is the check a byte-carrying round trip cannot make.
func objects(out io.Writer, f *sav.File, class string, n int) error {
	fmt.Fprintf(out, "classes introduced: ")
	for i, r := range f.ClassRecords() {
		if i > 0 {
			fmt.Fprintf(out, " ")
		}
		fmt.Fprintf(out, "%s", r.Name)
	}
	fmt.Fprintln(out)
	objs, chainErr := f.Chain(class)
	if chainErr != nil && len(objs) == 0 {
		return chainErr
	}
	if chainErr != nil {
		fmt.Fprintf(out, "the %s run does NOT chain: %v\n", class, chainErr)
		fmt.Fprintf(out, "  %d instance(s) decoded before it broke, re-encoded below\n", len(objs))
	}
	same := 0
	for i, o := range objs {
		ok, at, err := f.Reencode(o)
		if err != nil {
			return err
		}
		if ok {
			same++
		}
		if n > 0 && i >= n {
			continue
		}
		state := "re-encodes identically"
		if !ok {
			state = fmt.Sprintf("DIFFERS at %#x", at)
		}
		fmt.Fprintf(out, "  %3d @%#06x  id %-4d unit %-5d identity %#x ref %#x  %s\n",
			i, o.Off, o.Fields.Value["RuntimeID"], o.Fields.Value["T08"]&0xffff,
			o.Fields.Value["Identity"], o.Fields.Value["Reference"], state)
	}
	// Three checks the length alone does not make: identity keys are the
	// writing process's own addresses and must therefore be distinct; the
	// creation-order id is a run; and the map unit id is the map's own.
	ids := map[uint32]bool{}
	dup, lo, hi := 0, ^uint32(0), uint32(0)
	for _, o := range objs {
		k := o.Fields.Value["Identity"]
		if ids[k] {
			dup++
		}
		ids[k] = true
		if r := o.Fields.Value["RuntimeID"]; r < lo || lo == ^uint32(0) {
			lo = r
		} else if r > hi {
			hi = r
		}
	}
	fmt.Fprintf(out, "identity keys: %d distinct of %d (%d repeated)   creation order %d..%d\n",
		len(ids), len(objs), dup, lo, hi)
	fmt.Fprintf(out, "%d %s instance(s), %d re-encoded identically from the decoded form\n",
		len(objs), class, same)
	if same != len(objs) {
		return fmt.Errorf("%d of %d records did not re-encode", len(objs)-same, len(objs))
	}
	return chainErr
}

func session(out io.Writer, f *sav.File) error {
	if f.World == nil {
		return fmt.Errorf("this save has no world half and therefore no session block")
	}
	won, lost, err := f.Counters()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "session block @%#x\n  won %d  lost %d\n", f.World.SessionOff, won, lost)
	set := 0
	var first []int
	for i := 0; i < 1000; i++ {
		v, err := f.TriggerLatch(i)
		if err != nil {
			return err
		}
		if v != 0 {
			set++
			if len(first) < 24 {
				first = append(first, i)
			}
		}
	}
	fmt.Fprintf(out, "  %d of 1000 fire-once trigger latches set: %v\n", set, first)
	nz := 0
	for i := 0; i < 100; i++ {
		v, err := f.TriggerResult(i)
		if err != nil {
			return err
		}
		if v != 0 {
			nz++
		}
	}
	fmt.Fprintf(out, "  %d of 100 trigger result slots non-zero\n", nz)
	rows := 0
	for a := 0; a < 50 && rows < 8; a++ {
		var line []string
		any := false
		for b := 0; b < 8; b++ {
			v, err := f.Diplomacy(a, b)
			if err != nil {
				return err
			}
			if v != 0 {
				any = true
			}
			line = append(line, strconv.Itoa(int(v)))
		}
		if any {
			fmt.Fprintf(out, "  diplomacy[%2d][0:8] = %s\n", a, strings.Join(line, " "))
			rows++
		}
	}
	return nil
}

func verify(out io.Writer, paths []string) error {
	same, total := 0, 0
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		total++
		f, err := sav.Open(b)
		if err != nil {
			fmt.Fprintf(out, "%-64s OPEN FAILED: %v\n", path, err)
			continue
		}
		got := f.Marshal()
		if d := firstDiff(b, got); d < 0 {
			same++
			fmt.Fprintf(out, "%-64s identical (%d bytes)\n", path, len(b))
		} else {
			fmt.Fprintf(out, "%-64s DIFFERS at %#x (%d in, %d out)\n", path, d, len(b), len(got))
		}
	}
	fmt.Fprintf(out, "%d/%d byte-identical\n", same, total)
	if same != total {
		return fmt.Errorf("%d of %d files did not round-trip", total-same, total)
	}
	return nil
}

// firstDiff answers the first offset at which two slices differ, or -1. A length
// difference at the end of an otherwise equal prefix reports at the shorter
// length, which is where a reader would notice it.
func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

func set(out io.Writer, args []string) error {
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outPath := fs.String("out", "", "where to write the result (required; never in place)")
	player := fs.Int("player", 0, "which roster slot the -money and -outcome edits name")
	money := fs.Int64("money", -1, "set that player's purse")
	outcome := fs.Int("outcome", -1, "set that player's mission-outcome latch: 0 in progress, 1 complete, 2 failed")
	label := fs.String("label", "", "set the save's slot text")
	mission := fs.Int("mission", -1, "set the campaign mission number")
	actor := fs.String("actor", "", "move an actor: INDEX:COL:ROW[:FINEX:FINEY]")
	latch := fs.String("latch", "", "set a fire-once trigger latch: INDEX:VALUE")
	diplo := fs.String("diplomacy", "", "set a diplomacy entry: A:B:VALUE")
	block := fs.String("block", "", "set a block record's bytes: INDEX:DYN:STATIC")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("set takes exactly one file")
	}
	if *outPath == "" {
		return fmt.Errorf("set requires -out: it never writes in place")
	}
	f, err := open(fs.Arg(0))
	if err != nil {
		return err
	}
	if *money >= 0 {
		if err := f.SetMoney(*player, uint32(*money)); err != nil {
			return err
		}
	}
	if *outcome >= 0 {
		if err := f.SetOutcome(*player, uint8(*outcome)); err != nil {
			return err
		}
	}
	if *label != "" {
		if err := f.SetLabel([]byte(*label)); err != nil {
			return err
		}
	}
	if *mission >= 0 {
		f.SetMission(uint32(*mission))
	}
	if *actor != "" {
		n, err := fields(*actor, 3, 5)
		if err != nil {
			return fmt.Errorf("-actor: %w", err)
		}
		fx, fy := 0x80, 0x80
		if len(n) == 5 {
			fx, fy = n[3], n[4]
		}
		if err := f.SetActorPosition(n[0], n[2], n[1], uint8(fx), uint8(fy)); err != nil {
			return err
		}
	}
	if *latch != "" {
		n, err := fields(*latch, 2, 2)
		if err != nil {
			return fmt.Errorf("-latch: %w", err)
		}
		if err := f.SetTriggerLatch(n[0], uint8(n[1])); err != nil {
			return err
		}
	}
	if *diplo != "" {
		n, err := fields(*diplo, 3, 3)
		if err != nil {
			return fmt.Errorf("-diplomacy: %w", err)
		}
		if err := f.SetDiplomacy(n[0], n[1], uint8(n[2])); err != nil {
			return err
		}
	}
	if *block != "" {
		n, err := fields(*block, 3, 3)
		if err != nil {
			return fmt.Errorf("-block: %w", err)
		}
		if err := f.SetBlockRecord(n[0], uint8(n[1]), uint8(n[2])); err != nil {
			return err
		}
	}
	b := f.Marshal()
	if err := os.WriteFile(*outPath, b, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "wrote %s: %d bytes\n", *outPath, len(b))
	return nil
}

// fields parses a colon-separated list of integers with a permitted arity range.
// It is a pure function so that every way an edit flag can be wrong is witnessed
// with no file present.
func fields(s string, lo, hi int) ([]int, error) {
	parts := strings.Split(s, ":")
	if len(parts) < lo || len(parts) > hi {
		return nil, fmt.Errorf("%q has %d fields, want %d..%d", s, len(parts), lo, hi)
	}
	out := make([]int, len(parts))
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("%q: %q is not a number", s, p)
		}
		if v < 0 {
			return nil, fmt.Errorf("%q: %d is negative", s, v)
		}
		out[i] = v
	}
	return out, nil
}

// fog prints the tail's state store and the explored-terrain record it
// carries.
//
// IT PRINTS THE THREE REGIONS' EXTENTS BESIDE THE RECORD, because the extent is
// what a reader of this format gets wrong: the store ends where its own framing
// says and a further region follows it, so "store + rest == tail" is the line
// that shows a reader did not run past the end. The rest's first dword is the
// one field of that region research has measured; nothing here interprets it.
//
// A file carrying no record prints so and is not an error. A save taken between
// missions has no Fog section at all.
func fog(out io.Writer, paths []string) error {
	for _, p := range paths {
		f, err := open(p)
		if err != nil {
			return err
		}
		store, ok := f.StateStore()
		if !ok {
			fmt.Fprintf(out, "%s\n  NO STATE STORE: the %d-byte tail does not frame as one\n",
				p, len(f.TailRest))
			continue
		}
		leaves := 0
		for _, sec := range store.Root.Children {
			leaves += len(sec.Children)
		}
		var head uint32
		if len(f.TailRest) >= 4 {
			head = uint32(f.TailRest[0]) | uint32(f.TailRest[1])<<8 |
				uint32(f.TailRest[2])<<16 | uint32(f.TailRest[3])<<24
		}
		fmt.Fprintf(out, "%s\n", p)
		fmt.Fprintf(out, "  store     %d records, %d sections, %d leaves, %d bytes"+
			"   rest %d bytes (first dword %d)\n",
			store.NodeCount, len(store.Root.Children), leaves, len(f.Store), len(f.TailRest), head)
		g, ok, err := f.Fog()
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		if !ok {
			fmt.Fprintf(out, "  fog       NO RECORD: the store carries no Fog section\n")
			continue
		}
		pct := 0.0
		if len(g.Cells) > 0 {
			pct = 100 * float64(g.Set) / float64(len(g.Cells))
		}
		fmt.Fprintf(out, "  fog       firstState %d   %d runs   %d cells   %d explored (%.1f%%)\n",
			g.FirstState, len(g.Runs), len(g.Cells), g.Set, pct)
	}
	return nil
}
