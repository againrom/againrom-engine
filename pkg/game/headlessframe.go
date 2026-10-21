package game

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/ui"
)

// HeadlessTown is the town a capture on a town screen names: the party's town
// ID, its name as the world map draws it (UTF-8) and the keys of the pictures
// its square draws, an install key or a mod path such as "mod:art/east.bmp".
type HeadlessTown struct {
	ID   int      `json:"id"`
	Name string   `json:"name"`
	Art  []string `json:"art"`
}

// HeadlessFrame is the picture a frame command wrote.
type HeadlessFrame struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

var (
	// ErrHeadlessFrameInstall refuses a frame path inside an install.
	ErrHeadlessFrameInstall = errors.New("a frame is never written into an install")
	// ErrHeadlessFrameExists refuses a frame path that already names something.
	ErrHeadlessFrameExists = errors.New("a frame is never written over an existing file")
)

// headlessTown answers the town a town screen shows, nil where none is installed. home is the town word a town
// with no name of its own shows; selector is the install's text selector.
func headlessTown(town *Town, screen *townScreen, home string, selector int) *HeadlessTown {
	if town == nil || screen == nil {
		return nil
	}
	id := town.townID()
	out := &HeadlessTown{ID: id, Art: []string{}}
	name := home
	if e, ok := screen.townBook().catalog.Town(id); ok && e.Name != "" {
		name = e.Name
	}
	if shown, ok := decodeSaveLabel(name, selector); ok {
		name = shown
	} else {
		name = fmt.Sprintf("bytes % x", name)
	}
	out.Name = name
	d := screen.townDescription()
	if d == nil {
		return out
	}
	byName := map[string]int{}
	for i, e := range d.Art {
		byName[e.Name] = i
	}
	seen := map[string]bool{}
	for _, l := range d.Layers {
		i, ok := byName[l.Art]
		if !ok || d.Art[i].Format != "picture" {
			continue
		}
		for _, key := range d.Art[i].Keys() {
			if !seen[key] {
				seen[key] = true
				out.Art = append(out.Art, key)
			}
		}
	}
	return out
}

// checkHeadlessFramePath refuses a frame path, absolute, inside an install or
// already taken. It writes nothing.
func (f *FrontEnd) checkHeadlessFramePath(path string) error {
	if f == nil || f.Archives == nil || f.Archives.Root == "" {
		return errors.New("a frame needs an opened install")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	install, inside, err := insideInstall(f.Archives.Root, abs)
	if err != nil {
		return err
	}
	if inside {
		return fmt.Errorf("%s is inside the install %s: %w", abs, install, ErrHeadlessFrameInstall)
	}
	if _, err := os.Lstat(abs); err == nil {
		return fmt.Errorf("%s: %w", abs, ErrHeadlessFrameExists)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// writeHeadlessFrame writes the frame the headless renderer composes for the
// app's current state to a new file at path as a PNG. A refusal or a failure
// leaves no file behind. It returns the renderer's note about a frame that is
// correct but incomplete.
func (f *FrontEnd) writeHeadlessFrame(app *ui.App, path string) (HeadlessFrame, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return HeadlessFrame{}, "", err
	}
	if err := f.checkHeadlessFramePath(abs); err != nil {
		return HeadlessFrame{}, "", err
	}
	pix, note, err := app.HeadlessFrame()
	if err != nil {
		return HeadlessFrame{}, "", err
	}
	var encoded screenshotBuffer
	if err := png.Encode(&encoded, pix); err != nil {
		return HeadlessFrame{}, "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return HeadlessFrame{}, "", err
	}
	out, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return HeadlessFrame{}, "", fmt.Errorf("%s: %w", abs, ErrHeadlessFrameExists)
	}
	if err != nil {
		return HeadlessFrame{}, "", err
	}
	_, werr := out.Write(encoded.Bytes())
	if cerr := out.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		os.Remove(abs)
		return HeadlessFrame{}, "", werr
	}
	b := pix.Bounds()
	return HeadlessFrame{Path: abs, Width: b.Dx(), Height: b.Dy()}, note, nil
}

// checkHeadlessFramePaths refuses, before the first step runs, a scenario
// whose frame commands name a path inside an install, an existing file, a
// path another frame command names, or a path it cannot judge.
func (f *FrontEnd) checkHeadlessFramePaths(steps []HeadlessStep) error {
	named := map[string]int{}
	for i, step := range steps {
		if !strings.EqualFold(strings.TrimSpace(step.Command), "frame") {
			continue
		}
		abs, err := filepath.Abs(step.Path)
		if err != nil {
			return fmt.Errorf("step %d (frame): %w", i+1, err)
		}
		if first, dup := named[strings.ToLower(abs)]; dup {
			return fmt.Errorf("step %d (frame): %s: step %d writes the same file", i+1, abs, first)
		}
		named[strings.ToLower(abs)] = i + 1
		if err := f.checkHeadlessFramePath(abs); err != nil {
			return fmt.Errorf("step %d (frame): %w", i+1, err)
		}
	}
	return nil
}
