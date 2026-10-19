package sav

import (
	"bytes"
	"encoding/binary"
	"errors"
	"slices"
	"strings"
)

// Supplement leaves: Againrom state as registry dword arrays under
// /CurrentState (REG-099..102). This file alone finds, frames, stores and
// places one.
const (
	// NativeActionsPath: original application acceptance unknown (DIV-1369).
	NativeActionsPath = "/" + supplementDirectory + "/" + nativeActionsName
	NativeModsPath    = "/" + supplementDirectory + "/" + nativeModsName
	// NativeRandomPath: SAV-914..916 do not establish original preservation.
	NativeRandomPath  = "/" + supplementDirectory + "/" + nativeRandomName
	NativeSessionPath = "/" + supplementDirectory + "/" + nativeSessionName
)

const (
	supplementDirectory = "CurrentState"
	nativeActionsName   = "AgainromActions"
	nativeModsName      = "AgainromMods"
	nativeRandomName    = "AgainromRng"
	nativeSessionName   = "AgainromSeed"
)

type leafFraming uint8

const (
	// framedPayload: version 1, byte length, payload, zero dword padding.
	framedPayload leafFraming = iota + 1
	fixedDwords
)

type leafForms uint8

const (
	cityForm leafForms = 1 << iota
	worldForm
)

// leafRefusals keep each leaf's own error texts.
type leafRefusals struct {
	ambiguous, kind, framing, length, padding, bound string
}

type supplementLeaf struct {
	directory string
	name      string
	path      string
	kind      uint32
	framing   leafFraming
	bound     int
	validate  func([]byte) error // fixedDwords layout
	forms     leafForms
	refusals  leafRefusals
}

var (
	nativeActionsLeaf = &supplementLeaf{directory: supplementDirectory, name: nativeActionsName, path: NativeActionsPath,
		kind: 6, framing: framedPayload, bound: MaxNativeActions, forms: cityForm | worldForm,
		refusals: leafRefusals{
			ambiguous: "sav: ambiguous native action leaf", kind: "sav: ambiguous native action leaf",
			framing: "sav: invalid native action framing", length: "sav: invalid native action length",
			padding: "sav: nonzero native action padding", bound: "sav: native actions require a bounded application leaf"}}
	nativeModsLeaf = &supplementLeaf{directory: supplementDirectory, name: nativeModsName, path: NativeModsPath,
		kind: 6, framing: framedPayload, bound: MaxNativeMods, forms: cityForm | worldForm,
		refusals: leafRefusals{
			ambiguous: "sav: ambiguous mod mark leaf", kind: "sav: invalid mod mark framing",
			framing: "sav: invalid mod mark framing", length: "sav: invalid mod mark length",
			padding: "sav: nonzero mod mark padding", bound: "sav: the mod mark requires a bounded application leaf"}}
	nativeRandomLeaf = &supplementLeaf{directory: supplementDirectory, name: nativeRandomName, path: NativeRandomPath,
		kind: 6, framing: fixedDwords, validate: validateNativeRandom, forms: worldForm,
		refusals: leafRefusals{ambiguous: "sav: ambiguous or invalid AgainromRng leaf", kind: nativeRandomRefusal}}
	nativeSessionLeaf = &supplementLeaf{directory: supplementDirectory, name: nativeSessionName, path: NativeSessionPath,
		kind: 6, framing: fixedDwords, validate: validateNativeSession, forms: cityForm | worldForm,
		refusals: leafRefusals{ambiguous: "sav: ambiguous or invalid AgainromSeed leaf", kind: nativeSessionRefusal}}
)

// supplementLeaves is every leaf, in shape validation order.
var supplementLeaves = []*supplementLeaf{nativeRandomLeaf, nativeSessionLeaf, nativeModsLeaf, nativeActionsLeaf}

func (d *supplementLeaf) check(kind uint32, b []byte) error {
	if kind != d.kind {
		return errors.New(d.refusals.kind)
	}
	if d.framing != framedPayload {
		return d.validate(b)
	}
	if len(b) < 8 || len(b) > d.bound+12 || len(b)%4 != 0 || binary.LittleEndian.Uint32(b) != 1 {
		return errors.New(d.refusals.framing)
	}
	n := uint64(binary.LittleEndian.Uint32(b[4:]))
	if n == 0 || n > uint64(d.bound) || n > uint64(len(b)-8) || uint64(len(b)) != (n+11)&^3 {
		return errors.New(d.refusals.length)
	}
	for _, pad := range b[8+n:] {
		if pad != 0 {
			return errors.New(d.refusals.padding)
		}
	}
	return nil
}

func (d *supplementLeaf) lookup(state DocumentStateData) ([]byte, bool, error) {
	value, found, err := state.ReadLeaf(d.path)
	if err != nil {
		return nil, false, errors.New(d.refusals.ambiguous)
	}
	if !found {
		return nil, false, nil
	}
	if value.Int32 != 0 {
		return nil, false, errors.New(d.refusals.ambiguous)
	}
	if err := d.check(value.Kind, value.Bytes); err != nil {
		return nil, false, err
	}
	return value.Bytes, true, nil
}

func (d *supplementLeaf) payload(b []byte) []byte {
	return b[8 : 8+binary.LittleEndian.Uint32(b[4:])]
}

func (d *supplementLeaf) lookupPayload(state DocumentStateData) ([]byte, bool, error) {
	b, found, err := d.lookup(state)
	if err != nil || !found {
		return nil, false, err
	}
	return d.payload(b), true, nil
}

// store replaces or appends the framed value; records stay sorted by path.
func (d *supplementLeaf) store(state *DocumentStateData, b []byte) error {
	if _, _, err := d.lookup(*state); err != nil {
		return err
	}
	r := CityStateRecordData{Path: d.path, Value: CityStateValueData{Kind: d.kind, Bytes: b}}
	records := slices.Clone(state.ValueRecords)
	if at := StateLeafRecords(state.RootKind, state.DirectoryRecords, records, d.path); len(at) == 0 {
		records = append(records, r)
	} else {
		r.Path = records[at[0]].Path
		records[at[0]] = r
	}
	slices.SortFunc(records, func(a, b CityStateRecordData) int { return strings.Compare(a.Path, b.Path) })
	state.ValueRecords = records
	return nil
}

func (d *supplementLeaf) storePayload(state *DocumentStateData, value []byte) error {
	if state == nil || len(value) == 0 || len(value) > d.bound {
		return errors.New(d.refusals.bound)
	}
	b := make([]byte, (len(value)+11)&^3)
	binary.LittleEndian.PutUint32(b, 1)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(value)))
	copy(b[8:], value)
	return d.store(state, b)
}

func (d *supplementLeaf) remove(state *DocumentStateData) {
	if state == nil {
		return
	}
	state.ValueRecords = slices.DeleteFunc(slices.Clone(state.ValueRecords), func(r CityStateRecordData) bool {
		return StateRecordIs(state.RootKind, state.DirectoryRecords, r.Path, d.path)
	})
}

// placeSupplementLeaves inserts each present leaf the form carries where its
// name sorts: REG-100 sorted lookup compares case-sensitive bytes.
func placeSupplementLeaves(state *cityState, shape []stateDirectory, form leafForms) ([]stateDirectory, error) {
	if state == nil {
		return shape, nil
	}
	lookup := state.leafIndex()
	for _, d := range supplementLeaves {
		record, ok, err := lookup.read(d.path)
		if err != nil {
			return nil, err
		}
		value := record.Value
		if d.forms&form == 0 || !ok {
			continue
		}
		if err := d.check(value.Kind, value.Bytes); err != nil {
			return nil, err
		}
		for i := range shape {
			if shape[i].directory != d.directory {
				continue
			}
			children := slices.Clone(shape[i].children)
			at := slices.IndexFunc(children, func(l stateLeaf) bool { return l.name > d.name })
			if at < 0 {
				at = len(children)
			}
			shape[i].children = slices.Insert(children, at, stateLeaf{d.name, d.kind})
		}
	}
	return shape, nil
}

// HasNativeActionsLeaf includes collisions so a malformed authored save cannot
// fall through to the original-save route.
func (f *File) HasNativeActionsLeaf() bool {
	_, present, err := f.StateLeaves().ReadLeaf(nativeActionsLeaf.path)
	return present || err != nil
}

// MayCarryNativeMods: the mod mark's name appears in the save's bytes.
func MayCarryNativeMods(saved []byte) bool {
	return bytes.Contains(saved, []byte(nativeModsLeaf.name))
}
