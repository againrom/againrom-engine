package ui

// Words is the set of program-chosen words this build resolves from the
// install.
//
// A FIELD PER SCALAR WORD OR ONE FINITE ARRAY PER DECODED TABLE. AuthoredWords
// fills the production positions with English fallback strings, and resolution
// overwrites fields the install states a line for. An unresolved word is a
// visible default rather than a missing key that draws as nothing.
//
// Installed fields have a decoded index or finite table. The few owner-authored
// controls and units are named explicitly; this is not a general translation
// table for arbitrary prose or unidentified callback targets.
type Words struct {
	// TEXT-HOVERCHAR-050 / TEXT-HOVERROOM-051 / TEXT-HOVERTEXT-052:
	// finite main.txt table, indexed by the confirmed hover consumers.
	// Missing installed entries stay empty and suppress that tooltip.
	Hover     [274]string
	SiteHints [19]string // TEXT-HOVERTEXT-052, sites.txt local marker indices.
	// The dialogue and legacy single-button panel control, global slot 77
	// (`MENU-STRTAB-008`, L06181). Success and failure use their own
	// decoded two-control labels below.
	NoticeButton string

	// The mission-outcome panels, global slots 140 and 141
	// (`MENU-STRTAB-008`, L06182 and L06183).
	MissionWon  string
	MissionLost string

	// The `Pause` key's own modal text, global slot 119 (`AI-KEY-125`, and its
	// evidence `keyboard.tsv` row 20: "Pause | map | campaign phase 2 | show
	// modal text main.txt[119]"). The index is the claim's, which is this
	// type's membership rule exactly.
	PauseNotice string

	// NoHeroSelected is global slot 51: the line the original centres on an
	// empty spellbook strip while no hero is selected (DIV-2264). It has no
	// authored fallback: an install without the string draws the strip empty.
	NoHeroSelected string

	// HelpText is `text/help.txt` whole, as shipped (TEXT-086), shown by
	// the F1 panel. It has no authored fallback: an install without the file
	// opens no panel.
	HelpText string

	// The floating tip panel's two captioned controls, global main.txt slots
	// 127 and 128 (`TOWN-206`). They belong to the same install-resolved word
	// set as every other program-chosen caption; the tip body remains the
	// separate text/tips payload selected by its screen.
	TipClose    string
	TipShowNext string

	MenuSave            string
	MenuLoad            string
	MenuDiplomacy       string
	MenuGameOptions     string
	MenuSoundOptions    string
	MenuQuestObjectives string
	QuestHeading        string
	MenuEndQuest        string
	MenuReturn          string
	MenuAbort           string
	MenuChangeMap       string
	MenuVictory         string
	// OutcomeContinue is dialogs.txt's local string 154, the second control on
	// the campaign-success panel (MISSION-VICTORY-029).
	OutcomeContinue string
	MenuExitMain    string
	MenuExitWindows string

	// The shop's four command-button captions, global slots 72, 70, 71 and 73
	// (`SHOP-050`, `0x48`/`0x46`/`0x47`/`0x49`), in the panel's own button
	// order (SHOP-SCREEN-035): clear-table, buy, sell, leave.
	ShopUndo string
	ShopBuy  string
	ShopSell string
	ShopExit string

	// The school's two caption sources, main.txt global slots 231 and 232
	// (TOWN-383). Prices are separate per-paint values, not these strings.
	SchoolTrain string
	SchoolExit  string

	// TOWN-391/392: active tavern captions use main.txt slots 258/259
	// for hire/fire, 242 for talk and 232 for exit. Activation replaces the
	// constructor's combined slot 243 before painting.
	TavernHire  string
	TavernHired string
	TavernFire  string
	TavernTalk  string
	TavernExit  string
	TavernSleep string

	// The command panel's own eight cell labels, main.txt global slots 0-7 in
	// the panel's own cell order — Attack, Move, Guard, Defend, Cast, Swarm,
	// Stand Ground, Retreat (`MENU-COMBAT-019`; docs/1028-command-panel
	// contract B1). AN ARRAY AND NOT EIGHT NAMED FIELDS, unlike every other
	// member of this struct: the eight are one homogeneous vocabulary
	// indexed identically to the panel's own cell constants
	// (commandpanel.go's commandCellAttack..commandCellRetreat), so the
	// array index IS the decoded index and no second mapping between a name
	// and a slot number is needed. It is still one statically-known field
	// per decoded index, which is the membership rule above; it is grouped
	// because the eight are read together, by cell, rather than by name.
	Command [8]string

	// The original unit sheet reads its class name from unitname.txt and its
	// fixed captions from main.txt. Arrays preserve those two decoded index
	// spaces without making an index in one table usable in the other.
	UnitNames [81]string
	// BuildingNames is building.txt indexed by structure class ID (line ID-1);
	// an empty entry leaves the registry's own name text.
	BuildingNames [67]string
	PanelCaptions [192]string

	// ItemStats is stats.txt's local index space. The production formatter
	// reads only the positions named by an item effect, but retaining the
	// finite table here keeps every one of its three accessor sites on this
	// same install-word value.
	ItemStats [50]string
	ItemMagic string
	// ItemCasts, ItemDamage, ItemRange and ItemRays are the weapon-borne spell
	// lines of the item description; the install states them for its own language.
	ItemCasts  string
	ItemDamage string
	ItemRange  string
	ItemRays   string
	// ItemSpellOf and ItemSpellOfSuffix are the two words the item formatter puts
	// round a spell name on the name line of a book (main.txt lines 90 and 91).
	ItemSpellOf       string
	ItemSpellOfSuffix string
	ItemSpellNames    [29]string // spell.txt, indexed by spell ID (1..28)
	// DurationUnit is authored UI copy for nominal game seconds. The install
	// names the duration caption, while its language selector chooses this unit.
	DurationUnit string

	// SpellBookNames maps spells.txt book cells to spell IDs for the shipped catalog.
	// A row joins name and description with '#'; only the name is kept.
	SpellBookNames [29]string

	// MapListSize is the map-selection list's size-column caption,
	// dialogs.txt local slot134, and MapListColumns the captions of the two
	// record-word columns, slots135 and 136 (TEXT-083).
	MapListSize    string
	MapListColumns [2]string

	SaveAcknowledgement string
	// SaveDialog is authored player copy for the explicit save choices. Unlike
	// installed table fields, its strings are UTF-8 and encoded when drawn.
	SaveDialog      SaveDialogWords
	WorldHomeTitle  string
	WorldHomeDetail string
	WorldPayment    string
	SelectionStatus [4]string

	// SkillRaised is main.txt global slots 130..139: the weapon-skill raise
	// lines Blade to Shooting, then the spell-school lines Fire to Astral, each
	// run in skill-slot order 1..5. No claim names their consumer (DIV-1449).
	SkillRaised [10]string

	// PickedUp, PickedUpNow and PickedUpPieces are main.txt global slots 85,
	// 86 and 87: the pickup line states 85 and an Item's name, then in
	// parentheses 86, the Item's count and 87 when that count is above one.
	// PickedUpGold and PickedUpGoldUnit are slots 88 and 89, around a purse
	// gain in the gold line (ITEM-PICKTEXT-145).
	PickedUp         string
	PickedUpNow      string
	PickedUpPieces   string
	PickedUpGold     string
	PickedUpGoldUnit string

	// SettingNotice is main.txt global slots 94..116 and 218..220 at their own
	// subscripts: the line a settings key posts for its new state (retreat
	// 94..96, formation 97..99, show health 100..101, flying damage 102..103,
	// day/night 104..105, smoothing 106..107, autohealing 218..220) and the
	// line the speed step posts (108..116). Every other entry is empty, and an
	// empty entry posts nothing (MENU-057, MENU-058).
	SettingNotice [221]string
}

// authoredSkillRaised is the EN root's own main.txt[130..139], the authored
// fallback on the shop words' rule in AuthoredWords.
func authoredSkillRaised() [10]string {
	return [10]string{
		"Blade skill improved", "Axe skill improved", "Bludgeon skill improved",
		"Pike skill improved", "Shooting skill improved",
		"Fire skill improved", "Water skill improved", "Air skill improved",
		"Earth skill improved", "Astral skill improved",
	}
}

// SkillRaisedLine is the raise line for skill slot 1..5: the weapon skill's
// for a fighter, the spell school's for a mage. Slot 0 and a slot outside
// 1..5 have no line. An empty word is no word and takes the authored line.
func (w *Words) SkillRaisedLine(mage bool, slot int) (string, bool) {
	if slot < 1 || slot > 5 {
		return "", false
	}
	i := slot - 1
	if mage {
		i += 5
	}
	if w.SkillRaised[i] != "" {
		return w.SkillRaised[i], true
	}
	return authoredSkillRaised()[i], true
}

// The authored English words are the fallback for an install that does not
// state a line and the whole word set for a build with no install behind it
// — the standalone developer viewer, and every test that assembles a
// viewer by hand.
const (
	// AuthoredNoticeButton is the dialogue and outcome notices' button word.
	AuthoredNoticeButton = "OK"

	// MissionWonText and MissionLostText are the two endings. They moved here
	// from pkg/game with 0168 so that the authored words sit together;
	// pkg/game keeps exported aliases, so the names a caller already used still
	// name these values.
	MissionWonText  = "MISSION COMPLETE"
	MissionLostText = "MISSION FAILED"

	// AuthoredPauseNotice is the EN root's own resolved `main.txt[119]`,
	// legitimate as the authored fallback exactly like the four shop words and
	// the eight command labels below. It names the notice window's own button
	// word, which is why the `Pause` key reaches that window and not a second
	// display path.
	AuthoredPauseNotice = "Game paused. Click OK to continue."

	// The EN install's exact tip-control captions. They are visible fallbacks
	// for a standalone viewer or an incomplete install, on every other scalar
	// word's rule in AuthoredWords.
	AuthoredTipClose    = "Close"
	AuthoredTipShowNext = "Show tips next time"
)

// AuthoredWords is this build's own English word set.
//
// It is a FUNCTION and not a package variable, on AuthoredPanelLayout's own
// rule: a variable is writable from anywhere, and "two viewers show the same
// words" would then be true by mutation rather than by construction.
func AuthoredWords() Words {
	out := Words{
		NoticeButton: AuthoredNoticeButton,
		MissionWon:   MissionWonText,
		MissionLost:  MissionLostText,
		ItemMagic:    "MAGIC:",
		ItemCasts:    "Casts",
		ItemDamage:   "Damage",
		ItemRange:    "Range",
		ItemRays:     "Rays",
		ItemSpellOf:  "of",
		DurationUnit: "sec",
		PauseNotice:  AuthoredPauseNotice,
		TipClose:     AuthoredTipClose,
		TipShowNext:  AuthoredTipShowNext,

		// UPPER CASE, AND THE INSTALL'S ARE NOT. The shipped
		// English labels read `~Save Game`; these are this build's own since
		// 0158. Case-folding either way would need the CP866 fold 0168 leaves
		// out, so the two differ visibly and the difference is disclosed.
		MenuSave:            "~SAVE GAME",
		MenuLoad:            "~LOAD GAME",
		MenuDiplomacy:       "DIPLOMACY",
		MenuGameOptions:     "GAME ~OPTIONS",
		MenuSoundOptions:    "SOU~ND OPTIONS",
		MenuQuestObjectives: "~QUEST OBJECTIVES",
		QuestHeading:        "Quest Objectives and Hints:",
		MenuEndQuest:        "~END QUEST",
		MenuReturn:          "~RETURN TO GAME",
		MenuAbort:           "ABORT GAME",
		MenuChangeMap:       "CHANGE MAP",
		MenuVictory:         "~VICTORY!",
		OutcomeContinue:     "CONTINUE",
		MenuExitMain:        "~EXIT TO MAIN MENU",
		MenuExitWindows:     "EXIT TO ~WINDOWS",

		// The EN root's own resolved SHOP-050 strings, so an install with no
		// main.txt (or the standalone viewer) still shows the shipped words.
		ShopUndo:    "Undo",
		ShopBuy:     "Buy",
		ShopSell:    "Sell",
		ShopExit:    "Exit",
		SchoolTrain: "Train",
		SchoolExit:  "EXIT",
		TavernHire:  "Hire",
		TavernHired: "HIRED",
		TavernFire:  "Fire",
		TavernTalk:  "Talk",
		TavernExit:  "EXIT",
		TavernSleep: "Sleep",

		// The EN root's own resolved main.txt lines 0-7 (docs/1028-command-
		// panel, measured-at-seat premises), legitimate as the authored
		// fallback exactly like the four shop words above.
		Command: [8]string{
			"Attack", "Move", "Guard", "Defend",
			"Cast", "Swarm", "Stand Ground", "Retreat",
		},

		SaveAcknowledgement: "Your character is saved",
		SaveDialog:          authoredSaveDialogWords(false),
		WorldHomeTitle:      "TOWN",
		WorldHomeDetail:     "Return to town",
		WorldPayment:        "Reward",
		SelectionStatus:     [4]string{"No units", "selected", "Units", "selected:"},
		MapListSize:         "Size of map",
		SkillRaised:         authoredSkillRaised(),

		// The EN root's own main.txt[85..89], on the shop words' rule above.
		PickedUp:         "Picked up",
		PickedUpNow:      "now",
		PickedUpPieces:   "pieces",
		PickedUpGold:     "Picked up",
		PickedUpGoldUnit: "gold",
	}
	for i, s := range map[int]string{
		15: "BODY", 16: "AGILITY", 17: "MIND", 18: "SPIRIT",
		19: "HEALTH", 20: "MANA", 21: "SIGHT", 22: "SPEED",
		23: "DMG", 24: "ABSORB", 25: "ATTACK", 26: "DEFENSE",
		27: "SKILLS", 28: "RESISTANCE", 30: "BLADE", 31: "AXE",
		32: "BLUDGEON", 33: "PIKE", 34: "SHOOTING", 35: "WEIGHT",
		36: "FIRE", 37: "WATER", 38: "AIR", 39: "EARTH", 40: "ASTRAL",
		41: "FIRE", 42: "WATER", 43: "AIR", 44: "EARTH", 45: "ASTRAL",
		46: "XP", 84: "MERCENARY", 190: "SPELLCASTER", 191: "ARMOR PIERCING",
	} {
		out.PanelCaptions[i] = s
	}
	for i, s := range map[int]string{
		1: "Value", 2: "Body", 3: "Mind", 4: "Reaction", 5: "Spirit",
		6: "Health", 7: "Maximum health", 8: "Health regeneration",
		9: "Mana", 10: "Maximum mana", 11: "Mana regeneration",
		12: "To-hit", 13: "Minimum damage", 14: "Maximum damage",
		15: "Defence", 16: "Absorption", 17: "Speed", 18: "Rotation speed",
		19: "Sight", 20: "Protection", 21: "Fire protection", 22: "Water protection",
		23: "Air protection", 24: "Earth protection", 25: "Astral protection",
		26: "Fighter skill", 27: "Blade skill", 28: "Axe skill",
		29: "Bludgeon skill", 30: "Pike skill", 31: "Shooting skill",
		32: "Mage skill", 33: "Fire skill", 34: "Water skill", 35: "Air skill",
		36: "Earth skill", 37: "Astral skill", 38: "Item lore", 39: "Magic lore",
		40: "Creature lore", 41: "Cast", 42: "Teach", 43: "Damage",
		44: "Fire damage", 45: "Water damage", 46: "Air damage",
		47: "Earth damage", 48: "Astral damage", 49: "Damage bonus",
	} {
		out.ItemStats[i] = s
	}
	return out
}

// OutcomeText is the sentence the outcome notice states for an ending. It
// takes a bool rather than a sim.Outcome because pkg/ui does not know the
// simulation's types and must not learn them for a word.
func (w Words) OutcomeText(lost bool) string {
	if lost {
		return w.MissionLost
	}
	return w.MissionWon
}

// OnLayout is l with the resolved button word on it.
//
// IT IS A VALUE AND NOT A MUTATION, on WithPortrait's own rule: a caller holds
// an authored layout and gets back the one this word set makes of it, and the
// layout it passed is unchanged. The town screen composes its dialogue layout
// itself, off AuthoredDialogueLayout, so this is how the word reaches a notice
// the viewer never held.
//
// AN EMPTY WORD IS NO WORD AND LEAVES THE LAYOUT'S OWN. The zero Words is what a
// front end assembled by hand in a test carries, and blanking a button caption
// because nobody filled a struct in would be a defect that draws as an empty
// button rather than as an error.
func (w Words) OnLayout(l NoticeLayout) NoticeLayout {
	if w.NoticeButton == "" {
		return l
	}
	l.ButtonLabel = w.NoticeButton
	return l
}

// SetWords replaces the viewer's word set and each notice control word with
// it (MISSION-VICTORY-029).
//
// IT REWRITES ONLY CONTROL WORDS. Everything else about the layouts — geometry,
// colours and pitch — is untouched, so a caller that already
// replaced them through SetNoticeLayouts keeps its own, and this stays the one
// writer of a word rather than a second writer of an appearance.
func (v *Viewer) SetWords(w Words) {
	v.words = w
	v.noticeLayouts[0] = w.OnLayout(v.noticeLayouts[0])
	v.noticeLayouts[1] = w.OnLayout(v.noticeLayouts[1])
	if w.MenuVictory != "" {
		v.noticeLayouts[NoticeSuccess].ButtonLabel = w.MenuVictory
	}
	if w.OutcomeContinue != "" {
		v.noticeLayouts[NoticeSuccess].SecondaryButtonLabel = w.OutcomeContinue
	}
	// MISSION-DEFEAT-046: dialogs[44] and dialogs[35], already used by
	// the in-game menu. No separate translated or authored failure labels.
	if w.MenuExitMain != "" {
		v.noticeLayouts[NoticeFailure].ButtonLabel = w.MenuExitMain
	}
	if w.MenuLoad != "" {
		v.noticeLayouts[NoticeFailure].SecondaryButtonLabel = w.MenuLoad
	}
}

// Words is the viewer's resolved word set. The tier that owns the world asks
// for it rather than holding a second copy, so the sentence a notice states and
// the word its button states cannot come from two different installs.
func (v *Viewer) Words() Words { return v.words }
