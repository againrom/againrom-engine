package game

import (
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/vfs"
)

// tableAddress is where the placeable-definition table lives: WorldArchive's
// identity segment and the entry path below it.
const tableAddress = worldPrefix + "data/data.bin"

// LoadTable reads the placeable-definition table out of the container
// filesystem and returns the two collections a world builder resolves against.
//
// It is the ONE place this package reads that file, and it is called once, at
// front-end construction, so the walk runs once per process rather than once per
// trip through the map picker.
//
// EVERY WAY IT CAN FAIL IS A STARTUP FAILURE, not a table a caller may go on
// without. The archive being absent is already fatal, and what lies behind that
// archive is the movement domain and the per-class health of every placed unit —
// so an archive that opens but yields no table, or yields bytes that will not
// walk, has to be fatal too. Were it not, the archive check would guard nothing:
// the same all-ground, one-health world would lie behind both doors, and only
// one of them would say so.
//
// It returns the collections and not the parsed file. A world builder resolves
// against Units, Humans and Buildings and reads none of the other eight, and
// handing over the whole file would let a later caller reach for a collection
// whose meaning nothing in this tree has established. Three more of the eight
// now have one — Shapes, Materials and Weapons, which between them say what an
// item of a given name carries — and they are read HERE, once, by
// LoadDefinitions below. They now ride on the returned table as well, because a
// scenario human names his own equipment and the world builder is what has to
// resolve it; the walk is still one walk.
//
// THE BUILDINGS COLLECTION IS THE THIRD, and it was missing here for four
// stories while every builder downstream already took the table it rides on.
// What resolves against it is the placed-structure footprint pass, which is
// the second stage of the block plane every world routes on: with no
// collection it resolves nothing, so a placed building closed no cell and
// — worse, because it is the arm nobody looks for — the SUBTRACTIVE arm
// never ran and every bridge deck kept the blocked terrain the ingest stage
// gave it. The defect looked like two (buildings do not collide; the bridge
// is impassable) and like a fault of the map screen, because that is the
// screen it was seen on; it was one missing field, and it broke the mission
// path identically.
//
// It is read here and not somewhere structure-shaped for the reason the other
// two are read here: what a placement is worth is a property of the installed
// file, this is the one place this package reads that file, and a second reader
// would be a second answer.
//
// THE SCENARIO NPC REGISTRY IS READ HERE TOO, and its absence is fatal for
// this function's own reason. It is not a second table: it is the first rung
// of the humans band, and what lies behind it is WHO fifteen shipped
// placements are. A front-end that ran without it would put a generic
// peasant where a mission's own dialogue names a witch — silently, with
// every test still green, which is exactly the shape of failure the
// paragraph above exists to close.
func LoadTable(fsys *vfs.FS) (*mapload.Table, error) {
	d, err := LoadDefinitions(fsys)
	if err != nil {
		return nil, err
	}
	return d.Table, nil
}

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

// LoadDefinitions reads the placeable-definition table once and returns
// everything this package takes from it, plus the one thing that is not part
// of that walk: the shipped body list, read straight off fsys because it
// lives in a different archive (main.res, not world.res). LoadTable is this
// function's first field.
func LoadDefinitions(fsys *vfs.FS) (*Definitions, error) {
	return LoadDefinitionsFor(fsys, "")
}

// LoadDefinitionsFor is LoadDefinitions over the table layout of g; the empty
// game is the first.
func LoadDefinitionsFor(fsys *vfs.FS, g base.Game) (*Definitions, error) {
	b, err := fsys.ReadFile(tableAddress)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", tableAddress, err)
	}
	f, err := databin.ParseWith(b, databinLayout(g))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", tableAddress, err)
	}
	npc, err := LoadNPCDefs(fsys)
	if err != nil {
		return nil, err
	}
	d := &Definitions{Table: &mapload.Table{
		Game:      g,
		Units:     f.Collection(databin.Units),
		Humans:    f.Collection(databin.Humans),
		Buildings: f.Collection(databin.Buildings),
		NPC:       npc,
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
	}}
	d.StartWeapon, d.StartWeaponErr = resolveStartingWeapon(f, PartySkillSlot())
	// Not fatal, and reported by nothing beyond the field itself — see
	// Definitions.Bodies.
	d.Bodies, _ = ReadBodyList(fsys)
	// NOT FATAL, for the same reason: a missing item-name table narrows what
	// itemName (world.go) can show a code by, it does not stop a mission
	// from opening. It reads a different archive (main.res text, not
	// world.res), so it rides beside the table rather than inside the walk
	// above.
	d.Table.Names, _ = ReadItemNames(fsys)
	// NOT FATAL either: a table that read fewer than the four hero names carries
	// none, and MissionPartyAs then names a hero started without the generator's
	// name field itself. The generator reads the same four through the same
	// reader and refuses to open without them.
	if names, err := heroPictureNames(fsys); err == nil {
		d.Table.HeroNames = names
	}
	// Not fatal: a table without them names no map placement for the hero
	// ordinal scan, which then reaches the party alone.
	if b, err := fsys.ReadFile(chargenNamesPath); err == nil {
		rows := SplitTextTable(b)
		d.Table.NPCNames = rows.lines
	}
	return d, nil
}

// LoadNPCDefs reads the scenario NPC table out of the container filesystem.
//
// It is a function of its own beside LoadTable, rather than inline in it,
// because the two files live in two different archives and a developer tool
// pointed at one of them must be able to load the other's half without being
// handed a whole install.
func LoadNPCDefs(fsys *vfs.FS) (*data.NPCDefs, error) {
	b, err := fsys.ReadFile(NPCRegistry)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", NPCRegistry, err)
	}
	r, err := reg.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", NPCRegistry, err)
	}
	return data.LoadNPCDefs(r), nil
}

// LoadNPCFaces reads the same registry LoadNPCDefs does, for the three keys a
// DIALOGUE PORTRAIT needs rather than the one a placement does (0141;
// `REG-NPC-058`, whose gloss on those keys is Unknown — data/npcface.go carries
// the reading and the measurement behind it).
//
// IT PARSES THE FILE A SECOND TIME rather than widening the one above. The two
// answers have different lifetimes and different consumers — one is on the
// placement path of every map, the other is read only when a mission's script
// raises a dialogue — and npc.go's own doc refuses to hold "keys that belong to
// screens" for exactly that reason. The file is 105 sections and is read twice
// per process, not per map.
//
// A REGISTRY THAT WILL NOT OPEN IS NO FACES AND NO ERROR. The placement path
// above already fails loudly on the same file, so a second failure here could
// only be the same one reported twice — and a mission whose speakers have no
// pictures still runs, which is what the seam this feeds has always promised.
func LoadNPCFaces(fsys *vfs.FS) map[int32]data.NPCFace {
	b, err := fsys.ReadFile(NPCRegistry)
	if err != nil {
		return nil
	}
	r, err := reg.Parse(b)
	if err != nil {
		return nil
	}
	return data.LoadNPCFaces(r)
}

func databinLayout(g base.Game) databin.Layout {
	if g.Edition().SecondTable {
		return databin.ROM2Layout
	}
	return databin.ROM1Layout
}
