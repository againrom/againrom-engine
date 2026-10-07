# 1179 — a portable Smacker decoder

`pkg/video/smacker` decodes ROM1's `.smk` cutscene video and audio in pure Go, with
no cgo, no native helper process and no installed `smackw32.dll` at runtime. It is
a from-scratch port of libsmacker 1.2.0 (Greg Kennedy, LGPL-2.1-or-later), verified
against two independent oracles — ffmpeg's own Smacker decoder and the installed
DLL through a disposable windows/386 helper process — rather than against ROM1
directly: this port is third-party derived code under attribution, not research
evidence, and produces no claim of its own. It is now the live cutscene decode
path: a player who starts or reaches a mission movie is decoded and heard through
this port, not the installed DLL, which is retained only as this story's own
gate-time oracle.

## Authority

- `VIDEO-033` (promoted): ROM1 links a bundled Smacker decoder, `smackw32.dll`,
  through a fixed imported API surface at runtime, a Windows/386-only
  dependency.
- `VIDEO-034` (promoted): the game-side call order opens decoder state through
  that DLL, obtains dimensions, configures a destination and enables decoder
  sound.
- No promoted claim reaches the decoder's own internal bitstream, Huffman-tree
  or DPCM algorithm; those are properties of the third-party Smacker container
  format, documented by libsmacker itself, not ROM1-specific research.
- Knowledge pin: snapshot 37, `c4073aef7e22bbc849b0efdd2a04a13047e37ae5`.

## As-built behaviour

- `pkg/video/smacker` mirrors libsmacker's own file layout for auditability:
  `bitstream.go` (LSB-first bit reader), `huff.go` (the 8-bit and 16-bit Huffman
  tree shapes, the latter with its 3-entry MRU cache), `container.go` (header,
  frame index and Huffman-tree-chunk parsing, with allocation bounds this port
  adds beyond upstream's trust-the-header behaviour), `video.go` (palette
  rendering and the six block types), `audio.go` (raw and DPCM audio-chunk
  decode), and `decoder.go` (the top-level `Open`/`EnableVideo`/`EnableAudio`/
  `Next` API, restated with idiomatic Go error returns).
- It is a leaf package: no intra-module imports, registered as such in
  `internal/archtest`'s dependency-DAG allow-map. It cannot import `pkg/video`,
  its one caller, without creating the cycle the port exists to avoid.
- **The decoder is the live cutscene decode path.** `pkg/video/smackerdecode.go`
  adds `StartSmackerDecoder(media []byte) (*Player, error)`: it decodes in this
  process with `pkg/video/smacker` and feeds the same ARV2 stream
  (`pkg/video/stream.go`, `player.go`) the UI already reads for frames and
  audio (`pkg/ui/cutsceneaudiodev.go`). `pkg/game/cutscene.go`'s
  `CutsceneBank.Open` calls it directly; no helper subprocess, no installed
  DLL, no `runtime.GOOS` check, at runtime. It carries an audio track only
  when the track's rate, bit depth and channel count match this project's one
  fixed playback rate (`nativeAudioRate`, `native.go`) — the same gate the
  installed-decoder path already applied, so a track this project cannot play
  back cleanly is silently left off rather than played at the wrong pitch.
- **The former helper-subprocess machinery is retired, not kept as a second
  path.** `StartDecoder`, `processReader` and the temp-directory/window-hiding
  code around it (formerly `pkg/video/process.go`, `process_windows.go`,
  `process_other.go`, `process_test.go`) are deleted: once the swap above
  landed, nothing called them. `MaxMediaBytes` and `ErrAbsent`, which other
  callers still use, moved to a new small `pkg/video/media.go`. The installed
  decoder is not gone: `cmd/cutscenehelper` and `pkg/video`'s
  `DecodeNative`/`InspectNativeInput` are unchanged and still built, because
  `pkg/game/cutscene_witness.go`'s `WitnessCutscene` spawns
  `cmd/cutscenehelper` directly (with `-oracle565`) as an independent,
  gate-time-only oracle cross-checking this port's production output — a
  measured reason to keep it, not a default. `cmd/againrom`'s normal `-movies`
  run no longer looks up `cutscenehelper.exe` beside the executable at all;
  `-cutscene-check` still does, for that same oracle role.
- Attribution: `LICENSES/LGPL-2.1.txt` carries the license text (copied from
  `review/libsmacker-reference/COPYING`, outside every repository); every
  ported file carries a citing header; `THIRD_PARTY_NOTICES.md` gains a
  "Source-derived (ported) code" section for module-less ports, distinct from
  the go.mod-tracked compiled-module table above it; `internal/notices` gains
  `TestPortedSourceNoticesPresent`, which fails if the ported tree, its notice
  entry or its license file ever go out of sync, the same way
  `TestNoticesMatchGoMod` already guards go.mod dependencies.

## Proof

`TestReleaseSmackerDecoderAgainstOracles` (`pkg/video/smacker/release_test.go`,
gated on `AGAINROM_ASSETS`, registered in
`internal/gatedtests/testdata/population.txt`) is the gate-time evidence:

- **Structural census, whole corpus.** Every payload in both shipped video
  archives decodes without error and, when ffmpeg/ffprobe are present, is
  cross-checked against them for dimensions, frame count and (single-track)
  audio byte total: 33/33 on the EN root, 33/33 on the RU root.
- **Byte-exact sample, every shape.** Five payloads spanning mono/stereo,
  VIDEO4/VIDEO8, 15/25fps and smallest/largest were compared byte-for-byte
  against both oracles, all five with zero diff bytes:

  | Payload | Video bytes (RGBA) | Audio bytes (PCM) | DLL frames matched |
  |---|---|---|---|
  | video4/m10/01.smk | 103,680,000 | 264,556 | 90/90 |
  | video4/logos/buka.smk | 430,080,000 | 1,234,800 | 350/350 |
  | video4/intro/04.smk | 301,593,600 | 3,848,034 | 1,309/1,309 |
  | video8/m10/01.smk | 103,680,000 | 529,112 | 90/90 |
  | video8/m20/01.smk | 69,120,000 | 440,912 | 75/75 |

  The two audio totals the coordinator supplied from an independent DLL
  measurement — 264,556 bytes for `video4/m10/01.smk` and 440,912 bytes for
  `video8/m20/01.smk` — match this port's own ffmpeg-compared totals exactly.
- **Player result, live.** `againrom.exe -assets <EN root> -cutscene-check
  <dir>` drives the actual FrontEnd -> mission entry -> App overlay route
  through the production `CutsceneBank.Open`, now backed by
  `StartSmackerDecoder`. Its own report line from this landing:
  `cutscene: video4 m10/01.smk sha256=f6252abb235db26b09dda3c3ddeb7e5531bb67b50ca2c46b1e30479b3e9216bc native=800x360 frames=90 composed=90 oracle565=6/6 motion=yes route=chargen-to-map complete=map tick=0 unchanged`,
  followed by all four skip routes (`key`, `left`, `right`, `close`) each
  reporting `native-frame=yes destination=map ... unchanged`. `oracle565=6/6`
  is this port's production frames matching the installed DLL's independent
  packed-word rendering, composed through the ordinary App path, not a
  fixture. The witness's own audio-format assertion (rate=22050, channels=1
  for `m10/01.smk`) passed inside the same run, read through the same
  production wiring.
- **Gates on the landing commit (this stage, after the swap):** `gofmt -l`
  clean; `go test -trimpath -count=1 ./...` 50/50 packages `ok`, none failing;
  `scripts/check-no-game-assets.sh` clean; `pipeline/check-release-tests.sh`
  against both `gameversions/en` and `gameversions/ru` in one invocation,
  284/284 gated tests run on each root, 0 lacking a subject on either;
  `pipeline/check-preserved-installs.sh` unchanged (554 files, every root as
  recorded); `pipeline/check-div-claims.sh` exit 0, none of this story's rows
  among the 127 citing a retracted claim. The gated-test count moved from 283
  (story 1178's own landing) to 284 with no new test added this stage: the
  extra one is `TestReleaseSmackerDecoderAgainstOracles`, registered in an
  earlier stage of this same story and now included in the merged population.
- **Portability, measured, not assumed.** `pkg/video/smacker` and its one
  caller `pkg/video` both build clean with `CGO_ENABLED=0` for
  `GOOS=linux/amd64` and `GOOS=darwin/arm64`. Sweeping every non-`pkg/ui`,
  non-`cmd` package individually found exactly one failure on both target
  OSes: `pkg/game`, which is pre-existing and unrelated to this story — caused
  by `pkg/game`'s import of `pkg/ui`, which directly imports
  `github.com/hajimehoshi/ebiten/v2` and `ebitengine/oto` (37 files); the
  build fails inside those two modules. `pkg/audio/mix.go` imports only
  `encoding/binary` and `math` and is not the cause. 31 of the 32 swept
  packages build clean on both OSes.

## Open debt

- **`pkg/game` does not cross-compile for `linux/amd64` or `darwin/arm64` with
  `CGO_ENABLED=0`.** Pre-existing, outside this story's scope. The cause is
  `pkg/game`'s import of `pkg/ui`, which itself directly imports
  `github.com/hajimehoshi/ebiten/v2` and `ebitengine/oto` (37 files); the
  build fails inside `ebitengine/oto/v3` and ebiten's `opengl`/`glfw` graphics
  driver. `pkg/audio/mix.go` imports only `encoding/binary` and `math` and is
  not the cause — confirmed by reading every file in `pkg/audio/` and by
  cross-compiling `./pkg/audio/...` alone clean with `CGO_ENABLED=0
  GOOS=linux GOARCH=amd64`. This failure predates this story (present on
  `origin/main` before this story started) and is untouched by it. This
  story does not close that gap and does not claim to: swapping the cutscene
  decoder to pure Go makes cutscene decode itself portable, not the rest of
  the game. A player on Linux or macOS still cannot build or run `pkg/game`
  today; a player on Windows is the only one this story changes anything for
  in practice, even though the decoder and `pkg/video` themselves build clean
  without cgo on both other target OSes.
- **The installed decoder is retained, deliberately, as a gate-time-only
  oracle.** `cmd/cutscenehelper` and the windows/386 export-binding decode it
  runs (`pkg/video/native.go`, `native_windows_386.go`) stay, because
  `WitnessCutscene`'s release test needs an oracle independent of this port to
  keep catching a regression in either one. Nothing else calls them: the
  former production entry point (`StartDecoder`, `pkg/video/process.go`'s
  subprocess management) is deleted rather than kept unused beside the new
  one.
- `DIV-1257` records the decoder-vs-DLL replacement itself; its status moves
  from `OPEN` to `ACCEPTED` now that the replacement is live and the DLL's
  role is gate-time-only. `DIV-1258` and `DIV-1259` record two bounded gaps (a
  non-block-aligned extent refusal, and no Bink-audio-codec support) that this
  project's own shipped corpus never exercises; both are unaffected by which
  decoder is live and are left as they were. The review pass found two further
  bounded gaps in the new decoder itself, both unexercised by the shipped
  corpus: `DIV-1260` (no frame-interval sanity bound, where the retired
  installed-decoder path had one) and `DIV-1261` (a hypothetical Bink-
  compressed track that also matches this project's accepted rate/depth/
  channel gate would truncate the whole movie rather than being silently left
  off). Both were spent in the hotfix correction pass that followed review.
  `DIV-1262`, the last of this story's six reserved ids, is left unspent: an
  unspent id is never reissued.
