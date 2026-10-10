package game

import (
	"againrom/pkg/base"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/locale"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
	"againrom/pkg/words"
	"strings"
)

// The two install text tables this build reads by index, and the decoded
// line numbers it reads out of them.
//
// The addresses are the loader's own (`TEXT-STRTAB-023`): fifteen of the
// sixteen tables come from `main\text\`, and `main.txt` is the first call,
// which is why its line numbers are also the global subscripts.
const (
	MainTextPath       = mainPrefix + "text/main.txt"
	DialogsTextPath    = mainPrefix + "text/dialogs.txt"
	StatsTextPath      = mainPrefix + "text/stats.txt"
	UnitNameTextPath   = mainPrefix + "text/unitname.txt"
	SitesTextPath      = mainPrefix + "text/sites.txt"
	BuildingTextPath   = mainPrefix + "text/building.txt"
	SpellNamesTextPath = mainPrefix + "text/spell.txt"
	// SpellBookNamesTextPath is the mission/shop spellbook popup's own name
	// source (TEXT-HOVERTEXT-052, getter R1207) — spells.txt, plural, a
	// different file from SpellNamesTextPath above.
	SpellBookNamesTextPath = mainPrefix + "text/spells.txt"
	// HelpTextPath is the F1 help panel's text (TEXT-086), read whole and
	// never split into lines.
	HelpTextPath = mainPrefix + "text/help.txt"

	mainSlotNoticeButton = 77  // the dialogue panel's button, L06181
	mainSlotMissionWon   = 140 // the mission-outcome panels, L06182
	mainSlotMissionLost  = 141 // and L06183

	// The `Pause` key's own modal text. The index is `AI-KEY-125`'s, off its
	// evidence `keyboard.tsv` row 20 ("show modal text main.txt[119]"), and
	// not `MENU-STRTAB-008`'s: no instruction address is read for it, the
	// key handler's own row states the subscript.
	mainSlotPauseNotice = 119
	// The spellbook strip's line while no hero is selected, identified by the
	// owner's screenshot of the original and present at this global slot in
	// both installed main.txt tables.
	mainSlotNoHeroSelected = 51
	// The role line above a selected tavern candidate's class name. The owner
	// screenshot identifies the consumer and both installed main.txt tables
	// place the localized word at global slot 84 (DIV-426).
	mainSlotMercenary           = 84
	mainSlotTipClose            = 127
	mainSlotTipShowNext         = 128
	mainSlotSaveAcknowledgement = 203
	mainSlotWorldReturn         = 261
	mainSlotWorldPayment        = 262

	mainSlotShopUndo = 0x48 // 72
	mainSlotShopBuy  = 0x46 // 70
	mainSlotShopSell = 0x47 // 71
	mainSlotShopExit = 0x49 // 73

	// TOWN-383: both school constructors copy these global main.txt slots.
	mainSlotSchoolTrain = 231
	mainSlotSchoolExit  = 232

	// The first of the ten skill-raise lines, ui.Words.SkillRaised (DIV-1449).
	mainSlotSkillRaised = 130

	// ITEM-PICKTEXT-145: the pickup line's opening words, the two around a
	// count, and the two around a purse gain.
	mainSlotPickedUp         = 85
	mainSlotPickedUpNow      = 86
	mainSlotPickedUpPieces   = 87
	mainSlotPickedUpGold     = 88
	mainSlotPickedUpGoldUnit = 89

	// TOWN-391/392: selection replaces the first caption before activation.
	mainSlotTavernHire = 258
	mainSlotTavernFire = 259
	mainSlotTavernTalk = 242
	mainSlotTavernExit = 232

	dialogsSlotSave            = 0x22
	dialogsSlotLoad            = 0x23
	dialogsSlotDiplomacy       = 0x4c
	dialogsSlotGameOptions     = 0x24
	dialogsSlotSoundOptions    = 0x25
	dialogsSlotQuestObjectives = 0x26
	dialogsSlotQuestHeading    = 68
	dialogsSlotEndQuest        = 0x27
	dialogsSlotReturn          = 0x28
	dialogsSlotChangeMap       = 0x2a
	dialogsSlotVictory         = 0x2b
	dialogsSlotContinue        = 154
	dialogsSlotExitMain        = 0x2c
	dialogsSlotExitWindows     = 0x2d
	dialogsSlotAbort           = 0x4d

	// The map-selection list's column captions (TEXT-083).
	dialogsSlotMapListSize    = 134
	dialogsSlotMapListColumn0 = 135
	dialogsSlotMapListColumn1 = 136
)

// TextTable is one install text file split into lines by the decoded
// loader's own walk.
//
// THE WALK IS THE DECODE AND `bytes.Split` IS NOT. On a well-formed `CRLF`
// file that is the same answer splitting gives; on `A\rXB` it is not, and
// the walk is what the game does.
//
// The first game's bytes are kept exactly as shipped. No code-page pass runs
// at load in the original and none runs here: a Russian install's CP866 bytes
// are converted at DRAW, by the font's selector (`TEXT-CHARGEN-029`,
// `TEXT-DOM-010`), which is where this tree already applies it. A game whose
// loaders convert their text reaches the table through TextCode first.
type TextTable struct {
	lines []string
}

// SplitTextTable is the loader's walk over one payload.
//
// A trailing run with no `CR` is not a line. The decoded scan has no end check
// and runs off the buffer there, which is undefined rather than authored, so
// this reader stops instead of inventing a last line.
func SplitTextTable(b []byte) *TextTable {
	t := &TextTable{}
	for i := 0; i < len(b); {
		j := i
		for j < len(b) && b[j] != '\r' {
			j++
		}
		if j == len(b) {
			break
		}
		t.lines = append(t.lines, string(b[i:j]))
		i = j + 2 // the CR and the byte after it, whatever it is
	}
	return t
}

// EncodeInstallText turns a decoded UTF-8 string back into the install's own
// single-byte alphabet, rune by rune, under selector — the same conversion
// LanguageSelector's own digit drives for every other install-resolved word.
//
// It exists for text this tree decodes to UTF-8 for a reason of its own
// (alm.Info.Description, round-tripped by a map tool) but which still has to
// share a line with the game's own byte-indexed font: the font's Draw walks
// bytes, not runes, so a string in the wrong alphabet paints the wrong
// glyphs rather than failing to compile or to run. A rune this selector
// cannot encode is dropped rather than widening the string with a
// placeholder byte a real installed line would never contain.
func EncodeInstallText(s string, selector int) string {
	var b []byte
	for _, r := range s {
		if c, ok := textinput.EncodeRune(r, selector); ok {
			b = append(b, c)
		}
	}
	return string(b)
}

// spellBookRowName is spells.txt's own two-field row, name and description
// joined by '#' (TEXT-HOVERTEXT-052), cut back to the name the popup shows.
// A row with no '#' names itself whole rather than being read as absent.
func spellBookRowName(s string) string {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i]
	}
	return s
}

// TextCode is how an install's text files reach its font: the code page the
// install writes them in and the one the font draws. The zero TextCode keeps
// every byte.
type TextCode struct {
	From, To int
}

// InstallTextCode is the text code of src's install under edition: its
// language's text code page there, and the language's font code page.
func InstallTextCode(src terrain.EntrySource, edition base.Edition) TextCode {
	l, _ := locale.BySelector(LanguageSelector(src))
	c := TextCode{From: l.CodePage, To: l.CodePage}
	if edition.TextCodePage != nil {
		c.From = edition.TextCodePage(l.CodePage, l.WindowsCodePage)
	}
	return c
}

// Bytes is an install text file's bytes in the font's code page. Windows
// Cyrillic becomes DOS Cyrillic, as the second game's loaders call
// CharToOemA (R2-ENGINE-052, R2-ENGINE-093; DIV-2378, DIV-2844); any other
// text keeps its bytes.
func (c TextCode) Bytes(b []byte) []byte {
	if c.From != 1251 || c.To != 866 {
		return b
	}
	return vfs.WindowsCyrillicToDOS(b)
}

// textCode is the install's text code; an install with no archives keeps
// every byte.
func (in *InstallResources) textCode() TextCode {
	if in == nil || in.Archives == nil {
		return TextCode{}
	}
	return InstallTextCode(in.Archives.Containers, in.Archives.Game().Edition())
}

// LoadTextTable reads one address out of the container filesystem, in the
// font's code page under code, and splits it. A read failure is no table,
// which every accessor reports as absent.
func LoadTextTable(src terrain.EntrySource, addr string, code TextCode) *TextTable {
	if src == nil {
		return nil
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil
	}
	return SplitTextTable(code.Bytes(b))
}

// Lines is how many lines the table holds. A nil table holds none.
func (t *TextTable) Lines() int {
	if t == nil {
		return 0
	}
	return len(t.lines)
}

// At is line i, and whether the table states one there.
//
// AN EMPTY LINE IS ABSENT. `main.txt` ships blank lines at 91 and 93, and the
// original's own reader would hand back a zero-length string there; a caller
// asking for a word wants the authored one rather than nothing drawn at all.
func (t *TextTable) At(i int) (string, bool) {
	if t == nil || i < 0 || i >= len(t.lines) || t.lines[i] == "" {
		return "", false
	}
	return t.lines[i], true
}

// InstallWords is the install's own text, read once, in the two indexing
// styles the decode distinguishes.
type InstallWords struct {
	main           *TextTable
	dialogs        *TextTable
	stats          *TextTable
	unitNames      *TextTable
	buildingNames  *TextTable
	sites          *TextTable
	spellNames     *TextTable
	spellBookNames *TextTable
	help           string

	// Selector is the install's language digit, `main\id`'s last character
	// minus '0' (`TEXT-CHARGEN-029`). It is carried beside the tables because
	// the words and the byte conversion that draws them are one property of
	// one install; the tables already hold the font's code page (TextCode).
	Selector int

	// Language is the install's language entry, the base profile's Language
	// ("english", "russian"); empty reads English. It chooses the engine's own
	// words (pkg/words); Selector only converts bytes.
	Language string
}

// LoadInstallWords reads the tables once, in the font's code page under
// code. It cannot fail: an install
// missing either file yields a word set in which every index is absent, and
// every program-chosen word then stays the authored English one.
func LoadInstallWords(src terrain.EntrySource, code TextCode) *InstallWords {
	w := &InstallWords{
		main:           LoadTextTable(src, MainTextPath, code),
		dialogs:        LoadTextTable(src, DialogsTextPath, code),
		stats:          LoadTextTable(src, StatsTextPath, code),
		unitNames:      LoadTextTable(src, UnitNameTextPath, code),
		buildingNames:  LoadTextTable(src, BuildingTextPath, code),
		sites:          LoadTextTable(src, SitesTextPath, code),
		spellNames:     LoadTextTable(src, SpellNamesTextPath, code),
		spellBookNames: LoadTextTable(src, SpellBookNamesTextPath, code),
		Selector:       LanguageSelector(src),
	}
	if src != nil {
		if b, err := src.ReadFile(HelpTextPath); err == nil {
			w.help = string(code.Bytes(b))
		}
	}
	return w
}

// Stats, UnitName and Site are distinct table-local index spaces. Keeping
// separate accessors makes it impossible for a caller to accidentally treat a
// local item label or actor id as a global main.txt subscript.
func (w *InstallWords) Stats(i int) (string, bool) {
	if w == nil {
		return "", false
	}
	return w.stats.At(i)
}

func (w *InstallWords) UnitName(i int) (string, bool) {
	if w == nil {
		return "", false
	}
	return w.unitNames.At(i)
}

func (w *InstallWords) Site(i int) (string, bool) {
	if w == nil {
		return "", false
	}
	return w.sites.At(i)
}

func (w *InstallWords) Global(k int) (string, bool) {
	if w == nil || k >= w.main.Lines() {
		return "", false
	}
	return w.main.At(k)
}

func (w *InstallWords) Dialogs(i int) (string, bool) {
	if w == nil {
		return "", false
	}
	return w.dialogs.At(i)
}

// Words is the resolved word set: this build's authored English, with every
// field the install states at its decoded index overwritten by the install's
// own line.
//
// RESOLUTION IS PER FIELD AND NOT PER TABLE (AC-8). An install that states six
// of the eight menu labels resolves six; the other two stay English. There is no
// all-or-nothing rule anywhere in the decode to reproduce, and a partial table
// is the case a fallback exists for.
func (w *InstallWords) Words() ui.Words {
	out := ui.AuthoredWords()
	if w != nil {
		book := words.For(w.Language)
		out.Engine = book
		encode := func(s string) string {
			var b []byte
			for _, r := range s {
				c, _ := textinput.EncodeRune(r, w.Selector)
				b = append(b, c)
			}
			return string(b)
		}
		out.ItemMagic = encode(book.Text("item.magic"))
		out.DurationUnit = encode(book.Text("item.duration_unit"))
		out.ItemSpellOf = encode(book.Text("item.spell_of"))
		out.ItemRays = encode(book.Text("item.rays"))
		out.TavernSleep = encode(book.Text("tavern.sleep"))
		// An install in any language but the engine's reference one states
		// these three words in its own stats.txt; the reference language keeps
		// the engine's.
		if book.Lang() != locale.Fallback {
			for dst, k := range map[*string]int{&out.ItemCasts: 42, &out.ItemDamage: 43, &out.ItemRange: 38} {
				if s, ok := w.Stats(k); ok {
					*dst = s
				}
			}
		}
	}
	if w != nil && w.spellNames != nil {
		for id := 1; id < len(out.ItemSpellNames); id++ {
			if s, ok := w.spellNames.At(id - 1); ok {
				out.ItemSpellNames[id] = s
			}
		}
		// DIV-2177: a language may name Drain Life by its own alias.
		if alias := out.Engine.Text("item.drain_life_alias"); alias != "" {
			out.ItemSpellNames[11] = EncodeInstallText(alias, w.Selector)
		}
	}
	if w != nil && w.spellBookNames.Lines() == len(originalBookIDs) {
		for cell, id := range originalBookIDs {
			if s, ok := w.spellBookNames.At(cell); ok {
				out.SpellBookNames[id] = spellBookRowName(s)
			}
		}
	}
	g := func(dst *string, k int) {
		if s, ok := w.Global(k); ok {
			*dst = s
		}
	}
	g(&out.ItemMagic, 189) // Installed item enchantment heading.
	g(&out.ItemSpellOf, 90)
	g(&out.ItemSpellOfSuffix, 91)
	g(&out.ItemCasts, 92) // The item formatter's own verb for a cast-spell effect.

	d := func(dst *string, i int) {
		if s, ok := w.Dialogs(i); ok {
			*dst = s
		}
	}

	g(&out.NoticeButton, mainSlotNoticeButton)
	g(&out.MissionWon, mainSlotMissionWon)
	g(&out.MissionLost, mainSlotMissionLost)
	g(&out.PauseNotice, mainSlotPauseNotice)
	g(&out.NoHeroSelected, mainSlotNoHeroSelected)
	if w != nil {
		out.HelpText = w.help
	}
	g(&out.PanelCaptions[mainSlotMercenary], mainSlotMercenary)
	g(&out.TipClose, mainSlotTipClose)
	g(&out.TipShowNext, mainSlotTipShowNext)
	g(&out.SaveAcknowledgement, mainSlotSaveAcknowledgement)
	g(&out.WorldHomeDetail, mainSlotWorldReturn)
	g(&out.WorldPayment, mainSlotWorldPayment)
	g(&out.SelectionStatus[0], 47)
	g(&out.SelectionStatus[1], 48)
	g(&out.SelectionStatus[2], 49)
	g(&out.SelectionStatus[3], 50)
	for i := range out.SkillRaised {
		g(&out.SkillRaised[i], mainSlotSkillRaised+i)
	}
	g(&out.PickedUp, mainSlotPickedUp)
	g(&out.PickedUpNow, mainSlotPickedUpNow)
	g(&out.PickedUpPieces, mainSlotPickedUpPieces)
	g(&out.PickedUpGold, mainSlotPickedUpGold)
	g(&out.PickedUpGoldUnit, mainSlotPickedUpGoldUnit)
	for _, slots := range [][2]int{{94, 116}, {218, 220}} {
		for i := slots[0]; i <= slots[1]; i++ {
			g(&out.SettingNotice[i], i)
		}
	}

	for _, i := range []int{15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
		27, 28, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
		45, 46, 190, 191} {
		g(&out.PanelCaptions[i], i)
	}
	for id := 1; id < len(out.BuildingNames); id++ {
		if w != nil && w.buildingNames != nil {
			if s, ok := w.buildingNames.At(id - 1); ok {
				out.BuildingNames[id] = s
			}
		}
	}
	for i := range out.UnitNames {
		if s, ok := w.UnitName(i); ok {
			out.UnitNames[i] = s
		}
	}
	for i := range out.ItemStats {
		if s, ok := w.Stats(i); ok {
			out.ItemStats[i] = s
		}
	}
	if s, ok := w.Site(0); ok {
		out.WorldHomeTitle = s
	}

	d(&out.MenuSave, dialogsSlotSave)
	d(&out.MenuLoad, dialogsSlotLoad)
	d(&out.MenuDiplomacy, dialogsSlotDiplomacy)
	d(&out.MenuGameOptions, dialogsSlotGameOptions)
	d(&out.MenuSoundOptions, dialogsSlotSoundOptions)
	d(&out.MenuQuestObjectives, dialogsSlotQuestObjectives)
	d(&out.QuestHeading, dialogsSlotQuestHeading)
	d(&out.MenuEndQuest, dialogsSlotEndQuest)
	d(&out.MenuReturn, dialogsSlotReturn)
	d(&out.MenuAbort, dialogsSlotAbort)
	d(&out.MenuChangeMap, dialogsSlotChangeMap)
	d(&out.MenuVictory, dialogsSlotVictory)
	d(&out.OutcomeContinue, dialogsSlotContinue)
	d(&out.MenuExitMain, dialogsSlotExitMain)
	d(&out.MenuExitWindows, dialogsSlotExitWindows)
	d(&out.MapListSize, dialogsSlotMapListSize)
	d(&out.MapListColumns[0], dialogsSlotMapListColumn0)
	d(&out.MapListColumns[1], dialogsSlotMapListColumn1)

	g(&out.ShopUndo, mainSlotShopUndo)
	g(&out.ShopBuy, mainSlotShopBuy)
	g(&out.ShopSell, mainSlotShopSell)
	g(&out.ShopExit, mainSlotShopExit)
	g(&out.SchoolTrain, mainSlotSchoolTrain)
	g(&out.SchoolExit, mainSlotSchoolExit)
	g(&out.TavernHire, mainSlotTavernHire)
	g(&out.TavernHired, 257)
	g(&out.TavernFire, mainSlotTavernFire)
	g(&out.TavernTalk, mainSlotTavernTalk)
	g(&out.TavernExit, mainSlotTavernExit)

	// The command panel's eight cell labels, main.txt global slots 0-7 in the
	// panel's own cell order (docs/1028-command-panel contract; `MENU-COMBAT-
	// 019`). NO NAMED SLOT CONST HERE, unlike every field above: the global
	// index IS the cell index for these eight, so a loop over ui.Words.
	// Command's own length reads exactly the slots the panel's own cell
	// constants name, with no second table translating one into the other.
	for i := range out.Command {
		g(&out.Command[i], i)
	}
	for i := range out.Hover {
		g(&out.Hover[i], i)
	}
	for i := range out.SiteHints {
		if s, ok := w.Site(i); ok {
			out.SiteHints[i] = s
		}
	}
	return out
}
