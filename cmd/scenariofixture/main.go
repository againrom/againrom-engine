// Command scenariofixture prepares the controlled mission-20 endpoint used by
// the legacy frontend transition scenarios. It never decides the mission.
package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/game"
	"againrom/pkg/sim"
)

const sourceSHA256 = "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6"
const fixtureLabel = "fresh mission 20 endpoint - save666 party"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("scenariofixture", flag.ContinueOnError)
	flags.SetOutput(stderr)
	assets := flags.String("assets", "", "read-only lawful install (required)")
	source := flags.String("source", "", "read-only save-666 game0009.sav (required, SHA-256 checked)")
	out := flags.String("out", "", "empty existing output directory outside the install and source directory (required)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *assets == "" || *source == "" || *out == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "scenariofixture: -assets, -source and -out are required")
		return 2
	}
	if err := prepare(*assets, *source, *out, stdout); err != nil {
		fmt.Fprintln(stderr, "scenariofixture:", err)
		return 1
	}
	return 0
}

func prepare(assets, source, out string, stdout io.Writer) error {
	if err := validateOutput(assets, source, out); err != nil {
		return err
	}
	saved, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if digest := fmt.Sprintf("%x", sha256.Sum256(saved)); digest != sourceSHA256 {
		return fmt.Errorf("source SHA-256 %s, want save-666 %s", digest, sourceSHA256)
	}
	game.SetSoundOptions(game.SoundOptions{})
	front, err := game.NewFrontEnd(assets)
	if err != nil {
		return err
	}
	front.SetDeterministicFrames(true)
	open, town, err := front.RestoreOriginal(saved)
	if err != nil {
		return err
	}
	if town || open == nil {
		return errors.New("source did not prepare a mission")
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		return err
	}
	imported, _, err := front.Snapshot(true)
	if err != nil {
		return err
	}
	if imported.Mission != 20 {
		return fmt.Errorf("source mission %d, want 20", imported.Mission)
	}
	// Save 666 has already spent mission 20's completion latch, while its
	// original outcome is outside the supported import. It cannot win again.
	// Build a NEW mission from its party and purse; never clear an imported
	// latch or pretend this controlled fixture continues the original world.
	party := front.LiveParty()
	if _, _, _, _, _, _, _, _, _, _, err := front.MissionOpenerWith(20, party)(); err != nil {
		return err
	}
	snapshot, _, err := front.Snapshot(true)
	if err != nil {
		return err
	}
	// Modify a detached native snapshot, not the frontend's active world.
	// HeadlessPlace is the script's placement of the escort near its cell. No Step runs
	// here: the loaded scenario must let the installed script raise Victory.
	var world sim.World
	if err := world.UnmarshalBinary(snapshot.World); err != nil {
		return err
	}
	escort, err := placeEscort(&world)
	if err != nil {
		return err
	}
	snapshot.World, err = world.MarshalBinary()
	if err != nil {
		return err
	}
	encoded, err := front.ExportCurrentSave(snapshot, fixtureLabel)
	if err != nil {
		return err
	}
	// Scenarios choose the saved label; slot names need no fixed filename.
	name, err := (game.SaveStore{Dir: out}).WriteOriginal(assets, encoded)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "source_sha256=%s\nfixture=%s\nfixture_sha256=%x\nlabel=%s\ncontrolled_move=map_unit:136 entity:%d placed near (111,132) at (%d,%d)\nmission=20 tick=%d outcome=undecided; NEW mission with save666 party and purse, fresh script latches unchanged; NOT original continuation or campaign playthrough\n",
		sourceSHA256, filepath.Join(out, name), sha256.Sum256(encoded), fixtureLabel,
		escort.ID, escort.X, escort.Y, world.Tick())
	return nil
}

func placeEscort(world *sim.World) (sim.Entity, error) {
	if world.Outcome() != sim.OutcomeUndecided {
		return sim.Entity{}, errors.New("endpoint fixture requires an undecided source world")
	}
	var escort sim.Entity
	count := 0
	for _, entity := range world.Entities() {
		if entity.MapUnitID == 136 {
			escort = entity
			count++
		}
	}
	if count != 1 || escort.HP <= 0 || escort.OffMap {
		return sim.Entity{}, fmt.Errorf("expected one living on-map escort u136; matches=%d hp=%d off_map=%v", count, escort.HP, escort.OffMap)
	}
	if err := world.HeadlessPlace(escort.ID, 111, 132); err != nil {
		return sim.Entity{}, err
	}
	placed, _ := world.Entity(escort.ID)
	return placed, nil
}

func validateOutput(assets, source, out string) error {
	canonical := func(path string) (string, error) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			return resolved, nil
		}
		// Windows may refuse reparse metadata. The physical ancestor check
		// below still rejects aliases into the protected input directories.
		return filepath.Clean(abs), nil
	}
	target, err := canonical(out)
	if err != nil {
		return fmt.Errorf("output must be an existing empty directory: %w", err)
	}
	for _, path := range []string{assets, filepath.Dir(source)} {
		protected, err := canonical(path)
		if err != nil {
			return err
		}
		// Case folding also rejects Windows aliases for the same directory.
		targetFold, protectedFold := strings.ToLower(target), strings.ToLower(protected)
		if targetFold == protectedFold || strings.HasPrefix(targetFold, protectedFold+string(filepath.Separator)) {
			return fmt.Errorf("output %q is inside protected input directory %q", target, protected)
		}
		inputInfo, err := os.Stat(protected)
		if err != nil {
			return err
		}
		for probe := target; ; probe = filepath.Dir(probe) {
			info, err := os.Stat(probe)
			if err != nil {
				return err
			}
			if os.SameFile(inputInfo, info) {
				return fmt.Errorf("output %q aliases protected input directory %q", target, protected)
			}
			if filepath.Dir(probe) == probe {
				break
			}
		}
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("output directory must be empty; existing files are never removed")
	}
	return nil
}
