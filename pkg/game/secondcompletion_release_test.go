package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/video"
	"golang.org/x/text/encoding/charmap"
)

func firstSuccessApp(t *testing.T) (*FrontEnd, *ui.App, *mapWorld) {
	t.Helper()
	f := secondGameFront(t)
	f.Cutscenes = OpenCutscenes(f.Archives.Root, "video4")
	a := f.App("first success")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	enterSecondCampaignMission(t, a)
	m := f.live
	dismiss := func() {
		for pages := 0; m.mission.open && m.mission.kind == ui.NoticeDialogue; pages++ {
			if pages >= 64 {
				t.Fatal("dialogue exceeded 64 pages")
			}
			if err := a.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 20; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismiss()
	id := m.mission.ids[0]
	if err := m.world.HeadlessPlace(id, 60, 36); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	m.view.Camera().CenterOn(60*32, 36*32)
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	x, y := 0, 0
	for row := 64; row < 650 && x == 0; row += 8 {
		for col := 180; col < 950; col += 8 {
			cx, cy, err := a.HeadlessDropCell(col, row)
			if err == nil && cx == 63 && cy == 36 {
				x, y = col, row
				break
			}
		}
	}
	if x == 0 {
		t.Fatal("first success destination outside map hit test")
	}
	for _, phase := range []string{"press", "release"} {
		if err := a.HeadlessPointer(phase, x, y); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 1200 && !(m.mission.open && m.mission.kind == ui.NoticeSuccess); i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismiss()
	}
	if m.world.Outcome() != sim.OutcomeWon || !m.mission.open || m.mission.kind != ui.NoticeSuccess {
		t.Fatalf("ordinary move did not show first success: %v", m.world.Outcome())
	}
	return f, a, m
}

func TestReleaseSecondGameFirstSuccessPresentation(t *testing.T) {
	f, a, m := firstSuccessApp(t)
	sound := &completionAudio{}
	a.SetCutsceneAudio(sound)
	body, kind, shown := m.view.NoticeState()
	if !shown || kind != ui.NoticeSuccess || body != f.Words.MissionWon {
		t.Fatalf("report = %q/%v/%v", body, kind, shown)
	}
	if strings.TrimSpace(body) == "" {
		t.Fatal("first success report is empty")
	}
	for _, row := range []struct {
		path  string
		index int
		got   string
	}{{MainTextPath, 140, body}, {DialogsTextPath, 43, f.Words.MenuVictory}, {DialogsTextPath, 154, f.Words.OutcomeContinue}} {
		raw, err := f.Archives.Containers.ReadFile(row.path)
		if err != nil {
			t.Fatal(err)
		}
		lines := bytes.Split(raw, []byte("\r\n"))
		want := string(lines[row.index])
		if f.Font.Value().Selector == 1 {
			utf8, err := charmap.Windows1251.NewDecoder().Bytes(lines[row.index])
			if err != nil {
				t.Fatal(err)
			}
			want = EncodeInstallText(string(utf8), 1)
		}
		if row.got != want {
			t.Fatalf("report table %s[%d] = %x, want %x", row.path, row.index, row.got, want)
		}
	}
	report, err := a.HeadlessNoticeFrame()
	if err != nil || report == nil {
		t.Fatal("report rendering", err)
	}
	output := os.Getenv("AGAINROM_PRESENTATION_OUTPUT")
	if output != "" {
		if err := os.MkdirAll(output, 0700); err != nil {
			t.Fatal(err)
		}
		if err := writeMediaFrame(output, "report", report); err != nil {
			t.Fatal(err)
		}
	}
	media, err := f.Cutscenes.Media("teleport/01.smk")
	if err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(t.TempDir(), "first.smk")
	if err := os.WriteFile(scratch, media, 0600); err != nil {
		t.Fatal(err)
	}
	refs := map[uint32][]byte{}
	for _, frame := range []uint32{1, 6, 84} {
		refs[frame] = completionFFmpeg(t, scratch, "-vf", fmt.Sprintf("select=eq(n\\,%d)", frame-1), "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "rgba")
		if len(refs[frame]) != 640*360*4 {
			t.Fatal("independent first-movie frame size", len(refs[frame]))
		}
	}
	tick, hash := m.world.Tick(), m.world.Hash()
	if err := a.HeadlessActivate("victory"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenCutscene || a.CutsceneName() != "teleport/01.smk" {
		t.Fatalf("first acknowledgement: screen=%s movie=%q error=%v", a.Screen(), a.CutsceneName(), a.CutsceneError())
	}
	hash = m.world.Hash()
	last, composed := uint32(0), uint32(0)
	deadline := time.Now().Add(30 * time.Second)
	for a.Screen() == ui.ScreenCutscene && time.Now().Before(deadline) {
		if err := a.HeadlessCutsceneStep(""); err != nil {
			t.Fatal(err)
		}
		if n := a.CutsceneFrameNumber(); n > last {
			if n != last+1 {
				t.Fatalf("frame order %d -> %d", last, n)
			}
			last = n
			pix, note, err := a.HeadlessFrame()
			if err != nil || note != "" {
				t.Fatal("movie composition", note, err)
			}
			composed++
			if ref, ok := refs[n]; ok {
				compareFirstMovieFrame(t, pix, ref, n)
				if err := writeMediaFrame(output, fmt.Sprintf("frame%03d", n), pix); err != nil {
					t.Fatal(err)
				}
			}
			if n == 167 {
				compareFirstMovieFrame(t, pix, make([]byte, 640*360*4), n)
				if err := writeMediaFrame(output, "frame167", pix); err != nil {
					t.Fatal(err)
				}
			}
		}
		time.Sleep(time.Millisecond)
	}
	if last != 167 || composed != 167 || a.CutsceneError() != nil {
		t.Fatalf("playback frames=%d composed=%d err=%v", last, composed, a.CutsceneError())
	}
	pcm := completionFFmpeg(t, scratch, "-vn", "-f", "s16le", "-acodec", "pcm_s16le")
	if sound.starts != 1 || sound.stops != 1 || sound.rate != 22050 || sound.channels != 2 || !bytes.Equal(sound.pcm.Bytes(), pcm) {
		t.Fatalf("App audio: start/stop=%d/%d format=%d/%d bytes=%d oracle=%d", sound.starts, sound.stops, sound.rate, sound.channels, sound.pcm.Len(), len(pcm))
	}
	if output != "" {
		if err := os.WriteFile(filepath.Join(output, "movie.pcm"), sound.pcm.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	checkCompletionDestination(t, f, a, m, tick, hash)
	t.Logf("ordinary first-success report->teleport/01.smk sha256=%x frames=167 rendered=167 selected-independent-RGBA=3 final-black=yes PCM=%d sha256=%x start/stop=1/1 destination=available20 tick=%d unchanged", sha256.Sum256(media), len(pcm), sha256.Sum256(pcm), tick)
}

type completionAudio struct {
	pcm                           bytes.Buffer
	starts, stops, rate, channels int
}

func (s *completionAudio) Start(rate, channels int) { s.starts++; s.rate, s.channels = rate, channels }
func (s *completionAudio) Push(pcm []byte)          { s.pcm.Write(pcm) }
func (s *completionAudio) Stop()                    { s.stops++ }
func (*completionAudio) SetSettings(audio.Settings) {}

func completionFFmpeg(t *testing.T, input string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("ffmpeg", append([]string{"-hide_banner", "-loglevel", "error", "-i", input}, append(args, "-")...)...)
	quietWitnessProcess(cmd)
	result, err := cmd.Output()
	if err != nil {
		t.Fatal("FFmpeg comparison", err)
	}
	return result
}

func compareFirstMovieFrame(t *testing.T, got *image.RGBA, ref []byte, n uint32) {
	t.Helper()
	if got.Bounds() != image.Rect(0, 0, 640, 480) {
		t.Fatal("movie canvas", got.Bounds())
	}
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			g := got.RGBAAt(x, y)
			want := [3]byte{}
			if y >= 60 && y < 420 {
				i := ((y-60)*640 + x) * 4
				copy(want[:], ref[i:i+3])
				if n == 1 {
					for c := range want {
						want[c] = byte(float32(want[c]) * 0.2)
					}
				}
			}
			if g.R != want[0] || g.G != want[1] || g.B != want[2] || g.A != 255 {
				t.Fatalf("independent movie frame%d pixel%d,%d got%v want%v", n, x, y, g, want)
			}
		}
	}
}

func checkCompletionDestination(t *testing.T, f *FrontEnd, a *ui.App, m *mapWorld, tick, hash uint64) {
	t.Helper()
	if a.Screen() != ui.ScreenTown || f.live != nil || m.world.Tick() != tick || m.world.Hash() != hash {
		t.Fatalf("movie destination=%s live=%v tick=%d/%d", a.Screen(), f.live != nil, m.world.Tick(), tick)
	}
	c := f.Town.second
	if c.bank[768] != 20 || c.bank[906] != 1 || c.current != (secondLocation{}) || len(c.available) != 1 || c.available[0] != (secondLocation{kind: 1, id: 20}) {
		t.Fatal("campaign did not advance exactly once", c)
	}
	before := *c
	if err := a.HeadlessCutsceneStep(""); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("notice"); err == nil {
		t.Fatal("duplicate acknowledgement found a notice")
	}
	if !reflect.DeepEqual(before, *c) {
		t.Fatal("duplicate acknowledgement advanced campaign")
	}
}

type completionFault struct {
	bank   *CutsceneBank
	broken bool
	late   bool
	names  []string
}

func (f *completionFault) Open(name string) (*video.Player, error) {
	f.names = append(f.names, name)
	if name == "teleport/01.smk" && (f.broken || f.late) {
		media, err := f.bank.Media(name)
		if err != nil {
			return nil, err
		}
		if f.broken {
			return video.StartSmackerDecoder(media[:200])
		}
		media = append([]byte(nil), media...)
		if len(media) < 108 {
			return nil, fmt.Errorf("installed movie has no frame index")
		}
		count := int(binary.LittleEndian.Uint32(media[12:16]))
		if binary.LittleEndian.Uint32(media[20:24])&1 != 0 {
			count++
		}
		if count < 2 || count > (len(media)-104)/5 {
			return nil, fmt.Errorf("installed movie has no second-frame index")
		}
		offset := 104 + 5*count + int(binary.LittleEndian.Uint32(media[52:56])) + int(binary.LittleEndian.Uint32(media[104:108])&^3)
		if offset < 0 || offset >= len(media) {
			return nil, fmt.Errorf("installed movie has no second-frame fault target")
		}
		media[104+4*count+1] |= 1
		media[offset] = 0
		return video.StartSmackerDecoder(media)
	}
	return nil, video.ErrAbsent
}

func TestReleaseSecondGameFirstMovieControls(t *testing.T) {
	for _, action := range []string{"key", "left", "right", "close", "enter", "escape", "teardown", "continue", "missing", "corrupt", "decode-error"} {
		t.Run(action, func(t *testing.T) {
			f, a, m := firstSuccessApp(t)
			sound := &completionAudio{}
			a.SetCutsceneAudio(sound)
			tick, hash := m.world.Tick(), m.world.Hash()
			var fault *completionFault
			if action == "missing" || action == "corrupt" || action == "decode-error" {
				fault = &completionFault{bank: f.Cutscenes, broken: action == "corrupt", late: action == "decode-error"}
				a.SetCutscenes(fault)
			}
			if action == "continue" {
				if err := a.HeadlessActivate("continue"); err != nil {
					t.Fatal(err)
				}
				if a.Screen() != ui.ScreenMap || m.mission.open || f.Town.second.bank[906] != 0 {
					t.Fatal("Continue took victory")
				}
				if err := a.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				for _, row := range []string{"end", "victory"} {
					if err := a.HeadlessGameMenuAction(row); err != nil {
						t.Fatal(err)
					}
				}
			} else if err := a.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			tick, hash = m.world.Tick(), m.world.Hash()
			if fault != nil && !fault.late {
				if len(fault.names) != 99 || sound.starts != 0 || sound.stops != 0 || (action == "corrupt") != (a.CutsceneError() != nil) {
					t.Fatal("optional movie scan", len(fault.names), sound, a.CutsceneError())
				}
			} else {
				deadline := time.Now().Add(5 * time.Second)
				for a.CutsceneFrameNumber() == 0 && time.Now().Before(deadline) {
					if err := a.HeadlessCutsceneStep(""); err != nil {
						t.Fatal(err)
					}
					time.Sleep(time.Millisecond)
				}
				if a.Screen() != ui.ScreenCutscene || sound.pcm.Len() == 0 {
					t.Fatal("no decoded audiovisual frame before control")
				}
				switch action {
				case "decode-error":
					deadline := time.Now().Add(5 * time.Second)
					for a.Screen() == ui.ScreenCutscene && time.Now().Before(deadline) {
						if err := a.HeadlessCutsceneStep(""); err != nil {
							t.Fatal(err)
						}
						time.Sleep(time.Millisecond)
					}
					if len(fault.names) != 99 || a.CutsceneError() == nil {
						t.Fatal("decoded-frame error did not finish the optional scan", fault.names, a.CutsceneError())
					}
				case "left", "right":
					press, release := "press", "release"
					if action == "right" {
						press, release = "right-press", "right-release"
					}
					for _, edge := range []string{press, release} {
						if err := a.HeadlessPointer(edge, 512, 384); err != nil {
							t.Fatal(err)
						}
					}
				case "enter", "escape":
					if err := a.HeadlessKey(action); err != nil {
						t.Fatal(err)
					}
				case "teardown":
					a.StopAudio()
				case "continue":
					if err := a.HeadlessCutsceneStep("key"); err != nil {
						t.Fatal(err)
					}
				default:
					if err := a.HeadlessCutsceneStep(action); err != nil {
						t.Fatal(err)
					}
				}
				if sound.starts != 1 || sound.stops != 1 {
					t.Fatal("movie audio lifecycle", sound)
				}
			}
			checkCompletionDestination(t, f, a, m, tick, hash)
			for presses := 0; a.Screen() == ui.ScreenTown && presses < 4; presses++ {
				if err := a.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
			}
			for _, row := range []string{"abort", "confirm-abort"} {
				if err := a.HeadlessGameMenuAction(row); err != nil {
					t.Fatal(err)
				}
			}
			if a.Screen() != ui.ScreenMenu {
				t.Fatal("destination menu return", a.Screen())
			}
			if err := a.HeadlessActivate("new game"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != ui.ScreenTown || !reflect.DeepEqual(f.Town.second, newSecondCampaign()) {
				t.Fatal("new game retained completion")
			}
			t.Logf("control=%s decodedPCM=%d stop=%d campaignOnce=yes duplicateAck=no menu/newgame=yes", action, sound.pcm.Len(), sound.stops)
		})
	}
}
