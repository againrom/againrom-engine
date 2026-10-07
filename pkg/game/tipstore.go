package game

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Local persisted game preferences (1018 spec behaviour 4).
//
// THE FILE HOLDS ONLY THE OPTIONS THIS BUILD ACTUALLY CONSUMES. TipsMode,
// GameSpeed, WimpyMode, FormationMode, ShowAllHitPoints, ShowFlyingHP,
// ShowTimeFlow, TooltipDelay and the local audio master controls are consumed.
// Unknown key/value pairs survive writes; line order and comments do not.

// OptionsStore is the small persisted-preference store beside saves/.
type OptionsStore struct {
	Path        string
	originalDir string
	profile     *runtimeProfileAccess
}

const optionsFileName = "options.txt"

// DefaultOptionsPath is the store beside saves/, not inside it: DefaultSaveDir
// resolves the saves directory itself (override forwarded unchanged, so
// cmd/againrom's own -saves flag places the two consistently), and this
// takes the parent of whatever that answers — beside the running executable,
// beside the working directory for a `go run` launch, or beside an
// overridden saves directory.
func DefaultOptionsPath(saveDirOverride string) (string, error) {
	dir, err := DefaultSaveDir(saveDirOverride)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(dir), optionsFileName), nil
}

func (s OptionsStore) readAll() (map[string]string, error) {
	m := map[string]string{}
	if s.Path == "" {
		return m, nil
	}
	b, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return m, nil
		}
		return nil, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		k, v, ok := strings.Cut(line, "=")
		if !ok || k == "" {
			continue
		}
		m[k] = v
	}
	return m, nil
}

func (s OptionsStore) writeAll(m map[string]string) error {
	if s.Path == "" {
		return errors.New("no options path configured")
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, m[k])
	}
	dir, err := namedSaveDirectory(filepath.Dir(s.Path), []string{s.originalDir}, s.profile)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, filepath.Base(s.Path))
	target := namedSaveTarget{path: path, data: []byte(b.String())}
	target.before, err = os.Lstat(path)
	if err == nil {
		if !target.before.Mode().IsRegular() {
			return fmt.Errorf("options target is not a regular file")
		}
		target.old, err = os.ReadFile(path)
		if err != nil {
			return err
		}
		target.oldHash = sha256.Sum256(target.old)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	identity, err := resolveSaveDirectory(dir)
	if err != nil {
		return err
	}
	if _, err := namedSaveDirectory(identity.path, []string{s.originalDir}, s.profile); err != nil {
		return err
	}
	return commitNamedSaveSet(identity, []namedSaveTarget{target}, []string{s.originalDir}, osNamedSaveFiles{}, s.profile)
}

// tipsModeKey is TOWN-186's own registry value name, unchanged.
const tipsModeKey = "TipsMode"
const textSmoothingKey = "TextSmoothing"

// frameSmoothingKey stores the final-frame scaler choice: absent, malformed
// or any value but "0" is the Catmull-Rom scaler, "0" is method B.
const frameSmoothingKey = "FrameSmoothing"

// smoothingSwitches holds the two presentation smoothing switches, each
// negated so the zero value leaves both on.
type smoothingSwitches struct {
	text, frame bool
}

// gameSpeedKey is TOWN-186's own registry value name, unchanged. The local
// file stores this build's complete normal cadence rung rather than the
// unpaced owner-loop selector: the ladder deliberately extends beyond the
// original's nine shipped speed indexes, and every one of its rungs remains a
// normal deadline-paced speed.
const gameSpeedKey = "GameSpeed"

// wimpyModeKey is TOWN-186's own registry value name, unchanged. The local
// value is the current label in Ctrl+W's Off/Low/High cycle. Reading it does
// not rewrite a loaded world's canonical thresholds; only the next player
// command does that.
const wimpyModeKey = "WimpyMode"

const (
	wimpyModeOff = iota
	wimpyModeLow
	wimpyModeHigh
)

func defaultGameSpeedRung() int {
	return terrain.DefaultCadenceRung
}

// TipsMode reads the stored preference.
func (s OptionsStore) TipsMode() (bool, error) {
	if s.Path == "" {
		return true, nil
	}
	m, err := s.readAll()
	if err != nil {
		return true, err
	}
	v, ok := m[tipsModeKey]
	if !ok {
		return true, nil
	}
	return v != "0", nil
}

// TextSmoothing reads owner decision method C's own stored preference
// (DIV-1385), the presentation-layer text overlay's on/off switch. It
// defaults true (on), the same shape TipsMode gives an old options file that
// predates its own key: absent or malformed reads as the new default rather
// than silently turning it off.
func (s OptionsStore) TextSmoothing() (bool, error) {
	if s.Path == "" {
		return true, nil
	}
	m, err := s.readAll()
	if err != nil {
		return true, err
	}
	v, ok := m[textSmoothingKey]
	if !ok {
		return true, nil
	}
	return v != "0", nil
}

// SetTextSmoothing writes the preference, keeping every other key already in
// the file untouched.
func (s OptionsStore) SetTextSmoothing(on bool) error {
	return s.setSwitch(textSmoothingKey, on)
}

// FrameSmoothing reads the final-frame scaler choice the way TextSmoothing
// is read: true, the Catmull-Rom scaler, unless the stored value is "0".
func (s OptionsStore) FrameSmoothing() (bool, error) {
	if s.Path == "" {
		return true, nil
	}
	m, err := s.readAll()
	if err != nil {
		return true, err
	}
	v, ok := m[frameSmoothingKey]
	return !ok || v != "0", nil
}

// SetFrameSmoothing writes the choice, keeping every other key in the file.
func (s OptionsStore) SetFrameSmoothing(on bool) error {
	return s.setSwitch(frameSmoothingKey, on)
}

func (s OptionsStore) setSwitch(key string, on bool) error {
	if s.Path == "" {
		return errors.New("no options path configured")
	}
	m, err := s.readAll()
	if err != nil {
		m = map[string]string{}
	}
	if on {
		m[key] = "1"
	} else {
		m[key] = "0"
	}
	return s.writeAll(m)
}

// SetTipsMode writes the preference, keeping every other key already in the
// file untouched.
func (s OptionsStore) SetTipsMode(on bool) error {
	if s.Path == "" {
		return errors.New("no options path configured")
	}
	m, err := s.readAll()
	if err != nil {
		m = map[string]string{}
	}
	if on {
		m[tipsModeKey] = "1"
	} else {
		m[tipsModeKey] = "0"
	}
	return s.writeAll(m)
}

// GameSpeed reads the normal deadline-paced cadence rung. A missing store,
// key, malformed integer or value outside the complete ladder all answer the
// existing shipped default; a damaged preference must never select an edge
// speed merely because it was out of range. The unpaced selector has no key
// and cannot be restored through this method.
func (s OptionsStore) GameSpeed() (int, error) {
	fallback := defaultGameSpeedRung()
	if s.Path == "" {
		return fallback, nil
	}
	m, err := s.readAll()
	if err != nil {
		return fallback, err
	}
	raw, ok := m[gameSpeedKey]
	if !ok {
		return fallback, nil
	}
	rung, err := strconv.Atoi(raw)
	if err != nil || rung < terrain.CadenceRungMin || rung > terrain.CadenceRungMax {
		return fallback, nil
	}
	return rung, nil
}

// SetGameSpeed persists one normal cadence rung while preserving every other
// option. Invalid caller values are refused rather than clamped: clamping a
// corrupted value here would make a later process treat an unintended ladder
// edge as a deliberate player choice.
func (s OptionsStore) SetGameSpeed(rung int) error {
	if s.Path == "" {
		return errors.New("no options path configured")
	}
	if rung < terrain.CadenceRungMin || rung > terrain.CadenceRungMax {
		return fmt.Errorf("game speed rung %d outside [%d,%d]", rung,
			terrain.CadenceRungMin, terrain.CadenceRungMax)
	}
	m, err := s.readAll()
	if err != nil {
		m = map[string]string{}
	}
	m[gameSpeedKey] = strconv.Itoa(rung)
	return s.writeAll(m)
}

// WimpyMode reads Ctrl+W's current persisted label. Missing, malformed and
// out-of-range values all answer Off, the inert fallback: a damaged process
// preference must not silently arm retreat. This method returns only the
// cycle state and never applies it to a world.
func (s OptionsStore) WimpyMode() (int, error) {
	if s.Path == "" {
		return wimpyModeOff, nil
	}
	m, err := s.readAll()
	if err != nil {
		return wimpyModeOff, err
	}
	raw, ok := m[wimpyModeKey]
	if !ok {
		return wimpyModeOff, nil
	}
	mode, err := strconv.Atoi(raw)
	if err != nil || mode < wimpyModeOff || mode > wimpyModeHigh {
		return wimpyModeOff, nil
	}
	return mode, nil
}

// SetWimpyMode persists one decoded retreat-mode label while preserving every
// other option. Invalid values are refused rather than folded onto a real
// label.
func (s OptionsStore) SetWimpyMode(mode int) error {
	if s.Path == "" {
		return errors.New("no options path configured")
	}
	if mode < wimpyModeOff || mode > wimpyModeHigh {
		return fmt.Errorf("wimpy mode %d outside [%d,%d]", mode, wimpyModeOff, wimpyModeHigh)
	}
	m, err := s.readAll()
	if err != nil {
		m = map[string]string{}
	}
	m[wimpyModeKey] = strconv.Itoa(mode)
	return s.writeAll(m)
}

// LoadOptions reads the process preferences cached by FrontEnd. Called once
// after Options is set (cmd/againrom's own wiring, on SaveSeams's own
// precedent — the store's location is a main-package concern, not
// NewFrontEnd's). A FrontEnd a test builds by hand and never calls this keeps
// both zero-value defaults: tips shown and retreat Off.
//
// Loading WimpyMode deliberately changes no world and no entity threshold.
// It is only the current label from which the next Ctrl+W press cycles; a
// loaded world's canonical Withdraw/Wimpy values remain the save's values
// until an ordinary parameter-3 command is executed.
func (f *FrontEnd) LoadOptions() {
	if f == nil {
		return
	}
	on, _ := f.Options.TipsMode()
	f.tipsOff = !on
	smoothing, _ := f.Options.TextSmoothing()
	f.smoothingOff.text = !smoothing
	frame, _ := f.Options.FrameSmoothing()
	f.smoothingOff.frame = !frame
	f.wimpyMode, _ = f.Options.WimpyMode()
	acknowledgments, _ := f.Options.Acknowledgments()
	f.acknowledgmentsOff = !acknowledgments
	options, present, _ := f.Options.gameOptions()
	defaults := defaultGraphicsValues()
	for o := ui.GameOptionSmoothing; o <= ui.GameOptionAnimation; o++ {
		if !present[o] {
			options[o] = defaults[o]
		}
	}
	f.graphics = graphicsValues(options)
	if f.graphics.StaticObjects {
		f.graphics.DisableLighting = true
	}
	f.showPathfinding = options[ui.GameOptionPathfinding] != 0
}

// TipsOff is the permanent suppression's current state (1018 spec behaviour
// 4). A room reads this at its own entry, on TOWN-186's own gate-tested-at-
// construction reading: toggling it does not retroactively hide an already
// open panel.
func (f *FrontEnd) TipsOff() bool { return f != nil && f.PersistenceContext.tipsOffNow() }

func (p *PersistenceContext) tipsOffNow() bool { return p.tipsOff }

// SetTipsOff updates the cached flag and persists it. A store with no Path
// (Options never set) still updates the in-memory flag, matching every
// other install-backed field's "absent source, no-op write" shape.
func (f *FrontEnd) SetTipsOff(off bool) {
	if f == nil {
		return
	}
	f.PersistenceContext.setTipsOff(off)
}

func (p *PersistenceContext) setTipsOff(off bool) {
	p.tipsOff = off
	_ = p.Options.SetTipsMode(!off)
}
