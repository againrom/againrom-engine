package ui

import (
	"image"
	"image/color"
	"strings"
)

// SavePage is one load or save page as data: its layout, captions, list
// rules and the save role's slot rule. The first game's LOAD window is the
// page firstLoadPage builds; another game supplies its pages through
// SetSavePages. One builder draws and drives every page.
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
	// OKNeedsSave disables Ok while no save is listed.
	OKNeedsSave bool

	// The save role (SavePages.Save). Row 0 is EmptySlot, the other rows
	// the listed saves except the Hidden names. NameBox holds the typed
	// label, at most NameLimit characters when positive. Ok writes it to the
	// selected row's file, or to the file Slot names for row 0, with no
	// overwrite confirmation.
	NameBox   image.Rectangle
	NameLimit int
	EmptySlot string
	Hidden    []string
	Slot      func(directory string, listed []string) string
}

// PageText is one caption: centred in Box when Box is set, else drawn from
// At, in Ink or the window text colour when Ink is zero. Listing hides it
// while a deletion awaits confirmation.
type PageText struct {
	Text    string
	At      image.Point
	Box     image.Rectangle
	Ink     color.RGBA
	Listing bool
}

// PageButton is one push button.
type PageButton struct {
	Rect    image.Rectangle
	Caption string
}

// PageConfirm is the delete confirmation: Before at At and the selected
// label in Label. Inline instead draws Before, the label and After as one
// text wrapped to Label's width from At. A '#' in either text is a space.
type PageConfirm struct {
	Before string
	At     image.Point
	Label  image.Rectangle
	After  string
	Inline bool
}

// pageSeams holds a game's pages: loadSet and saveSet nil keep the first
// game's LOAD window and the save dialog; save is the open save page.
// files answers whether any save file exists; hasSave is its answer when the
// menu last showed.
type pageSeams struct {
	loadSet, saveSet *SavePage
	// menu is the game's game-menu data.
	menu    GameMenuProfile
	save    *savePageState
	files   func() bool
	hasSave bool
	// loadable answers whether the load list has a row without building it.
	loadable func() bool
}

// SavePages are a game's load page and, when set, its save page. A nil Save
// keeps the save dialog.
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
	p := SavePage{
		Panel: loadPanel,
		Texts: []PageText{
			{Text: w.Title, Box: image.Rect(128, 88, 512, 116), Ink: townShellText},
			{Text: w.Subtitle, At: image.Pt(125, 130), Ink: loadSelectedText, Listing: true},
		},
		List:    loadListArg,
		Rows:    loadVisibleRows,
		Message: loadMessageBox,
		Confirm: PageConfirm{Before: w.Confirm, At: image.Pt(125, 146), Label: image.Rect(122, 180, 520, 210)},
	}
	for i, caption := range []string{w.OK, w.Delete, w.Cancel} {
		p.Buttons[i] = PageButton{Rect: image.Rect(156+i*124, 380, 252+i*124, 404), Caption: caption}
	}
	return p
}

// loadPage is the page LOAD shows.
func (f *flow) loadPage() SavePage {
	if f.pages.loadSet != nil {
		return *f.pages.loadSet
	}
	return firstLoadPage(f.loadUI.words)
}

// hidden reports whether a save page leaves name out of its list.
func (p SavePage) hidden(name string) bool {
	for _, h := range p.Hidden {
		if strings.EqualFold(h, name) {
			return true
		}
	}
	return false
}

// pageView is one open page: its data, its pointer state, its list and the
// saves behind the list's rows. A save page's row 0 is its Empty Slot, so its
// save i is row i+1.
type pageView struct {
	page  SavePage
	ui    *loadWindow
	list  *Picker
	saves []SaveEntry
	save  *savePageState
}

// first is the list row of saves[0].
func (v pageView) first() int {
	if v.save != nil {
		return 1
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
