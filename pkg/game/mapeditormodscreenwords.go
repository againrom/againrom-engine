package game

// The words of the Screens page's choices, in the order of the choice values.
var (
	screenActionWords = []string{"editor.screens.action.abandon", "editor.screens.action.restart"}
	screenKindWords   = []string{"editor.screens.kind.info", "editor.screens.kind.list", "editor.screens.kind.table"}
	screenPlaceWords  = []string{"editor.screens.place.main", "editor.screens.place.game", "editor.screens.place.both"}
)

// screenNewWords is the text of each new text a screen is given, by the
// place the text is for.
var screenNewWords = map[string]string{
	"title":  "editor.screens.new.title",
	"menu":   "editor.screens.new.menu",
	"text":   "editor.screens.new.text",
	"label":  "editor.screens.new.label",
	"value":  "editor.screens.new.value",
	"action": "editor.screens.new.action",
}
