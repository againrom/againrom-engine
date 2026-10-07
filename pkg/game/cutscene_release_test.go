package game

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
)

func TestReleaseCutsceneNativeAppCompletionAndSkip(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}
	if runtime.GOOS != "windows" {
		t.Skip("native cutscene adapter requires Windows")
	}
	helper := os.Getenv("AGAINROM_CUTSCENE_HELPER")
	if helper == "" {
		helper = filepath.Join(t.TempDir(), "cutscenehelper.exe")
		cmd := exec.Command("go", "build", "-trimpath", "-o", helper, "./cmd/cutscenehelper")
		cmd.Dir = filepath.Join("..", "..")
		for _, v := range os.Environ() {
			if !strings.HasPrefix(v, "GOARCH=") {
				cmd.Env = append(cmd.Env, v)
			}
		}
		cmd.Env = append(cmd.Env, "GOARCH=386")
		quietWitnessProcess(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build 386 helper: %v\n%s", err, out)
		}
	}
	f, err := NewFrontEnd(root)
	if err != nil {
		t.Fatal(err)
	}
	f.Cutscenes = OpenCutscenes(root, "video4")
	var report bytes.Buffer
	if err := f.WitnessCutscene(root, helper, t.TempDir(), &report); err != nil {
		t.Fatalf("%v\n%s", err, &report)
	}
	t.Log(report.String())
}

func testSoundOptionsWitnessConsumer(t *testing.T) {
	f := releaseFront(t)
	output := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(output, "options.txt")}
	var report bytes.Buffer
	if err := f.WitnessSoundOptions(f.Archives.Root, output, &report); err != nil {
		t.Fatal(err, report.String())
	}
	if err := audioWitnessClosed(ui.DeliveryOwner(f.SoundPlayer)); err != nil {
		t.Fatal(err)
	}
	t.Log(report.String())
}

type delayedSpeechPositionPlayer struct {
	*deliveryWitnessPlayer
	pending int
}

func (p *delayedSpeechPositionPlayer) Pause() {
	p.deliveryWitnessPlayer.Pause()
	p.pending = 4
}

func (p *delayedSpeechPositionPlayer) Position() time.Duration {
	if !p.playing && p.closes == 0 && p.pending > 0 {
		p.pending--
		if p.pending == 0 {
			p.position += time.Millisecond
		}
	}
	return p.position
}

func (p *delayedSpeechPositionPlayer) DeviceGain() float64 { return 0 }

func testDelayedSpeechPositionCache(t *testing.T) {
	var players []*delayedSpeechPositionPlayer
	restore := ui.SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (ui.DeliveryDevicePlayer, error) {
		player := &delayedSpeechPositionPlayer{deliveryWitnessPlayer: &deliveryWitnessPlayer{reader: reader, position: 17 * time.Millisecond}}
		players = append(players, player)
		return player, nil
	})
	t.Cleanup(restore)
	f := releaseFront(t)
	a := f.App("delayed position cache witness")
	t.Cleanup(a.StopAudio)
	var report bytes.Buffer
	if err := f.witnessTownResponses(&report); err != nil {
		t.Fatal(err)
	}
	if strings.Count(report.String(), "backend=native-cache-cuts before=374 immediate=374 paused=396 resumed=396") != 34 {
		t.Fatal("delayed cache control did not publish one later paused position for all 34 responses", report.String())
	}
	a.StopAudio()
	if err := audioWitnessClosed(ui.DeliveryOwner(f.SoundPlayer)); err != nil {
		t.Fatal(err)
	}
	for _, player := range players {
		if player.closes != 1 || player.playing || player.seeks != 0 {
			t.Fatal("delayed cache control lost close-once or no-rewind semantics", player)
		}
	}
	t.Log("software control emulates one delayed native Position cache publication; positive settled phase, same owner/generation/SAV and exact destruction retained")
}

func TestReleaseTownMediaKeepsSharedSpeechAcrossFocus(t *testing.T) {
	var players []*deliveryWitnessPlayer
	pcmBuffers := 0
	restore := ui.SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (ui.DeliveryDevicePlayer, error) {
		size, err := reader.Seek(0, io.SeekEnd)
		if err != nil || size <= 0 || size > 64<<20 {
			t.Fatalf("invalid installed PCM size %d: %v", size, err)
		}
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		raw := make([]byte, size)
		if _, err := io.ReadFull(reader, raw); err != nil {
			t.Fatal(err)
		}
		if bytes.Count(raw, []byte{0}) == len(raw) {
			t.Fatal("installed software PCM buffer is all zero")
		}
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		pcmBuffers++
		player := &deliveryWitnessPlayer{reader: reader, position: 17 * time.Millisecond}
		players = append(players, player)
		return player, nil
	})
	t.Cleanup(restore)
	f := releaseFront(t)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	a := f.App("town media focus witness")
	t.Cleanup(a.StopAudio)
	var report bytes.Buffer
	if err := f.WitnessTownMedia("", &report); err != nil {
		t.Fatal(err, report.String())
	}
	a.StopAudio()
	if err := audioWitnessClosed(ui.DeliveryOwner(f.SoundPlayer)); err != nil {
		t.Fatal(err)
	}
	if pcmBuffers != 68 || strings.Count(report.String(), "backend=controlled-exact") != 34 {
		t.Fatalf("PCM buffers=%d exact focus cuts=%d, want 68/34", pcmBuffers, strings.Count(report.String(), "backend=controlled-exact"))
	}
	for _, player := range players {
		if player.closes != 1 || player.playing || player.seeks != 0 {
			t.Fatal("controlled player survived, closed twice or rewound", player)
		}
	}
	t.Log("installed software PCM; positive 17ms phase; exact shared ID/generation/phase, SAV and close-once controls; no physical audibility claim")
	t.Log(report.String())
	restore()
	t.Run("asynchronous-cache-control", testDelayedSpeechPositionCache)
	t.Run("sound-options-consumer", testSoundOptionsWitnessConsumer)
}
