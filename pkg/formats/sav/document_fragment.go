package sav

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

// DocumentFragment is one detached actor and its ordinary child records. Its
// indices are local, and do not declare a Player, World or dead-manager root.
type DocumentFragment struct {
	Actor   uint16
	Objects []DocumentRecordData
}

// Record text uses bytes in JSON because installed names need not be UTF-8.
type fragmentText struct {
	Name  string
	Value []byte
}

type fragmentRecord struct {
	DocumentRecordData
	Texts  []fragmentText
	Inline []fragmentInline
	Groups []fragmentRecord
}

type fragmentInline struct {
	Name   string
	Record fragmentRecord
}

func fragmentRecordJSON(r DocumentRecordData) fragmentRecord {
	v := fragmentRecord{DocumentRecordData: r}
	v.DocumentRecordData.Texts, v.DocumentRecordData.Inline, v.DocumentRecordData.Groups = nil, nil, nil
	for _, text := range r.Texts {
		v.Texts = append(v.Texts, fragmentText{Name: text.Name, Value: []byte(text.Value)})
	}
	for _, child := range r.Inline {
		v.Inline = append(v.Inline, fragmentInline{Name: child.Name, Record: fragmentRecordJSON(child.Record)})
	}
	for _, child := range r.Groups {
		v.Groups = append(v.Groups, fragmentRecordJSON(child))
	}
	return v
}

func (v fragmentRecord) record() DocumentRecordData {
	r := v.DocumentRecordData
	for _, text := range v.Texts {
		r.Texts = append(r.Texts, DocumentTextData{Name: text.Name, Value: string(text.Value)})
	}
	for _, child := range v.Inline {
		r.Inline = append(r.Inline, DocumentInlineData{Name: child.Name, Record: child.Record.record()})
	}
	for _, child := range v.Groups {
		r.Groups = append(r.Groups, child.record())
	}
	return r
}

func (f DocumentFragment) MarshalJSON() ([]byte, error) {
	if err := (&documentDataBudget{}).check(reflect.ValueOf(f), 0); err != nil {
		return nil, err
	}
	v := struct {
		Actor   uint16
		Objects []fragmentRecord
	}{Actor: f.Actor}
	for _, r := range f.Objects {
		v.Objects = append(v.Objects, fragmentRecordJSON(r))
	}
	return json.Marshal(v)
}

func (f *DocumentFragment) UnmarshalJSON(raw []byte) error {
	if len(raw) > MaxNativeActions {
		return fmt.Errorf("sav: fragment JSON exceeds bound")
	}
	var v struct {
		Actor   uint16
		Objects []fragmentRecord
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&v); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("sav: fragment JSON has trailing data")
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(v), 0); err != nil {
		return err
	}
	next := DocumentFragment{Actor: v.Actor}
	for _, r := range v.Objects {
		next.Objects = append(next.Objects, r.record())
	}
	*f = next
	return nil
}

func documentFragment(data DocumentFragment, strict bool) (DocumentFragment, []uint16, error) {
	if data.Actor == 0 || int(data.Actor) > len(data.Objects) || len(data.Objects) > maxDocumentDataObjects {
		return DocumentFragment{}, nil, fmt.Errorf("sav: invalid fragment actor or object count")
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(data), 0); err != nil {
		return DocumentFragment{}, nil, err
	}
	objects := make([]*Record, len(data.Objects))
	for i, r := range data.Objects {
		actor := r.Class == "Unit" || r.Class == "Human" || r.Class == "Humanoid"
		if actor != (uint16(i+1) == data.Actor) || !actor && r.Class != "Item" && r.Class != "Weapon" && r.Class != "Armor" && r.Class != "Shield" && r.Class != "Effect" && r.Class != "Spell" && r.Class != "Diary" {
			return DocumentFragment{}, nil, fmt.Errorf("sav: fragment object %d has invalid class %s", i+1, r.Class)
		}
		objects[i] = newRecord(r.Class, 0, uint16(i+1))
	}
	for i, r := range data.Objects {
		if err := documentRecordFromData(r, objects[i], objects, false, 0); err != nil {
			return DocumentFragment{}, nil, fmt.Errorf("sav: fragment object %d: %w", i+1, err)
		}
		for _, slot := range r.RefSlots {
			for _, index := range slot.Objects {
				if index == 0 {
					continue
				}
				class := data.Objects[index-1].Class
				valid := false
				switch slot.Name {
				case "HeldWeapon", "HeldShield", "Worn", "Inventory":
					valid = class == "Item" || class == "Weapon" || class == "Armor" || class == "Shield"
				case "Effects":
					valid = class == "Effect"
				case "Spells", "WeaponSpell":
					valid = class == "Spell"
				case "Diary":
					valid = class == "Diary"
				}
				if !valid {
					return DocumentFragment{}, nil, fmt.Errorf("sav: fragment %s reference has invalid class %s", slot.Name, class)
				}
			}
		}
	}
	b := documentDataBuilder{ids: map[*Record]uint16{}, inline: map[*Record]bool{}}
	actor, err := b.ref(objects[data.Actor-1], 0)
	if err != nil {
		return DocumentFragment{}, nil, err
	}
	if len(b.objects) != len(objects) {
		return DocumentFragment{}, nil, fmt.Errorf("sav: fragment has unreachable objects")
	}
	permutation := make([]uint16, len(objects)+1)
	for i, r := range objects {
		permutation[i+1] = b.ids[r]
		if strict && permutation[i+1] != uint16(i+1) {
			return DocumentFragment{}, nil, fmt.Errorf("sav: fragment indices are not in first-encounter order")
		}
	}
	return DocumentFragment{Actor: actor, Objects: b.objects}, permutation, nil
}

func CloneDocumentFragment(data DocumentFragment) (DocumentFragment, error) {
	out, _, err := documentFragment(data, true)
	return out, err
}

func ReindexDocumentFragment(data DocumentFragment) (DocumentFragment, []uint16, error) {
	return documentFragment(data, false)
}

// DocumentActorFromCityUnit reuses the ordinary city writer and archive reader
// for one unbound actor. Holdings are attached by the shared graph constructor.
func DocumentActorFromCityUnit(class string, data CityUnitData) (DocumentRecordData, error) {
	if class != "Unit" && class != "Human" && class != "Humanoid" {
		return DocumentRecordData{}, fmt.Errorf("sav: invalid standalone actor class")
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(data), 0); err != nil {
		return DocumentRecordData{}, err
	}
	bound := false
	u := cityFromDataUnit(data, func(id uint16) *cityObject {
		bound = bound || id != 0
		return nil
	})
	if bound {
		return DocumentRecordData{}, fmt.Errorf("sav: standalone actor already has child bindings")
	}
	w := newCityArchiveWriter(nil)
	if err := w.reference(&cityObject{class: class, unit: &u}); err != nil {
		return DocumentRecordData{}, err
	}
	reader := walker{b: w.b, next: 1, classes: map[uint16]string{}, objects: map[uint16]*Record{}, retainGraph: true}
	r, err := reader.object(0)
	if err != nil {
		return DocumentRecordData{}, err
	}
	if reader.p != len(w.b) {
		return DocumentRecordData{}, fmt.Errorf("sav: standalone actor has unread bytes")
	}
	b := documentDataBuilder{ids: map[*Record]uint16{}, inline: map[*Record]bool{}}
	if _, err := b.ref(r, 0); err != nil {
		return DocumentRecordData{}, err
	}
	return b.objects[0], nil
}
