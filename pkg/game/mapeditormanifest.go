package game

import (
	"errors"
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
	return mod.Manifest{Version: "1.0.0", Author: "Againrom", Description: "Map editor mission package", API: 1, AppliesTo: []string{base.ROM1}}
}

// packageBase is a new package's applies-to: the install game's family.
func (x *MapInspector) packageBase() string {
	var p base.Profile
	if x.archives != nil {
		p = x.archives.Base.Profile
	}
	return base.ModFamilyOf(p.GameOf())
}

// manifestValues are the keys the editor writes, from mod.ManifestKeys.
func (p editorPackageSpec) manifestValues() map[string]any {
	m := p.Manifest
	if m.Version == "" {
		m = defaultPackageManifest()
	}
	values := map[string]any{}
	for _, k := range mod.ManifestKeys {
		if k.Edit != mod.EditNone {
			values[k.Name] = k.Get(m)
		}
	}
	values["id"], values["title"], values["api"] = p.ID, p.Title, 1
	return values
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
		for _, k := range mod.ManifestKeys {
			if list, ok := values[k.Name].([]string); ok && !k.Bases && len(list) == 0 {
				delete(values, k.Name)
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
	k, ok := mod.ManifestKeyNamed(key)
	return ok && k.Edit != mod.EditNone
}

// manifestBases are the applies-to choices, from base.ModBases.
func manifestBases() []string {
	names := make([]string, len(base.ModBases))
	for i, b := range base.ModBases {
		names[i] = b.Name
	}
	return names
}

func (s *mapInspectionEdits) manifestFields() []ui.InspectionField {
	p := s.workspace.Package
	values := p.manifestValues()
	bases := manifestBases()
	var fields []ui.InspectionField
	for _, k := range mod.ManifestKeys {
		if k.Edit != mod.EditField {
			continue
		}
		value, ok := values[k.Name].(string)
		if !ok {
			value = strings.Join(values[k.Name].([]string), ", ")
		}
		f := workspaceText(k.Name, s.inspector.modFieldLabel(k.Name), value)
		if k.Bases {
			f.Text, f.Value = false, -1
			for n, base := range bases {
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
	row, _ := mod.ManifestKeyNamed(key)
	if row.Bases && text == "-1" {
		return nil, nil
	}
	return s.changeWorkspace(func(w *editorWorkspaceState) error {
		if key == "id" && s.inspector.mod != nil && text != w.Package.ID {
			return errors.New(s.inspector.word("editor.mod.err.keepid"))
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
		switch {
		case row.Bases:
			bases := manifestBases()
			if n, err := strconv.Atoi(text); err == nil && n >= 0 && n < len(bases) {
				text = bases[n]
			}
			if !slices.Contains(bases, text) {
				return errors.New(s.inspector.word("editor.mod.err.appliesto"))
			}
			value = []string{text}
		case row.Value == mod.ManifestList:
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
			return fmt.Errorf(s.inspector.word("editor.mod.err.reservedid"), TestPlayModID)
		}
		w.Package.ID, w.Package.Title, w.Package.Manifest = m.ID, m.Title, m
		return nil
	})
}
