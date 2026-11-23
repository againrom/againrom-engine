package game

import (
	"errors"
	"fmt"
	"io/fs"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
	"againrom/pkg/formats/itempkt"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// tableAddress is where the placeable-definition table lives: WorldArchive's
// identity segment and the entry path below it.
const tableAddress = worldPrefix + "data/data.bin"

// Definitions is everything this package reads before a mission's party can
// be assembled: the collections a world builder resolves against, the weapon
// character generation hands the party's hero, and — since 0105 — the
// shipped body list that weapon's own row is read against to name what he
// wears.
//
// THE TABLE AND THE WEAPON RIDE TOGETHER because they come out of ONE WALK of
// one file. Reading the file twice would let an install answer the placements
// and not the weapon, or the other way about, which is a state no install
// actually has.
//
// BODIES RIDES A SEPARATE READ, off a different archive (main.res, not
// world.res) — there is no shared walk to fold it into. It sits on this
// struct anyway, beside StartWeapon, because MissionParty needs both to
// answer one question — what body the resolved weapon puts on the hero — and
// a caller holding one without the other could only ask half of it.
type Definitions struct {
	Table *mapload.Table

	// StartWeapon is that weapon, or nil when the installed table does not
	// yield it — in which case StartWeaponErr says why.
	//
	// A NIL ONE IS NOT FATAL and the reason is CARRIED, exactly as FontErr is.
	// The party's hero is then BARE, which is a state of the original: he
	// swings for `ftol(1.1^Body/20)` and, at the character-generation start,
	// for nothing. That must not be silent — it is the very defect 0078 exists
	// to remove — so the headless check line reports it, and reports the hero's
	// damage band when there is one.
	//
	// Fatal was considered and refused: the two shipped roots differ byte for
	// byte in this file, and an install whose archive opens must not stop being
	// a game because one row is named differently.
	StartWeapon    *data.Weapon
	StartWeaponErr error

	Bodies data.BodyList
}

// LoadDefinitionsOf reads the placeable-definition table once under p's game
// and returns everything this package takes from it, plus the shipped body
// list read straight off main.res and the build's straight Acid Stream rows.
func LoadDefinitionsOf(fsys *vfs.FS, p base.Profile) (*Definitions, error) {
	d, err := loadDefinitionsFor(fsys, p.GameOf())
	if err == nil {
		d.Table.StraightCone = sim.StraightCone(p.StraightCone)
	}
	return d, err
}

// loadDefinitionsFor reads the table layout of g; the empty game is the first.
func loadDefinitionsFor(fsys *vfs.FS, g base.Game) (*Definitions, error) {
	b, err := fsys.ReadFile(tableAddress)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", tableAddress, err)
	}
	f, err := parseDefinitionTable(b, g)
	if err != nil {
		return nil, err
	}
	npc, err := LoadNPCCatalog(fsys)
	if err != nil {
		return nil, err
	}
	d := &Definitions{Table: readUnder(g, &mapload.Table{
		Units:      f.Collection(databin.Units),
		Humans:     f.Collection(databin.Humans),
		Buildings:  f.Collection(databin.Buildings),
		NPC:        npc.Defs,
		NPCCatalog: npc,
		// THE THREE ITEM COLLECTIONS RIDE ON THE SAME WALK the party's own
		// weapon comes off, one line below. A scenario human names his own
		// equipment and the loader has to be able to say what those names are
		// worth; reading them a second time would let an install answer the
		// hero and not the placements, which is a state no install has.
		Shapes:    f.Collection(databin.Shapes),
		Materials: f.Collection(databin.Materials),
		Weapons:   f.Collection(databin.Weapons),
		// THE TWO REMAINING PIECE COLLECTIONS RIDE ON THE SAME WALK: a scenario
		// person's row names a shield and eight armour cells beside his weapon,
		// and this is the only place a loader can find out what those names are
		// worth. They are read here rather than a second time for the reason the
		// three above already are: a second reader would be a second answer.
		Armors:  f.Collection(databin.Armors),
		Shields: f.Collection(databin.Shields),
		// THE SPELLS COLLECTION RIDES ON THE SAME WALK TOO: a mission world's own
		// cast table comes off this file exactly as its placements do, so reading
		// it a second time would let an install answer the placements and not the
		// spells, which is a state no install actually has.
		Spells:     f.Collection(databin.Spells),
		Magic:      f.Collection(databin.Magic),
		MagicItems: f.Collection(databin.MagicItems),
	})}
	costs, haveCosts, err := terrainCostTable(fsys, filesOf(g).terrainCosts)
	if err != nil {
		return nil, err
	}
	if haveCosts {
		d.Table.TerrainCosts = &costs
	}
	d.StartWeapon, d.StartWeaponErr = resolveStartingWeapon(f, PartySkillSlot())
	// Not fatal, and reported by nothing beyond the field itself — see
	// Definitions.Bodies.
	d.Bodies, _ = ReadBodyList(fsys)
	if d.Table.FlightInputs, err = loadFlightInputs(fsys, data.SpellPresentationNamed(g.Edition().Spells), filesOf(g).flight); err != nil {
		return nil, err
	}
	// NOT FATAL, for the same reason: a missing item-name table narrows what
	// itemName (world.go) can show a code by, it does not stop a mission
	// from opening. Its lines are main.res text, not world.res, so it rides
	// beside the table rather than inside the walk above.
	d.Table.Names, _ = readItemNamesAt(fsys, filesOf(g).itemNameKeys, InstallTextCode(fsys, g.Edition()))
	if addr := filesOf(g).itemPacket; addr != "" {
		raw, readErr := fsys.ReadFile(addr)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", addr, readErr)
		}
		if d.Table.ItemRecords, err = itempkt.Parse(raw); err != nil {
			return nil, err
		}
	}
	// NOT FATAL either: a table that read fewer than the four hero names carries
	// none, and MissionPartyAs then names a hero started without the generator's
	// name field itself. The generator reads the same four through the same
	// reader and refuses to open without them.
	if names, err := heroNames(fsys, g.Edition(), InstallTextCode(fsys, g.Edition())); err == nil {
		d.Table.HeroNames = names
	}
	// Not fatal: a table without them names no map placement for the hero
	// ordinal scan, which then reaches the party alone.
	if rows := LoadTextTable(fsys, npcNamesPath, InstallTextCode(fsys, g.Edition())); rows != nil {
		d.Table.NPCNames = rows.lines
	}
	return d, nil
}

// terrainCostTable is the installed Terrain cost table at addr, the one every
// first-game World prices cells with (ALM-TERR-043); the sim, the cell planes
// and the map checker read it here. ok is false for an install without the
// file, and for a game naming no address: they keep the compiled table
// (DIV-3093).
func terrainCostTable(src entrySource, addr string) (costs [11]byte, ok bool, err error) {
	if addr == "" {
		return costs, false, nil
	}
	raw, err := src.ReadFile(addr)
	if errors.Is(err, fs.ErrNotExist) {
		return costs, false, nil
	}
	if err != nil {
		return costs, false, fmt.Errorf("original terrain parameters: %w", err)
	}
	costs, err = data.OriginalTerrainCostTable(raw)
	if err != nil {
		return costs, false, fmt.Errorf("original terrain parameters: %w", err)
	}
	return costs, true, nil
}

// parseDefinitionTable parses the bytes of the placeable-definition table
// under the layout of game g; the empty game is the first.
func parseDefinitionTable(b []byte, g base.Game) (*databin.File, error) {
	f, err := databin.ParseWith(b, filesOf(g).table)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", tableAddress, err)
	}
	return f, nil
}

// readUnder gives t the edition of game g and the placement decoder's values
// its campaign service's files carry, and returns it.
func readUnder(g base.Game, t *mapload.Table) *mapload.Table {
	edition, files := g.Edition(), filesOf(g)
	t.Edition = &edition
	t.UnitKeys, t.SpellArms, t.FreshPlayers = files.unitKeys, files.spellArms, files.freshPlayers
	t.CarryStockOnly = files.carryStockOnly
	t.StoredFormMerge, t.CreatureBooks = files.storedFormMerge, files.creatureBooks
	t.CreatureDifficulty = files.creatureDifficulty
	t.CarriedWornSpellBuild = files.storedWeaponSpell
	t.SchoolPoolExperience = files.schoolPoolExperience
	t.ClassStatCaps, t.ActorKindWords = files.classStatCaps, files.actorKindWords
	t.ItemNamer = files.itemNamer
	t.DeathItems = nil
	if files.deathItems != nil {
		t.DeathItems = files.deathItems(t)
	}
	return t
}

// LoadNPCCatalog reads the scenario NPC registry out of the container
// filesystem once. The placement table, the dialogue portraits and the map
// editor's speaker list all ask the one catalog it answers (data.NPCCatalog).
//
// It is a function of its own beside LoadDefinitionsOf, rather than inline in it,
// because the two files live in two different archives and a developer tool
// pointed at one of them must be able to load the other's half without being
// handed a whole install.
func LoadNPCCatalog(fsys *vfs.FS) (*data.NPCCatalog, error) {
	b, err := fsys.ReadFile(NPCRegistry)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", NPCRegistry, err)
	}
	r, err := reg.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", NPCRegistry, err)
	}
	return data.LoadNPCCatalog(r), nil
}

// npcNamesPath is the text file the npc arm names a map placement from.
const npcNamesPath = mainPrefix + "text/npcnames.txt"
