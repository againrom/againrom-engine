// Command appearcheck reads a lawful install and reports what each generated
// archetype starts wearing and holding, the body he is drawn as, and whether
// that body LOADS through the game's own hero-body loader.
//
//	appearcheck [-assets DIR]
//
// It opens the install through the SAME door the game itself opens through,
// game.NewFrontEnd, and for each of the four generated archetypes — the two
// class choices crossed with the two sex choices — turns a confirmed
// ui.ChargenResult into a party through FrontEnd.ChargenParty, the exact call
// the generation screen makes on Confirm. Nothing here re-derives a worn set,
// a body name, a directory or a class: every one of the four is read straight
// off the mapload.PartyMember that call already produced.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "appearcheck:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("appearcheck", flag.ContinueOnError)
	fs.SetOutput(out)
	assets := fs.String("assets", "", "game install root (or AGAINROM_ASSETS)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	// NewFrontEnd IS THE ONE DOOR (spec Concrete shape): everything this tool
	// reports — the table, the body list, the unit bundle and the container
	// filesystem a body's load is probed through — comes off the SAME
	// construction the windowed game itself runs, never a second walk of the
	// install this file invents.
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return err
	}
	printReport(out, f)
	return nil
}

// The chargen row order (chargenRowSex, chargenRowClass, chargenRowSkill)
// RESTATES pkg/game's own unexported chargenChoiceSex/chargenChoiceClass/
// chargenChoiceSkill (chargen.go): the position each choice holds in a
// ui.ChargenResult.Choices slice, which FrontEnd.ChargenParty reads by that
// same position. A developer tool under cmd/ cannot reach an unexported
// constant of a tier it is allowed to import — cmd/wearcheck/main.go's own
// cellWeapon does the identical restatement for the identical reason — so the
// three positions are restated here rather than imported, and pkg/game's own
// tests (chargen_test.go) are what keeps this restatement honest: every one
// of them builds a ui.ChargenResult by this same [sex, class, skill] order.
const (
	chargenRowSex   = 0
	chargenRowClass = 1
	chargenRowSkill = 2
)

// trainedSkillChoice is the ONE skill row index every archetype this tool
// prints is confirmed with, and trainedSkillSlot is chargen.go's own row
// math applied to it — `data.SkillBlade + int32(chosen index)`, restated for
// chargenRowSex's own reason above. Printing which one was used, once per
// block, is what a reader needs to tell a mage's own one weapon apart from a
// fighter's five without re-reading this source (spec T5's own note).
const trainedSkillChoice = 0

var trainedSkillSlot int32 = data.SkillBlade + trainedSkillChoice

// printReport prints one block per archetype — the two class choices crossed
// with the two sex choices, fighter before mage and male before female, an
// order this file chose and nothing decoded — and then the shipped body list
// once.
func printReport(out io.Writer, f *game.FrontEnd) {
	for _, class := range []bool{false, true} {
		for _, female := range []bool{false, true} {
			res := ui.ChargenResult{Choices: make([]int, 3)}
			if female {
				res.Choices[chargenRowSex] = 1
			}
			if class {
				res.Choices[chargenRowClass] = 1
			}
			res.Choices[chargenRowSkill] = trainedSkillChoice

			party := f.ChargenParty(res)
			if len(party) == 0 {
				fmt.Fprintf(out, "%s / %s: ChargenParty answered no member\n\n",
					classLabel(class), sexLabel(female))
				continue
			}
			printBlock(out, f.Archives.Containers, f.Units.Classes, f.Table, classLabel(class), sexLabel(female), party[0])
		}
	}
	printBodyList(out, f.Bodies)
	printSkillWitness(out, f)
}

func classLabel(mage bool) string {
	if mage {
		return "mage"
	}
	return "fighter"
}

func sexLabel(female bool) string {
	if female {
		return "female"
	}
	return "male"
}

// printBlock prints one archetype's block exactly as the spec's own I/O
// example shows — the archetype, the weapon he was handed (or that he holds
// none), each occupied slot with its number, the resolved item name and its
// code, the composed body name, the directory, the composed sheet address,
// the drawn class and whether that body LOADS through the game's own loader
// — plus the trained skill choice this tool used (trainedSkillSlot's own
// doc).
//
// src is the front end's own container filesystem (terrain.EntrySource:
// f.Archives.Containers satisfies it, and so does a test's in-memory stand-in
// with no archive behind it at all, golden rule 2), and classes is the front
// end's own unit bundle's Classes map (f.Units.Classes) — the class records
// game.LoadHeroBody needs to resolve a body name against, read-only here and
// never written by this file or by LoadHeroBody itself.
//
// EVERY VALUE BUT THE SLOT NUMBER AND THE ARCHETYPE LABELS COMES OFF m OR t
// (spec: no shipped name, count or byte in this source): m is what
// FrontEnd.ChargenParty already assembled, and t is the install's own
// definition table.
func printBlock(out io.Writer, src terrain.EntrySource, classes map[int32]*terrain.UnitClass, t *mapload.Table, classLbl, sexLbl string, m mapload.PartyMember) {
	weapon := "none"
	if m.Weapon != nil {
		weapon = m.Weapon.Name
	}
	fmt.Fprintf(out, "%s / %s   weapon=%s\n", classLbl, sexLbl, weapon)
	fmt.Fprintf(out, "  trained %s\n", data.SkillName(trainedSkillSlot))

	for slot := 1; slot <= data.EquipSlots; slot++ {
		code := data.ItemCode(m.Worn[slot-1])
		if code.D() == 0 {
			continue //
		}
		if name, ok := resolveItemName(slot, code, t); ok {
			fmt.Fprintf(out, "  slot %d %s 0x%04x\n", slot, name, uint16(code))
		} else {
			fmt.Fprintf(out, "  slot %d <code names nothing this build can resolve> 0x%04x\n", slot, uint16(code))
		}
	}

	body := data.HeroBody(m.Body)
	sheet := data.HeroSheetPath(m.BodyDir, body)

	// probe is a FRESH *terrain.UnitSet, sharing only the front end's own
	// Classes map (LoadHeroBody reads Classes and never writes it) — so one
	// archetype's probe cannot satisfy another's, and the front end's own
	// Bodies bundle is never touched: Bodies here starts nil, exactly as a
	// bundle a test assembles by hand does, and LoadHeroBody allocates its
	// own map on the one entry it writes.
	probe := &terrain.UnitSet{Classes: classes}
	game.LoadHeroBody(src, probe, m.BodyDir, body)
	_, loaded := probe.Bodies[data.HeroBodyKey(m.BodyDir, body)]
	loads := "no"
	if loaded {
		loads = "yes"
	}
	fmt.Fprintf(out, "  body=%s dir=%s class=%d sheet=%s (address inside the graphics container) loads=%s\n\n",
		m.Body, m.BodyDir, m.Class, sheet, loads)
}

// resolveItemName is worn slot slot's own item name, read off code against
// t's collections on cmd/paneldump/main.go's own wornNames rule, restated
// here for chargenRowSex's own reason above: THE SLOT DECIDES THE
// COLLECTION, never the code's own class field — slot 1 against Weapons,
// slot 2 against Shields, every other slot against Armors — because an
// armour's own Slot column is not always one of the twelve real slots
// (data.Armor.Slot's own doc) and so cannot be trusted to name its own class
// back. THE ROW NAME ALONE is answered, wornNames' own choice: the row is the
// piece's identity, and this tool states that identity, not the shape or
// material word the piece's own resolution scaled it by.
//
// A nil table, a nil collection for the slot's own class, a row past the
// collection's length or a row whose entry is itself empty all answer
// ("", false) — UNLIKE wornNames, which drops such a slot silently, this
// tool's caller REPORTS the refusal rather than printing an empty name
// (spec T5's own instruction).
func resolveItemName(slot int, code data.ItemCode, t *mapload.Table) (string, bool) {
	if t == nil {
		return "", false
	}
	var coll data.Collection
	switch slot {
	case 1:
		coll = t.Weapons
	case 2:
		coll = t.Shields
	default:
		coll = t.Armors
	}
	if coll == nil {
		return "", false
	}
	row := code.D()
	if row < 0 || row >= coll.Len() {
		return "", false
	}
	name := coll.EntryName(row)
	return name, name != ""
}

// printBodyList prints the shipped body list ONCE, in full, with each line's
// own zero-based index and the drawn class data.HeroBodyClass resolves the
// name to (spec T5's own instruction) — the table HeroBodyFor reads every
// per-archetype body name out of, off equipment slot 1's own field D
// (data.HeroBodyFor's own doc), so a reader can check one block's body
// name against the list that produced it without opening a second tool.
func printBodyList(out io.Writer, list data.BodyList) {
	fmt.Fprintf(out, "body list (%d entries; index: name -> drawn class):\n", list.Len())
	for i := 0; i < list.Len(); i++ {
		name := list.Entry(i)
		class, matched := data.HeroBodyClass(name)
		if !matched {
			fmt.Fprintf(out, "  %d: %q -> class %d (no arm matches this name)\n", i, string(name), class)
			continue
		}
		fmt.Fprintf(out, "  %d: %q -> class %d\n", i, string(name), class)
	}
}

// printSkillWitness prints the compact witness table T5's own correction 2
// asks for: one line per class crossed with each of the five trainable skill
// slots (data.SkillBlade..data.SkillShoot, data.SkillNames' own five),
// naming the class, the skill trained, the weapon FrontEnd.ChargenParty
// handed him, the composed body name, the directory and the drawn class.
//
// EVERY VALUE BUT THE CLASS AND SKILL LABELS COMES OFF m (printBlock's own
// "no shipped name" rule above): m is what FrontEnd.ChargenParty already
// assembled for that one confirmed result, never recomputed here.
func printSkillWitness(out io.Writer, f *game.FrontEnd) {
	fmt.Fprintln(out, "skill witness (class, trained skill -> weapon, body, dir, drawn class):")
	for _, class := range []bool{false, true} {
		names := data.SkillNames(class)
		for choice, skillName := range names {
			res := ui.ChargenResult{Choices: make([]int, 3)}
			if class {
				res.Choices[chargenRowClass] = 1
			}
			res.Choices[chargenRowSkill] = choice

			party := f.ChargenParty(res)
			if len(party) == 0 {
				// assembleParty always answers one member (printReport's own
				// note above); this arm exists for the same reason.
				fmt.Fprintf(out, "  %-7s %-8s ChargenParty answered no member\n",
					classLabel(class), skillName)
				continue
			}
			m := party[0]
			weapon := "none"
			if m.Weapon != nil {
				weapon = m.Weapon.Name
			}
			fmt.Fprintf(out, "  %-7s %-8s weapon=%-16s body=%-12s dir=%-9s class=%d\n",
				classLabel(class), skillName, weapon, m.Body, m.BodyDir, m.Class)
		}
	}
}
