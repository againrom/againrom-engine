package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

type editorPackageSpec struct {
	ID, Title, Kind                     string
	Number, After, Guest, Chapter, Gold int
}

type editorWorkspaceState struct {
	Pages   map[string][]byte
	Package editorPackageSpec
}

type editorWorkspaceSnapshot struct {
	mapEdit   bool
	workspace *editorWorkspaceState
}

func cloneWorkspace(w *editorWorkspaceState) *editorWorkspaceState {
	if w == nil {
		return nil
	}
	next := *w
	next.Pages = make(map[string][]byte, len(w.Pages))
	for name, b := range w.Pages {
		next.Pages[name] = bytes.Clone(b)
	}
	return &next
}

func workspaceBytes(w *editorWorkspaceState) []byte {
	if w == nil {
		return nil
	}
	b, _ := json.Marshal(w)
	return b
}

func (s *mapInspectionEdits) checkpointPages() {
	var saved editorWorkspaceState
	_ = json.Unmarshal(s.savedWorkspace, &saved)
	w := *s.workspace
	w.Package = saved.Package
	s.savedWorkspace = workspaceBytes(&w)
}

func (s *mapInspectionEdits) prepareWorkspace() error {
	if s.workspace != nil {
		return nil
	}
	w := &editorWorkspaceState{Pages: map[string][]byte{}, Package: editorPackageSpec{ID: "authored-map", Title: "Authored map", Kind: mod.MissionGuest}}
	x := s.inspector
	if p := x.editorPages(s.source); p != nil {
		if p.err != nil {
			return p.err
		}
		for name, b := range p.files {
			if missionPageName(name) {
				w.Pages[name] = bytes.Clone(b)
			}
		}
		if x.mod != nil {
			if m, ok := x.mod.entryFor(s.source); ok {
				w.Package = editorPackageSpec{ID: m.Mod, Title: x.mod.title, Kind: m.Kind, Number: m.Number, After: m.After, Guest: m.Guest, Chapter: m.Chapter, Gold: m.Gold}
			}
		}
	}
	if w.Package.Number == 0 && x.archives != nil && x.archives.Containers != nil {
		if c, err := LoadCampaign(x.archives.Containers); err == nil {
			if m, err := testPlayGuest(c, x.archives.Containers); err == nil {
				w.Package.Number, w.Package.After, w.Package.Guest, w.Package.Chapter = m.Number, m.After, m.Guest, m.Chapter
			}
		}
	}
	s.workspace, s.workspaceImage = w, workspaceBytes(w)
	s.savedWorkspace = s.workspaceImage
	return nil
}

func (s *mapInspectionEdits) rememberWorkspace(mapEdit bool) {
	s.undoWorkspace = append(s.undoWorkspace, editorWorkspaceSnapshot{mapEdit, s.workspace})
	s.redoWorkspace = nil
	s.workspaceHistory = true
}

func (s *mapInspectionEdits) travelWorkspace(undo bool) (*ui.InspectionDocument, error) {
	from, to := &s.redoWorkspace, &s.undoWorkspace
	if undo {
		from, to = to, from
	}
	last := (*from)[len(*from)-1]
	next := s.model
	if last.mapEdit {
		next = s.model.Clone()
		if undo {
			next.Undo()
		} else {
			next.Redo()
		}
	}
	d, err := s.project(next)
	if err != nil {
		return nil, err
	}
	*to = append(*to, editorWorkspaceSnapshot{last.mapEdit, s.workspace})
	*from = (*from)[:len(*from)-1]
	s.model, s.workspace = next, last.workspace
	s.workspaceImage = workspaceBytes(s.workspace)
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
	image := workspaceBytes(next)
	if bytes.Equal(image, s.workspaceImage) {
		return nil, nil
	}
	d, err := s.project(s.model)
	if err != nil {
		return nil, err
	}
	s.rememberWorkspace(false)
	s.workspace = next
	s.workspaceImage = image
	return d, nil
}

func workspaceText(key, label, value string) ui.InspectionField {
	return ui.InspectionField{Key: key, Label: label, Text: true, TextValue: value}
}

func (s *mapInspectionEdits) WorkspaceFields(scope string) []ui.InspectionField {
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
		value := s.workspace.Pages[s.page]
		if !utf8.Valid(value) {
			selector := 0
			if s.inspector.font != nil {
				selector = s.inspector.font.Selector
			}
			if decoded, ok := DecodeInstallTextFile(value, selector); ok {
				value = decoded
			}
		}
		book := words.For(s.inspector.pageLanguage())
		return []ui.InspectionField{workspaceText("page", book.Text("editor.pages.file"), s.page), workspaceText("body", book.Text("editor.pages.body"), string(value))}
	case "package":
		p := s.workspace.Package
		dir := ""
		if s.inspector.mod != nil && s.inspector.mod.owned {
			dir = s.inspector.mod.dir
		}
		out := []ui.InspectionField{workspaceText("folder", "Mod folder: Enter creates or opens", dir), workspaceText("id", "Mod ID (new folder's name)", p.ID), workspaceText("title", "Mod title", p.Title), workspaceText("kind", "Route: guest or replace", p.Kind)}
		for _, row := range []struct {
			key, label string
			value      int
		}{{"number", "Mission number", p.Number}, {"after", "Guest after won mission", p.After}, {"guest", "Guest companion", p.Guest}, {"chapter", "Guest town chapter", p.Chapter}, {"gold", "Guest gold", p.Gold}} {
			out = append(out, ui.InspectionField{Key: row.key, Label: row.label, Value: int64(row.value)})
		}
		if m := s.inspector.mod; m != nil {
			field := ui.InspectionField{Key: "source", Label: "Source mission / Up, Down", Value: 1}
			for n, entry := range m.missions.Missions {
				field.Choices = append(field.Choices, ui.InspectionChoice{Value: int64(n + 1), Label: fmt.Sprintf("%s: mission %d", entry.Key, entry.Number)})
				if entry.Key == m.selected {
					field.Value = int64(n + 1)
				}
			}
			out = append(out, field)
		}
		return out
	}
	return nil
}

func (s *mapInspectionEdits) SetWorkspaceField(scope, key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if scope == "metadata" {
		return s.setMapMetadata(key, value)
	}
	if scope == "pages" {
		if key == "page" {
			name := strings.ToLower(strings.TrimSpace(value))
			if !missionPageName(name) {
				return nil, fmt.Errorf("use title.txt, briefing.txt, briefmap.txt or a canonical numbered event/tips page (01, 02, ..., 100, ...)")
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
	if key == "source" {
		index, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		return s.selectWorkspaceMission(index)
	}
	return s.changeWorkspace(func(w *editorWorkspaceState) error {
		p := &w.Package
		switch key {
		case "id":
			if !mod.ValidID(value) || value == TestPlayModID || !editorOrdinaryFilename(value) {
				return fmt.Errorf("use an ordinary mod ID other than %s", TestPlayModID)
			}
			if s.inspector.mod != nil && s.inspector.mod.owned && s.inspector.mod.id != value {
				return fmt.Errorf("an opened package keeps its ID; create another package in a new folder")
			}
			p.ID = value
		case "title":
			if strings.TrimSpace(value) == "" || len(value) > 1024 || !utf8.ValidString(value) {
				return fmt.Errorf("mod title must be 1 to 1024 UTF-8 bytes")
			}
			p.Title = value
		case "kind":
			if value != mod.MissionGuest && value != mod.MissionReplace {
				return fmt.Errorf("route must be guest or replace")
			}
			p.Kind = value
		default:
			n, err := strconv.ParseInt(strings.TrimSpace(value), 0, 32)
			maxValue, minValue := int64(65535), int64(1)
			if key == "gold" {
				maxValue, minValue = 1_000_000, 0
			}
			if err != nil || n < minValue || n > maxValue {
				return fmt.Errorf("%s must be %d to %d", key, minValue, maxValue)
			}
			switch key {
			case "number":
				p.Number = int(n)
			case "after":
				p.After = int(n)
			case "guest":
				p.Guest = int(n)
			case "chapter":
				p.Chapter = int(n)
			case "gold":
				p.Gold = int(n)
			default:
				return fmt.Errorf("unknown package field %q", key)
			}
		}
		return nil
	})
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

func (s *mapInspectionEdits) currentPages() *modPages {
	p := s.inspector.editorPages(s.source)
	if s.workspace == nil || p.err != nil {
		return p
	}
	var saved editorWorkspaceState
	_ = json.Unmarshal(s.savedWorkspace, &saved)
	for name, b := range s.workspace.Pages {
		if before, exists := saved.Pages[name]; !exists || !bytes.Equal(before, b) {
			p.files[name] = b
		}
	}
	m := s.workspace.Package.mission(s.model.Bytes(), p.files)
	m.Mod = p.entry.Mod
	p.entry = m
	return p
}

func (p editorPackageSpec) mission(b []byte, pages map[string][]byte) mod.Mission {
	m := mod.Mission{Mod: p.ID, File: mod.MissionsFile, Kind: p.Kind, Key: "map", Number: p.Number, Map: testPlayMap, MapBytes: bytes.Clone(b), TextFiles: pages}
	if p.Kind != mod.MissionReplace {
		m.After, m.Guest, m.Chapter, m.Gold, m.Carry = p.After, p.Guest, p.Chapter, p.Gold, mod.CarryJoin
	}
	if len(pages) > 0 {
		m.Texts = testPlayTexts
	}
	return m
}

func (s *mapInspectionEdits) packageProtected() []string {
	var out []string
	if !isNewSource(s.source) && !strings.HasPrefix(s.source, "scenario/") {
		input := s.source
		if strings.HasPrefix(input, "loose/") && s.inspector.archives != nil {
			input = filepath.Join(s.inspector.archives.Root, filepath.FromSlash(strings.TrimPrefix(input, "loose/")))
		}
		out = append(out, filepath.Dir(input))
	}
	if s.inspector.mod != nil {
		out = append(out, s.inspector.mod.dir)
	}
	return slices.Compact(out)
}
