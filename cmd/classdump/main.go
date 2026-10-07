// Command classdump prints the typed classes of the three graphics registries
// and resolves the classes shipped maps place (developer tool).
//
// Usage:
//
//	classdump <graphics.res>                 print units, objects and structures
//	classdump -sweep <dir> <graphics.res>    resolve every class placed by every .alm under <dir>
//	classdump -databin <world.res> [<map.alm> [<difficulty>]]
//	                                         census the definition table, and resolve one map's
//	                                         placements against it at a difficulty (default 2)
//	classdump -withdraw <asset-root>          census withdrawal definitions and placements
//
// classdump reads units/units.reg, objects/objects.reg and
// structures/structures.reg out of the given archive with pkg/formats/res,
// parses each with pkg/formats/reg, loads them through pkg/data — which is where
// inheritance is resolved — and prints every class of every collection with its
// resolved keys and its sprite and overlay paths.
//
// -sweep loads the same three collections and then resolves the placed class
// references of every map under <dir> against them, printing a census per source
// (see sweep.go). It does not print the classes themselves: the census is a
// small table of fixed tokens and 182 class blocks ahead of it would bury them.
//
// -databin reads the definition table out of the archive it is given — the
// archive's own identity plus data/data.bin, resolved through pkg/vfs — walks it
// and reports the bytes consumed against the node's size, every collection's
// entry and title counts, and which Units rows carry parameters. Named a map as
// well, it resolves each of that map's placements against the table and prints
// the arm it took, the entry it reached, and the health maximum and MOVEMENT
// DOMAIN the world built at the chosen difficulty carries — then counts the
// placements in each of the three domains, zeros included, summing to the
// placement count. Both per-placement numbers are read off the world rather than
// recomputed here, so the report witnesses what a player's world holds. It then
// censuses that map's placed STRUCTURES against the table's Buildings
// collection, at both levels: what happened to each placement, and what happened
// to each cell a footprint named. It reads the two registries not at all: the
// table and the graphics registries are different files with different
// consumers, and this verb is the only one of the three that touches the table.
//
// It closes with the TEMPLATE COMBAT NUMBERS: one row per placement giving the
// class key pair, the entry the placement resolved to or that it resolved to
// none, and the eight numbers a fight reads — the attack charge and relax, the
// to-hit, the defence, the absorption, the damage base and spread, and whether
// the blows always land — with the SIGHT RANGE beside them, which is not one of
// the eight and carries none of the equipment caveat below: it is a column of
// both bands and nothing a placement wears moves it. Every one is read off the
// built world, at the chosen difficulty, so the rows say what a player's world
// holds rather than what a second reading of the table would.
//
// THAT REPORT'S OWN HEADING STATES WHICH BAND EACH ROW IS, and the split is
// load-bearing rather than decorative. A CREATURE row is the TEMPLATE's numbers
// and EQUIPMENT IS NOT APPLIED: such a class fights at its weapon's cadence and
// at a damage, absorption, defence and reach the equipment moves, none of which
// is modelled, so a row compared against such a class will disagree and should.
// A PERSON row IS ARMED — his equipment is resolved and his weapon's numbers are
// already in the row — so the same sentence over both would have told a reader
// to discount the very numbers the person band exists to deliver. The charge and
// relax columns are on no panel the game displays and are compared against
// nothing at all.
//
// -databin THEN CLOSES WITH THE WHOLE CHARACTER SHEET: one block per
// placement, naming the placement, its band and the entry it resolved to,
// and stating the four statistics, health and mana as current over maximum,
// the damage range, absorption, attack, defence, the five weapon-keyed skill
// positions, the elemental family, weight, the row's own experience VALUE,
// sight and speed. Weight is unstated on EVERY row — no definition row
// carries one — and a PERSON's experience value is unstated too, because
// that column belongs to the units collection alone; both print as a dash
// and never as a zero, with the report's own legend saying why. The pools,
// the damage range, absorption, attack, defence, sight and speed are read
// off the same built world the template numbers above already are; only the
// four statistics, the skill positions and the elemental family come off the
// placement's own definition row or its derived-stat graph, through
// pkg/mapload's own PlacedSheets, and nothing here recomputes either. The
// -campaign verb's own named-map path prints this same block, through this
// same function, for the one map its [<map name>] argument names.
//
// It is a developer-run tool for verifying the loader against a lawful install
// and is never part of the test suite's game-facing path. The archive is
// whatever path you pass on the command line; classdump embeds none (golden rule
// 3). Its output is converted game data and must never be committed — send it to
// your own screen or a git-ignored path, never into this repository (golden rule
// 1).
//
// Exit codes follow regtool: 2 for a usage error, 1 for a failure to read, parse
// or load — or for a placed reference that does not resolve — 0 otherwise.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/res"
)

// The three registry entries, as spec.md's source contract names them. Lookup
// through pkg/formats/res folds case and accepts either separator, so these are
// written in the archive's own form and no fold is applied here.
const (
	unitsEntry      = "units/units.reg"
	objectsEntry    = "objects/objects.reg"
	structuresEntry = "structures/structures.reg"
)

// sweepFlag selects the sweep half. It is matched as a literal first argument
// rather than through the flag package, because the tool's two invocations are
// two fixed argument shapes and regtool's subcommand dispatch is the precedent.
const sweepFlag = "-sweep"

// databinFlag selects the definition-table verb, matched as a literal first
// argument for the same reason -sweep is: this tool's invocations are fixed
// argument shapes, not a flag set.
const databinFlag = "-databin"

// regenFlag selects the regeneration census (0109 AC-10), on the same literal
// dispatch as the three above it.
const regenFlag = "-regen"

// withdrawFlag selects the ranged-withdrawal corpus and mission witness (1037).
const withdrawFlag = "-withdraw"

// errUsage marks an argument-shape error. It exists because the argument count
// is no longer one number: main cannot decide "is this a usage error" by
// counting, so run says which it is, and the two exit codes stay what regtool
// fixed — 2 for a usage error, 1 for a failure of the work itself.
var errUsage = errors.New("usage")

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage(os.Stdout)
		return
	}
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "classdump:", err)
		if errors.Is(err, errUsage) {
			usage(os.Stderr)
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: classdump <graphics.res>")
	fmt.Fprintln(w, "       classdump -sweep <dir> <graphics.res>")
	fmt.Fprintln(w, "       classdump -databin <world.res> [<map.alm> [<difficulty 1|2|3>]]")
	fmt.Fprintln(w, "       classdump -campaign <asset-root> [<map name>]")
	fmt.Fprintln(w, "       classdump -regen <asset-root> [<map name>]")
	fmt.Fprintln(w, "       classdump -withdraw <asset-root>")
	fmt.Fprintln(w, "  print the unit, object and structure classes of the archive's three registries,")
	fmt.Fprintln(w, "  resolve every class placed by every .alm under <dir> against them,")
	fmt.Fprintln(w, "  census the definition table and resolve one map's placements against it,")
	fmt.Fprintln(w, "  resolve every shipped map's placements by arm and read its drop cells,")
	fmt.Fprintln(w, "  census the regeneration columns of the rows and of one map's placements,")
	fmt.Fprintln(w, "  or census withdrawal thresholds across every shipped map")
}

// run is main minus the process: it takes the argument list and the stream the
// output goes to, and reports a failure as an error rather than an exit status,
// so the whole tool is exercisable without a process or an install.
func run(args []string, w io.Writer) error {
	switch {
	// A lone verb name is that verb missing its arguments, not an archive that
	// happens to be named for it: reading it as a path would report "no such
	// file" for an invocation whose fault is its shape.
	case len(args) == 1 && args[0] != sweepFlag && args[0] != databinFlag &&
		args[0] != campaignFlag && args[0] != regenFlag && args[0] != withdrawFlag:
		c, err := openClasses(args[0])
		if err != nil {
			return err
		}
		return dumpClasses(w, c)
	case len(args) == 3 && args[0] == sweepFlag:
		c, err := openClasses(args[2])
		if err != nil {
			return err
		}
		return sweepDir(args[1], c, w)
	// The table verb takes an archive and, optionally, a map and a difficulty.
	// It does not open the graphics registries at all: they answer a different
	// question and 182 class blocks ahead of this census would bury it.
	case len(args) >= 2 && len(args) <= 4 && args[0] == databinFlag:
		return runDataBin(args[1:], w)
	// The campaign census takes an asset ROOT and not an archive, because the
	// corpus the published arm figures are measured over is both halves of an
	// install: the 28 maps inside the campaign container and the 10 loose ones
	// beside it.
	case len(args) == 2 && args[0] == campaignFlag:
		return runCampaign(args[1], "", w)
	case len(args) == 3 && args[0] == campaignFlag:
		return runCampaign(args[1], args[2], w)
	// The regeneration census takes an asset ROOT for the same reason the
	// campaign one does: the table and the maps live in two archives beside each
	// other, and a placement's answer needs both.
	case len(args) == 2 && args[0] == regenFlag:
		return runRegen(args[1], "", w)
	case len(args) == 3 && args[0] == regenFlag:
		return runRegen(args[1], args[2], w)
	case len(args) == 2 && args[0] == withdrawFlag:
		return runWithdrawal(args[1], w)
	default:
		return fmt.Errorf("%w: classdump [-sweep <dir> | -databin] <archive>", errUsage)
	}
}

// classes is the three loaded collections. Both halves of the tool work from
// one: the print half walks them, the sweep half resolves placed references
// against them, and neither knows how the other got there.
type classes struct {
	units      *data.UnitClasses
	objects    *data.ObjectClasses
	structures *data.StructureClasses
}

// openClasses is the archive-opening half: it reads the three registries out of
// archivePath and loads each. Reading and parsing are the only IO — loading is a
// pure function of the parsed trees.
func openClasses(archivePath string) (*classes, error) {
	a, err := res.Open(archivePath)
	if err != nil {
		return nil, err
	}

	unitReg, err := parseEntry(a, unitsEntry)
	if err != nil {
		return nil, err
	}
	objectReg, err := parseEntry(a, objectsEntry)
	if err != nil {
		return nil, err
	}
	structureReg, err := parseEntry(a, structuresEntry)
	if err != nil {
		return nil, err
	}

	units, err := data.LoadUnitClasses(unitReg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", unitsEntry, err)
	}
	objects, err := data.LoadObjectClasses(objectReg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", objectsEntry, err)
	}
	structures, err := data.LoadStructureClasses(structureReg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", structuresEntry, err)
	}
	return &classes{units: units, objects: objects, structures: structures}, nil
}

// dumpClasses prints all three collections, in the source contract's order.
func dumpClasses(w io.Writer, c *classes) error {
	if err := dump(w, unitsEntry, "Unit", c.units.All()); err != nil {
		return err
	}
	if err := dump(w, objectsEntry, "Object", c.objects.All()); err != nil {
		return err
	}
	return dump(w, structuresEntry, "Structure", c.structures.All())
}

func parseEntry(a *res.Archive, entry string) (*reg.Reg, error) {
	b, err := a.ReadFile(entry)
	if err != nil {
		return nil, err
	}
	r, err := reg.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", entry, err)
	}
	return r, nil
}

// class is the pair of path methods all three class types carry, expressed over
// the pointer so a type missing either fails to compile rather than at run time.
type class[T any] interface {
	*T
	SpritePath() string
	OverlayPath() string
}

// dump prints one collection: a header naming the entry and the class count —
// the 34 / 82 / 66 a manual run reads off — then one block per class in All()
// order, which is numeric section order, so the section name is the index.
//
// A class's keys are printed by walking its exported fields, not from a list
// written here: every exported field of a class type is one inventory key
// (pkg/data), so the walk cannot drift from the inventory the way a second
// copy of it would. The unexported sprite base is skipped and printed
// instead as the two paths it exists for, in lower case so a derived value
// is never mistaken for a key.
func dump[T any, PT class[T]](w io.Writer, entry, sectionPrefix string, list []PT) error {
	if _, err := fmt.Fprintf(w, "%s: %d classes\n", entry, len(list)); err != nil {
		return err
	}
	for i, c := range list {
		if _, err := fmt.Fprintf(w, "[%s%d]\n", sectionPrefix, i); err != nil {
			return err
		}
		if err := printKeys(w, reflect.ValueOf(c).Elem()); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  sprite = %s\n  overlay = %s\n",
			quote(c.SpritePath()), quote(c.OverlayPath())); err != nil {
			return err
		}
	}
	return nil
}

func printKeys(w io.Writer, v reflect.Value) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue // the sprite base: derived, not a key
		}
		val, err := formatValue(v.Field(i))
		if err != nil {
			return fmt.Errorf("%s.%s: %w", t.Name(), f.Name, err)
		}
		if _, err := fmt.Fprintf(w, "  %s = %s\n", f.Name, val); err != nil {
			return err
		}
	}
	return nil
}

// formatValue renders one key at its kind: a string quoted and byte-escaped, an
// int32 in decimal, an []int32 as space-separated elements in brackets (an
// unresolved array, which is nil, printing as the empty list it is). Any other
// kind is a class field that is not one of the three the inventory uses, and is
// an error rather than a guess.
func formatValue(v reflect.Value) (string, error) {
	switch v.Kind() {
	case reflect.String:
		return quote(v.String()), nil
	case reflect.Int32:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.Int32 {
			return "", fmt.Errorf("unrendered slice element kind %s", v.Type().Elem().Kind())
		}
		var b strings.Builder
		b.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(strconv.FormatInt(v.Index(i).Int(), 10))
		}
		b.WriteByte(']')
		return b.String(), nil
	default:
		return "", fmt.Errorf("unrendered field kind %s", v.Kind())
	}
}

func quote(s string) string { return `"` + escapeBytes(s) + `"` }

// escapeBytes renders s's bytes under the convention regtool fixes (0011
// AC-10): a byte in 0x20-0x7E prints as itself and every other byte prints as
// \xNN, two lower-case hex digits, always two. A registry string holds the
// registry's own bytes in an encoding this project maps nowhere, so this is the
// only function here that turns one into a character, and the output is pure
// ASCII by construction.
func escapeBytes(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x20 && c <= 0x7e {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, `\x%02x`, c)
		}
	}
	return b.String()
}
