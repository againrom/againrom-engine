# 1178 — movie audio through the installed decoder

Cutscenes play their own audio. A player who watches `m10/01.smk` hears a
mono 22050Hz track through the process's normal audio device; `m20/01.smk`
plays stereo. Both come from the same lawful `smackw32.dll` this project
already opens for video, through a per-frame pull already proven against
both movies against the installed decoder's real output, cross-checked
byte-for-byte against an independent ffmpeg decode.

## The first probe's negative was an instrument error, not a decoder limit

An earlier pass of this story swept 46 call-shape variants against both
movies and reported zero audio bytes from every one, and concluded the
installed decoder's pull-model exports do not hand over track data under any
flag or argument shape. That conclusion was false. The probe itself carried
two independent argument faults:

1. `SmackGetTrackData`'s call passed its destination-buffer and track-flag
   arguments in the wrong order. The real signature is
   `(state, dest, trackflag)`; every variant called it as
   `(state, trackflag, dest)`, so the DLL wrote into the address given as a
   flag value and read the destination pointer as the track selector.
2. Every variant selected an audio track by index (`0` or `1`). The
   selector `SmackSoundInTrack` and `SmackGetTrackData` both take is a SMK2
   *bit flag*, not an index: the working value is `0x2000`. An index of `0`
   or `1` never matches that flag, so `SmackSoundInTrack` was always false
   and the pull never ran.

Corrected, the same file-handle-owned `SmackOpen` route `decodeNative`
already used for video pulls complete decompressed PCM per frame, with no
new `SmackOpen` flag and no sound device opened anywhere in the helper
process:

    SmackOpen(fd, 0x1000, -1)                         // already proven, unchanged
    // per frame, after SmackDoFrame(state):
    if SmackSoundInTrack(state, 0x2000) != 0 {
        n := SmackGetTrackData(state, dest, 0x2000)   // n is the actual byte count
    }
    SmackNextFrame(state)

`pkg/video/audioprobe_windows_386.go` carries this shape as the corrected
diagnostic instrument (`audioTrackFlag = 0x2000`), still swept against both
movies for independent confirmation; `pkg/video/native_windows_386.go`
carries the production pull `decodeNative` now performs on the same
`SmackOpen` state it already holds.

## What now reaches the player

- `pkg/video/stream.go`'s wire protocol between the decoder helper process
  and the game is now "ARV2": a 24-byte header carries width, height, frame
  count, audio rate, channel count and bit depth; each frame carries its own
  optional length-prefixed audio chunk, bounded by `MaxAudioChunk` (1 MiB).
  Every existing bound and refusal is preserved; `ReadHeader`/`WriteFrame`
  refuse a non-zero reserved header byte, an over-length or non-whole-sample
  audio chunk, and an audio/channel-count mismatch.
- `cmd/cutscenehelper` and `pkg/video.DecodeNative` pull the corrected shape
  above and write it into that stream; `InspectNativeInput` reads the SMK2
  header's own `AudioSize[0]` (offset 24) and `AudioRate[0]` (offset 72,
  low 24 bits = Hz, bit 28 = stereo) once per movie to size the destination
  buffer and populate the ARV2 header before decoding a single frame.
- `pkg/ui/cutsceneaudiodev.go` streams the pulled PCM to a player on the
  process's existing retained ebiten audio context, the same pattern already
  used for music and ambience (`pkg/ui/musicdev.go`, `pkg/ui/ambientdev.go`),
  with one structural difference those two do not need: this player's
  source is an unbounded queue fed one video frame's own chunk at a time,
  not a whole already-decoded buffer. ebiten's own `Play` primes its buffer
  by reading that queue synchronously, on whatever goroutine calls it, in a
  loop that does not return until the buffer fills or the queue reports
  EOF; since the only thing that can ever fill it is the same App step that
  would call `Play`, that call runs on its own goroutine (`Start`), leaving
  the step goroutine free to keep pushing while the priming read waits on
  the queue's own condition variable for exactly that data. 22050Hz equals
  `audio.DeviceRate`, so no resampling is needed; a mono track is
  duplicated to interleaved stereo before it reaches the player, since the
  shared context always mixes stereo. `pkg/game/gamemenu.go`
  routes `SoundOptions` to this device exactly like `MusicPlayer` and
  `AmbientPlayer`: an explicit `-sound=false` launch or a saved mute is
  silence, master volume scales it, and a headless run never opens a device.
  Every skip route — key, left click, right click, window close — reaches
  the same `closeCutscene` teardown seam in `pkg/ui/cutscene.go`, which stops
  the streamed track first, so no route can leave audio running past its
  video. A movie that fails to open, or a device that fails to open, stays
  an ordinary silent skip; neither ever raises an error modal over the
  screen the cutscene was meant to precede.

## Proof

Extraction (movie bytes never enter a repository):

    go run <extract-tool> "<EN root>/Allods/VIDEO4.RES" video4/m10/01.smk <scratch>/m10-01.smk
    go run <extract-tool> "<EN root>/Allods/VIDEO8.RES" video8/m20/01.smk <scratch>/m20-01.smk

using `pkg/vfs.OpenFileBacked` against the one named archive, the same
lookup `CutsceneBank.Media` performs.

Production decode:

    GOARCH=386 GOOS=windows go build -o <scratch>/cutscenehelper.exe ./cmd/cutscenehelper
    cutscenehelper.exe -dll "<EN root>/smackw32.dll" -input <scratch>/m10-01.smk > <scratch>/m10-01.arv2
    cutscenehelper.exe -dll "<EN root>/smackw32.dll" -input <scratch>/m20-01.smk > <scratch>/m20-01.arv2

Both runs: exit 0. The resulting ARV2 stream reports `Width=800 Height=360
Frames=90 AudioRate=22050 AudioChannels=1` for `m10/01.smk` and `Width=640
Height=360 Frames=75 AudioRate=22050 AudioChannels=2` for `m20/01.smk`.
Concatenating every frame's own audio chunk in order gives 264556 bytes for
`m10/01.smk` and 440912 bytes for `m20/01.smk` — the same totals the
corrected diagnostic probe reports independently, run directly against the
DLL with no ARV2 framing in between.

Independent ffmpeg cross-check (ffmpeg 9.0.1, `ffmpeg -i <movie> -map 0:a:0
-f s16le -acodec pcm_s16le <out>.pcm`): byte-identical to both totals above.
`ffprobe` independently reports `codec_name=smackaudio`, `sample_rate=22050`,
`channels=1` for m10 and `channels=2` for m20, matching this project's own
SMK2 header parse. Full manifest, exact commands, byte counts and SHA256
hashes: `review/story1178/MANIFEST.md`; reference PCM:
`review/story1178/m10-01.pcm`, `review/story1178/m20-01.pcm`.

The release-gated `TestReleaseCutsceneNativeAppCompletionAndSkip`
(`pkg/game/cutscene_release_test.go`, requires `AGAINROM_ASSETS`) drives the
whole production route — chargen entry, native decode, App composition and
now this device — against the EN install with a real WASAPI device open
throughout: `PASS (7.45s)`, reporting `native=800x360 frames=90 composed=90
oracle565=6/6 motion=yes route=chargen-to-map complete=map` and all four
skip variants (key/left/right/close) landing on the map with the simulation
unchanged. The same route through the shipped `cmd/againrom
-cutscene-check` CLI exits 0 with an identical report.

`gofmt -l` is clean over every touched file. `go build ./...` and
`GOARCH=386 GOOS=windows go build ./pkg/video/... ./cmd/audioprobe/...
./cmd/cutscenehelper/...` are clean.
`go test -trimpath -count=1 ./...` passes, including new coverage in
`pkg/video/stream_test.go` (ARV2 header/frame round trip and refusals, with
and without an audio chunk) and `pkg/game/cutscene_witness.go`'s existing
witness, which now also asserts the delivered audio format
(rate=22050, channels=1) for mission 10's entry cutscene.

## Authority

- `VIDEO-034` records that game-side cutscene setup enables decoder sound
  after successful setup: ROM1 does play the track it decodes, which this
  story's corrected pull confirms is recoverable.
- `VIDEO-045` (promoted, amended) establishes the borrowed file-handle
  `SmackOpen` route this story's pull reuses unchanged.
- `VIDEO-047` records the ordinary indexed-output decode/write path this
  project's own per-frame video pull already follows.
- `VIDEO-050` records sound-progress-dependent wait paths in the original
  decoder's own readiness check — evidence that ROM1 paces frame delivery
  against its own device's playback progress, which this story does not
  reproduce (see Open debt).
- `DIV-1254` is corrected to record that audio now plays and to name the two
  argument faults above; it moves from `OPEN` to `CLOSED`. `DIV-1255`
  (mixing/volume policy) and `DIV-1256` (timing/synchronisation policy)
  record the two differences from decoder-driven ROM1 behaviour that remain,
  both `OPEN`.

The lane uses public knowledge snapshot `c4073aef7e22bbc849b0efdd2a04a13047e37ae5`.

## Open debt

- **Open-bit rationale confusion.** An earlier pass of this probe explained
  the exclusion of `SmackOpen` flag bits 24-31 by analogy to the SMK2
  header's own `AudioRate` words (whose high bits carry a stereo flag, not
  part of an open-time argument at all). That analogy does not hold: bits
  24-31 are excluded because bit 25 crashes the decoder process, observed
  directly, not because of any resemblance to the header layout. The stereo
  bit in `AudioRate[0]` (bit 28, container header offset 72) and the
  `SmackOpen` argument bits are unrelated fields that happen to share a bit
  position; the resemblance is coincidence, not a shared contract.
- **`SmackOpen(0x1000 | 0x02000000, ...)` kills the process.** Combining the
  proven file-handle bit with the second allocation-selector bit reliably
  crashes the decoder with an access violation. Nothing in this story's
  shipped path constructs that combination; it is recorded so a future
  caller does not. Research's own `VIDEO-045` retraction independently flags
  this exact combination's callee branch as "untested and not proved usable
  or safe" from the disassembly side, which corroborates the crash from a
  second, independent source.
- **Untestable second-track Unknown.** The audio-track selector `0x2000` is
  one of four bit positions a Smacker file's track-selection byte can carry
  (the format allows up to four tracks). No shipped movie in either install
  carries a second track, so whether `0x4000` or `0x8000` behave the same
  way as `0x2000` is untested and cannot be tested against real installed
  data; this story does not claim they work.
- **Timing/synchronisation policy (`DIV-1256`).** Audio and video for a
  frame are delivered together and played independently once queued; this
  story does not reproduce whatever sound-progress-dependent pacing
  `VIDEO-050` shows the original decoder's own readiness wait performs. No
  drift measurement against original playback exists.
