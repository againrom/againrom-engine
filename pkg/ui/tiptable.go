package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"strings"
)

// TipTable is one game's tip popups as data: every popup the engine shows is
// one row, read by the same code for every game. Art names the screen art
// manifest the panel draws with; Panels are the text styles rows may name.
type TipTable struct {
	Name   string          `json:"tips"`
	Art    string          `json:"art"`
	Panels []TipPanelStyle `json:"panels"`
	Rows   []TipRow        `json:"rows"`
	Cite   []string        `json:"cite"`
}

// TipPanelStyle is how a row's text draws: Font "" is the install's tip
// font, "text" the generator's text font, also in a town, and "install"
// the generator's install font; Inset (left, top, right, bottom) places the text inside the
// panel, zero for the list child's own inset (MENU-137).
type TipPanelStyle struct {
	Name  string   `json:"name"`
	Font  string   `json:"font"`
	Inset [4]int   `json:"inset"`
	Cite  []string `json:"cite"`
}

// TipText is one tip text: a whole archive file, or one section of it. A
// mission row's File is a pattern formatted with the mission and tip number.
type TipText struct {
	File    string   `json:"file"`
	Section string   `json:"section"`
	Cite    []string `json:"cite"`
}

// TipRow is one tip popup: the event that raises it (tipEvents), its text,
// its rectangle, its once-only rule and the pointer route that owns its
// presses. Room names a room-enter row's room and Town the town description
// it serves, empty for every town without a row of its own; Class names a
// detail-enter row's class, "fighter" or "mage". Second replaces a room's
// text once per activation when the host's condition holds; Fit moves the
// bottom edge up to the text's own height.
type TipRow struct {
	Name    string   `json:"name"`
	Event   string   `json:"event"`
	Town    string   `json:"town"`
	Room    string   `json:"room"`
	Class   string   `json:"class"`
	Text    TipText  `json:"text"`
	Second  TipText  `json:"second"`
	Rect    [4]int   `json:"rect"`
	Fit     bool     `json:"fit"`
	Panel   string   `json:"panel"`
	Once    string   `json:"once"`
	Pointer string   `json:"pointer"`
	Cite    []string `json:"cite"`
}

// TipOwner is the row a showing popup was built from and the pointer route
// that owns its presses: a held Close belongs to one row's popup.
type TipOwner struct {
	Row     string
	Pointer string
}

// Rectangle is the row's panel rectangle.
func (r *TipRow) Rectangle() image.Rectangle {
	return image.Rect(r.Rect[0], r.Rect[1], r.Rect[2], r.Rect[3])
}

// Owner is the row's pointer owner.
func (r *TipRow) Owner() TipOwner { return TipOwner{Row: r.Name, Pointer: r.Pointer} }

// tipEvents are the events a row may name, each with the once-only rule and
// the pointer route its host enacts: a room is built at every enter while
// TipsMode is set and Close hides it until the next ("entry"); a generator
// page builds its popup at its enter and steps it on a portrait, level or
// first skill click ("page-enter"); a mission dialogue's tips= tag shows one
// popup at a time, a new tip replacing it ("replace").
var tipEvents = map[string]struct{ once, pointer string }{
	"room-enter":        {"entry", "town"},
	"pre-create-enter":  {"page-enter", "generator"},
	"portrait-click":    {"page-enter", "generator"},
	"level-click":       {"page-enter", "generator"},
	"detail-enter":      {"page-enter", "generator"},
	"first-skill-click": {"page-enter", "generator"},
	"dialogue-close":    {"replace", "mission"},
}

// PreCreateTipEvents are the pre-create page's tip events in step order.
var PreCreateTipEvents = []string{"pre-create-enter", "portrait-click", "level-click"}

// DecodeTipTable reads one table strictly: an unknown field, trailing data
// or a failed check is an error naming the table.
func DecodeTipTable(data []byte) (*TipTable, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var t TipTable
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("tip table: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("tip table %s: trailing data", t.Name)
	}
	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("tip table %s: %w", t.Name, err)
	}
	return &t, nil
}

func (t *TipTable) validate() error {
	if t.Name == "" || t.Art == "" {
		return fmt.Errorf("no name or art")
	}
	panels := map[string]bool{}
	for _, p := range t.Panels {
		if p.Name == "" || panels[p.Name] || !oneOf(p.Font, "", "text", "install") {
			return fmt.Errorf("panel %q font %q", p.Name, p.Font)
		}
		for _, v := range p.Inset {
			if v < 0 {
				return fmt.Errorf("panel %s inset %v", p.Name, p.Inset)
			}
		}
		panels[p.Name] = true
	}
	names := map[string]bool{}
	keys := map[string]bool{}
	for i := range t.Rows {
		r := &t.Rows[i]
		ev, ok := tipEvents[r.Event]
		switch {
		case r.Name == "" || names[r.Name]:
			return fmt.Errorf("row %d: name %q is empty or named twice", i, r.Name)
		case !ok:
			return fmt.Errorf("row %s: event %q", r.Name, r.Event)
		case r.Once != ev.once || r.Pointer != ev.pointer:
			return fmt.Errorf("row %s: once %q pointer %q, the %s host enacts %q and %q", r.Name, r.Once, r.Pointer, r.Event, ev.once, ev.pointer)
		case (r.Event == "room-enter") != (r.Room != ""):
			return fmt.Errorf("row %s: room %q on event %s", r.Name, r.Room, r.Event)
		case r.Town != "" && r.Event != "room-enter":
			return fmt.Errorf("row %s: town %q on event %s", r.Name, r.Town, r.Event)
		case (r.Event == "detail-enter") != oneOf(r.Class, "fighter", "mage"):
			return fmt.Errorf("row %s: class %q on event %s", r.Name, r.Class, r.Event)
		case r.Second.File != "" && r.Event != "room-enter":
			return fmt.Errorf("row %s: a second text on event %s", r.Name, r.Event)
		case r.Fit && r.Event != "room-enter":
			return fmt.Errorf("row %s: fit on event %s", r.Name, r.Event)
		case r.Text.File == "":
			return fmt.Errorf("row %s: no text", r.Name)
		case r.Rectangle().Empty():
			return fmt.Errorf("row %s: rectangle %v is empty", r.Name, r.Rect)
		case r.Panel != "" && !panels[r.Panel]:
			return fmt.Errorf("row %s: panel %q", r.Name, r.Panel)
		}
		if r.Event == "dialogue-close" && (r.Text.Section != "" || strings.Contains(fmt.Sprintf(r.Text.File, 1, 1), "%!")) {
			return fmt.Errorf("row %s: mission text %q does not take a mission and a tip number", r.Name, r.Text.File)
		}
		if p, host := t.panel(r.Panel), tipEvents[r.Event].pointer; p.Font != "" && host != "generator" && (p.Font != "text" || host != "town") {
			return fmt.Errorf("row %s: panel font %q on a %s row", r.Name, p.Font, host)
		}
		key := r.Event + "/" + r.Town + "/" + r.Room + "/" + r.Class
		if keys[key] {
			return fmt.Errorf("row %s: a second row for %s", r.Name, key)
		}
		names[r.Name], keys[key] = true, true
	}
	return nil
}

func (t *TipTable) panel(name string) TipPanelStyle {
	if t != nil {
		for _, p := range t.Panels {
			if p.Name == name {
				return p
			}
		}
	}
	return TipPanelStyle{}
}

// Row is the row raised by event, for room or class where the event names
// one; nil when the table has none.
func (t *TipTable) Row(event, room, class string) *TipRow {
	if t == nil {
		return nil
	}
	for i := range t.Rows {
		if r := &t.Rows[i]; r.Event == event && r.Town == "" && r.Room == room && r.Class == class {
			return r
		}
	}
	return nil
}

// RoomRow is room's room-enter row in town, else its row for every town.
func (t *TipTable) RoomRow(town, room string) *TipRow {
	if t == nil {
		return nil
	}
	if town != "" {
		for i := range t.Rows {
			if r := &t.Rows[i]; r.Event == "room-enter" && r.Town == town && r.Room == room {
				return r
			}
		}
	}
	return t.Row("room-enter", room, "")
}

// Panel is the text style row names; the zero style when it names none.
func (t *TipTable) Panel(r *TipRow) TipPanelStyle {
	if r == nil {
		return TipPanelStyle{}
	}
	return t.panel(r.Panel)
}

// MissionTip is the archive path of tip n of a mission, from the table's
// dialogue-close row; false when the table has no mission tips.
func (t *TipTable) MissionTip(mission, n int) (string, bool) {
	r := t.Row("dialogue-close", "", "")
	if r == nil {
		return "", false
	}
	return fmt.Sprintf(r.Text.File, mission, n), true
}
