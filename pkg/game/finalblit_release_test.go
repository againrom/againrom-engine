package game

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

type finalMovieAudioRecorder struct {
	starts, stops, chunks int
	bytes                 int
	live                  bool
}

func (r *finalMovieAudioRecorder) Start(int, int)           { r.starts++; r.live = true }
func (r *finalMovieAudioRecorder) Push(p []byte)            { r.chunks++; r.bytes += len(p) }
func (r *finalMovieAudioRecorder) Stop()                    { r.stops++; r.live = false }
func (*finalMovieAudioRecorder) SetSettings(audio.Settings) {}

func TestReleaseCutsceneFinalBlitStopsAudioWithPictureRetained(t *testing.T) {
	f := releaseFront(t)
	a := f.App("final movie presentation")
	t.Cleanup(a.StopAudio)
	a.Layout(640, 480)
	if err := a.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	hash, tick := f.live.world.Hash(), f.live.world.Tick()
	bank := OpenCutscenes(f.Archives.Root, "video4")
	a.SetCutscenes(bank)
	rec := &finalMovieAudioRecorder{}
	a.SetCutsceneAudio(rec)
	for _, name := range []string{"m10/01.smk", "m50/01.smk"} {
		t.Run(name, func(t *testing.T) {
			media, err := bank.Media(name)
			if err != nil || len(media) < 16 || string(media[:4]) != "SMK2" {
				t.Fatal("installed movie header", err)
			}
			frames := binary.LittleEndian.Uint32(media[12:16])
			if frames < 90 {
				t.Fatal("installed movie has no final-frame subject", frames)
			}
			if !a.PlayCutscene(name) {
				t.Fatal("installed movie did not open", a.CutsceneError())
			}
			before := *rec
			var last uint32
			var final []byte
			var peak byte
			deadline := time.Now().Add(30 * time.Second)
			for time.Now().Before(deadline) && a.Screen() == ui.ScreenCutscene {
				if err := a.HeadlessCutsceneStep(""); err != nil {
					t.Fatal(err)
				}
				if a.Screen() != ui.ScreenCutscene {
					break
				}
				n := a.CutsceneFrameNumber()
				if n != 0 && n != last {
					if n != last+1 {
						t.Fatal("installed frame skipped", last, n)
					}
					last = n
					if rec.stops != before.stops || !rec.live {
						t.Fatal("movie audio stopped before the final picture")
					}
					pix, note, err := a.HeadlessFrame()
					if err != nil || note != "" {
						t.Fatal("installed movie composition", note, err)
					}
					if n == 40 {
						for i := 0; i < len(pix.Pix); i += 4 {
							peak = max(peak, pix.Pix[i], pix.Pix[i+1], pix.Pix[i+2])
						}
					}
					if n == frames {
						final = append([]byte(nil), pix.Pix...)
						break
					}
				}
				time.Sleep(time.Millisecond)
			}
			if last != frames || peak == 0 || len(final) != 640*480*4 || a.CutsceneFrameNumber() != frames {
				t.Fatal("final installed picture was not retained", last, frames, peak)
			}
			if rec.stops != before.stops+1 || rec.live || rec.starts != before.starts+1 || rec.bytes <= before.bytes {
				t.Fatal("final installed composition did not stop its delivered audio", *rec, before)
			}
			for i := 0; i < len(final); i += 4 {
				if !bytes.Equal(final[i:i+4], []byte{0, 0, 0, 255}) {
					t.Fatal("installed final fade/picture was lost", i)
				}
			}
			pix, _, err := a.HeadlessFrame()
			if err != nil || !bytes.Equal(pix.Pix, final) || rec.stops != before.stops+1 {
				t.Fatal("duplicate final composition changed picture or stopped twice", err)
			}
			for time.Now().Before(deadline) && a.Screen() == ui.ScreenCutscene {
				a.HeadlessCutsceneStep("")
				time.Sleep(time.Millisecond)
			}
			if a.Screen() != ui.ScreenMap || a.CutsceneError() != nil || rec.stops != before.stops+1 || f.live.world.Hash() != hash || f.live.world.Tick() != tick {
				t.Fatal("EOF continuation changed frozen mission or stopped twice", a.CutsceneError())
			}
			t.Logf("installed=%s declared=%d composed=%d final=opaque-black audio-bytes=%d stops=one-at-final-composition retained=yes world/tick=frozen; recording device, no physical audibility claim", name, frames, last, rec.bytes-before.bytes)
		})
	}
}
