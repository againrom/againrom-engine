package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
)

// ownedSlotKind is a family of three owned save slots: a SAV and a sidecar
// naming kind, slot, sequence and SHA-256. Rotation stays with each kind.
type ownedSlotKind struct {
	word     string // the kind's name in error texts
	tag      string // the sidecar's Kind value
	base     string // the file base; slot n is "<base>-<n+1>"
	ownerExt string // the sidecar's extension
	game     string // leads the base; empty is the older pair
}

var (
	quickSlots = ownedSlotKind{"quick", "againrom-quick-sav", "quick-save", ".quick-owner", ""}
	timedSlots = ownedSlotKind{"timed", "againrom-timed-sav", "timed-autosave", ".timed-owner", ""}
)

type ownedSlot struct {
	free     bool
	sequence uint64
	raw      []byte
}

func (k ownedSlotKind) scan(dir string) [3]ownedSlot {
	var slots [3]ownedSlot
	for slot := range slots {
		base := filepath.Join(dir, string(k.slotBase(slot)))
		info, err := os.Lstat(base + ".sav")
		ownerInfo, ownerErr := os.Lstat(base + k.ownerExt)
		if os.IsNotExist(err) && os.IsNotExist(ownerErr) {
			slots[slot].free = true
			continue
		}
		if err != nil || ownerErr != nil || !info.Mode().IsRegular() || !ownerInfo.Mode().IsRegular() {
			continue
		}
		raw, err := ReadSaveFile(base + ".sav")
		owner, ownerErr := ReadSaveFile(base + k.ownerExt)
		if err != nil || ownerErr != nil {
			continue
		}
		if sequence, err := k.validatePair(slot, raw, owner); err == nil {
			slots[slot] = ownedSlot{sequence: sequence, raw: raw}
		}
	}
	return slots
}

// forGame is the kind's slots of game g.
func (k ownedSlotKind) forGame(g base.Game) ownedSlotKind {
	k.game = g.Edition().SlotWord
	return k
}

// allOwnedSlotKinds is every kind a save folder may hold.
func allOwnedSlotKinds() []ownedSlotKind {
	out := []ownedSlotKind{timedSlots, quickSlots}
	for _, word := range base.SlotWords() {
		for _, k := range []ownedSlotKind{timedSlots, quickSlots} {
			k.game = word
			out = append(out, k)
		}
	}
	return out
}

// ownedSlotRecord's field names and order are the sidecar bytes.
type ownedSlotRecord struct {
	Kind     string
	Slot     int
	Sequence uint64
	SHA256   string
}

func (k ownedSlotKind) slotBase(slot int) validatedSaveName {
	name := k.base
	if k.game != "" {
		name = k.game + "-" + name
	}
	return validatedSaveName(fmt.Sprintf("%s-%d", name, slot+1))
}

func (k ownedSlotKind) validatePair(slot int, raw, owner []byte) (uint64, error) {
	var record ownedSlotRecord
	if len(owner) > 1024 || json.Unmarshal(owner, &record) != nil || record.Kind != k.tag || record.Slot != slot ||
		record.Sequence == 0 || record.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		return 0, fmt.Errorf("%s slot %d has no matching ownership record", k.word, slot+1)
	}
	if _, err := sav.DecodeDocumentData(raw); err != nil {
		return 0, fmt.Errorf("%s slot %d is corrupt: %w", k.word, slot+1, err)
	}
	return record.Sequence, nil
}

// deleteCompanion is the owning sidecar a delete of path takes with it.
func (k ownedSlotKind) deleteCompanion(path string, raw []byte) (namedSaveTarget, bool) {
	base := strings.TrimSuffix(path, filepath.Ext(path))
	for slot := range 3 {
		if !sameSaveName(filepath.Base(base), string(k.slotBase(slot))) {
			continue
		}
		target := namedSaveTarget{path: base + k.ownerExt}
		var err error
		target.before, err = os.Lstat(target.path)
		if err != nil || !target.before.Mode().IsRegular() {
			return namedSaveTarget{}, false
		}
		target.old, err = ReadSaveFile(target.path)
		if err != nil {
			return namedSaveTarget{}, false
		}
		if _, err := k.validatePair(slot, raw, target.old); err != nil {
			return namedSaveTarget{}, false
		}
		target.oldHash = sha256.Sum256(target.old)
		return target, true
	}
	return namedSaveTarget{}, false
}

// write replaces a free slot or this kind's older pair.
func (k ownedSlotKind) write(store SaveStore, fences []string, slot int, sequence uint64, raw []byte, files namedSaveFileOps) error {
	owner, err := json.Marshal(ownedSlotRecord{k.tag, slot, sequence, fmt.Sprintf("%x", sha256.Sum256(raw))})
	if err != nil {
		return err
	}
	prepared, err := prepareOwnedNamedSave(store.Dir, k.slotBase(slot), []namedSavePayload{{".sav", raw}, {k.ownerExt, owner}}, fences, files,
		func(targets []namedSaveTarget) error {
			if targets[0].before == nil && targets[1].before == nil {
				return nil
			}
			if targets[0].before == nil || targets[1].before == nil {
				return fmt.Errorf("%s slot %d is occupied by an unowned file", k.word, slot+1)
			}
			previous, err := k.validatePair(slot, targets[0].old, targets[1].old)
			if err != nil {
				return err
			}
			if previous >= sequence {
				return fmt.Errorf("%s slot changed after rotation selection", k.word)
			}
			return nil
		}, store.profile)
	if err != nil {
		return err
	}
	_, err = prepared.Commit(true)
	return err
}
