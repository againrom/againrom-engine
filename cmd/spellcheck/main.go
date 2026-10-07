// Command spellcheck reports what a cast can draw against a lawful install
// (developer tool, docs/0161-spell-art).
//
// It loads graphics/projectiles/projectiles.reg and every sheet behind it
// through the same loader the game uses, then prints three tables: the built
// sheets with the frame law each satisfies, what each of the 28 spell ids
// reaches at both parities, and one flying spell walked tick by tick.
//
// It is a developer-run tool for verifying the loader against a real install and
// is never part of the test suite. Its output is converted game data and must
// never be committed — send it to your own screen or a git-ignored path (golden
// rule 1). The asset root comes from -assets or AGAINROM_ASSETS and this file
// contains no install path (golden rule 3).
//
// Usage:
//
//	spellcheck [-assets <root>] [-spell <id>] [-cells <n>]
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

func main() {
	assets := flag.String("assets", "", "game install root (defaults to AGAINROM_ASSETS)")
	spell := flag.Int("spell", 1, "spell id to walk tick by tick")
	cells := flag.Int("cells", 5, "caster-to-target distance in cells for the walk")
	flag.Parse()

	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		fmt.Fprintln(os.Stderr, "spellcheck: no asset root: pass -assets or set AGAINROM_ASSETS")
		os.Exit(1)
	}
	if err := run(root, *spell, *cells); err != nil {
		fmt.Fprintln(os.Stderr, "spellcheck:", err)
		os.Exit(1)
	}
}

func run(root string, spell, cells int) error {
	archives, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	set, err := game.LoadProjectiles(archives.Containers)
	if err != nil {
		return err
	}

	ids := make([]int, 0, len(set.Sheets))
	for id := range set.Sheets {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	fmt.Printf("sheets built: %d\n", len(ids))
	for _, id := range ids {
		s := set.Sheets[id]
		// The frame law the draw implies: Phases per stored facing, nine of
		// them when the halving bit is set.
		law := s.Phases * s.RotationPhases
		if s.Flip {
			law = s.Phases * 9
		}
		mark := " "
		if law != len(s.Frames) {
			mark = "!"
		}
		w, h, painted := 0, 0, 0
		if f := s.Frame(0); f != nil {
			w, h = f.Width, f.Height
			for _, p := range f.Pixels {
				if p.A != 0 {
					painted++
				}
			}
		}
		fmt.Printf("%s pic %2d frames %3d law %3d phases %2d rot %2d flip %-5v clock %d "+
			"art %dx%d painted %d centre %d,%d\n",
			mark, id, len(s.Frames), law, s.Phases, s.RotationPhases, s.Flip, s.Clock,
			w, h, painted, s.CenterX, s.CenterY)
	}
	heal, err := game.InspectProjectileArt(archives.Containers, 20)
	if err != nil {
		return err
	}
	fmt.Printf("heal picture 20 path %q payload sha256 %x frames-walked %d phases-accessible %d art %dx%d centre %d,%d\n",
		heal.Path, heal.PayloadSHA256, heal.Frames, heal.Phases, heal.Width, heal.Height, heal.CenterX, heal.CenterY)

	fmt.Println("--- what each spell id reaches")
	casts, bursts, flying := 0, 0, 0
	for id := 1; id <= 28; id++ {
		cast, burst := data.CastPicture(id), data.BurstPicture(id)
		cs, bs := set.Sheet(cast) != nil, set.Sheet(burst) != nil
		life := data.CastFlight(cast, cells*data.PictureCellUnits)
		if cs {
			casts++
		}
		if bs {
			bursts++
		}
		if life > 0 {
			flying++
		}
		if !cs && !bs && life == 0 {
			continue
		}
		fmt.Printf("spell %2d: cast %2d sheet %-5v flight %2d | burst %2d sheet %-5v life %d\n",
			id, cast, cs, life, burst, bs, data.BurstLife(burst))
	}
	fmt.Printf("cast sheets %d, burst sheets %d, spells that fly %d\n", casts, bursts, flying)

	fmt.Printf("--- spell %d over %d cells east\n", spell, cells)
	picture := data.CastPicture(spell)
	s := set.Sheet(picture)
	if s == nil {
		fmt.Printf("picture %d names no sheet\n", picture)
		return nil
	}
	facing := terrain.EffectFacing(cells, 0)
	for age := range data.CastFlight(picture, cells*data.PictureCellUnits) {
		phase, ok := terrain.EffectPhase(s.Clock, age, s.Phases)
		if !ok {
			fmt.Printf("  age %d: no phase\n", age)
			continue
		}
		frame, mirror, ok := terrain.SelectEffectFrame(s, facing, phase)
		fmt.Printf("  age %2d facing %2d phase %2d -> frame %3d mirror %-5v drawn %v\n",
			age, facing, phase, frame, mirror, ok)
	}

	// 1004's own witness: the whole cast driven through the production
	// observation and the production draw, on this install's art.
	fmt.Printf("--- spell %d figure, drawn tick by tick (1004)\n", spell)
	figure, err := game.InspectCastFigure(archives.Containers, spell, cells)
	if err != nil {
		return err
	}
	fmt.Printf("  smoke sheets loaded: %v %v\n",
		set.SmokeSheet(0) != nil, set.SmokeSheet(1) != nil)
	for _, f := range figure {
		fmt.Printf("  age %2d of %2d: %3d stamps, %d trail, %d burst, %v..%v, max offset %d units\n",
			f.Age, f.Life, f.Stamps, f.Trail, f.Burst, f.First, f.Last, f.MaxOffset)
	}

	// The same picture through the other producer: a staff whose castSpell is
	// this spell, released on the swing rather than from a book.
	fmt.Printf("--- spell %d from a weapon, drawn swing by swing (1004 FR-7)\n", spell)
	staff, err := game.InspectWeaponRelease(archives.Containers, spell, cells)
	if err != nil {
		return err
	}
	for _, f := range staff {
		fmt.Printf("  swing %2d of %2d: %3d stamps, %d trail, %v..%v, max offset %d units\n",
			f.Age, f.Life, f.Stamps, f.Trail, f.First, f.Last, f.MaxOffset)
	}
	return nil
}
