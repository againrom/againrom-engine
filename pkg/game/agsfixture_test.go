package game

// Test-only AGS fixtures. The shipped game neither writes nor reads the retired
// AGS format. Tests that compare a whole in-process Snapshot across serialization keep this encoder
// and decoder so their fixtures stay byte-stable, and tests that need a file in
// a SaveStore directory keep the Write helper below.

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
	"encoding/binary"
	"hash/crc32"
)

const saveExt = ".ags"

var ErrNotSave = errors.New("not an againrom save")

// AGS fixture files written by the test Write helper stage under their own
// extension, so their staging names are recovered like a SAV's.
func init() { ownedSaveTempExtensions = append(ownedSaveTempExtensions, saveExt) }

const (
	saveMagic     = "AGRMSAVE"
	saveVersion   = 1
	saveHeaderLen = len(saveMagic) + 1 + 2
	saveMaxLabel  = 4096
	saveMaxBytes  = 16 << 20
)

// agsParse validates the fixture envelope and returns its label and payload.
func agsParse(b []byte) (label string, payload []byte, err error) {
	if len(b) > saveMaxBytes {
		return "", nil, fmt.Errorf("save is %d bytes; safe maximum is %d", len(b), saveMaxBytes)
	}
	if len(b) < saveHeaderLen {
		return "", nil, fmt.Errorf("%w: file is %d bytes, shorter than a header", ErrNotSave, len(b))
	}
	if string(b[:len(saveMagic)]) != saveMagic {
		return "", nil, ErrNotSave
	}
	if v := b[len(saveMagic)]; v != saveVersion {
		return "", nil, fmt.Errorf("save version %d, this build reads version %d", v, saveVersion)
	}
	n := int(binary.LittleEndian.Uint16(b[len(saveMagic)+1:]))
	rest := b[saveHeaderLen:]
	if len(rest) < n {
		return "", nil, fmt.Errorf("%w: truncated label", ErrNotSave)
	}
	label, rest = string(rest[:n]), rest[n:]
	if len(rest) < 8 {
		return "", nil, fmt.Errorf("%w: truncated", ErrNotSave)
	}
	sum := binary.LittleEndian.Uint32(rest)
	length := binary.LittleEndian.Uint32(rest[4:])
	body := rest[8:]
	if uint64(len(body)) != uint64(length) {
		return "", nil, fmt.Errorf("save payload is %d bytes, header declares %d", len(body), length)
	}
	if crc32.ChecksumIEEE(body) != sum {
		return "", nil, errors.New("save is corrupt: the payload checksum does not match")
	}
	return label, body, nil
}

// agsBuild wraps a gob payload in the fixture envelope under a label.
func agsBuild(label string, payload []byte) ([]byte, error) {
	if len(label) > saveMaxLabel {
		label = label[:saveMaxLabel]
	}
	total := saveHeaderLen + len(label) + 8 + len(payload)
	if total > saveMaxBytes {
		return nil, fmt.Errorf("encoded save is %d bytes; safe maximum is %d", total, saveMaxBytes)
	}
	var out bytes.Buffer
	out.Grow(total)
	out.WriteString(saveMagic)
	out.WriteByte(saveVersion)
	out.Write(binary.LittleEndian.AppendUint16(nil, uint16(len(label))))
	out.WriteString(label)
	out.Write(binary.LittleEndian.AppendUint32(nil, crc32.ChecksumIEEE(payload)))
	out.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(payload))))
	out.Write(payload)
	return out.Bytes(), nil
}

// EncodeSave builds an AGS file for a snapshot under a label.
func EncodeSave(s Snapshot, label string) ([]byte, error) {
	if err := validatePendingGameOptions(s.Residue.PendingGameOptions); err != nil {
		return nil, err
	}
	if err := validateSnapshotFame(s.Fame); err != nil {
		return nil, err
	}
	if err := validateApplicationState(s.ApplicationState); err != nil {
		return nil, err
	}
	if err := validateNativeFogResidue(s.Residue); err != nil {
		return nil, err
	}
	ownedDocument, err := savedDocumentFromSnapshot(s)
	if err != nil {
		return nil, err
	}
	s.SavedDocument = ownedDocument
	if err := validateSnapshotActorManifest(s); err != nil {
		return nil, err
	}
	if err := validateQuickSpells(s.QuickSpells); err != nil {
		return nil, err
	}
	if err := validateSnapshotItemWeights(s); err != nil {
		return nil, err
	}
	if err := validateSnapshotBooks(s); err != nil {
		return nil, err
	}
	city, err := originalCityFromSnapshot(s)
	if err != nil {
		return nil, err
	}
	if city != nil {
		s.OriginalCity = city.snapshot()
	}
	if _, err := campaignDifficulty(int64(s.Difficulty)); err != nil {
		return nil, err
	}
	var payload bytes.Buffer
	if err := gob.NewEncoder(&payload).Encode(s); err != nil {
		return nil, fmt.Errorf("encode save: %w", err)
	}
	return agsBuild(label, payload.Bytes())
}

// DecodeSave reads a whole AGS file back. It does not adopt the two historical
// field names.
func DecodeSave(b []byte) (Snapshot, string, error) {
	label, body, err := agsParse(b)
	if err != nil {
		return Snapshot{}, "", err
	}
	var s Snapshot
	dec := gob.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&s); err != nil {
		return Snapshot{}, "", fmt.Errorf("save is corrupt: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return Snapshot{}, "", errors.New("save is corrupt: trailing data after the snapshot")
		}
		return Snapshot{}, "", fmt.Errorf("save is corrupt: trailing data after the snapshot: %w", err)
	}
	if err := adoptHistoricalNames(&s, body); err != nil {
		return Snapshot{}, "", err
	}
	if err := adoptDecodedAGS(&s); err != nil {
		return Snapshot{}, "", err
	}
	return s, label, nil
}

// Write puts an AGS file in the store directory under a generated name.
func (s SaveStore) Write(now time.Time, data []byte) (string, error) {
	base := now.Format("save-20060102-150405")
	return s.write(data, base, saveExt, nil, func(at int) (string, bool) {
		if at == 0 {
			return base + saveExt, true
		}
		return fmt.Sprintf("%s-%d%s", base, at+1, saveExt), true
	})
}

// listAGS lists the AGS files in a store directory, newest first.
func listAGS(s SaveStore) ([]SaveEntry, error) {
	ents, err := os.ReadDir(s.Dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []SaveEntry
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), saveExt) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		b, err := s.Read(e.Name())
		if err != nil {
			continue
		}
		label, _, err := agsParse(b)
		if err != nil {
			continue
		}
		out = append(out, SaveEntry{Name: e.Name(), Label: label, Mod: fi.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Mod.Equal(out[j].Mod) {
			return out[i].Name > out[j].Name
		}
		return out[i].Mod.After(out[j].Mod)
	})
	return out, nil
}

// ConvertSave converts a test AGS fixture to SAV through the converter's own
// entry point. A nil label keeps the AGS header label; a typed one replaces it.
func (f *FrontEnd) ConvertSave(input []byte, to string, label *string) ([]byte, string, error) {
	if to != "sav" {
		return nil, "", fmt.Errorf("target format %q must be sav", to)
	}
	snapshot, source, err := DecodeSave(input)
	if err != nil {
		return nil, "", err
	}
	if label != nil {
		source = *label
	}
	conv, err := f.ConvertLegacySnapshot(snapshot, source)
	return conv.SAV, conv.Label, err
}

func (f *FrontEnd) ConvertCitySave(input []byte, to string, label *string) ([]byte, string, error) {
	return f.ConvertSave(input, to, label)
}

// agsSaveSeams is SaveSeams plus test-only handling of AGS fixture files in the
// store directory: they are listed ahead of the SAV rows and a LOAD of one runs
// the test decoder and the production Restore. The shipped seams read SAV only.
func agsSaveSeams(f *FrontEnd, store SaveStore, orig OriginalStore, now func() time.Time) (ui.SaveGame, ui.SaveList, ui.LoadGame) {
	save, list, load := f.SaveSeams(store, orig, now)
	agsList := func() []ui.SaveEntry {
		var out []ui.SaveEntry
		if rows, err := listAGS(store); err == nil {
			for _, row := range rows {
				out = append(out, ui.SaveEntry{Name: row.Name, Label: drawableLabel(row.Label)})
			}
		}
		return append(out, list()...)
	}
	agsLoad := func(name string) (ui.MapOpener, bool, error) {
		if !strings.EqualFold(filepath.Ext(name), saveExt) {
			return load(name)
		}
		b, err := store.Read(name)
		if err != nil {
			return nil, false, err
		}
		s, _, err := DecodeSave(b)
		if err != nil {
			return nil, false, err
		}
		return f.Restore(s)
	}
	return save, agsList, agsLoad
}

// historicalEnvelope frames an arbitrary gob payload in the AGS envelope.
func historicalEnvelope(t *testing.T, payload []byte, label string) []byte {
	t.Helper()
	out, err := agsBuild(label, payload)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// adoptDecodedAGS finishes a Snapshot decoded from an AGS gob payload with the
// validations and normalisations the fixture LOAD relies on.
func adoptDecodedAGS(s *Snapshot) error {
	if err := validateSnapshotActorManifest(*s); err != nil {
		return err
	}
	if err := validatePendingGameOptions(s.Residue.PendingGameOptions); err != nil {
		return err
	}
	if err := validateSnapshotFame(s.Fame); err != nil {
		return err
	}
	if err := validateNativeFogResidue(s.Residue); err != nil {
		return err
	}
	if err := validateApplicationState(s.ApplicationState); err != nil {
		return err
	}
	ownedDocument, err := savedDocumentFromSnapshot(*s)
	if err != nil {
		return err
	}
	s.SavedDocument = ownedDocument
	if _, err := campaignDifficulty(int64(s.Difficulty)); err != nil {
		return err
	}
	city, err := originalCityFromSnapshot(*s)
	if err != nil {
		return err
	}
	if city != nil {
		s.OriginalCity = city.snapshot()
	}
	if err := validateSnapshotBooks(*s); err != nil {
		return err
	}
	if err := validateQuickSpells(s.QuickSpells); err != nil {
		return err
	}
	return validateSnapshotItemWeights(*s)
}

func validateSnapshotActorManifest(s Snapshot) error {
	if s.ActorManifest == nil {
		return nil
	}
	if s.ActorManifest.Version != actorManifestVersion || s.Mission == 0 || len(s.World) == 0 {
		return fmt.Errorf("saved actor manifest requires a supported mission world")
	}
	return nil
}
