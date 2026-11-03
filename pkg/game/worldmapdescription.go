package game

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"

	"againrom/pkg/base"
	"againrom/pkg/screenart"
	"againrom/pkg/ui"
)

// The world-map description is one edition's campaign map as data: its art,
// where its records and their points come from, its modes, the rule that picks
// a record, how travel runs and where text comes from. The one world-map
// builder (worldMapState, ui.ComposeWorldMap) reads it; no game's map has code
// of its own. Every entry carries the claim, divergence row or owner ruling
// behind it.

//go:embed worldmaps/rom1.json
var rom1WorldMapJSON []byte

//go:embed worldmaps/rom2.json
var rom2WorldMapJSON []byte

// WorldMapDescription is one map. Art names are the manifest's entry names.
type WorldMapDescription struct {
	Name string             `json:"worldmap"`
	Art  []screenart.Entry  `json:"art"`
	Page WorldMapPagePolicy `json:"page"`
	// Records is where the records and their points come from: "registry"
	// reads the install's global map registry, "catalog" is Catalog.
	Records string                  `json:"records"`
	Catalog []WorldMapCatalogRecord `json:"catalog"`
	Modes   []WorldMapMode          `json:"modes"`
	// Pick is the rule that chooses a record: "scroll" (a scroll card, or
	// Up and Down, chooses and starts travel) or "nearest" (the pointer
	// targets the nearest record and a press starts travel).
	Pick string `json:"pick"`
	// Preview draws the whole route to the target before travel starts.
	Preview bool `json:"preview"`
	// SkipAtStart lets a skip finish a travel before its first reveal.
	SkipAtStart bool `json:"skip-at-start"`
	// Ball names the route dot; Cross the travel target sheet.
	Ball  string        `json:"ball"`
	Cross WorldMapPlace `json:"cross"`
	// CrossHold is how many paints past its sheet the Cross counter runs
	// before a travel may end.
	CrossHold    int  `json:"cross-hold"`
	CrossAtEnter bool `json:"cross-at-enter"`
	RoutePastEnd bool `json:"route-past-end"`
	RevealLead   int  `json:"reveal-lead"`
	// Task is the sheet drawn at the chosen record while the party stands at
	// the map's home point; empty for none.
	Task WorldMapPlace `json:"task"`
	// Cards names the scroll-card art the scroll pick draws its records on:
	// the three scroll pieces and the three pressed pieces.
	Cards *WorldMapCards `json:"cards"`
	// RecordArt names the bound entry a registry record's own picture is
	// read through; empty for none.
	RecordArt string               `json:"record-art"`
	Markers   []WorldMapMarkerRule `json:"markers"`
	Text      *WorldMapTextSpec    `json:"text"`
	Cite      []string             `json:"cite"`
}

// WorldMapPagePolicy supplies the host conditions, record sources and actions
// of one map opening. Location entries remain campaign operations.
type WorldMapPagePolicy struct {
	Open         string                `json:"open"`
	Build        string                `json:"build"`
	Mode         string                `json:"mode"`
	RouteMode    string                `json:"route-mode"`
	Point        string                `json:"point"`
	Records      []string              `json:"records"`
	Font         string                `json:"font"`
	HomeTitle    bool                  `json:"home-title"`
	Task         bool                  `json:"task"`
	History      bool                  `json:"history"`
	RecordText   bool                  `json:"record-text"`
	TravelOnOpen bool                  `json:"travel-on-open"`
	Leave        string                `json:"leave"`
	Arrival      WorldMapArrivalPolicy `json:"arrival"`
}

type WorldMapArrivalPolicy struct {
	ReturnHome   bool   `json:"return-home"`
	Snap         bool   `json:"snap"`
	ClearCross   bool   `json:"clear-cross"`
	KeepPosition bool   `json:"keep-position"`
	Town         string `json:"town"`
	Mission      string `json:"mission"`
}

// WorldMapCards names the scroll-card pieces.
type WorldMapCards struct {
	Scroll  string   `json:"scroll"`
	Pressed string   `json:"pressed"`
	Cite    []string `json:"cite"`
}

// WorldMapCatalogRecord is one record of a fixed catalog and its point.
type WorldMapCatalogRecord struct {
	Kind  int             `json:"kind"`
	ID    int             `json:"id"`
	Point screenart.Point `json:"point"`
}

// WorldMapMode is one way the map opens: its background, its route image,
// the party sprite and whether record markers and text show.
type WorldMapMode struct {
	Name       string        `json:"name"`
	Background string        `json:"background"`
	Path       string        `json:"path"`
	Party      WorldMapPlace `json:"party"`
	Markers    bool          `json:"markers"`
	Text       bool          `json:"text"`
	// Travel "first" opens the map travelling to its first record.
	Travel string   `json:"travel"`
	Cite   []string `json:"cite"`
}

// WorldMapPlace names a sheet and where it stands against its point: At is
// the offset of the sprite's top-left corner; no At centres it.
type WorldMapPlace struct {
	Art  string           `json:"art"`
	At   *screenart.Point `json:"at"`
	Cite []string         `json:"cite"`
}

func (p WorldMapPlace) place() ui.WorldMapPlace {
	if p.At == nil {
		return ui.WorldMapPlace{}
	}
	return ui.WorldMapPlace{Offset: image.Pt(p.At[0], p.At[1]), TopLeft: true}
}

// WorldMapMarkerRule draws Art at the records Who names while no travel
// runs: "target" is the targeted record, "other" every other record not at
// the party's point.
type WorldMapMarkerRule struct {
	Who  string          `json:"who"`
	Art  string          `json:"art"`
	At   screenart.Point `json:"at"`
	Cite []string        `json:"cite"`
}

// WorldMapTextSpec is where a record's name and description come from and
// where the text block draws them. Keys are formats over the record ID by
// record kind; a key that finds nothing gives no text.
type WorldMapTextSpec struct {
	Source      string            `json:"source"`
	Name        map[string]string `json:"name"`
	Description map[string]string `json:"description"`
	// Trim drops this many characters from the end of each text.
	Trim     int             `json:"trim"`
	NameAt   screenart.Point `json:"name-at"`
	LinesAt  screenart.Point `json:"lines-at"`
	LineStep int             `json:"line-step"`
	Width    int             `json:"width"`
	MaxLines int             `json:"max-lines"`
	Cite     []string        `json:"cite"`
}

// worldMapDescriptions are the decoded descriptions by the name an edition's
// WorldMap gives. One that does not decode is a build defect.
var worldMapDescriptions = map[string]*WorldMapDescription{
	"rom1": mustDecodeWorldMap(rom1WorldMapJSON),
	"rom2": mustDecodeWorldMap(rom2WorldMapJSON),
}

// WorldMapDescriptionFor is the world-map description the profile's edition
// names. Callers must not change it.
func WorldMapDescriptionFor(p base.Profile) *WorldMapDescription {
	return worldMapDescriptions[p.Edition().WorldMap]
}

func mustDecodeWorldMap(data []byte) *WorldMapDescription {
	d, err := decodeWorldMap(data)
	if err != nil {
		panic(err)
	}
	return d
}

func decodeWorldMap(data []byte) (*WorldMapDescription, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var d WorldMapDescription
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("world-map description: %w", err)
	}
	fail := func(format string, args ...any) (*WorldMapDescription, error) {
		return nil, fmt.Errorf("world-map description %s: "+format, append([]any{d.Name}, args...)...)
	}
	if err := screenart.Check(d.Art); err != nil {
		return fail("%w", err)
	}
	names := map[string]bool{}
	for _, e := range d.Art {
		names[e.Name] = true
	}
	art := func(name string) bool { return name == "" || names[name] }
	if d.Records != "registry" && d.Records != "catalog" {
		return fail("records %q is neither registry nor catalog", d.Records)
	}
	if d.Pick != "scroll" && d.Pick != "nearest" {
		return fail("pick %q is neither scroll nor nearest", d.Pick)
	}
	if len(d.Modes) == 0 {
		return fail("no mode")
	}
	for _, rule := range []struct {
		name, value string
		allowed     []string
	}{
		{"open", d.Page.Open, []string{"gates", "unlocated"}},
		{"build", d.Page.Build, []string{"entry", "opening"}},
		{"point", d.Page.Point, []string{"retained", "from-record"}},
		{"font", d.Page.Font, []string{"document", "install"}},
		{"leave", d.Page.Leave, []string{"square", "none"}},
		{"arrival town", d.Page.Arrival.Town, []string{"town-state", "mission-action", "location"}},
		{"arrival mission", d.Page.Arrival.Mission, []string{"town-state", "mission-action", "location"}},
	} {
		valid := false
		for _, allowed := range rule.allowed {
			valid = valid || rule.value == allowed
		}
		if !valid {
			return fail("page %s %q is unknown", rule.name, rule.value)
		}
	}
	if len(d.Page.Records) == 0 {
		return fail("page has no record source")
	}
	for _, source := range d.Page.Records {
		if source != "available-missions" && source != "travel-towns" && source != "destinations" {
			return fail("page record source %q is unknown", source)
		}
	}
	for i, name := range []string{d.Page.Mode, d.Page.RouteMode} {
		if i == 1 && name == "" || i == 0 && name == "first" {
			continue
		}
		found := false
		for _, m := range d.Modes {
			found = found || m.Name == name
		}
		if !found {
			return fail("page mode %q is unknown", name)
		}
	}
	if !art(d.Ball) || !art(d.Cross.Art) || !art(d.Task.Art) || !art(d.RecordArt) {
		return fail("ball, cross, task or record art names no art entry")
	}
	if c := d.Cards; c != nil && (!art(c.Scroll) || !art(c.Pressed)) {
		return fail("cards name no art entry")
	}
	if d.CrossHold < 0 || d.RevealLead < 0 {
		return fail("cross-hold or reveal-lead is negative")
	}
	for _, m := range d.Modes {
		if m.Name == "" || !art(m.Background) || !art(m.Path) || !art(m.Party.Art) {
			return fail("mode %q is unnamed or names no art entry", m.Name)
		}
		if m.Travel != "" && m.Travel != "first" {
			return fail("mode %s travel %q is not first", m.Name, m.Travel)
		}
	}
	for _, r := range d.Markers {
		if (r.Who != "target" && r.Who != "other") || !art(r.Art) {
			return fail("marker %q names no art entry or no rule", r.Who)
		}
	}
	seen := map[[2]int]bool{}
	for _, r := range d.Catalog {
		if r.ID <= 0 || seen[[2]int{r.Kind, r.ID}] {
			return fail("catalog record %d/%d is invalid or listed twice", r.Kind, r.ID)
		}
		seen[[2]int{r.Kind, r.ID}] = true
	}
	if t := d.Text; t != nil && (t.Source == "" || t.Trim < 0 || t.LineStep <= 0 || t.Width <= 0) {
		return fail("text needs a source, a line step and a width")
	}
	return &d, nil
}

// mode is the named mode, or the first one.
func (d *WorldMapDescription) mode(name string) WorldMapMode {
	for _, m := range d.Modes {
		if m.Name == name {
			return m
		}
	}
	return d.Modes[0]
}

// point is the catalog point of a record.
func (d *WorldMapDescription) point(kind, id int) (image.Point, bool) {
	for _, r := range d.Catalog {
		if r.Kind == kind && r.ID == id {
			return image.Pt(r.Point[0], r.Point[1]), true
		}
	}
	return image.Point{}, false
}
