package game

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func defaultPackageManifest() mod.Manifest {
	return mod.Manifest{Version: "1.0.0", Author: "Againrom", Description: "Map editor mission package", API: 1, AppliesTo: []string{"rom1"}}
}

func (p editorPackageSpec) manifestValues() map[string]any {
	m := p.Manifest
	if m.Version == "" {
		m = defaultPackageManifest()
	}
	return map[string]any{"id": p.ID, "title": p.Title, "version": m.Version, "author": m.Author, "description": m.Description, "api": 1, "applies-to": m.AppliesTo, "requires": m.Requires, "conflicts": m.Conflicts, "load-after": m.LoadAfter}
}

func (p editorPackageSpec) manifestBytes(source []byte) ([]byte, error) {
	values := p.manifestValues()
	if len(source) != 0 {
		m, err := mod.ParseManifest(source)
		if err != nil {
			return nil, err
		}
		before := editorPackageSpec{ID: m.ID, Title: m.Title, Manifest: m}.manifestValues()
		before["api"] = m.API
		for key, value := range values {
			if reflect.DeepEqual(value, before[key]) {
				delete(values, key)
			}
		}
	} else {
		for _, key := range []string{"requires", "conflicts", "load-after"} {
			if len(values[key].([]string)) == 0 {
				delete(values, key)
			}
		}
	}
	out, err := mod.WriteManifest(source, values)
	if err == nil {
		_, err = mod.ParseManifest(out)
	}
	return out, err
}

func isManifestField(key string) bool {
	switch key {
	case "id", "title", "version", "author", "description", "applies-to", "requires", "conflicts", "load-after":
		return true
	}
	return false
}

var manifestBases = []string{"common", base.ROM1, base.ROM1EN, base.ROM1RU, base.ROM2EN, base.ROM2RU}

func (s *mapInspectionEdits) manifestFields() []ui.InspectionField {
	p := s.workspace.Package
	values := p.manifestValues()
	var fields []ui.InspectionField
	for _, key := range []string{"version", "author", "description", "applies-to", "requires", "conflicts", "load-after"} {
		value, ok := values[key].(string)
		if !ok {
			value = strings.Join(values[key].([]string), ", ")
		}
		f := workspaceText(key, s.inspector.modFieldLabel(key), value)
		if key == "applies-to" {
			f.Text, f.Value = false, -1
			for n, base := range manifestBases {
				f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(n), Label: base})
				if base == value {
					f.Value = int64(n)
				}
			}
			if f.Value == -1 {
				f.Choices = append(f.Choices, ui.InspectionChoice{Value: -1, Label: value})
			}
		}
		fields = append(fields, f)
	}
	return fields
}

func (s *mapInspectionEdits) setManifestField(key, text string) (*ui.InspectionDocument, error) {
	if key == "applies-to" && text == "-1" {
		return nil, nil
	}
	return s.changeWorkspace(func(w *editorWorkspaceState) error {
		if key == "id" && s.inspector.mod != nil && text != w.Package.ID {
			return fmt.Errorf("an opened package keeps its ID; create another package in a new folder")
		}
		source := []byte(nil)
		if w.Project != nil {
			source = w.Project.Manifest
		}
		source, err := w.Package.manifestBytes(source)
		if err != nil {
			return err
		}
		var value any = text
		switch key {
		case "applies-to":
			if n, err := strconv.Atoi(text); err == nil && n >= 0 && n < len(manifestBases) {
				text = manifestBases[n]
			}
			if !slices.Contains(manifestBases, text) {
				return fmt.Errorf("choose an applies-to option")
			}
			value = []string{text}
		case "requires", "conflicts", "load-after":
			var list []string
			if strings.TrimSpace(text) != "" {
				for _, id := range strings.Split(text, ",") {
					list = append(list, strings.TrimSpace(id))
				}
			}
			value = list
		}
		out, err := mod.WriteManifest(source, map[string]any{key: value})
		if err != nil {
			return err
		}
		m, err := mod.ParseManifest(out, true)
		if err != nil {
			return err
		}
		if key == "id" && (m.ID == TestPlayModID || !editorOrdinaryFilename(m.ID)) {
			return fmt.Errorf("use an ordinary mod ID other than %s", TestPlayModID)
		}
		w.Package.ID, w.Package.Title, w.Package.Manifest = m.ID, m.Title, m
		return nil
	})
}
