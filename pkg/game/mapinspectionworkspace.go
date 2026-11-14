package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

type editorPackageSpec struct {
	ID, Title, Kind                     string
	Manifest                            mod.Manifest
	Number, After, Guest, Chapter, Gold int
	Party                               []mod.GuestMember
	AutoGetMission                      *int `json:",omitempty"`
	MainParty                           bool `json:",omitempty"`
	// Class and Sex are the guest row's hero words, empty when the row
	// names none.
	Class, Sex string `json:",omitempty"`
	// Kept is the route held under the other kind of the kind choice.
	Kept *editorRouteSpec `json:",omitempty"`
}

type editorRouteSpec struct {
	After, Guest, Chapter, Gold int
	Party                       []mod.GuestMember
	AutoGetMission              *int
	MainParty                   bool
	Class, Sex                  string
}

func (p *editorPackageSpec) swapRoute() {
	party := func(g []mod.GuestMember) []mod.GuestMember { return mod.CloneMission(mod.Mission{Party: g}).Party }
	auto := func(v *int) *int {
		if v == nil {
			return nil
		}
		n := *v
		return &n
	}
	kept := &editorRouteSpec{After: p.After, Guest: p.Guest, Chapter: p.Chapter, Gold: p.Gold, Party: party(p.Party),
		AutoGetMission: auto(p.AutoGetMission), MainParty: p.MainParty, Class: p.Class, Sex: p.Sex}
	next := editorRouteSpec{}
	if p.Kept != nil {
		next = *p.Kept
	}
	p.After, p.Guest, p.Chapter, p.Gold = next.After, next.Guest, next.Chapter, next.Gold
	p.Party, p.AutoGetMission, p.MainParty = party(next.Party), auto(next.AutoGetMission), next.MainParty
	p.Class, p.Sex, p.Kept = next.Class, next.Sex, kept
}

type editorWorkspaceState struct {
	Pages   map[string][]byte
	Package editorPackageSpec
	Project *editorModDocument
	// Labels are the mapmaker's names of variable registers, UTF-8; the
	// labels file beside the map holds them.
	Labels map[uint32]string `json:",omitempty"`
}

type editorWorkspaceSnapshot struct {
	workspace *editorWorkspaceState
	model     *mapedit.Editor
	source    string
	// carried is the edits' carry count the step was recorded under.
	carried int
}

func cloneWorkspace(w *editorWorkspaceState) *editorWorkspaceState {
	if w == nil {
		return nil
	}
	next := *w
	next.Project = w.Project.clone()
	party := mod.CloneMission(mod.Mission{Party: w.Package.Party})
	next.Package.Party = party.Party
	if w.Package.AutoGetMission != nil {
		v := *w.Package.AutoGetMission
		next.Package.AutoGetMission = &v
	}
	next.Package.Manifest = mod.CloneManifest(w.Package.Manifest)
	next.Pages = make(map[string][]byte, len(w.Pages))
	for name, b := range w.Pages {
		next.Pages[name] = bytes.Clone(b)
	}
	next.Labels = maps.Clone(w.Labels)
	return &next
}

func workspaceBytes(w *editorWorkspaceState) []byte {
	if w == nil {
		return nil
	}
	b, _ := json.Marshal(w)
	return b
}

func decodeWorkspace(b []byte, w *editorWorkspaceState) { _ = json.Unmarshal(b, w) }

func (s *mapInspectionEdits) checkpointPages() {
	var saved editorWorkspaceState
	_ = json.Unmarshal(s.savedWorkspace, &saved)
	w := *s.workspace
	w.Package = saved.Package
	s.savedWorkspace = workspaceBytes(&w)
}

func (s *mapInspectionEdits) prepareWorkspace() error {
	x := s.inspector
	if s.workspace != nil {
		if s.workspace.Project == nil && x.mod != nil {
			s.workspace = cloneWorkspace(s.workspace)
			s.bindWorkspaceMod(s.workspace, s.source)
			var saved editorWorkspaceState
			decodeWorkspace(s.savedWorkspace, &saved)
			s.bindWorkspaceMod(&saved, s.source)
			s.savedWorkspace = workspaceBytes(&saved)
			s.syncModDocument(s.workspace, saved.Pages, saved.Labels, false)
			s.workspaceImage = workspaceBytes(s.workspace)
			for _, history := range [][]editorWorkspaceSnapshot{s.undoWorkspace, s.redoWorkspace} {
				for i := range history {
					if history[i].workspace != nil && history[i].workspace.Project == nil {
						history[i].workspace = cloneWorkspace(history[i].workspace)
						s.bindWorkspaceMod(history[i].workspace, history[i].source)
						s.syncModDocument(history[i].workspace, saved.Pages, saved.Labels, false)
					}
				}
			}
		}
		return nil
	}
	w := &editorWorkspaceState{Pages: map[string][]byte{}, Package: editorPackageSpec{ID: "authored-map", Title: "Authored map", Kind: mod.MissionGuest, Manifest: defaultPackageManifest()}}
	w.Package.Manifest.AppliesTo = []string{s.inspector.packageBase()}
	if s.model == nil {
		// No map is open: the workspace is the opened mod's alone.
		s.bindWorkspaceMod(w, "")
		w.Package.Manifest.ID, w.Package.Manifest.Title = w.Package.ID, w.Package.Title
		s.workspace, s.workspaceImage = w, workspaceBytes(w)
		s.savedWorkspace = s.workspaceImage
		return nil
	}
	if p := x.editorPages(s.source); p != nil {
		if p.Err != nil {
			return p.Err
		}
		for name, b := range p.Files() {
			if missionPageName(name) {
				w.Pages[name] = bytes.Clone(b)
			}
		}
	}
	s.bindWorkspaceMod(w, s.source)
	if w.Package.Number == 0 && x.archives != nil && x.archives.Containers != nil {
		if c, err := LoadCampaign(x.archives.Containers); err == nil {
			if m, err := testPlayGuest(c, x.archives.Containers); err == nil {
				w.Package.Number, w.Package.After, w.Package.Guest, w.Package.Chapter = m.Number, m.After, m.Guest, m.Chapter
			}
		}
	}
	if labels, err := x.mapLabels(s.source); err != nil {
		s.labelsUnread = true
		s.openNote = fmt.Sprintf(words.For(x.pageLanguage()).Text("editor.labels.unread"), err)
	} else if _, sc, err := wizardScript(s.model); err == nil {
		w.Labels = liveLabels(sc, labels)
	}
	w.Package.Manifest.ID, w.Package.Manifest.Title = w.Package.ID, w.Package.Title
	s.workspace, s.workspaceImage = w, workspaceBytes(w)
	s.savedWorkspace = s.workspaceImage
	return nil
}

func (s *mapInspectionEdits) bindWorkspaceMod(w *editorWorkspaceState, source string) {
	m := s.inspector.mod
	if m == nil || w.Project != nil {
		return
	}
	w.Project = m.document()
	w.Project.Selected = -1
	w.Package.ID, w.Package.Title = m.id, m.title
	w.Package.Manifest, _ = mod.ParseManifest(m.manifest)
	if s.model != nil {
		if entry, ok := m.entryFor(source); ok {
			w.Package = modPackageSpec(entry, w.Package)
			for i, mission := range w.Project.Missions {
				if mission.RowID() == entry.RowID() {
					w.Project.Selected = i
					break
				}
			}
		} else if rel, err := filepath.Rel(m.dir, source); err == nil && filepath.IsLocal(rel) && ui.ParseMapSource(source).Kind == ui.SourceHost {
			// A map of the mod no row names is held by its path.
			w.Project.Loose = filepath.ToSlash(rel)
		}
	}
}

// liveLabels is the labels whose registers are still variables: a register a
// node addresses, or one no check result takes, at or above the condition
// count; on an extended map any variable of the array. A label of any other
// register is dropped.
func liveLabels(sc alm.Script, labels map[uint32]string) map[uint32]string {
	addressed := variableRegisters(sc)
	var out map[uint32]string
	for r, label := range labels {
		if sc.Extended && r < scriptprofile.ExtendedVariables || r < scriptprofile.FirstRegisters && (addressed[r] || uint64(r) >= uint64(len(sc.Conditions))) {
			if out == nil {
				out = map[uint32]string{}
			}
			out[r] = label
		}
	}
	return out
}

// labelsForSave is what the save writes for the labels file: the live labels,
// or, with none, the file the open could not read left as it is.
func (s *mapInspectionEdits) labelsForSave() []byte {
	var labels map[uint32]string
	if _, sc, err := wizardScript(s.model); err == nil && s.workspace != nil {
		labels = liveLabels(sc, s.workspace.Labels)
	}
	if len(labels) == 0 && s.labelsUnread {
		return []byte{}
	}
	return encodeMapLabels(labels)
}

func (s *mapInspectionEdits) rememberWorkspace(mapEdit bool) {
	s.undoWorkspace = append(s.undoWorkspace, editorWorkspaceSnapshot{workspace: s.workspace, model: s.model, source: s.source, carried: s.carried})
	s.redoWorkspace = nil
	s.workspaceHistory = true
}

func (s *mapInspectionEdits) travelWorkspace(undo bool) (*ui.InspectionDocument, error) {
	from, to := &s.redoWorkspace, &s.undoWorkspace
	if undo {
		from, to = to, from
	}
	last := (*from)[len(*from)-1]
	if last.carried != s.carried {
		// A step that changes which map these edits hold comes back whole:
		// its own map's part is no part of the map held now.
		if sameSaveName(last.source, s.source) {
			last.workspace = s.carriedOver(last.workspace)
		} else {
			last.workspace = s.wholeStep(last.workspace)
		}
	}
	trial := *s
	trial.workspace, trial.model, trial.source = last.workspace, last.model, last.source
	oldMod := s.inspector.mod
	if !bytes.Equal(s.workspace.speakerDraft(), last.workspace.speakerDraft()) {
		if err := s.draftModContext(last.workspace); err != nil {
			return nil, err
		}
	}
	d, err := trial.project(last.model)
	if err != nil {
		s.inspector.setEditorMod(oldMod)
		return nil, err
	}
	*to = append(*to, editorWorkspaceSnapshot{workspace: s.workspace, model: s.model, source: s.source, carried: s.carried})
	*from = (*from)[:len(*from)-1]
	s.model, s.workspace, s.source = last.model, last.workspace, last.source
	if d != nil {
		d.Edits = s
	}
	s.refreshModSelection()
	s.workspaceImage = workspaceBytes(s.workspace)
	if d != nil && d.Script != nil {
		s.inspector.messageTargets(d.Script.Targets, s.workspace.Pages)
	}
	return d, nil
}

func (s *mapInspectionEdits) modelHistory(undo bool) (*ui.InspectionDocument, error) {
	next := s.model.Clone()
	changed := false
	if undo {
		changed = next.Undo()
	} else {
		changed = next.Redo()
	}
	if !changed {
		return nil, nil
	}
	d, err := s.project(next)
	if err == nil {
		s.model = next
	}
	return d, err
}

func (s *mapInspectionEdits) changeWorkspace(apply func(*editorWorkspaceState) error) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	next := cloneWorkspace(s.workspace)
	if err := apply(next); err != nil {
		return nil, err
	}
	s.syncModDocument(next, s.workspace.Pages, s.workspace.Labels, false)
	image := workspaceBytes(next)
	if bytes.Equal(image, s.workspaceImage) {
		return nil, nil
	}
	oldMod := s.inspector.mod
	if !bytes.Equal(s.workspace.speakerDraft(), next.speakerDraft()) {
		if err := s.draftModContext(next); err != nil {
			return nil, err
		}
	}
	old := s.workspace
	s.workspace = next
	d, err := s.project(s.model)
	s.workspace = old
	if err != nil {
		s.inspector.setEditorMod(oldMod)
		return nil, err
	}
	s.rememberWorkspace(false)
	s.workspace = next
	s.workspaceImage = image
	if d != nil && d.Script != nil {
		s.inspector.messageTargets(d.Script.Targets, s.workspace.Pages)
	}
	return d, nil
}

func workspaceText(key, label, value string) ui.InspectionField {
	return ui.InspectionField{Key: key, Label: label, Text: true, TextValue: value}
}

func (s *mapInspectionEdits) WorkspaceFields(scope string) []ui.InspectionField {
	if strings.HasPrefix(scope, "campaign.") {
		return s.campaignFields(scope)
	}
	if scope == "handout" {
		return s.handoutFields()
	}
	if s.model == nil && scope != "package" {
		return nil
	}
	if err := s.prepareWorkspace(); err != nil {
		return nil
	}
	switch scope {
	case "metadata":
		m, err := s.model.Map()
		if err != nil {
			return nil
		}
		out := []ui.InspectionField{workspaceText("name", "Map name (1251, 63 bytes)", alm.MapNameText.Decode([]byte(m.Name))), workspaceText("description", "Description (1251, 63 bytes)", m.Description)}
		for _, row := range []struct {
			key   string
			value uint32
		}{{"angle", math.Float32bits(m.Angle)}, {"0c", m.Meta.Word0C}, {"10", m.Meta.Word10}, {"14", m.Meta.Word14}, {"18", m.Meta.Bitmask}, {ui.ModeFieldKey, m.Meta.Word70}, {"74", m.Meta.Word74}} {
			if row.key == ui.ModeFieldKey {
				out = append(out, s.inspector.modeField(row.value))
				continue
			}
			out = append(out, ui.InspectionField{Key: row.key, Label: "Raw " + row.key + " bits (effect unknown)", Value: int64(row.value)})
		}
		return out
	case "pages":
		if s.page == "" {
			s.page = "title.txt"
		}
		value := installText(nil, editorPageSource.holding(s.workspace.Pages[s.page]), s.inspector.pageCode()).bytes()
		book := words.For(s.inspector.pageLanguage())
		return []ui.InspectionField{workspaceText("page", book.Text("editor.pages.file"), s.page), workspaceText("body", book.Text("editor.pages.body"), string(value))}
	case "package":
		p := s.workspace.Package
		// A mod with no folder yet shows the working folder proposed for it.
		dir := s.inspector.proposedModFolder(p.ID)
		if s.inspector.mod != nil {
			dir = s.inspector.mod.dir
		}
		out := []ui.InspectionField{workspaceText("folder", "Mod folder: Enter creates or opens", dir), workspaceText("id", "Mod ID (new folder's name)", p.ID), workspaceText("title", "Mod title", p.Title), workspaceText("kind", "Route: guest or replace", p.Kind)}
		for _, row := range []struct {
			key, label string
			value      int
		}{{"number", "Mission number", p.Number}, {"after", "Guest after won mission", p.After}, {"guest", "Guest companion", p.Guest}, {"chapter", "Guest town chapter", p.Chapter}, {"gold", "Guest gold", p.Gold}} {
			out = append(out, ui.InspectionField{Key: row.key, Label: row.label, Value: int64(row.value)})
		}
		if m := s.workspace.Project; m != nil {
			field := ui.InspectionField{Key: "source", Label: "Source mission / Up, Down"}
			for n, entry := range m.Missions {
				name := entry.RowID()
				if entry.Title != "" {
					name = entry.Title + " (" + entry.RowID() + ")"
				}
				if entry.Off {
					name = s.inspector.word("editor.mod.disabled")
				}
				if entry.Kind == mod.MissionReplace {
					name += " / " + s.inspector.word("editor.mod.replace")
				}
				field.Choices = append(field.Choices, ui.InspectionChoice{Value: int64(n + 1), Label: s.inspector.inspectionUTF8(fmt.Sprintf("%s: %d", name, entry.Number))})
				if n == m.Selected {
					field.Value = int64(n + 1)
				}
			}
			out = append(out, field)
		}
		out = append(out[:3], append(s.manifestFields(), out[3:]...)...)
		return s.modFields(out)
	}
	return nil
}

// opensMod reports whether a W field change opens another mod rather than
// writing: a folder named to open, or one that exists.
func opensMod(scope, key, value string) bool {
	if scope != "package" {
		return false
	}
	if key == "folder-open" {
		return true
	}
	_, err := os.Lstat(strings.TrimSpace(value))
	return key == "folder" && strings.TrimSpace(value) != "" && err == nil
}

func (s *mapInspectionEdits) SetWorkspaceField(scope, key, value string) (*ui.InspectionDocument, error) {
	if strings.HasPrefix(scope, "campaign.") {
		return s.setCampaignField(scope, key, value)
	}
	if scope == "handout" {
		return s.setHandoutField(key, value)
	}
	if s.model == nil && (scope != "package" || !isManifestField(key) && !campaignFieldKey(key) && key != "source" && key != "map" && key != "save" && key != "output" && key != "folder" && !isModAssetKey(key)) {
		if scope != "package" || !opensMod(scope, key, value) {
			return nil, errors.New(s.inspector.word("editor.mod.nomap"))
		}
		return s.openWorkspace(strings.TrimSpace(value))
	}
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if err := s.inspector.modReadOnly(); err != nil && !opensMod(scope, key, value) {
		return nil, err
	}
	if scope == "metadata" {
		return s.setMapMetadata(key, value)
	}
	if scope == "pages" {
		if key == "page" {
			name := strings.ToLower(strings.TrimSpace(value))
			if !missionPageName(name) {
				return nil, fmt.Errorf("use title.txt, briefing.txt, briefmap.txt, %s or a canonical numbered event/tips page (01, 02, ..., 100, ...)", unitNamesPage)
			}
			s.page = name
			return nil, nil
		}
		if key != "body" {
			return nil, fmt.Errorf("unknown page field %q", key)
		}
		return s.changeWorkspace(func(w *editorWorkspaceState) error {
			if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
				return fmt.Errorf("page text must be UTF-8 without NUL")
			}
			if len(value) > mod.MaxFileBytes {
				return fmt.Errorf("page exceeds %d bytes", mod.MaxFileBytes)
			}
			if _, exists := w.Pages[s.page]; !exists && len(w.Pages) >= mod.MaxMissionTexts {
				return fmt.Errorf("a mission holds at most %d pages", mod.MaxMissionTexts)
			}
			selector := 0
			if s.inspector.font != nil {
				selector = s.inspector.font.Selector
			}
			if r, dropped := ModTextDropped([]byte(value), selector); dropped {
				return fmt.Errorf("this install cannot display %U", r)
			}
			w.Pages[s.page] = []byte(value)
			return nil
		})
	}
	if scope != "package" {
		return nil, fmt.Errorf("unknown workspace %q", scope)
	}
	if key == "folder" {
		return s.openOrCreateWorkspace(strings.TrimSpace(value))
	}
	if key == "folder-open" {
		return s.openWorkspace(strings.TrimSpace(value))
	}
	if key == "source" {
		index, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		return s.openModMap(index, "")
	}
	if key == "map" {
		index, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		return s.openMapEntry(index)
	}
	if isManifestField(key) {
		return s.setManifestField(key, value)
	}
	if (s.workspace.Package.Kind == mod.MissionCampaign || s.workspace.Package.Kind == mod.MissionReplace) && campaignFieldKey(key) {
		return s.setCampaignField("campaign."+strconv.Itoa(s.workspace.Package.Number), key, value)
	}
	if next, handled, err := s.setModControl(key, value); handled {
		return next, err
	}
	return s.changeWorkspace(func(w *editorWorkspaceState) error {
		return s.editPackageField(w, key, value)
	})
}

func (s *mapInspectionEdits) editPackageField(w *editorWorkspaceState, key, value string) error {
	if err := s.fileMissionFixed(w); err != nil {
		return err
	}
	p := &w.Package
	switch key {
	case "AutoGetMission":
		if strings.TrimSpace(value) == "" {
			p.AutoGetMission = nil
		} else {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return errors.New(s.inspector.word("editor.graph.err.successor"))
			}
			p.AutoGetMission = &n
		}
	case "kind":
		if value == "0" {
			value = mod.MissionGuest
		}
		if value == "1" {
			value = mod.MissionReplace
		}
		if value != mod.MissionGuest && value != mod.MissionReplace {
			return errors.New(s.inspector.word("editor.mod.err.routekind"))
		}
		if value != p.Kind {
			p.Kind = value
			p.swapRoute()
		}
	case "class", "sex":
		word, ok := heroWord(key, value)
		if !ok {
			return fmt.Errorf(s.inspector.word("editor.mod.err.packagefield"), key)
		}
		if key == "class" {
			p.Class = word
		} else {
			p.Sex = word
		}
	default:
		n, err := strconv.ParseInt(strings.TrimSpace(value), 0, 32)
		maxValue, minValue := int64(65535), int64(1)
		if key == "gold" {
			maxValue, minValue = 1_000_000, 0
		}
		if err != nil || n < minValue || n > maxValue {
			return fmt.Errorf(s.inspector.word("editor.mod.err.range"), s.inspector.modFieldLabel(key), minValue, maxValue)
		}
		switch key {
		case "number":
			p.Number = int(n)
		case "after":
			p.After = int(n)
			if len(p.Party) == 0 {
				if town, ok := s.modDestination(w); ok {
					p.Chapter = town
				}
			}
		case "guest":
			p.Guest = int(n)
		case "chapter":
			p.Chapter = int(n)
		case "gold":
			p.Gold = int(n)
		default:
			return fmt.Errorf(s.inspector.word("editor.mod.err.packagefield"), key)
		}
	}
	return nil
}

func (s *mapInspectionEdits) setMapMetadata(key, value string) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		before := m.Bytes()
		var err error
		switch key {
		case "name":
			var encoded []byte
			encoded, err = alm.MapNameText.Encode(value)
			if err == nil {
				err = m.SetEncodedName(encoded)
			}
		case "description":
			err = m.SetDescription(value)
		default:
			v, parseErr := strconv.ParseUint(strings.TrimSpace(value), 0, 32)
			if parseErr != nil {
				return false, fmt.Errorf("raw metadata uses a u32, decimal or 0x hexadecimal: %w", parseErr)
			}
			switch key {
			case "angle":
				err = m.SetAngle(math.Float32frombits(uint32(v)))
			case "0c":
				err = m.SetWord0C(uint32(v))
			case "10":
				err = m.SetWord10(uint32(v))
			case "14":
				err = m.SetWord14(uint32(v))
			case "18":
				err = m.SetBitmask(uint32(v))
			case ui.ModeFieldKey:
				err = m.SetWord70(uint32(v))
			case "74":
				err = m.SetWord74(uint32(v))
			default:
				return false, fmt.Errorf("unknown map metadata %q", key)
			}
		}
		return err == nil && !bytes.Equal(before, m.Bytes()), err
	})
}

func (s *mapInspectionEdits) currentPages() *mod.MissionPages {
	p := s.inspector.editorPages(s.source)
	if s.workspace == nil || p.Err != nil {
		return p
	}
	var saved editorWorkspaceState
	_ = json.Unmarshal(s.savedWorkspace, &saved)
	for name, b := range s.workspace.Pages {
		if before, exists := saved.Pages[name]; !exists || !bytes.Equal(before, b) {
			p.Put(name, b, s.inspector.pageLanguage())
		}
	}
	m := s.workspace.Package.mission(s.model.Bytes(), p.Files(), s.place())
	m.Mod = p.Entry.Mod
	p.Entry = m
	return p
}

// mission is the package's mission row for map b and its pages, the map at
// place; the pages lie beside it, so the row names no texts folder.
func (p editorPackageSpec) mission(b []byte, pages map[string][]byte, place mod.MissionPlace) mod.Mission {
	m := mod.Mission{Mod: p.ID, File: mod.MissionsFile, Kind: p.Kind, Key: "map", Number: p.Number, Map: place.Map, MapBytes: bytes.Clone(b), TextFiles: pages}
	if p.Kind != mod.MissionReplace {
		m.After, m.Guest, m.Chapter, m.Gold, m.Carry = p.After, p.Guest, p.Chapter, p.Gold, mod.CarryJoin
		m.Class, m.Sex = p.Class, p.Sex
		m.Party = mod.CloneMission(mod.Mission{Party: p.Party}).Party
		if len(m.Party) != 0 {
			m.Guest, m.Chapter, m.Carry = 0, 0, ""
		}
	}
	m.AutoGetMission, m.MainParty = p.AutoGetMission, p.MainParty
	return m
}

// missionPlace is where the map from source goes in a mod: the place rule
// under the stem of the source map's file name, a new map's stem "map", with
// .almx when the map holds the extended record.
func missionPlace(source string, extended bool) mod.MissionPlace {
	if ui.ParseMapSource(source).Kind == ui.SourceNew {
		source = ""
	}
	return mod.PlaceMission(mod.MissionStem(source), extended)
}

// place is missionPlace of the open map.
func (s *mapInspectionEdits) place() mod.MissionPlace {
	return missionPlace(s.source, s.model != nil && s.model.Extended())
}

// packageProtected is what test play and Save mod package must not overlap.
// The digest, pack and package save walk the selected mod folder.
func (s *mapInspectionEdits) packageProtected() []packageGuard {
	var out []packageGuard
	if input, ok := mapHostPath(ui.ParseMapSource(s.source), s.inspector.archives); ok && s.source != "" {
		out = append(out, packageGuard{dir: filepath.Dir(input), name: "the opened map's folder"})
	}
	if s.inspector.mod != nil {
		out = append(out, packageGuard{dir: s.inspector.mod.dir, name: "the selected mod folder", walked: true})
	}
	return out
}

func (w *editorWorkspaceState) speakerDraft() []byte {
	if w == nil || w.Project == nil {
		return nil
	}
	return w.Project.Files[mod.SpeakersFile]
}
