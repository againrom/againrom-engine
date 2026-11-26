package ui

import (
	"image"
	"image/color"
	"strings"

	"againrom/pkg/namefold"
)

// SavePage describes a load or save screen. Its handlers read the layout,
// captions, list rules and optional slot rule.
type SavePage struct {
	// Panel is the window frame's body.
	Panel image.Rectangle
	// Texts are drawn in order after the frame.
	Texts []PageText
	// List is the list's argument rectangle and Rows its visible rows.
	List image.Rectangle
	Rows int
	// Message is where the refusal or the selected row's detail is drawn.
	Message image.Rectangle
	// Buttons are Ok, Delete and Cancel.
	Buttons [3]PageButton
	// Confirm lays out the delete confirmation.
	Confirm PageConfirm
	// Ruled rules every list row and fills the chosen row opaque and dark.
	Ruled bool
	// Framed draws every list row as its own sunken frame, with no well
	// around the list, and fills the chosen row dark; BarGap is the space
	// between the rows and the bar.
	Framed bool
	BarGap int
	// RowHint shows a row's detail as a hover caption over the row, not as
	// the line under the list.
	RowHint bool
	// NameHint is the hover caption of the save page's name box.
	NameHint string
	// OKNeedsSave disables Ok while no save is listed.
	OKNeedsSave bool

	// By default the save role starts with EmptySlot, followed by
	// the listed saves except the Hidden names. NameBox holds the typed
	// label, at most NameLimit characters when positive. Ok writes it to the
	// selected row's file, or to the file Slot names from the box text for row 0, with no
	// overwrite confirmation.
	NameBox   image.Rectangle
	NameLimit int
	EmptySlot string
	// EmptyInk draws the Empty Slot row in this colour; zero keeps the list's.
	EmptyInk color.RGBA
	Hidden   []string
	Slot     func(directory, label string, listed []string) string

	DirectoryBox image.Rectangle
	Navigation   [2]PageButton
	// The defaults are label-and-slot entry; these rows describe optional
	// file browsing, keyboard focus and the page's additional paint.
	fileNames, directories, overwriteQuestion, keyboardFocus bool
	activateRows, retainDeletedRow, clearDeleteAttempt       bool
	clearListMessage, preserveListPointer, stopListError     bool
	initialName                                              string
	focusOrder                                               []saveControl
	selection                                                TextSelection
	background, fill, border                                 color.RGBA
	frame                                                    image.Rectangle
	artOnly, deferredText                                    bool
	blocks                                                   []saveTextBlock
	empty, detail                                            PageText
	confirmHeading, deleteHeading, scrollHint                PageText
	confirmAt                                                image.Point
	confirmWidth, confirmRows, confirmPitch                  int
	confirmButtons                                           [3]PageButton
	wheel, hoverWheel                                        scrollSpec
	controlSound, standalone                                 bool
	errorName                                                string
	latchIDs                                                 []int
	ink                                                      color.RGBA
	fitButtons, selectNameOnOpen, reportDeletePath           bool
}

// PageText is one caption: centred in Box when Box is set, else drawn from
// At, in Ink or the window text colour when Ink is zero. Listing hides it
// while a deletion awaits confirmation. Focus draws it gold while the list
// has the focus.
type PageText struct {
	Text    string
	At      image.Point
	Box     image.Rectangle
	Ink     color.RGBA
	Listing bool
	Focus   bool
	word    string
}

// PageButton is one push button.
type PageButton struct {
	Rect    image.Rectangle
	Caption string
	word    string
}

// PageConfirm is the delete confirmation: Before at At and the selected
// label in Label, in the page's body. A Window with a Size instead asks in a
// notice window of that shape over the page: Before, the label and After as
// one text, and the answers Yes and No. A '#' in either text is a space.
type PageConfirm struct {
	Before  string
	At      image.Point
	Label   image.Rectangle
	After   string
	Window  Dialog
	Yes, No string
}

// windowed reports whether the page asks in a notice window.
func (c PageConfirm) windowed() bool { return c.Window.ShapeSize != (image.Point{}) }

// question is the window asking to delete the save whose label the font
// draws as label.
func (c PageConfirm) question(label string) dialogRun {
	s := strings.ReplaceAll(c.Before, "#", " ") + label + strings.ReplaceAll(c.After, "#", " ")
	w := dialogRun{Dialog: c.Window}
	w.Runs, w.Answers = []noticeRow{{{Text: s}}}, []string{c.Yes, c.No}
	return w
}

// pageSeams holds page descriptions; nil selects the built-in descriptions.
// files answers whether any save file exists; hasSave is its answer when the
// menu last showed.
type pageSeams struct {
	loadSet, saveSet *SavePage
	// menu is the game's game-menu data.
	menu    GameMenuProfile
	files   func() bool
	hasSave bool
	// loadable answers whether the load list has a row without building it.
	loadable func() bool
	// characterSet is the character role's data; nil keeps the defaults.
	characterSet *CharacterPage
	// characterFiles is the character role's store.
	characterFiles CharacterFiles
	// mapWords is the map role's words.
	mapWords MapChoiceWords
}

// CharacterPage is the list page's character role, the page the menu's
// Multiplayer button opens (MENU-168, R2-ENGINE-1844). It keeps the game's
// load page window, list and Cancel, with its own title and Ok caption;
// Delete shows only where the game's original page has it. New is the
// list's last row, after the character files; Ok on it opens the store's
// generator.
type CharacterPage struct {
	Title, OK, New string
	Delete         bool
}

func defaultCharacterPage() CharacterPage {
	return CharacterPage{Title: "Characters", OK: "Accept", New: "New Character", Delete: true}
}

// CharacterFiles is the character role's store. List answers the character
// rows and the message line, which counts the files it left out; CanDelete
// and PrepareDelete are the Load page's delete seams for a row's name. New
// is the generator the New row opens, whose accept stores the character; nil
// leaves the New row doing nothing. Maps are the rows of the map choice a
// character row opens, and Enter opens the chosen map with that character;
// either nil leaves a character row doing nothing.
type CharacterFiles struct {
	List          func() ([]SaveEntry, string)
	CanDelete     func(string) bool
	PrepareDelete func(string) (func() error, error)
	New           func() *ChargenEntry
	Maps          func() []MapChoiceItem
	Enter         func(character, mapSource string) (MapOpener, error)
}

// SetCharacterPage installs a game's character role data.
func (a *App) SetCharacterPage(p CharacterPage) { a.flow.pages.characterSet = &p }

// SetCharacterFiles installs the character role's store.
func (a *App) SetCharacterFiles(c CharacterFiles) { a.flow.pages.characterFiles = c }

// characterPage is the installed character role data, or the defaults.
func (f *flow) characterPage() CharacterPage {
	if f.pages.characterSet != nil {
		return *f.pages.characterSet
	}
	return defaultCharacterPage()
}

// OpenCharacters shows the character list over the menu, as the menu's
// Multiplayer button does.
func (a *App) OpenCharacters() { a.flow.openCharacters() }

// CharacterList reports whether the list page shows its character role.
func (a *App) CharacterList() bool {
	return a.flow.screen == ScreenLoad && a.flow.loadUI.openRole().roleName == characterRoleName
}

// asCharacterList is load page p as the character list: the load page's
// boxed title holds c's title and its other captions go.
func (p SavePage) asCharacterList(c CharacterPage) SavePage {
	var texts []PageText
	for _, t := range p.Texts {
		if t.Box != (image.Rectangle{}) {
			t.Text = c.Title
			texts = append(texts, t)
			break
		}
	}
	p.Texts = texts
	p.Buttons[loadOKButton].Caption = c.OK
	if !c.Delete {
		p.Buttons[loadDeleteButton] = PageButton{}
	}
	p.OKNeedsSave = false
	return p
}

// SavePages supplies the load page and an optional save page description.
type SavePages struct {
	Load SavePage
	Save *SavePage
}

// SetSavePages installs a game's pages. The first game's LOAD window is the
// default and needs no call.
func (a *App) SetSavePages(p SavePages) {
	load := p.Load
	a.flow.pages.loadSet = &load
	a.flow.pages.saveSet = p.Save
}

// firstLoadPage is the first game's LOAD window built from its words.
func firstLoadPage(w LoadWindowWords) SavePage {
	if w.Title == "" {
		w = defaultLoadWindowWords()
	}
	// The rows are the bare save labels with no detail line under the list
	// (RowHint with no detail), each row framed, as the EN original draws
	// the page (DIV-4143).
	p := SavePage{
		Panel: loadPanel,
		Texts: []PageText{
			{Text: w.Title, Box: image.Rect(129, 86, 513, 114), Ink: townShellText},
			{Text: w.Subtitle, At: image.Pt(116, 128), Ink: loadSelectedText, Listing: true},
		},
		List:    loadListArg,
		Rows:    loadVisibleRows,
		Framed:  true,
		BarGap:  3,
		RowHint: true,
		Message: loadMessageBox,
		Confirm: PageConfirm{Before: w.Confirm, At: image.Pt(125, 146), Label: image.Rect(122, 180, 520, 210)},
	}
	for i, caption := range []string{w.OK, w.Delete, w.Cancel} {
		p.Buttons[i] = PageButton{Rect: image.Rect(149+i*122, 368, 246+i*122, 392), Caption: caption}
	}
	return p
}

// loadPage is the page the load screen shows: the open role's page.
func (f *flow) loadPage() SavePage { return f.loadUI.openRole().page(f) }

// installedLoadPage is the game's Load page, or the first game's LOAD
// window.
func (f *flow) installedLoadPage() SavePage {
	if f.pages.loadSet != nil {
		return *f.pages.loadSet
	}
	return firstLoadPage(f.loadUI.words)
}

// hidden reports whether a save page leaves name out of its list.
func (p SavePage) hidden(name string) bool {
	for _, h := range p.Hidden {
		if namefold.HostEqual(h, name) {
			return true
		}
	}
	return false
}

// pageView is one open page: its data, its pointer state, its list and the
// saves behind the list's rows. Optional empty and directory rows precede
// the save rows.
type pageView struct {
	page  SavePage
	ui    *loadWindow
	list  *Picker
	saves []SaveEntry
	save  *savePageState
}

// asking reports whether a deletion awaits its answer in the page's question
// window.
func (v pageView) asking() bool {
	confirm := v.ui.confirm
	if v.save != nil {
		confirm = v.save.confirming()
	}
	return confirm && v.page.Confirm.windowed()
}

// confirmingInPage reports whether a deletion awaits confirmation in the
// page's own body.
func (v pageView) confirmingInPage() bool {
	confirm := v.ui.confirm
	if v.save != nil {
		confirm = v.save.confirming()
	}
	return confirm && !v.page.Confirm.windowed()
}

// listFocus reports whether the list has the focus: always on a load page,
// on a save page once a list key or a click in the list set it.
func (v pageView) listFocus() bool { return v.save == nil || v.save.listFocus }

// first is the list row of saves[0].
func (v pageView) first() int {
	if v.save != nil {
		return v.save.first()
	}
	return 0
}

// selected is the save behind the selected row.
func (v pageView) selected() (SaveEntry, int, bool) {
	if v.list == nil {
		return SaveEntry{}, 0, false
	}
	i := v.list.Selection() - v.first()
	if i < 0 || i >= len(v.saves) {
		return SaveEntry{}, 0, false
	}
	return v.saves[i], i, true
}

func (f *flow) loadView() pageView {
	return pageView{page: f.loadPage(), ui: &f.loadUI, list: f.loadList, saves: f.saves}
}

// refreshMenuSave asks the front end again whether a save file exists.
func (f *flow) refreshMenuSave() {
	f.pages.hasSave = f.pages.files != nil && f.pages.files()
}
