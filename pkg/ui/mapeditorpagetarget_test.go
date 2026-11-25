package ui

import (
	"slices"
	"testing"

	"againrom/pkg/render/terrain"
)

// sectionPagesHost is a host whose pages scope is a page key and the page's text.
type sectionPagesHost struct {
	mapPanelEdits
	page string
	sets []string
}

func (h *sectionPagesHost) WorkspaceFields(scope string) []InspectionField {
	if scope != "pages" {
		return nil
	}
	return []InspectionField{{Key: "page", Label: "Section", Text: true, TextValue: h.page}, {Key: "body", Label: "Text", Text: true, TextValue: "Body of " + h.page}}
}

func (h *sectionPagesHost) SetWorkspaceField(scope, key, value string) (*InspectionDocument, error) {
	h.sets = append(h.sets, scope+" "+key+" "+value)
	if key == "page" {
		h.page = value
	}
	return nil, nil
}

// A target that names a page opens the pages prompt on that page, at its
// text, as a multi-line field: the second game's event section opens from
// the action that sends the event.
func TestPageTargetOpensThePagesPromptAtItsText(t *testing.T) {
	h := &sectionPagesHost{page: "briefing"}
	open := func(source string) (*InspectionDocument, error) {
		v, err := NewViewer("editor", terrain.Grid{Width: 32, Height: 32, Tiles: make([]uint16, 32*32)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		return &InspectionDocument{Viewer: v, Source: source, Width: 32, Height: 32, SourceBytes: []byte("map"), Edits: h}, nil
	}
	e := NewMapEditor(panelFont(), nil, open)
	e.Layout(1920, 1080)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.openPage("event5")
	if want := []string{"pages page event5"}; !slices.Equal(h.sets, want) {
		t.Fatalf("host writes %q, want %q", h.sets, want)
	}
	if e.prompt.kind != promptWorkspace || e.prompt.scope != "pages" || e.prompt.field != 1 || e.prompt.path != "Body of event5" || !e.multilinePrompt() {
		t.Fatalf("prompt %+v", e.prompt)
	}
}
