package ui

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// newPageMessageHost keeps its messages as pages and has no message window.
type newPageMessageHost struct {
	sectionPagesHost
	open  func(string) (*InspectionDocument, error)
	owner *InspectionMessageOwner
}

func (h *newPageMessageHost) NewMessage(owner InspectionMessageOwner) (*InspectionDocument, error) {
	h.owner = &owner
	d, err := h.open("scenario/10.alm")
	if err == nil {
		d.Script.Targets[ScriptMessageParameter] = append(d.Script.Targets[ScriptMessageParameter], InspectionScriptTarget{Value: 2, Record: -1, Page: "event2"})
	}
	return d, err
}

// A new message on a host with no message window is made by the host and
// opens the pages prompt at its page, at its text.
func TestNewMessageWithoutMessageWindowOpensItsPage(t *testing.T) {
	h := &newPageMessageHost{sectionPagesHost: sectionPagesHost{page: "briefing"}}
	h.open = func(source string) (*InspectionDocument, error) {
		v, err := NewViewer("editor", terrain.Grid{Width: 32, Height: 32, Tiles: make([]uint16, 32*32)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		script := &InspectionScriptModel{Targets: map[uint32][]InspectionScriptTarget{ScriptMessageParameter: {
			{Value: 5, Record: -1, Page: "event5"}, {Record: -1, NewChoice: NewMessageChoice},
		}}}
		return &InspectionDocument{Viewer: v, Source: source, Width: 32, Height: 32, SourceBytes: []byte("map"), Edits: h, Script: script}, nil
	}
	e := NewMapEditor(panelFont(), nil, h.open)
	e.Layout(1920, 1080)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.newPageMessage(h, InspectionMessageOwner{Kind: KindAction, Key: "value0"})
	if h.owner == nil || h.owner.Kind != KindAction {
		t.Fatalf("the host made no message: %+v", h.owner)
	}
	if e.prompt.kind != promptWorkspace || e.prompt.scope != "pages" || e.prompt.field != 1 || e.prompt.path != "Body of event2" {
		t.Fatalf("prompt %+v", e.prompt)
	}
}
