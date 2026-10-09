package game

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"againrom/pkg/ui"
	"againrom/pkg/video"
)

// WitnessCutscene drives startup and a controlled completion through the
// production App overlay. The expected pixels come from the DLL's separate packed-word decoder,
// not from the gameplay palette conversion or App compositor. It observes no
// physical desktop pixels. Only the explicitly supplied private output is written.
func (f *FrontEnd) WitnessCutscene(root, helper, output string, report io.Writer) error {
	root, err := editorPhysicalDirectory(root)
	if err != nil {
		return err
	}
	output, err = editorPhysicalDirectory(output)
	if err != nil {
		return err
	}
	root, _ = filepath.Abs(root)
	output, _ = filepath.Abs(output)
	rel, err := filepath.Rel(root, output)
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return fmt.Errorf("video witness: output must be outside install")
	}
	return f.witnessMedia(root, helper, output, report)
}

// WitnessMedia runs the production decoder, UI routes and audio device without
// creating files or loading the installed native decoder.
func (f *FrontEnd) WitnessMedia(report io.Writer) error {
	return f.witnessMedia("", "", "", report)
}

func (f *FrontEnd) witnessMedia(root, helper, output string, report io.Writer) error {
	bank := f.Cutscenes
	if bank == nil {
		return fmt.Errorf("video witness: no configured bank")
	}
	media, err := bank.Media("m10/01.smk")
	if err != nil {
		return err
	}
	if len(media) < 104 || string(media[:4]) != "SMK2" {
		return fmt.Errorf("video witness: missing M10 header")
	}
	w, h, frames := binary.LittleEndian.Uint32(media[4:]), binary.LittleEndian.Uint32(media[8:]), binary.LittleEndian.Uint32(media[12:])
	if w == 0 || h == 0 || w > video.MaxDimension || h > video.MaxDimension || frames < 6 || frames > 1000 {
		return fmt.Errorf("video witness: M10 outside witness bounds")
	}
	sidecar, err := bank.sidecar("m10/01.smk")
	if err != nil {
		return err
	}
	if sidecar == nil {
		return fmt.Errorf("video witness: M10 sidecar missing")
	}
	var refs []*image.RGBA
	if helper != "" {
		private, err := os.MkdirTemp(output, "cutscene-input-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(private)
		input := filepath.Join(private, "movie.smk")
		if err := os.WriteFile(input, media, 0600); err != nil {
			return err
		}
		dll, err := cutsceneInstallPath(root, "smackw32.dll")
		if err != nil {
			return err
		}
		refs, err = packedReference(helper, dll, input, w, h, frames)
		if err != nil {
			return err
		}
		if bytes.Equal(refs[0].Pix, refs[5].Pix) {
			return fmt.Errorf("video witness: first six frames show no movement")
		}
	}
	a := f.App("cutscene read-only witness")
	defer a.StopAudio()
	// This is the command's normal startup door, including Play's dispatch
	// into the mission opener. Later skip cases use direct mission entry too.
	begin := func(result ui.ChargenResult) (ui.MapOpener, error) {
		return f.MissionOpenerWith(10, f.ChargenParty(result)), nil
	}
	a.SetNewGameChargen(func() *ui.ChargenEntry {
		return &ui.ChargenEntry{Model: ui.NewChargen(f.ChargenSetup()), Begin: begin}
	})
	if !a.PlayStartupCutscenes() {
		return fmt.Errorf("video witness: startup route absent: %v", a.CutsceneError())
	}
	if err := witnessMovieEntry(a, "startup", ui.ScreenMenu, output, report); err != nil {
		return err
	}
	if err := a.HeadlessActivate("new game"); err != nil {
		return err
	}
	if err := witnessMovieEntry(a, "newgame", ui.ScreenChargen, output, report); err != nil {
		return err
	}
	if err := headlessCreateCharacter(a, HeadlessCharacter{Name: "Cutscene"}); err != nil {
		return err
	}
	if a.Screen() != ui.ScreenCutscene {
		return fmt.Errorf("video witness: production entry did not play: %v", a.CutsceneError())
	}
	hash, tick := f.live.world.Hash(), f.live.world.Tick()
	// The character-creation prelude finishes into gameplay. M10 belongs to
	// the completion event, and must not be requested by this entry.
	preludeDeadline := time.Now().Add(60 * time.Second)
	preludeFrames := uint32(0)
	for strings.HasPrefix(a.CutsceneName(), "start/") && time.Now().Before(preludeDeadline) {
		if err := a.HeadlessCutsceneStep(""); err != nil {
			return err
		}
		if strings.HasPrefix(a.CutsceneName(), "start/") {
			preludeFrames = a.CutsceneFrameNumber()
			if _, note, err := a.HeadlessFrame(); err != nil || note != "" {
				return fmt.Errorf("video witness: prelude composition: %s %v", note, err)
			}
		}
		time.Sleep(time.Millisecond)
	}
	if preludeFrames == 0 || a.Screen() != ui.ScreenMap || a.CutsceneName() != "" || f.live.world.Hash() != hash || f.live.world.Tick() != tick {
		return fmt.Errorf("video witness: campaign prelude did not reach the unchanged mission: frames=%d movie=%s", preludeFrames, a.CutsceneName())
	}
	fmt.Fprintf(report, "cutscene: route=character-created movie=start/01.smk frames=%d complete=map no-mission-movie tick=%d unchanged\n", preludeFrames, tick)
	if err := f.witnessCutsceneCompletion(a, 10); err != nil {
		return err
	}
	hash, tick = f.live.world.Hash(), f.live.world.Tick()
	deadline := time.Now().Add(45 * time.Second)
	last, composed, checked := uint32(0), uint32(0), 0
	var audioRate uint32
	var audioChannels uint8
	for a.Screen() == ui.ScreenCutscene && time.Now().Before(deadline) {
		if err := a.HeadlessCutsceneStep(""); err != nil {
			return err
		}
		if a.Screen() != ui.ScreenCutscene {
			break
		}
		audioRate, audioChannels = a.CutsceneAudioFormat()
		n := a.CutsceneFrameNumber()
		if n != 0 && n != last {
			if n != last+1 {
				return fmt.Errorf("video witness: frame sequence %d -> %d", last, n)
			}
			last = n
			pix, note, err := a.HeadlessFrame()
			if err != nil || note != "" {
				return fmt.Errorf("video witness: composition: %s %v", note, err)
			}
			composed++
			if n <= 6 && len(refs) > 0 {
				if err := comparePackedComposition(pix, refs[n-1], int(n-1), sidecar); err != nil {
					return fmt.Errorf("frame %d: %w", n, err)
				}
				checked++
				if n == 1 || n == 6 {
					file, err := os.Create(filepath.Join(output, fmt.Sprintf("frame%03d.png", n)))
					if err != nil {
						return err
					}
					err = png.Encode(file, pix)
					closeErr := file.Close()
					if err != nil {
						return err
					}
					if closeErr != nil {
						return closeErr
					}
				}
			}
		}
		time.Sleep(time.Millisecond)
	}
	// The installed M10 track is mono in VIDEO4 and stereo in VIDEO8.
	// Capture the production stream's format before completion closes it.
	wantChannels := uint8(1)
	if bank.archive == "video8" {
		wantChannels = 2
	}
	if audioRate != 22050 || audioChannels != wantChannels {
		return fmt.Errorf("video witness: m10 audio format rate=%d channels=%d, want 22050/%d", audioRate, audioChannels, wantChannels)
	}
	if a.Screen() != ui.ScreenMap || a.CutsceneError() != nil || last != frames || composed != frames || checked != len(refs) {
		return fmt.Errorf("video witness: completion screen=%s frames=%d/%d oracle=%d err=%v", a.Screen(), last, frames, checked, a.CutsceneError())
	}
	if f.live.world.Hash() != hash || f.live.world.Tick() != tick {
		return fmt.Errorf("video witness: completion advanced simulation")
	}
	oracle := "not-requested"
	if len(refs) > 0 {
		oracle = "6/6"
	}
	fmt.Fprintf(report, "cutscene: %s m10/01.smk sha256=%x native=%dx%d frames=%d composed=%d oracle565=%s route=controlled-victory complete=map tick=%d unchanged\n", bank.archive, sha256.Sum256(media), w, h, last, composed, oracle, tick)
	for _, skip := range []string{"key", "left", "right", "close"} {
		if err := f.witnessCutsceneCompletion(a, 10); err != nil {
			return err
		}
		hash, tick = f.live.world.Hash(), f.live.world.Tick()
		if a.Screen() != ui.ScreenCutscene {
			return fmt.Errorf("video witness: %s skip route absent", skip)
		}
		// Require a decoded frame before exercising each skip input.
		until := time.Now().Add(10 * time.Second)
		for a.CutsceneFrameNumber() == 0 && a.Screen() == ui.ScreenCutscene && time.Now().Before(until) {
			a.HeadlessCutsceneStep("")
			time.Sleep(time.Millisecond)
		}
		if a.CutsceneFrameNumber() == 0 {
			return fmt.Errorf("video witness: no native frame before %s", skip)
		}
		if err := a.HeadlessCutsceneStep(skip); err != nil {
			return err
		}
		if a.Screen() != ui.ScreenMap || f.live.world.Hash() != hash || f.live.world.Tick() != tick {
			return fmt.Errorf("video witness: %s skip changed destination/world", skip)
		}
		fmt.Fprintf(report, "cutscene: skip=%s decoded-frame=yes destination=map tick=%d unchanged\n", skip, tick)
	}
	if err := f.witnessCutsceneCompletion(a, 20); err != nil {
		return err
	}
	if a.CutsceneName() != "m20/01.smk" {
		return fmt.Errorf("video witness: later mission requested %s", a.CutsceneName())
	}
	hash, tick = f.live.world.Hash(), f.live.world.Tick()
	if err := witnessMovieEntry(a, "later-controlled-victory", ui.ScreenMap, output, report); err != nil {
		return err
	}
	if f.live.world.Hash() != hash || f.live.world.Tick() != tick {
		return fmt.Errorf("video witness: later mission advanced simulation")
	}
	a.StopAudio()
	return f.witnessTownAfterClosed(bank.root, output, report)
}

func (f *FrontEnd) witnessTownAfterClosed(root, output string, report io.Writer) error {
	if err := audioWitnessClosed(ui.DeliveryOwner(f.SoundPlayer)); err != nil {
		return err
	}
	town, err := newAudioWitnessFront(root, f.Options, f.Sound, f.SoundChannels, f.runtime.deterministicFrames)
	if err != nil {
		return err
	}
	townApp := town.App("town media witness")
	defer townApp.StopAudio()
	fmt.Fprintln(report, "cutscene: original shared service closed/empty; town witness uses a fresh installed front end and service")
	return town.WitnessTownMedia(output, report)
}

// This witness controls only the outcome decision, not campaign gameplay.
// Entry, the real Victory input, viewer replacement and movie scheduling are
// production routes. Reopen the same map to compare frozen simulation state
// across every skip variant without advancing the owner's campaign.
func (f *FrontEnd) witnessCutsceneCompletion(a *ui.App, mission int) error {
	open := f.MissionOpener(mission)
	err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
		if err == nil {
			v.SetNotice(f.Words.MissionWon, ui.NoticeSuccess)
			taken := false
			advance = func(actions ...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
				if taken || len(actions) != 0 && actions[0] != ui.NoticeVictory && actions[0] != ui.NoticeAdvance {
					return ui.NoticeStay, "", nil
				}
				taken = true
				return ui.NoticeToMission, "", open
			}
		}
		return v, tick, order, cadence, affect, advance, attack, grab, stance, march, err
	})
	if err != nil {
		return err
	}
	if a.Screen() != ui.ScreenMap || a.CutsceneName() != "" {
		return fmt.Errorf("video witness: mission%d played before Victory", mission)
	}
	// Drain a preceding skip without issuing another gameplay command.
	if err := a.HeadlessCutsceneStep(""); err != nil {
		return err
	}
	if err := a.HeadlessKey("enter"); err != nil {
		return err
	}
	if a.CutsceneName() != fmt.Sprintf("m%d/01.smk", mission) {
		return fmt.Errorf("video witness: Victory%d requested %q", mission, a.CutsceneName())
	}
	return nil
}

func witnessMovieEntry(a *ui.App, route string, destination ui.Screen, output string, report io.Writer) error {
	if a.Screen() != ui.ScreenCutscene {
		return fmt.Errorf("video witness: %s route did not play", route)
	}
	name := a.CutsceneName()
	deadline := time.Now().Add(10 * time.Second)
	for a.Screen() == ui.ScreenCutscene && a.CutsceneFrameNumber() < 6 && time.Now().Before(deadline) {
		if err := a.HeadlessCutsceneStep(""); err != nil {
			return err
		}
		time.Sleep(time.Millisecond)
	}
	if a.CutsceneName() != name || a.CutsceneFrameNumber() < 6 {
		return fmt.Errorf("video witness: %s produced fewer than six frames", route)
	}
	pix, note, err := a.HeadlessFrame()
	if err != nil || note != "" {
		return fmt.Errorf("video witness: %s composition: %s %v", route, note, err)
	}
	if err := writeMediaFrame(output, route, pix); err != nil {
		return err
	}
	rate, channels := a.CutsceneAudioFormat()
	if rate == 0 || channels == 0 {
		return fmt.Errorf("video witness: %s audio absent", route)
	}
	if err := a.HeadlessCutsceneStep("key"); err != nil {
		return err
	}
	if a.Screen() != destination {
		return fmt.Errorf("video witness: %s skipped to %s", route, a.Screen())
	}
	if err := a.HeadlessCutsceneStep(""); err != nil {
		return err
	}
	fmt.Fprintf(report, "cutscene: route=%s movie=%s frames=6 audio=%d/%d skip=%s\n", route, name, rate, channels, destination)
	return nil
}

func writeMediaFrame(output, name string, pix image.Image) error {
	if output == "" {
		return nil
	}
	file, err := os.Create(filepath.Join(output, name+".png"))
	if err != nil {
		return err
	}
	err = png.Encode(file, pix)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func packedReference(helper, dll, input string, w, h, frames uint32) ([]*image.RGBA, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, helper, "-dll", dll, "-input", input, "-oracle565")
	quietWitnessProcess(cmd)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() { pipe.Close(); cmd.Process.Kill(); cmd.Wait() }()
	info, err := video.ReadHeader(pipe)
	if err != nil {
		return nil, err
	}
	if info.Width != w || info.Height != h || info.Frames != frames {
		return nil, fmt.Errorf("video witness: packed header mismatch")
	}
	refs := make([]*image.RGBA, 6)
	for n := range refs {
		refs[n], _, err = video.ReadFrame(pipe, info)
		if err != nil {
			return nil, err
		}
	}
	return refs, nil
}

// comparePackedComposition checks one composed canvas against the packed
// reference frame of the same movie (VIDEO-071, VIDEO-072). The expectation is
// built independently of the presenter: a 640x360 source window of the
// reference at the sidecar's pan origin, centred in 640x480, its palette scaled
// by the fade factor in force at that frame. Where a fade is in force the
// packed reference cannot restore the unscaled palette, so channels agree
// within fadeTolerance; elsewhere they agree exactly.
func comparePackedComposition(got, reference *image.RGBA, k int, sc *video.Sidecar) error {
	if got.Bounds() != image.Rect(0, 0, 640, 480) {
		return fmt.Errorf("unexpected canvas %v", got.Bounds())
	}
	const winW, winH, y0 = 640, 360, 60
	if reference.Rect.Dx() < winW || reference.Rect.Dy() != winH {
		return fmt.Errorf("reference %v is not a windowed movie", reference.Rect)
	}
	originX, originY := int(sc.StartX), int(sc.StartY)
	for _, p := range sc.Pans {
		n := k - int(p.Start)
		if n < 0 {
			continue
		}
		if n > int(p.End-p.Start) {
			n = int(p.End - p.Start)
		}
		originX += n * int(p.StepX)
		originY += n * int(p.StepY)
	}
	factor, faded := 1.0, false
	for _, f := range sc.Fades {
		if k >= int(f.Start) && k < int(f.End) {
			factor = float64(f.From) + float64(f.To-f.From)*float64(k-int(f.Start)+1)/float64(f.End-f.Start)
			faded = true
		}
	}
	const fadeTolerance = 16
	near := func(g uint8, r uint8, mask uint8) bool {
		want := float64(r) * factor
		if !faded {
			return g&mask == r
		}
		d := float64(g) - want
		return d <= fadeTolerance && d >= -fadeTolerance
	}
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			g := got.RGBAAt(x, y)
			if y < y0 || y >= y0+winH {
				if g.R != 0 || g.G != 0 || g.B != 0 || g.A != 255 {
					return fmt.Errorf("letterbox pixel %d,%d", x, y)
				}
				continue
			}
			r := reference.RGBAAt(x+originX, y-y0+originY)
			if !near(g.R, r.R, 0xf8) || !near(g.G, r.G, 0xfc) || !near(g.B, r.B, 0xf8) || g.A != 255 {
				return fmt.Errorf("packed mismatch at %d,%d got=%v reference=%v factor=%.3f", x, y, g, r, factor)
			}
		}
	}
	return nil
}
