package game

import (
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/locale"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
	"againrom/pkg/words"
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
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
	return string(encodeInstallRunes(s, selector, false))
}

// modTextFallback is the plain form of each typographic character an author
// types: the install has a byte for none of them (the installed pages are
// ASCII or CP866 letters), so a mod's text meets the encoder's drop unless
// this table gives the character a form first.
var modTextFallback = map[rune]string{
	'«': `"`, '»': `"`, '“': `"`, '”': `"`, '„': `"`, '‟': `"`,
	'‘': `'`, '’': `'`, '‚': `'`, '‛': `'`, '‹': `'`, '›': `'`,
	'‐': `-`, '‑': `-`, '‒': `-`, '–': `-`, '—': `-`, '―': `-`, '−': `-`,
	'…': `...`,
	' ': ` `, ' ': ` `, ' ': ` `,
}

// EncodeModText is EncodeInstallText for text a mod author wrote: a character
// in modTextFallback becomes its plain form before the encoder runs.
func EncodeModText(s string, selector int) string {
	return string(encodeInstallRunes(s, selector, true))
}

// encodeInstallRunes is the install text codec under its Drop policy. With
// fallback set, a rune in modTextFallback is replaced by its plain form first.
func encodeInstallRunes(s string, selector int, fallback bool) []byte {
	if fallback {
		var plain strings.Builder
		for _, r := range s {
			if p, ok := modTextFallback[r]; ok {
				plain.WriteString(p)
			} else {
				plain.WriteRune(r)
			}
		}
		s = plain.String()
	}
	b, _ := textinput.EncodeText(s, selector, textinput.Drop)
	return []byte(b)
}

// ModTextDropped reports the first rune of a mod's text file that the base
// under selector cannot show after the plain-form table, and whether the file
// is read as UTF-8 at all. The Byte Order Mark is not counted.
func ModTextDropped(b []byte, selector int) (rune, bool) {
	if !utf8.Valid(b) {
		return 0, false
	}
	for _, r := range string(b) {
		if r < 0x20 || r == 0x7f || r == 0xfeff {
			continue
		}
		if _, ok := modTextFallback[r]; ok {
			continue
		}
		if _, ok := textinput.EncodeRune(r, selector); !ok {
			return r, true
		}
	}
	return 0, false
}

// EncodeInstallTextFile turns a mod's UTF-8 text file into the install's own
// bytes under selector, rune by rune through the one encoder after the
// plain-form table. A control byte (the CR LF line ends, a tab) is kept as it
// stands; a rune the selector cannot encode is dropped. A file that is not
// valid UTF-8 was written in the install's own bytes before mods wrote UTF-8,
// and is returned unchanged.
func EncodeInstallTextFile(b []byte, selector int) []byte {
	if !utf8.Valid(b) {
		return slices.Clone(b)
	}
	out := make([]byte, 0, len(b))
	for _, r := range string(b) {
		if r < 0x20 || r == 0x7f {
			out = append(out, byte(r))
			continue
		}
		out = append(out, encodeInstallRunes(string(r), selector, true)...)
	}
	return out
}

// DecodeInstallTextFile is EncodeInstallTextFile's inverse for an installed
// text file: each byte becomes the rune textinput.DecodeByte reads it as under
// selector, and a control byte is kept. It answers false when some byte has no
// rune or the UTF-8 it makes does not encode back to b, so the caller never
// writes text that changes the file.
func DecodeInstallTextFile(b []byte, selector int) ([]byte, bool) {
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		if c < 0x20 || c == 0x7f {
			out = append(out, c)
			continue
		}
		r, ok := textinput.DecodeByte(c, selector)
		if !ok {
			return nil, false
		}
		out = utf8.AppendRune(out, r)
	}
	if !utf8.Valid(out) || !bytes.Equal(EncodeInstallTextFile(out, selector), b) {
		return nil, false
	}
	return out, true
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
// every byte. Selector is the font selector the mod editor's page text is
// decoded under; only the page codecs read it.
type TextCode struct {
	From, To int
	Selector int
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
	if c.From != locale.WindowsCyrillic || c.To != locale.DOSCyrillic {
		return b
	}
	return vfs.WindowsCyrillicToDOS(b)
}

// FileBytes is Bytes' inverse: text in the font's code page as the install's
// text files hold it, so a file a mod serves reads back through Bytes as
// written.
func (c TextCode) FileBytes(b []byte) []byte {
	if c.From != locale.WindowsCyrillic || c.To != locale.DOSCyrillic {
		return b
	}
	return vfs.DOSCyrillicToWindows(b)
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
	return installText(src, textTableSource.named(addr), code).table()
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

	// Spells is the game's spell presentation set: how many rows `spell.txt`
	// names and which row each `spells.txt` line names. Nil reads the first
	// game's.
	Spells *data.SpellPresentation
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
	if help := installText(src, helpTextSource.at(), code); help.present() {
		w.help = string(help.bytes())
	}
	return w
}

// Stats, UnitName and Site are distinct table-local index spaces. Keeping
// separate accessors makes it impossible for a caller to accidentally treat a
// local item label or actor id as a global main.txt subscript.
// spellSet is the spell presentation set the words name spells by; an
// install words value no game named takes the first game's.
func (w *InstallWords) spellSet() *data.SpellPresentation {
	if w == nil || w.Spells == nil {
		return data.FirstSpellPresentation()
	}
	return w.Spells
}

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
		encode := func(s string) string { return EncodeInstallText(s, w.Selector) }
		out.ItemMagic = encode(book.Text("item.magic"))
		out.DurationUnit = encode(book.Text("item.duration_unit"))
		out.ItemSpellOf = encode(book.Text("item.spell_of"))
		out.ItemRays = encode(book.Text("item.rays"))
		out.TavernSleep = encode(book.Text("tavern.sleep"))
		out.Received = encode(book.Text("item.received"))
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
	spells := w.spellSet()
	if w != nil && w.spellNames != nil {
		for id := 1; id <= spells.NamedRows && id < len(out.ItemSpellNames); id++ {
			if s, ok := w.spellNames.At(id - 1); ok {
				out.ItemSpellNames[id] = s
			}
		}
		// DIV-2177: a language may name Drain Life by its own alias.
		if alias := out.Engine.Text("item.drain_life_alias"); alias != "" && spells.DrainLifeAliasRow > 0 {
			out.ItemSpellNames[spells.DrainLifeAliasRow] = EncodeInstallText(alias, w.Selector)
		}
	}
	if w != nil && w.spellBookNames.Lines() == len(spells.BookRows) {
		for cell, id := range spells.BookRows {
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

// textCodec is how a text source row's bytes reach their reader.
type textCodec uint8

const (
	storedBytes  textCodec = iota // the bytes as the install stores them
	fontBytes                     // TextCode.Bytes: the font's code page
	storedUTF8                    // the stored bytes as DOS Cyrillic, in UTF-8
	fontUTF8                      // the font's bytes as DOS Cyrillic, in UTF-8
	pageUTF8                      // DecodeInstallTextFile under the selector; kept when it does not decode
	heldPageUTF8                  // pageUTF8 for bytes that are kept when they are UTF-8 already
)

// textAbsence is which read of a source row is no text.
type textAbsence uint8

const (
	absentMissing textAbsence = iota // only a failed read; an empty file is text
	absentEmpty                      // a failed read or a file of no bytes
	absentBlank                      // a failed read or a file of only white space
)

// textForm is how a reader takes a row's text apart: whole, the loader's CR
// walk, CRLF-or-LF lines, or #key sections.
type textForm uint8

const (
	wholeText textForm = iota
	tableLines
	lfLines
	keySections
)

// textEdge is what the row trims: white space at both ends before the split,
// or one final LF after CRLF becomes LF.
type textEdge uint8

const (
	edgeKept textEdge = iota
	edgeTrimSpace
	edgeFinalLF
)

// textSource is one row of installText. Its address form is fixed (address),
// numbered (address is a pattern of positive numbers), named by the caller
// (no address) or held (the caller already holds the bytes). A file over limit
// bytes is absent; maxLines cuts a line list. A numbered row takes its first
// number from zero when zeroFirst is set.
type textSource struct {
	address   string
	numbered  bool
	zeroFirst bool
	held      bool
	codec     textCodec
	absence   textAbsence
	limit     int
	maxLines  int
	form      textForm
	edge      textEdge
}

// The text source rows: every install text file pkg/game reads is read by
// installText under one of them.
var (
	textTableSource       = &textSource{codec: fontBytes, form: tableLines}
	helpTextSource        = &textSource{address: HelpTextPath, codec: fontBytes}
	tipTextSource         = &textSource{codec: fontBytes}
	tipSectionSource      = &textSource{codec: fontBytes, form: keySections}
	briefingSource        = &textSource{address: mainPrefix + "text/battle/m%d/briefing.txt", numbered: true, codec: storedUTF8, edge: edgeTrimSpace}
	eventTextSource       = &textSource{address: mainPrefix + "text/battle/m%d/event%02d.txt", numbered: true}
	missionTextSource     = &textSource{address: mainPrefix + "text/mission%d.txt", numbered: true, codec: fontBytes, form: keySections}
	townTextSource        = &textSource{address: secondTownText, codec: fontBytes, form: keySections}
	townDialogueSource    = &textSource{}
	missionTitleSource    = &textSource{address: mainPrefix + "text/battle/m%d/title.txt", numbered: true, absence: absentBlank}
	missionBriefMapSource = &textSource{address: mainPrefix + "text/battle/m%d/briefmap.txt", numbered: true, absence: absentBlank}
	editorTitleSource     = &textSource{address: mainPrefix + "text/battle/m%d/title.txt", numbered: true, codec: heldPageUTF8, absence: absentBlank, edge: edgeTrimSpace}
	worldMapPageSource    = &textSource{absence: absentEmpty}
	worldMapRecordSource  = &textSource{codec: fontBytes}
	heldSectionSource     = &textSource{held: true, form: keySections}
	missionPageSource     = &textSource{}
	installedPageSource   = &textSource{codec: pageUTF8}
	editorPageSource      = &textSource{held: true, codec: heldPageUTF8}
	reportPartSource      = &textSource{held: true, codec: storedUTF8}
	creditsRollSource     = &textSource{address: mainPrefix + "text/credits.txt", codec: fontBytes, limit: 256 * 1024, maxLines: 4096, form: lfLines, edge: edgeFinalLF}
	hallCreditsSource     = &textSource{address: mainPrefix + "text/credits.txt", codec: fontUTF8, form: lfLines, edge: edgeTrimSpace}
	npcNameLinesSource    = &textSource{address: mainPrefix + "text/npcnames.txt", codec: fontBytes, form: lfLines}
	cheatLinesSource      = &textSource{address: MainTextPath, form: lfLines}
	itemNameTextSource    = &textSource{address: ItemNameTextAddress, codec: fontBytes}
	heroPictureSource     = &textSource{address: HeroPictureAddress}
	missionPageDirSource  = &textSource{address: mainPrefix + "text/battle/m%d/", numbered: true}
	innNPCTextSource      = &textSource{address: mainPrefix + "text/inn/npc/npc%02dm%d.txt", numbered: true, zeroFirst: true}
	innMercenarySource    = &textSource{address: mainPrefix + "text/inn/mercenary/npc%02d.txt", numbered: true}
	shopTalkTextSource    = &textSource{address: mainPrefix + "text/shop/npc31m%d.txt", numbered: true}
	schoolTalkTextSource  = &textSource{address: mainPrefix + "text/training/npc34m%d.txt", numbered: true}
)

// townBuildingTalkSource is the row of the text a town building's keeper
// speaks.
var townBuildingTalkSource = map[TownBuilding]*textSource{TownShop: shopTalkTextSource, TownSchool: schoolTalkTextSource}

// textAddress is one file of a row: its address, or the bytes a held row
// holds. A numbered row refuses a number that is not positive.
type textAddress struct {
	row  *textSource
	path string
	ok   bool
	held []byte
}

// at is the row's fixed address, or its pattern filled with n.
func (s *textSource) at(n ...int) textAddress {
	if !s.numbered {
		return textAddress{row: s, path: s.address, ok: true}
	}
	args := make([]any, len(n))
	for i, v := range n {
		if v < 0 || v == 0 && !(i == 0 && s.zeroFirst) {
			return textAddress{row: s}
		}
		args[i] = v
	}
	return textAddress{row: s, path: fmt.Sprintf(s.address, args...), ok: true}
}

// within is the file called name inside the directory a directory row names.
func (a textAddress) within(name string) textAddress {
	if !a.ok {
		return a
	}
	a.path += name
	return a
}

// numbersIn reads the numbers of a numbered row's address out of path, as at
// forms them. It reports false when path is not the row's form.
func (s *textSource) numbersIn(path string, count int) ([]int, bool) {
	n := make([]int, count)
	args := make([]any, count)
	for i := range n {
		args[i] = &n[i]
	}
	form := strings.ReplaceAll(s.address, "%02d", "%d")
	if got, err := fmt.Sscanf(path, form, args...); err != nil || got != count {
		return nil, false
	}
	return n, true
}

// named is the row read at the caller's address.
func (s *textSource) named(path string) textAddress {
	return textAddress{row: s, path: path, ok: true}
}

// holding is a held row over bytes the caller already holds.
func (s *textSource) holding(b []byte) textAddress {
	return textAddress{row: s, ok: true, held: b}
}

var (
	errNoTextArchive = errors.New("no install text archive")
	errNoTextAddress = errors.New("no install text address")
)

// lineText is one install text file as its row reads it.
type lineText struct {
	row     *textSource
	body    []byte
	ok      bool
	readErr error
}

// installText reads the file at a under its row: the read, the absence rule
// and the codec. Every install text file pkg/game reads comes through here.
func installText(src entrySource, a textAddress, code TextCode) lineText {
	t := lineText{row: a.row}
	b := a.held
	switch {
	case a.row.held:
	case !a.ok:
		t.readErr = errNoTextAddress
		return t
	case src == nil:
		t.readErr = errNoTextArchive
		return t
	default:
		raw, err := src.ReadFile(a.path)
		if err != nil {
			t.readErr = err
			return t
		}
		b = raw
	}
	switch {
	case a.row.limit > 0 && len(b) > a.row.limit,
		a.row.absence == absentEmpty && len(b) == 0,
		a.row.absence == absentBlank && len(strings.TrimSpace(string(b))) == 0:
		return t
	}
	t.body, t.ok = a.row.codec.decode(b, code), true
	return t
}

func (c textCodec) decode(b []byte, code TextCode) []byte {
	switch c {
	case fontBytes:
		return code.Bytes(b)
	case storedUTF8:
		return []byte(vfs.DecodeText(b))
	case fontUTF8:
		return []byte(vfs.DecodeText(code.Bytes(b)))
	case heldPageUTF8:
		if utf8.Valid(b) {
			return b
		}
		fallthrough
	case pageUTF8:
		if u, ok := DecodeInstallTextFile(b, code.Selector); ok {
			return u
		}
	}
	return b
}

// present reports whether the row found text there.
func (t lineText) present() bool { return t.ok }

// err is the failed read's error, nil when the read succeeded.
func (t lineText) err() error { return t.readErr }

// bytes is the whole text, nil when absent.
func (t lineText) bytes() []byte { return t.body }

// text is the whole text with the row's edge trimmed.
func (t lineText) text() string {
	if t.row.edge == edgeTrimSpace {
		return strings.TrimSpace(string(t.body))
	}
	return string(t.body)
}

func (t lineText) must(form textForm) {
	if t.row.form != form {
		panic("install text row read in another form")
	}
}

// table is the loader's CR walk over the text, nil when absent.
func (t lineText) table() *TextTable {
	t.must(tableLines)
	if !t.ok {
		return nil
	}
	return SplitTextTable(t.body)
}

// lines is the text split on LF after CRLF becomes LF, nil when absent.
func (t lineText) lines() []string {
	t.must(lfLines)
	if !t.ok {
		return nil
	}
	s := string(t.body)
	if t.row.edge == edgeTrimSpace {
		s = strings.TrimSpace(s)
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if t.row.edge == edgeFinalLF {
		s = strings.TrimSuffix(s, "\n")
	}
	out := strings.Split(s, "\n")
	if t.row.maxLines > 0 && len(out) > t.row.maxLines {
		out = out[:t.row.maxLines]
	}
	return out
}

// section is key's section of the text, empty when it has none.
func (t lineText) section(key string) string {
	t.must(keySections)
	if !t.ok {
		return ""
	}
	return secondGameTextSection(t.body, key)
}

// numberedSection is the section the key prefix followed by n names.
func (t lineText) numberedSection(prefix string, n int) string {
	return t.section(fmt.Sprintf("%s%d", prefix, n))
}

// sectionText is key's section in UTF-8, trimmed of white space, and whether
// the text has the section.
func (t lineText) sectionText(key string) (string, bool) {
	s := t.section(key)
	return strings.TrimSpace(vfs.DecodeText([]byte(s))), s != ""
}

// sectionCut is key's section less its last n bytes and the NULs before them;
// empty when the section is shorter than n.
func (t lineText) sectionCut(key string, n int) string {
	s := t.section(key)
	if len(s) < n {
		return ""
	}
	return strings.TrimRight(s[:len(s)-n], "\x00")
}

// R2-ENGINE-049/050: first substring, two separator bytes, then the next '#'.
func secondGameTextSection(payload []byte, key string) string {
	source := string(dialoguePayload(payload))
	at := strings.Index(source, "#"+strings.ToLower(key))
	if at < 0 {
		return ""
	}
	start := at + len(key) + 3
	if start >= len(source) {
		return ""
	}
	body := source[start:]
	if end := strings.IndexByte(body, '#'); end >= 0 {
		body = body[:end]
	}
	return body
}

// modTextFile is a mod's UTF-8 text file as the install's text files hold it:
// EncodeInstallTextFile's bytes in the install's code page under c.
func (c TextCode) modTextFile(b []byte, selector int) []byte {
	return c.FileBytes(EncodeInstallTextFile(b, selector))
}
