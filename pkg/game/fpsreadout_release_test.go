package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/res"
	"againrom/pkg/formats/sav"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

type fpsSourceCell struct {
	level   uint8
	painted bool
}

type fpsSourceGlyph struct {
	w, h, advance int
	cells         []fpsSourceCell
}

type fpsFontInput map[string][]byte

func (s fpsFontInput) ReadFile(path string) ([]byte, error) {
	if b, ok := s[path]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("unexpected font input %s", path)
}

func fpsPaintedZeroControl(t *testing.T, atlas, advances []byte) {
	t.Helper()
	changed := bytes.Clone(atlas)
	mutated := false
	for offset := 0; offset+12 <= len(changed)-4 && !mutated; {
		n := int(binary.LittleEndian.Uint32(changed[offset+8:]))
		start := offset + 12
		for cursor := start; cursor < start+n; {
			op := changed[cursor]
			cursor++
			if op>>6 == 0 {
				operands := int(op & 63)
				if operands > 1 {
					changed[cursor] &= 15
					mutated = true
					break
				}
				cursor += operands
			}
		}
		offset = start + n
	}
	if !mutated {
		t.Fatal("derived raw zero control has no mid-literal operand")
	}
	expected := fpsSourceFont(t, changed, advances)
	font, err := LoadFont(fpsFontInput{"graphics/font1/font1.16": changed, "graphics/font1/font1.dat": advances}, "font1")
	if err != nil {
		t.Fatal(err)
	}
	zero, transparent := 0, 0
	for i, g := range expected {
		for n, p := range g.cells {
			actual := font.Glyphs[i].Pixels[n]
			if actual.Painted != p.painted || actual.Level != p.level {
				t.Fatal("derived mid-literal-zero differs from raw source", i, n)
			}
			if p.painted && p.level == 0 {
				zero++
			} else if !p.painted {
				transparent++
			}
		}
	}
	if zero == 0 || transparent == 0 || bytes.Equal(changed, atlas) {
		t.Fatal("derived raw painted-zero/transparent loss control", zero, transparent)
	}
}

func fpsRawEntry(t *testing.T, archive *res.FileArchive, suffix string) []byte {
	t.Helper()
	var matches []string
	for _, entry := range archive.Entries() {
		path := strings.ToLower(strings.ReplaceAll(entry.Path, "\\", "/"))
		if path == suffix || strings.HasSuffix(path, "/"+suffix) {
			matches = append(matches, entry.Path)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("raw graphics.res %s: %d matches", suffix, len(matches))
	}
	b, err := archive.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func fpsSourceFont(t *testing.T, atlas, advances []byte) []fpsSourceGlyph {
	t.Helper()
	if len(atlas) < 4 {
		t.Fatal("raw glyph trailer absent")
	}
	count := int(binary.LittleEndian.Uint32(atlas[len(atlas)-4:]))
	if count != 224 || len(advances) != count*4 {
		t.Fatal("measured font1 record/advance identity", count, len(advances))
	}
	glyphs := make([]fpsSourceGlyph, count)
	offset := 0
	for i := range glyphs {
		if offset+12 > len(atlas)-4 {
			t.Fatal("raw glyph header crosses trailer", i)
		}
		w := int(binary.LittleEndian.Uint32(atlas[offset:]))
		h := int(binary.LittleEndian.Uint32(atlas[offset+4:]))
		n := int(binary.LittleEndian.Uint32(atlas[offset+8:]))
		offset += 12
		if w > 64 || h > 64 || n > len(atlas)-4-offset {
			t.Fatal("raw font1 dimensions/block", i, w, h, n)
		}
		g := fpsSourceGlyph{w: w, h: h, advance: int(binary.LittleEndian.Uint32(advances[i*4:])), cells: make([]fpsSourceCell, w*h)}
		block := atlas[offset : offset+n]
		offset += n
		cell := 0
		for cursor := 0; cursor < len(block); {
			op := block[cursor]
			cursor++
			n := int(op & 63)
			if op>>6 == 0 {
				if cursor+n > len(block) {
					t.Fatal("raw literal operand crosses block", i)
				}
				for j := 0; j < n; j++ {
					operand := block[cursor]
					cursor++
					levels := []uint8{operand & 15}
					if j != n-1 || operand>>4 != 0 {
						levels = append(levels, operand>>4)
					}
					for _, level := range levels {
						if cell >= len(g.cells) {
							t.Fatal("raw literal crosses glyph", i)
						}
						g.cells[cell] = fpsSourceCell{level, true}
						cell++
					}
				}
			} else {
				if op>>6 == 1 {
					n *= w
				}
				cell += n
				if cell > len(g.cells) {
					t.Fatal("raw skip crosses glyph", i)
				}
			}
		}
		glyphs[i] = g
	}
	return glyphs
}

func fpsSourcePicture(glyphs []fpsSourceGlyph, line string, solid, reverse bool, dx int) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 90, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 90; x++ {
			pic.SetRGBA(x, y, color.RGBA{8, 8, 8, 255})
		}
	}
	advance := func(b byte) int {
		g := glyphs[int(b)-32]
		a := g.advance + 2
		if b == ' ' {
			a += g.h / 2
		}
		return a
	}
	width := 0
	for i := range line {
		width += advance(line[i])
	}
	pass := func(shadow bool) {
		pen := 82 - width + dx
		for i := range line {
			g := glyphs[int(line[i])-32]
			for n, p := range g.cells {
				if !p.painted {
					continue
				}
				x, y := pen+n%g.w, n/g.w
				c := uint8(17 * p.level)
				if solid {
					c = 255
				}
				if shadow {
					x++
					y++
					c = 8
				}
				pic.SetRGBA(x, y, color.RGBA{c, c, c, 255})
			}
			pen += advance(line[i])
		}
	}
	pass(!reverse)
	pass(reverse)
	return pic
}

func fpsCurrentSAV(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestReleaseFPSReadoutCompositionAndSAVLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	out := os.Getenv("AGAINROM_FPS_READOUT_WITNESS_DIR")
	if !filepath.IsAbs(out) {
		t.Fatal("AGAINROM_FPS_READOUT_WITNESS_DIR must name an absolute output directory outside the install")
	}
	resolvedOut, err := filepath.EvalSymlinks(out)
	if err != nil {
		t.Fatal("explicit output directory must already exist", err)
	}
	resolvedInstall, err := filepath.EvalSymlinks(f.Archives.Root)
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(resolvedInstall, resolvedOut); err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("F12 output is inside the install", resolvedOut, err)
	}
	dir := filepath.Join(resolvedOut, filepath.Base(f.Archives.Root))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	graphics, err := res.OpenFileIndex(filepath.Join(f.Archives.Root, "graphics.res"))
	if err != nil {
		t.Fatal(err)
	}
	atlas := fpsRawEntry(t, graphics, "font1/font1.16")
	advances := fpsRawEntry(t, graphics, "font1/font1.dat")
	if len(atlas) != 32932 || len(advances) != 896 {
		t.Fatal("measured font1 node sizes", len(atlas), len(advances))
	}
	glyphs := fpsSourceFont(t, atlas, advances)
	font := f.Font.Value()
	if font == nil || font.Spacing != 2 || len(font.Glyphs) != len(glyphs) {
		t.Fatal("production font1/spacing identity")
	}
	var levels [16]int
	transparent := 0
	for i, source := range glyphs {
		g := font.Glyphs[i]
		if g.Width != source.w || g.Height != source.h || g.Advance != source.advance || len(g.Pixels) != len(source.cells) || g.CoverageLevels {
			t.Fatal("raw font1 geometry/advances versus production", i)
		}
		for n, p := range source.cells {
			if actual := g.Pixels[n]; actual.Painted != p.painted || actual.Level != p.level {
				t.Fatal("raw nibble/painted state versus production", i, n, p, actual)
			}
			if p.painted {
				levels[p.level]++
			} else {
				transparent++
			}
		}
	}
	if transparent == 0 || levels[15] == 0 || levels[8] == 0 {
		t.Fatal("installed transparency/intermediate/white controls absent", levels, transparent)
	}
	fpsPaintedZeroControl(t, atlas, advances)
	store := SaveStore{Dir: filepath.Join(dir, "saves")}
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	app := f.App("F12 installed readout")
	app.Layout(1024, 768)
	app.SetCutscenes(nil)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	party := f.ChargenParty(ui.ChargenResult{Name: "F12 source", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	key := func(k string) {
		t.Helper()
		if err := app.HeadlessKey(k); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 16 && f.live.mission.open; i++ {
		key("enter")
	}
	key("0")
	if !f.live.stopped {
		t.Fatal("fixture did not stop the World")
	}
	text.ResetCapture()
	t.Cleanup(text.ResetCapture)
	wantRate := 52.0 * 1000 / 1040
	for i := 0; i <= 50; i++ {
		rate, pic, _, shown := f.live.view.FPSReadout(5000 + int64(i)*20)
		if rate != 0 || math.Signbit(rate) || pic != nil || shown {
			t.Fatal("constructor/priming/hidden 1000-ms boundary", i, rate, shown)
		}
	}
	if rate, _, _, shown := f.live.view.FPSReadout(6040); rate != wantRate || shown {
		t.Fatal("hidden sampling", rate, shown)
	}
	beforeHash, beforeSAV := f.live.world.Hash(), fpsCurrentSAV(t, f)
	key("f12")
	app.SetTextSmoothing(false)
	rate, pic, at, shown := f.live.view.FPSReadout(6040)
	expected := fpsSourcePicture(glyphs, "50.0 fps", false, false, 0)
	if rate != wantRate || !shown || pic == nil || pic.Bounds() != expected.Bounds() || at != image.Pt(f.live.view.Camera().ViewW-120, 0) || !bytes.Equal(pic.Pix, expected.Pix) || len(text.Captured()) != 0 {
		t.Fatal("installed production F12 composition/hidden sample retention", rate, at, shown)
	}
	for name, changed := range map[string]*image.RGBA{
		"solid-white":      fpsSourcePicture(glyphs, "50.0 fps", true, false, 0),
		"shadow-after-ink": fpsSourcePicture(glyphs, "50.0 fps", false, true, 0),
		"shifted":          fpsSourcePicture(glyphs, "50.0 fps", false, false, 1),
	} {
		if bytes.Equal(pic.Pix, changed.Pix) {
			t.Fatal("installed image loss control did not discriminate", name)
		}
	}
	app.SetTextSmoothing(true)
	_, smooth, _, _ := f.live.view.FPSReadout(6040)
	calls := text.Captured()
	if !bytes.Equal(pic.Pix, smooth.Pix) || len(calls) != 16 || text.Capturing() {
		t.Fatal("smoothing changed CPU raster or lost cached capture", len(calls))
	}
	for i := 0; i < 8; i++ {
		shadow, ink := calls[i], calls[i+8]
		if !shadow.Flat || ink.Flat || shadow.Color != (color.RGBA{8, 8, 8, 255}) || ink.Color != (color.RGBA{255, 255, 255, 255}) || shadow.X != ink.X+1 || shadow.Y != ink.Y+1 || ink.Clip != pic.Bounds().Add(at) {
			t.Fatal("installed shadow/ink order, flat ramp or clipped capture", i)
		}
	}
	if f.live.world.Hash() != beforeHash || !bytes.Equal(fpsCurrentSAV(t, f), beforeSAV) {
		t.Fatal("sampling/composition/F12 changed World or SAV")
	}
	imageFile, err := os.Create(filepath.Join(dir, "readout.png"))
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(imageFile, pic)
	closeErr := imageFile.Close()
	if encodeErr != nil || closeErr != nil {
		t.Fatal(encodeErr, closeErr)
	}
	id := f.live.mission.ids[0]
	body, _ := f.live.entity(id)
	if err := f.live.world.HeadlessDamage(id, 1); err != nil {
		t.Fatal(err)
	}
	f.live.push()
	if f.live.world.Hash() == beforeHash || bytes.Equal(fpsCurrentSAV(t, f), beforeSAV) {
		t.Fatal("changed-state source loss control did not change World/SAV")
	}
	key("f2")
	if err := app.HeadlessSaveEdit(store.Dir, "fps-readout", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if state, open := app.HeadlessSaveState(); open && state.Confirmation {
		if err := app.HeadlessSaveAction("overwrite"); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(store.Dir, "fps-readout.sav")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	hp, err := savedStructureValue(generatedActorRecord(t, &doc, id), "Health")
	if err != nil || int32(int16(hp)) != body.HP-1 {
		t.Fatal("source Health bytes lost changed current state", hp, body.HP, err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	cold.Options = OptionsStore{Path: filepath.Join(dir, "cold-options.txt")}
	coldApp := cold.App("F12 cold ordinary LOAD")
	coldApp.Layout(1024, 768)
	coldApp.SetCutscenes(nil)
	cold.ConfigureSaveSeams(coldApp, store, OriginalStore{}, nil)
	_, list, _ := cold.SaveSeams(store, OriginalStore{}, nil)
	groundAppLoad(t, coldApp, list, "fps-readout.sav")
	loaded, ok := cold.live.entity(id)
	if !ok || loaded.HP != int32(int16(hp)) || cold.live.world.Tick() != uint64(file.Head.CounterA) {
		t.Fatal("source SAV Health/tick versus cold World", loaded, file.Head.CounterA)
	}
	coldHash, coldSAV := cold.live.world.Hash(), fpsCurrentSAV(t, cold)
	if rate, pic, _, shown := cold.live.view.FPSReadout(9000); rate != 0 || pic != nil || shown {
		t.Fatal("cold ordinary LOAD inherited FPS state", rate, shown)
	}
	if err := coldApp.HeadlessKey("f12"); err != nil {
		t.Fatal(err)
	}
	if rate, pic, _, shown := cold.live.view.FPSReadout(9000); rate != 0 || !shown || !bytes.Equal(pic.Pix, fpsSourcePicture(glyphs, "0.0 fps", false, false, 0).Pix) {
		t.Fatal("cold primed zero readout", rate, shown)
	}
	retainedRate, _, _, _ := cold.live.view.FPSReadout(10040)
	if retainedRate != 3.0*1000/1040 {
		t.Fatal("nonzero LOAD retention control was not primed", retainedRate)
	}
	if cold.live.world.Hash() != coldHash || !bytes.Equal(fpsCurrentSAV(t, cold), coldSAV) {
		t.Fatal("cold sampling/composition/F12 changed World or SAV")
	}
	for _, failed := range []bool{false, true} {
		view, hash, saved := cold.live.view, cold.live.world.Hash(), fpsCurrentSAV(t, cold)
		if err := headlessOpenLoad(coldApp); err != nil {
			t.Fatal(err)
		}
		if failed {
			if err := os.WriteFile(path, []byte("broken SAV"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := coldApp.HeadlessKey("enter"); err != nil || coldApp.Screen() != ui.ScreenLoad || coldApp.HeadlessMessage() == "" {
				t.Fatal("failed LOAD did not retain chooser", err, coldApp.Screen())
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := coldApp.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if cold.live.view != view || !view.FPSShown() || cold.live.world.Hash() != hash || !bytes.Equal(fpsCurrentSAV(t, cold), saved) {
			t.Fatal("cancelled/failed LOAD replaced FPS or World state", failed)
		}
		if rate, _, _, shown := view.FPSReadout(10040); rate != retainedRate || !shown {
			t.Fatal("cancelled/failed LOAD reset FPS", failed, rate, shown)
		}
	}
	groundAppLoad(t, coldApp, list, "fps-readout.sav")
	if rate, _, _, shown := cold.live.view.FPSReadout(9000); rate != 0 || shown {
		t.Fatal("same SAV/tick new successful LOAD inherited FPS", rate, shown)
	}
	nextHash := cold.live.world.Hash()
	cold.live.tick()
	if cold.live.world.Tick() != uint64(file.Head.CounterA)+1 || cold.live.world.Hash() == nextHash {
		t.Fatal("next production tick lost the cold World continuation")
	}
	if unchanged, err := os.ReadFile(path); err != nil || !bytes.Equal(unchanged, raw) {
		t.Fatal("sampling/composition/LOAD/tick changed the source file", err)
	}
	proof := map[string]any{
		"atlas_sha256": fmt.Sprintf("%x", sha256.Sum256(atlas)), "advances_sha256": fmt.Sprintf("%x", sha256.Sum256(advances)),
		"glyphs": len(glyphs), "levels": levels, "transparent_cells": transparent, "hidden_sample": wantRate,
		"readout_pixels_sha256": fmt.Sprintf("%x", sha256.Sum256(pic.Pix)), "save_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)),
		"source_health": int16(hp), "restored_health": loaded.HP, "source_tick": file.Head.CounterA,
		"native_first_visible_rate": "Unknown", "native_masks_and_pixels": "Unknown", "gpu_smoothing_output": "not measured",
		"painted_zero_control": "derived mid-literal-zero atlas and synthetic UI ramp; measured font1 has no painted-zero cells",
		"next_tick":            cold.live.world.Tick(), "cancel_and_fail_retained_rate": retainedRate,
	}
	b, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proof.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("font1 raw identity, %d glyphs, levels %v; source Health %d -> cold %d; hidden rate %v", len(glyphs), levels, int16(hp), loaded.HP, wantRate)
}
