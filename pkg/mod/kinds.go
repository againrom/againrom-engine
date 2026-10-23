package mod

import (
	"fmt"
	"slices"
	"strings"
)

// Every kind of data a mod carries is declared once, in dataKinds: its file
// names, its record key, how a file's records are read and validated, and
// how the records of several mods combine. The script's game.data.add, the
// mod check, the game's one apply step, the base dump and docs/MODS.md all
// walk this list; a kind that is not declared here is not read.

// DataClass is whether a kind of mod data is a catalog of game data the base
// fills, or configuration the mod brings with no base catalog.
type DataClass uint8

const (
	// CatalogData is game data the base holds as layer 0; each mod's
	// records are a layer over it by record key.
	CatalogData DataClass = iota + 1
	// ConfigurationData has no base catalog: the mod's records are its
	// own, and DataKind.Base says why.
	ConfigurationData
)

func (c DataClass) String() string {
	if c == CatalogData {
		return "catalog"
	}
	return "configuration"
}

// Layering is how the records of the mods of a set combine.
type Layering uint8

const (
	// LayerMerged records are patches merged by Merge over layer 0: two
	// mods set one field only when load-after orders them, the later wins
	// and the check names the override.
	LayerMerged Layering = iota + 1
	// LayerKept records keep the combination rules the kind had before it
	// was declared here; Owner names the story that migrates the kind.
	LayerKept
	// LayerOwn records are each mod's own; no two mods name one record.
	LayerOwn
)

func (l Layering) String() string {
	switch l {
	case LayerMerged:
		return "merged"
	case LayerKept:
		return "kept"
	}
	return "own"
}

// DataKind declares one kind of mod data.
type DataKind struct {
	// Name names the kind in docs/MODS.md and in the base dump.
	Name string
	// Files are the names the kind's file is read under, relative to the
	// mod; the first is the name to use. A mod holds at most one of them.
	Files []string
	// Key is the field or fields that name one record.
	Key   string
	Class DataClass
	Layer Layering
	// Base names layer 0 of a catalog kind, or says why a configuration
	// kind has no base catalog.
	Base string
	// Shared is the rule for two mods that name one record.
	Shared string
	// Owner is the story that migrates a kept kind onto Merge.
	Owner string
	// Dump reports that the base dump writes the kind's layer 0.
	Dump bool
	// Script reports that the entry script loads the file with
	// game.data.add; the file of any other kind is read beside the
	// manifest of every mod that holds it.
	Script bool
	// read parses one file of the kind from one mod, refusing a value at
	// its file and line, and adds the records to d.
	read func(r DataReader, d *Data, rel string, src []byte) error
	// summary is the check's line on the kind's records; "" when none.
	summary func(d Data) string
}

// mergedShared is the rule a merged kind's records follow.
const mergedShared = "yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused"

var dataKinds = []DataKind{
	{
		Name: "items", Files: []string{ItemsFile}, Key: "`key` (an added item), `item` (a change of a shipped row)",
		Class: CatalogData, Layer: LayerKept, Owner: "T9",
		Base:   "the item rows of the definition table; added items take free code rows",
		Shared: "added items are each mod's own; two changes of one shipped row both apply, the later's values last, unnamed",
		Script: true, read: readItems,
		summary: func(d Data) string {
			if d.Items.Empty() {
				return ""
			}
			return fmt.Sprintf("items added=%d changed=%d", len(d.Items.Rows), len(d.Items.Changes))
		},
	},
	{
		Name: "characters", Files: []string{CharactersFile}, Key: "`key` (a character of the catalog); `target` (an edit of a definition row or a family of tier rows)",
		Class: CatalogData, Layer: LayerMerged,
		Base:   "the install's npc.reg records, read once, keyed `npc<n>`; `target` edits existing definition rows",
		Shared: "catalog keys: " + mergedShared + "; `off` switches a mod's character off; target edits: all operations apply in load order, including overlapping families and rows, without requiring load-after",
		Script: true, read: readCharacters,
		summary: func(d Data) string {
			if d.Characters.Empty() {
				return ""
			}
			edited, added := 0, 0
			for _, r := range d.Characters.Rows {
				if r.Target != "" {
					edited++
				} else {
					added++
				}
			}
			if added == 0 {
				return fmt.Sprintf("characters edited=%d", edited)
			}
			return fmt.Sprintf("characters edited=%d added=%d", edited, added)
		},
	},
	{
		Name: "screens", Files: []string{ScreensFile}, Key: "`key`, within the mod",
		Class: ConfigurationData, Layer: LayerOwn,
		Base:   "a mod's information screens and menu actions are additions to the menus; the base has no screen of this kind to change",
		Shared: "no: a screen is its mod's own; one action is added by one mod",
		Script: true, read: readScreens,
		summary: func(d Data) string {
			if d.Screens.Empty() {
				return ""
			}
			main, game := d.Screens.Count()
			return fmt.Sprintf("screens=%d main-menu=%d game-menu=%d", len(d.Screens.Screens), main, game)
		},
	},
	{
		Name: "companions", Files: []string{CompanionsFile}, Key: "`chapter` and `companion`",
		Class: CatalogData, Layer: LayerMerged,
		Base:   "empty: the base grants every town companion on arrival, with no join condition",
		Shared: mergedShared + "; `off` switches a condition off",
		Script: true, read: readCompanions,
		summary: func(d Data) string {
			if d.Companions.Empty() {
				return ""
			}
			return fmt.Sprintf("companion joins=%d", len(d.Companions.Joins))
		},
	},
	{
		Name: "spells", Files: []string{SpellsFile}, Key: "`target` and the field; `[global]` keys",
		Class: CatalogData, Layer: LayerKept, Owner: "an unscheduled spell catalog story",
		Base:   "the spell rows, names, rules and formulas of several derived reads of the definition table: no one list",
		Shared: "no: one field of one spell, and one global key, is set by one mod",
		Script: true, read: readSpells,
		summary: func(d Data) string {
			if d.Spells.Empty() {
				return ""
			}
			return fmt.Sprintf("spells edited=%d global=%d", len(d.Spells.Rows), len(d.Spells.Global.Globals()))
		},
	},
	{
		Name: "body sheets", Files: []string{BodiesFile}, Key: "`name`",
		Class: CatalogData, Layer: LayerMerged,
		Base:   "empty: a shipped body is a registry class and its sheets, not a body sheet; a sheet may not take a shipped name",
		Shared: mergedShared + "; a later sheet replaces the whole sheet; `off` switches a sheet off",
		Script: true, read: readBodies,
	},
	{
		Name: "weapon bodies", Files: WeaponBodiesFiles, Key: "the weapon row, by `weapon`, `row` or both",
		Class: CatalogData, Layer: LayerMerged,
		Base:   "the shipped body list: the body of each weapon row",
		Shared: mergedShared + "; a weapon row is never switched off",
		Script: true, read: readWeaponBodies,
		summary: func(d Data) string {
			if d.Bodies.Empty() {
				return ""
			}
			return fmt.Sprintf("weapon bodies=%d body sheets=%d", len(d.Bodies.Weapons), len(d.Bodies.Bodies))
		},
	},
	{
		Name: "missions", Files: []string{MissionsFile}, Key: "`number`; `key` within the mod",
		Class: CatalogData, Layer: LayerMerged,
		Base:   "empty: the campaign's missions are the campaign decoder's; a mod mission adds a guest mission or replaces a campaign mission's map, by number",
		Shared: mergedShared + "; a later mission replaces the whole mission; `off` switches a mission off; one guest mission plays after one win",
		Script: true, read: readMissions,
	},
	{
		Name: "towns", Files: []string{TownsFile}, Key: "`id`",
		Class: CatalogData, Layer: LayerMerged, Dump: true,
		Base:   "the install's town catalog",
		Shared: mergedShared + "; `off` switches a town off",
		Script: true, read: readTowns,
		summary: func(d Data) string {
			if d.Towns.Empty() {
				return ""
			}
			return fmt.Sprintf("town records=%d", len(d.Towns.Records))
		},
	},
	{
		Name: "settings", Files: []string{SettingsName}, Key: "the table name",
		Class: ConfigurationData, Layer: LayerOwn,
		Base:   "a setting is a knob of its mod, chosen per launch and kept in the save's mod mark; the base has none",
		Shared: "no: a setting is its mod's own",
		read:   readSettings,
	},
}

// DataKinds lists the declared kinds of mod data, in the order the game
// applies them.
func DataKinds() []DataKind { return slices.Clone(dataKinds) }

// ScriptDataKind is the kind whose file game.data.add reads under rel.
func ScriptDataKind(rel string) (DataKind, bool) {
	for _, k := range dataKinds {
		if k.Script && slices.Contains(k.Files, rel) {
			return k, true
		}
	}
	return DataKind{}, false
}

// ScriptDataFiles lists every name game.data.add reads, sorted and joined.
func ScriptDataFiles() string {
	var names []string
	for _, k := range dataKinds {
		if k.Script {
			names = append(names, k.Files...)
		}
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

// Read parses one file of the kind from the mod r reads, refusing a value at
// its file and line, and adds its records to d.
func (k DataKind) Read(r DataReader, d *Data, rel string, src []byte) error {
	return k.read(r, d, rel, src)
}

// Validates reports that the kind declares how its files are read.
func (k DataKind) Validates() bool { return k.read != nil }

// Summary is the check's line on the kind's records in d; "" when d holds
// none or the kind reports none.
func (k DataKind) Summary(d Data) string {
	if k.summary == nil {
		return ""
	}
	return k.summary(d)
}

// checkKindNames refuses a mod that holds one kind's file under two of its
// names, naming both.
func checkKindNames(t Tree) error {
	for _, k := range dataKinds {
		var held []string
		for _, n := range k.Files {
			if st, err := t.Stat(n); err == nil && !st.IsDir() {
				held = append(held, n)
			}
		}
		if len(held) > 1 {
			return fmt.Errorf("the mod holds both %s and %s; keep one of them", held[0], held[1])
		}
	}
	return nil
}

// DataReader is what a kind's reader knows of the mod whose file it reads.
type DataReader struct {
	// Mod is the mod's id and Dir its folder or RES file.
	Mod string
	Dir string
	// Lang is the language the mod's text is read in.
	Lang string
	// Settings are the mod's resolved settings.
	Settings []SettingValue
}

// Data is what the mods of a set declare, each kind in load order. A merged
// kind holds the mods' layer records; the game merges them over layer 0.
type Data struct {
	Items      ItemData
	Characters CharacterData
	Screens    ScreenData
	Companions CompanionData
	Spells     SpellData
	// Bodies are the body sheets and weapon bodies.
	Bodies   BodyData
	Missions MissionData
	Towns    TownData

	// spellGlobals and spellFields name the mod that set each spell global
	// and each field of a spell.
	spellGlobals map[string]string
	spellFields  map[string]string
}

// SetOrder gives each merged kind that holds records the load order the
// mods state.
func (d *Data) SetOrder(order Precedence) {
	if !d.Companions.Empty() {
		d.Companions.Order = order
	}
	if !d.Bodies.Empty() {
		d.Bodies.Order = order
	}
	if !d.Towns.Empty() {
		d.Towns.Order = order
	}
	if !d.Characters.Empty() {
		d.Characters.Order = order
	}
	if !d.Missions.Empty() {
		d.Missions.Order = order
	}
}
